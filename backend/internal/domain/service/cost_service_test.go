package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// --- Mock implementations ---

type mockCostRepo struct {
	insertCostRecordFn                 func(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error)
	getCostByRunStepFn                 func(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error)
	sumCostByProjectFn                 func(ctx context.Context, projectID uuid.UUID, since time.Time) (float64, int64, int64, error)
	sumCostByRunFn                     func(ctx context.Context, runID uuid.UUID) (float64, error)
	sumCostByStoryFn                   func(ctx context.Context, storyID uuid.UUID) (float64, int64, int64, int, error)
	listCostsByProjectByStoryFn        func(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.StoryCostBreakdown, error)
	listCostsByProjectByRunFn          func(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.RunCostBreakdown, error)
	listCostsByProjectByModelFn        func(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostByModel, error)
	listStepCostsByRunFn               func(ctx context.Context, runID uuid.UUID) ([]model.StepCostBreakdown, error)
	listDailyCostsByProjectFn          func(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostDataPoint, error)
	listCostsByProjectByRunPaginatedFn func(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error)
	countCostsByProjectByRunFn         func(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error)
	listByProjectByAgentFn             func(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error)

	insertCalls []model.CostRecord
}

func (m *mockCostRepo) InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error) {
	m.insertCalls = append(m.insertCalls, *record)
	if m.insertCostRecordFn != nil {
		return m.insertCostRecordFn(ctx, record)
	}
	record.ID = uuid.New()
	return record, nil
}

func (m *mockCostRepo) GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error) {
	if m.getCostByRunStepFn != nil {
		return m.getCostByRunStepFn(ctx, runStepID)
	}
	return nil, errors.NewNotFound("cost_record", runStepID)
}

func (m *mockCostRepo) SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (float64, int64, int64, error) {
	if m.sumCostByProjectFn != nil {
		return m.sumCostByProjectFn(ctx, projectID, since)
	}
	return 0, 0, 0, nil
}

func (m *mockCostRepo) SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error) {
	if m.sumCostByRunFn != nil {
		return m.sumCostByRunFn(ctx, runID)
	}
	return 0, nil
}

func (m *mockCostRepo) SumCostByStory(ctx context.Context, storyID uuid.UUID) (float64, int64, int64, int, error) {
	if m.sumCostByStoryFn != nil {
		return m.sumCostByStoryFn(ctx, storyID)
	}
	return 0, 0, 0, 0, nil
}

func (m *mockCostRepo) ListCostsByProjectByStory(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.StoryCostBreakdown, error) {
	if m.listCostsByProjectByStoryFn != nil {
		return m.listCostsByProjectByStoryFn(ctx, projectID, since)
	}
	return []model.StoryCostBreakdown{}, nil
}

func (m *mockCostRepo) ListCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.RunCostBreakdown, error) {
	if m.listCostsByProjectByRunFn != nil {
		return m.listCostsByProjectByRunFn(ctx, projectID, since)
	}
	return []model.RunCostBreakdown{}, nil
}

func (m *mockCostRepo) ListCostsByProjectByModel(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostByModel, error) {
	if m.listCostsByProjectByModelFn != nil {
		return m.listCostsByProjectByModelFn(ctx, projectID, since)
	}
	return []model.CostByModel{}, nil
}

func (m *mockCostRepo) ListStepCostsByRun(ctx context.Context, runID uuid.UUID) ([]model.StepCostBreakdown, error) {
	if m.listStepCostsByRunFn != nil {
		return m.listStepCostsByRunFn(ctx, runID)
	}
	return []model.StepCostBreakdown{}, nil
}

func (m *mockCostRepo) ListDailyCostsByProject(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostDataPoint, error) {
	if m.listDailyCostsByProjectFn != nil {
		return m.listDailyCostsByProjectFn(ctx, projectID, since)
	}
	return []model.CostDataPoint{}, nil
}

func (m *mockCostRepo) ListCostsByProjectByRunPaginated(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error) {
	if m.listCostsByProjectByRunPaginatedFn != nil {
		return m.listCostsByProjectByRunPaginatedFn(ctx, projectID, since, limit, offset)
	}
	return []model.RunCostRow{}, nil
}

