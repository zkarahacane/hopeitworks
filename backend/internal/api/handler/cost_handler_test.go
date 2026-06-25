package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- Mock implementations for cost handler tests ---

type mockCostRepoForHandler struct {
	listByProjectByAgentFn             func(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error)
	listByProjectByRoleFn              func(ctx context.Context, projectID uuid.UUID) ([]model.ProjectRoleCostBreakdown, error)
	listCostsByRunByRoleFn             func(ctx context.Context, runID uuid.UUID) ([]model.RoleCostBreakdown, error)
	listCostsByProjectByRunPaginatedFn func(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error)
	countCostsByProjectByRunFn         func(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error)
	sumCostByRunFn                     func(ctx context.Context, runID uuid.UUID) (float64, error)
	sumTokensByRunFn                   func(ctx context.Context, runID uuid.UUID) (int64, int64, error)
}

func (m *mockCostRepoForHandler) InsertCostRecord(_ context.Context, r *model.CostRecord) (*model.CostRecord, error) {
	r.ID = uuid.New()
	return r, nil
}
func (m *mockCostRepoForHandler) GetCostByRunStep(_ context.Context, _ uuid.UUID) (*model.CostRecord, error) {
	return nil, errors.NewNotFound("cost_record", uuid.Nil)
}
func (m *mockCostRepoForHandler) SumCostByProject(_ context.Context, _ uuid.UUID, _ time.Time) (float64, int64, int64, error) {
	return 0, 0, 0, nil
}
func (m *mockCostRepoForHandler) SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error) {
	if m.sumCostByRunFn != nil {
		return m.sumCostByRunFn(ctx, runID)
	}
	return 0, nil
}
func (m *mockCostRepoForHandler) SumCostByStory(_ context.Context, _ uuid.UUID) (float64, int64, int64, int, error) {
	return 0, 0, 0, 0, nil
}
func (m *mockCostRepoForHandler) ListCostsByProjectByStory(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.StoryCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepoForHandler) ListCostsByProjectByRun(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.RunCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepoForHandler) ListCostsByProjectByModel(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostByModel, error) {
	return nil, nil
}
func (m *mockCostRepoForHandler) ListStepCostsByRun(_ context.Context, _ uuid.UUID) ([]model.StepCostBreakdown, error) {
	return nil, nil
}
func (m *mockCostRepoForHandler) ListDailyCostsByProject(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostDataPoint, error) {
	return nil, nil
}
func (m *mockCostRepoForHandler) ListCostsByProjectByRunPaginated(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error) {
	if m.listCostsByProjectByRunPaginatedFn != nil {
		return m.listCostsByProjectByRunPaginatedFn(ctx, projectID, since, limit, offset)
	}
	return nil, nil
}
func (m *mockCostRepoForHandler) CountCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error) {
	if m.countCostsByProjectByRunFn != nil {
		return m.countCostsByProjectByRunFn(ctx, projectID, since)
	}
	return 0, nil
}
func (m *mockCostRepoForHandler) ListByProjectByAgent(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error) {
	if m.listByProjectByAgentFn != nil {
		return m.listByProjectByAgentFn(ctx, projectID)
	}
	return []model.AgentCostBreakdown{}, nil
}

func (m *mockCostRepoForHandler) ListByProjectByRole(ctx context.Context, projectID uuid.UUID) ([]model.ProjectRoleCostBreakdown, error) {
	if m.listByProjectByRoleFn != nil {
		return m.listByProjectByRoleFn(ctx, projectID)
	}
	return []model.ProjectRoleCostBreakdown{}, nil
}

func (m *mockCostRepoForHandler) ListCostsByRunByRole(ctx context.Context, runID uuid.UUID) ([]model.RoleCostBreakdown, error) {
	if m.listCostsByRunByRoleFn != nil {
		return m.listCostsByRunByRoleFn(ctx, runID)
	}
	return []model.RoleCostBreakdown{}, nil
}

func (m *mockCostRepoForHandler) SumTokensByRun(ctx context.Context, runID uuid.UUID) (int64, int64, error) {
	if m.sumTokensByRunFn != nil {
		return m.sumTokensByRunFn(ctx, runID)
	}
	return 0, 0, nil
}

type mockProjectRepoForCostHandler struct {
	project *model.Project
	err     error
}

