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

	if eventType, ok := data["type"].(string); ok {
		event.Type = eventType
	}

	if event.Type == "cost" {
		if v, ok := data["input_tokens"].(float64); ok {
			event.InputTokens = int64(v)
		}
		if v, ok := data["output_tokens"].(float64); ok {
			event.OutputTokens = int64(v)
		}
		if v, ok := data["model"].(string); ok {
			event.Model = v
		}
	}

	// Handle Claude Code stream-json "result" events, which contain authoritative
	// cumulative token usage for the entire run.
	if event.Type == "result" {
		event.Type = "cost"
		if usageMap, ok := data["usage"].(map[string]any); ok {
			if v, ok := usageMap["input_tokens"].(float64); ok {
				event.InputTokens = int64(v)
			}
			if v, ok := usageMap["output_tokens"].(float64); ok {
				event.OutputTokens = int64(v)
			}
		}
		if modelUsage, ok := data["modelUsage"].(map[string]any); ok {
			event.Model = pickPrimaryModel(modelUsage)
		}
	}

	return event
}

// pickPrimaryModel selects the primary model from a modelUsage map by choosing
// the key whose entry has the highest inputTokens count. Falls back to the first
// key if no numeric usage data is present.
func pickPrimaryModel(modelUsage map[string]any) string {
	var bestModel string
	var bestTokens float64
	for modelID, v := range modelUsage {
		entry, ok := v.(map[string]any)
		if !ok {
			if bestModel == "" {
				bestModel = modelID
			}
			continue
		}
		inputTokens, _ := entry["inputTokens"].(float64)
		if bestModel == "" || inputTokens > bestTokens {
			bestModel = modelID
			bestTokens = inputTokens
		}
	}
	return bestModel
}
