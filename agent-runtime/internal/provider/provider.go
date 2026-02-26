// Package provider abstracts LLM CLI execution for the agent runtime.
// It supports multiple providers (Claude Code, OpenCode) through a common interface.
package provider

import (
	"context"
	"fmt"
)

// Event represents a structured event from the LLM provider.
type Event struct {
	Type         string  // "log", "cost", "result"
	Message      string  // log message or result summary
	InputTokens  int64   // token usage
	OutputTokens int64   // token usage
	Model        string  // model used
	CostUSD      float64 // computed cost (for cost/result events)
	ExitCode     int     // only for "result" events
}

// Provider abstracts LLM CLI execution.
type Provider interface {
	// Run executes the LLM CLI in workDir with the given prompt.
	// Returns a channel of events (logs, cost, final result).
	Run(ctx context.Context, workDir string, prompt string, model string) (<-chan Event, error)
}

// New creates a Provider based on provider name ("claude" or "opencode").
func New(name string, apiKey string) (Provider, error) {
	switch name {
	case "claude":
		return &ClaudeProvider{apiKey: apiKey}, nil
	case "opencode":
		return &OpenCodeProvider{apiKey: apiKey}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q: must be \"claude\" or \"opencode\"", name)
	}
}
