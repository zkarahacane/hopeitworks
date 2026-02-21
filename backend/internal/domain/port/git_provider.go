package port

import "context"

// GitProvider abstracts Git repository operations.
// Implementations must use conventional branch naming and commit messages.
type GitProvider interface {
	// CloneRepo clones a repository to the target directory.
	// repoURL format: "owner/repo" or full HTTPS URL.
	CloneRepo(ctx context.Context, repoURL string, targetDir string) error

	// CreateBranch creates and checks out a new branch in the given working directory.
	// branchName must follow convention: feat/{story-key}-{slug} or fix/{story-key}-{slug}.
	CreateBranch(ctx context.Context, workDir string, branchName string) error

	// Push stages all changes, commits with the given message, and pushes to origin.
	// commitMsg must follow conventional commit format: type(scope): message.
	Push(ctx context.Context, workDir string, commitMsg string) error

	// CreatePR creates a pull request and returns the PR URL.
	// title: PR title (should follow conventional commit format for squash merge).
	// body: PR description/body.
	// baseBranch: target branch (typically "main" or "develop").
	// Returns: PR URL (e.g., "https://github.com/owner/repo/pull/123").
	CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (prURL string, err error)

	// MergePR squash-merges a pull request and deletes the source branch.
	// prIdentifier: PR number (e.g., "123") or PR URL.
	// Performs squash merge to maintain clean commit history.
	MergePR(ctx context.Context, workDir string, prIdentifier string) error

	// GetCIStatus returns the CI check status for the current branch's PR.
	// Returns: "pass" (all checks successful), "fail" (any check failed),
	//          "pending" (checks running), "no_checks" (no CI configured).
	GetCIStatus(ctx context.Context, workDir string) (status string, err error)

	// GetPRDiff returns the diff content for the given pull request URL.
	// prURL: full PR URL (e.g., "https://github.com/owner/repo/pull/123").
	GetPRDiff(ctx context.Context, prURL string) (string, error)
}
