package exec

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRealCommandRunner_Run_Success(t *testing.T) {
	runner := NewRealCommandRunner()

	stdout, err := runner.Run(context.Background(), "", "echo", "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(stdout)
	if got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestRealCommandRunner_Run_FailedCommand(t *testing.T) {
	runner := NewRealCommandRunner()

	_, err := runner.Run(context.Background(), "", "ls", "/nonexistent-path-that-should-not-exist")
	if err == nil {
		t.Fatal("expected error for failed command, got nil")
	}
}

func TestRealCommandRunner_Run_ContextCancellation(t *testing.T) {
	runner := NewRealCommandRunner()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := runner.Run(ctx, "", "sleep", "10")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestRealCommandRunner_Run_WorkingDirectory(t *testing.T) {
	runner := NewRealCommandRunner()

	tmpDir, err := os.MkdirTemp("", "cmdrunner-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	stdout, err := runner.Run(context.Background(), tmpDir, "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(stdout)
	// Resolve symlinks for macOS /var -> /private/var
	resolvedTmp, _ := os.Readlink(tmpDir)
	if resolvedTmp == "" {
		resolvedTmp = tmpDir
	}
	if got != tmpDir && got != resolvedTmp {
		t.Errorf("expected working directory %q, got %q", tmpDir, got)
	}
}

func TestRealCommandRunner_Run_CombinedOutputOnError(t *testing.T) {
	runner := NewRealCommandRunner()

	_, err := runner.Run(context.Background(), "", "sh", "-c", "echo 'stderr msg' >&2; exit 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "stderr msg") {
		t.Errorf("expected error to contain stderr output, got: %v", err)
	}
}

func TestRealCommandRunner_Run_EmptyWorkDir(t *testing.T) {
	runner := NewRealCommandRunner()

	// Empty workDir should use the current directory (no error)
	stdout, err := runner.Run(context.Background(), "", "echo", "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(stdout)
	if got != "test" {
		t.Errorf("expected 'test', got %q", got)
	}
}
