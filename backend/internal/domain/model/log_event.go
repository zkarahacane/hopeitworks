package model

import "time"

// LogEvent represents a single log event from an agent container.
type LogEvent struct {
	// RunID is the ID of the run this log belongs to.
	RunID string `json:"run_id"`

	// StepID is the ID of the step this log belongs to.
	StepID string `json:"step_id"`

	// Timestamp is when the log event was generated.
	Timestamp time.Time `json:"timestamp"`

	// Level is the log level (info, warn, error, debug).
	Level string `json:"level"`

	// Message is the log message.
	Message string `json:"message"`

	// RawLine is the raw log line before parsing.
	RawLine string `json:"raw_line"`

	// IsJSON indicates whether the line was valid NDJSON.
	IsJSON bool `json:"is_json"`

	// Data contains parsed JSON fields (only populated when IsJSON is true).
	Data map[string]any `json:"data,omitempty"`

	// Type is populated from Data["type"] when IsJSON is true (e.g., "cost").
	Type string `json:"type,omitempty"`

	// InputTokens is the number of input tokens (populated when Type == "cost").
	InputTokens int64 `json:"input_tokens,omitempty"`

	// OutputTokens is the number of output tokens (populated when Type == "cost").
	OutputTokens int64 `json:"output_tokens,omitempty"`

	// Model is the model name (populated when Type == "cost").
	Model string `json:"model,omitempty"`
}
