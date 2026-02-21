package port

import (
	"context"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// Notifier dispatches a single notification for a given event.
type Notifier interface {
	Send(ctx context.Context, event model.Event, config map[string]string) error
}