func (m *mockProjectRepoForCostHandler) Create(_ context.Context, _ *model.Project) (*model.Project, error) {
	return m.project, m.err
}
func (m *mockProjectRepoForCostHandler) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.project != nil && m.project.ID == id {
		return m.project, nil
	}
	return nil, errors.NewNotFound("project", id)
}
func (m *mockProjectRepoForCostHandler) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepoForCostHandler) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *mockProjectRepoForCostHandler) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *mockProjectRepoForCostHandler) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockProjectRepoForCostHandler) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepoForCostHandler) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

func setupCostHandler(costRepo *mockCostRepoForHandler, projectRepo *mockProjectRepoForCostHandler) *CostHandler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := service.NewCostService(costRepo, projectRepo, nil, nil, logger)
	return NewCostHandler(svc)
}

func setupCostHandlerWithRunRepo(costRepo *mockCostRepoForHandler, runRepo *epicRunRepo) *CostHandler {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := service.NewCostService(costRepo, nil, nil, runRepo, logger)
	return NewCostHandler(svc)
}

// --- GetProjectCostsByAgent handler tests ---

func TestGetProjectCostsByAgent_200WithResults(t *testing.T) {
	projectID := uuid.New()
	agentID := uuid.New()

	costRepo := &mockCostRepoForHandler{
		listByProjectByAgentFn: func(_ context.Context, _ uuid.UUID) ([]model.AgentCostBreakdown, error) {
			return []model.AgentCostBreakdown{
				{AgentID: agentID, AgentName: "dev-story", TokensInput: 500000, TokensOutput: 100000, CostUSD: 12.50, RunsCount: 3},
				{AgentID: uuid.New(), AgentName: "code-review", TokensInput: 200000, TokensOutput: 50000, CostUSD: 5.25, RunsCount: 2},
			}, nil
		},
	}
	projectRepo := &mockProjectRepoForCostHandler{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	h := setupCostHandler(costRepo, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/agents", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByAgent(rec, req, projectID)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result []AgentCostBreakdown
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	require.Len(t, result, 2)

	assert.Equal(t, agentID, uuid.UUID(result[0].AgentId))
	assert.Equal(t, "dev-story", result[0].AgentName)
	assert.Equal(t, int64(500000), result[0].TokensInput)
	assert.Equal(t, int64(100000), result[0].TokensOutput)
	assert.InDelta(t, 12.50, result[0].CostUsd, 0.001)
	assert.Equal(t, int32(3), result[0].RunsCount)
}

func TestGetProjectCostsByAgent_200EmptyResults(t *testing.T) {
	projectID := uuid.New()

	projectRepo := &mockProjectRepoForCostHandler{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	h := setupCostHandler(&mockCostRepoForHandler{}, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/agents", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByAgent(rec, req, projectID)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result []AgentCostBreakdown
	err := json.NewDecoder(rec.Body).Decode(&result)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetProjectCostsByAgent_404ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCostHandler{
		err: errors.NewNotFound("project", uuid.New()),
	}
	h := setupCostHandler(&mockCostRepoForHandler{}, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+uuid.New().String()+"/costs/agents", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByAgent(rec, req, uuid.New())

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- GetProjectCostRuns token serialization (RG2) ---

// TestGetProjectCostRuns_SerializesTokens proves the Recent Runs rows carry the
// real token sums in the JSON response (not 0/omitted), so the front shows them.
func TestGetProjectCostRuns_SerializesTokens(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	costRepo := &mockCostRepoForHandler{}
	costRepo.listCostsByProjectByRunPaginatedFn = func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ int32) ([]model.RunCostRow, error) {
		return []model.RunCostRow{
			{RunID: runID, StoryKey: "S-01", Status: "completed", StartedAt: time.Now(), TotalCostUSD: 5.0, TokensInput: 120000, TokensOutput: 30000},
		}, nil
	}
	costRepo.countCostsByProjectByRunFn = func(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
		return 1, nil
	}
	projectRepo := &mockProjectRepoForCostHandler{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	h := setupCostHandler(costRepo, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/runs", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostRuns(rec, req, projectID, GetProjectCostRunsParams{})

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result struct {
		Data []RunCostRow `json:"data"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	require.Len(t, result.Data, 1)
	require.NotNil(t, result.Data[0].TokensInput)
	require.NotNil(t, result.Data[0].TokensOutput)
	assert.Equal(t, int64(120000), *result.Data[0].TokensInput)
	assert.Equal(t, int64(30000), *result.Data[0].TokensOutput)
}

// --- GetProjectCostsByRole handler tests ---

func TestGetProjectCostsByRole_200WithResults(t *testing.T) {
	projectID := uuid.New()
	costRepo := &mockCostRepoForHandler{
		listByProjectByRoleFn: func(_ context.Context, _ uuid.UUID) ([]model.ProjectRoleCostBreakdown, error) {
			return []model.ProjectRoleCostBreakdown{
				{Role: "implement", TokensInput: 500000, TokensOutput: 100000, CostUSD: 12.50, RunsCount: 3},
				{Role: "review", TokensInput: 200000, TokensOutput: 50000, CostUSD: 5.25, RunsCount: 2},
			}, nil
		},
	}
	projectRepo := &mockProjectRepoForCostHandler{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	h := setupCostHandler(costRepo, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByRole(rec, req, projectID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var result ProjectCostByRole
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	require.Len(t, result.Roles, 2)
	assert.Equal(t, "implement", result.Roles[0].Role)
	assert.InDelta(t, 12.50, result.Roles[0].CostUsd, 0.001)
	assert.Equal(t, int32(3), result.Roles[0].RunsCount)
	// Roll-up totals are summed across roles.
	assert.InDelta(t, 17.75, result.TotalCost, 0.001)
	assert.Equal(t, int64(700000), result.TotalTokensInput)
	assert.Equal(t, int64(150000), result.TotalTokensOutput)
}

func TestGetProjectCostsByRole_200EmptyResults(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCostHandler{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	h := setupCostHandler(&mockCostRepoForHandler{}, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByRole(rec, req, projectID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result ProjectCostByRole
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Empty(t, result.Roles)
	assert.Equal(t, float64(0), result.TotalCost)
}

func TestGetProjectCostsByRole_404ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCostHandler{
		err: errors.NewNotFound("project", uuid.New()),
	}
	h := setupCostHandler(&mockCostRepoForHandler{}, projectRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+uuid.New().String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetProjectCostsByRole(rec, req, uuid.New())

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- GetRunCostsByRole handler tests ---

func TestGetRunCostsByRole_200WithResults(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	costRepo := &mockCostRepoForHandler{
		sumCostByRunFn: func(_ context.Context, _ uuid.UUID) (float64, error) {
			return 6.30, nil
		},
		sumTokensByRunFn: func(_ context.Context, _ uuid.UUID) (int64, int64, error) {
			return 300000, 50000, nil
		},
		listCostsByRunByRoleFn: func(_ context.Context, _ uuid.UUID) ([]model.RoleCostBreakdown, error) {
			return []model.RoleCostBreakdown{
				{Role: "implement", TokensInput: 200000, TokensOutput: 30000, CostUSD: 4.50},
				{Role: "review", TokensInput: 100000, TokensOutput: 20000, CostUSD: 1.80},
			}, nil
		},
	}
	runRepo := newEpicRunRepo()
	runRepo.run = &model.Run{ID: runID, ProjectID: projectID}
	h := setupCostHandlerWithRunRepo(costRepo, runRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetRunCostsByRole(rec, req, projectID, runID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var result RunCostByRole
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, runID, result.RunId)
	assert.InDelta(t, 6.30, result.TotalCost, 0.001)
	assert.Equal(t, int64(300000), result.TotalTokensInput)
	assert.Equal(t, int64(50000), result.TotalTokensOutput)
	require.Len(t, result.Roles, 2)
	assert.Equal(t, "implement", result.Roles[0].Role)
	assert.InDelta(t, 4.50, result.Roles[0].CostUsd, 0.001)
}

func TestGetRunCostsByRole_404RunNotFound(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	runRepo := newEpicRunRepo() // run not configured -> GetRun returns NotFound
	h := setupCostHandlerWithRunRepo(&mockCostRepoForHandler{}, runRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetRunCostsByRole(rec, req, projectID, runID)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetRunCostsByRole_200EmptyRoles(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	runRepo := newEpicRunRepo()
	runRepo.run = &model.Run{ID: runID, ProjectID: projectID}
	h := setupCostHandlerWithRunRepo(&mockCostRepoForHandler{}, runRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/runs/"+runID.String()+"/costs/by-role", nil)
	rec := httptest.NewRecorder()

	h.GetRunCostsByRole(rec, req, projectID, runID)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result RunCostByRole
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	assert.Equal(t, runID, result.RunId)
	assert.Empty(t, result.Roles)
}
