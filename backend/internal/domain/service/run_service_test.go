package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockRunRepo implements port.RunRepository for testing.
type mockRunRepo struct {
	createRunFn              func(ctx context.Context, run *model.Run) (*model.Run, error)
	getRunFn                 func(ctx context.Context, id uuid.UUID) (*model.Run, error)
	getActiveRunByStoryFn    func(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
	listRunsByProjectFn      func(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	listRunsByStoryFn        func(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	updateRunStatusFn        func(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errorMsg *string) (*model.Run, error)
	countRunsByProjectFn     func(ctx context.Context, projectID uuid.UUID) (int64, error)
	countRunsByStoryFn       func(ctx context.Context, storyID uuid.UUID) (int64, error)
	createRunStepFn          func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	getRunStepFn             func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
	listRunStepsByRunFn      func(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error)
	updateRunStepStatusFn    func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.RunStep, error)
	createRetryRunStepFn     func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	listRetryStepsByParentFn func(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error)
	updateRunMetadataFn      func(ctx context.Context, runID uuid.UUID, metadata map[string]interface{}) error
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
func (m *mockRunRepo) CreateRetryRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	if m.createRetryRunStepFn != nil {
		return m.createRetryRunStepFn(ctx, step)
	}
	return step, nil
}
func (m *mockRunRepo) ListRetryStepsByParent(ctx context.Context, parentStepID uuid.UUID) ([]*model.RunStep, error) {
	if m.listRetryStepsByParentFn != nil {
		return m.listRetryStepsByParentFn(ctx, parentStepID)
	}
	return nil, nil
}

// mockStoryRepoForRun implements port.StoryRepository for testing.
type mockStoryRepoForRun struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
	updateFn  func(ctx context.Context, story *model.Story) (*model.Story, error)
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
func (m *mockStoryRepoForRun) Update(ctx context.Context, story *model.Story) (*model.Story, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, story)
	}
	return story, nil
}
func (m *mockStoryRepoForRun) UpdateStoryCurrentStage(_ context.Context, id uuid.UUID, currentStage *string) (*model.Story, error) {
	return &model.Story{ID: id, CurrentStage: currentStage}, nil
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
	if domainErr.Code != errors.ErrCodeInvalidStateTransition {
		t.Errorf("expected %s code, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
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
	if domainErr.Code != errors.ErrCodeInvalidStateTransition {
		t.Errorf("expected %s code, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
	}
}

// ── LaunchRun Tests ──────────────────────────────────────────────────────────

// testAgentIDs are fixed UUIDs used in testPipelineYAML for deterministic tests.
var (
	testAgentIDImplement = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testAgentIDReview    = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testAgentIDMerge     = uuid.MustParse("00000000-0000-0000-0000-000000000003")
)

const testPipelineYAML = `steps:
  - id: "step-1"
    name: "implement"
    action_type: "implement"
    agent_id: "00000000-0000-0000-0000-000000000001"
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: "none"
  - id: "step-2"
    name: "review"
    action_type: "review"
    agent_id: "00000000-0000-0000-0000-000000000002"
    auto_approve: true
    retry_policy:
      max_retries: 1
      retry_type: "on-failure"
  - id: "step-3"
    name: "merge"
    action_type: "merge"
    agent_id: "00000000-0000-0000-0000-000000000003"
    auto_approve: true
    retry_policy:
      max_retries: 0
      retry_type: "none"
`

// newTestAgentRepo returns a mock agent repo pre-loaded with the agents used in testPipelineYAML.
func newTestAgentRepo() *mockAgentRepo {
	repo := newMockAgentRepo()
	repo.agents[testAgentIDImplement] = &model.Agent{
		ID:    testAgentIDImplement,
		Model: "claude-opus-4-6",
		Image: "hopeitworks/agent:latest",
	}
	repo.agents[testAgentIDReview] = &model.Agent{
		ID:    testAgentIDReview,
		Model: "claude-sonnet-4-6",
		Image: "hopeitworks/agent:latest",
	}
	repo.agents[testAgentIDMerge] = &model.Agent{
		ID:    testAgentIDMerge,
		Model: "claude-sonnet-4-6",
		Image: "hopeitworks/agent:latest",
	}
	return repo
}

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
	svc.SetAgentRepo(newTestAgentRepo())
	run, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)

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

func TestLaunchRun_BranchNameInMetadata(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "runtime-4",
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

	var capturedRun *model.Run
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			capturedRun = run
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			return step, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, &mockJobQueue{})
	svc.SetAgentRepo(newTestAgentRepo())
	run, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// branch_name is no longer pre-seeded at launch — git_branch action computes and
	// persists it after the step executes. Verify it is absent from the initial metadata.
	if capturedRun.Metadata == nil {
		t.Fatal("expected run metadata to be set")
	}
	if _, hasBranch := capturedRun.Metadata["branch_name"]; hasBranch {
		t.Error("branch_name must NOT be pre-seeded in metadata at launch; git_branch action sets it")
	}

	// launched_by_user_id must still be present
	if _, ok := capturedRun.Metadata["launched_by_user_id"]; !ok {
		t.Error("expected launched_by_user_id in metadata")
	}

	// Verify returned run also has metadata
	if run.Metadata == nil {
		t.Fatal("expected returned run metadata to be set")
	}
}

func TestLaunchRun_ModelInMetadata(t *testing.T) {
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

	var capturedRun *model.Run
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			capturedRun = run
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			return step, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, &mockJobQueue{})
	svc.SetAgentRepo(newTestAgentRepo())
	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify per-step model keys in metadata (testPipelineYAML has agents on all 3 steps)
	if capturedRun.Metadata == nil {
		t.Fatal("expected run metadata to be set")
	}
	step0Model, ok := capturedRun.Metadata["step_0_model"].(string)
	if !ok {
		t.Fatal("expected step_0_model in metadata")
	}
	if step0Model != "claude-opus-4-6" {
		t.Errorf("expected step_0_model %q, got %q", "claude-opus-4-6", step0Model)
	}
	step1Model, ok := capturedRun.Metadata["step_1_model"].(string)
	if !ok {
		t.Fatal("expected step_1_model in metadata")
	}
	if step1Model != "claude-sonnet-4-6" {
		t.Errorf("expected step_1_model %q, got %q", "claude-sonnet-4-6", step1Model)
	}
}

