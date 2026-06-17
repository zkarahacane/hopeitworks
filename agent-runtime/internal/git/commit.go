package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommitAndPush stages every change in workDir, commits it under a bot identity,
// and pushes to origin/branch. It is a no-op (returns nil) when the working tree
// is clean. The origin remote already carries the auth token injected by Clone,
// so no additional credential setup is required for the push.
func CommitAndPush(ctx context.Context, workDir, branch, message string) error {
	// Configure a bot identity so `git commit` does not fail with "identity unknown".
	identity := [][2]string{
		{"user.email", "agent@hopeitworks.dev"},
		{"user.name", "hopeitworks agent"},
	}
	for _, kv := range identity {
		cmd := exec.CommandContext(ctx, "git", "config", kv[0], kv[1])
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config %s: %s: %w", kv[0], strings.TrimSpace(string(out)), err)
		}
	}

	// Stage everything the agent produced.
	addCmd := exec.CommandContext(ctx, "git", "add", "-A")
	addCmd.Dir = workDir
	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Nothing staged → the agent produced no changes; treat as a clean success.
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	diffCmd.Dir = workDir
	if err := diffCmd.Run(); err == nil {
		return nil
	}

	commitCmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
	commitCmd.Dir = workDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %s: %w", strings.TrimSpace(string(out)), err)
	}

	pushCmd := exec.CommandContext(ctx, "git", "push", "origin", branch)
	pushCmd.Dir = workDir
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push origin %s: %s: %w", branch, strings.TrimSpace(string(out)), err)
	}

	return nil
}
