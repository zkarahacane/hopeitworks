package handler

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// testCallbackLogger returns a silent logger for the cost callback tests.
func testCallbackLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- Mocks scoped to the agent callback handler tests ---

// recordingCostRepoForCallback captures every InsertCostRecord so a test can
// assert what CostUSD the cost callback ultimately persisted.
type recordingCostRepoForCallback struct {
	inserts []model.CostRecord
}

func (m *recordingCostRepoForCallback) InsertCostRecord(_ context.Context, r *model.CostRecord) (*model.CostRecord, error) {
	m.inserts = append(m.inserts, *r)
	r.ID = uuid.New()
	return r, nil
}
func (m *recordingCostRepoForCallback) GetCostByRunStep(_ context.Context, _ uuid.UUID) (*model.CostRecord, error) {
	return nil, errors.NewNotFound("cost_record", uuid.Nil)
}
func (m *recordingCostRepoForCallback) SumCostByProject(_ context.Context, _ uuid.UUID, _ time.Time) (float64, int64, int64, error) {
	return 0, 0, 0, nil
}
func (m *recordingCostRepoForCallback) SumCostByRun(_ context.Context, _ uuid.UUID) (float64, error) {
	return 0, nil
}
func (m *recordingCostRepoForCallback) SumCostByStory(_ context.Context, _ uuid.UUID) (float64, int64, int64, int, error) {
	return 0, 0, 0, 0, nil
}
func (m *recordingCostRepoForCallback) ListCostsByProjectByStory(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.StoryCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListCostsByProjectByRun(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.RunCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListCostsByProjectByModel(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostByModel, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListStepCostsByRun(_ context.Context, _ uuid.UUID) ([]model.StepCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListDailyCostsByProject(_ context.Context, _ uuid.UUID, _ time.Time) ([]model.CostDataPoint, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListCostsByProjectByRunPaginated(_ context.Context, _ uuid.UUID, _ time.Time, _, _ int32) ([]model.RunCostRow, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) CountCostsByProjectByRun(_ context.Context, _ uuid.UUID, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *recordingCostRepoForCallback) ListByProjectByAgent(_ context.Context, _ uuid.UUID) ([]model.AgentCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListByProjectByRole(_ context.Context, _ uuid.UUID) ([]model.ProjectRoleCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) ListCostsByRunByRole(_ context.Context, _ uuid.UUID) ([]model.RoleCostBreakdown, error) {
	return nil, nil
}
func (m *recordingCostRepoForCallback) SumTokensByRun(_ context.Context, _ uuid.UUID) (int64, int64, error) {
	return 0, 0, nil
}

// runRepoForCallback is a RunRepository mock whose GetRunStep/GetRun resolve the
// fixed step+run the cost callback needs (the run carries the project id).
type runRepoForCallback struct {
	step *model.RunStep
	run  *model.Run
}

func (m *runRepoForCallback) CreateRun(_ context.Context, run *model.Run) (*model.Run, error) {
	return run, nil
}
func (m *runRepoForCallback) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	if m.run != nil && m.run.ID == id {
		return m.run, nil
	}
	return nil, errors.NewNotFound("run", id)
}
func (m *runRepoForCallback) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *runRepoForCallback) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}
func (m *runRepoForCallback) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}
func (m *runRepoForCallback) GetDAGNodeRunInfoByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]model.DAGNodeRunInfo, error) {
	return map[uuid.UUID]model.DAGNodeRunInfo{}, nil
}
func (m *runRepoForCallback) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *runRepoForCallback) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *runRepoForCallback) ListRunsByStatus(_ context.Context, _ model.RunStatus) ([]*model.Run, error) {
	return nil, nil
}
func (m *runRepoForCallback) MarkRunOrphanedIfRunning(_ context.Context, _ uuid.UUID, _ time.Time, _ string) (bool, error) {
	return false, nil
}
func (m *runRepoForCallback) UpdateRunStatus(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return &model.Run{ID: id, Status: status}, nil
}
func (m *runRepoForCallback) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}
func (m *runRepoForCallback) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *runRepoForCallback) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *runRepoForCallback) CreateRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *runRepoForCallback) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	if m.step != nil && m.step.ID == id {
		return m.step, nil
	}
	return nil, errors.NewNotFound("run_step", id)
}
func (m *runRepoForCallback) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *runRepoForCallback) UpdateRunStepStatus(_ context.Context, id uuid.UUID, status model.StepStatus, _, _ *time.Time, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id, Status: status}, nil
}
func (m *runRepoForCallback) UpdateRunStepContainerInfo(_ context.Context, id uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return &model.RunStep{ID: id}, nil
}
func (m *runRepoForCallback) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}
func (m *runRepoForCallback) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *runRepoForCallback) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// noopEventPublisherForCallback is a no-op EventPublisher for the cost callback,
// which does not publish events.
type noopEventPublisherForCallback struct{}

