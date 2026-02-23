package git

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// ciStatusPending and ciStatusFail are package-level aliases kept for readability
// inside the adapter methods; the canonical values are the exported CIStatus* constants.
const (
	ciStatusPending = CIStatusPending
	ciStatusFail    = CIStatusFail

	// conclusionFailure is the GitHub check conclusion for a failed check.
	conclusionFailure = "failure"
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
		return CIStatusNoChecks, nil
	}

	hasPending := false
	for _, check := range checks {
		if check.State == ciStatusPending || check.State == "queued" || check.State == "in_progress" {
			hasPending = true
			continue
		}

		if check.Conclusion == conclusionFailure || check.Conclusion == "timed_out" || check.Conclusion == "action_required" {
			return ciStatusFail, nil
		}
	}

	if hasPending {
		return ciStatusPending, nil
	}

	return CIStatusPass, nil
}

// GetPRDiff returns the diff content for the given pull request URL using gh CLI.
func (a *GhCliAdapter) GetPRDiff(ctx context.Context, prURL string) (string, error) {
	a.logger.DebugContext(ctx, "fetching PR diff",
		"pr_url", prURL,
	)

	out, err := a.runner.Run(ctx, "", "gh", "pr", "diff", prURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get PR diff for %s: %v", prURL, err),
			map[string]any{"pr_url": prURL},
		)
	}
	return out, nil
}

// githubRepoPattern matches GitHub repo URLs like:
// https://github.com/owner/repo
// https://github.com/owner/repo.git
var githubRepoPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)

