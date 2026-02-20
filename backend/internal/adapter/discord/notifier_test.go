package discord_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zakari/hopeitworks/backend/internal/adapter/discord"
	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

func TestNotifier_Send_CorrectPayload(t *testing.T) {
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
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	n := discord.NewNotifier()
	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		EntityType: "run",
		EntityID:   uuid.New(),
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
	embeds, ok := payload["embeds"].([]interface{})
	if !ok || len(embeds) == 0 {
		t.Fatal("expected embeds in payload")
	}
}

func TestNotifier_Send_Colors(t *testing.T) {
	tests := []struct {
		entityType string
		action     string
		wantColor  float64
	}{
		{"run", "completed", 0x2ECC71},
		{"run", "failed", 0xE74C3C},
		{"hitl_gate", "pending", 0xF1C40F},
		{"run", "unknown", 0x95A5A6},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.entityType+"."+tt.action, func(t *testing.T) {
			var received []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received, _ = io.ReadAll(r.Body)
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()

			n := discord.NewNotifier()
			event := model.Event{
				ID:         uuid.New(),
				ProjectID:  uuid.New(),
				EntityType: tt.entityType,
				EntityID:   uuid.New(),
				Action:     tt.action,
				CreatedAt:  time.Now(),
			}

			err := n.Send(context.Background(), event, map[string]string{"url": srv.URL})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var payload map[string]interface{}
			_ = json.Unmarshal(received, &payload)
			embeds := payload["embeds"].([]interface{})
			embed := embeds[0].(map[string]interface{})
			color := embed["color"].(float64)

			if color != tt.wantColor {
				t.Errorf("color: got %.0f, want %.0f", color, tt.wantColor)
			}
		})
	}
}

func TestNotifier_Send_MissingURL(t *testing.T) {
	n := discord.NewNotifier()
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

func TestNotifier_Send_Non2xxResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := discord.NewNotifier()
	event := model.Event{
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		EntityType: "run",
		EntityID:   uuid.New(),
		Action:     "completed",
		CreatedAt:  time.Now(),
	}

	err := n.Send(context.Background(), event, map[string]string{"url": srv.URL})
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}
