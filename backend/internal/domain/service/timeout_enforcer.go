package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// TimeoutEnforcer monitors running containers and enforces timeout limits.
// Containers exceeding their configured timeout (project-specific or default)
// are stopped and their associated run steps and runs are marked as failed.
type TimeoutEnforcer struct {
	containerMgr   port.ContainerManager
	runRepo        port.RunRepository
	projectRepo    port.ProjectRepository
	logger         *slog.Logger
	defaultTimeout time.Duration
	checkInterval  time.Duration
}

// NewTimeoutEnforcer creates a new TimeoutEnforcer.
// defaultTimeout is the maximum time a container may run before being stopped (default 30 minutes).
// checkInterval is how often the enforcer checks for timed-out containers (default 30 seconds).
func NewTimeoutEnforcer(
	containerMgr port.ContainerManager,
	runRepo port.RunRepository,
	projectRepo port.ProjectRepository,
	logger *slog.Logger,
	defaultTimeout time.Duration,
	checkInterval time.Duration,
) *TimeoutEnforcer {
	return &TimeoutEnforcer{
		containerMgr:   containerMgr,
		runRepo:        runRepo,
		projectRepo:    projectRepo,
		logger:         logger,
		defaultTimeout: defaultTimeout,
		checkInterval:  checkInterval,
	}
}

// Start begins monitoring all active containers in a background loop.
// It checks at the configured interval for containers exceeding their timeout.
// Start blocks until the context is cancelled.
func (t *TimeoutEnforcer) Start(ctx context.Context) error {
	t.logger.Info("timeout enforcer started",
		"default_timeout", t.defaultTimeout,
		"check_interval", t.checkInterval,
	)

	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("timeout enforcer stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := t.CheckTimeouts(ctx); err != nil {
				t.logger.Error("timeout check failed", "error", err)
			}
		}
	}
}

// CheckTimeouts iterates active containers and enforces timeouts.
// Containers that have run longer than their allowed timeout are stopped,
// and their associated run steps and runs are marked as failed.
func (t *TimeoutEnforcer) CheckTimeouts(ctx context.Context) error {
	containers, err := t.containerMgr.ListContainers(ctx, map[string]string{
		"managed_by": "hopeitworks",
	})
	if err != nil {
		return err
	}

	for _, container := range containers {
		runIDStr := container.Labels["run_id"]
		stepIDStr := container.Labels["step_id"]

		if runIDStr == "" || stepIDStr == "" {
			continue
		}

		runID, err := uuid.Parse(runIDStr)
		if err != nil {
			t.logger.Warn("invalid run_id label", "run_id", runIDStr, "container_id", container.ID)
			continue
		}

		stepID, err := uuid.Parse(stepIDStr)
		if err != nil {
			t.logger.Warn("invalid step_id label", "step_id", stepIDStr, "container_id", container.ID)
			continue
		}

		runStep, err := t.runRepo.GetRunStep(ctx, stepID)
		if err != nil {
			t.logger.Warn("failed to fetch run step", "step_id", stepID, "error", err)
			continue
		}

		if runStep.StartedAt == nil {
			continue
		}

		timeout := t.getTimeout(ctx, runID)

		elapsed := time.Since(*runStep.StartedAt)
		if elapsed <= timeout {
			continue
		}

		t.logger.Warn("container timeout exceeded",
			"container_id", container.ID,
			"run_id", runID,
			"step_id", stepID,
			"elapsed", elapsed,
			"timeout", timeout,
		)

		if err := t.containerMgr.Stop(ctx, container.ID); err != nil {
			t.logger.Error("failed to stop timed-out container", "container_id", container.ID, "error", err)
			continue
		}

		now := time.Now()
		errMsg := "container_timeout"
		if _, err := t.runRepo.UpdateRunStepStatus(ctx, stepID, model.StepStatusFailed, nil, &now, &errMsg); err != nil {
			t.logger.Error("failed to update run step status", "step_id", stepID, "error", err)
		}

		runErrMsg := "container_timeout"
		if _, err := t.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusFailed, nil, &now, &runErrMsg); err != nil {
			t.logger.Error("failed to update run status", "run_id", runID, "error", err)
		}

		t.logger.Info("container stopped due to timeout",
			"container_id", container.ID,
			"run_id", runID,
			"step_id", stepID,
		)
	}

	return nil
}

// getTimeout returns the project-specific timeout if configured, otherwise the default.
func (t *TimeoutEnforcer) getTimeout(ctx context.Context, runID uuid.UUID) time.Duration {
	run, err := t.runRepo.GetRun(ctx, runID)
	if err != nil {
		return t.defaultTimeout
	}

	project, err := t.projectRepo.GetByID(ctx, run.ProjectID)
	if err != nil {
		return t.defaultTimeout
	}

	if project.MaxContainerTimeout != nil && *project.MaxContainerTimeout > 0 {
		t.logger.Debug("using project-specific timeout",
			"project_id", project.ID,
			"timeout", *project.MaxContainerTimeout,
		)
		return *project.MaxContainerTimeout
	}

	return t.defaultTimeout
}
