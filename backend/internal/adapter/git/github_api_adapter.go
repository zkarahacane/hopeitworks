package git

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// checkStatusQueued and checkStatusInProgress are the GitHub check-run statuses
// that map to a pending CI state. conclusionTimedOut and conclusionActionRequired
// are the non-failure conclusions that we still treat as a CI failure, matching
// the gh-CLI adapter behaviour exactly.
const (
	checkStatusQueued        = "queued"
	checkStatusInProgress    = "in_progress"
	conclusionTimedOut       = "timed_out"
	conclusionActionRequired = "action_required"

	mergeMethodSquash = "squash"
)

// Compile-time check that GitHubAPIAdapter implements GitProvider.
var _ port.GitProvider = (*GitHubAPIAdapter)(nil)

// GitHubAPIAdapter implements GitProvider using the GitHub REST API via
// google/go-github and a personal access token. It removes the dependency on
// the `gh` CLI for all API operations (branch/PR/CI/diff). Local git operations
// (clone/branch/commit/push) still shell out to plain `git` via the
// CommandRunner — `git` is available everywhere, unlike `gh`.
//
//nolint:revive // "GitHub" prefix is the provider name (mirrors GhCliAdapter/GiteaAPIAdapter), not package stutter.
type GitHubAPIAdapter struct {
	client *github.Client
	token  string
	runner port.CommandRunner
	logger *slog.Logger
}

// NewGitHubAPIAdapter creates a GitHubAPIAdapter.
//
// baseURL is optional: when non-empty it overrides the GitHub API base URL
// (used by tests pointing at an httptest.Server). When empty, the default
// public GitHub API endpoint is used.
// token is the GitHub personal access token used for both API auth and
// authenticated git clone (injected into the clone URL).
func NewGitHubAPIAdapter(baseURL, token string, runner port.CommandRunner, logger *slog.Logger) *GitHubAPIAdapter {
	client := github.NewClient(nil).WithAuthToken(token)
	if baseURL != "" {
		normalized := baseURL
		if !strings.HasSuffix(normalized, "/") {
			normalized += "/"
		}
		if parsed, err := url.Parse(normalized); err == nil {
			client.BaseURL = parsed
		}
	}
	return &GitHubAPIAdapter{
		client: client,
		token:  token,
		runner: runner,
		logger: logger,
	}
}

// CloneRepo clones a repository using plain `git` with the token injected into
// the URL (https://{token}@host/...). It does NOT use `gh repo clone`.
func (a *GitHubAPIAdapter) CloneRepo(ctx context.Context, repoURL string, targetDir string) error {
	a.logger.DebugContext(ctx, "cloning repository via git",
		"repo_url", repoURL,
		"target_dir", targetDir,
	)

	cloneURL := repoURL
	if a.token != "" {
		injected, err := injectTokenInURL(repoURL, a.token)
		if err != nil {
			return errors.NewDomainError(
				errors.ErrCodeGitOperationFailed,
				fmt.Sprintf("failed to build clone URL: %v", err),
				map[string]any{"repo_url": repoURL},
			)
		}
		cloneURL = injected
	}

	if _, err := a.runner.Run(ctx, "", "git", "clone", cloneURL, targetDir); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to clone repository %s: %v", repoURL, err),
			map[string]any{"repo_url": repoURL, "target_dir": targetDir},
		)
	}
	return nil
}

// CreateBranch creates and checks out a new branch in the given working
// directory using plain `git`.
func (a *GitHubAPIAdapter) CreateBranch(ctx context.Context, workDir string, branchName string) error {
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

	if _, err := a.runner.Run(ctx, workDir, "git", "checkout", "-b", branchName); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create branch %s: %v", branchName, err),
			map[string]any{"branch_name": branchName, "work_dir": workDir},
		)
	}
	return nil
}

