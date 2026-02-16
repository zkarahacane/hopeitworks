package exec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// RealCommandRunner executes shell commands using os/exec.
type RealCommandRunner struct{}

// NewRealCommandRunner creates a new RealCommandRunner.
func NewRealCommandRunner() *RealCommandRunner {
	return &RealCommandRunner{}
}

// Run executes a command in the specified working directory.
// Returns stdout on success, or an error with combined output on failure.
func (r *RealCommandRunner) Run(ctx context.Context, workDir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		combined := stdout.String() + stderr.String()
		return "", fmt.Errorf("%w: %s", err, combined)
	}

	return stdout.String(), nil
}
