package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// mockHITLRepo implements port.HITLRepository for testing.
type mockHITLRepo struct {
	requests  map[uuid.UUID]*model.HITLRequest
	pending   []*model.PendingHITLRequest
	createFn  func(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
	updateFn  func(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, reason *string, at time.Time) (*model.HITLRequest, error)
	getByIDFn func(ctx context.Context, id uuid.UUID) (*model.HITLRequest, error)
}

func newMockHITLRepo() *mockHITLRepo {
	return &mockHITLRepo{
		requests: make(map[uuid.UUID]*model.HITLRequest),
	}
}

func (m *mockHITLRepo) Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	m.requests[req.ID] = req
	return req, nil
}

func (m *mockHITLRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.HITLRequest, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	req, ok := m.requests[id]
	if !ok {
		return nil, apperrors.NewNotFound("hitl_request", id)
	}
	return req, nil
}

func (m *mockHITLRepo) GetByRunStepID(_ context.Context, runStepID uuid.UUID) (*model.HITLRequest, error) {
	for _, req := range m.requests {
		if req.RunStepID == runStepID {
			return req, nil
		}
	}
	return nil, apperrors.NewNotFound("hitl_request", runStepID)
}

func (m *mockHITLRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, reason *string, at time.Time) (*model.HITLRequest, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, status, resolvedBy, reason, at)
	}
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

func (m *mockHITLRepo) ListPendingByProject(_ context.Context, _ uuid.UUID) ([]*model.PendingHITLRequest, error) {
	return m.pending, nil
}

func (m *mockHITLRepo) CountPendingByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return int64(len(m.pending)), nil
}

func (m *mockHITLRepo) ListFiltered(_ context.Context, status *string, limit, offset int32) ([]*model.HITLRequest, error) {
	var result []*model.HITLRequest
	for _, req := range m.requests {
		if status == nil || string(req.Status) == *status {
			result = append(result, req)
		}
	}
	// Apply offset/limit
	if int(offset) >= len(result) {
		return []*model.HITLRequest{}, nil
	}
	end := int(offset + limit)
	if end > len(result) {
		end = len(result)
	}
	return result[offset:end], nil
}

func (m *mockHITLRepo) CountFiltered(_ context.Context, status *string) (int64, error) {
	var count int64
	for _, req := range m.requests {
		if status == nil || string(req.Status) == *status {
			count++
		}
	}
	return count, nil
}

// mockRunRepoForHITL implements a minimal port.RunRepository for HITL tests.
type mockRunRepoForHITL struct {
	steps  map[uuid.UUID]*model.RunStep
	runs   map[uuid.UUID]*model.Run
	stepFn func(ctx context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errMsg *string) (*model.RunStep, error)
}

func newMockRunRepoForHITL() *mockRunRepoForHITL {
	return &mockRunRepoForHITL{
		steps: make(map[uuid.UUID]*model.RunStep),
		runs:  make(map[uuid.UUID]*model.Run),
	}
}

func (m *mockRunRepoForHITL) CreateRun(_ context.Context, _ *model.Run) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) GetRun(_ context.Context, id uuid.UUID) (*model.Run, error) {
	run, ok := m.runs[id]
	if !ok {
		return nil, apperrors.NewNotFound("run", id)
	}
	return run, nil
}
func (m *mockRunRepoForHITL) GetActiveRunByStory(_ context.Context, _ uuid.UUID) (*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) ListRunsByProject(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) ListRunsByStory(_ context.Context, _ uuid.UUID, _, _ int32) ([]*model.Run, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) UpdateRunStatus(_ context.Context, id uuid.UUID, status model.RunStatus, _, _, _ *time.Time, _ *string) (*model.Run, error) {
	run, ok := m.runs[id]
	if !ok {
		return nil, apperrors.NewNotFound("run", id)
	}
	run.Status = status
	return run, nil
}
func (m *mockRunRepoForHITL) CountRunsByProject(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForHITL) CountRunsByStory(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockRunRepoForHITL) CreateRunStep(_ context.Context, _ *model.RunStep) (*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) GetRunStep(_ context.Context, id uuid.UUID) (*model.RunStep, error) {
	step, ok := m.steps[id]
	if !ok {
		return nil, apperrors.NewNotFound("run_step", id)
	}
	return step, nil
}
func (m *mockRunRepoForHITL) ListRunStepsByRun(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}
func (m *mockRunRepoForHITL) UpdateRunStepStatus(_ context.Context, id uuid.UUID, status model.StepStatus, startedAt, completedAt *time.Time, errMsg *string) (*model.RunStep, error) {
	if m.stepFn != nil {
		return m.stepFn(nil, id, status, startedAt, completedAt, errMsg)
	}
	step, ok := m.steps[id]
	if !ok {
		return nil, apperrors.NewNotFound("run_step", id)
	}
	step.Status = status
	return step, nil
}
func (m *mockRunRepoForHITL) UpdateRunStepContainerInfo(_ context.Context, _ uuid.UUID, _ *string, _ *string) (*model.RunStep, error) {
	return nil, nil
}

