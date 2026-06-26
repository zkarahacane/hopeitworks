package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
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

func TestScrub_TokenPatterns(t *testing.T) {
	cases := []struct {
		name string
		in   string
		leak string // substring that must NOT survive
	}{
		{"classic PAT", "auth failed for ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"},
		{"oauth token", "token gho_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345", "gho_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"},
		{"fine-grained PAT", "using github_pat_11ABCDE0123_secretpart", "github_pat_11ABCDE0123_secretpart"},
		{"url credentials", "clone https://x-access-token:ghp_TOKEN1234567890abcdef@github.com/o/r.git failed", "ghp_TOKEN1234567890abcdef@"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := Scrub(tc.in)
			if strings.Contains(out, tc.leak) {
				t.Fatalf("Scrub leaked %q in %q", tc.leak, out)
			}
			if !strings.Contains(out, redactedValue) {
				t.Fatalf("Scrub did not redact: %q", out)
			}
		})
	}
}

func TestScrubHandler_RedactsTokenInMessageAndAttrs(t *testing.T) {
	const tok = "ghp_LEAKED000111222333444555666777888"
	var buf bytes.Buffer
	logger := slog.New(&ScrubHandler{Handler: slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})})

	logger.Info("cloning with "+tok,
		slog.String("repo_url", "https://user:"+tok+"@github.com/o/r"),
		slog.String("note", "raw token "+tok),
		slog.Group("nested", slog.String("inner_token_field", tok)),
	)

	out := buf.String()
	if strings.Contains(out, tok) {
		t.Fatalf("token survived scrubbing in log output: %s", out)
	}
}

func TestScrubHandler_KeySubstringAndSafeIdentifiers(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(&ScrubHandler{Handler: slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})})

	logger.Info("t",
		slog.String("encryption_key", "supersecret"), // *key* -> redacted
		slog.String("access_token_value", "tok"),     // *token* -> redacted
		slog.String("story_key", "S-14"),             // identifier -> preserved
		slog.String("stack_key", "go-1.23"),          // identifier -> preserved
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log: %v", err)
	}
	if entry["encryption_key"] != redactedValue {
		t.Errorf("encryption_key should be redacted, got %v", entry["encryption_key"])
	}
	if entry["access_token_value"] != redactedValue {
		t.Errorf("access_token_value should be redacted, got %v", entry["access_token_value"])
	}
	if entry["story_key"] != "S-14" {
		t.Errorf("story_key must stay readable, got %v", entry["story_key"])
	}
	if entry["stack_key"] != "go-1.23" {
		t.Errorf("stack_key must stay readable, got %v", entry["stack_key"])
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