func TestLaunchRun_NoModelWhenEmpty(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	pipelineYAMLNoModel := `steps:
  - id: "step-1"
    name: "git-branch"
    action_type: "git_branch"
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: "none"
`

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-02",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}

	pipelineConfigRepo := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ID:         uuid.New(),
				ProjectID:  projectID,
				ConfigYAML: pipelineYAMLNoModel,
				Version:    1,
			}, nil
		},
	}

	var capturedRun *model.Run
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			capturedRun = run
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			return step, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, &mockJobQueue{})
	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify no step_0_model key when model is empty
	if _, exists := capturedRun.Metadata["step_0_model"]; exists {
		t.Error("expected no step_0_model key when pipeline step has no model")
	}
}

// stackPipelineYAML is a single implement step bound to testAgentIDImplement.
const stackPipelineYAML = `steps:
  - id: "step-1"
    name: "implement"
    action_type: "implement"
    agent_id: "00000000-0000-0000-0000-000000000001"
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: "none"
`

// launchRunCapture runs LaunchRun with a single-step pipeline bound to the given
// agent, optionally wiring a stack repo, and returns the captured run metadata.
func launchRunCapture(t *testing.T, agent *model.Agent, stackRepo *mockStackRepo) map[string]interface{} {
	t.Helper()
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-01", Status: model.StoryStatusBacklog}, nil
		},
	}
	pipelineConfigRepo := &mockPipelineConfigRepoForRun{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{ID: uuid.New(), ProjectID: projectID, ConfigYAML: stackPipelineYAML, Version: 1}, nil
		},
	}
	var capturedRun *model.Run
	runRepo := &mockRunRepo{
		createRunFn: func(_ context.Context, run *model.Run) (*model.Run, error) {
			run.ID = uuid.New()
			capturedRun = run
			return run, nil
		},
		createRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			step.ID = uuid.New()
			return step, nil
		},
	}

	agentRepo := newMockAgentRepo()
	agentRepo.agents[testAgentIDImplement] = agent

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, pipelineConfigRepo, &mockJobQueue{})
	svc.SetAgentRepo(agentRepo)
	if stackRepo != nil {
		svc.SetStackRepo(stackRepo)
	}
	if _, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	return capturedRun.Metadata
}