// Push stages all changes, commits with the given message, and pushes to origin
// using plain `git`.
func (a *GitHubAPIAdapter) Push(ctx context.Context, workDir string, commitMsg string) error {
	a.logger.DebugContext(ctx, "pushing changes",
		"work_dir", workDir,
	)

	if _, err := a.runner.Run(ctx, workDir, "git", "add", "."); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to stage changes: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}

	if _, err := a.runner.Run(ctx, workDir, "git", "commit", "-m", commitMsg); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to commit changes: %v", err),
			map[string]any{"work_dir": workDir, "commit_msg": commitMsg},
		)
	}

	if _, err := a.runner.Run(ctx, workDir, "git", "push", "-u", "origin", "HEAD"); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to push changes: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}

	return nil
}

// CreatePR creates a pull request via the GitHub API and returns the PR URL.
// owner/repo and the head branch are derived from the workDir git remote.
func (a *GitHubAPIAdapter) CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (string, error) {
	remoteURL, err := a.remoteURL(ctx, workDir)
	if err != nil {
		return "", err
	}

	headBranch, err := a.currentBranch(ctx, workDir)
	if err != nil {
		return "", err
	}

	return a.CreateRemotePR(ctx, remoteURL, title, body, headBranch, baseBranch)
}

// MergePR squash-merges a pull request and deletes the source branch via the
// GitHub API. prIdentifier is a PR number or a full PR URL; owner/repo are
// derived from the workDir git remote when only a number is supplied.
func (a *GitHubAPIAdapter) MergePR(ctx context.Context, workDir string, prIdentifier string) error {
	owner, repo, number, err := a.resolvePR(ctx, workDir, prIdentifier)
	if err != nil {
		return err
	}

	a.logger.DebugContext(ctx, "merging pull request via API",
		"owner", owner, "repo", repo, "pr_number", number,
	)

	_, _, err = a.client.PullRequests.Merge(ctx, owner, repo, number, "", &github.PullRequestOptions{
		MergeMethod: mergeMethodSquash,
	})
	if err != nil {
		return classifyMergeError(prIdentifier, owner, repo, number, err)
	}

	// Delete the source branch after a successful squash merge.
	pr, _, getErr := a.client.PullRequests.Get(ctx, owner, repo, number)
	if getErr != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("merged PR %d but failed to resolve head branch for deletion: %v", number, getErr),
			map[string]any{"owner": owner, "repo": repo, "pr_number": number},
		)
	}
	if pr.GetHead() != nil && pr.GetHead().GetRef() != "" {
		ref := fmt.Sprintf("heads/%s", pr.GetHead().GetRef())
		if _, delErr := a.client.Git.DeleteRef(ctx, owner, repo, ref); delErr != nil {
			return errors.NewDomainError(
				errors.ErrCodeGitOperationFailed,
				fmt.Sprintf("merged PR %d but failed to delete branch %q: %v", number, pr.GetHead().GetRef(), delErr),
				map[string]any{"owner": owner, "repo": repo, "branch": pr.GetHead().GetRef()},
			)
		}
	}

	return nil
}

// GetCIStatus returns the CI check status for the current branch's PR via the
// GitHub API. owner/repo come from the workDir remote; the open PR for the
// current branch is located, then its head SHA's check runs are evaluated.
func (a *GitHubAPIAdapter) GetCIStatus(ctx context.Context, workDir string) (string, error) {
	remoteURL, err := a.remoteURL(ctx, workDir)
	if err != nil {
		return "", err
	}
	owner, repo, err := parseGitHubOwnerRepo(remoteURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse repo URL: %v", err),
			map[string]any{"repo_url": remoteURL},
		)
	}

	headBranch, err := a.currentBranch(ctx, workDir)
	if err != nil {
		return "", err
	}

	prs, _, err := a.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		Head:  fmt.Sprintf("%s:%s", owner, headBranch),
		State: "open",
	})
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to list PRs for branch %q: %v", headBranch, err),
			map[string]any{"owner": owner, "repo": repo, "head": headBranch},
		)
	}
	if len(prs) == 0 {
		return "", errors.NewDomainError(
			errors.ErrCodePRNotFound,
			"no pull request found for current branch",
			map[string]any{"owner": owner, "repo": repo, "head": headBranch},
		)
	}

	sha := prs[0].GetHead().GetSHA()
	return a.ciStatusForSHA(ctx, owner, repo, sha)
}

