package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents a system event persisted in the events table.
type Event struct {
	ID         uuid.UUID       `json:"id"`
	ProjectID  uuid.UUID       `json:"project_id"`
	EntityType string          `json:"entity_type"` // e.g., "run", "step", "hitl"
	EntityID   uuid.UUID       `json:"entity_id"`
	Action     string          `json:"action"` // e.g., "started", "completed", "pending"
	Payload    json.RawMessage `json:"payload"`
	CreatedAt  time.Time       `json:"created_at"`
}

// EventName returns the dot-notation event name (entity_type.action).
func (e Event) EventName() string {
	return e.EntityType + "." + e.Action
}
