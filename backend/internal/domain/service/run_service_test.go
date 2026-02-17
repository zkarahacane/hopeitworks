package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockRunRepo implements port.RunRepository for testing.
type mockRunRepo struct {
	createRunFn           func(ctx context.Context, run *model.Run) (*model.Run, error)
	getRunFn              func(ctx context.Context, id uuid.UUID) (*model.Run, error)
	getActiveRunByStoryFn func(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
	listRunsByProjectFn   func(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	listRunsByStoryFn     func(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	updateRunStatusFn     func(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error)
	countRunsByProjectFn  func(ctx context.Context, projectID uuid.UUID) (int64, error)
	countRunsByStoryFn    func(ctx context.Context, storyID uuid.UUID) (int64, error)
	createRunStepFn       func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	getRunStepFn          func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	listRunStepsByRunFn   func(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error)
	updateRunStepStatusFn func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)
}

func (m *mockRunRepo) CreateRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	if m.createRunFn != nil {
		return m.createRunFn(ctx, run)
	}
	return run, nil
}
func (m *mockRunRepo) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *mockRunRepo) GetActiveRunByStory(_ context.Context, storyID uuid.UUID) (*model.Run, error) {
	if m.getActiveRunByStoryFn != nil {
		return m.getActiveRunByStoryFn(context.Background(), storyID)
	}
	return nil, nil
}
func (m *mockRunRepo) ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	if m.listRunsByProjectFn != nil {
		return m.listRunsByProjectFn(ctx, projectID, limit, offset)
	}
	return nil, nil
}
func (m *mockRunRepo) ListRunsByStory(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error) {
	if m.listRunsByStoryFn != nil {
		return m.listRunsByStoryFn(ctx, storyID, limit, offset)
	}
	return nil, nil
}
func (m *mockRunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error) {
	if m.updateRunStatusFn != nil {
		return m.updateRunStatusFn(ctx, id, status, startedAt, completedAt, pausedAt, errorMsg)
	}
	return nil, nil
}
func (m *mockRunRepo) CountRunsByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	if m.countRunsByProjectFn != nil {
		return m.countRunsByProjectFn(ctx, projectID)
	}
	return 0, nil
}
func (m *mockRunRepo) CountRunsByStory(ctx context.Context, storyID uuid.UUID) (int64, error) {
	if m.countRunsByStoryFn != nil {
		return m.countRunsByStoryFn(ctx, storyID)
	}
	return 0, nil
}
func (m *mockRunRepo) CreateRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	if m.createRunStepFn != nil {
		return m.createRunStepFn(ctx, step)
	}
	step.ID = uuid.New()
	return step, nil
}
func (m *mockRunRepo) GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error) {
	if m.getRunStepFn != nil {
		return m.getRunStepFn(ctx, id)
	}
	return nil, errors.NewNotFound("run step", id)
}
func (m *mockRunRepo) ListRunStepsByRun(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error) {
	if m.listRunStepsByRunFn != nil {
		return m.listRunStepsByRunFn(ctx, runID)
	}
	return nil, nil
}
func (m *mockRunRepo) UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error) {
	if m.updateRunStepStatusFn != nil {
		return m.updateRunStepStatusFn(ctx, id, status, startedAt, completedAt, errorMsg)
	}
	return nil, nil
}
func (m *mockRunRepo) UpdateRunStepContainerInfo(_ context.Context, id uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}

// mockStoryRepoForRun implements port.StoryRepository for testing.
type mockStoryRepoForRun struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

func (m *mockStoryRepoForRun) Create(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.NewNotFound("story", id)
}
func (m *mockStoryRepoForRun) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForRun) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForRun) Update(_ context.Context, _ *model.Story) (*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForRun) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockPipelineConfigRepoForRun implements port.PipelineConfigRepository for testing.
type mockPipelineConfigRepoForRun struct {
	getByProjectIDFn func(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
}

func (m *mockPipelineConfigRepoForRun) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	if m.getByProjectIDFn != nil {
		return m.getByProjectIDFn(ctx, projectID)
	}
	return nil, errors.NewNotFound("pipeline_config", projectID)
}
func (m *mockPipelineConfigRepoForRun) Upsert(_ context.Context, _ *model.PipelineConfig) (*model.PipelineConfig, error) {
	return nil, nil
}

