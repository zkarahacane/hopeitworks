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
	// GetByID returns a HITL request by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.HITLRequest, error)
	// GetByRunStepID returns the HITL request for the given run step.
	GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error)
	// UpdateStatus transitions the HITL request to approved or rejected.
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error)
	// ListPendingByProject returns all pending HITL requests for a project with denormalized context.
	ListPendingByProject(ctx context.Context, projectID uuid.UUID) ([]*model.PendingHITLRequest, error)
	// CountPendingByProject returns the count of pending HITL requests for a project.
	CountPendingByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
	// ListFiltered returns HITL requests optionally filtered by status with pagination.
	ListFiltered(ctx context.Context, status *string, limit, offset int32) ([]*model.HITLRequest, error)
	// CountFiltered returns the count of HITL requests optionally filtered by status.
	CountFiltered(ctx context.Context, status *string) (int64, error)
}
