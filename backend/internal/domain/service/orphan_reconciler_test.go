package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// stepPlan describes the active step of a run for the reconciler harness: its
// status, optional container_id, and how long ago it started. A run is orphaned
// IFF its active running step launched a container (containerID != "") that is
// not in the live-running set, past the grace window.
type stepPlan struct {
	status      model.StepStatus
	containerID string // "" means no container was launched (ci_poll/hitl/git_*)
	startedAgo  time.Duration
	noStartedAt bool // active step with a nil started_at (launch in flight)
}

// runPlan binds a run id to the steps the repo should return for it.
type runPlan struct {
	id    uuid.UUID
	steps []stepPlan
}

// runningStep builds a `running` step with a launched container started ago.
func runningStep(containerID string, startedAgo time.Duration) stepPlan {
	return stepPlan{status: model.StepStatusRunning, containerID: containerID, startedAgo: startedAgo}
}

// reconcileResult captures one MarkRunOrphanedIfRunning call for assertions.
type reconcileResult struct {
	id          uuid.UUID
	completedAt time.Time
	errMsg      string
}

// reconcilerHarness wires a mockContainerManager (driving ListRunningContainers)
// and a mockRunRepo (driving ListRunsByStatus / ListRunStepsByRun /
// MarkRunOrphanedIfRunning) for the reconciler RG tests.
//
// liveContainerIDs is the set of currently RUNNING managed container ids.
// markAffected controls whether MarkRunOrphanedIfRunning reports a row updated
// (false models the TOCTOU case: the run already transitioned terminal).
func reconcilerHarness(
	t *testing.T,
	liveContainerIDs []string,
	listErr error,
	runs []runPlan,
	markAffected bool,
) (*OrphanReconciler, *[]reconcileResult) {
	t.Helper()

	containers := make([]port.ContainerInfo, 0, len(liveContainerIDs))
	for _, id := range liveContainerIDs {
		containers = append(containers, port.ContainerInfo{
			ID:     id,
			Labels: map[string]string{"managed_by": "hopeitworks"},
		})
	}

	containerMgr := &mockContainerManager{
		listRunningFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			if listErr != nil {
				return nil, listErr
			}
			return containers, nil
		},
		// ListContainers must NOT be consulted by the reconciler; make it loud.
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			t.Error("reconciler must use ListRunningContainers, not ListContainers")
			return nil, nil
		},
	}

	now := time.Now()
	stepsByRun := make(map[uuid.UUID][]*model.RunStep, len(runs))
	running := make([]*model.Run, 0, len(runs))
	for _, rp := range runs {
		running = append(running, &model.Run{ID: rp.id, Status: model.RunStatusRunning})
		steps := make([]*model.RunStep, 0, len(rp.steps))
		for _, sp := range rp.steps {
			step := &model.RunStep{ID: uuid.New(), RunID: rp.id, Status: sp.status}
			if sp.containerID != "" {
				cid := sp.containerID
				step.ContainerID = &cid
			}
			if !sp.noStartedAt {
				started := now.Add(-sp.startedAgo)
				step.StartedAt = &started
			}
			steps = append(steps, step)
		}
		stepsByRun[rp.id] = steps
	}

	results := &[]reconcileResult{}
	runRepo := &mockRunRepo{
		listRunsByStatusFn: func(_ context.Context, status model.RunStatus) ([]*model.Run, error) {
			if status != model.RunStatusRunning {
				t.Errorf("reconciler must query only running runs, got %s", status)
			}
			return running, nil
		},
		listRunStepsByRunFn: func(_ context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
			return stepsByRun[runID], nil
		},
		markRunOrphanedFn: func(_ context.Context, id uuid.UUID, completedAt time.Time, errMsg string) (bool, error) {
			*results = append(*results, reconcileResult{id: id, completedAt: completedAt, errMsg: errMsg})
			return markAffected, nil
		},
	}

	r := NewOrphanReconciler(containerMgr, runRepo, discardLogger(), DefaultOrphanGraceWindow)
	return r, results
}

// RG1 + RG4: a run whose running step launched a container that is no longer in
// ListRunningContainers (crashed/exited/removed), past the grace window, is
// marked failed with reason orphaned_no_container and a completed_at timestamp
// (RG4: bounded duration).
func TestReconcileOrphanedRuns_CrashedContainer_MarkedFailed(t *testing.T) {
	orphanID := uuid.New()
	// Step launched container "c-dead"; live-running set is EMPTY (container
	// exited — present in ListContainers but absent from ListRunningContainers).
	r, results := reconcilerHarness(t,
		nil, // no live running containers
		nil,
		[]runPlan{{id: orphanID, steps: []stepPlan{runningStep("c-dead", 10*time.Minute)}}},
		true,
	)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(*results) != 1 {
		t.Fatalf("expected exactly 1 run reconciled, got %d", len(*results))
	}
	got := (*results)[0]
	if got.id != orphanID {
		t.Errorf("expected orphan run %s reconciled, got %s", orphanID, got.id)
	}
	if got.errMsg != orphanedNoContainerReason {
		t.Errorf("expected reason %q, got %q", orphanedNoContainerReason, got.errMsg)
	}
	// RG4: completed_at is set → duration is bounded.
	if got.completedAt.IsZero() {
		t.Error("expected completed_at to be set (RG4: bounded duration)")
	}
}