// mockJobQueue implements port.JobQueue for testing.
type mockJobQueue struct {
	enqueueExecuteRunFn func(ctx context.Context, runID uuid.UUID) error
}

func (m *mockJobQueue) EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error {
	if m.enqueueExecuteRunFn != nil {
		return m.enqueueExecuteRunFn(ctx, runID)
	}
	return nil
}

// newMockProjectRepoWithProject creates a mockProjectRepo preloaded with a project.
func newMockProjectRepoWithProject(project *model.Project) *mockProjectRepo {
	repo := newMockProjectRepoForService()
	repo.projects[project.ID] = project
	return repo
}

// newRunServiceForTest creates a RunService with all dependencies for existing tests that don't need LaunchRun.
func newRunServiceForTest(runRepo *mockRunRepo, projectRepo *mockProjectRepo) *RunService {
	return NewRunService(runRepo, projectRepo, nil, nil, nil)
}

func TestCreateRun_Success(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	config := json.RawMessage(`{"steps":[{"name":"dev","action":"code"},{"name":"review","action":"review"},{"name":"merge","action":"merge"}]}`)

	var createdStepCount int
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			run.CreatedAt = time.Now()
			run.UpdatedAt = time.Now()
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			step.CreatedAt = time.Now()
			createdStepCount++
			return step, nil
		},
	}
	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test-project",
	})

	svc := newRunServiceForTest(runRepo, projectRepo)
	run, err := svc.CreateRun(context.Background(), CreateRunParams{
		ProjectID:      projectID,
		StoryID:        storyID,
		PipelineConfig: config,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if run.Status != model.RunStatusPending {
		t.Errorf("expected status pending, got %s", run.Status)
	}
	if run.ProjectID != projectID {
		t.Errorf("expected project_id %s, got %s", projectID, run.ProjectID)
	}
	if run.StoryID != storyID {
		t.Errorf("expected story_id %s, got %s", storyID, run.StoryID)
	}
	if createdStepCount != 3 {
		t.Errorf("expected 3 steps created, got %d", createdStepCount)
	}
	if len(run.Steps) != 3 {
		t.Errorf("expected 3 steps in run, got %d", len(run.Steps))
	}
	for i, step := range run.Steps {
		if step.StepOrder != i {
			t.Errorf("step %d: expected order %d, got %d", i, i, step.StepOrder)
		}
		if step.Status != model.StepStatusPending {
			t.Errorf("step %d: expected status pending, got %s", i, step.Status)
		}
	}
}

func TestCreateRun_MissingProject(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	config := json.RawMessage(`{"steps":[{"name":"dev","action":"code"}]}`)

	runRepo := &mockRunRepo{}
	projectRepo := newMockProjectRepoForService()

	svc := newRunServiceForTest(runRepo, projectRepo)
	_, err := svc.CreateRun(context.Background(), CreateRunParams{
		ProjectID:      projectID,
		StoryID:        storyID,
		PipelineConfig: config,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestCreateRun_InvalidConfig(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	projectRepo := newMockProjectRepoWithProject(&model.Project{
		ID:   projectID,
		Name: "test",
	})

	tests := []struct {
		name   string
		config json.RawMessage
	}{
		{"nil config", nil},
		{"empty config", json.RawMessage(``)},
		{"invalid JSON", json.RawMessage(`{invalid}`)},
		{"empty steps", json.RawMessage(`{"steps":[]}`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newRunServiceForTest(&mockRunRepo{}, projectRepo)
			_, err := svc.CreateRun(context.Background(), CreateRunParams{
				ProjectID:      projectID,
				StoryID:        storyID,
				PipelineConfig: tt.config,
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			domainErr, ok := err.(*errors.DomainError)
			if !ok {
				t.Fatalf("expected *errors.DomainError, got %T", err)
			}
			if domainErr.Category != errors.CategoryValidation {
				t.Errorf("expected validation category, got %s", domainErr.Category)
			}
		})
	}
}

func TestCreateRun_MissingRequiredFields(t *testing.T) {
	svc := newRunServiceForTest(&mockRunRepo{}, newMockProjectRepoForService())

	_, err := svc.CreateRun(context.Background(), CreateRunParams{
		StoryID:        uuid.New(),
		PipelineConfig: json.RawMessage(`{"steps":[{"name":"dev","action":"code"}]}`),
	})
	if err == nil {
		t.Fatal("expected error for missing project_id")
	}

	_, err = svc.CreateRun(context.Background(), CreateRunParams{
		ProjectID:      uuid.New(),
		PipelineConfig: json.RawMessage(`{"steps":[{"name":"dev","action":"code"}]}`),
	})
	if err == nil {
		t.Fatal("expected error for missing story_id")
	}
}

func TestTransitionRun_ValidTransitions(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name       string
		fromStatus model.RunStatus
		toStatus   model.RunStatus
	}{
		{"pending to running", model.RunStatusPending, model.RunStatusRunning},
		{"running to completed", model.RunStatusRunning, model.RunStatusCompleted},
		{"running to failed", model.RunStatusRunning, model.RunStatusFailed},
		{"running to cancelled", model.RunStatusRunning, model.RunStatusCancelled},
		{"pending to cancelled", model.RunStatusPending, model.RunStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:     runID,
						Status: tt.fromStatus,
					}, nil
				},
				updateRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.RunStatus, startedAt, completedAt, _ *time.Time, _ *string) (*model.Run, error) {
					run := &model.Run{
						ID:     runID,
						Status: status,
					}
					if startedAt != nil {
						run.StartedAt = startedAt
					}
					if completedAt != nil {
						run.CompletedAt = completedAt
					}
					return run, nil
				},
			}
			svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

			result, err := svc.TransitionRun(context.Background(), runID, tt.toStatus)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Status != tt.toStatus {
				t.Errorf("expected status %s, got %s", tt.toStatus, result.Status)
			}
			if tt.toStatus == model.RunStatusRunning && result.StartedAt == nil {
				t.Error("expected started_at to be set when transitioning to running")
			}
			if (tt.toStatus == model.RunStatusCompleted || tt.toStatus == model.RunStatusFailed || tt.toStatus == model.RunStatusCancelled) && result.CompletedAt == nil {
				t.Errorf("expected completed_at to be set when transitioning to %s", tt.toStatus)
			}
		})
	}
}

