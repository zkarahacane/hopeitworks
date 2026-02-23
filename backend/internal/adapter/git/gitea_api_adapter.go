package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/zakari/hopeitworks/backend/internal/domain/port"
	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

// Compile-time check that GiteaAPIAdapter implements GitProvider.
var _ port.GitProvider = (*GiteaAPIAdapter)(nil)

// GiteaAPIAdapter implements GitProvider using the Gitea API for PR/CI operations
// and git CLI (via CommandRunner) for local git operations.
type GiteaAPIAdapter struct {
	baseURL    string
	token      string
	runner     port.CommandRunner
	httpClient *http.Client
	logger     *slog.Logger
}

// NewGiteaAPIAdapter creates a new GiteaAPIAdapter.
func NewGiteaAPIAdapter(baseURL, token string, runner port.CommandRunner, logger *slog.Logger) *GiteaAPIAdapter {
	return &GiteaAPIAdapter{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		token:      token,
		runner:     runner,
		httpClient: &http.Client{},
		logger:     logger,
	}
}

// CloneRepo clones a repository with the token injected in the URL.
func (a *GiteaAPIAdapter) CloneRepo(ctx context.Context, repoURL string, targetDir string) error {
	a.logger.DebugContext(ctx, "cloning repository via gitea",
		"repo_url", repoURL,
		"target_dir", targetDir,
	)

	cloneURL, err := injectTokenInURL(repoURL, a.token)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to build clone URL: %v", err),
			map[string]any{"repo_url": repoURL},
		)
	}

	_, err = a.runner.Run(ctx, "", "git", "clone", cloneURL, targetDir)
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
func (a *GiteaAPIAdapter) CreateBranch(ctx context.Context, workDir string, branchName string) error {
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
func (a *GiteaAPIAdapter) Push(ctx context.Context, workDir string, commitMsg string) error {
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

// giteaPRRequest is the JSON body for creating a Gitea pull request.
type giteaPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// giteaPRResponse represents the relevant fields of a Gitea pull request response.
type giteaPRResponse struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
}

// CreatePR creates a pull request via the Gitea API and returns the PR URL.
func (a *GiteaAPIAdapter) CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (string, error) {
	a.logger.DebugContext(ctx, "creating pull request via gitea API",
		"work_dir", workDir,
		"title", title,
		"base_branch", baseBranch,
	)

	// Get the remote URL to determine owner/repo
	remoteURL, err := a.getRemoteURL(ctx, workDir)
	if err != nil {
		return "", fmt.Errorf("get remote URL: %w", err)
	}

	owner, repo, err := parseGiteaRepoOwnerAndName(remoteURL)
	if err != nil {
		return "", fmt.Errorf("parse repo owner/name: %w", err)
	}

	headBranch, err := getCurrentBranch(ctx, a.runner, workDir)
	if err != nil {
		return "", fmt.Errorf("get current branch: %w", err)
	}

	reqBody := giteaPRRequest{
		Title: title,
		Body:  body,
		Head:  headBranch,
		Base:  baseBranch,
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls", a.baseURL, owner, repo)
	respBody, err := a.doJSON(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to create PR via Gitea API: %v", err),
			map[string]any{"owner": owner, "repo": repo, "head": headBranch, "base": baseBranch},
		)
	}

	var pr giteaPRResponse
	if err := json.Unmarshal(respBody, &pr); err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse PR response: %v", err),
			map[string]any{"response": string(respBody)},
		)
	}

	prURL := pr.HTMLURL
	if prURL == "" {
		// Fallback: construct URL manually
		prURL = fmt.Sprintf("%s/%s/%s/pulls/%d", a.baseURL, owner, repo, pr.Number)
	}

	return prURL, nil
}

// giteaMergeRequest is the JSON body for merging a Gitea pull request.
type giteaMergeRequest struct {
	Do                     string `json:"Do"`
	DeleteBranchAfterMerge bool   `json:"delete_branch_after_merge"`
}

// MergePR squash-merges a pull request and deletes the source branch via Gitea API.
func (a *GiteaAPIAdapter) MergePR(ctx context.Context, _ string, prIdentifier string) error {
	a.logger.DebugContext(ctx, "merging pull request via gitea API",
		"pr_identifier", prIdentifier,
	)

	owner, repo, index, err := parseGiteaPRIndex(prIdentifier)
	if err != nil {
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse PR identifier %s: %v", prIdentifier, err),
			map[string]any{"pr_identifier": prIdentifier},
		)
	}

	reqBody := giteaMergeRequest{
		Do:                     "squash",
		DeleteBranchAfterMerge: true,
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge", a.baseURL, owner, repo, index)
	_, err = a.doJSON(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return errors.NewDomainError(
				errors.ErrCodePRNotFound,
				fmt.Sprintf("pull request not found: %s", prIdentifier),
				map[string]any{"pr_identifier": prIdentifier},
			)
		}
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "conflict") {
			return errors.NewDomainError(
				errors.ErrCodeMergeConflict,
				fmt.Sprintf("merge conflict detected for PR %s: %v", prIdentifier, err),
				map[string]any{"pr_identifier": prIdentifier},
			)
		}
		return errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to merge PR %s: %v", prIdentifier, err),
			map[string]any{"pr_identifier": prIdentifier},
		)
	}

	return nil
}

// giteaCommitStatus represents a single commit status from the Gitea API.
type giteaCommitStatus struct {
	Status string `json:"status"`
	State  string `json:"state"`
}

