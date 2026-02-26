// Package git provides repository cloning and branch management for the agent runtime.
package git

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// Clone clones the repository at repoURL into workDir, checking out the specified branch.
// If the branch does not exist on the remote, it clones the default branch and creates
// a new local branch with the given name.
// gitProvider determines how the token is injected into the URL:
//   - "github": https://{token}@host/...
//   - "gitea":  https://oauth2:{token}@host/...
func Clone(ctx context.Context, repoURL, branch, gitToken, gitProvider, workDir string) error {
	authedURL, err := injectToken(repoURL, gitToken, gitProvider)
	if err != nil {
		return fmt.Errorf("inject git token: %w", err)
	}

	// Try cloning the specific branch first (shallow clone for speed)
	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, authedURL, workDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		// Branch might not exist on remote — clone default branch and create it
		if !strings.Contains(string(output), "not found") && !strings.Contains(string(output), "Could not find") && !strings.Contains(string(output), "Remote branch") {
			return fmt.Errorf("git clone failed: %s: %w", strings.TrimSpace(string(output)), err)
		}

		// Fallback: clone default branch
		fallbackCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", authedURL, workDir)
		if fbOutput, fbErr := fallbackCmd.CombinedOutput(); fbErr != nil {
			return fmt.Errorf("git clone (fallback) failed: %s: %w", strings.TrimSpace(string(fbOutput)), fbErr)
		}

		// Create and checkout new branch
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", "-b", branch)
		checkoutCmd.Dir = workDir
		if coOutput, coErr := checkoutCmd.CombinedOutput(); coErr != nil {
			return fmt.Errorf("git checkout -b %s failed: %s: %w", branch, strings.TrimSpace(string(coOutput)), coErr)
		}
	}

	return nil
}

// injectToken inserts the authentication token into a git URL based on the provider.
func injectToken(repoURL, token, gitProvider string) (string, error) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parse repo URL %q: %w", repoURL, err)
	}

	switch gitProvider {
	case "gitea":
		parsed.User = url.UserPassword("oauth2", token)
	case "github":
		parsed.User = url.User(token)
	default:
		parsed.User = url.User(token)
	}

	return parsed.String(), nil
}