func (m *mockCostRepo) CountCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error) {
	if m.countCostsByProjectByRunFn != nil {
		return m.countCostsByProjectByRunFn(ctx, projectID, since)
	}
	return 0, nil
}

func (m *mockCostRepo) ListByProjectByAgent(ctx context.Context, projectID uuid.UUID) ([]model.AgentCostBreakdown, error) {
	if m.listByProjectByAgentFn != nil {
		return m.listByProjectByAgentFn(ctx, projectID)
	}
	return []model.AgentCostBreakdown{}, nil
}

type mockProjectRepoForCost struct {
	project *model.Project
	err     error
}

func (m *mockProjectRepoForCost) Create(_ context.Context, _ *model.Project) (*model.Project, error) {
	return m.project, m.err
}
func (m *mockProjectRepoForCost) GetByID(_ context.Context, id uuid.UUID) (*model.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.project != nil && m.project.ID == id {
		return m.project, nil
	}
	return nil, errors.NewNotFound("project", id)
}
func (m *mockProjectRepoForCost) List(_ context.Context, _, _ int32) ([]*model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepoForCost) Count(_ context.Context) (int64, error) { return 0, nil }
func (m *mockProjectRepoForCost) Update(_ context.Context, p *model.Project) (*model.Project, error) {
	return p, nil
}
func (m *mockProjectRepoForCost) Delete(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockProjectRepoForCost) IncrementCircuitBreakerCount(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}
func (m *mockProjectRepoForCost) ResetCircuitBreaker(_ context.Context, _ uuid.UUID) (*model.Project, error) {
	return nil, nil
}

type mockStoryRepoForCost struct {
	story *model.Story
	err   error
}

func (m *mockStoryRepoForCost) Create(_ context.Context, _ *model.Story) (*model.Story, error) {
	return m.story, m.err
}
func (m *mockStoryRepoForCost) GetByID(_ context.Context, id uuid.UUID) (*model.Story, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.story != nil && m.story.ID == id {
		return m.story, nil
	}
	return nil, errors.NewNotFound("story", id)
}
func (m *mockStoryRepoForCost) GetByKey(_ context.Context, _ uuid.UUID, _ string) (*model.Story, error) {
	return m.story, m.err
}
func (m *mockStoryRepoForCost) ListByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForCost) ListByStatus(_ context.Context, _ uuid.UUID, _ []string, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForCost) ListByEpic(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Story, error) {
	return nil, nil
}
func (m *mockStoryRepoForCost) CountByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForCost) CountByStatus(_ context.Context, _ uuid.UUID, _ []string) (int64, error) {
	return 0, nil
}
func (m *mockStoryRepoForCost) Update(_ context.Context, s *model.Story) (*model.Story, error) {
	return s, nil
}
func (m *mockStoryRepoForCost) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type mockRunRepoForCost struct {
	run *model.Run
	err error
}

func (m *mockRunRepoForCost) CreateRun(_ context.Context, r *model.Run) (*model.Run, error) {
	return r, nil
}
func (m *mockRunRepoForCost) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.run != nil && m.run.ID == id {
		return m.run, nil
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *mockRunRepoForCost) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) UpdateRunStatus(_ context.Context, _ uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForCost) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForCost) CreateRunStep(_ context.Context, s *model.RunStep) (*model.RunStep, error) {
	return s, nil
}
func (m *mockRunRepoForCost) GetRunStep(_ context.Context, _ uuid.UUID) (*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) UpdateRunStepStatus(_ context.Context, _ uuid.UUID, _ model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForCost) CreateRetryRunStep(_ context.Context, s *model.RunStep) (*model.RunStep, error) {
	return s, nil
}
func (m *mockRunRepoForCost) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

func newTestCostService(costRepo *mockCostRepo, projectRepo *mockProjectRepoForCost, storyRepo *mockStoryRepoForCost, runRepo *mockRunRepoForCost) *CostService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return NewCostService(costRepo, projectRepo, storyRepo, runRepo, logger)
}

// --- RecordStepCost tests ---

func TestRecordStepCost_EmptyEvents_NoOp(t *testing.T) {
	costRepo := &mockCostRepo{}
	svc := newTestCostService(costRepo, nil, nil, nil)

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), []model.CostEvent{}, nil)
	assert.NoError(t, err)
	assert.Empty(t, costRepo.insertCalls)
}

