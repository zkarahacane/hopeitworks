package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// ParallelGroupExecutor processes DAG layers sequentially, with stories
// within each layer executing in parallel.
type ParallelGroupExecutor struct {
	epicRunRepo port.EpicRunRepository
	runSvc      *RunService
	executor    *PipelineExecutor
	eventPub    port.EventPublisher
	logger      *slog.Logger
}

// NewParallelGroupExecutor creates a new ParallelGroupExecutor.
func NewParallelGroupExecutor(
	epicRunRepo port.EpicRunRepository,
	runSvc *RunService,
	executor *PipelineExecutor,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *ParallelGroupExecutor {
	return &ParallelGroupExecutor{
		epicRunRepo: epicRunRepo,
		runSvc:      runSvc,
		executor:    executor,
		eventPub:    eventPub,
		logger:      logger,
	}
}

// Execute processes all DAG groups sequentially. Within each group, stories
// run in parallel using errgroup. If any story in a group fails, subsequent
// groups are not started (fail-fast).
func (e *ParallelGroupExecutor) Execute(ctx context.Context, epicRun *model.EpicRun, dag model.DAGResult) error {
	// Transition epic run to running
	if _, err := e.epicRunRepo.UpdateEpicRunStatus(ctx, epicRun.ID, model.EpicRunStatusRunning, nil); err != nil {
		return fmt.Errorf("failed to transition epic run to running: %w", err)
	}
	e.publishEvent(ctx, epicRun.ProjectID, "epic_run", epicRun.ID, "started", json.RawMessage(`{}`))

	// Process each DAG layer sequentially
	for groupIdx, group := range dag.Groups {
		e.logger.Info("starting DAG layer",
			"epic_run_id", epicRun.ID,
			"group_index", groupIdx,
			"story_count", len(group),
		)

		payload := json.RawMessage(fmt.Sprintf(`{"group_index":%d,"story_count":%d}`, groupIdx, len(group)))
		e.publishEvent(ctx, epicRun.ProjectID, "epic_run_group", epicRun.ID, "started", payload)

		eg, egCtx := errgroup.WithContext(ctx)
		for _, story := range group {
			story := story // capture loop var
			eg.Go(func() error {
				return e.runStory(egCtx, epicRun, story, groupIdx)
			})
		}

		if err := eg.Wait(); err != nil {
			// Fail-fast: mark epic run as failed
			e.logger.Error("DAG layer failed, aborting epic run",
				"epic_run_id", epicRun.ID,
				"group_index", groupIdx,
				"error", err,
			)
			now := time.Now()
			if _, updateErr := e.epicRunRepo.UpdateEpicRunStatus(ctx, epicRun.ID, model.EpicRunStatusFailed, &now); updateErr != nil {
				e.logger.Error("failed to mark epic run as failed", "epic_run_id", epicRun.ID, "error", updateErr)
			}
			e.publishEvent(ctx, epicRun.ProjectID, "epic_run", epicRun.ID, "failed", json.RawMessage(`{}`))
			return err
		}

		e.logger.Info("DAG layer completed",
			"epic_run_id", epicRun.ID,
			"group_index", groupIdx,
		)
	}

	// All layers completed successfully
	now := time.Now()
	if _, err := e.epicRunRepo.UpdateEpicRunStatus(ctx, epicRun.ID, model.EpicRunStatusCompleted, &now); err != nil {
		e.logger.Error("failed to mark epic run as completed", "epic_run_id", epicRun.ID, "error", err)
		return err
	}
	e.publishEvent(ctx, epicRun.ProjectID, "epic_run", epicRun.ID, "completed", json.RawMessage(`{}`))

	e.logger.Info("epic run completed successfully", "epic_run_id", epicRun.ID)
	return nil
}

// runStory creates a Run for a single story and executes it.
func (e *ParallelGroupExecutor) runStory(ctx context.Context, epicRun *model.EpicRun, story model.Story, groupIndex int) error {
	storyID := story.ID

	// Create a run for this story via LaunchRun (which creates run + steps + enqueues job).
	// Epic runs do not yet track a launching user, so uuid.Nil is passed as userID.
	run, err := e.runSvc.LaunchRun(ctx, epicRun.ProjectID, storyID, uuid.Nil)
	if err != nil {
		e.logger.Error("failed to create run for story",
			"epic_run_id", epicRun.ID,
			"story_id", storyID,
			"group_index", groupIndex,
			"error", err,
		)
		// Mark story as failed
		if updateErr := e.epicRunRepo.UpdateEpicRunStoryStatus(ctx, epicRun.ID, storyID, "failed", nil); updateErr != nil {
			e.logger.Error("failed to update epic run story status", "error", updateErr)
		}

		e.publishStoryCompleted(ctx, epicRun.ProjectID, epicRun.ID, storyID, uuid.Nil, "failed")
		return fmt.Errorf("story %s: failed to create run: %w", story.Key, err)
	}

	// Update epic_run_stories: set run_id, status running
	if updateErr := e.epicRunRepo.UpdateEpicRunStoryStatus(ctx, epicRun.ID, storyID, "running", &run.ID); updateErr != nil {
		e.logger.Error("failed to update epic run story status to running", "error", updateErr)
	}

	// Execute the run directly (bypasses job queue — we are already async)
	if err := e.executor.ExecuteRun(ctx, run.ID); err != nil {
		e.logger.Error("story run failed",
			"epic_run_id", epicRun.ID,
			"story_id", storyID,
			"run_id", run.ID,
			"group_index", groupIndex,
			"error", err,
		)
		// Mark story as failed
		if updateErr := e.epicRunRepo.UpdateEpicRunStoryStatus(ctx, epicRun.ID, storyID, "failed", &run.ID); updateErr != nil {
			e.logger.Error("failed to update epic run story status", "error", updateErr)
		}

		e.publishStoryCompleted(ctx, epicRun.ProjectID, epicRun.ID, storyID, run.ID, "failed")
		return fmt.Errorf("story %s: run %s failed: %w", story.Key, run.ID, err)
	}

	// Mark story as completed
	if updateErr := e.epicRunRepo.UpdateEpicRunStoryStatus(ctx, epicRun.ID, storyID, "completed", &run.ID); updateErr != nil {
		e.logger.Error("failed to update epic run story status to completed", "error", updateErr)
	}

	e.logger.Info("story run completed",
		"epic_run_id", epicRun.ID,
		"story_id", storyID,
		"run_id", run.ID,
		"group_index", groupIndex,
	)
	e.publishStoryCompleted(ctx, epicRun.ProjectID, epicRun.ID, storyID, run.ID, "completed")
	return nil
}

// publishEvent publishes an event, logging errors but not aborting execution.
func (e *ParallelGroupExecutor) publishEvent(ctx context.Context, projectID uuid.UUID, entityType string, entityID uuid.UUID, action string, payload json.RawMessage) {
	event := model.Event{
		ProjectID:  projectID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Payload:    payload,
	}
	if err := e.eventPub.Publish(ctx, event); err != nil {
		e.logger.Error("failed to publish event",
			"entity_type", entityType,
			"entity_id", entityID,
			"action", action,
			"error", err,
		)
	}
}

// publishStoryCompleted publishes an epic_run.story.completed event.
func (e *ParallelGroupExecutor) publishStoryCompleted(ctx context.Context, projectID, epicRunID, storyID, runID uuid.UUID, status string) {
	payload := json.RawMessage(fmt.Sprintf(`{"story_id":%q,"run_id":%q,"status":%q}`, storyID, runID, status))
	e.publishEvent(ctx, projectID, "epic_run", epicRunID, "story.completed", payload)
}