// TestLaunchRun_ImageOnly_BackCompat proves the hard invariant: an agent with only a
// free-form image (no StackRef) threads that image as step_0_agent_image — even when a
// stack repo is wired, it is not consulted.
func TestLaunchRun_ImageOnly_BackCompat(t *testing.T) {
	agent := &model.Agent{ID: testAgentIDImplement, Model: "claude-opus-4-6", Image: "hopeitworks/agent-go-node:latest"}
	stackRepo := newMockStackRepo() // wired but must stay untouched (agent has no StackRef)

	meta := launchRunCapture(t, agent, stackRepo)

	img, ok := meta["step_0_agent_image"].(string)
	if !ok {
		t.Fatal("expected step_0_agent_image in metadata")
	}
	if img != "hopeitworks/agent-go-node:latest" {
		t.Errorf("expected free-form image to be threaded unchanged, got %q", img)
	}
}

// TestLaunchRun_StackRef_ResolvesImage proves a stack-referencing agent resolves its
// launch image from the catalogue (stacks.image_ref), overriding the free-form image.
func TestLaunchRun_StackRef_ResolvesImage(t *testing.T) {
	stackID := uuid.New()
	stackRepo := newMockStackRepo()
	stackRepo.stacks[stackID] = &model.Stack{ID: stackID, Key: model.StackKeyGo, ImageRef: "ghcr.io/zkarahacane/hopeitworks/agent-go:pinned"}

	agent := &model.Agent{
		ID:       testAgentIDImplement,
		Model:    "claude-opus-4-6",
		Image:    "should-be-ignored:latest",
		StackRef: &stackID,
	}

	meta := launchRunCapture(t, agent, stackRepo)

	img, ok := meta["step_0_agent_image"].(string)
	if !ok {
		t.Fatal("expected step_0_agent_image in metadata")
	}
	if img != "ghcr.io/zkarahacane/hopeitworks/agent-go:pinned" {
		t.Errorf("expected stack image_ref to be resolved, got %q", img)
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

	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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

	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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

	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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

	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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

	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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
	svc.SetAgentRepo(newTestAgentRepo())
	_, err := svc.LaunchRun(context.Background(), projectID, storyID, uuid.Nil)
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
			if domainErr.Code != errors.ErrCodeInvalidStateTransition {
				t.Errorf("expected %s code, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
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
			if domainErr.Code != errors.ErrCodeInvalidStateTransition {
				t.Errorf("expected %s code, got %s", errors.ErrCodeInvalidStateTransition, domainErr.Code)
			}
		})
	}
}

// ── Progress Computation Tests ───────────────────────────────────────────────

func TestGetRun_ProgressComputed(t *testing.T) {
	runID := uuid.New()

	tests := []struct {
		name             string
		steps            []*model.RunStep
		expectedProgress int
	}{
		{
			name:             "no steps returns 0",
			steps:            []*model.RunStep{},
			expectedProgress: 0,
		},
		{
			name: "1 of 2 completed returns 50",
			steps: []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusRunning},
			},
			expectedProgress: 50,
		},
		{
			name: "2 of 3 completed returns 66",
			steps: []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusRunning},
			},
			expectedProgress: 66,
		},
		{
			name: "all completed returns 100",
			steps: []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusCompleted},
			},
			expectedProgress: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runRepo := &mockRunRepo{
				getRunFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
					return &model.Run{
						ID:     runID,
						Status: model.RunStatusRunning,
					}, nil
				},
				listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
					return tt.steps, nil
				},
			}
			svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

			run, err := svc.GetRun(context.Background(), runID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if run.Progress != tt.expectedProgress {
				t.Errorf("expected progress %d, got %d", tt.expectedProgress, run.Progress)
			}
		})
	}
}

