package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EventRepository defines read access to persisted events.
type EventRepository interface {
	// GetEventsSince returns all events for the project created after the event
	// identified by afterEventID. Returns empty slice if afterEventID is unknown.
	GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error)
}
