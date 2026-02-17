package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EventSubscriber defines the interface for subscribing to real-time events.
type EventSubscriber interface {
	// Subscribe returns a channel of events for the given project and a cleanup function.
	Subscribe(ctx context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error)

	// Close gracefully shuts down all subscriptions and releases resources.
	Close() error
}