func (m *mockRunRepoForHITL) CreateRetryRunStep(_ context.Context, step *model.RunStep) (*model.RunStep, error) {
	return step, nil
}

func (m *mockRunRepoForHITL) ListRetryStepsByParent(_ context.Context, _ uuid.UUID) ([]*model.RunStep, error) {
	return nil, nil
}

// mockEventPubForHITL implements port.EventPublisher for HITL tests.
type mockEventPubForHITL struct {
	events []model.Event
}

func (m *mockEventPubForHITL) Publish(_ context.Context, event model.Event) error {
	m.events = append(m.events, event)
	return nil
}

// mockJobQueueForHITL implements port.JobQueue for HITL tests, recording
// the run IDs that were enqueued for execution.
type mockJobQueueForHITL struct {
	enqueued []uuid.UUID
}

func (m *mockJobQueueForHITL) EnqueueExecuteRun(_ context.Context, runID uuid.UUID) error {
	m.enqueued = append(m.enqueued, runID)
	return nil
}

func hitlTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHITLService_Approve(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	hitlID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name        string
		hitlStatus  model.HITLStatus
		wantErr     bool
		wantErrCode string
	}{
		{
			name:       "approve pending request succeeds",
			hitlStatus: model.HITLStatusPending,
			wantErr:    false,
		},
		{
			name:        "approve already approved request fails",
			hitlStatus:  model.HITLStatusApproved,
			wantErr:     true,
			wantErrCode: "VALIDATION_ERROR",
		},
		{
			name:        "approve already rejected request fails",
			hitlStatus:  model.HITLStatusRejected,
			wantErr:     true,
			wantErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hitlRepo := newMockHITLRepo()
			hitlRepo.requests[hitlID] = &model.HITLRequest{
				ID:        hitlID,
				RunStepID: stepID,
				GateType:  "approval",
				Status:    tt.hitlStatus,
				CreatedAt: time.Now(),
			}

			runRepo := newMockRunRepoForHITL()
			runRepo.steps[stepID] = &model.RunStep{
				ID:     stepID,
				RunID:  runID,
				Status: model.StepStatusWaitingApproval,
			}
			runRepo.runs[runID] = &model.Run{
				ID:        runID,
				ProjectID: projectID,
				Status:    model.RunStatusPaused,
			}

			eventPub := &mockEventPubForHITL{}
			jobQueue := &mockJobQueueForHITL{}
			svc := NewHITLService(hitlRepo, runRepo, jobQueue, eventPub, hitlTestLogger())

			result, err := svc.Approve(context.Background(), hitlID, userID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*apperrors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T: %v", err, err)
				}
				if domainErr.Code != tt.wantErrCode {
					t.Errorf("expected error code %q, got %q", tt.wantErrCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != model.HITLStatusApproved {
				t.Errorf("expected status approved, got %s", result.Status)
			}
			if result.ResolvedBy == nil || *result.ResolvedBy != userID {
				t.Error("expected resolved_by to be set to user ID")
			}
			if result.ResolvedAt == nil {
				t.Error("expected resolved_at to be set")
			}
			// Verify the gate step was marked completed so the executor
			// advances to the next step on resume (rather than re-suspending).
			step := runRepo.steps[stepID]
			if step.Status != model.StepStatusCompleted {
				t.Errorf("expected step status completed, got %s", step.Status)
			}
			// Verify the run was resumed to running.
			run := runRepo.runs[runID]
			if run.Status != model.RunStatusRunning {
				t.Errorf("expected run status running, got %s", run.Status)
			}
			// Verify the run was re-enqueued for execution exactly once.
			if len(jobQueue.enqueued) != 1 || jobQueue.enqueued[0] != runID {
				t.Errorf("expected run %s enqueued once, got %v", runID, jobQueue.enqueued)
			}
			// Verify event was published
			if len(eventPub.events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(eventPub.events))
			}
			if eventPub.events[0].Action != "approved" {
				t.Errorf("expected event action 'approved', got %q", eventPub.events[0].Action)
			}
		})
	}
}