// githubPRPattern matches GitHub PR URLs like:
// https://github.com/owner/repo/pull/123
var githubPRPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+)/pull/(\d+)$`)

// parseGitHubOwnerRepo extracts owner and repo from a GitHub repository URL.
func parseGitHubOwnerRepo(repoURL string) (owner, repo string, err error) {
	matches := githubRepoPattern.FindStringSubmatch(repoURL)
	if matches == nil {
		return "", "", fmt.Errorf("cannot parse GitHub repo URL: %s", repoURL)
	}
	return matches[1], matches[2], nil
}

// parseGitHubPR extracts owner, repo, and PR number from a GitHub PR URL.
func parseGitHubPR(prURL string) (owner, repo string, number int, err error) {
	matches := githubPRPattern.FindStringSubmatch(prURL)
	if matches == nil {
		return "", "", 0, fmt.Errorf("cannot parse GitHub PR URL: %s", prURL)
	}
	n, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number in URL %s: %w", prURL, err)
	}
	return matches[1], matches[2], n, nil
}

// CreateRemoteBranch creates a new branch on the remote repository via the GitHub API.
func (a *GhCliAdapter) CreateRemoteBranch(ctx context.Context, repoURL string, branchName string, baseBranch string) error {
	owner, repo, err := parseGitHubOwnerRepo(repoURL)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse repo URL: %v", err),
			map[string]any{"repo_url": repoURL},
		)
	}

	a.logger.DebugContext(ctx, "creating remote branch via API",
		"owner", owner, "repo", repo,
		"branch_name", branchName, "base_branch", baseBranch,
	)

	// Get the SHA of the base branch
	apiPath := fmt.Sprintf("repos/%s/%s/git/ref/heads/%s", owner, repo, baseBranch)
	sha, err := a.runner.Run(ctx, ".", "gh", "api", apiPath, "--jq", ".object.sha")
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get base branch %q SHA: %v", baseBranch, err),
			map[string]any{"owner": owner, "repo": repo, "base_branch": baseBranch},
		)
	}

	// Create the new branch ref
	sha = strings.TrimSpace(sha)
	refPath := fmt.Sprintf("repos/%s/%s/git/refs", owner, repo)
	ref := fmt.Sprintf("refs/heads/%s", branchName)
	_, err = a.runner.Run(ctx, ".", "gh", "api", refPath,
		"-f", fmt.Sprintf("ref=%s", ref),
		"-f", fmt.Sprintf("sha=%s", sha),
	)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create remote branch %q: %v", branchName, err),
			map[string]any{"owner": owner, "repo": repo, "branch_name": branchName, "sha": sha},
		)
	}

	return nil
}

// CreateRemotePR creates a pull request via the GitHub API and returns the PR URL.
func (a *GhCliAdapter) CreateRemotePR(ctx context.Context, repoURL string, title string, body string, headBranch string, baseBranch string) (string, error) {
	owner, repo, err := parseGitHubOwnerRepo(repoURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse repo URL: %v", err),
			map[string]any{"repo_url": repoURL},
		)
	}

	a.logger.DebugContext(ctx, "creating remote PR via API",
		"owner", owner, "repo", repo,
		"head", headBranch, "base", baseBranch,
	)

	apiPath := fmt.Sprintf("repos/%s/%s/pulls", owner, repo)
	stdout, err := a.runner.Run(ctx, ".", "gh", "api", apiPath,
		"-f", fmt.Sprintf("title=%s", title),
		"-f", fmt.Sprintf("body=%s", body),
		"-f", fmt.Sprintf("head=%s", headBranch),
		"-f", fmt.Sprintf("base=%s", baseBranch),
		"--jq", ".html_url",
	)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create remote PR: %v", err),
			map[string]any{"owner": owner, "repo": repo, "head": headBranch, "base": baseBranch},
		)
	}

	prURL := strings.TrimSpace(stdout)
	if !strings.HasPrefix(prURL, "http") {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			"failed to parse PR URL from GitHub API response",
			map[string]any{"output": stdout},
		)
	}

	return prURL, nil
}

// ghCheckRun represents a single check run from the GitHub API.
type ghCheckRun struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// ghCheckRunsResponse is the API response for listing check runs.
type ghCheckRunsResponse struct {
	TotalCount int          `json:"total_count"`
	CheckRuns  []ghCheckRun `json:"check_runs"`
}

// GetRemoteCIStatus returns CI status for a PR identified by its URL via the GitHub API.
func (a *GhCliAdapter) GetRemoteCIStatus(ctx context.Context, prURL string) (string, error) {
	owner, repo, prNumber, err := parseGitHubPR(prURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse PR URL: %v", err),
			map[string]any{"pr_url": prURL},
		)
	}

	a.logger.DebugContext(ctx, "getting remote CI status via API",
		"owner", owner, "repo", repo, "pr_number", prNumber,
	)

	// Get the PR head SHA
	prPath := fmt.Sprintf("repos/%s/%s/pulls/%d", owner, repo, prNumber)
	sha, err := a.runner.Run(ctx, ".", "gh", "api", prPath, "--jq", ".head.sha")
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get PR head SHA: %v", err),
			map[string]any{"pr_url": prURL},
		)
	}
	sha = strings.TrimSpace(sha)

	// Get check runs for that SHA
	checksPath := fmt.Sprintf("repos/%s/%s/commits/%s/check-runs", owner, repo, sha)
	stdout, err := a.runner.Run(ctx, ".", "gh", "api", checksPath)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get check runs: %v", err),
			map[string]any{"pr_url": prURL, "sha": sha},
		)
	}

	var checksResp ghCheckRunsResponse
	if err := json.Unmarshal([]byte(stdout), &checksResp); err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse check runs JSON: %v", err),
			map[string]any{"output": stdout},
		)
	}

	if checksResp.TotalCount == 0 {
		return CIStatusNoChecks, nil
	}

	hasPending := false
	for _, check := range checksResp.CheckRuns {
		if check.Status == "queued" || check.Status == "in_progress" {
			hasPending = true
			continue
		}
		if check.Conclusion == conclusionFailure || check.Conclusion == "timed_out" || check.Conclusion == "action_required" {
			return CIStatusFail, nil
		}
	}

	if hasPending {
		return CIStatusPending, nil
	}

	return CIStatusPass, nil
}
