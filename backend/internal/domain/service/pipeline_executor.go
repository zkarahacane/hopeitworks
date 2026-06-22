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

// errStepSuspended is returned by executeStep when the step transitions
// to waiting_approval. It is NOT a failure — it signals the executor to
// stop processing further steps without marking the run as failed.
var errStepSuspended = errors.New("step suspended for approval")

// errStageSuspended is returned at a stage boundary when the run is parked by a
// transition policy (a not-yet-started manual stage, or an unapproved gate). Like
// errStepSuspended it is NOT a failure — the executor stops cleanly without marking
// the run failed; a manual "Go" or a gate approval re-enqueues it.
var errStageSuspended = errors.New("stage suspended pending transition")

// metaStageStartedPrefix keys, in run metadata, the manual stages that have already
// been triggered ("Go"). Suffixed with the stage id: stage_started_<stageID> = true.
const metaStageStartedPrefix = "stage_started_"

// pausedReasonManualStart is recorded in run metadata ("paused_reason") when the run
// is parked at the entry of a not-yet-started manual stage. It lets the resume path
// (stage/start) distinguish a manual-start pause from a generic user pause.
const pausedReasonManualStart = "awaiting_manual_start"

// stageStartedKey is the run-metadata key recording that a manual stage was started.
func stageStartedKey(stageID string) string { return metaStageStartedPrefix + stageID }

