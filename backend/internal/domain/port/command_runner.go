package port

import "context"

// CommandRunner abstracts shell command execution for testability.
type CommandRunner interface {
	// Run executes a command in the specified working directory.
	// Returns stdout on success, or an error with combined output on failure.
	Run(ctx context.Context, workDir string, name string, args ...string) (stdout string, err error)
}