func TestListRunsByProject_ProgressPopulated(t *testing.T) {
	projectID := uuid.New()
	run1ID := uuid.New()
	run2ID := uuid.New()

	runRepo := &mockRunRepo{
		listRunsByProjectFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
			return []*model.Run{
				{ID: run1ID, Status: model.RunStatusRunning},
				{ID: run2ID, Status: model.RunStatusCompleted},
			}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, rID uuid.UUID) ([]*model.RunStep, error) {
			if rID == run1ID {
				return []*model.RunStep{
					{ID: uuid.New(), Status: model.StepStatusCompleted},
					{ID: uuid.New(), Status: model.StepStatusPending},
				}, nil
			}
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusCompleted},
			}, nil
		},
		countRunsByProjectFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 2, nil
		},
	}
	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	result, err := svc.ListRunsByProject(context.Background(), projectID, 1, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(result.Runs))
	}
	if result.Runs[0].Progress != 50 {
		t.Errorf("run1: expected progress 50, got %d", result.Runs[0].Progress)
	}
	if result.Runs[1].Progress != 100 {
		t.Errorf("run2: expected progress 100, got %d", result.Runs[1].Progress)
	}
}

// ── RetryStep Tests ──────────────────────────────────────────────────────────

func TestRetryStep_Success(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()
	errMsg := "agent failed"

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusFailed,
			}, nil
		},
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:           id,
				RunID:        runID,
				StepName:     "implement",
				StepOrder:    0,
				Action:       "agent_run",
				Status:       model.StepStatusFailed,
				ErrorMessage: &errMsg,
				RetryCount:   0,
			}, nil
		},
		listRetryStepsByParentFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return nil, nil // no existing retries
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
		createRetryRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			return step, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: stepID, Status: model.StepStatusFailed},
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

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, jobQueue)
	run, err := svc.RetryStep(context.Background(), runID, stepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if run == nil {
		t.Fatal("expected run, got nil")
	}
	if enqueuedRunID != runID {
		t.Errorf("expected enqueued run ID %s, got %s", runID, enqueuedRunID)
	}
}

func TestRetryStep_StepNotFailed(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, Status: model.RunStatusCompleted}, nil
		},
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:     id,
				RunID:  runID,
				Status: model.StepStatusCompleted,
			}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
	_, err := svc.RetryStep(context.Background(), runID, stepID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "STEP_NOT_FAILED" {
		t.Errorf("expected STEP_NOT_FAILED code, got %s", domainErr.Code)
	}
}

func TestRetryStep_StepNotInRun(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	otherRunID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, Status: model.RunStatusFailed}, nil
		},
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:     id,
				RunID:  otherRunID, // different run
				Status: model.StepStatusFailed,
			}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
	_, err := svc.RetryStep(context.Background(), runID, stepID)
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

func TestRetryStep_MaxRetriesExceeded(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, Status: model.RunStatusFailed}, nil
		},
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:         id,
				RunID:      runID,
				Status:     model.StepStatusFailed,
				RetryCount: 0,
			}, nil
		},
		listRetryStepsByParentFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			// 3 existing retries means max exceeded (default max is 3)
			return []*model.RunStep{
				{ID: uuid.New(), RetryCount: 1},
				{ID: uuid.New(), RetryCount: 2},
				{ID: uuid.New(), RetryCount: 3},
			}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil)
	_, err := svc.RetryStep(context.Background(), runID, stepID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != "RETRY_MAX_EXCEEDED" {
		t.Errorf("expected RETRY_MAX_EXCEEDED code, got %s", domainErr.Code)
	}
}

