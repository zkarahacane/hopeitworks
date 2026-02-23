package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// ErrRunPaused is returned when a run is paused mid-execution.
var ErrRunPaused = fmt.Errorf("run paused")

// actionTypeToTemplateName maps pipeline config action_type values to prompt template names.
// Includes both new action types and legacy aliases for backward compatibility.
var actionTypeToTemplateName = map[string]string{
	"agent_run":  TemplateNameImplement,
	"implement":  TemplateNameImplement,
	"review":     TemplateNameReview,
	"merge":      TemplateNameMerge,
}

// errStepSuspended is returned by executeStep when the step transitions
// to waiting_approval. It is NOT a failure — it signals the executor to
// stop processing further steps without marking the run as failed.
var errStepSuspended = errors.New("step suspended for approval")

// PipelineExecutor orchestrates sequential execution of pipeline steps.
type PipelineExecutor struct {
	runRepo        port.RunRepository
	storyRepo      port.StoryRepository
	actionReg      port.ActionRegistry
	eventPub       port.EventPublisher
	circuitBreaker *CircuitBreakerService
	logger         *slog.Logger
}

// NewPipelineExecutor creates a new pipeline executor.
func NewPipelineExecutor(
	runRepo port.RunRepository,
	storyRepo port.StoryRepository,
	actionReg port.ActionRegistry,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *PipelineExecutor {
	return &PipelineExecutor{
		runRepo:   runRepo,
		storyRepo: storyRepo,
		actionReg: actionReg,
		eventPub:  eventPub,
		logger:    logger,
	}
}

// SetCircuitBreaker configures the circuit breaker service for the executor.
func (e *PipelineExecutor) SetCircuitBreaker(cb *CircuitBreakerService) {
	e.circuitBreaker = cb
}

// ExecuteRun executes all steps of a run sequentially.
// Steps execute in step_order sequence. Execution stops on first failure, cancellation, or pause.
// The circuit breaker is checked before execution and updated after completion/failure.
func (e *PipelineExecutor) ExecuteRun(ctx context.Context, runID uuid.UUID) error {
	// 1. Verify run exists
	run, err := e.runRepo.GetRun(ctx, runID)
	if err != nil {
		return err
	}

	// 2. Check circuit breaker before starting
	if e.circuitBreaker != nil {
		if err := e.circuitBreaker.CheckCircuitBreaker(ctx, run.ProjectID); err != nil {
			e.logger.Warn("circuit breaker open, aborting run",
				"run_id", runID,
				"project_id", run.ProjectID,
			)
			errMsg := err.Error()
			failedAt := time.Now()
			if _, updateErr := e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusFailed, nil, &failedAt, nil, &errMsg); updateErr != nil {
				e.logger.Error("failed to update run status after circuit breaker check", "error", updateErr)
			}
			return err
		}
	}

	steps, err := e.runRepo.ListRunStepsByRun(ctx, runID)
	if err != nil {
		return err
	}

	// Sort steps by step_order
	slices.SortFunc(steps, func(a, b *model.RunStep) int {
		return cmp.Compare(a.StepOrder, b.StepOrder)
	})

	// 3. Transition run to "running", publish run.started
	now := time.Now()
	run, err = e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusRunning, &now, nil, nil, nil)
	if err != nil {
		return err
	}

	e.publishEvent(ctx, run.ProjectID, "run", run.ID, "started", map[string]any{
		"run_id":     runID.String(),
		"status":     string(model.RunStatusRunning),
		"started_at": now.Format(time.RFC3339),
	})

	// Transition story to "running"
	e.updateStoryStatus(ctx, run, model.StoryStatusRunning)

	// 4. Merge persisted run metadata into the shared metadata map.
	// This carries branch_name and per-step model keys set by LaunchRun.
	metadata := make(map[string]any)
	for k, v := range run.Metadata {
		metadata[k] = v
	}

	for i := range steps {
		step := steps[i]

		// Skip already-completed steps (supports resume from paused state)
		if step.Status == model.StepStatusCompleted {
			continue
		}

		// Check for cancellation before each step
		select {
		case <-ctx.Done():
			e.handleCancellation(run, step)
			return ctx.Err()
		default:
		}

		// Check if the run has been paused (re-read status from DB)
		currentRun, err := e.runRepo.GetRun(ctx, run.ID)
		if err != nil {
			e.logger.Error("failed to check run status for pause", "run_id", run.ID, "error", err)
		} else if currentRun.Status == model.RunStatusPaused {
			e.logger.Info("run paused, stopping execution", "run_id", run.ID)
			return ErrRunPaused
		}

		if err := e.executeStep(ctx, run, step, metadata); err != nil {
			if errors.Is(err, errStepSuspended) {
				e.logger.Info("pipeline step suspended for approval",
					"run_id", run.ID, "step_id", step.ID)
				return nil
			}
			e.handleStepFailure(ctx, run, step, err)
			// Record failure in circuit breaker
			if e.circuitBreaker != nil {
				if cbErr := e.circuitBreaker.RecordFailure(ctx, run.ProjectID); cbErr != nil {
					e.logger.Error("failed to record circuit breaker failure", "error", cbErr)
				}
			}
			return err
		}
	}

	// 5. All steps completed — mark run as completed
	completedAt := time.Now()
	if _, err := e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusCompleted, nil, &completedAt, nil, nil); err != nil {
		return err
	}

	e.publishEvent(ctx, run.ProjectID, "run", run.ID, "completed", map[string]any{
		"run_id":       runID.String(),
		"status":       string(model.RunStatusCompleted),
		"completed_at": completedAt.Format(time.RFC3339),
	})

	// Transition story to "done"
	e.updateStoryStatus(ctx, run, model.StoryStatusDone)

	// Reset circuit breaker count on success
	if e.circuitBreaker != nil {
		if cbErr := e.circuitBreaker.RecordSuccess(ctx, run.ProjectID); cbErr != nil {
			e.logger.Error("failed to record circuit breaker success", "error", cbErr)
		}
	}

	return nil
}

