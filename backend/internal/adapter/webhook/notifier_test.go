package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/webhook"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func TestNotifier_Send_FullEventPayload(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json Content-Type, got %s", r.Header.Get("Content-Type"))
		}
		var err error
		received, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := webhook.NewNotifier()
	eventID := uuid.New()
	projectID := uuid.New()
	entityID := uuid.New()
	event := model.Event{
		ID:         eventID,
		ProjectID:  projectID,
		EntityType: "run",
		EntityID:   entityID,
		Action:     "completed",
		CreatedAt:  time.Now(),
	}

	err := n.Send(context.Background(), event, map[string]string{"url": srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(received, &payload); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if payload["id"] == nil {
		t.Error("expected 'id' field in payload")
	}
	if payload["project_id"] == nil {
		t.Error("expected 'project_id' field in payload")
	}
	if payload["entity_type"] != "run" {
		t.Errorf("expected entity_type=run, got %v", payload["entity_type"])
	}
	if payload["action"] != "completed" {
		t.Errorf("expected action=completed, got %v", payload["action"])
	}
}

func TestNotifier_Send_Non2xxResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	n := webhook.NewNotifier()
	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "failed",
		CreatedAt:  time.Now(),
	}

	err := n.Send(context.Background(), event, map[string]string{"url": srv.URL})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestNotifier_Send_MissingURL(t *testing.T) {
	n := webhook.NewNotifier()
	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "completed",
		CreatedAt:  time.Now(),
	}

	err := n.Send(context.Background(), event, map[string]string{})
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
}
