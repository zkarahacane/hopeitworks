package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// HITLService provides business logic for approving and rejecting HITL gates.
type HITLService struct {
	hitlRepo port.HITLRepository
	runRepo  port.RunRepository
	eventPub port.EventPublisher
	jobQueue port.JobQueue
	logger   *slog.Logger
}

// NewHITLService creates a new HITLService.
func NewHITLService(
	hitlRepo port.HITLRepository,
	runRepo port.RunRepository,
	eventPub port.EventPublisher,
	jobQueue port.JobQueue,
	logger *slog.Logger,
) *HITLService {
	return &HITLService{
		hitlRepo: hitlRepo,
		runRepo:  runRepo,
		eventPub: eventPub,
		jobQueue: jobQueue,
		logger:   logger,
	}
}

// ApproveResult holds the data returned by a successful approval.
type ApproveResult struct {
	RunID         uuid.UUID
	HITLRequestID uuid.UUID
	Status        string
}

// RejectResult holds the data returned by a successful rejection.
type RejectResult struct {
	RunID         uuid.UUID
	HITLRequestID uuid.UUID
	Status        string
}

// Approve approves a pending HITL gate for the given run, resuming pipeline execution.
func (s *HITLService) Approve(ctx context.Context, projectID, runID, reviewerID uuid.UUID) (*ApproveResult, error) {
	// Fetch run and verify project ownership
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	// Fetch pending HITL request
	hitlReq, err := s.hitlRepo.GetPendingByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Guard idempotency
	if hitlReq.Status != model.HITLStatusPending {
		return nil, &errors.DomainError{
			Category: errors.CategoryConflict,
			Code:     "HITL_ALREADY_RESOLVED",
			Message:  fmt.Sprintf("HITL request %s is already %s", hitlReq.ID, hitlReq.Status),
		}
	}

	// Update HITL status to approved
	now := time.Now()
	if _, err := s.hitlRepo.UpdateStatus(ctx, hitlReq.ID, model.HITLStatusApproved, &reviewerID, nil, now); err != nil {
		return nil, err
	}

	// Transition step from waiting_approval to running
	if _, err := s.runRepo.UpdateRunStepStatus(ctx, hitlReq.RunStepID, model.StepStatusRunning, nil, nil, nil); err != nil {
		return nil, err
	}

	// Publish hitl_gate.approved event (non-fatal)
	s.publishApprovedEvent(ctx, projectID, runID, hitlReq.RunStepID, hitlReq.ID)

	// Enqueue resume job (non-fatal)
	if err := s.jobQueue.EnqueueResumeRun(ctx, runID, hitlReq.RunStepID); err != nil {
		s.logger.Error("failed to enqueue resume_run job",
			"run_id", runID, "step_id", hitlReq.RunStepID, "error", err)
	}

	return &ApproveResult{
		RunID:         runID,
		HITLRequestID: hitlReq.ID,
		Status:        string(run.Status),
	}, nil
}

// Reject rejects a pending HITL gate for the given run, failing the pipeline.
func (s *HITLService) Reject(ctx context.Context, projectID, runID, reviewerID uuid.UUID, reason string) (*RejectResult, error) {
	// Fetch run and verify project ownership
	run, err := s.runRepo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ProjectID != projectID {
		return nil, errors.NewNotFound("run", runID)
	}

	// Fetch pending HITL request
	hitlReq, err := s.hitlRepo.GetPendingByRunID(ctx, runID)
	if err != nil {
		return nil, err
	}

	// Guard idempotency
	if hitlReq.Status != model.HITLStatusPending {
		return nil, &errors.DomainError{
			Category: errors.CategoryConflict,
			Code:     "HITL_ALREADY_RESOLVED",
			Message:  fmt.Sprintf("HITL request %s is already %s", hitlReq.ID, hitlReq.Status),
		}
	}

	// Build error message
	errorMsg := "HITL_REJECTED"
	if reason != "" {
		errorMsg = fmt.Sprintf("HITL_REJECTED: %s", reason)
	}

	// Update HITL status to rejected
	now := time.Now()
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	if _, err := s.hitlRepo.UpdateStatus(ctx, hitlReq.ID, model.HITLStatusRejected, &reviewerID, reasonPtr, now); err != nil {
		return nil, err
	}

	// Transition step to failed
	if _, err := s.runRepo.UpdateRunStepStatus(ctx, hitlReq.RunStepID, model.StepStatusFailed, nil, &now, &errorMsg); err != nil {
		return nil, err
	}

	// Transition run to failed
	if _, err := s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusFailed, nil, &now, &errorMsg); err != nil {
		return nil, err
	}

	// Publish hitl_gate.rejected event (non-fatal)
	s.publishRejectedEvent(ctx, projectID, runID, hitlReq.RunStepID, hitlReq.ID, reason)

	return &RejectResult{
		RunID:         runID,
		HITLRequestID: hitlReq.ID,
		Status:        "failed",
	}, nil
}

// publishApprovedEvent publishes a hitl_gate.approved event.
func (s *HITLService) publishApprovedEvent(ctx context.Context, projectID, runID, stepID, hitlRequestID uuid.UUID) {
	payload := map[string]string{
		"run_id":          runID.String(),
		"step_id":         stepID.String(),
		"hitl_request_id": hitlRequestID.String(),
		"project_id":      projectID.String(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal hitl_gate.approved payload", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "hitl_gate",
		EntityID:   stepID,
		Action:     "approved",
		Payload:    payloadJSON,
	}

	if err := s.eventPub.Publish(ctx, event); err != nil {
		s.logger.Error("failed to publish hitl_gate.approved event", "error", err)
	}
}

// publishRejectedEvent publishes a hitl_gate.rejected event.
func (s *HITLService) publishRejectedEvent(ctx context.Context, projectID, runID, stepID, hitlRequestID uuid.UUID, reason string) {
	payload := map[string]string{
		"run_id":          runID.String(),
		"step_id":         stepID.String(),
		"hitl_request_id": hitlRequestID.String(),
		"project_id":      projectID.String(),
		"reason":          reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("failed to marshal hitl_gate.rejected payload", "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  projectID,
		EntityType: "hitl_gate",
		EntityID:   stepID,
		Action:     "rejected",
		Payload:    payloadJSON,
	}

	if err := s.eventPub.Publish(ctx, event); err != nil {
		s.logger.Error("failed to publish hitl_gate.rejected event", "error", err)
	}
}
