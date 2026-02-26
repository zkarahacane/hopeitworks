package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// RunService provides business logic for run operations.
type RunService struct {
	runRepo            port.RunRepository
	projectRepo        port.ProjectRepository
	storyRepo          port.StoryRepository
	pipelineConfigRepo port.PipelineConfigRepository
	jobQueue           port.JobQueue
	eventPub           port.EventPublisher
	containerMgr       port.ContainerManager
	agentRepo          port.AgentRepository
}

// NewRunService creates a new RunService.
func NewRunService(
	runRepo port.RunRepository,
	projectRepo port.ProjectRepository,
	storyRepo port.StoryRepository,
	pipelineConfigRepo port.PipelineConfigRepository,
	jobQueue port.JobQueue,
	eventPub ...port.EventPublisher,
) *RunService {
	svc := &RunService{
		runRepo:            runRepo,
		projectRepo:        projectRepo,
		storyRepo:          storyRepo,
		pipelineConfigRepo: pipelineConfigRepo,
		jobQueue:           jobQueue,
	}
	if len(eventPub) > 0 {
		svc.eventPub = eventPub[0]
	}
	return svc
}

// SetContainerManager configures the container manager for cancellation support.
func (s *RunService) SetContainerManager(cm port.ContainerManager) {
	s.containerMgr = cm
}

// SetAgentRepo configures the agent repository for agent resolution at run launch.
func (s *RunService) SetAgentRepo(repo port.AgentRepository) {
	s.agentRepo = repo
}

// PipelineStepConfig represents a step in a pipeline configuration.
type PipelineStepConfig struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

// PipelineConfig represents a pipeline configuration with steps.
type PipelineConfig struct {
	Steps []PipelineStepConfig `json:"steps"`
}

// CreateRunParams holds parameters for creating a run.
type CreateRunParams struct {
	ProjectID      uuid.UUID
	StoryID        uuid.UUID
	PipelineConfig json.RawMessage
}

// CreateRun creates a new run with steps from the provided pipeline config.
func (s *RunService) CreateRun(ctx context.Context, params CreateRunParams) (*model.Run, error) {
	if params.ProjectID == uuid.Nil {
		return nil, errors.NewValidation("project_id", "is required")
	}
	if params.StoryID == uuid.Nil {
		return nil, errors.NewValidation("story_id", "is required")
	}

	// Verify project exists
	_, err := s.projectRepo.GetByID(ctx, params.ProjectID)
	if err != nil {
		return nil, err
	}

	// Parse pipeline config
	if len(params.PipelineConfig) == 0 {
		return nil, errors.NewValidation("pipeline_config", "is required")
	}

	var config PipelineConfig
	if err := json.Unmarshal(params.PipelineConfig, &config); err != nil {
		return nil, errors.NewValidation("pipeline_config", fmt.Sprintf("invalid JSON: %v", err))
	}

	if len(config.Steps) == 0 {
		return nil, errors.NewValidation("pipeline_config", "must contain at least one step")
	}

	// Create run
	run := &model.Run{
		ProjectID:              params.ProjectID,
		StoryID:                params.StoryID,
		Status:                 model.RunStatusPending,
		PipelineConfigSnapshot: params.PipelineConfig,
	}

	createdRun, err := s.runRepo.CreateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	// Create steps
	steps := make([]model.RunStep, 0, len(config.Steps))
	for i, stepConfig := range config.Steps {
		step := &model.RunStep{
			RunID:     createdRun.ID,
			StepName:  stepConfig.Name,
			StepOrder: i,
			Action:    stepConfig.Action,
			Status:    model.StepStatusPending,
		}
		createdStep, err := s.runRepo.CreateRunStep(ctx, step)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *createdStep)
	}

	createdRun.Steps = steps
	return createdRun, nil
}

// GetRun retrieves a run by ID with its steps.
func (s *RunService) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
	run, err := s.runRepo.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}

	steps, err := s.runRepo.ListRunStepsByRun(ctx, id)
	if err != nil {
		return nil, err
	}

	run.Steps = make([]model.RunStep, len(steps))
	for i, step := range steps {
		run.Steps[i] = *step
	}
	run.Progress = run.ComputeProgress(run.Steps)

	return run, nil
}

// RunListResult holds the result of a paginated run list operation.
type RunListResult struct {
	Runs  []*model.Run
	Total int64
}