// GetCIStatus returns the CI check status for the current branch using Gitea commit statuses API.
func (a *GiteaAPIAdapter) GetCIStatus(ctx context.Context, workDir string) (string, error) {
	a.logger.DebugContext(ctx, "getting CI status via gitea API",
		"work_dir", workDir,
	)

	remoteURL, err := a.getRemoteURL(ctx, workDir)
	if err != nil {
		return "", fmt.Errorf("get remote URL: %w", err)
	}

	owner, repo, err := parseGiteaRepoOwnerAndName(remoteURL)
	if err != nil {
		return "", fmt.Errorf("parse repo owner/name: %w", err)
	}

	sha, err := getHeadSHA(ctx, a.runner, workDir)
	if err != nil {
		return "", fmt.Errorf("get HEAD SHA: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/commits/%s/statuses", a.baseURL, owner, repo, sha)
	respBody, err := a.doGet(ctx, endpoint, "application/json")
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get CI status: %v", err),
			map[string]any{"owner": owner, "repo": repo, "sha": sha},
		)
	}

	var statuses []giteaCommitStatus
	if err := json.Unmarshal(respBody, &statuses); err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse commit statuses: %v", err),
			map[string]any{"response": string(respBody)},
		)
	}

	if len(statuses) == 0 {
		return CIStatusNoChecks, nil
	}

	hasPending := false
	for _, s := range statuses {
		state := s.Status
		if state == "" {
			state = s.State
		}
		switch state {
		case "failure", "error":
			return CIStatusFail, nil
		case "pending", "running":
			hasPending = true
		}
	}

	if hasPending {
		return CIStatusPending, nil
	}

	return CIStatusPass, nil
}

// GetPRDiff returns the diff content for the given pull request URL via the Gitea API.
func (a *GiteaAPIAdapter) GetPRDiff(ctx context.Context, prURL string) (string, error) {
	a.logger.DebugContext(ctx, "fetching PR diff via gitea API",
		"pr_url", prURL,
	)

	owner, repo, index, err := parseGiteaPRIndex(prURL)
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to parse PR URL %s: %v", prURL, err),
			map[string]any{"pr_url": prURL},
		)
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d.diff", a.baseURL, owner, repo, index)
	respBody, err := a.doGet(ctx, endpoint, "text/plain")
	if err != nil {
		return "", errors.NewDomainError(
			errors.ErrCodeGitOperationFailed,
			fmt.Sprintf("failed to get PR diff for %s: %v", prURL, err),
			map[string]any{"pr_url": prURL},
		)
	}

	return string(respBody), nil
}

// --- Helpers ---

// doJSON makes an HTTP request with a JSON body and returns the response body.
func (a *GiteaAPIAdapter) doJSON(ctx context.Context, method, endpoint string, body any) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+a.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// doGet makes an HTTP GET request and returns the response body.
func (a *GiteaAPIAdapter) doGet(ctx context.Context, endpoint, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+a.token)
	req.Header.Set("Accept", accept)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// getRemoteURL retrieves the origin remote URL from the git repo.
func (a *GiteaAPIAdapter) getRemoteURL(ctx context.Context, workDir string) (string, error) {
	out, err := a.runner.Run(ctx, workDir, "git", "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("get remote URL: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// giteaRepoPattern matches Gitea repo URLs like:
// https://gitea.example.com/owner/repo
// https://gitea.example.com/owner/repo.git
// http://host:3030/owner/repo.git
var giteaRepoPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+?)(?:\.git)?$`)

// parseGiteaRepoOwnerAndName extracts the owner and repo name from a Gitea repo URL.
func parseGiteaRepoOwnerAndName(repoURL string) (owner, repo string, err error) {
	// Strip any embedded credentials (token@host -> host)
	cleanURL := stripCredentials(repoURL)

	matches := giteaRepoPattern.FindStringSubmatch(cleanURL)
	if matches == nil {
		return "", "", fmt.Errorf("cannot parse Gitea repo URL: %s", repoURL)
	}
	return matches[1], matches[2], nil
}

// giteaPRPattern matches Gitea PR URLs like:
// https://gitea.example.com/owner/repo/pulls/123
var giteaPRPattern = regexp.MustCompile(`^https?://[^/]+/([^/]+)/([^/]+)/pulls/(\d+)`)

// parseGiteaPRIndex extracts owner, repo, and PR index from a Gitea PR URL.
func parseGiteaPRIndex(prURL string) (owner, repo string, index int, err error) {
	matches := giteaPRPattern.FindStringSubmatch(prURL)
	if matches == nil {
		return "", "", 0, fmt.Errorf("cannot parse Gitea PR URL: %s", prURL)
	}
	idx, err := strconv.Atoi(matches[3])
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR index in URL %s: %w", prURL, err)
	}
	return matches[1], matches[2], idx, nil
}

// getHeadSHA returns the HEAD commit SHA in the given working directory.
func getHeadSHA(ctx context.Context, runner port.CommandRunner, workDir string) (string, error) {
	out, err := runner.Run(ctx, workDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// getCurrentBranch returns the current branch name in the given working directory.
func getCurrentBranch(ctx context.Context, runner port.CommandRunner, workDir string) (string, error) {
	out, err := runner.Run(ctx, workDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("git rev-parse --abbrev-ref HEAD: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// extractBaseURL extracts the scheme + host from a URL string.
func extractBaseURL(repoURL string) string {
	if repoURL == "" {
		return ""
	}
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
}

// injectTokenInURL injects a token into a URL for authenticated git clone.
// Produces: https://{token}@{host}/{path}
func injectTokenInURL(repoURL, token string) (string, error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parse URL %s: %w", repoURL, err)
	}
	parsed.User = url.User(token)
	return parsed.String(), nil
}

// stripCredentials removes user info from a URL for safe pattern matching.
func stripCredentials(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parsed.User = nil
	return parsed.String()
}
