package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const codeRunNotFound = "RUN_NOT_FOUND"

// mockHITLRepo implements port.HITLRepository for testing.
type mockHITLRepo struct {
	createFn            func(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
	getByRunStepIDFn    func(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error)
	getPendingByRunIDFn func(ctx context.Context, runID uuid.UUID) (*model.HITLRequest, error)
	updateStatusFn      func(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error)
}

func (m *mockHITLRepo) Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return req, nil
}

func (m *mockHITLRepo) GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error) {
	if m.getByRunStepIDFn != nil {
		return m.getByRunStepIDFn(ctx, runStepID)
	}
	return nil, errors.NewNotFound("hitl_request", runStepID)
}

func (m *mockHITLRepo) GetPendingByRunID(ctx context.Context, runID uuid.UUID) (*model.HITLRequest, error) {
	if m.getPendingByRunIDFn != nil {
		return m.getPendingByRunIDFn(ctx, runID)
	}
	return nil, errors.NewNotFound("hitl_request", runID)
}

func (m *mockHITLRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, resolvedBy, rejectionReason, resolvedAt)
	}
	return &model.HITLRequest{ID: id, Status: status}, nil
}

// hitlTestFixture provides shared test setup for HITLService tests.
type hitlTestFixture struct {
	projectID   uuid.UUID
	runID       uuid.UUID
	reviewerID  uuid.UUID
	stepID      uuid.UUID
	hitlReqID   uuid.UUID
	run         *model.Run
	hitlRequest *model.HITLRequest

	hitlRepo *mockHITLRepo
	runRepo  *mockRunRepo
	eventPub *mockEventPublisher
	jobQueue *mockJobQueue
	service  *HITLService
}

