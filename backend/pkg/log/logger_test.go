package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}
	for _, level := range levels {
		logger := New(level)
		if logger == nil {
			t.Errorf("New(%q) returned nil", level)
		}
	}
}

const redactedValue = "[REDACTED]"

func TestScrubHandler(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	scrubbed := &ScrubHandler{Handler: jsonHandler}
	logger := slog.New(scrubbed)

	logger.Info("test",
		slog.String("username", "alice"),
		slog.String("password", "secret123"),
		slog.String("token", "abc"),
		slog.String("api_key", "key123"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["username"] != "alice" {
		t.Errorf("username should not be redacted, got %q", entry["username"])
	}
	if entry["password"] != redactedValue {
		t.Errorf("password should be redacted, got %q", entry["password"])
	}
	if entry["token"] != redactedValue {
		t.Errorf("token should be redacted, got %q", entry["token"])
	}
	if entry["api_key"] != redactedValue {
		t.Errorf("api_key should be redacted, got %q", entry["api_key"])
	}
}

func TestContextHelpers(t *testing.T) {
	logger := New("info")
	ctx := context.Background()

	// FromContext with no logger should return default
	got := FromContext(ctx)
	if got == nil {
		t.Fatal("FromContext should return default logger, not nil")
	}

	// WithLogger + FromContext round-trip
	ctx = WithLogger(ctx, logger)
	got = FromContext(ctx)
	if got != logger {
		t.Error("FromContext should return the logger stored with WithLogger")
	}
}
