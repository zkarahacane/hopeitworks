package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/api/middleware"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/service"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockHITLRepoForHandler implements port.HITLRepository for handler tests.
type mockHITLRepoForHandler struct {
	requests map[uuid.UUID]*model.HITLRequest
	pending  []*model.PendingHITLRequest
}

func newMockHITLRepoForHandler() *mockHITLRepoForHandler {
	return &mockHITLRepoForHandler{
		requests: make(map[uuid.UUID]*model.HITLRequest),
	}
}

func (m *mockHITLRepoForHandler) Create(_ context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	m.requests[req.ID] = req
	return req, nil
}

func (m *mockHITLRepoForHandler) GetByID(_ context.Context, id uuid.UUID) (*model.HITLRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, apperrors.NewNotFound("hitl_request", id)
	}
	return req, nil
}

func (m *mockHITLRepoForHandler) GetByRunStepID(_ context.Context, runStepID uuid.UUID) (*model.HITLRequest, error) {
	for _, req := range m.requests {
		if req.RunStepID == runStepID {
			return req, nil
		}
	}
	return nil, apperrors.NewNotFound("hitl_request", runStepID)
}

func (m *mockHITLRepoForHandler) UpdateStatus(_ context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, reason *string, at time.Time) (*model.HITLRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return nil, apperrors.NewNotFound("hitl_request", id)
	}
	req.Status = status
	req.ResolvedBy = resolvedBy
	req.RejectionReason = reason
	req.ResolvedAt = &at
	return req, nil
}

func (m *mockHITLRepoForHandler) ListPendingByProject(_ context.Context, _ uuid.UUID) ([]*model.PendingHITLRequest, error) {
	return m.pending, nil
}

func (m *mockHITLRepoForHandler) CountPendingByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return int64(len(m.pending)), nil
}

func (m *mockHITLRepoForHandler) ListFiltered(_ context.Context, _ *string, _, _ int32) ([]*model.HITLRequest, error) {
	return nil, nil
}

func (m *mockHITLRepoForHandler) CountFiltered(_ context.Context, _ *string) (int64, error) {
	return 0, nil
}

// mockRunRepoForHITLHandler implements a minimal port.RunRepository for HITL handler tests.
type mockRunRepoForHITLHandler struct {
	steps map[uuid.UUID]*model.RunStep
	runs  map[uuid.UUID]*model.Run
}

func newMockRunRepoForHITLHandler() *mockRunRepoForHITLHandler {
	return &mockRunRepoForHITLHandler{
		steps: make(map[uuid.UUID]*model.RunStep),
		runs:  make(map[uuid.UUID]*model.Run),
	}
}

func (m *mockRunRepoForHITLHandler) CreateRun(_ context.Context, _ *model.Run) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	run, ok := m.runs[id]
	if !ok {
		return nil, apperrors.NewNotFound("run", id)
	}
	return run, nil
}
func (m *mockRunRepoForHITLHandler) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) UpdateRunStatus(_ context.Context, _ uuid.UUID, _ model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForHITLHandler) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForHITLHandler) CreateRunStep(_ context.Context, _ *model.RunStep) (*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	step, ok := m.steps[id]
	if !ok {
		return nil, apperrors.NewNotFound("run_step", id)
	}
	return step, nil
}
func (m *mockRunRepoForHITLHandler) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForHITLHandler) UpdateRunStepStatus(_ context.Context, id uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, _ *string) (*model.RunStep, error) {
	step, ok := m.steps[id]
	if !ok {
		return nil, apperrors.NewNotFound("run_step", id)
	}
	step.Status = status
	return step, nil
}
func (m *mockRunRepoForHITLHandler) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}

func (m *mockRunRepoForHITLHandler) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}

func (m *mockRunRepoForHITLHandler) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

type mockEventPubForHITLHandler struct{}

func (m *mockEventPubForHITLHandler) Publish(_ context.Context, _ model.Event) error { return nil }

func setupHITLHandler() (*HITLHandler, *mockHITLRepoForHandler, *mockRunRepoForHITLHandler) {
	hitlRepo := newMockHITLRepoForHandler()
	runRepo := newMockRunRepoForHITLHandler()
	eventPub := &mockEventPubForHITLHandler{}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := service.NewHITLService(hitlRepo, runRepo, nil, eventPub, logger)
	handler := NewHITLHandler(svc)
	return handler, hitlRepo, runRepo
}