func newHITLTestFixture() *hitlTestFixture {
	f := &hitlTestFixture{
		projectID:  uuid.New(),
		runID:      uuid.New(),
		reviewerID: uuid.New(),
		stepID:     uuid.New(),
		hitlReqID:  uuid.New(),
	}

	f.run = &model.Run{
		ID:        f.runID,
		ProjectID: f.projectID,
		StoryID:   uuid.New(),
		Status:    model.RunStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	f.hitlRequest = &model.HITLRequest{
		ID:        f.hitlReqID,
		RunStepID: f.stepID,
		GateType:  "approval",
		Status:    model.HITLStatusPending,
		CreatedAt: time.Now(),
	}

	f.hitlRepo = &mockHITLRepo{
		getPendingByRunIDFn: func(_ context.Context, runID uuid.UUID) (*model.HITLRequest, error) {
			if runID == f.runID {
				cp := *f.hitlRequest
				return &cp, nil
			}
			return nil, errors.NewNotFound("hitl_request", runID)
		},
		updateStatusFn: func(_ context.Context, _ uuid.UUID, status model.HITLStatus, _ *uuid.UUID, _ *string, _ time.Time) (*model.HITLRequest, error) {
			cp := *f.hitlRequest
			cp.Status = status
			return &cp, nil
		},
	}

	f.runRepo = &mockRunRepo{
		getRunFn: func(_ context.Context, id uuid.UUID) (*model.Run, error) {
			if id == f.runID {
				cp := *f.run
				return &cp, nil
			}
			return nil, errors.NewNotFound("run", id)
		},
		updateRunStepStatusFn: func(_ context.Context, id uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, _ *string) (*model.RunStep, error) {
			return &model.RunStep{ID: id, Status: status}, nil
		},
		updateRunStatusFn: func(_ context.Context, _ uuid.UUID, status model.RunStatus, _ *time.Time, _ *time.Time, _ *string) (*model.Run, error) {
			cp := *f.run
			cp.Status = status
			return &cp, nil
		},
	}

	f.eventPub = newMockEventPublisher()
	f.jobQueue = &mockJobQueue{}
	f.service = NewHITLService(f.hitlRepo, f.runRepo, f.eventPub, f.jobQueue, testLogger())

	return f
}

func TestHITLService_Approve_HappyPath(t *testing.T) {
	f := newHITLTestFixture()

	var updatedStatus model.HITLStatus
	var updatedReviewer *uuid.UUID
	f.hitlRepo.updateStatusFn = func(_ context.Context, _ uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, _ *string, _ time.Time) (*model.HITLRequest, error) {
		updatedStatus = status
		updatedReviewer = resolvedBy
		cp := *f.hitlRequest
		cp.Status = status
		return &cp, nil
	}

	var stepStatusUpdated model.StepStatus
	f.runRepo.updateRunStepStatusFn = func(_ context.Context, _ uuid.UUID, status model.StepStatus, _ *time.Time, _ *time.Time, _ *string) (*model.RunStep, error) {
		stepStatusUpdated = status
		return &model.RunStep{ID: f.stepID, Status: status}, nil
	}

	result, err := f.service.Approve(context.Background(), f.projectID, f.runID, f.reviewerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.RunID != f.runID {
		t.Errorf("expected run_id %s, got %s", f.runID, result.RunID)
	}
	if result.HITLRequestID != f.hitlReqID {
		t.Errorf("expected hitl_request_id %s, got %s", f.hitlReqID, result.HITLRequestID)
	}
	if result.Status != string(model.RunStatusRunning) {
		t.Errorf("expected status 'running', got %s", result.Status)
	}
	if updatedStatus != model.HITLStatusApproved {
		t.Errorf("expected HITL status approved, got %s", updatedStatus)
	}
	if updatedReviewer == nil || *updatedReviewer != f.reviewerID {
		t.Errorf("expected reviewer %s, got %v", f.reviewerID, updatedReviewer)
	}
	if stepStatusUpdated != model.StepStatusRunning {
		t.Errorf("expected step status running, got %s", stepStatusUpdated)
	}

	// Verify event published
	events := f.eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event.Action != "approved" {
		t.Errorf("expected event action 'approved', got %s", events[0].Event.Action)
	}
	if events[0].Event.EntityType != "hitl_gate" {
		t.Errorf("expected entity_type 'hitl_gate', got %s", events[0].Event.EntityType)
	}

	// Verify job enqueued
	if len(f.jobQueue.enqueuedResumes) != 1 {
		t.Fatalf("expected 1 resume job enqueued, got %d", len(f.jobQueue.enqueuedResumes))
	}
	if f.jobQueue.enqueuedResumes[0].RunID != f.runID {
		t.Errorf("expected enqueued run_id %s, got %s", f.runID, f.jobQueue.enqueuedResumes[0].RunID)
	}
	if f.jobQueue.enqueuedResumes[0].StepID != f.stepID {
		t.Errorf("expected enqueued step_id %s, got %s", f.stepID, f.jobQueue.enqueuedResumes[0].StepID)
	}
}

func TestHITLService_Approve_RunNotFound(t *testing.T) {
	f := newHITLTestFixture()

	unknownRunID := uuid.New()
	_, err := f.service.Approve(context.Background(), f.projectID, unknownRunID, f.reviewerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != codeRunNotFound {
		t.Errorf("expected code %s, got %s", codeRunNotFound, domainErr.Code)
	}
}

func TestHITLService_Approve_ProjectMismatch(t *testing.T) {
	f := newHITLTestFixture()

	otherProjectID := uuid.New()
	_, err := f.service.Approve(context.Background(), otherProjectID, f.runID, f.reviewerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != codeRunNotFound {
		t.Errorf("expected code %s, got %s", codeRunNotFound, domainErr.Code)
	}
}

func TestHITLService_Approve_NoHITLRequest(t *testing.T) {
	f := newHITLTestFixture()
	f.hitlRepo.getPendingByRunIDFn = func(_ context.Context, _ uuid.UUID) (*model.HITLRequest, error) {
		return nil, errors.NewNotFound("hitl_request", f.runID)
	}

	_, err := f.service.Approve(context.Background(), f.projectID, f.runID, f.reviewerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != "HITL_REQUEST_NOT_FOUND" {
		t.Errorf("expected code HITL_REQUEST_NOT_FOUND, got %s", domainErr.Code)
	}
}

func TestHITLService_Approve_AlreadyResolved(t *testing.T) {
	f := newHITLTestFixture()
	f.hitlRequest.Status = model.HITLStatusApproved

	_, err := f.service.Approve(context.Background(), f.projectID, f.runID, f.reviewerID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != "HITL_ALREADY_RESOLVED" {
		t.Errorf("expected code HITL_ALREADY_RESOLVED, got %s", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryConflict {
		t.Errorf("expected category conflict, got %s", domainErr.Category)
	}

	// Verify no mutations occurred
	if len(f.eventPub.getEvents()) != 0 {
		t.Error("expected no events published for already-resolved request")
	}
	if len(f.jobQueue.enqueuedResumes) != 0 {
		t.Error("expected no jobs enqueued for already-resolved request")
	}
}

func TestHITLService_Reject_HappyPathWithReason(t *testing.T) {
	f := newHITLTestFixture()

	var updatedStatus model.HITLStatus
	var capturedReason *string
	f.hitlRepo.updateStatusFn = func(_ context.Context, _ uuid.UUID, status model.HITLStatus, _ *uuid.UUID, rejectionReason *string, _ time.Time) (*model.HITLRequest, error) {
		updatedStatus = status
		capturedReason = rejectionReason
		cp := *f.hitlRequest
		cp.Status = status
		return &cp, nil
	}

	var stepErrorMsg *string
	f.runRepo.updateRunStepStatusFn = func(_ context.Context, _ uuid.UUID, _ model.StepStatus, _ *time.Time, _ *time.Time, errorMsg *string) (*model.RunStep, error) {
		stepErrorMsg = errorMsg
		return &model.RunStep{ID: f.stepID, Status: model.StepStatusFailed}, nil
	}

	var runErrorMsg *string
	f.runRepo.updateRunStatusFn = func(_ context.Context, _ uuid.UUID, _ model.RunStatus, _ *time.Time, _ *time.Time, errorMsg *string) (*model.Run, error) {
		runErrorMsg = errorMsg
		cp := *f.run
		cp.Status = model.RunStatusFailed
		return &cp, nil
	}

	reason := "code review required"
	result, err := f.service.Reject(context.Background(), f.projectID, f.runID, f.reviewerID, reason)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %s", result.Status)
	}
	if updatedStatus != model.HITLStatusRejected {
		t.Errorf("expected HITL status rejected, got %s", updatedStatus)
	}
	if capturedReason == nil || *capturedReason != reason {
		t.Errorf("expected reason '%s', got %v", reason, capturedReason)
	}

	expectedErrorMsg := "HITL_REJECTED: code review required"
	if stepErrorMsg == nil || *stepErrorMsg != expectedErrorMsg {
		t.Errorf("expected step error '%s', got %v", expectedErrorMsg, stepErrorMsg)
	}
	if runErrorMsg == nil || *runErrorMsg != expectedErrorMsg {
		t.Errorf("expected run error '%s', got %v", expectedErrorMsg, runErrorMsg)
	}

	// Verify event published
	events := f.eventPub.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event.Action != "rejected" {
		t.Errorf("expected event action 'rejected', got %s", events[0].Event.Action)
	}

	// Verify no resume job enqueued for rejection
	if len(f.jobQueue.enqueuedResumes) != 0 {
		t.Error("expected no resume jobs enqueued for rejection")
	}
}

func TestHITLService_Reject_HappyPathWithoutReason(t *testing.T) {
	f := newHITLTestFixture()

	var stepErrorMsg *string
	f.runRepo.updateRunStepStatusFn = func(_ context.Context, _ uuid.UUID, _ model.StepStatus, _ *time.Time, _ *time.Time, errorMsg *string) (*model.RunStep, error) {
		stepErrorMsg = errorMsg
		return &model.RunStep{ID: f.stepID, Status: model.StepStatusFailed}, nil
	}

	var runErrorMsg *string
	f.runRepo.updateRunStatusFn = func(_ context.Context, _ uuid.UUID, _ model.RunStatus, _ *time.Time, _ *time.Time, errorMsg *string) (*model.Run, error) {
		runErrorMsg = errorMsg
		cp := *f.run
		cp.Status = model.RunStatusFailed
		return &cp, nil
	}

	result, err := f.service.Reject(context.Background(), f.projectID, f.runID, f.reviewerID, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %s", result.Status)
	}

	expectedErrorMsg := "HITL_REJECTED"
	if stepErrorMsg == nil || *stepErrorMsg != expectedErrorMsg {
		t.Errorf("expected step error '%s', got %v", expectedErrorMsg, stepErrorMsg)
	}
	if runErrorMsg == nil || *runErrorMsg != expectedErrorMsg {
		t.Errorf("expected run error '%s', got %v", expectedErrorMsg, runErrorMsg)
	}
}

func TestHITLService_Reject_AlreadyResolved(t *testing.T) {
	f := newHITLTestFixture()
	f.hitlRequest.Status = model.HITLStatusRejected

	_, err := f.service.Reject(context.Background(), f.projectID, f.runID, f.reviewerID, "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != "HITL_ALREADY_RESOLVED" {
		t.Errorf("expected code HITL_ALREADY_RESOLVED, got %s", domainErr.Code)
	}
	if domainErr.Category != errors.CategoryConflict {
		t.Errorf("expected category conflict, got %s", domainErr.Category)
	}
}

func TestHITLService_Reject_ProjectMismatch(t *testing.T) {
	f := newHITLTestFixture()

	otherProjectID := uuid.New()
	_, err := f.service.Reject(context.Background(), otherProjectID, f.runID, f.reviewerID, "reason")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != codeRunNotFound {
		t.Errorf("expected code %s, got %s", codeRunNotFound, domainErr.Code)
	}
}