func TestTransitionRun_InvalidTransition(t *testing.T) {
	runID := uuid.New()
	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:     runID,
				Status: model.RunStatusCompleted,
			}, nil
		},
	}
	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	_, err := svc.TransitionRun(context.Background(), runID, model.RunStatusRunning)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "INVALID_STATE_TRANSITION" {
		t.Errorf("expected INVALID_STATE_TRANSITION code, got %s", domainErr.Code)
	}
}

func TestTransitionRunStep_ValidTransitions(t *testing.T) {
	stepID := uuid.New()

	tests := []struct {
		name       string
		fromStatus model.StepStatus
		toStatus   model.StepStatus
	}{
		{"pending to running", model.StepStatusPending, model.StepStatusRunning},
		{"running to completed", model.StepStatusRunning, model.StepStatusCompleted},
		{"running to failed", model.StepStatusRunning, model.StepStatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
					return &model.RunStep{
						ID:     stepID,
						Status: tt.fromStatus,
					}, nil
				},
				updateRunStepStatusFn: func(_ context.Context, _ uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, _ *string) (*model.RunStep, error) {
					step := &model.RunStep{
						ID:     stepID,
						Status: status,
					}
					if startedAt != nil {
						step.StartedAt = startedAt
					}
					if completedAt != nil {
						step.CompletedAt = completedAt
					}
					return step, nil
				},
			}
			svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

			result, err := svc.TransitionRunStep(context.Background(), stepID, tt.toStatus)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Status != tt.toStatus {
				t.Errorf("expected status %s, got %s", tt.toStatus, result.Status)
			}
		})
	}
}