// enrichRunsWithSteps fetches and attaches steps (with progress) to each run.
// TODO(perf): batch fetch steps in single query for large lists (S-4-2)
func (s *RunService) enrichRunsWithSteps(ctx context.Context, runs []*model.Run) error {
	for _, r := range runs {
		steps, err := s.runRepo.ListRunStepsByRun(ctx, r.ID)
		if err != nil {
			return err
		}
		r.Steps = make([]model.RunStep, len(steps))
		for i, step := range steps {
			r.Steps[i] = *step
		}
		r.Progress = r.ComputeProgress(r.Steps)
	}
	return nil
}

// ListRunsByProject retrieves a paginated list of runs for a project.
func (s *RunService) ListRunsByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*RunListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	runs, err := s.runRepo.ListRunsByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	if err := s.enrichRunsWithSteps(ctx, runs); err != nil {
		return nil, err
	}

	total, err := s.runRepo.CountRunsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &RunListResult{
		Runs:  runs,
		Total: total,
	}, nil
}

// ListRunsByStory retrieves a paginated list of runs for a story.
func (s *RunService) ListRunsByStory(ctx context.Context, storyID uuid.UUID, page, perPage int) (*RunListResult, error) {
	limit, offset := paginationToLimitOffset(page, perPage)

	runs, err := s.runRepo.ListRunsByStory(ctx, storyID, limit, offset)
	if err != nil {
		return nil, err
	}

	if err := s.enrichRunsWithSteps(ctx, runs); err != nil {
		return nil, err
	}

	total, err := s.runRepo.CountRunsByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}

	return &RunListResult{
		Runs:  runs,
		Total: total,
	}, nil
}

// TransitionRun validates and transitions a run to a new status.
func (s *RunService) TransitionRun(ctx context.Context, runID uuid.UUID, newStatus model.RunStatus) (*model.Run, error) {
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if err := model.ValidateRunTransition(run.Status, newStatus); err != nil {
		return nil, err
	}

	now := time.Now()
	var startedAt, completedAt *time.Time

	switch newStatus {
	case model.RunStatusRunning:
		startedAt = &now
	case model.RunStatusCompleted, model.RunStatusFailed, model.RunStatusCancelled:
		completedAt = &now
	}

	var pausedAt *time.Time
	if newStatus == model.RunStatusPaused {
		pausedAt = &now
	}

	return s.runRepo.UpdateRunStatus(ctx, runID, newStatus, startedAt, completedAt, pausedAt, nil)
}

