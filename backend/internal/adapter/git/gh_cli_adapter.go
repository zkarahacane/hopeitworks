package git

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

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

// CreatePR creates a pull request using gh CLI and returns the PR URL.
func (a *GhCliAdapter) CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (string, error) {
	a.logger.DebugContext(ctx, "creating pull request",
		"work_dir", workDir,
		"title", title,
		"base_branch", baseBranch,
	)

	stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "create", "--title", title, "--body", body, "--base", baseBranch)
	if err != nil {
		if strings.Contains(stdout, "authentication") || strings.Contains(stdout, "login required") {
			return "", errors.NewDomainError(
				errors.ErrCodeGitAuthFailed,
				fmt.Sprintf("GitHub authentication failed: %v", err),
				map[string]any{"work_dir": workDir, "output": stdout},
			)
		}

		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create PR: %v", err),
			map[string]any{"work_dir": workDir, "title": title, "base_branch": baseBranch, "output": stdout},
		)
	}

	// Parse PR URL from stdout (gh pr create returns URL on last line)
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	prURL := strings.TrimSpace(lines[len(lines)-1])

	if !strings.HasPrefix(prURL, "http") {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			"failed to parse PR URL from gh CLI output",
			map[string]any{"output": stdout},
		)
	}

	return prURL, nil
}

// MergePR squash-merges a pull request and deletes the source branch using gh CLI.
func (a *GhCliAdapter) MergePR(ctx context.Context, workDir string, prIdentifier string) error {
	a.logger.DebugContext(ctx, "merging pull request",
		"work_dir", workDir,
		"pr_identifier", prIdentifier,
	)

	stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "merge", prIdentifier, "--squash", "--delete-branch")
	if err != nil {
		if strings.Contains(stdout, "merge conflict") || strings.Contains(stdout, "conflicts") {
			return errors.NewDomainError(
				errors.ErrCodeMergeConflict,
				fmt.Sprintf("merge conflict detected for PR %s: %v", prIdentifier, err),
				map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
			)
		}

		if strings.Contains(stdout, "no pull requests found") || strings.Contains(stdout, "not found") {
			return errors.NewDomainError(
				errors.ErrCodePRNotFound,
				fmt.Sprintf("pull request not found: %s", prIdentifier),
				map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
			)
		}

		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to merge PR %s: %v", prIdentifier, err),
			map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
		)
	}

	return nil
}

// prCheck represents a single CI check result from gh pr checks --json output.
type prCheck struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Conclusion string `json:"conclusion"`
}

// GetCIStatus returns the CI check status for the current branch's PR using gh CLI.
func (a *GhCliAdapter) GetCIStatus(ctx context.Context, workDir string) (string, error) {
	a.logger.DebugContext(ctx, "getting CI status",
		"work_dir", workDir,
	)

	stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "checks", "--json", "name,state,conclusion")
	if err != nil {
		if strings.Contains(stdout, "no pull request") {
			return "", errors.NewDomainError(
				errors.ErrCodePRNotFound,
				"no pull request found for current branch",
				map[string]any{"work_dir": workDir, "output": stdout},
			)
		}

		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get CI status: %v", err),
			map[string]any{"work_dir": workDir, "output": stdout},
		)
	}

	var checks []prCheck
	if err := json.Unmarshal([]byte(stdout), &checks); err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse CI check JSON: %v", err),
			map[string]any{"output": stdout},
		)
	}

	if len(checks) == 0 {
		return "no_checks", nil
	}

	hasPending := false
	for _, check := range checks {
		if check.State == "pending" || check.State == "queued" || check.State == "in_progress" {
			hasPending = true
			continue
		}

		if check.Conclusion == "failure" || check.Conclusion == "timed_out" || check.Conclusion == "action_required" {
			return "fail", nil
		}
	}

	if hasPending {
		return "pending", nil
	}

	return "pass", nil
}
