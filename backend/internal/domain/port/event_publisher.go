package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EventPublisher defines the interface for persisting and publishing events.
type EventPublisher interface {
	// Publish persists an event to the database, triggering NOTIFY automatically.
	Publish(ctx context.Context, event model.Event) error
}
