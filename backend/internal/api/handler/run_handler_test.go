package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ── Minimal mock repo implementations for RunHandler tests ──────────────────

// runHandlerRunRepo is a minimal mock of port.RunRepository for handler tests.
type runHandlerRunRepo struct {
	createRunFn           func(ctx context.Context, run *model.Run) (*model.Run, error)
	createRunStepFn       func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
	getActiveRunByStoryFn func(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
	getRunFn              func(ctx context.Context, id uuid.UUID) (*model.Run, error)
	updateRunStatusFn     func(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errMsg *string) (*model.Run, error)
}

var _ port.RunRepository = (*runHandlerRunRepo)(nil)

func (m *runHandlerRunRepo) CreateRun(ctx context.Context, run *model.Run) (*model.Run, error) {
	if m.createRunFn != nil {
		return m.createRunFn(ctx, run)
	}
	run.ID = uuid.New()
	run.CreatedAt = time.Now()
	run.UpdatedAt = time.Now()
	return run, nil
}
func (m *runHandlerRunRepo) GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error) {
	if m.getRunFn != nil {
		return m.getRunFn(ctx, id)
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *runHandlerRunRepo) GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error) {
	if m.getActiveRunByStoryFn != nil {
		return m.getActiveRunByStoryFn(ctx, storyID)
	}
	return nil, nil
}
func (m *runHandlerRunRepo) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *runHandlerRunRepo) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *runHandlerRunRepo) UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, startedAt, completedAt, pausedAt *time.Time, errMsg *string) (*model.Run, error) {
	if m.updateRunStatusFn != nil {
		return m.updateRunStatusFn(ctx, id, status, startedAt, completedAt, pausedAt, errMsg)
	}
	return nil, nil
}
func (m *runHandlerRunRepo) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *runHandlerRunRepo) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *runHandlerRunRepo) CreateRunStep(ctx context.Context, step *model.RunStep) (*model.RunStep, error) {
	if m.createRunStepFn != nil {
		return m.createRunStepFn(ctx, step)
	}
	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	return step, nil
}
func (m *runHandlerRunRepo) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	return nil, errors.NewNotFound("run_step", id)
}
func (m *runHandlerRunRepo) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *runHandlerRunRepo) UpdateRunStepStatus(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *runHandlerRunRepo) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *runHandlerRunRepo) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *runHandlerRunRepo) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

// runHandlerStoryRepo is a minimal mock of port.StoryRepository for handler tests.
type runHandlerStoryRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}

var _ port.StoryRepository = (*runHandlerStoryRepo)(nil)

func (m *runHandlerStoryRepo) Create(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *runHandlerStoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, errors.NewNotFound("story", id)
}
func (m *runHandlerStoryRepo) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return nil, nil
}
func (m *runHandlerStoryRepo) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *runHandlerStoryRepo) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *runHandlerStoryRepo) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *runHandlerStoryRepo) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *runHandlerStoryRepo) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *runHandlerStoryRepo) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *runHandlerStoryRepo) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

// runHandlerPipelineConfigRepo is a minimal mock of port.PipelineConfigRepository.
type runHandlerPipelineConfigRepo struct {
	getByProjectIDFn func(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
}

var _ port.PipelineConfigRepository = (*runHandlerPipelineConfigRepo)(nil)

func (m *runHandlerPipelineConfigRepo) GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error) {
	if m.getByProjectIDFn != nil {
		return m.getByProjectIDFn(ctx, projectID)
	}
	return nil, errors.NewNotFound("pipeline_config", projectID)
}
func (m *runHandlerPipelineConfigRepo) Upsert(_ context.Context, cfg *model.PipelineConfig) (*model.PipelineConfig, error) {
	return cfg, nil
}

// runHandlerProjectRepo is a minimal mock of port.ProjectRepository.
type runHandlerProjectRepo struct{}

var _ port.ProjectRepository = (*runHandlerProjectRepo)(nil)

func (m *runHandlerProjectRepo) Create(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *runHandlerProjectRepo) GetByID(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *runHandlerProjectRepo) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *runHandlerProjectRepo) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *runHandlerProjectRepo) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *runHandlerProjectRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *runHandlerProjectRepo) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}
func (m *runHandlerProjectRepo) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return &model.Project{}, nil
}

// runHandlerJobQueue is a minimal mock of port.JobQueue.
type runHandlerJobQueue struct {
	enqueueExecuteRunFn func(ctx context.Context, runID uuid.UUID) error
}

var _ port.JobQueue = (*runHandlerJobQueue)(nil)

func (m *runHandlerJobQueue) EnqueueExecuteRun(ctx context.Context, runID uuid.UUID) error {
	if m.enqueueExecuteRunFn != nil {
		return m.enqueueExecuteRunFn(ctx, runID)
	}
	return nil
}

// handlerTestAgentID is the fixed agent UUID used in handlerTestPipelineYAML.
var handlerTestAgentID = uuid.MustParse("00000000-0000-0000-0000-000000000099")

// handlerTestPipelineYAML is a minimal valid pipeline config for handler tests.
const handlerTestPipelineYAML = `steps:
  - id: "step-1"
    name: "implement"
    action_type: "implement"
    agent_id: "00000000-0000-0000-0000-000000000099"
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: "none"
`

// setupRunHandler constructs a RunHandler backed by the provided mocks.
func setupRunHandler(
	runRepo port.RunRepository,
	storyRepo port.StoryRepository,
	pipelineConfigRepo port.PipelineConfigRepository,
	jobQueue port.JobQueue,
) *RunHandler {
	svc := service.NewRunService(runRepo, &runHandlerProjectRepo{}, storyRepo, pipelineConfigRepo, jobQueue)
	return NewRunHandler(svc)
}

