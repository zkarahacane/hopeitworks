package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// orphanedNoContainerReason is the error message set on a run that is marked
// failed because its active (running) step launched a container that is no
// longer running.
const orphanedNoContainerReason = "orphaned_no_container"

// DefaultOrphanGraceWindow is the minimum age (since the active step's
// started_at) a running step must have before its run can be reconciled as
// orphaned. It gives the launch→persist(container_id) window time to complete so
// a freshly launched container is never mistaken for a missing one.
const DefaultOrphanGraceWindow = 60 * time.Second

// OrphanReconciler reconciles DB run statuses against live container state.
//
// Whereas OrphanCleaner removes leftover *containers* (and never touches run
// statuses), this reconciler does the inverse: it finds runs whose active step
// launched a container that has since died (crashed/killed) yet left the run
// stuck `running` in the DB forever (it shows in "Active runs" with an
// ever-growing duration). The timeout enforcer can't help because it iterates
// EXISTING containers, so a run whose container is already gone is never
// visited.
//
// Correct orphan invariant (the crux of this service):
//
// A `running` run is orphaned IFF it has a run_step with status=running that
// carries a non-nil container_id, whose started_at is older than the grace
// window, AND that container_id is absent from the set of currently RUNNING
// managed containers. Concretely:
//
//   - Only steps that actually launched a container (agent_run) carry a
//     container_id. ci_poll / hitl_gate / git_pr / git_branch run NO container,
//     so their running step has a nil container_id → never orphaned (a run can
//     legitimately sit `running` for the ci_poll 15-min timeout or a human gate).
//   - Between two agent_run steps there is a gap with no running step / no
//     container — no running step with a container_id → never orphaned.
//   - A just-launched container is covered by the per-step grace window.
//   - A crashed agent_run container leaves its step `running` with a posted
//     container_id absent from ListRunningContainers → orphaned after grace.
//
// We list via ContainerManager.ListRunningContainers (RUNNING only, not
// ListContainers): an exited-but-not-yet-removed container must count as gone,
// otherwise a crashed run would never be detected.
//
// Substrate scope (ADR Stage 4): like OrphanCleaner/TimeoutEnforcer this reaper
// is Docker-shaped — it enumerates live executions via ListRunningContainers
// (managed_by=hopeitworks) keyed by container id. Reconciling a NON-Docker
// substrate needs a runtime-level "list managed executions" capability and is
// DEFERRED until one ships live.
type OrphanReconciler struct {
	containerMgr port.ContainerManager
	runRepo      port.RunRepository
	logger       *slog.Logger
	graceWindow  time.Duration
}

// NewOrphanReconciler creates a new OrphanReconciler. graceWindow is the minimum
// age the active running step must have before its run can be reconciled (use
// DefaultOrphanGraceWindow).
func NewOrphanReconciler(
	containerMgr port.ContainerManager,
	runRepo port.RunRepository,
	logger *slog.Logger,
	graceWindow time.Duration,
) *OrphanReconciler {
	return &OrphanReconciler{
		containerMgr: containerMgr,
		runRepo:      runRepo,
		logger:       logger,
		graceWindow:  graceWindow,
	}
}

// ReconcileOrphanedRuns marks `running` runs whose active container-backed step
// has no live container as `failed` with reason orphaned_no_container.
//
// Safety contract (the whole point of this method):
//   - It reconciles ONLY on PROOF OF ABSENCE: the running-container listing must
//     SUCCEED, and the step's container_id must be absent from it. If
//     ListRunningContainers errors (Docker/runtime unreachable) it logs and
//     aborts WITHOUT touching any run — a transient listing failure must never
//     mass-fail healthy runs.
//   - A run whose active step launched no container (ci_poll/hitl_gate/git_*) is
//     left untouched: that is a normal long-lived `running` state.
//   - A run whose active step is still within the grace window is left untouched,
//     giving a just-launched container time to appear.
//   - The DB write is conditional on status='running' (TOCTOU-safe): a run that
//     transitioned to completed/cancelled between the snapshot and the write is
//     never overwritten (0 rows affected → no-op).
//   - Only `status=running` runs are considered (ListRunsByStatus), so terminal
//     runs (failed/completed/cancelled) are idempotently left alone.
func (o *OrphanReconciler) ReconcileOrphanedRuns(ctx context.Context) error {
	containers, err := o.containerMgr.ListRunningContainers(ctx, map[string]string{
		"managed_by": "hopeitworks",
	})
	if err != nil {
		// Proof of absence requires a successful listing of RUNNING containers. On
		// error we cannot tell a dead run from an unreachable runtime, so we touch
		// nothing — abort before querying DB runs.
		o.logger.Error("orphan reconcile aborted: failed to list running containers", "error", err)
		return err
	}

	liveContainerIDs := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		if container.ID != "" {
			liveContainerIDs[container.ID] = struct{}{}
		}
	}

	runningRuns, err := o.runRepo.ListRunsByStatus(ctx, model.RunStatusRunning)
	if err != nil {
		o.logger.Error("orphan reconcile aborted: failed to list running runs", "error", err)
		return err
	}

	now := time.Now()
	reconciled := 0
	for _, run := range runningRuns {
		orphaned, err := o.isOrphaned(ctx, run, liveContainerIDs, now)
		if err != nil {
			o.logger.Error("failed to evaluate run for reconciliation", "run_id", run.ID, "error", err)
			continue
		}
		if !orphaned {
			continue
		}

		// TOCTOU-safe: only flip the run if it is STILL running. A run that
		// completed/cancelled between the snapshot above and this write is not
		// overwritten (affected == false).
		affected, err := o.runRepo.MarkRunOrphanedIfRunning(ctx, run.ID, now, orphanedNoContainerReason)
		if err != nil {
			o.logger.Error("failed to reconcile orphaned run", "run_id", run.ID, "error", err)
			continue
		}
		if !affected {
			// Run already transitioned to a terminal state concurrently — no-op.
			continue
		}
		o.logger.Warn("reconciled orphaned run (no live container for active step)",
			"run_id", run.ID,
			"reason", orphanedNoContainerReason,
		)
		reconciled++
	}

	o.logger.Info("orphan run reconciliation completed", "runs_reconciled", reconciled)
	return nil
}

// isOrphaned reports whether a `running` run has an active (running) step that
// launched a container which is no longer running, past the grace window. It is
// the sole place the orphan invariant is decided.
//
// Returns false (NOT orphaned) when the run has no running step, when the
// running step launched no container (nil container_id), when the step is still
// within the grace window, or when the step's container is still running.
func (o *OrphanReconciler) isOrphaned(
	ctx context.Context,
	run *model.Run,
	liveContainerIDs map[string]struct{},
	now time.Time,
) (bool, error) {
	steps, err := o.runRepo.ListRunStepsByRun(ctx, run.ID)
	if err != nil {
		return false, err
	}

	for _, step := range steps {
		if step.Status != model.StepStatusRunning {
			continue
		}
		// Steps without a launched container (ci_poll/hitl_gate/git_*, or a step
		// not yet started) are legitimately long-lived running states.
		if step.ContainerID == nil || *step.ContainerID == "" {
			continue
		}
		// Grace is relative to the STEP's start: covers the launch→persist window.
		if step.StartedAt == nil || now.Sub(*step.StartedAt) < o.graceWindow {
			continue
		}
		if _, alive := liveContainerIDs[*step.ContainerID]; alive {
			continue
		}
		// Running step, container was launched, past grace, container not running.
		return true, nil
	}

	return false, nil
}
