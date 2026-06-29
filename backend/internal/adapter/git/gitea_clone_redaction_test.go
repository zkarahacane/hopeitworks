package git

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// TestGiteaCloneRepo_A1_NoTokenLeakInError proves that when `git clone` fails and its
// stderr echoes the credential-bearing clone URL, the wrapped error does NOT carry the
// token (security hardening A1: credentials stripped/scrubbed before surfacing).
func TestGiteaCloneRepo_A1_NoTokenLeakInError(t *testing.T) {
	const token = "ghp_secrettoken0123456789ABCDEFGHabcd"
	repoURL := "https://gitea.example.com/org/project.git"

	// git typically prints the full remote (with embedded credentials) on failure.
	gitStderr := fmt.Errorf("fatal: unable to access 'https://%s@gitea.example.com/org/project.git/': 403", token)
	runner := newMockCommandRunner(mockResult{err: gitStderr})

	adapter := NewGiteaAPIAdapter("https://gitea.example.com", token, runner, testLogger())
	err := adapter.CloneRepo(context.Background(), repoURL, "/tmp/target")
	if err == nil {
		t.Fatal("expected clone failure")
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("A1 violation: clone error leaked the token: %q", err.Error())
	}
}
