package model

import "context"

// Action represents a pipeline step action (e.g., agent_run, git_create_pr, hitl_gate).
// Concrete implementations handle specific action types.
type Action interface {
	// Name returns the action identifier matching pipeline config step action field.
	// Examples: "agent_run", "git_create_pr", "hitl_gate"
	Name() string

	// Execute runs the action with the given run context.
	// Returns nil on success, error on failure.
	// The error will be stored in run_step.error_message and cause run failure.
	Execute(ctx context.Context, runCtx *RunContext) error
}
