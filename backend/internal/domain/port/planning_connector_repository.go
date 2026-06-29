package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// PlanningConnectorRepository persists one planning connector configuration per
// project (status field, done options, status mapping, write-back toggles).
type PlanningConnectorRepository interface {
	// Get returns the project's connector, or a not-found DomainError when absent.
	Get(ctx context.Context, projectID uuid.UUID) (*model.PlanningConnector, error)
	// Upsert inserts or replaces the project's connector, returning the stored row.
	Upsert(ctx context.Context, c *model.PlanningConnector) (*model.PlanningConnector, error)
}

// PlanningWriteBackRepository appends audit rows for every write-back attempt.
type PlanningWriteBackRepository interface {
	// Create appends one audit row (success or failure) and returns the stored row.
	Create(ctx context.Context, w *model.PlanningWriteBack) (*model.PlanningWriteBack, error)
	// ListByStory returns a story's most recent write-back attempts (newest first).
	ListByStory(ctx context.Context, storyID uuid.UUID, limit int32) ([]*model.PlanningWriteBack, error)
}