func TestRecordStepCost_KnownModel_CorrectCost(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		inputTokens  int64
		outputTokens int64
		expectedCost float64
	}{
		{
			name:         "opus pricing",
			model:        "claude-opus-4-6",
			inputTokens:  1_000_000,
			outputTokens: 100_000,
			expectedCost: 15.0 + 7.5, // 15*1 + 75*0.1
		},
		{
			name:         "sonnet pricing",
			model:        "claude-sonnet-4-6",
			inputTokens:  2_000_000,
			outputTokens: 500_000,
			expectedCost: 6.0 + 7.5, // 3*2 + 15*0.5
		},
		{
			name:         "haiku pricing",
			model:        "claude-haiku-4-5",
			inputTokens:  10_000_000,
			outputTokens: 1_000_000,
			expectedCost: 2.5 + 1.25, // 0.25*10 + 1.25*1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			costRepo := &mockCostRepo{}
			svc := newTestCostService(costRepo, nil, nil, nil)

			stepID := uuid.New()
			projectID := uuid.New()
			events := []model.CostEvent{
				{InputTokens: tt.inputTokens, OutputTokens: tt.outputTokens, Model: tt.model},
			}

			err := svc.RecordStepCost(context.Background(), stepID, projectID, events, nil)
			require.NoError(t, err)
			require.Len(t, costRepo.insertCalls, 1)

			inserted := costRepo.insertCalls[0]
			assert.Equal(t, stepID, inserted.RunStepID)
			assert.Equal(t, projectID, inserted.ProjectID)
			assert.Equal(t, tt.inputTokens, inserted.TokensInput)
			assert.Equal(t, tt.outputTokens, inserted.TokensOutput)
			assert.InDelta(t, tt.expectedCost, inserted.CostUSD, 0.001)
			assert.Equal(t, tt.model, inserted.Model)
		})
	}
}