// PipelineExecutor orchestrates sequential execution of pipeline steps.
type PipelineExecutor struct {
	runRepo        port.RunRepository
	storyRepo      port.StoryRepository
	actionReg      port.ActionRegistry
	eventPub       port.EventPublisher
	circuitBreaker *CircuitBreakerService
	hitlRepo       port.HITLRepository
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

// SetHITLRepo configures the HITL repository used to enforce gate-transition stages.
// Optional: when unset, a stage's "gate" transition degrades to "auto" (no approval
// gate is raised at the boundary). All non-gate behaviour is unaffected.
func (e *PipelineExecutor) SetHITLRepo(repo port.HITLRepository) {
	e.hitlRepo = repo
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

	// Terminal guard: if run is already in a terminal state (failed, completed, cancelled),
	// skip re-execution and return success so River doesn't retry.
	// paused is NOT terminal and should be resumed.
	if run.Status == model.RunStatusFailed || run.Status == model.RunStatusCompleted || run.Status == model.RunStatusCancelled {
		e.logger.Info("run already terminal, skipping re-execution",
			"run_id", runID,
			"status", string(run.Status),
		)
		return nil
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
		"story_id":   run.StoryID.String(),
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

	// Parse the run's config snapshot once so stage-boundary transition policies
	// (manual/gate) can be resolved by stage id. A nil parse degrades every stage
	// to "auto", preserving the pre-INC-3 always-advancing behaviour.
	policy := e.parseTransitionPolicy(run)

	// curStage tracks the stage the card is currently in. It is nil until the first
	// step is entered. On resume (skipped completed steps) it stays nil until the
	// first not-yet-completed step runs, which re-establishes current_stage even if
	// the orchestrator pod restarted mid-run.
	var curStage *stageRef
	// prevStep is the last step processed in curStage; it anchors a gate HITL raised
	// when that stage's segment completes. nil until the first step runs.
	var prevStep *model.RunStep

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

		// Stage boundary: entering this step's stage (the first step enters its
		// stage; a step whose stage differs from the current one crosses a boundary).
		// On a boundary we exit the previous stage and enter the new one, advancing
		// stories.current_stage and emitting stage.exited/stage.entered events. The
		// transition policy of both stages is enforced here: a "gate" on the stage we
		// are leaving parks the run for approval; a not-yet-started "manual" on the
		// stage we are entering parks the run for a "Go".
		if !curStage.matches(step.StageID) {
			next := newStageRef(step)
			// Enforce the exiting stage's gate before advancing.
			if susErr := e.enforceGateOnExit(ctx, run, curStage, policy, prevStep); susErr != nil {
				if errors.Is(susErr, errStageSuspended) {
					return nil
				}
				return susErr
			}
			// Enforce the entering stage's manual policy before running its segment.
			if susErr := e.enforceManualOnEnter(ctx, run, next, policy, metadata); susErr != nil {
				if errors.Is(susErr, errStageSuspended) {
					return nil
				}
				return susErr
			}
			e.exitStage(ctx, run, curStage, next)
			e.enterStage(ctx, run, next)
			curStage = next
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
		prevStep = step
	}

	// 5. All steps completed — enforce the final stage's gate, then exit it and
	// clear current_stage. A gate on the last stage parks the run for approval
	// before it is marked completed.
	if curStage != nil {
		if susErr := e.enforceGateOnExit(ctx, run, curStage, policy, prevStep); susErr != nil {
			if errors.Is(susErr, errStageSuspended) {
				return nil
			}
			return susErr
		}
		e.exitStage(ctx, run, curStage, nil)
		e.clearStoryCurrentStage(ctx, run)
	}

	// Mark run as completed
	completedAt := time.Now()
	if _, err := e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusCompleted, nil, &completedAt, nil, nil); err != nil {
		return err
	}

	e.publishEvent(ctx, run.ProjectID, "run", run.ID, "completed", map[string]any{
		"run_id":       runID.String(),
		"story_id":     run.StoryID.String(),
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
		"story_id":   run.StoryID.String(),
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

	// Inject per-step config from the pipeline config snapshot into step.Config.
	// This makes step-specific config (e.g., branch_pattern, base_branch) available
	// to actions via runCtx.RunStep.Config without parsing the snapshot themselves.
	if step.Config == nil {
		step.Config = e.extractStepConfig(run.PipelineConfigSnapshot, step.StepOrder)
	}

	// Build run context
	runCtx := &model.RunContext{
		Run:       run,
		RunStep:   step,
		ProjectID: run.ProjectID,
		StoryID:   run.StoryID,
		Metadata:  metadata,
	}

	// Extract the launching user ID from run metadata so actions can resolve
	// user-specific API keys for agent containers.
	if userIDStr, ok := metadata["launched_by_user_id"].(string); ok && userIDStr != "" {
		if parsedID, parseErr := uuid.Parse(userIDStr); parseErr == nil {
			runCtx.UserID = parsedID
		}
	}

	// Inject per-step template_content from run metadata (set by LaunchRun as step_<order>_template_content).
	templateContentKey := fmt.Sprintf("step_%d_template_content", step.StepOrder)
	if tc, ok := runCtx.Metadata[templateContentKey].(string); ok && tc != "" {
		runCtx.Metadata["template_content"] = tc
	} else {
		delete(runCtx.Metadata, "template_content")
	}

	// Inject per-step model from run metadata (set by LaunchRun as step_<order>_model).
	// The model key is consumed per-step and removed after use to avoid leaking to the next step.
	modelKey := fmt.Sprintf("step_%d_model", step.StepOrder)
	if m, ok := runCtx.Metadata[modelKey].(string); ok && m != "" {
		runCtx.Metadata["model"] = m
	} else {
		delete(runCtx.Metadata, "model")
	}

	// Inject per-step agent_id and agent_image from run metadata when set by LaunchRun.
	agentIDKey := fmt.Sprintf("step_%d_agent_id", step.StepOrder)
	if agentID, ok := runCtx.Metadata[agentIDKey].(string); ok && agentID != "" {
		runCtx.Metadata["agent_id"] = agentID
	} else {
		delete(runCtx.Metadata, "agent_id")
	}

	agentImageKey := fmt.Sprintf("step_%d_agent_image", step.StepOrder)
	if agentImage, ok := runCtx.Metadata[agentImageKey].(string); ok && agentImage != "" {
		runCtx.Metadata["agent_image"] = agentImage
	} else {
		delete(runCtx.Metadata, "agent_image")
	}

	runtimeKindKey := fmt.Sprintf("step_%d_runtime_kind", step.StepOrder)
	if runtimeKind, ok := runCtx.Metadata[runtimeKindKey].(string); ok && runtimeKind != "" {
		runCtx.Metadata["runtime_kind"] = runtimeKind
	} else {
		delete(runCtx.Metadata, "runtime_kind")
	}

	// Execute action
	if err := action.Execute(ctx, runCtx); err != nil {
		return err
	}

	// Persist metadata mutations made by the action (e.g. branch_name set by git_branch)
	// so they survive a HITL suspend/resume where a fresh ExecuteRun reads from the DB.
	// Non-fatal: log and continue if the write fails.
	if persistErr := e.runRepo.UpdateRunMetadata(ctx, run.ID, metadata); persistErr != nil {
		e.logger.Warn("failed to persist run metadata after step",
			"run_id", run.ID,
			"step_id", step.ID,
			"error", persistErr,
		)
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
		"story_id":     run.StoryID.String(),
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
		"story_id":      run.StoryID.String(),
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
		"story_id":      run.StoryID.String(),
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
		"story_id":  run.StoryID.String(),
		"step_id":   step.ID.String(),
		"step_name": step.StepName,
		"status":    string(model.StepStatusCancelled),
	})

	// Mark run as cancelled
	if _, err := e.runRepo.UpdateRunStatus(bgCtx, run.ID, model.RunStatusCancelled, nil, &cancelledAt, nil, &cancelMsg); err != nil {
		e.logger.Error("failed to update run status to cancelled", "run_id", run.ID, "error", err)
	}

	e.publishEvent(bgCtx, run.ProjectID, "run", run.ID, "cancelled", map[string]any{
		"run_id":   run.ID.String(),
		"story_id": run.StoryID.String(),
		"status":   string(model.RunStatusCancelled),
	})

	// Transition story back to "backlog" so it can be relaunched (a cancelled run
	// must not leave the story stuck in "running" forever).
	e.updateStoryStatus(bgCtx, run, model.StoryStatusBacklog)
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

// stageRef is the in-memory identity of the stage a run is currently in.
type stageRef struct {
	id   string
	name string
}

// newStageRef builds a stageRef from a run step's stamped stage identity.
func newStageRef(step *model.RunStep) *stageRef {
	return &stageRef{id: step.StageID, name: step.StageName}
}

// matches reports whether the receiver represents the same stage as stageID.
// A nil receiver (no stage entered yet) matches nothing, so the first step always
// crosses a boundary into its stage.
func (s *stageRef) matches(stageID string) bool {
	return s != nil && s.id == stageID
}

// stageEntityID derives a stable UUID for a (run, stage) pair so stage events
// carry a consistent entity_id across entered/exited without a real stage row.
func stageEntityID(runID uuid.UUID, stageID string) uuid.UUID {
	return uuid.NewSHA1(runID, []byte("stage:"+stageID))
}

// enterStage advances stories.current_stage to the stage's name and emits a
// stage.entered event. Best-effort: failures are logged, never fatal.
func (e *PipelineExecutor) enterStage(ctx context.Context, run *model.Run, stage *stageRef) {
	if stage == nil {
		return
	}
	if run.StoryID != uuid.Nil {
		name := stage.name
		if _, err := e.storyRepo.UpdateStoryCurrentStage(ctx, run.StoryID, &name); err != nil {
			e.logger.Error("failed to advance story current_stage",
				"story_id", run.StoryID, "run_id", run.ID, "stage_id", stage.id, "error", err)
		}
	}
	e.publishEvent(ctx, run.ProjectID, "stage", stageEntityID(run.ID, stage.id), "entered", map[string]any{
		"stage_id":   stage.id,
		"stage_name": stage.name,
		"story_id":   run.StoryID.String(),
		"run_id":     run.ID.String(),
	})
}

// exitStage emits a stage.exited event for the stage being left. prev is the stage
// we are leaving (no-op if nil); next is the stage we are entering (nil at run end),
// included in the payload for board projection. Best-effort.
func (e *PipelineExecutor) exitStage(ctx context.Context, run *model.Run, prev, next *stageRef) {
	if prev == nil {
		return
	}
	payload := map[string]any{
		"stage_id":   prev.id,
		"stage_name": prev.name,
		"story_id":   run.StoryID.String(),
		"run_id":     run.ID.String(),
	}
	if next != nil {
		payload["next_stage_id"] = next.id
		payload["next_stage_name"] = next.name
	}
	e.publishEvent(ctx, run.ProjectID, "stage", stageEntityID(run.ID, prev.id), "exited", payload)
}

// clearStoryCurrentStage sets stories.current_stage to NULL at run completion.
// Best-effort: failures are logged, never fatal.
func (e *PipelineExecutor) clearStoryCurrentStage(ctx context.Context, run *model.Run) {
	if run.StoryID == uuid.Nil {
		return
	}
	if _, err := e.storyRepo.UpdateStoryCurrentStage(ctx, run.StoryID, nil); err != nil {
		e.logger.Error("failed to clear story current_stage",
			"story_id", run.StoryID, "run_id", run.ID, "error", err)
	}
}

// parseTransitionPolicy parses the run's config snapshot into a stage-transition
// resolver. Returns nil when the snapshot is empty or unparseable, in which case
// callers treat every stage as "auto" (the pre-INC-3 behaviour).
func (e *PipelineExecutor) parseTransitionPolicy(run *model.Run) *model.PipelineConfigYAML {
	if len(run.PipelineConfigSnapshot) == 0 {
		return nil
	}
	var parsed model.PipelineConfigYAML
	if err := json.Unmarshal(run.PipelineConfigSnapshot, &parsed); err != nil {
		e.logger.Warn("failed to parse config snapshot for transition policy; treating stages as auto",
			"run_id", run.ID, "error", err)
		return nil
	}
	return &parsed
}

// transitionFor resolves a stage's exit policy, defaulting to "auto" when the
// policy is nil (unparseable snapshot) or the stage is unknown.
func transitionFor(policy *model.PipelineConfigYAML, stageID string) string {
	if policy == nil {
		return model.TransitionAuto
	}
	return policy.TransitionForStage(stageID)
}

// enforceManualOnEnter parks the run before entering a "manual" stage that has not
// yet been triggered by a "Go" (its stage_started_<id> metadata flag is unset).
// Returns errStageSuspended when the run is parked; nil to proceed. The run is paused
// via the existing pause mechanism — NOT a HITL request — and the paused reason is
// recorded in metadata so the stage/start path knows to set the flag and resume.
func (e *PipelineExecutor) enforceManualOnEnter(
	ctx context.Context, run *model.Run, next *stageRef,
	policy *model.PipelineConfigYAML, metadata map[string]any,
) error {
	if next == nil || transitionFor(policy, next.id) != model.TransitionManual {
		return nil
	}
	if started, ok := metadata[stageStartedKey(next.id)].(bool); ok && started {
		// Already triggered (this is a resume after "Go") — proceed into the segment.
		return nil
	}

	// Record the awaiting-manual-start reason in run metadata so the resume path can
	// distinguish a manual-start pause from a generic pause, and so the board can show
	// why the card is parked. Best-effort: a failed write still parks the run.
	metadata["paused_reason"] = pausedReasonManualStart
	metadata["paused_stage_id"] = next.id
	metadata["paused_stage_name"] = next.name
	if persistErr := e.runRepo.UpdateRunMetadata(ctx, run.ID, metadata); persistErr != nil {
		e.logger.Warn("failed to persist manual-start pause reason",
			"run_id", run.ID, "stage_id", next.id, "error", persistErr)
	}

	now := time.Now()
	if _, err := e.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusPaused, nil, nil, &now, nil); err != nil {
		return fmt.Errorf("pause run for manual stage %q: %w", next.id, err)
	}

	// Advance current_stage so the card visibly sits idle IN the manual stage (the
	// Kanban intuition: card is in the column, work not started).
	if run.StoryID != uuid.Nil {
		name := next.name
		if _, err := e.storyRepo.UpdateStoryCurrentStage(ctx, run.StoryID, &name); err != nil {
			e.logger.Error("failed to advance story current_stage for manual stage",
				"story_id", run.StoryID, "run_id", run.ID, "stage_id", next.id, "error", err)
		}
	}

	e.publishEvent(ctx, run.ProjectID, "stage", stageEntityID(run.ID, next.id), "awaiting_start", map[string]any{
		"stage_id":   next.id,
		"stage_name": next.name,
		"story_id":   run.StoryID.String(),
		"run_id":     run.ID.String(),
		"transition": model.TransitionManual,
	})
	e.logger.Info("run parked awaiting manual stage start",
		"run_id", run.ID, "stage_id", next.id, "stage_name", next.name)
	return errStageSuspended
}

// enforceGateOnExit parks the run for human approval when the stage being left has a
// "gate" transition and its segment has just completed. It reuses the existing HITL
// approval gate: a HITL request is anchored to the stage's last step, the step is set
// to waiting_approval, and the run is paused — exactly the state hitl_service.Approve
// expects, so approval resumes the run unchanged. Returns errStageSuspended when the
// run is parked; nil to proceed (auto/manual stages, an already-approved gate, or no
// HITL repository wired).
func (e *PipelineExecutor) enforceGateOnExit(
	ctx context.Context, run *model.Run, leaving *stageRef,
	policy *model.PipelineConfigYAML, anchor *model.RunStep,
) error {
	if leaving == nil || transitionFor(policy, leaving.id) != model.TransitionGate {
		return nil
	}
	if e.hitlRepo == nil || anchor == nil {
		// Without a HITL repo (or an anchor step) a gate cannot be raised; degrade to
		// auto so the pipeline still advances rather than wedging.
		return nil
	}

	// Idempotency across resume: if a HITL already exists for the anchor step, the gate
	// has been raised before. Approved → proceed; otherwise the run stays parked.
	if existing, err := e.hitlRepo.GetByRunStepID(ctx, anchor.ID); err == nil && existing != nil {
		if existing.Status == model.HITLStatusApproved {
			return nil
		}
		return errStageSuspended
	}

	req := &model.HITLRequest{
		ID:        uuid.New(),
		RunStepID: anchor.ID,
		GateType:  "approval",
		Status:    model.HITLStatusPending,
		CreatedAt: time.Now(),
	}
	if _, err := e.hitlRepo.Create(ctx, req); err != nil {
		return fmt.Errorf("create gate HITL request for stage %q: %w", leaving.id, err)
	}

	now := time.Now()
	if _, err := e.runRepo.UpdateRunStepStatus(ctx, anchor.ID, model.StepStatusWaitingApproval, &now, nil, nil); err != nil {
		return fmt.Errorf("set anchor step waiting_approval for gate stage %q: %w", leaving.id, err)
	}
	if _, err := e.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusPaused, nil, nil, &now, nil); err != nil {
		return fmt.Errorf("pause run for gate stage %q: %w", leaving.id, err)
	}

	e.publishEvent(ctx, run.ProjectID, "hitl_gate", req.ID, "pending", map[string]any{
		"run_id":          run.ID.String(),
		"story_id":        run.StoryID.String(),
		"step_id":         anchor.ID.String(),
		"hitl_request_id": req.ID.String(),
		"stage_id":        leaving.id,
		"stage_name":      leaving.name,
	})
	e.logger.Info("run parked at gate awaiting approval",
		"run_id", run.ID, "stage_id", leaving.id, "hitl_request_id", req.ID)
	return errStageSuspended
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

// extractStepConfig parses the pipeline config snapshot and returns the Config
// map for the step at the given order index. Returns nil if parsing fails or
// the step has no config.
func (e *PipelineExecutor) extractStepConfig(snapshot json.RawMessage, stepOrder int) map[string]string {
	if len(snapshot) == 0 {
		return nil
	}
	var parsed model.PipelineConfigYAML
	if err := json.Unmarshal(snapshot, &parsed); err != nil {
		e.logger.Warn("failed to parse pipeline config snapshot for step config",
			"step_order", stepOrder, "error", err)
		return nil
	}
	flatSteps := parsed.FlatSteps()
	if stepOrder < 0 || stepOrder >= len(flatSteps) {
		return nil
	}
	return flatSteps[stepOrder].Config
}
