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
}