// executeStep executes a single pipeline step.
func (e *PipelineExecutor) executeStep(ctx context.Context, run *model.Run, step *model.RunStep, metadata map[string]any) error {
	// Transition step to "running"
	startedAt := time.Now()
	updatedStep, err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusRunning, &startedAt, nil, nil)
	if err != nil {
		return err
	}
	*step = *updatedStep

	e.publishEvent(ctx, run.ProjectID, "step", step.ID, "started", map[string]any{
		"run_id":     run.ID.String(),
		"step_id":    step.ID.String(),
		"step_name":  step.StepName,
		"action":     step.Action,
		"status":     string(model.StepStatusRunning),
		"started_at": startedAt.Format(time.RFC3339),
	})

	// Lookup action
	action, err := e.actionReg.Get(step.Action)
	if err != nil {
		return fmt.Errorf("action lookup failed for %q: %w", step.Action, err)
	}

	// Build run context
	runCtx := &model.RunContext{
		Run:       run,
		RunStep:   step,
		ProjectID: run.ProjectID,
		StoryID:   run.StoryID,
		Metadata:  metadata,
	}

	// Inject template_name based on action_type alias (implement, review, merge).
	// An explicit template_name already in metadata (e.g. from incremental retry) takes precedence.
	if _, exists := runCtx.Metadata["template_name"]; !exists {
		if tmplName, ok := actionTypeToTemplateName[step.Action]; ok {
			runCtx.Metadata["template_name"] = tmplName
		}
	}

	// Inject per-step model from run metadata (set by LaunchRun as step_<order>_model).
	// The model key is consumed per-step and removed after use to avoid leaking to the next step.
	modelKey := fmt.Sprintf("step_%d_model", step.StepOrder)
	if m, ok := runCtx.Metadata[modelKey].(string); ok && m != "" {
		runCtx.Metadata["model"] = m
	} else {
		delete(runCtx.Metadata, "model")
	}

	// Execute action
	if err := action.Execute(ctx, runCtx); err != nil {
		return err
	}

	// Re-fetch step to detect suspension (e.g., hitl_gate sets waiting_approval)
	refetchedStep, fetchErr := e.runRepo.GetRunStep(ctx, step.ID)
	if fetchErr != nil {
		e.logger.Warn("failed to re-fetch step after execute", "step_id", step.ID, "error", fetchErr)
	} else if refetchedStep.Status == model.StepStatusWaitingApproval {
		return errStepSuspended
	}

	// Transition step to "completed"
	completedAt := time.Now()
	updatedStep, err = e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCompleted, nil, &completedAt, nil)
	if err != nil {
		return err
	}
	*step = *updatedStep

	e.publishEvent(ctx, run.ProjectID, "step", step.ID, "completed", map[string]any{
		"run_id":       run.ID.String(),
		"step_id":      step.ID.String(),
		"step_name":    step.StepName,
		"status":       string(model.StepStatusCompleted),
		"completed_at": completedAt.Format(time.RFC3339),
	})

	return nil
}

