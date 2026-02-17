package model

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestEvent_EventName(t *testing.T) {
	tests := []struct {
		entityType string
		action     string
		want       string
	}{
		{"run", "started", "run.started"},
		{"step", "completed", "step.completed"},
		{"hitl", "pending", "hitl.pending"},
		{"pipeline", "failed", "pipeline.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			e := Event{
				ID:         uuid.New(),
				ProjectID:  uuid.New(),
				EntityType: tt.entityType,
				EntityID:   uuid.New(),
				Action:     tt.action,
			}
			got := e.EventName()
			if got != tt.want {
				t.Errorf("EventName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEvent_JSONSerialization(t *testing.T) {
	e := Event{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "started",
		Payload:    json.RawMessage(`{"step_count":5}`),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.EntityType != e.EntityType {
		t.Errorf("entity_type = %q, want %q", decoded.EntityType, e.EntityType)
	}
	if decoded.Action != e.Action {
		t.Errorf("action = %q, want %q", decoded.Action, e.Action)
	}
	if string(decoded.Payload) != string(e.Payload) {
		t.Errorf("payload = %q, want %q", string(decoded.Payload), string(e.Payload))
	}
}