// LaunchRun validates the story, creates a pending run with steps, and enqueues
// a River job for async execution. The userID identifies the launching user so
// that user-specific API keys can be resolved for agent containers.
func (s *RunService) LaunchRun(ctx context.Context, projectID, storyID, userID uuid.UUID) (*model.Run, error) {
	// 1. Verify story exists and belongs to project
	story, err := s.storyRepo.GetByID(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if story.ProjectID != projectID {
		return nil, errors.NewNotFound("story", storyID)
	}

	// 2. Guard: story must not be 'done'
	if story.Status == model.StoryStatusDone {
		return nil, &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "STORY_ALREADY_COMPLETED",
			Message:  fmt.Sprintf("story %s is already completed", story.Key),
		}
	}

	// 3. Guard: no active run (pending or running) for this story
	activeRun, err := s.runRepo.GetActiveRunByStory(ctx, storyID)
	if err != nil {
		return nil, err
	}
	if activeRun != nil {
		return nil, &errors.DomainError{
			Category: errors.CategoryConflict,
			Code:     "STORY_ALREADY_RUNNING",
			Message:  fmt.Sprintf("story %s already has an active run (%s)", story.Key, activeRun.ID),
		}
	}

	// 4. Fetch pipeline config for the project
	pipelineCfg, err := s.pipelineConfigRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		if isNotFound(err) {
			return nil, &errors.DomainError{
				Category: errors.CategoryNotFound,
				Code:     "PIPELINE_CONFIG_NOT_FOUND",
				Message:  fmt.Sprintf("no pipeline config found for project %s", projectID),
			}
		}
		return nil, err
	}

	// 5. Parse YAML steps (backward-compatible: legacy flat steps auto-wrapped in group)
	parsed, err := model.ParsePipelineConfigYAML([]byte(pipelineCfg.ConfigYAML))
	if err != nil {
		return nil, errors.NewInternal("parse pipeline config", err)
	}
	flatSteps := parsed.FlatSteps()
	if len(flatSteps) == 0 {
		return nil, &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "PIPELINE_CONFIG_EMPTY",
			Message:  "pipeline config has no steps",
		}
	}

	// 6. Snapshot config as JSON for the run record
	snapshotJSON, err := json.Marshal(parsed)
	if err != nil {
		return nil, errors.NewInternal("marshal pipeline config snapshot", err)
	}

	// 7. Compute run metadata
	branchName := "feat/" + story.Key
	runMetadata := map[string]interface{}{
		"branch_name":          branchName,
		"launched_by_user_id":  userID.String(),
	}

	// Build per-step metadata from pipeline config (keyed by step order).
	// For agent_run steps, agent_id is required — resolve the agent and snapshot
	// model, image, and template_content.
	for i, stepCfg := range flatSteps {
		if stepCfg.ActionType == "agent_run" || stepCfg.ActionType == "implement" || stepCfg.ActionType == "review" || stepCfg.ActionType == "merge" {
			if stepCfg.AgentID == "" {
				return nil, errors.NewValidation(
					fmt.Sprintf("step[%d].agent_id", i),
					"agent_id is required for agent_run steps")
			}
		}

		if stepCfg.AgentID != "" {
			agentUUID, parseErr := uuid.Parse(stepCfg.AgentID)
			if parseErr != nil {
				return nil, errors.NewValidation(fmt.Sprintf("step[%d].agent_id", i), "invalid UUID format")
			}
			if s.agentRepo == nil {
				return nil, errors.NewInternal("resolve agent", fmt.Errorf("agent repository not configured"))
			}
			agent, fetchErr := s.agentRepo.GetAgent(ctx, agentUUID)
			if fetchErr != nil {
				return nil, errors.NewNotFound("agent", stepCfg.AgentID)
			}
			runMetadata[fmt.Sprintf("step_%d_agent_id", i)] = agent.ID.String()
			runMetadata[fmt.Sprintf("step_%d_model", i)] = agent.Model
			if agent.Provider != "" {
				runMetadata[fmt.Sprintf("step_%d_provider", i)] = agent.Provider
			}
			if agent.Image != "" {
				runMetadata[fmt.Sprintf("step_%d_agent_image", i)] = agent.Image
			}
			if agent.TemplateContent != "" {
				runMetadata[fmt.Sprintf("step_%d_template_content", i)] = agent.TemplateContent
			}
		} else if stepCfg.Model != "" {
			runMetadata[fmt.Sprintf("step_%d_model", i)] = stepCfg.Model
		}
	}

	// 8. Create Run
	run := &model.Run{
		ProjectID:              projectID,
		StoryID:                storyID,
		Status:                 model.RunStatusPending,
		PipelineConfigSnapshot: snapshotJSON,
		Metadata:               runMetadata,
	}
	createdRun, err := s.runRepo.CreateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	// 9. Create RunSteps
	steps := make([]model.RunStep, 0, len(flatSteps))
	for i, stepCfg := range flatSteps {
		step := &model.RunStep{
			RunID:     createdRun.ID,
			StepName:  stepCfg.Name,
			StepOrder: i,
			Action:    stepCfg.ActionType,
			Status:    model.StepStatusPending,
		}
		createdStep, err := s.runRepo.CreateRunStep(ctx, step)
		if err != nil {
			return nil, err
		}
		steps = append(steps, *createdStep)
	}
	createdRun.Steps = steps

	// 10. Enqueue River job (non-transactional for MVP)
	if s.jobQueue == nil {
		return nil, errors.NewInternal("enqueue execute_run job", fmt.Errorf("job queue unavailable"))
	}
	if err := s.jobQueue.EnqueueExecuteRun(ctx, createdRun.ID); err != nil {
		return nil, errors.NewInternal("enqueue execute_run job", err)
	}

	return createdRun, nil
}

// isNotFound returns true if the error is a not_found domain error.
func isNotFound(err error) bool {
	domErr, ok := err.(*errors.DomainError)
	return ok && domErr.Category == errors.CategoryNotFound
}

// PauseRun transitions a running run to paused status and publishes a run.paused event.
// The currently executing step continues to completion, but no new steps are launched.
func (s *RunService) PauseRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error) {
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	if err := model.ValidateRunTransition(run.Status, model.RunStatusPaused); err != nil {
		return nil, err
	}

	now := time.Now()
	updated, err := s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusPaused, nil, nil, &now, nil)
	if err != nil {
		return nil, err
	}

	s.publishRunEvent(ctx, updated.ProjectID, updated.ID, "paused", map[string]any{
		"run_id":    runID.String(),
		"status":    string(model.RunStatusPaused),
		"paused_at": now.Format(time.RFC3339),
	})

	return updated, nil
}

