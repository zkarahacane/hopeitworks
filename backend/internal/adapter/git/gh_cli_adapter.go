package git

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// branchNamePattern validates conventional branch naming: feat/{key}-{slug} or fix/{key}-{slug}.
var branchNamePattern = regexp.MustCompile(`^(feat|fix)/[a-zA-Z0-9]+-[a-zA-Z0-9-]+$`)

// Compile-time check that GhCliAdapter implements GitProvider.
var _ port.GitProvider = (*GhCliAdapter)(nil)

// GhCliAdapter implements GitProvider using the gh CLI and git commands
// via a CommandRunner for testability.
type GhCliAdapter struct {
	runner port.CommandRunner
	logger *slog.Logger
}

// NewGhCliAdapter creates a new GhCliAdapter with the given CommandRunner and logger.
func NewGhCliAdapter(runner port.CommandRunner, logger *slog.Logger) *GhCliAdapter {
	return &GhCliAdapter{
		runner: runner,
		logger: logger,
	}
}

// CloneRepo clones a repository to the target directory using gh repo clone.
func (a *GhCliAdapter) CloneRepo(ctx context.Context, repoURL string, targetDir string) error {
	a.logger.DebugContext(ctx, "cloning repository",
		"repo_url", repoURL,
		"target_dir", targetDir,
	)

	_, err := a.runner.Run(ctx, "", "gh", "repo", "clone", repoURL, targetDir)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to clone repository %s: %v", repoURL, err),
			map[string]any{"repo_url": repoURL, "target_dir": targetDir},
		)
	}
	return nil
}

// CreateBranch creates and checks out a new branch in the given working directory.
// The branch name must follow convention: feat/{story-key}-{slug} or fix/{story-key}-{slug}.
func (a *GhCliAdapter) CreateBranch(ctx context.Context, workDir string, branchName string) error {
	if !branchNamePattern.MatchString(branchName) {
		return errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("invalid branch name format: %s (expected feat/{story-key}-{slug} or fix/{story-key}-{slug})", branchName),
			map[string]any{"branch_name": branchName},
		)
	}

	a.logger.DebugContext(ctx, "creating branch",
		"work_dir", workDir,
		"branch_name", branchName,
	)

	_, err := a.runner.Run(ctx, workDir, "git", "checkout", "-b", branchName)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create branch %s: %v", branchName, err),
			map[string]any{"branch_name": branchName, "work_dir": workDir},
		)
	}
	return nil
}

// Push stages all changes, commits with the given message, and pushes to origin.
func (a *GhCliAdapter) Push(ctx context.Context, workDir string, commitMsg string) error {
	a.logger.DebugContext(ctx, "pushing changes",
		"work_dir", workDir,
	)

	// Stage all changes
	if _, err := a.runner.Run(ctx, workDir, "git", "add", "."); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to stage changes: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}

	// Commit with conventional message
	if _, err := a.runner.Run(ctx, workDir, "git", "commit", "-m", commitMsg); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to commit changes: %v", err),
			map[string]any{"work_dir": workDir, "commit_msg": commitMsg},
		)
	}

	// Push to origin
	if _, err := a.runner.Run(ctx, workDir, "git", "push", "-u", "origin", "HEAD"); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to push changes: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}

	return nil
}
