package service

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// CircuitBreakerService manages circuit breaker state for projects.
// When consecutive run failures exceed a threshold, the circuit breaker trips
// and blocks new pipeline runs until an admin resets it.
type CircuitBreakerService struct {
	projectRepo port.ProjectRepository
	eventPub    port.EventPublisher
	logger      *slog.Logger
}

// NewCircuitBreakerService creates a new CircuitBreakerService.
func NewCircuitBreakerService(
	projectRepo port.ProjectRepository,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *CircuitBreakerService {
	return &CircuitBreakerService{
		projectRepo: projectRepo,
		eventPub:    eventPub,
		logger:      logger,
	}
}

// CheckCircuitBreaker verifies that the circuit breaker is not active for a project.
// Returns a CIRCUIT_BREAKER_OPEN error if the circuit breaker is active.
func (s *CircuitBreakerService) CheckCircuitBreaker(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	if project.CircuitBreakerActive {
		return &errors.DomainError{
			Category: errors.CategoryValidation,
			Code:     "CIRCUIT_BREAKER_OPEN",
			Message:  "circuit breaker is active for this project — all pipeline runs are paused",
		}
	}

	return nil
}

// RecordFailure increments the circuit breaker failure count for a project.
// If the threshold is reached, the circuit breaker trips and an event is published.
func (s *CircuitBreakerService) RecordFailure(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.projectRepo.IncrementCircuitBreakerCount(ctx, projectID)
	if err != nil {
		return err
	}

	s.logger.Info("circuit breaker failure recorded",
		"project_id", projectID,
		"count", project.CircuitBreakerCount,
		"max", project.CircuitBreakerMax,
		"active", project.CircuitBreakerActive,
	)

	if project.CircuitBreakerActive {
		s.logger.Warn("circuit breaker triggered",
			"project_id", projectID,
			"count", project.CircuitBreakerCount,
		)
		s.publishCircuitBreakerEvent(ctx, project, "triggered")
	}

	return nil
}

// RecordSuccess resets the circuit breaker failure count on a successful run.
// This prevents stale failure counts from tripping the breaker later.
func (s *CircuitBreakerService) RecordSuccess(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	// Only reset if there are pending failure counts
	if project.CircuitBreakerCount > 0 {
		if _, err := s.projectRepo.ResetCircuitBreaker(ctx, projectID); err != nil {
			return err
		}
		s.logger.Info("circuit breaker count reset on success",
			"project_id", projectID,
		)
	}

	return nil
}

// Reset resets the circuit breaker for a project. Only admins should call this.
func (s *CircuitBreakerService) Reset(ctx context.Context, projectID uuid.UUID) error {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return err
	}

	if !project.CircuitBreakerActive && project.CircuitBreakerCount == 0 {
		return nil // nothing to reset
	}

	project, err = s.projectRepo.ResetCircuitBreaker(ctx, projectID)
	if err != nil {
		return err
	}

	s.logger.Info("circuit breaker reset by admin",
		"project_id", projectID,
	)

	s.publishCircuitBreakerEvent(ctx, project, "reset")

	return nil
}

// publishCircuitBreakerEvent publishes a circuit breaker event.
func (s *CircuitBreakerService) publishCircuitBreakerEvent(ctx context.Context, project *model.Project, action string) {
	payload, err := json.Marshal(map[string]any{
		"project_id": project.ID.String(),
		"count":      project.CircuitBreakerCount,
		"max":        project.CircuitBreakerMax,
	})
	if err != nil {
		s.logger.Error("failed to marshal circuit breaker event payload",
			"project_id", project.ID,
			"error", err,
		)
		return
	}

	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  project.ID,
		EntityType: "circuit_breaker",
		EntityID:   project.ID,
		Action:     action,
		Payload:    payload,
	}

	if err := s.eventPub.Publish(ctx, event); err != nil {
		s.logger.Error("failed to publish circuit breaker event",
			"project_id", project.ID,
			"action", action,
			"error", err,
		)
	}
}