func TestTransitionRunStep_InvalidTransition(t *testing.T) {
	stepID := uuid.New()
	runRepo := &mockRunRepo{
		getRunStepFn: func(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:     stepID,
				Status: model.StepStatusFailed,
			}, nil
		},
	}
	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	_, err := svc.TransitionRunStep(context.Background(), stepID, model.StepStatusPending)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "INVALID_STATE_TRANSITION" {
		t.Errorf("expected INVALID_STATE_TRANSITION code, got %s", domainErr.Code)
	}
}

// ── LaunchRun Tests ──────────────────────────────────────────────────────────

const testPipelineYAML = `steps:
  - id: "step-1"
    name: "implement"
    action_type: "implement"
    model: "claude-opus-4-6"
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: "none"
  - id: "step-2"
    name: "review"
    action_type: "review"
    model: "claude-sonnet-4-5"
    auto_approve: true
    retry_policy:
      max_retries: 1
      retry_type: "on-failure"
  - id: "step-3"
    name: "merge"
    action_type: "merge"
    model: "claude-sonnet-4-5"
    auto_approve: true
    retry_policy:
      max_retries: 0
      retry_type: "none"
`

func TestLaunchRun_Success(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}

	pipelineConfigRepo := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ID:         uuid.New(),
				ProjectID:  projectID,
				ConfigYAML: testPipelineYAML,
				Version:    1,
			}, nil
		},
	}

	var enqueuedRunID uuid.UUID
	jobQueue := &mockJobQueue{
		enqueueExecuteRunFn: func(_ context.Context, runID uuid.UUID) error {
			enqueuedRunID = runID
			return nil
		},
	}

	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			run.CreatedAt = time.Now()
			run.UpdatedAt = time.Now()
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			step.CreatedAt = time.Now()
			return step, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, jobQueue)
	run, err := svc.LaunchRun(context.Background(), projectID, storyID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if run.Status != model.RunStatusPending {
		t.Errorf("expected status pending, got %s", run.Status)
	}
	if run.ProjectID != projectID {
		t.Errorf("expected project_id %s, got %s", projectID, run.ProjectID)
	}
	if run.StoryID != storyID {
		t.Errorf("expected story_id %s, got %s", storyID, run.StoryID)
	}
	if len(run.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(run.Steps))
	}
	expectedNames := []string{"implement", "review", "merge"}
	expectedActions := []string{"implement", "review", "merge"}
	for i, step := range run.Steps {
		if step.StepName != expectedNames[i] {
			t.Errorf("step %d: expected name %q, got %q", i, expectedNames[i], step.StepName)
		}
		if step.Action != expectedActions[i] {
			t.Errorf("step %d: expected action %q, got %q", i, expectedActions[i], step.Action)
		}
		if step.StepOrder != i {
			t.Errorf("step %d: expected order %d, got %d", i, i, step.StepOrder)
		}
		if step.Status != model.StepStatusPending {
			t.Errorf("step %d: expected status pending, got %s", i, step.Status)
		}
	}
	if enqueuedRunID != run.ID {
		t.Errorf("expected enqueued run ID %s, got %s", run.ID, enqueuedRunID)
	}
}

