package service

import (
	"context"
	"encoding/json"
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
	listRunsByProjectFn   func(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	listRunsByStoryFn     func(ctx context.Context, storyID uuid.UUID, limit, offset int32) ([]*model.Run, error)
	updateRunStatusFn     func(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.Run, error)
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
func (m *mockRunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt *time.Time, errorMsg *string) (*model.Run, error) {
	if m.updateRunStatusFn != nil {
		return m.updateRunStatusFn(ctx, id, status, startedAt, completedAt, errorMsg)
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

// newMockProjectRepoWithProject creates a mockProjectRepo preloaded with a project.
func newMockProjectRepoWithProject(project *model.Project) *mockProjectRepo {
	repo := newMockProjectRepoForService()
	repo.projects[project.ID] = project
	return repo
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

	svc := NewRunService(runRepo, projectRepo)
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
	// Empty project repo — project not found
	projectRepo := newMockProjectRepoForService()

	svc := NewRunService(runRepo, projectRepo)
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
			svc := NewRunService(&mockRunRepo{}, projectRepo)
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
	svc := NewRunService(&mockRunRepo{}, newMockProjectRepoForService())

	// Missing project_id
	_, err := svc.CreateRun(context.Background(), CreateRunParams{
		StoryID:        uuid.New(),
		PipelineConfig: json.RawMessage(`{"steps":[{"name":"dev","action":"code"}]}`),
	})
	if err == nil {
		t.Fatal("expected error for missing project_id")
	}

	// Missing story_id
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
				updateRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.RunStatus, startedAt, completedAt *time.Time, _ *string) (*model.Run, error) {
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
			svc := NewRunService(runRepo, newMockProjectRepoForService())

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
	svc := NewRunService(runRepo, newMockProjectRepoForService())

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
			svc := NewRunService(runRepo, newMockProjectRepoForService())

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
	svc := NewRunService(runRepo, newMockProjectRepoForService())

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