// GetPRDiff returns the unified diff content for the given pull request URL via
// the GitHub API (application/vnd.github.v3.diff media type).
func (a *GitHubAPIAdapter) GetPRDiff(ctx context.Context, prURL string) (string, error) {
	owner, repo, number, err := parseGitHubPR(prURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse PR URL: %v", err),
			map[string]any{"pr_url": prURL},
		)
	}

	a.logger.DebugContext(ctx, "fetching PR diff via API",
		"owner", owner, "repo", repo, "pr_number", number,
	)

	diff, _, err := a.client.PullRequests.GetRaw(ctx, owner, repo, number, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get PR diff for %s: %v", prURL, err),
			map[string]any{"pr_url": prURL},
		)
	}
	return diff, nil
}

// CreateRemoteBranch creates a new branch on the remote repository via the
// GitHub API. It resolves the base branch SHA then creates refs/heads/<branch>.
func (a *GitHubAPIAdapter) CreateRemoteBranch(ctx context.Context, repoURL string, branchName string, baseBranch string) error {
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

	baseRef, _, err := a.client.Git.GetRef(ctx, owner, repo, fmt.Sprintf("heads/%s", baseBranch))
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get base branch %q SHA: %v", baseBranch, err),
			map[string]any{"owner": owner, "repo": repo, "base_branch": baseBranch},
		)
	}
	sha := baseRef.GetObject().GetSHA()

	newRef := &github.Reference{
		Ref:    github.String(fmt.Sprintf("refs/heads/%s", branchName)),
		Object: &github.GitObject{SHA: github.String(sha)},
	}
	if _, _, err := a.client.Git.CreateRef(ctx, owner, repo, newRef); err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create remote branch %q: %v", branchName, err),
			map[string]any{"owner": owner, "repo": repo, "branch_name": branchName, "sha": sha},
		)
	}

	return nil
}

// CreateRemotePR creates a pull request via the GitHub API and returns its URL.
func (a *GitHubAPIAdapter) CreateRemotePR(ctx context.Context, repoURL string, title string, body string, headBranch string, baseBranch string) (string, error) {
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

	pr, _, err := a.client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(title),
		Body:  github.String(body),
		Head:  github.String(headBranch),
		Base:  github.String(baseBranch),
	})
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create remote PR: %v", err),
			map[string]any{"owner": owner, "repo": repo, "head": headBranch, "base": baseBranch},
		)
	}

	prURL := pr.GetHTMLURL()
	if !strings.HasPrefix(prURL, "http") {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			"failed to parse PR URL from GitHub API response",
			map[string]any{"owner": owner, "repo": repo},
		)
	}

	return prURL, nil
}

// GetRemoteCIStatus returns CI status for a PR identified by its URL via the
// GitHub API. Mapping matches GhCliAdapter.GetRemoteCIStatus exactly.
func (a *GitHubAPIAdapter) GetRemoteCIStatus(ctx context.Context, prURL string) (string, error) {
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

	pr, _, err := a.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get PR head SHA: %v", err),
			map[string]any{"pr_url": prURL},
		)
	}

	return a.ciStatusForSHA(ctx, owner, repo, pr.GetHead().GetSHA())
}