func TestHITLService_Reject(t *testing.T) {
	projectID := uuid.New()
	runID := uuid.New()
	stepID := uuid.New()
	hitlID := uuid.New()
	userID := uuid.New()
	reason := "needs refactor"

	tests := []struct {
		name        string
		hitlStatus  model.HITLStatus
		reason      *string
		wantErr     bool
		wantErrCode string
	}{
		{
			name:       "reject pending request succeeds",
			hitlStatus: model.HITLStatusPending,
			reason:     &reason,
			wantErr:    false,
		},
		{
			name:       "reject with nil reason succeeds",
			hitlStatus: model.HITLStatusPending,
			reason:     nil,
			wantErr:    false,
		},
		{
			name:        "reject already approved request fails",
			hitlStatus:  model.HITLStatusApproved,
			reason:      &reason,
			wantErr:     true,
			wantErrCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hitlRepo := newMockHITLRepo()
			hitlRepo.requests[hitlID] = &model.HITLRequest{
				ID:        hitlID,
				RunStepID: stepID,
				GateType:  "approval",
				Status:    tt.hitlStatus,
				CreatedAt: time.Now(),
			}

			runRepo := newMockRunRepoForHITL()
			runRepo.steps[stepID] = &model.RunStep{
				ID:     stepID,
				RunID:  runID,
				Status: model.StepStatusWaitingApproval,
			}
			runRepo.runs[runID] = &model.Run{
				ID:        runID,
				ProjectID: projectID,
			}

			eventPub := &mockEventPubForHITL{}
			svc := NewHITLService(hitlRepo, runRepo, nil, eventPub, hitlTestLogger())

			result, err := svc.Reject(context.Background(), hitlID, userID, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				domainErr, ok := err.(*apperrors.DomainError)
				if !ok {
					t.Fatalf("expected DomainError, got %T: %v", err, err)
				}
				if domainErr.Code != tt.wantErrCode {
					t.Errorf("expected error code %q, got %q", tt.wantErrCode, domainErr.Code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != model.HITLStatusRejected {
				t.Errorf("expected status rejected, got %s", result.Status)
			}
			if result.ResolvedBy == nil || *result.ResolvedBy != userID {
				t.Error("expected resolved_by to be set to user ID")
			}
			if tt.reason != nil && (result.RejectionReason == nil || *result.RejectionReason != *tt.reason) {
				t.Error("expected rejection reason to be set")
			}
			// Verify step was transitioned to failed
			step := runRepo.steps[stepID]
			if step.Status != model.StepStatusFailed {
				t.Errorf("expected step status failed, got %s", step.Status)
			}
		})
	}
}

func TestHITLService_Approve_NotFound(t *testing.T) {
	hitlRepo := newMockHITLRepo()
	runRepo := newMockRunRepoForHITL()
	svc := NewHITLService(hitlRepo, runRepo, nil, nil, hitlTestLogger())

	_, err := svc.Approve(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*apperrors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T: %v", err, err)
	}
	if domainErr.Category != apperrors.CategoryNotFound {
		t.Errorf("expected not_found category, got %s", domainErr.Category)
	}
}

func TestHITLService_ListPendingByProject(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name      string
		pending   []*model.PendingHITLRequest
		wantCount int64
	}{
		{
			name:      "no pending requests",
			pending:   nil,
			wantCount: 0,
		},
		{
			name: "multiple pending requests",
			pending: []*model.PendingHITLRequest{
				{ID: uuid.New(), RunID: uuid.New(), StepID: uuid.New(), StoryKey: "S-01", CreatedAt: time.Now()},
				{ID: uuid.New(), RunID: uuid.New(), StepID: uuid.New(), StoryKey: "S-02", CreatedAt: time.Now()},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hitlRepo := newMockHITLRepo()
			hitlRepo.pending = tt.pending
			runRepo := newMockRunRepoForHITL()
			svc := NewHITLService(hitlRepo, runRepo, nil, nil, hitlTestLogger())

			pending, count, err := svc.ListPendingByProject(context.Background(), projectID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if count != tt.wantCount {
				t.Errorf("expected count %d, got %d", tt.wantCount, count)
			}
			if int64(len(pending)) != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, len(pending))
			}
		})
	}
}