// handleStepFailure marks step and run as failed, publishes events.
func (e *PipelineExecutor) handleStepFailure(ctx context.Context, run *model.Run, step *model.RunStep, stepErr error) {
	errMsg := stepErr.Error()
	failedAt := time.Now()

	// Mark step as failed
	if _, err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusFailed, nil, &failedAt, &errMsg); err != nil {
		e.logger.Error("failed to update step status to failed", "step_id", step.ID, "error", err)
	}

	e.publishEvent(ctx, run.ProjectID, "step", step.ID, "failed", map[string]any{
		"run_id":        run.ID.String(),
		"step_id":       step.ID.String(),
		"step_name":     step.StepName,
		"status":        string(model.StepStatusFailed),
		"error_message": errMsg,
	})

	// Mark run as failed
	if _, err := e.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusFailed, nil, &failedAt, nil, &errMsg); err != nil {
		e.logger.Error("failed to update run status to failed", "run_id", run.ID, "error", err)
	}

	e.publishEvent(ctx, run.ProjectID, "run", run.ID, "failed", map[string]any{
		"run_id":        run.ID.String(),
		"status":        string(model.RunStatusFailed),
		"error_message": errMsg,
	})

	// Transition story to "failed"
	e.updateStoryStatus(ctx, run, model.StoryStatusFailed)
}

// handleCancellation marks step and run as cancelled, publishes events.
// Uses a background context since the caller's context is already cancelled.
func (e *PipelineExecutor) handleCancellation(run *model.Run, step *model.RunStep) {
	cancelledAt := time.Now()
	cancelMsg := "execution cancelled"

	// Use a background context for cancellation cleanup since the original context is cancelled
	bgCtx := context.Background()

	// Mark step as cancelled
	if _, err := e.runRepo.UpdateRunStepStatus(bgCtx, step.ID, model.StepStatusCancelled, nil, &cancelledAt, &cancelMsg); err != nil {
		e.logger.Error("failed to update step status to cancelled", "step_id", step.ID, "error", err)
	}

	e.publishEvent(bgCtx, run.ProjectID, "step", step.ID, "cancelled", map[string]any{
		"run_id":    run.ID.String(),
		"step_id":   step.ID.String(),
		"step_name": step.StepName,
		"status":    string(model.StepStatusCancelled),
	})

	// Mark run as cancelled
	if _, err := e.runRepo.UpdateRunStatus(bgCtx, run.ID, model.RunStatusCancelled, nil, &cancelledAt, nil, &cancelMsg); err != nil {
		e.logger.Error("failed to update run status to cancelled", "run_id", run.ID, "error", err)
	}

	e.publishEvent(bgCtx, run.ProjectID, "run", run.ID, "cancelled", map[string]any{
		"run_id": run.ID.String(),
		"status": string(model.RunStatusCancelled),
	})
}

// updateStoryStatus fetches the story linked to the run and updates its status.
// Errors are logged without failing the run execution — story status is best-effort.
func (e *PipelineExecutor) updateStoryStatus(ctx context.Context, run *model.Run, status string) {
	if run.StoryID == uuid.Nil {
		return
	}

	story, err := e.storyRepo.GetByID(ctx, run.StoryID)
	if err != nil {
		e.logger.Error("failed to fetch story for status update",
			"story_id", run.StoryID,
			"run_id", run.ID,
			"target_status", status,
			"error", err,
		)
		return
	}

	story.Status = status
	if _, err := e.storyRepo.Update(ctx, story); err != nil {
		e.logger.Error("failed to update story status",
			"story_id", run.StoryID,
			"run_id", run.ID,
			"target_status", status,
			"error", err,
		)
		return
	}

	e.logger.Info("story status updated",
		"story_id", run.StoryID,
		"run_id", run.ID,
		"status", status,
	)

	e.publishEvent(ctx, run.ProjectID, "story", run.StoryID, "status_updated", map[string]any{
		"story_id": run.StoryID.String(),
		"run_id":   run.ID.String(),
		"status":   status,
	})
}

// publishEvent publishes an event, logging errors without failing execution.
func (e *PipelineExecutor) publishEvent(ctx context.Context, projectID uuid.UUID, entityType string, entityID uuid.UUID, action string, payload map[string]any) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		e.logger.Error("failed to marshal event payload", "entity_type", entityType, "action", action, "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Payload:    payloadJSON,
	}

	if err := e.eventPub.Publish(ctx, event); err != nil {
		e.logger.Error("failed to publish event", "event_type", event.EventName(), "error", err)
	}
}
