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

// HITLService provides business logic for HITL approval operations.
type HITLService struct {
	hitlRepo port.HITLRepository
	runRepo  port.RunRepository
	jobQueue port.JobQueue
	eventPub port.EventPublisher
	logger   *slog.Logger
}

// NewHITLService creates a new HITLService.
// jobQueue is used to re-enqueue run execution after an approval so the
// paused run resumes from the gate step. It may be nil (resume is then a no-op).
func NewHITLService(
	hitlRepo port.HITLRepository,
	runRepo port.RunRepository,
	jobQueue port.JobQueue,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *HITLService {
	return &HITLService{
		hitlRepo: hitlRepo,
		runRepo:  runRepo,
		jobQueue: jobQueue,
		eventPub: eventPub,
		logger:   logger,
	}
}

// GetByID returns a HITL request by its ID.
func (s *HITLService) GetByID(ctx context.Context, id uuid.UUID) (*model.HITLRequest, error) {
	return s.hitlRepo.GetByID(ctx, id)
}

// GetProjectIDForHITLRequest returns the project ID associated with a HITL request.
// This is used for RBAC checks to ensure users have access to the project.
func (s *HITLService) GetProjectIDForHITLRequest(ctx context.Context, hitlRequestID uuid.UUID) (uuid.UUID, error) {
	req, err := s.hitlRepo.GetByID(ctx, hitlRequestID)
	if err != nil {
		return uuid.Nil, err
	}
	step, err := s.runRepo.GetRunStep(ctx, req.RunStepID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get run step: %w", err)
	}
	run, err := s.runRepo.GetRun(ctx, step.RunID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get run: %w", err)
	}
	return run.ProjectID, nil
}

// ListPendingByProject returns all pending HITL requests for a project.
func (s *HITLService) ListPendingByProject(ctx context.Context, projectID uuid.UUID) ([]*model.PendingHITLRequest, int64, error) {
	pending, err := s.hitlRepo.ListPendingByProject(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	count, err := s.hitlRepo.CountPendingByProject(ctx, projectID)
	if err != nil {
		return nil, 0, err
	}
	return pending, count, nil
}

// ListAll returns a paginated list of HITL requests, optionally filtered by status.
func (s *HITLService) ListAll(ctx context.Context, status *string, page, perPage int) ([]*model.HITLRequest, int64, error) {
	limit, offset := paginationToLimitOffset(page, perPage)
	items, err := s.hitlRepo.ListFiltered(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.hitlRepo.CountFiltered(ctx, status)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByStepID returns the HITL request associated with a run step.
func (s *HITLService) GetByStepID(ctx context.Context, stepID uuid.UUID) (*model.HITLRequest, error) {
	return s.hitlRepo.GetByRunStepID(ctx, stepID)
}

// Approve resolves a HITL request as approved and resumes the pipeline step.
func (s *HITLService) Approve(ctx context.Context, hitlRequestID uuid.UUID, userID uuid.UUID) (*model.HITLRequest, error) {
	req, err := s.hitlRepo.GetByID(ctx, hitlRequestID)
	if err != nil {
		return nil, err
	}
	if req.Status != model.HITLStatusPending {
		return nil, errors.NewValidation("status",
			fmt.Sprintf("HITL request is already %s, cannot approve", req.Status))
	}

	now := time.Now()
	updated, err := s.hitlRepo.UpdateStatus(ctx, hitlRequestID, model.HITLStatusApproved, &userID, nil, now)
	if err != nil {
		return nil, fmt.Errorf("update HITL status to approved: %w", err)
	}

	// Mark the gate step completed so the resumed executor skips it and
	// continues with the next step. Leaving it "running" would cause the gate
	// action to re-execute on resume and suspend the run again (infinite loop).
	if _, err := s.runRepo.UpdateRunStepStatus(ctx, req.RunStepID, model.StepStatusCompleted, nil, &now, nil); err != nil {
		s.logger.Warn("failed to mark gate step completed after approval",
			"hitl_request_id", hitlRequestID, "step_id", req.RunStepID, "error", err)
	}

	// Resume the paused run and re-enqueue execution so the pipeline continues.
	// The gate action paused the run on suspension; without re-enqueuing here the
	// run would never resume on its own.
	s.resumeRun(ctx, req.RunStepID)

	// Publish approval event
	s.publishEvent(ctx, req, "approved", userID)

	return updated, nil
}

// resumeRun transitions the run owning the given step back to running and
// re-enqueues it for execution. Failures are logged but do not fail the
// approval — the run can still be resumed manually via the run API.
func (s *HITLService) resumeRun(ctx context.Context, stepID uuid.UUID) {
	step, err := s.runRepo.GetRunStep(ctx, stepID)
	if err != nil {
		s.logger.Warn("failed to fetch step for run resume after approval",
			"step_id", stepID, "error", err)
		return
	}

	run, err := s.runRepo.GetRun(ctx, step.RunID)
	if err != nil {
		s.logger.Warn("failed to fetch run for resume after approval",
			"step_id", stepID, "run_id", step.RunID, "error", err)
		return
	}

	// Only transition paused runs back to running. If the run is already running
	// (e.g. resumed concurrently), just re-enqueue execution.
	if run.Status == model.RunStatusPaused {
		if _, err := s.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusRunning, nil, nil, nil, nil); err != nil {
			s.logger.Warn("failed to transition run to running after approval",
				"run_id", run.ID, "error", err)
		}
	}

	if s.jobQueue == nil {
		s.logger.Warn("job queue unavailable, run will not resume automatically after approval",
			"run_id", run.ID)
		return
	}
	if err := s.jobQueue.EnqueueExecuteRun(ctx, run.ID); err != nil {
		s.logger.Warn("failed to re-enqueue run execution after approval",
			"run_id", run.ID, "error", err)
	}
}

// Reject resolves a HITL request as rejected.
func (s *HITLService) Reject(ctx context.Context, hitlRequestID uuid.UUID, userID uuid.UUID, reason *string) (*model.HITLRequest, error) {
	req, err := s.hitlRepo.GetByID(ctx, hitlRequestID)
	if err != nil {
		return nil, err
	}
	if req.Status != model.HITLStatusPending {
		return nil, errors.NewValidation("status",
			fmt.Sprintf("HITL request is already %s, cannot reject", req.Status))
	}

	now := time.Now()
	updated, err := s.hitlRepo.UpdateStatus(ctx, hitlRequestID, model.HITLStatusRejected, &userID, reason, now)
	if err != nil {
		return nil, fmt.Errorf("update HITL status to rejected: %w", err)
	}

	// Transition step to failed
	failMsg := "rejected by reviewer"
	if reason != nil && *reason != "" {
		failMsg = fmt.Sprintf("rejected: %s", *reason)
	}
	if _, err := s.runRepo.UpdateRunStepStatus(ctx, req.RunStepID, model.StepStatusFailed, nil, &now, &failMsg); err != nil {
		s.logger.Warn("failed to transition step to failed after rejection",
			"hitl_request_id", hitlRequestID, "step_id", req.RunStepID, "error", err)
	}

	// Publish rejection event
	s.publishEvent(ctx, req, "rejected", userID)

	return updated, nil
}

func (s *HITLService) publishEvent(ctx context.Context, req *model.HITLRequest, action string, userID uuid.UUID) {
	if s.eventPub == nil {
		return
	}

	// Fetch the run step to get the run ID for the event
	step, err := s.runRepo.GetRunStep(ctx, req.RunStepID)
	if err != nil {
		s.logger.Warn("failed to fetch run step for event publish",
			"hitl_request_id", req.ID, "step_id", req.RunStepID, "error", err)
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"hitl_request_id": req.ID.String(),
		"run_id":          step.RunID.String(),
		"step_id":         req.RunStepID.String(),
		"user_id":         userID.String(),
	})

	// Get project ID from the run
	run, err := s.runRepo.GetRun(ctx, step.RunID)
	if err != nil {
		s.logger.Warn("failed to fetch run for event publish",
			"run_id", step.RunID, "error", err)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  run.ProjectID,
		EntityType: "hitl_gate",
		EntityID:   req.ID,
		Action:     action,
		Payload:    payload,
		CreatedAt:  time.Now(),
	}

	if pubErr := s.eventPub.Publish(ctx, event); pubErr != nil {
		s.logger.Warn("failed to publish HITL event",
			"event", event.EventName(), "error", pubErr)
	}
}