// ── LaunchRun handler tests ──────────────────────────────────────────────────

func TestLaunchRunHandler_Created(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &runHandlerStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusBacklog,
			}, nil
		},
	}
	pipelineConfigRepo := &runHandlerPipelineConfigRepo{
		getByProjectIDFn: func(_ context.Context, _ uuid.UUID) (*model.PipelineConfig, error) {
			return &model.PipelineConfig{
				ID:         uuid.New(),
				ProjectID:  projectID,
				ConfigYAML: handlerTestPipelineYAML,
				Version:    1,
			}, nil
		},
	}

	agentRepo := newMockAgentRepo()
	agentRepo.agents[handlerTestAgentID] = &model.Agent{
		ID:    handlerTestAgentID,
		Model: "claude-opus-4-6",
		Image: "hopeitworks/agent:latest",
	}
	svc := service.NewRunService(&runHandlerRunRepo{}, &runHandlerProjectRepo{}, storyRepo, pipelineConfigRepo, &runHandlerJobQueue{})
	svc.SetAgentRepo(agentRepo)
	h := NewRunHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String()+"/runs", nil)
	rec := httptest.NewRecorder()

	h.LaunchRun(rec, req, projectID, storyID)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "pending" {
		t.Errorf("expected run status 'pending', got %v", result["status"])
	}
	steps, ok := result["steps"].([]interface{})
	if !ok {
		t.Fatalf("expected steps array in response, got %T", result["steps"])
	}
	if len(steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(steps))
	}
}

func TestLaunchRunHandler_StoryNotFound(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	h := setupRunHandler(
		&runHandlerRunRepo{},
		&runHandlerStoryRepo{},
		&runHandlerPipelineConfigRepo{},
		&runHandlerJobQueue{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String()+"/runs", nil)
	rec := httptest.NewRecorder()

	h.LaunchRun(rec, req, projectID, storyID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestLaunchRunHandler_AlreadyRunning(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	activeRunID := uuid.New()

	storyRepo := &runHandlerStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusRunning,
			}, nil
		},
	}
	runRepo := &runHandlerRunRepo{
		getActiveRunByStoryFn: func(_ context.Context, _ uuid.UUID) (*model.Run, error) {
			return &model.Run{ID: activeRunID, Status: model.RunStatusRunning}, nil
		},
	}

	h := setupRunHandler(runRepo, storyRepo, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String()+"/runs", nil)
	rec := httptest.NewRecorder()

	h.LaunchRun(rec, req, projectID, storyID)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestLaunchRunHandler_AlreadyCompleted(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()

	storyRepo := &runHandlerStoryRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Story, error) {
			return &model.Story{
				ID:        id,
				ProjectID: projectID,
				Key:       "S-01",
				Status:    model.StoryStatusDone,
			}, nil
		},
	}

	h := setupRunHandler(&runHandlerRunRepo{}, storyRepo, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/stories/"+storyID.String()+"/runs", nil)
	rec := httptest.NewRecorder()

	h.LaunchRun(rec, req, projectID, storyID)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ── PauseRun handler tests ──────────────────────────────────────────────────

func TestPauseRunHandler_Success(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusRunning,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, pausedAt *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    status,
				PausedAt:  pausedAt,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/pause", nil)
	rec := httptest.NewRecorder()

	h.PauseRun(rec, req, projectID, runID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "paused" {
		t.Errorf("expected run status 'paused', got %v", result["status"])
	}
}

func TestPauseRunHandler_NotFound(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	h := setupRunHandler(&runHandlerRunRepo{}, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/pause", nil)
	rec := httptest.NewRecorder()

	h.PauseRun(rec, req, projectID, runID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestPauseRunHandler_InvalidState(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusCompleted,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/pause", nil)
	rec := httptest.NewRecorder()

	h.PauseRun(rec, req, projectID, runID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestPauseRunHandler_WrongProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: otherProjectID,
				Status:    model.RunStatusRunning,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/pause", nil)
	rec := httptest.NewRecorder()

	h.PauseRun(rec, req, projectID, runID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// ── ResumeRun handler tests ──────────────────────────────────────────────────

func TestResumeRunHandler_Success(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusPaused,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
		updateRunStatusFn: func(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    status,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/resume", nil)
	rec := httptest.NewRecorder()

	h.ResumeRun(rec, req, projectID, runID)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["status"] != "running" {
		t.Errorf("expected run status 'running', got %v", result["status"])
	}
}

func TestResumeRunHandler_NotFound(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	h := setupRunHandler(&runHandlerRunRepo{}, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/resume", nil)
	rec := httptest.NewRecorder()

	h.ResumeRun(rec, req, projectID, runID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestResumeRunHandler_InvalidState(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: projectID,
				Status:    model.RunStatusRunning,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/resume", nil)
	rec := httptest.NewRecorder()

	h.ResumeRun(rec, req, projectID, runID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestResumeRunHandler_WrongProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	runID := uuid.New()

	runRepo := &runHandlerRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			return &model.Run{
				ID:        id,
				ProjectID: otherProjectID,
				Status:    model.RunStatusPaused,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	h := setupRunHandler(runRepo, &runHandlerStoryRepo{}, &runHandlerPipelineConfigRepo{}, &runHandlerJobQueue{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/resume", nil)
	rec := httptest.NewRecorder()

	h.ResumeRun(rec, req, projectID, runID)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