// ciStatusForSHA lists the check runs for a commit SHA and maps them to one of
// "pass"|"fail"|"pending"|"no_checks", replicating the gh-CLI adapter logic.
func (a *GitHubAPIAdapter) ciStatusForSHA(ctx context.Context, owner, repo, sha string) (string, error) {
	results, _, err := a.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, nil)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get check runs: %v", err),
			map[string]any{"owner": owner, "repo": repo, "sha": sha},
		)
	}

	if results.GetTotal() == 0 {
		return CIStatusNoChecks, nil
	}

	hasPending := false
	for _, check := range results.CheckRuns {
		status := check.GetStatus()
		if status == checkStatusQueued || status == checkStatusInProgress {
			hasPending = true
			continue
		}
		conclusion := check.GetConclusion()
		if conclusion == conclusionFailure || conclusion == conclusionTimedOut || conclusion == conclusionActionRequired {
			return CIStatusFail, nil
		}
	}

	if hasPending {
		return CIStatusPending, nil
	}

	return CIStatusPass, nil
}

// remoteURL returns the origin remote URL of the given working directory,
// stripping any embedded credentials so owner/repo parsing is stable.
func (a *GitHubAPIAdapter) remoteURL(ctx context.Context, workDir string) (string, error) {
	out, err := a.runner.Run(ctx, workDir, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to resolve origin remote: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}
	return stripCredentials(strings.TrimSpace(out)), nil
}

// currentBranch returns the current branch name of the given working directory.
func (a *GitHubAPIAdapter) currentBranch(ctx context.Context, workDir string) (string, error) {
	out, err := a.runner.Run(ctx, workDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to resolve current branch: %v", err),
			map[string]any{"work_dir": workDir},
		)
	}
	return strings.TrimSpace(out), nil
}

// resolvePR resolves owner/repo/number from a PR identifier (URL or number).
// When only a number is given, owner/repo come from the workDir git remote.
func (a *GitHubAPIAdapter) resolvePR(ctx context.Context, workDir, prIdentifier string) (owner, repo string, number int, err error) {
	if strings.HasPrefix(prIdentifier, "http") {
		o, r, n, perr := parseGitHubPR(prIdentifier)
		if perr != nil {
			return "", "", 0, errors.NewDomainError(
				errors.ErrCodeInvalidInput,
				fmt.Sprintf("failed to parse PR URL: %v", perr),
				map[string]any{"pr_identifier": prIdentifier},
			)
		}
		return o, r, n, nil
	}

	n, perr := parsePRNumber(prIdentifier)
	if perr != nil {
		return "", "", 0, errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("invalid PR identifier %q: %v", prIdentifier, perr),
			map[string]any{"pr_identifier": prIdentifier},
		)
	}

	remoteURL, rerr := a.remoteURL(ctx, workDir)
	if rerr != nil {
		return "", "", 0, rerr
	}
	o, r, oerr := parseGitHubOwnerRepo(remoteURL)
	if oerr != nil {
		return "", "", 0, errors.NewDomainError(
			errors.ErrCodeInvalidInput,
			fmt.Sprintf("failed to parse repo URL: %v", oerr),
			map[string]any{"repo_url": remoteURL},
		)
	}
	return o, r, n, nil
}

// parsePRNumber parses a bare PR number (optionally prefixed with '#').
func parsePRNumber(s string) (int, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(s), "#")
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("not a PR number: %q", s)
	}
	return n, nil
}

// classifyMergeError maps a merge API error to the closest domain error,
// mirroring the gh-CLI adapter's conflict / not-found classification.
func classifyMergeError(prIdentifier, owner, repo string, number int, err error) error {
	msg := strings.ToLower(err.Error())
	details := map[string]any{"pr_identifier": prIdentifier, "owner": owner, "repo": repo, "pr_number": number}

	if strings.Contains(msg, "merge conflict") || strings.Contains(msg, "conflict") {
		return errors.NewDomainError(
			errors.ErrCodeMergeConflict,
			fmt.Sprintf("merge conflict detected for PR %s: %v", prIdentifier, err),
			details,
		)
	}
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no pull request") {
		return errors.NewDomainError(
			errors.ErrCodePRNotFound,
			fmt.Sprintf("pull request not found: %s", prIdentifier),
			details,
		)
	}
	return errors.NewDomainError(
		errors.ErrCodeGitOperationFailed,
		fmt.Sprintf("failed to merge PR %s: %v", prIdentifier, err),
		details,
	)
}
