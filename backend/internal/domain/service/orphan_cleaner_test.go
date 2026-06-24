package service

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	dockeradapter "github.com/zakari/hopeitworks/backend/internal/adapter/docker"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

func TestCleanupOrphans_NoOrphans(t *testing.T) {
	runID1 := uuid.New()
	runID2 := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-active-1",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID1.String()},
				},
				{
					ID:     "container-active-2",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID2.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case runID1:
				return &model.Run{ID: runID1, Status: model.RunStatusRunning}, nil
			case runID2:
				return &model.Run{ID: runID2, Status: model.RunStatusPending}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 0 {
		t.Errorf("expected no remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_MultipleOrphans(t *testing.T) {
	activeRunID := uuid.New()
	completedRunID := uuid.New()
	missingRunID := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-active",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": activeRunID.String()},
				},
				{
					ID:     "container-no-label",
					Labels: map[string]string{"managed_by": "hopeitworks"},
				},
				{
					ID:     "container-completed",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": completedRunID.String()},
				},
				{
					ID:     "container-missing-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": missingRunID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case activeRunID:
				return &model.Run{ID: activeRunID, Status: model.RunStatusRunning}, nil
			case completedRunID:
				return &model.Run{ID: completedRunID, Status: model.RunStatusCompleted}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 3 {
		t.Fatalf("expected 3 remove calls (no-label, completed, missing-run), got %d", len(removeCalls))
	}

	expected := map[string]bool{
		"container-no-label":    false,
		"container-completed":   false,
		"container-missing-run": false,
	}
	for _, id := range removeCalls {
		if _, ok := expected[id]; !ok {
			t.Errorf("unexpected remove call for %s", id)
		}
		expected[id] = true
	}
	for id, called := range expected {
		if !called {
			t.Errorf("expected remove call for %s, but it was not called", id)
		}
	}
}

func TestCleanupOrphans_OrphanReasons(t *testing.T) {
	failedRunID := uuid.New()
	cancelledRunID := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-no-run-id",
					Labels: map[string]string{"managed_by": "hopeitworks"},
				},
				{
					ID:     "container-invalid-uuid",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": "not-a-uuid"},
				},
				{
					ID:     "container-failed-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": failedRunID.String()},
				},
				{
					ID:     "container-cancelled-run",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": cancelledRunID.String()},
				},
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			switch id {
			case failedRunID:
				return &model.Run{ID: failedRunID, Status: model.RunStatusFailed}, nil
			case cancelledRunID:
				return &model.Run{ID: cancelledRunID, Status: model.RunStatusCancelled}, nil
			default:
				return nil, errors.NewNotFound("run", id)
			}
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 4 {
		t.Fatalf("expected 4 remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_RemoveFailureContinues(t *testing.T) {
	runID1 := uuid.New()
	runID2 := uuid.New()

	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{
				{
					ID:     "container-fail-remove",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID1.String()},
				},
				{
					ID:     "container-ok-remove",
					Labels: map[string]string{"managed_by": "hopeitworks", "run_id": runID2.String()},
				},
			}, nil
		},
		removeFn: func(_ context.Context, containerID string) error {
			if containerID == "container-fail-remove" {
				return errors.NewInternal("remove failed", nil)
			}
			return nil
		},
	}

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			// Both runs not found → both containers are orphans
			return nil, errors.NewNotFound("run", id)
		},
	}

	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error (cleanup continues on failure), got %v", err)
	}

	// Both containers should have Remove called, even though first one fails
	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 2 {
		t.Fatalf("expected 2 remove calls (cleanup continues), got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_EmptyContainerList(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return []port.ContainerInfo{}, nil
		},
	}

	runRepo := &mockRunRepo{}
	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	removeCalls := containerMgr.getRemoveCalls()
	if len(removeCalls) != 0 {
		t.Errorf("expected no remove calls, got %d", len(removeCalls))
	}
}

func TestCleanupOrphans_ListContainersError(t *testing.T) {
	containerMgr := &mockContainerManager{
		listFn: func(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
			return nil, errors.NewInternal("docker unreachable", nil)
		},
	}

	runRepo := &mockRunRepo{}
	cleaner := NewOrphanCleaner(containerMgr, runRepo, discardLogger())

	err := cleaner.CleanupOrphans(context.Background())
	if err == nil {
		t.Fatal("expected error when ListContainers fails, got nil")
	}
}

// reapableCM is a stateful fake port.ContainerManager that records containers on
// Create (with the labels stamped by the caller) and filters them on
// ListContainers / Remove. It is the shared substrate the contract test uses to
// prove that a container launched through docker.Runtime is later reapable by
// OrphanCleaner — the labels round-trip end-to-end, no special-casing.
type reapableCM struct {
	mu         sync.Mutex
	seq        int
	containers map[string]map[string]string // id -> labels
	removed    []string
}