// RG3 (a): a running step with NO container_id (ci_poll/hitl_gate/git_*) is a
// legit long-lived running state and is never reconciled, even past grace.
func TestReconcileOrphanedRuns_RunningStepNoContainer_Untouched(t *testing.T) {
	id := uuid.New()
	noContainer := stepPlan{status: model.StepStatusRunning, containerID: "", startedAgo: 30 * time.Minute}

	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: id, steps: []stepPlan{noContainer}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected no run reconciled (no-container step is legit), got %d", len(*results))
	}
}

// RG3 (b): a running step whose container IS in ListRunningContainers is healthy
// and never reconciled.
func TestReconcileOrphanedRuns_ContainerAlive_Untouched(t *testing.T) {
	id := uuid.New()

	r, results := reconcilerHarness(t,
		[]string{"c-alive"}, nil,
		[]runPlan{{id: id, steps: []stepPlan{runningStep("c-alive", 10*time.Minute)}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected no run reconciled (container alive), got %d", len(*results))
	}
}

// RG3 (c): a running step with a container but started < grace ago is left
// untouched — the container may still be starting / its id not yet persisted.
func TestReconcileOrphanedRuns_WithinGraceWindow_Untouched(t *testing.T) {
	id := uuid.New()

	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: id, steps: []stepPlan{runningStep("c-fresh", 5*time.Second)}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected no run reconciled (within grace window), got %d", len(*results))
	}
}

// RG3 (c, nil started_at): a running container-backed step with nil started_at
// (launch in flight) is left untouched — no lower bound for the grace check.
func TestReconcileOrphanedRuns_StepNilStartedAt_Untouched(t *testing.T) {
	id := uuid.New()
	inFlight := stepPlan{status: model.StepStatusRunning, containerID: "c-inflight", noStartedAt: true}

	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: id, steps: []stepPlan{inFlight}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected no run reconciled (step nil started_at), got %d", len(*results))
	}
}

// RG3 (d): a run in an inter-step gap (no step is `running`; previous completed,
// next not yet started) is left untouched — there is no active container to miss.
func TestReconcileOrphanedRuns_InterStepGap_Untouched(t *testing.T) {
	id := uuid.New()
	completed := stepPlan{status: model.StepStatusCompleted, containerID: "c-old", startedAgo: 20 * time.Minute}
	pending := stepPlan{status: model.StepStatusPending}

	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: id, steps: []stepPlan{completed, pending}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected no run reconciled (inter-step gap), got %d", len(*results))
	}
}

// RG5 (CRITICAL): when ListRunningContainers errors (runtime/Docker unreachable)
// the reconciler returns the error and touches NO run AND never queries DB runs —
// reconcile only on proof of absence, never on a listing failure.
func TestReconcileOrphanedRuns_ListError_TouchesNothing(t *testing.T) {
	listErr := errors.NewInternal("docker unreachable", nil)

	results := &[]reconcileResult{}
	containerMgr := &mockContainerManager{
		listRunningFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return nil, listErr
		},
	}
	listRunsCalled := false
	runRepo := &mockRunRepo{
		listRunsByStatusFn: func(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
			listRunsCalled = true
			return []*model.Run{{ID: uuid.New(), Status: model.RunStatusRunning}}, nil
		},
		markRunOrphanedFn: func(_ context.Context, id uuid.UUID, completedAt time.Time, errMsg string) (bool, error) {
			*results = append(*results, reconcileResult{id: id, completedAt: completedAt, errMsg: errMsg})
			return true, nil
		},
	}

	r := NewOrphanReconciler(containerMgr, runRepo, discardLogger(), DefaultOrphanGraceWindow)

	if err := r.ReconcileOrphanedRuns(context.Background()); err == nil {
		t.Fatal("expected error to propagate when ListRunningContainers fails, got nil")
	}
	if len(*results) != 0 {
		t.Fatalf("RG5 violated: %d run(s) marked failed on a listing error; must be 0", len(*results))
	}
	if listRunsCalled {
		t.Error("RG5: must abort before querying DB runs when listing fails (proof-of-absence only)")
	}
}

