//go:build integration

package git

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zakari/hopeitworks/backend/pkg/exec"
)

func TestIntegrationGhCliAdapter_CloneBranchPush(t *testing.T) {
	// This integration test requires:
	// - gh CLI installed and authenticated
	// - git installed and configured
	// - network access to GitHub

	tmpDir, err := os.MkdirTemp("", "ghcli-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := exec.NewRealCommandRunner()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewGhCliAdapter(runner, logger)

	ctx := context.Background()
	targetDir := filepath.Join(tmpDir, "repo")

	// Clone a known public repository
	err = adapter.CloneRepo(ctx, "zakari/hopeitworks", targetDir)
	if err != nil {
		t.Fatalf("CloneRepo failed: %v", err)
	}

	// Verify the clone succeeded by checking for a known file
	if _, err := os.Stat(filepath.Join(targetDir, "CLAUDE.md")); os.IsNotExist(err) {
		t.Fatal("expected CLAUDE.md to exist in cloned repo")
	}

	// Create a feature branch
	branchName := "feat/integration-test-branch"
	err = adapter.CreateBranch(ctx, targetDir, branchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Create a test file so we have something to commit
	testFile := filepath.Join(targetDir, "integration-test.txt")
	if err := os.WriteFile(testFile, []byte("integration test\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Push the commit
	err = adapter.Push(ctx, targetDir, "test(git): integration test commit")
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Clean up remote branch
	_, _ = runner.Run(ctx, targetDir, "git", "push", "origin", "--delete", branchName)
}

// TestIntegrationGhCliAdapter_PRWorkflow tests the full PR lifecycle:
// CreatePR -> GetCIStatus -> MergePR.
func TestIntegrationGhCliAdapter_PRWorkflow(t *testing.T) {
	// This integration test requires:
	// - gh CLI installed and authenticated
	// - git installed and configured
	// - network access to GitHub
	// - GITHUB_TOKEN environment variable set

	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("skipping: GITHUB_TOKEN not set")
	}

	tmpDir, err := os.MkdirTemp("", "ghcli-pr-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	runner := exec.NewRealCommandRunner()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewGhCliAdapter(runner, logger)

	ctx := context.Background()
	targetDir := filepath.Join(tmpDir, "repo")

	// Clone the repository
	err = adapter.CloneRepo(ctx, "zakari/hopeitworks", targetDir)
	if err != nil {
		t.Fatalf("CloneRepo failed: %v", err)
	}

	// Create a feature branch
	branchName := "feat/integration-test-pr-workflow"
	err = adapter.CreateBranch(ctx, targetDir, branchName)
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Create a test file so we have something to commit
	testFile := filepath.Join(targetDir, "integration-pr-test.txt")
	if writeErr := os.WriteFile(testFile, []byte("integration PR workflow test\n"), 0644); writeErr != nil {
		t.Fatalf("failed to create test file: %v", writeErr)
	}

	// Push the commit
	err = adapter.Push(ctx, targetDir, "test(git): integration test PR workflow")
	if err != nil {
		t.Fatalf("Push failed: %v", err)
	}

	// Ensure remote branch cleanup on test failure
	defer func() {
		_, _ = runner.Run(ctx, targetDir, "git", "push", "origin", "--delete", branchName)
	}()

	// Create a PR
	prURL, err := adapter.CreatePR(ctx, targetDir, "test(git): integration test PR workflow", "Automated integration test PR - safe to delete", "develop")
	if err != nil {
		t.Fatalf("CreatePR failed: %v", err)
	}

	if prURL == "" {
		t.Fatal("expected non-empty PR URL")
	}
	t.Logf("created PR: %s", prURL)

	// Poll CI status with timeout
	deadline := time.Now().Add(5 * time.Minute)
	var ciStatus string
	for time.Now().Before(deadline) {
		ciStatus, err = adapter.GetCIStatus(ctx, targetDir)
		if err != nil {
			t.Fatalf("GetCIStatus failed: %v", err)
		}

		t.Logf("CI status: %s", ciStatus)

		if ciStatus == "pass" || ciStatus == "fail" || ciStatus == "no_checks" {
			break
		}

		time.Sleep(15 * time.Second)
	}

	if ciStatus == "pending" {
		t.Log("CI checks did not complete within timeout, proceeding with merge attempt")
	}

	// Merge the PR (squash merge)
	err = adapter.MergePR(ctx, targetDir, prURL)
	if err != nil {
		t.Fatalf("MergePR failed: %v", err)
	}

	t.Log("PR merged successfully")

	// Verify branch was deleted by trying to check it on remote
	stdout, checkErr := runner.Run(ctx, targetDir, "git", "ls-remote", "--heads", "origin", branchName)
	if checkErr == nil && stdout != "" {
		t.Error("expected branch to be deleted after merge, but it still exists on remote")
	}
}