// ResumeRun transitions a paused run back to running status, publishes a run.resumed event,
// and re-enqueues the run for continued execution from the last completed step.
func (s *RunService) ResumeRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error) {
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	if run.Status != model.RunStatusPaused {
		return nil, errors.NewInvalidState(errors.ErrCodeInvalidStateTransition,
			fmt.Sprintf("cannot resume run from status %s, must be paused", run.Status))
	}

	updated, err := s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusRunning, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	s.publishRunEvent(ctx, updated.ProjectID, updated.ID, "resumed", map[string]any{
		"run_id": runID.String(),
		"status": string(model.RunStatusRunning),
	})

	// Re-enqueue the run for continued execution
	if s.jobQueue != nil {
		if err := s.jobQueue.EnqueueExecuteRun(ctx, runID); err != nil {
			return nil, errors.NewInternal("enqueue execute_run job for resume", err)
		}
	}

	return updated, nil
}

// PauseEpicRun pauses an epic run. This is a placeholder that operates on the run
// identified by the epicId/runId combination. When story 7-2 implements the epic run
// model with parent/child relationships, this method will also pause pending child runs.
func (s *RunService) PauseEpicRun(ctx context.Context, projectID, _ uuid.UUID, runID uuid.UUID) (*model.Run, error) {
	return s.PauseRun(ctx, projectID, runID)
}

// ResumeEpicRun resumes a paused epic run. This is a placeholder that operates on the run
// identified by the epicId/runId combination. When story 7-2 implements the epic run
// model with parent/child relationships, this method will also resume paused child runs.
func (s *RunService) ResumeEpicRun(ctx context.Context, projectID, _ uuid.UUID, runID uuid.UUID) (*model.Run, error) {
	return s.ResumeRun(ctx, projectID, runID)
}

// CancelRun transitions a run to cancelled status, stops any running containers,
// and marks pending/running steps as cancelled.
func (s *RunService) CancelRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error) {
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	if err := model.ValidateRunTransition(run.Status, model.RunStatusCancelled); err != nil {
		return nil, err
	}

	// Fetch steps to cancel running/pending ones and stop containers
	steps, err := s.runRepo.ListRunStepsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cancelMsg := "cancelled by user"

	// Stop running containers and cancel active steps
	for _, step := range steps {
		switch step.Status {
		case model.StepStatusRunning, model.StepStatusWaitingApproval:
			// Stop the container if one is running
			if s.containerMgr != nil && step.ContainerID != nil && *step.ContainerID != "" {
				_ = s.containerMgr.Stop(ctx, *step.ContainerID)
			}
			if _, err := s.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCancelled, nil, &now, &cancelMsg); err != nil {
				return nil, err
			}
		case model.StepStatusPending:
			if _, err := s.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCancelled, nil, &now, &cancelMsg); err != nil {
				return nil, err
			}
		}
	}

	// Transition run to cancelled
	updated, err := s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusCancelled, nil, &now, nil, &cancelMsg)
	if err != nil {
		return nil, err
	}

	s.publishRunEvent(ctx, updated.ProjectID, updated.ID, "cancelled", map[string]any{
		"run_id":       runID.String(),
		"status":       string(model.RunStatusCancelled),
		"cancelled_at": now.Format(time.RFC3339),
	})

	return updated, nil
}

// CancelEpicRun cancels an epic run. This is a placeholder that operates on the run
// identified by the epicId/runId combination. When story 7-2 implements the epic run
// model with parent/child relationships, this method will also cancel active child runs.
func (s *RunService) CancelEpicRun(ctx context.Context, projectID, _ uuid.UUID, runID uuid.UUID) (*model.Run, error) {
	return s.CancelRun(ctx, projectID, runID)
}

// publishRunEvent publishes an event for a run status change.
func (s *RunService) publishRunEvent(ctx context.Context, projectID, runID uuid.UUID, action string, payload map[string]any) {
	if s.eventPub == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   runID,
		Action:     action,
		Payload:    payloadJSON,
	}

	_ = s.eventPub.Publish(ctx, event)
}