// RG6 / TOCTOU: a run that transitioned to a terminal state between the snapshot
// and the conditional write is NOT overwritten. MarkRunOrphanedIfRunning reports
// 0 rows affected (markAffected=false); the reconciler treats it as a no-op.
func TestReconcileOrphanedRuns_ConcurrentlyCompleted_NotOverwritten(t *testing.T) {
	id := uuid.New()

	// Looks orphaned (crashed container, past grace) but the conditional update
	// affects 0 rows because the run already completed concurrently.
	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: id, steps: []stepPlan{runningStep("c-dead", 10*time.Minute)}}},
		false, // MarkRunOrphanedIfRunning → 0 rows affected
	)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// The conditional write was attempted exactly once...
	if len(*results) != 1 {
		t.Fatalf("expected exactly 1 conditional write attempt, got %d", len(*results))
	}
	// ...but reported 0 rows affected → the run keeps its terminal state (the
	// reconciler did not log it as reconciled; correctness lives in the WHERE
	// clause asserted by MarkRunOrphanedIfRunning returning false).
}

// RG6 (idempotence): with no running runs the reconciler is a no-op and stays a
// no-op on a second pass.
func TestReconcileOrphanedRuns_NoRunningRuns_Idempotent(t *testing.T) {
	r, results := reconcilerHarness(t, nil, nil, nil, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error on second pass, got %v", err)
	}
	if len(*results) != 0 {
		t.Fatalf("expected reconciliation to remain idempotent, got %d", len(*results))
	}
}

// Mixed batch: among a crashed orphan, a healthy run (container alive), a fresh
// run (within grace), and a ci_poll run (no-container step), only the orphan is
// reconciled — zero false positives at scale.
func TestReconcileOrphanedRuns_MixedBatch_OnlyOrphanReconciled(t *testing.T) {
	orphanID := uuid.New()
	healthyID := uuid.New()
	freshID := uuid.New()
	ciPollID := uuid.New()

	r, results := reconcilerHarness(t,
		[]string{"c-healthy"}, // only healthy's container is running
		nil,
		[]runPlan{
			{id: orphanID, steps: []stepPlan{runningStep("c-dead", 10*time.Minute)}},
			{id: healthyID, steps: []stepPlan{runningStep("c-healthy", 10*time.Minute)}},
			{id: freshID, steps: []stepPlan{runningStep("c-fresh", 2*time.Second)}},
			{id: ciPollID, steps: []stepPlan{{status: model.StepStatusRunning, startedAgo: 12 * time.Minute}}},
		},
		true,
	)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(*results) != 1 {
		t.Fatalf("expected exactly 1 run reconciled (the orphan), got %d", len(*results))
	}
	if (*results)[0].id != orphanID {
		t.Errorf("expected orphan %s reconciled, got %s", orphanID, (*results)[0].id)
	}
}

// RG2 (boot entrypoint): the boot path is the same ReconcileOrphanedRuns method
// main() calls at startup; this exercises that entrypoint and proves a boot-time
// orphan is reconciled.
func TestReconcileOrphanedRuns_BootEntrypoint_ReconcilesOrphan(t *testing.T) {
	orphanID := uuid.New()

	r, results := reconcilerHarness(t, nil, nil,
		[]runPlan{{id: orphanID, steps: []stepPlan{runningStep("c-dead", 30*time.Minute)}}}, true)

	if err := r.ReconcileOrphanedRuns(context.Background()); err != nil {
		t.Fatalf("boot reconciliation returned error: %v", err)
	}
	if len(*results) != 1 || (*results)[0].id != orphanID {
		t.Fatalf("expected boot reconciliation to fail the orphan, got %+v", *results)
	}
}

// RG2 (watchdog wiring): a TimeoutEnforcer constructed WITH a reconciler invokes
// it on each tick. Drives the real Start loop and asserts the orphan is
// reconciled with no live container present.
func TestTimeoutEnforcer_TickReconcilesOrphans(t *testing.T) {
	orphanID := uuid.New()
	cid := "c-dead"
	started := time.Now().Add(-10 * time.Minute)

	containerMgr := &mockContainerManager{
		// No running containers for the orphan's step.
		listRunningFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{}, nil
		},
		// CheckTimeouts iterates ListContainers; keep it empty.
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{}, nil
		},
	}

	reconciled := make(chan uuid.UUID, 1)
	runRepo := &mockRunRepo{
		listRunsByStatusFn: func(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
			return []*model.Run{{ID: orphanID, Status: model.RunStatusRunning}}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{{
				ID: uuid.New(), RunID: runID, Status: model.StepStatusRunning,
				ContainerID: &cid, StartedAt: &started,
			}}, nil
		},
		markRunOrphanedFn: func(_ context.Context, id uuid.UUID, _ time.Time, _ string) (bool, error) {
			select {
			case reconciled <- id:
			default:
			}
			return true, nil
		},
	}
	projectRepo := newMockProjectRepoForService()

	enforcer := NewTimeoutEnforcer(
		containerMgr, runRepo, projectRepo, discardLogger(),
		30*time.Minute, 20*time.Millisecond,
		NewOrphanReconciler(containerMgr, runRepo, discardLogger(), DefaultOrphanGraceWindow),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = enforcer.Start(ctx) }()

	select {
	case id := <-reconciled:
		if id != orphanID {
			t.Errorf("expected watchdog to reconcile orphan %s, got %s", orphanID, id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog tick did not reconcile the orphaned run")
	}
}
