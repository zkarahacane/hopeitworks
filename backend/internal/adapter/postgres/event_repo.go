package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	apperrors "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Ensure EventRepo implements port.EventPublisher at compile time.
var _ port.EventPublisher = (*EventRepo)(nil)

// Ensure EventRepo implements port.EventRepository at compile time.
var _ port.EventRepository = (*EventRepo)(nil)

// EventRepo implements port.EventPublisher using sqlc-generated queries.
type EventRepo struct {
	queries *Queries
}

// NewEventRepo creates a new EventRepo.
func NewEventRepo(queries *Queries) *EventRepo {
	return &EventRepo{queries: queries}
}

// Publish persists an event to the database. The Postgres trigger automatically
// fires NOTIFY on the "events" channel after insert.
func (r *EventRepo) Publish(ctx context.Context, event model.Event) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	var payload []byte
	if event.Payload != nil {
		payload = []byte(event.Payload)
	}

	params := CreateEventParams{
		ID:         event.ID,
		ProjectID:  event.ProjectID,
		EntityType: event.EntityType,
		EntityID:   event.EntityID,
		Action:     event.Action,
		Payload:    payload,
		CreatedAt:  event.CreatedAt,
	}

	_, err := r.queries.CreateEvent(ctx, params)
	if err != nil {
		if isForeignKeyViolation(err) {
			return apperrors.NewNotFound("project", event.ProjectID)
		}
		return apperrors.NewInternal("failed to publish event", err)
	}
	return nil
}

// GetEventsSince returns all events for the project created after the event
// identified by afterEventID. Returns empty slice if afterEventID is unknown
// (the subquery returns no rows, so the WHERE clause matches nothing).
func (r *EventRepo) GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error) {
	rows, err := r.queries.GetEventsSince(ctx, GetEventsSinceParams{
		ProjectID: projectID,
		ID:        afterEventID,
	})
	if err != nil {
		return nil, apperrors.NewInternal("failed to query events since", err)
	}

	events := make([]*model.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, &model.Event{
			ID:         row.ID,
			ProjectID:  row.ProjectID,
			EntityType: row.EntityType,
			EntityID:   row.EntityID,
			Action:     row.Action,
			Payload:    json.RawMessage(row.Payload),
			CreatedAt:  row.CreatedAt,
		})
	}
	return events, nil
}
