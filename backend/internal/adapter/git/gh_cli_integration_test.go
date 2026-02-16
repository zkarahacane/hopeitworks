//go:build integration

package git

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

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