// TestRetryStep_StageStampedFromParent verifies that a retry step carries the
// StageID and StageName of its parent step (Dette D1 fix).
func TestRetryStep_StageStampedFromParent(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()
	errMsg := "agent failed"

	const wantStageID = "stage-dev"
	const wantStageName = "Development"

	var capturedStep *model.RunStep

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusFailed,
			}, nil
		},
		getRunStepFn: func(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
			return &model.RunStep{
				ID:           id,
				RunID:        runID,
				StepName:     "implement",
				StepOrder:    0,
				Action:       "agent_run",
				Status:       model.StepStatusFailed,
				ErrorMessage: &errMsg,
				RetryCount:   0,
				StageID:      wantStageID,
				StageName:    wantStageName,
			}, nil
		},
		listRetryStepsByParentFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return nil, nil // no existing retries
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
		createRetryRunStepFn: func(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
			capturedStep = step
			return step, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: stepID, Status: model.StepStatusFailed},
			}, nil
		},
	}

	jobQueue := &mockJobQueue{
		enqueueExecuteRunFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, jobQueue)
	_, err := svc.RetryStep(context.Background(), runID, stepID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedStep == nil {
		t.Fatal("createRetryRunStepFn was not called")
	}
	if capturedStep.StageID != wantStageID {
		t.Errorf("retry step StageID = %q; want %q", capturedStep.StageID, wantStageID)
	}
	if capturedStep.StageName != wantStageName {
		t.Errorf("retry step StageName = %q; want %q", capturedStep.StageName, wantStageName)
	}
}

func TestListRunsByStory_ProgressPopulated(t *testing.T) {
	storyID := uuid.New()
	run1ID := uuid.New()

	runRepo := &mockRunRepo{
		listRunsByStoryFn: func(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
			return []*model.Run{
				{ID: run1ID, Status: model.RunStatusRunning},
			}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusPending},
			}, nil
		},
		countRunsByStoryFn: func(_ context.Context, _ uuid.UUID) (int64, error) {
			return 1, nil
		},
	}
	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	result, err := svc.ListRunsByStory(context.Background(), storyID, 1, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(result.Runs))
	}
	if result.Runs[0].Progress != 66 {
		t.Errorf("expected progress 66, got %d", result.Runs[0].Progress)
	}
}

func TestCancelRun_RunningRun(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	containerID := "container-123"
	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusRunning,
			}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusRunning, ContainerID: &containerID},
				{ID: uuid.New(), Status: model.StepStatusPending},
			}, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	var stoppedContainerID string
	containerMgr := &mockContainerManager{
		stopFn: func(_ context.Context, cid string) error {
			stoppedContainerID = cid
			return nil
		},
	}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())
	svc.SetContainerManager(containerMgr)

	result, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != model.RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}
	if stoppedContainerID != containerID {
		t.Errorf("expected container %s to be stopped, got %s", containerID, stoppedContainerID)
	}
}

func TestCancelRun_PendingRun(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	stepsCancelled := 0
	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusPending,
			}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusPending},
				{ID: uuid.New(), Status: model.StepStatusPending},
			}, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			stepsCancelled++
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	result, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != model.RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}
	if stepsCancelled != 2 {
		t.Errorf("expected 2 steps cancelled, got %d", stepsCancelled)
	}
}

func TestCancelRun_AlreadyCompleted(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusCompleted,
			}, nil
		},
	}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	_, err := svc.CancelRun(context.Background(), projectID, runID)
	if err == nil {
		t.Fatal("expected error for completed run, got nil")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Code != errors.ErrCodeInvalidStateTransition {
		t.Errorf("expected error code %s, got %s", errors.ErrCodeInvalidStateTransition, domErr.Code)
	}
}

func TestCancelRun_WrongProject(t *testing.T) {
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

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	_, err := svc.CancelRun(context.Background(), projectID, runID)
	if err == nil {
		t.Fatal("expected error for wrong project, got nil")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domErr.Category != errors.CategoryNotFound {
		t.Errorf("expected category not_found, got %s", domErr.Category)
	}
}

func TestCancelRun_PausedRun(t *testing.T) {
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
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusCompleted},
				{ID: uuid.New(), Status: model.StepStatusPending},
			}, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())

	result, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != model.RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}
}

func TestCancelRun_PublishesEvent(t *testing.T) {
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
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return nil, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	eventPub := newMockEventPublisher()
	svc := NewRunService(runRepo, newMockProjectRepoForService(), nil, nil, nil, eventPub)

	_, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	events := eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event.Action != "cancelled" {
		t.Errorf("expected event action 'cancelled', got %s", events[0].Event.Action)
	}
}