// RetryStep retries a failed run step by creating a new retry child step and
// re-enqueuing the run for execution. The run is transitioned back to running.
func (s *RunService) RetryStep(ctx context.Context, runID, stepID uuid.UUID) (*model.Run, error) {
	// 1. Fetch the run and validate ownership
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	// 2. Fetch the step and validate it belongs to this run
	step, err := s.runRepo.GetRunStep(ctx, stepID)
	if err != nil {
		return nil, err
	}
	if step.RunID != runID {
		return nil, errors.NewNotFound("step", stepID)
	}

	// 3. Validate step is in failed state
	if step.Status != model.StepStatusFailed {
		return nil, &errors.DomainError{
			Category: errors.CategoryConflict,
			Code:     "STEP_NOT_FAILED",
			Message:  fmt.Sprintf("step %s is in %s state, only failed steps can be retried", stepID, step.Status),
		}
	}

	// 4. Check retry limits — fetch pipeline config to get retry policy
	maxRetries := 3 // default
	if run.PipelineConfigSnapshot != nil {
		var parsed model.PipelineConfigYAML
		if err := json.Unmarshal(run.PipelineConfigSnapshot, &parsed); err == nil {
			for _, ps := range parsed.FlatSteps() {
				if ps.Name == step.StepName && ps.RetryPolicy.MaxRetries > 0 {
					maxRetries = ps.RetryPolicy.MaxRetries
					break
				}
			}
		}
	}

	// Check how many retries already exist for this step chain
	rootStepID := stepID
	if step.ParentStepID != nil {
		rootStepID = *step.ParentStepID
	}
	existingRetries, err := s.runRepo.ListRetryStepsByParent(ctx, rootStepID)
	if err != nil {
		return nil, err
	}
	if len(existingRetries) >= maxRetries {
		return nil, &errors.DomainError{
			Category: errors.CategoryConflict,
			Code:     "RETRY_MAX_EXCEEDED",
			Message:  fmt.Sprintf("max retries (%d) exceeded for step %s", maxRetries, step.StepName),
		}
	}

	// 5. Determine retry type: incremental for first 2, then full
	retryCount := step.RetryCount + 1
	retryType := "incremental"
	if retryCount > 2 {
		retryType = "full"
	}

	// 6. Create the retry step
	newStep := &model.RunStep{
		ID:           uuid.New(),
		RunID:        runID,
		StepName:     step.StepName,
		StepOrder:    step.StepOrder,
		Action:       step.Action,
		Status:       model.StepStatusPending,
		RetryCount:   retryCount,
		RetryType:    &retryType,
		ParentStepID: &rootStepID,
	}
	_, err = s.runRepo.CreateRetryRunStep(ctx, newStep)
	if err != nil {
		return nil, err
	}

	// 7. Transition run back to running if it was failed
	if run.Status == model.RunStatusFailed {
		_, err = s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusRunning, nil, nil, nil, nil)
		if err != nil {
			return nil, err
		}
	}

	// 8. Enqueue the run for continued execution
	if s.jobQueue != nil {
		if err := s.jobQueue.EnqueueExecuteRun(ctx, runID); err != nil {
			return nil, errors.NewInternal("enqueue execute_run job for retry", err)
		}
	}

	// 9. Publish event
	s.publishRunEvent(ctx, run.ProjectID, runID, "step.retry_initiated", map[string]any{
		"run_id":        runID.String(),
		"step_id":       stepID.String(),
		"retry_step_id": newStep.ID.String(),
		"retry_count":   retryCount,
		"retry_type":    retryType,
	})

	// 10. Return the updated run with all steps
	return s.GetRun(ctx, runID)
}

// TransitionRunStep validates and transitions a run step to a new status.
func (s *RunService) TransitionRunStep(ctx context.Context, stepID uuid.UUID, newStatus model.StepStatus) (*model.RunStep, error) {
	step, err := s.runRepo.GetRunStep(ctx, stepID)
	if err != nil {
		return nil, err
	}

	if err := model.ValidateStepTransition(step.Status, newStatus); err != nil {
		return nil, err
	}

	now := time.Now()
	var startedAt, completedAt *time.Time

	switch newStatus {
	case model.StepStatusRunning:
		startedAt = &now
	case model.StepStatusCompleted, model.StepStatusFailed, model.StepStatusCancelled:
		completedAt = &now
	}

	return s.runRepo.UpdateRunStepStatus(ctx, stepID, newStatus, startedAt, completedAt, nil)
}
