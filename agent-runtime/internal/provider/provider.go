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

// RunOptions carries the provisioned capabilities applied to a harness invocation.
// The zero value applies nothing — a capability-less run produces the exact same CLI
// command as before this struct existed (back-compat). Each provider translates the
// fields it supports and warn+skips the rest.
type RunOptions struct {
	// SystemPromptAppend is appended to the harness system prompt (claude:
	// --append-system-prompt). Empty means no extra prompt.
	SystemPromptAppend string
	// MCPConfigPath is the path to a materialised .mcp.json (claude: --mcp-config).
	// Empty means no MCP servers.
	MCPConfigPath string
	// AllowedTools / DisallowedTools are the tool policy (claude: --allowedTools /
	// --disallowedTools). Empty means no policy.
	AllowedTools    []string
	DisallowedTools []string
	// ExtraEnv are KEY=value pairs (resolved secrets) added to the harness child
	// process environment ONLY — never the container env, so they never leak via
	// `docker inspect`. The harness expands ${KEY} references (e.g. in .mcp.json headers).
	ExtraEnv []string
}

// Provider abstracts LLM CLI execution.
type Provider interface {
	// Run executes the LLM CLI in workDir with the given prompt and provisioned
	// capabilities. Returns a channel of events (logs, cost, final result).
	Run(ctx context.Context, workDir string, prompt string, model string, opts RunOptions) (<-chan Event, error)
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