// TestCancelRun_ResetsStoryToBacklog verifies cancelling a run resets its story
// to backlog (relaunchable) and emits the corresponding events with story_id.
func TestCancelRun_ResetsStoryToBacklog(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	storyID := uuid.New()

	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, StoryID: storyID, Status: model.RunStatusRunning}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return nil, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, StoryID: storyID, Status: status}, nil
		},
	}

	var updatedStatus string
	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Status: model.StoryStatusRunning}, nil
		},
		updateFn: func(_ context.Context, s *model.Story) (*model.Story, error) {
			updatedStatus = s.Status
			return s, nil
		},
	}

	eventPub := newMockEventPublisher()
	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, nil, nil, eventPub)

	_, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updatedStatus != model.StoryStatusBacklog {
		t.Errorf("expected story reset to backlog, got %q", updatedStatus)
	}

	var foundCancelledWithStory, foundStoryBacklog bool
	for _, e := range eventPub.getEvents() {
		var payload map[string]any
		_ = json.Unmarshal(e.Event.Payload, &payload)
		switch e.Event.EventName() {
		case eventRunCancelled:
			if payload["story_id"] == storyID.String() {
				foundCancelledWithStory = true
			}
		case "story.status_updated":
			if payload["status"] == model.StoryStatusBacklog && payload["story_id"] == storyID.String() {
				foundStoryBacklog = true
			}
		}
	}
	if !foundCancelledWithStory {
		t.Error("expected run.cancelled event with story_id")
	}
	if !foundStoryBacklog {
		t.Error("expected story.status_updated event with status backlog")
	}
}

// mockAgentRuntime is a minimal port.AgentRuntime recording Stop handles. Only Stop
// is exercised by CancelRun; the other methods satisfy the interface and are no-ops.
type mockAgentRuntime struct {
	stopHandles []port.RunHandle
}

func (m *mockAgentRuntime) Provision(_ context.Context, _ model.CapabilitySpec) (model.ProvisionResult, error) {
	return model.ProvisionResult{}, nil
}

func (m *mockAgentRuntime) Launch(_ context.Context, _ port.RunSpec) (port.RunHandle, error) {
	return port.RunHandle{}, nil
}

func (m *mockAgentRuntime) Wait(_ context.Context, _ port.RunHandle) (port.RunResult, error) {
	return port.RunResult{}, nil
}

func (m *mockAgentRuntime) Stop(_ context.Context, h port.RunHandle) error {
	m.stopHandles = append(m.stopHandles, h)
	return nil
}

func (m *mockAgentRuntime) SupportedCapabilities() model.CapabilitySet {
	return model.CapabilitySet{}
}

// TestCancelRun_StopsViaAgentRuntime asserts that, when an AgentRuntime is wired,
// CancelRun stops the running step's execution through the runtime port (handle =
// the persisted container_id) and does NOT touch the raw ContainerManager.
func TestCancelRun_StopsViaAgentRuntime(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	containerID := "cid-1"
	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: model.RunStatusRunning}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusRunning, ContainerID: &containerID},
			}, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	mockRT := &mockAgentRuntime{}
	containerMgr := &mockContainerManager{}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())
	// Wire BOTH: the runtime must win over the raw ContainerManager.
	svc.SetContainerManager(containerMgr)
	svc.SetAgentRuntime(mockRT)

	result, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != model.RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}

	if len(mockRT.stopHandles) != 1 {
		t.Fatalf("expected 1 runtime Stop call, got %d", len(mockRT.stopHandles))
	}
	if got := mockRT.stopHandles[0]; got != (port.RunHandle{ID: containerID}) {
		t.Errorf("expected runtime Stop with handle %q, got %+v", containerID, got)
	}
	if calls := containerMgr.getStopCalls(); len(calls) != 0 {
		t.Errorf("expected ContainerManager.Stop NOT called when runtime is present, got %d calls", len(calls))
	}
}

