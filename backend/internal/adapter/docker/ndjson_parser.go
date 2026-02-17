package docker

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// parseNDJSONLine parses a single log line as NDJSON.
// Returns nil if the line is empty (skip).
// Returns a LogEvent with IsJSON=false if JSON parsing fails.
func parseNDJSONLine(line string, runID string, stepID string) *model.LogEvent {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	event := &model.LogEvent{
		RunID:     runID,
		StepID:    stepID,
		RawLine:   line,
		Timestamp: time.Now(),
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		event.IsJSON = false
		event.Level = "info"
		event.Message = line
		return event
	}

	event.IsJSON = true
	event.Data = data

	if level, ok := data["level"].(string); ok {
		event.Level = level
	} else {
		event.Level = "info"
	}

	if message, ok := data["message"].(string); ok {
		event.Message = message
	}

	if ts, ok := data["timestamp"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			event.Timestamp = parsed
		}
	}

	return event
}