func (m *noopEventPublisherForCallback) Publish(_ context.Context, _ model.Event) error {
	return nil
}

// noopStatusStoreForCallback is a no-op CallbackStatusStore; the cost callback
// never touches it.
type noopStatusStoreForCallback struct{}

func (m *noopStatusStoreForCallback) WaitForStatus(_ context.Context, _ uuid.UUID, _ time.Duration) (int, string, error) {
	return 0, "", nil
}
func (m *noopStatusStoreForCallback) SetStatus(_ context.Context, _ uuid.UUID, _ int, _ string) error {
	return nil
}

// TestHandleCost_ProviderCostUSD_ReachesRecordStepCost proves the cost callback
// threads the provider-real cost_usd from the request body all the way into the
// persisted CostRecord — so the agent/provider cost is what gets stored, not a
// pricing-table derivation.
func TestHandleCost_ProviderCostUSD_ReachesRecordStepCost(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	costRepo := &recordingCostRepoForCallback{}
	costSvc := service.NewCostService(costRepo, nil, nil, nil, testCallbackLogger())

	runRepo := &runRepoForCallback{
		step: &model.RunStep{ID: stepID, RunID: runID},
		run:  &model.Run{ID: runID, ProjectID: projectID},
	}

	h := NewAgentCallbackHandler(
		&noopEventPublisherForCallback{},
		costSvc,
		&noopStatusStoreForCallback{},
		runRepo,
	)

	body := `{"input_tokens":1000,"output_tokens":500,"model":"claude-opus-4-6","cost_usd":0.05}`
	rec := callbackRequest(t, h.HandleCost, runID, stepID, body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, costRepo.inserts, 1, "expected one cost record to be persisted")

	inserted := costRepo.inserts[0]
	// The provider-real 0.05 must be persisted verbatim (not the ~22.5 pricing
	// derivation for these tokens on opus).
	assert.Equal(t, 0.05, inserted.CostUSD)
	assert.Equal(t, stepID, inserted.RunStepID)
	assert.Equal(t, projectID, inserted.ProjectID)
	assert.Equal(t, int64(1000), inserted.TokensInput)
	assert.Equal(t, int64(500), inserted.TokensOutput)
}

// TestHandleCost_NoCostUSD_FallsBackToPricing proves a cost callback that omits
// cost_usd (legacy agent) falls back to the pricing-table derivation.
func TestHandleCost_NoCostUSD_FallsBackToPricing(t *testing.T) {
	runID := uuid.New()
	stepID := uuid.New()
	projectID := uuid.New()

	costRepo := &recordingCostRepoForCallback{}
	costSvc := service.NewCostService(costRepo, nil, nil, nil, testCallbackLogger())

	runRepo := &runRepoForCallback{
		step: &model.RunStep{ID: stepID, RunID: runID},
		run:  &model.Run{ID: runID, ProjectID: projectID},
	}

	h := NewAgentCallbackHandler(
		&noopEventPublisherForCallback{},
		costSvc,
		&noopStatusStoreForCallback{},
		runRepo,
	)

	// No cost_usd field → 0 → pricing fallback. 1M in + 100k out on opus = 22.5.
	body := `{"input_tokens":1000000,"output_tokens":100000,"model":"claude-opus-4-6"}`
	rec := callbackRequest(t, h.HandleCost, runID, stepID, body)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, costRepo.inserts, 1)
	assert.InDelta(t, 22.5, costRepo.inserts[0].CostUSD, 0.001)
}

// callbackRequest issues a POST to the given handler with the runId/stepId chi
// URL params populated, returning the recorder.
func callbackRequest(t *testing.T, h http.HandlerFunc, runID, stepID uuid.UUID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/internal/agent/callback/runs/"+runID.String()+"/steps/"+stepID.String()+"/cost", bytes.NewBufferString(body))
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("runId", runID.String())
	rctx.URLParams.Add("stepId", stepID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}