// TestCancelRun_FallsBackToContainerManager asserts that, with no AgentRuntime
// configured, CancelRun stops the running step through the raw ContainerManager
// (Docker back-compat).
func TestCancelRun_FallsBackToContainerManager(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	containerID := "cid-1"
	runRepo := &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: model.RunStatusRunning}, nil
		},
		listRunStepsByRunFn: func(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
			return []*model.RunStep{
				{ID: uuid.New(), Status: model.StepStatusRunning, ContainerID: &containerID},
			}, nil
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{ID: id, ProjectID: projectID, Status: status}, nil
		},
	}

	containerMgr := &mockContainerManager{}

	svc := newRunServiceForTest(runRepo, newMockProjectRepoForService())
	svc.SetContainerManager(containerMgr) // no runtime → fallback

	result, err := svc.CancelRun(context.Background(), projectID, runID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Status != model.RunStatusCancelled {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}

	calls := containerMgr.getStopCalls()
	if len(calls) != 1 || calls[0] != containerID {
		t.Errorf("expected ContainerManager.Stop(%q) once, got %v", containerID, calls)
	}
}

func (m *mockStoryRepoForRun) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

func (m *mockRunRepo) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *mockRunRepo) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *mockRunRepo) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}

func (m *mockRunRepo) UpdateRunMetadata(ctx context.Context, runID uuid.UUID, metadata map[string]interface{}) error {
	if m.updateRunMetadataFn != nil {
		return m.updateRunMetadataFn(ctx, runID, metadata)
	}
	return nil
}

func (m *mockRunRepo) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// TestRunService_StartStage_ResumesParkedManualRun verifies StartStage flags the
// parked stage started, clears the manual-start reason, transitions the run back to
// running and re-enqueues it.
func TestRunService_StartStage_ResumesParkedManualRun(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	runID := uuid.New()

	parkedRun := &model.Run{
		ID:        runID,
		ProjectID: projectID,
		StoryID:   storyID,
		Status:    model.RunStatusPaused,
		Metadata: map[string]interface{}{
			"paused_reason":     pausedReasonManualStart,
			"paused_stage_id":   "review",
			"paused_stage_name": "Review",
		},
	}

	var savedMeta map[string]interface{}
	var enqueued bool
	var resumedStatus model.RunStatus
	runRepo := &mockRunRepo{
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return parkedRun, nil
		},
		updateRunMetadataFn: func(_ context.Context, _ uuid.UUID, m map[string]interface{}) error {
			savedMeta = m
			return nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			resumedStatus = status
			return &model.Run{ID: id, ProjectID: projectID, StoryID: storyID, Status: status}, nil
		},
	}
	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-01", Status: model.StoryStatusRunning}, nil
		},
	}
	jobQueue := &mockJobQueue{enqueueExecuteRunFn: func(_ context.Context, _ uuid.UUID) error {
		enqueued = true
		return nil
	}}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, nil, jobQueue)

	updated, err := svc.StartStage(context.Background(), projectID, storyID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != model.RunStatusRunning || resumedStatus != model.RunStatusRunning {
		t.Errorf("expected run resumed to running, got %s", updated.Status)
	}
	if !enqueued {
		t.Error("expected run to be re-enqueued for execution")
	}
	if started, ok := savedMeta[stageStartedKey("review")].(bool); !ok || !started {
		t.Errorf("expected stage_started_review=true in metadata, got %v", savedMeta)
	}
	if _, exists := savedMeta["paused_reason"]; exists {
		t.Error("expected paused_reason to be cleared from metadata")
	}
}

// TestRunService_StartStage_RejectsRunningRun verifies StartStage 409s when the run
// is not paused awaiting a manual start.
func TestRunService_StartStage_RejectsRunningRun(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	runRepo := &mockRunRepo{
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: uuid.New(), ProjectID: projectID, StoryID: storyID, Status: model.RunStatusRunning}, nil
		},
	}
	storyRepo := &mockStoryRepoForRun{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{ID: id, ProjectID: projectID, Key: "S-01", Status: model.StoryStatusRunning}, nil
		},
	}

	svc := NewRunService(runRepo, newMockProjectRepoForService(), storyRepo, nil, &mockJobQueue{})

	_, err := svc.StartStage(context.Background(), projectID, storyID)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	domErr, ok := err.(*errors.DomainError)
	if !ok || domErr.Category != errors.CategoryConflict {
		t.Errorf("expected conflict DomainError, got %v", err)
	}
}
