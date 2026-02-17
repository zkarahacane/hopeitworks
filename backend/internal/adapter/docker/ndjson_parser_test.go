package docker

import (
	"testing"
	"time"
)

func TestParseNDJSONLine(t *testing.T) {
	const testRunID = "run-123"
	const testStepID = "step-456"

	tests := []struct {
		name        string
		line        string
		wantNil     bool
		wantIsJSON  bool
		wantLevel   string
		wantMessage string
		wantDataLen int
		wantTS      time.Time
	}{
		{
			name:        "valid JSON with all fields",
			line:        `{"level":"error","message":"something failed","timestamp":"2026-02-17T10:30:00Z","extra":"data"}`,
			wantNil:     false,
			wantIsJSON:  true,
			wantLevel:   "error",
			wantMessage: "something failed",
			wantDataLen: 4,
			wantTS:      time.Date(2026, 2, 17, 10, 30, 0, 0, time.UTC),
		},
		{
			name:        "valid JSON missing level defaults to info",
			line:        `{"message":"no level here"}`,
			wantNil:     false,
			wantIsJSON:  true,
			wantLevel:   "info",
			wantMessage: "no level here",
			wantDataLen: 1,
		},
		{
			name:        "valid JSON missing timestamp uses time.Now",
			line:        `{"level":"debug","message":"no timestamp"}`,
			wantNil:     false,
			wantIsJSON:  true,
			wantLevel:   "debug",
			wantMessage: "no timestamp",
			wantDataLen: 2,
		},
		{
			name:        "valid JSON missing message defaults to empty",
			line:        `{"level":"warn"}`,
			wantNil:     false,
			wantIsJSON:  true,
			wantLevel:   "warn",
			wantMessage: "",
			wantDataLen: 1,
		},
		{
			name:        "malformed JSON wraps as raw text",
			line:        "this is not json",
			wantNil:     false,
			wantIsJSON:  false,
			wantLevel:   "info",
			wantMessage: "this is not json",
		},
		{
			name:    "empty line returns nil",
			line:    "",
			wantNil: true,
		},
		{
			name:    "whitespace-only line returns nil",
			line:    "   \t  ",
			wantNil: true,
		},
		{
			name:        "valid JSON with invalid timestamp falls back to time.Now",
			line:        `{"level":"info","message":"bad ts","timestamp":"not-a-timestamp"}`,
			wantNil:     false,
			wantIsJSON:  true,
			wantLevel:   "info",
			wantMessage: "bad ts",
			wantDataLen: 3,
		},
		{
			name:        "valid JSON array is not valid NDJSON object",
			line:        `[1,2,3]`,
			wantNil:     false,
			wantIsJSON:  false,
			wantLevel:   "info",
			wantMessage: "[1,2,3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			event := parseNDJSONLine(tt.line, testRunID, testStepID)

			if tt.wantNil {
				if event != nil {
					t.Fatalf("expected nil, got %+v", event)
				}
				return
			}

			if event == nil {
				t.Fatal("expected non-nil event, got nil")
			}

			if event.RunID != testRunID {
				t.Errorf("expected RunID=%s, got %s", testRunID, event.RunID)
			}
			if event.StepID != testStepID {
				t.Errorf("expected StepID=%s, got %s", testStepID, event.StepID)
			}
			if event.IsJSON != tt.wantIsJSON {
				t.Errorf("expected IsJSON=%v, got %v", tt.wantIsJSON, event.IsJSON)
			}
			if event.Level != tt.wantLevel {
				t.Errorf("expected Level=%s, got %s", tt.wantLevel, event.Level)
			}
			if event.Message != tt.wantMessage {
				t.Errorf("expected Message=%q, got %q", tt.wantMessage, event.Message)
			}

			if tt.wantIsJSON && len(event.Data) != tt.wantDataLen {
				t.Errorf("expected Data length=%d, got %d", tt.wantDataLen, len(event.Data))
			}

			if !tt.wantTS.IsZero() {
				if !event.Timestamp.Equal(tt.wantTS) {
					t.Errorf("expected Timestamp=%v, got %v", tt.wantTS, event.Timestamp)
				}
			} else if !tt.wantNil {
				// Timestamp should be approximately time.Now()
				if event.Timestamp.Before(before) {
					t.Errorf("expected Timestamp >= %v, got %v", before, event.Timestamp)
				}
			}

			// RawLine should always be set for non-nil events.
			if event.RawLine == "" {
				t.Error("expected RawLine to be set")
			}
		})
	}
}
