package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

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
}

// NewRunService creates a new RunService.
func NewRunService(
	runRepo port.RunRepository,
	projectRepo port.ProjectRepository,
	storyRepo port.StoryRepository,
	pipelineConfigRepo port.PipelineConfigRepository,
	jobQueue port.JobQueue,
) *RunService {
	return &RunService{
		runRepo:            runRepo,
		projectRepo:        projectRepo,
		storyRepo:          storyRepo,
		pipelineConfigRepo: pipelineConfigRepo,
		jobQueue:           jobQueue,
	}
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

	return run, nil
}

// RunListResult holds the result of a paginated run list operation.
type RunListResult struct {
	Runs  []*model.Run
	Total int64
}

// ListRunsByProject retrieves a paginated list of runs for a project.
func (s *RunService) ListRunsByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*RunListResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	runs, err := s.runRepo.ListRunsByProject(ctx, projectID, limit, offset)
	if err != nil {
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
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	offset := int32((page - 1) * perPage)
	limit := int32(perPage)

	runs, err := s.runRepo.ListRunsByStory(ctx, storyID, limit, offset)
	if err != nil {
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

	return s.runRepo.UpdateRunStatus(ctx, runID, newStatus, startedAt, completedAt, nil)
}

// LaunchRun validates the story, creates a pending run with steps, and enqueues
// a River job for async execution.
func (s *RunService) LaunchRun(ctx context.Context, projectID, storyID uuid.UUID) (*model.Run, error) {
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

	// 5. Parse YAML steps
	var parsed model.PipelineConfigYAML
	if err := yaml.Unmarshal([]byte(pipelineCfg.ConfigYAML), &parsed); err != nil {
		return nil, errors.NewInternal("parse pipeline config", err)
	}
	if len(parsed.Steps) == 0 {
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

	// 7. Create Run
	run := &model.Run{
		ProjectID:              projectID,
		StoryID:                storyID,
		Status:                 model.RunStatusPending,
		PipelineConfigSnapshot: snapshotJSON,
	}
	createdRun, err := s.runRepo.CreateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	// 8. Create RunSteps
	steps := make([]model.RunStep, 0, len(parsed.Steps))
	for i, stepCfg := range parsed.Steps {
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

	// 9. Enqueue River job (non-transactional for MVP)
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