func newReapableCM() *reapableCM {
	return &reapableCM{containers: map[string]map[string]string{}}
}

func (c *reapableCM) Create(_ context.Context, opts model.ContainerOpts) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	id := uuid.NewString()
	labels := make(map[string]string, len(opts.Labels))
	for k, v := range opts.Labels {
		labels[k] = v
	}
	c.containers[id] = labels
	return id, nil
}

func (c *reapableCM) Start(_ context.Context, _ string) error { return nil }

func (c *reapableCM) Stop(_ context.Context, _ string) error { return nil }

func (c *reapableCM) Remove(_ context.Context, containerID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removed = append(c.removed, containerID)
	delete(c.containers, containerID)
	return nil
}

func (c *reapableCM) Wait(_ context.Context, _ string) (int, error) { return 0, nil }

func (c *reapableCM) ListContainers(_ context.Context, filter map[string]string) ([]port.ContainerInfo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []port.ContainerInfo
	for id, labels := range c.containers {
		if labelsMatch(labels, filter) {
			out = append(out, port.ContainerInfo{ID: id, Labels: labels})
		}
	}
	return out, nil
}

func (c *reapableCM) ListRunningContainers(_ context.Context, _ map[string]string) ([]port.ContainerInfo, error) {
	return nil, nil
}

func (c *reapableCM) CreateNetwork(_ context.Context, _ string, _ map[string]string) (string, error) {
	return "", nil
}
func (c *reapableCM) RemoveNetwork(_ context.Context, _ string) error { return nil }
func (c *reapableCM) ConnectContainer(_ context.Context, _, _ string, _ []string) error {
	return nil
}
func (c *reapableCM) ListNetworks(_ context.Context, _ map[string]string) ([]model.NetworkInfo, error) {
	return nil, nil
}
func (c *reapableCM) InspectHealth(_ context.Context, _ string) (string, error) {
	return model.HealthRunning, nil
}

func (c *reapableCM) getRemoved() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.removed))
	copy(out, c.removed)
	return out
}

// labelsMatch reports whether every key/value in filter is present in labels.
func labelsMatch(labels, filter map[string]string) bool {
	for k, v := range filter {
		if labels[k] != v {
			return false
		}
	}
	return true
}

// TestOrphanCleaner_FindsContainerLaunchedViaDockerRuntime proves the reaper
// contract end-to-end: a container launched through docker.Runtime.Launch carries
// the managed_by/run_id labels (stamped via RunSpec.Labels), so OrphanCleaner finds
// and removes it when its run_id is unknown (orphan) — and leaves it alone when the
// run is active. This is the substrate-dispatch proof: the Docker reaping contract
// holds through the port with zero special-casing.
func TestOrphanCleaner_FindsContainerLaunchedViaDockerRuntime(t *testing.T) {
	orphanRunID := uuid.New()
	activeRunID := uuid.New()
	stepID := uuid.New()

	cm := newReapableCM()
	rt := dockeradapter.NewRuntime(cm, "agent-net", discardLogger())

	// Labels mirror buildAgentLabels: managed_by + run/step/story identity.
	launch := func(runID uuid.UUID) port.RunHandle {
		h, err := rt.Launch(context.Background(), port.RunSpec{
			Image: "img",
			Labels: map[string]string{
				"managed_by": "hopeitworks",
				"run_id":     runID.String(),
				"step_id":    stepID.String(),
				"story_key":  "S-1",
			},
		})
		if err != nil {
			t.Fatalf("launch via docker.Runtime failed: %v", err)
		}
		return h
	}

	orphanHandle := launch(orphanRunID)
	activeHandle := launch(activeRunID)

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == activeRunID {
				return &model.Run{ID: activeRunID, Status: model.RunStatusRunning}, nil
			}
			// orphanRunID (and anything else) is unknown → orphan.
			return nil, errors.NewNotFound("run", id)
		},
	}

	cleaner := NewOrphanCleaner(cm, runRepo, discardLogger())
	if err := cleaner.CleanupOrphans(context.Background()); err != nil {
		t.Fatalf("CleanupOrphans returned error: %v", err)
	}

	removed := cm.getRemoved()
	if len(removed) != 1 {
		t.Fatalf("expected exactly 1 reaped container (the orphan), got %d: %v", len(removed), removed)
	}
	if removed[0] != orphanHandle.ID {
		t.Errorf("expected orphan container %q reaped, got %q", orphanHandle.ID, removed[0])
	}
	if removed[0] == activeHandle.ID {
		t.Errorf("active-run container %q must NOT be reaped", activeHandle.ID)
	}
}