func TestLaunchRun_StoryNotFound(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	svc := NewRunService(
		&mockRunRepo{},
		newMockProjectRepoForService(),
		&mockStoryRepoForRun{},
		&mockPipelineConfigRepoForRun{},
		&mockJobQueue{},
	)

	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestLaunchRun_StoryWrongProject(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	otherProjectID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: otherProjectID,
				Key:       "S-01",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}

	svc := NewRunService(
		&mockRunRepo{},
		newMockProjectRepoForService(),
		storyRepo,
		&mockPipelineConfigRepoForRun{},
		&mockJobQueue{},
	)

	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestLaunchRun_StoryAlreadyCompleted(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusDone,
			}, nil
		},
	}

	svc := NewRunService(
		&mockRunRepo{},
		newMockProjectRepoForService(),
		storyRepo,
		&mockPipelineConfigRepoForRun{},
		&mockJobQueue{},
	)

	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "STORY_ALREADY_COMPLETED" {
		t.Errorf("expected STORY_ALREADY_COMPLETED code, got %s", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryValidation {
		t.Errorf("expected validation category, got %s", domainErr.Category)
	}
}

func TestLaunchRun_StoryAlreadyRunning(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	activeRunID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusRunning,
			}, nil
		},
	}

	runRepo := &mockRunRepo{
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:     activeRunID,
				Status: model.RunStatusRunning,
			}, nil
		},
	}

	svc := NewRunService(
		runRepo,
		newMockProjectRepoForService(),
		storyRepo,
		&mockPipelineConfigRepoForRun{},
		&mockJobQueue{},
	)

	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "STORY_ALREADY_RUNNING" {
		t.Errorf("expected STORY_ALREADY_RUNNING code, got %s", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryConflict {
		t.Errorf("expected conflict category, got %s", domainErr.Category)
	}
}

func TestLaunchRun_NoPipelineConfig(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}

	svc := NewRunService(
		&mockRunRepo{},
		newMockProjectRepoForService(),
		storyRepo,
		&mockPipelineConfigRepoForRun{},
		&mockJobQueue{},
	)

	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "PIPELINE_CONFIG_NOT_FOUND" {
		t.Errorf("expected PIPELINE_CONFIG_NOT_FOUND code, got %s", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestLaunchRun_JobEnqueueFails(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}

	pipelineConfigRepo := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ID:         uuid.New(),
				ProjectID:  projectID,
				ConfigYAML: testPipelineYAML,
				Version:    1,
			}, nil
		},
	}

	jobQueue := &mockJobQueue{
		enqueueExecuteRunFn: func(_ context.Context, _ uuid.UUID) error {
			return fmt.Errorf("connection refused")
		},
	}

	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			run.CreatedAt = time.Now()
			run.UpdatedAt = time.Now()
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			step.CreatedAt = time.Now()
			return step, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, jobQueue)
	_, err := svc.LaunchRun(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryInternal {
		t.Errorf("expected internal category, got %s", domainErr.Category)
	}
}

// ── PauseRun Tests ───────────────────────────────────────────────────────────

func TestPauseRun_Success(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusRunning,
			}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, pausedAt *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    status,
				PausedAt:  pausedAt,
			}, nil
		},
	}

	eventPub := newMockEventPublisher()
	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil, eventPub)

	run, err := svc.PauseRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if run.Status != model.RunStatusPaused {
		t.Errorf("expected status paused, got %s", run.Status)
	}
	if run.PausedAt == nil {
		t.Error("expected paused_at to be set")
	}

	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event.EventName() != "run.paused" {
		t.Errorf("expected run.paused event, got %s", events[0].Event.EventName())
	}
}

func TestPauseRun_InvalidState(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	tests := []struct {
		name   string
		status model.RunStatus
	}{
		{"pending run", model.RunStatusPending},
		{"completed run", model.RunStatusCompleted},
		{"failed run", model.RunStatusFailed},
		{"cancelled run", model.RunStatusCancelled},
		{"already paused run", model.RunStatusPaused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:        id,
						ProjectID: projectID,
						Status:    tt.status,
					}, nil
				},
			}

			svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
			_, err := svc.PauseRun(context.Background(), projectID, runID)
			if err == nil {
				t.Fatal("expected error for invalid state transition")
			}
			domainErr, ok := err.(*errors.DomainError)
			if !ok {
				t.Fatalf("expected *errors.DomainError, got %T", err)
			}
			if domainErr.Code != "INVALID_STATE_TRANSITION" {
				t.Errorf("expected INVALID_STATE_TRANSITION code, got %s", domainErr.Code)
			}
		})
	}
}