func TestRecordStepCost_UnknownModel_ZeroCost(t *testing.T) {
	costRepo := &mockCostRepo{}
	svc := newTestCostService(costRepo, nil, nil, nil)

	events := []model.CostEvent{
		{InputTokens: 1000, OutputTokens: 500, Model: "unknown-model"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events, nil)
	require.NoError(t, err)
	require.Len(t, costRepo.insertCalls, 1)

	inserted := costRepo.insertCalls[0]
	assert.Equal(t, float64(0), inserted.CostUSD)
	assert.Equal(t, "unknown-model", inserted.Model)
	assert.Nil(t, inserted.AgentID)
}

func TestRecordStepCost_MultipleEvents_Aggregated(t *testing.T) {
	costRepo := &mockCostRepo{}
	svc := newTestCostService(costRepo, nil, nil, nil)

	events := []model.CostEvent{
		{InputTokens: 500_000, OutputTokens: 100_000, Model: "claude-opus-4-6"},
		{InputTokens: 500_000, OutputTokens: 100_000, Model: "claude-opus-4-6"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events, nil)
	require.NoError(t, err)
	require.Len(t, costRepo.insertCalls, 1) // Single insert, not two

	inserted := costRepo.insertCalls[0]
	assert.Equal(t, int64(1_000_000), inserted.TokensInput)
	assert.Equal(t, int64(200_000), inserted.TokensOutput)
	// 15*1 + 75*0.2 = 15 + 15 = 30
	assert.InDelta(t, 30.0, inserted.CostUSD, 0.001)
}

func TestRecordStepCost_RepoError_Propagated(t *testing.T) {
	costRepo := &mockCostRepo{
		insertCostRecordFn: func(_ context.Context, _ *model.CostRecord) (*model.CostRecord, error) {
			return nil, errors.NewInternal("db error", nil)
		},
	}
	svc := newTestCostService(costRepo, nil, nil, nil)

	events := []model.CostEvent{
		{InputTokens: 1000, OutputTokens: 500, Model: "claude-opus-4-6"},
	}

	err := svc.RecordStepCost(context.Background(), uuid.New(), uuid.New(), events, nil)
	assert.Error(t, err)
}

func TestRecordStepCost_WithAgentID(t *testing.T) {
	costRepo := &mockCostRepo{}
	svc := newTestCostService(costRepo, nil, nil, nil)

	stepID := uuid.New()
	projectID := uuid.New()
	agentID := uuid.New()
	events := []model.CostEvent{
		{InputTokens: 1_000_000, OutputTokens: 100_000, Model: "claude-opus-4-6"},
	}

	err := svc.RecordStepCost(context.Background(), stepID, projectID, events, &agentID)
	require.NoError(t, err)
	require.Len(t, costRepo.insertCalls, 1)

	inserted := costRepo.insertCalls[0]
	assert.Equal(t, stepID, inserted.RunStepID)
	assert.Equal(t, projectID, inserted.ProjectID)
	require.NotNil(t, inserted.AgentID)
	assert.Equal(t, agentID, *inserted.AgentID)
	assert.Equal(t, int64(1_000_000), inserted.TokensInput)
	assert.Equal(t, int64(100_000), inserted.TokensOutput)
	// 15*1 + 75*0.1 = 22.5
	assert.InDelta(t, 22.5, inserted.CostUSD, 0.001)
}

func TestRecordStepCost_WithoutAgentID(t *testing.T) {
	costRepo := &mockCostRepo{}
	svc := newTestCostService(costRepo, nil, nil, nil)

	stepID := uuid.New()
	projectID := uuid.New()
	events := []model.CostEvent{
		{InputTokens: 1_000_000, OutputTokens: 100_000, Model: "claude-opus-4-6"},
	}

	err := svc.RecordStepCost(context.Background(), stepID, projectID, events, nil)
	require.NoError(t, err)
	require.Len(t, costRepo.insertCalls, 1)

	inserted := costRepo.insertCalls[0]
	assert.Equal(t, stepID, inserted.RunStepID)
	assert.Equal(t, projectID, inserted.ProjectID)
	assert.Nil(t, inserted.AgentID)
	assert.InDelta(t, 22.5, inserted.CostUSD, 0.001)
}

// --- GetProjectCosts tests ---

func TestGetProjectCosts_ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCost{err: errors.NewNotFound("project", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	_, err := svc.GetProjectCosts(context.Background(), uuid.New(), "7d")
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryNotFound, domErr.Category)
}

func TestGetProjectCosts_InvalidPeriod(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	_, err := svc.GetProjectCosts(context.Background(), projectID, "invalid")
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryValidation, domErr.Category)
}

func TestGetProjectCosts_ZeroCosts_EmptyBreakdowns(t *testing.T) {
	projectID := uuid.New()
	budget := 100.0
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test", MaxBudget: &budget},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	result, err := svc.GetProjectCosts(context.Background(), projectID, "7d")
	require.NoError(t, err)
	assert.Equal(t, float64(0), result.TotalCost)
	assert.Equal(t, int64(0), result.TotalInput)
	assert.Equal(t, int64(0), result.TotalOutput)
	assert.NotNil(t, result.MaxBudget)
	assert.Equal(t, 100.0, *result.MaxBudget)
	assert.Empty(t, result.ByStory)
	assert.Empty(t, result.ByRun)
	assert.Empty(t, result.ByModel)
}

func TestGetProjectCosts_DefaultPeriod(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	result, err := svc.GetProjectCosts(context.Background(), projectID, "")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// --- GetStoryCosts tests ---

func TestGetStoryCosts_StoryNotFound(t *testing.T) {
	storyRepo := &mockStoryRepoForCost{err: errors.NewNotFound("story", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, nil, storyRepo, nil)

	_, err := svc.GetStoryCosts(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetStoryCosts_WrongProject(t *testing.T) {
	storyID := uuid.New()
	storyRepo := &mockStoryRepoForCost{
		story: &model.Story{ID: storyID, ProjectID: uuid.New()},
	}
	svc := newTestCostService(&mockCostRepo{}, nil, storyRepo, nil)

	_, err := svc.GetStoryCosts(context.Background(), uuid.New(), storyID) // different projectID
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryNotFound, domErr.Category)
}

func TestGetStoryCosts_ZeroCosts(t *testing.T) {
	projectID := uuid.New()
	storyID := uuid.New()
	storyRepo := &mockStoryRepoForCost{
		story: &model.Story{ID: storyID, ProjectID: projectID},
	}
	svc := newTestCostService(&mockCostRepo{}, nil, storyRepo, nil)

	result, err := svc.GetStoryCosts(context.Background(), projectID, storyID)
	require.NoError(t, err)
	assert.Equal(t, storyID, result.StoryID)
	assert.Equal(t, float64(0), result.TotalCost)
	assert.Equal(t, int64(0), result.TotalInput)
	assert.Equal(t, int64(0), result.TotalOutput)
	assert.Equal(t, 0, result.RunCount)
}

// --- GetRunCosts tests ---

func TestGetRunCosts_RunNotFound(t *testing.T) {
	runRepo := &mockRunRepoForCost{err: errors.NewNotFound("run", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, nil, nil, runRepo)

	_, err := svc.GetRunCosts(context.Background(), uuid.New(), uuid.New())
	assert.Error(t, err)
}

func TestGetRunCosts_WrongProject(t *testing.T) {
	runID := uuid.New()
	runRepo := &mockRunRepoForCost{
		run: &model.Run{ID: runID, ProjectID: uuid.New()},
	}
	svc := newTestCostService(&mockCostRepo{}, nil, nil, runRepo)

	_, err := svc.GetRunCosts(context.Background(), uuid.New(), runID) // different projectID
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryNotFound, domErr.Category)
}

func TestGetRunCosts_ZeroCosts(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	runRepo := &mockRunRepoForCost{
		run: &model.Run{ID: runID, ProjectID: projectID},
	}
	svc := newTestCostService(&mockCostRepo{}, nil, nil, runRepo)

	result, err := svc.GetRunCosts(context.Background(), projectID, runID)
	require.NoError(t, err)
	assert.Equal(t, runID, result.RunID)
	assert.Equal(t, float64(0), result.TotalCost)
	assert.Empty(t, result.Steps)
}

// --- ComputeCostUSD tests ---

func TestComputeCostUSD(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		inputTokens   int64
		outputTokens  int64
		expectedCost  float64
		expectedKnown bool
	}{
		{"opus", "claude-opus-4-6", 1_000_000, 1_000_000, 15.0 + 75.0, true},
		{"sonnet", "claude-sonnet-4-6", 1_000_000, 1_000_000, 3.0 + 15.0, true},
		{"haiku", "claude-haiku-4-5", 1_000_000, 1_000_000, 0.25 + 1.25, true},
		{"unknown", "gpt-4", 1_000_000, 1_000_000, 0, false},
		{"zero tokens", "claude-opus-4-6", 0, 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, known := model.ComputeCostUSD(tt.model, tt.inputTokens, tt.outputTokens)
			assert.Equal(t, tt.expectedKnown, known)
			assert.InDelta(t, tt.expectedCost, cost, 0.001)
		})
	}
}

// --- GetProjectCostChart tests ---

func TestGetProjectCostChart_Success(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	costRepo := &mockCostRepo{
		listDailyCostsByProjectFn: func(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostDataPoint, error) {
			return []model.CostDataPoint{
				{Date: "2026-02-20", TotalCostUSD: 1.50},
				{Date: "2026-02-21", TotalCostUSD: 3.25},
			}, nil
		},
	}
	svc := newTestCostService(costRepo, projectRepo, nil, nil)

	points, err := svc.GetProjectCostChart(context.Background(), projectID, "7d")
	require.NoError(t, err)
	require.Len(t, points, 2)
	assert.Equal(t, "2026-02-20", points[0].Date)
	assert.InDelta(t, 1.50, points[0].TotalCostUSD, 0.001)
	assert.Equal(t, "2026-02-21", points[1].Date)
	assert.InDelta(t, 3.25, points[1].TotalCostUSD, 0.001)
}

func TestGetProjectCostChart_ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCost{err: errors.NewNotFound("project", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	_, err := svc.GetProjectCostChart(context.Background(), uuid.New(), "7d")
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryNotFound, domErr.Category)
}

func TestGetProjectCostChart_DefaultPeriod(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	points, err := svc.GetProjectCostChart(context.Background(), projectID, "")
	require.NoError(t, err)
	assert.Empty(t, points)
}

// --- GetProjectCostRuns tests ---

func TestGetProjectCostRuns_Success(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	runID := uuid.New()
	costRepo := &mockCostRepo{
		listCostsByProjectByRunPaginatedFn: func(_ context.Context, _ uuid.UUID, _ time.Time, _, _ int32) ([]model.RunCostRow, error) {
			return []model.RunCostRow{
				{RunID: runID, StoryKey: "S-01", Status: "completed", StartedAt: time.Now(), TotalCostUSD: 5.0},
			}, nil
		},
		countCostsByProjectByRunFn: func(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
			return 1, nil
		},
	}
	svc := newTestCostService(costRepo, projectRepo, nil, nil)

	rows, total, err := svc.GetProjectCostRuns(context.Background(), projectID, "7d", 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Equal(t, runID, rows[0].RunID)
	assert.Equal(t, "S-01", rows[0].StoryKey)
	assert.InDelta(t, 5.0, rows[0].TotalCostUSD, 0.001)
}

func TestGetProjectCostRuns_ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCost{err: errors.NewNotFound("project", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	_, _, err := svc.GetProjectCostRuns(context.Background(), uuid.New(), "7d", 1, 20)
	assert.Error(t, err)
}

func TestGetProjectCostRuns_PaginationDefaults(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	rows, total, err := svc.GetProjectCostRuns(context.Background(), projectID, "", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, rows)
}

// --- GetProjectCostsByAgent tests ---

func TestGetProjectCostsByAgent_Success(t *testing.T) {
	projectID := uuid.New()
	agentID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	costRepo := &mockCostRepo{
		listByProjectByAgentFn: func(_ context.Context, _ uuid.UUID) ([]model.AgentCostBreakdown, error) {
			return []model.AgentCostBreakdown{
				{AgentID: agentID, AgentName: "dev-story", TokensInput: 500000, TokensOutput: 100000, CostUSD: 12.50, RunsCount: 3},
			}, nil
		},
	}
	svc := newTestCostService(costRepo, projectRepo, nil, nil)

	result, err := svc.GetProjectCostsByAgent(context.Background(), projectID)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, agentID, result[0].AgentID)
	assert.Equal(t, "dev-story", result[0].AgentName)
	assert.Equal(t, int64(500000), result[0].TokensInput)
	assert.Equal(t, int64(100000), result[0].TokensOutput)
	assert.InDelta(t, 12.50, result[0].CostUSD, 0.001)
	assert.Equal(t, int32(3), result[0].RunsCount)
}

func TestGetProjectCostsByAgent_EmptyResults(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	result, err := svc.GetProjectCostsByAgent(context.Background(), projectID)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGetProjectCostsByAgent_ProjectNotFound(t *testing.T) {
	projectRepo := &mockProjectRepoForCost{err: errors.NewNotFound("project", uuid.New())}
	svc := newTestCostService(&mockCostRepo{}, projectRepo, nil, nil)

	_, err := svc.GetProjectCostsByAgent(context.Background(), uuid.New())
	assert.Error(t, err)
	domErr, ok := err.(*errors.DomainError)
	assert.True(t, ok)
	assert.Equal(t, errors.CategoryNotFound, domErr.Category)
}

func TestGetProjectCostsByAgent_RepoError(t *testing.T) {
	projectID := uuid.New()
	projectRepo := &mockProjectRepoForCost{
		project: &model.Project{ID: projectID, Name: "test"},
	}
	costRepo := &mockCostRepo{
		listByProjectByAgentFn: func(_ context.Context, _ uuid.UUID) ([]model.AgentCostBreakdown, error) {
			return nil, errors.NewInternal("db error", nil)
		},
	}
	svc := newTestCostService(costRepo, projectRepo, nil, nil)

	_, err := svc.GetProjectCostsByAgent(context.Background(), projectID)
	assert.Error(t, err)
}

// --- parsePeriod tests ---

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		period  string
		wantErr bool
		daysAgo int
	}{
		{"7d", false, 7},
		{"30d", false, 30},
		{"90d", false, 90},
		{"invalid", true, 0},
		{"1d", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			result, err := parsePeriod(tt.period)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			expected := time.Now().UTC().AddDate(0, 0, -tt.daysAgo)
			// Allow 2 seconds of tolerance
			assert.WithinDuration(t, expected, result, 2*time.Second)
		})
	}
}

func (m *mockStoryRepoForCost) CountByEpicGroupedByStatus(_ context.Context, _ uuid.UUID) (model.StoryCounts, error) {
	return model.StoryCounts{}, nil
}

func (m *mockRunRepoForCost) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *mockRunRepoForCost) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}