func TestHITLHandler_ListPendingHITLRequests(t *testing.T) {
	h, hitlRepo, _ := setupHITLHandler()
	projectID := uuid.New()

	tests := []struct {
		name       string
		pending    []*model.PendingHITLRequest
		wantStatus int
		wantTotal  int
	}{
		{
			name:       "empty list returns 200",
			pending:    nil,
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
		{
			name: "returns pending items",
			pending: []*model.PendingHITLRequest{
				{ID: uuid.New(), RunID: uuid.New(), StepID: uuid.New(), StoryKey: "S-01", CreatedAt: time.Now()},
				{ID: uuid.New(), RunID: uuid.New(), StepID: uuid.New(), StoryKey: "S-02", CreatedAt: time.Now()},
			},
			wantStatus: http.StatusOK,
			wantTotal:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hitlRepo.pending = tt.pending

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/projects/%s/hitl/pending", projectID), nil)
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.ListPendingHITLRequests(rec, req, projectID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			var resp PendingHITLRequestList
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if resp.Total != tt.wantTotal {
				t.Errorf("expected total %d, got %d", tt.wantTotal, resp.Total)
			}
			if len(resp.Data) != tt.wantTotal {
				t.Errorf("expected %d items, got %d", tt.wantTotal, len(resp.Data))
			}
		})
	}
}

func TestHITLHandler_GetHITLRequest(t *testing.T) {
	h, hitlRepo, _ := setupHITLHandler()
	hitlID := uuid.New()

	tests := []struct {
		name       string
		id         uuid.UUID
		seed       bool
		wantStatus int
	}{
		{
			name:       "existing request returns 200",
			id:         hitlID,
			seed:       true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "non-existent request returns 404",
			id:         uuid.New(),
			seed:       false,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.seed {
				hitlRepo.requests[hitlID] = &model.HITLRequest{
					ID:        hitlID,
					RunStepID: uuid.New(),
					GateType:  "approval",
					Status:    model.HITLStatusPending,
					CreatedAt: time.Now(),
				}
			}

			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/hitl-requests/%s", tt.id), nil)
			ctx := middleware.SetUserContext(req.Context(), uuid.New(), model.RoleUser)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.GetHITLRequest(rec, req, tt.id)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.seed {
				var resp HITLRequest
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Id != hitlID {
					t.Errorf("expected id %s, got %s", hitlID, resp.Id)
				}
			}
		})
	}
}

func TestHITLHandler_ApproveHITLRequest(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	hitlID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		hitlStatus model.HITLStatus
		hasAuth    bool
		wantStatus int
	}{
		{
			name:       "approve pending request returns 200",
			hitlStatus: model.HITLStatusPending,
			hasAuth:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "approve without auth returns 401",
			hitlStatus: model.HITLStatusPending,
			hasAuth:    false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "approve already approved returns 400",
			hitlStatus: model.HITLStatusApproved,
			hasAuth:    true,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hitlRepo, runRepo := setupHITLHandler()

			hitlRepo.requests[hitlID] = &model.HITLRequest{
				ID:        hitlID,
				RunStepID: stepID,
				GateType:  "approval",
				Status:    tt.hitlStatus,
				CreatedAt: time.Now(),
			}
			runRepo.steps[stepID] = &model.RunStep{
				ID:     stepID,
				RunID:  runID,
				Status: model.StepStatusWaitingApproval,
			}
			runRepo.runs[runID] = &model.Run{
				ID:        runID,
				ProjectID: projectID,
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/hitl-requests/%s/approve", hitlID), nil)
			if tt.hasAuth {
				ctx := middleware.SetUserContext(req.Context(), userID, model.RoleUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.ApproveHITLRequest(rec, req, hitlID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp HITLRequest
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Status != HITLRequestStatusApproved {
					t.Errorf("expected status approved, got %s", resp.Status)
				}
			}
		})
	}
}

func TestHITLHandler_RejectHITLRequest(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	hitlID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name       string
		body       string
		hasAuth    bool
		wantStatus int
	}{
		{
			name:       "reject with reason returns 200",
			body:       `{"reason":"needs refactor"}`,
			hasAuth:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "reject without body returns 200",
			body:       "",
			hasAuth:    true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "reject without auth returns 401",
			body:       `{"reason":"test"}`,
			hasAuth:    false,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hitlRepo, runRepo := setupHITLHandler()

			hitlRepo.requests[hitlID] = &model.HITLRequest{
				ID:        hitlID,
				RunStepID: stepID,
				GateType:  "approval",
				Status:    model.HITLStatusPending,
				CreatedAt: time.Now(),
			}
			runRepo.steps[stepID] = &model.RunStep{
				ID:     stepID,
				RunID:  runID,
				Status: model.StepStatusWaitingApproval,
			}
			runRepo.runs[runID] = &model.Run{
				ID:        runID,
				ProjectID: projectID,
			}

			var reqBody *bytes.Buffer
			if tt.body != "" {
				reqBody = bytes.NewBufferString(tt.body)
			} else {
				reqBody = bytes.NewBuffer(nil)
			}

			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/hitl-requests/%s/reject", hitlID), reqBody)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			if tt.hasAuth {
				ctx := middleware.SetUserContext(req.Context(), userID, model.RoleUser)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.RejectHITLRequest(rec, req, hitlID)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.wantStatus, rec.Code, rec.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp HITLRequest
				if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Status != HITLRequestStatusRejected {
					t.Errorf("expected status rejected, got %s", resp.Status)
				}
			}
		})
	}
}

func (m *mockRunRepoForHITLHandler) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *mockRunRepoForHITLHandler) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *mockRunRepoForHITLHandler) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *mockRunRepoForHITLHandler) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