func TestPauseRun_WrongProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	runID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: otherProjectID,
				Status:    model.RunStatusRunning,
			}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
	_, err := svc.PauseRun(context.Background(), projectID, runID)
	if err == nil {
		t.Fatal("expected error for wrong project")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

// ── ResumeRun Tests ──────────────────────────────────────────────────────────

func TestResumeRun_Success(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusPaused,
			}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    status,
			}, nil
		},
	}

	var enqueuedRunID uuid.UUID
	jobQueue := &mockJobQueue{
		enqueueExecuteRunFn: func(_ context.Context, id uuid.UUID) error {
			enqueuedRunID = id
			return nil
		},
	}

	eventPub := newMockEventPublisher()
	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, jobQueue, eventPub)

	run, err := svc.ResumeRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if run.Status != model.RunStatusRunning {
		t.Errorf("expected status running, got %s", run.Status)
	}
	if enqueuedRunID != runID {
		t.Errorf("expected enqueued run ID %s, got %s", runID, enqueuedRunID)
	}

	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event.EventName() != "run.resumed" {
		t.Errorf("expected run.resumed event, got %s", events[0].Event.EventName())
	}
}

func TestResumeRun_InvalidState(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	tests := []struct {
		name   string
		status model.RunStatus
	}{
		{"pending run", model.RunStatusPending},
		{"running run", model.RunStatusRunning},
		{"completed run", model.RunStatusCompleted},
		{"failed run", model.RunStatusFailed},
		{"cancelled run", model.RunStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:        id,
						ProjectID: projectID,
						Status:    tt.status,
					}, nil
				},
			}

			svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
			_, err := svc.ResumeRun(context.Background(), projectID, runID)
			if err == nil {
				t.Fatal("expected error for invalid state transition")
			}
		})
	}
}

func TestResumeRun_WrongProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	runID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: otherProjectID,
				Status:    model.RunStatusPaused,
			}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
	_, err := svc.ResumeRun(context.Background(), projectID, runID)
	if err == nil {
		t.Fatal("expected error for wrong project")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Category != errors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

// ── Transition Tests with Paused ─────────────────────────────────────────────

func TestTransitionRun_PauseAndResume(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name       string
		fromStatus model.RunStatus
		toStatus   model.RunStatus
	}{
		{"running to paused", model.RunStatusRunning, model.RunStatusPaused},
		{"paused to running", model.RunStatusPaused, model.RunStatusRunning},
		{"paused to cancelled", model.RunStatusPaused, model.RunStatusCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:     runID,
						Status: tt.fromStatus,
					}, nil
				},
				updateRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
					return &model.Run{
						ID:     runID,
						Status: status,
					}, nil
				},
			}
			svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

			result, err := svc.TransitionRun(context.Background(), runID, tt.toStatus)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if result.Status != tt.toStatus {
				t.Errorf("expected status %s, got %s", tt.toStatus, result.Status)
			}
		})
	}
}

func TestTransitionRun_PausedInvalidTransitions(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name       string
		fromStatus model.RunStatus
		toStatus   model.RunStatus
	}{
		{"paused to completed", model.RunStatusPaused, model.RunStatusCompleted},
		{"paused to failed", model.RunStatusPaused, model.RunStatusFailed},
		{"pending to paused", model.RunStatusPending, model.RunStatusPaused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:     runID,
						Status: tt.fromStatus,
					}, nil
				},
			}
			svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

			_, err := svc.TransitionRun(context.Background(), runID, tt.toStatus)
			if err == nil {
				t.Fatalf("expected error for invalid transition from %s to %s", tt.fromStatus, tt.toStatus)
			}
			domainErr, ok := err.(*errors.DomainError)
			if !ok {
				t.Fatalf("expected *errors.DomainError, got %T", err)
			}
			if domainErr.Code != "INVALID_STATE_TRANSITION" {
				t.Errorf("expected INVALID_STATE_TRANSITION code, got %s", domainErr.Code)
			}
		})
	}
}
