package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// OrphanCleaner removes containers that are not associated with active runs.
// It is designed to be run once during API startup to clean up containers
// left behind by previous crashes or unexpected shutdowns.
type OrphanCleaner struct {
	containerMgr port.ContainerManager
	runRepo      port.RunRepository
	logger       *slog.Logger
}

// NewOrphanCleaner creates a new OrphanCleaner.
func NewOrphanCleaner(
	containerMgr port.ContainerManager,
	runRepo port.RunRepository,
	logger *slog.Logger,
) *OrphanCleaner {
	return &OrphanCleaner{
		containerMgr: containerMgr,
		runRepo:      runRepo,
		logger:       logger,
	}
}

// Substrate scope (ADR Stage 4): this reaper is Docker-shaped — it lists managed
// executions via ContainerManager.ListContainers(managed_by=hopeitworks) and reads
// the run_id label. DockerRuntime.Launch preserves those labels and persists the
// real container id, so the Docker reaping contract holds through the substrate
// dispatch with zero change. Reaping a NON-Docker substrate (listing managed
// microVMs/pods) needs a runtime-level "list managed executions" capability and is
// DEFERRED until a non-Docker substrate ships live (ADR Stage 4 / Decision §7#4).
// CancelRun already stops via port.AgentRuntime (substrate-correct); orphan/timeout
// listing remains Docker-bound for now.
//
// CleanupOrphans removes containers not associated with active runs.
// A container is considered an orphan if:
//   - It has no run_id label
//   - Its run_id does not correspond to an existing run
//   - Its associated run is not in an active state (running, pending)
//
// Cleanup continues even if individual removals fail.
func (o *OrphanCleaner) CleanupOrphans(ctx context.Context) error {
	containers, err := o.containerMgr.ListContainers(ctx, map[string]string{
		"managed_by": "hopeitworks",
	})
	if err != nil {
		return err
	}

	orphanCount := 0

	for _, container := range containers {
		runIDStr := container.Labels["run_id"]

		if runIDStr == "" {
			o.removeOrphan(ctx, container.ID, "no_run_id_label")
			orphanCount++
			continue
		}

		runID, err := uuid.Parse(runIDStr)
		if err != nil {
			o.removeOrphan(ctx, container.ID, "invalid_run_id")
			orphanCount++
			continue
		}

		run, err := o.runRepo.GetRun(ctx, runID)
		if err != nil {
			o.removeOrphan(ctx, container.ID, "run_not_found")
			orphanCount++
			continue
		}

		if run.Status != model.RunStatusRunning && run.Status != model.RunStatusPending {
			o.removeOrphan(ctx, container.ID, "run_not_active")
			orphanCount++
			continue
		}
	}

	o.logger.Info("orphan cleanup completed", "orphans_removed", orphanCount)
	return nil
}

// removeOrphan removes a single orphan container, logging success or failure.
func (o *OrphanCleaner) removeOrphan(ctx context.Context, containerID, reason string) {
	if err := o.containerMgr.Remove(ctx, containerID); err != nil {
		o.logger.Error("failed to remove orphan container",
			"container_id", containerID,
			"reason", reason,
			"error", err,
		)
	} else {
		o.logger.Info("removed orphan container",
			"container_id", containerID,
			"reason", reason,
		)
	}
}