func TestHITLService_Reject_RepoUpdateFails(t *testing.T) {
	hitlID := uuid.New()
	stepID := uuid.New()
	userID := uuid.New()

	hitlRepo := newMockHITLRepo()
	hitlRepo.requests[hitlID] = &model.HITLRequest{
		ID:        hitlID,
		RunStepID: stepID,
		GateType:  "approval",
		Status:    model.HITLStatusPending,
		CreatedAt: time.Now(),
	}
	hitlRepo.updateFn = func(_ context.Context, _ uuid.UUID, _ model.HITLStatus, _ *uuid.UUID, _ *string, _ time.Time) (*model.HITLRequest, error) {
		return nil, fmt.Errorf("db connection lost")
	}

	runRepo := newMockRunRepoForHITL()
	svc := NewHITLService(hitlRepo, runRepo, nil, nil, hitlTestLogger())

	_, err := svc.Reject(context.Background(), hitlID, userID, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHITLService_ListAll(t *testing.T) {
	hitlRepo := newMockHITLRepo()

	// Seed requests with different statuses
	for i := 0; i < 5; i++ {
		id := uuid.New()
		hitlRepo.requests[id] = &model.HITLRequest{
			ID:        id,
			RunStepID: uuid.New(),
			GateType:  "approval",
			Status:    model.HITLStatusPending,
			CreatedAt: time.Now(),
		}
	}
	approvedID := uuid.New()
	hitlRepo.requests[approvedID] = &model.HITLRequest{
		ID:        approvedID,
		RunStepID: uuid.New(),
		GateType:  "approval",
		Status:    model.HITLStatusApproved,
		CreatedAt: time.Now(),
	}

	svc := NewHITLService(hitlRepo, newMockRunRepoForHITL(), nil, nil, hitlTestLogger())

	t.Run("no filter returns all", func(t *testing.T) {
		items, total, err := svc.ListAll(context.Background(), nil, 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 6 {
			t.Errorf("expected total 6, got %d", total)
		}
		if len(items) != 6 {
			t.Errorf("expected 6 items, got %d", len(items))
		}
	})

	t.Run("filter by pending", func(t *testing.T) {
		status := "pending"
		items, total, err := svc.ListAll(context.Background(), &status, 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}
		if len(items) != 5 {
			t.Errorf("expected 5 items, got %d", len(items))
		}
	})

	t.Run("filter by approved", func(t *testing.T) {
		status := "approved"
		items, total, err := svc.ListAll(context.Background(), &status, 1, 20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 item, got %d", len(items))
		}
	})

	t.Run("pagination defaults", func(t *testing.T) {
		items, total, err := svc.ListAll(context.Background(), nil, 0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 6 {
			t.Errorf("expected total 6, got %d", total)
		}
		if len(items) > 20 {
			t.Errorf("expected at most 20 items (default), got %d", len(items))
		}
	})
}

func TestHITLService_GetByStepID(t *testing.T) {
	hitlRepo := newMockHITLRepo()
	stepID := uuid.New()
	hitlID := uuid.New()
	hitlRepo.requests[hitlID] = &model.HITLRequest{
		ID:        hitlID,
		RunStepID: stepID,
		GateType:  "approval",
		Status:    model.HITLStatusPending,
		CreatedAt: time.Now(),
	}

	svc := NewHITLService(hitlRepo, newMockRunRepoForHITL(), nil, nil, hitlTestLogger())

	t.Run("existing step returns request", func(t *testing.T) {
		req, err := svc.GetByStepID(context.Background(), stepID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req.ID != hitlID {
			t.Errorf("expected hitl ID %s, got %s", hitlID, req.ID)
		}
	})

	t.Run("non-existent step returns not found", func(t *testing.T) {
		_, err := svc.GetByStepID(context.Background(), uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		domainErr, ok := err.(*apperrors.DomainError)
		if !ok {
			t.Fatalf("expected DomainError, got %T", err)
		}
		if domainErr.Category != apperrors.CategoryNotFound {
			t.Errorf("expected not_found category, got %s", domainErr.Category)
		}
	})
}

func (m *mockRunRepoForHITL) GetLatestRunByStory(_ context.Context, _ uuid.UUID) (*model.LatestRun, error) {
	return nil, nil
}

func (m *mockRunRepoForHITL) GetLatestRunsByStories(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]*model.LatestRun, error) {
	return map[uuid.UUID]*model.LatestRun{}, nil
}

func (m *mockRunRepoForHITL) UpdateRunMetadata(_ context.Context, _ uuid.UUID, _ map[string]interface{}) error {
	return nil
}

func (m *mockRunRepoForHITL) AppendStepLogTail(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
