package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// HITLRepository defines persistence operations for HITL approval requests.
type HITLRepository interface {
	// Create persists a new HITL request with status "pending".
	Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
	// GetByRunStepID returns the HITL request for the given run step.
	GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error)
	// UpdateStatus transitions the HITL request to approved or rejected.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error)
}
