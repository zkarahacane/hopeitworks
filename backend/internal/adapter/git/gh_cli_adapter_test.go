package git

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/zakari/hopeitworks/backend/pkg/errors"
)

const gitCommand = "git"

// commandInvocation records a single command call.
type commandInvocation struct {
	WorkDir string
	Name    string
	Args    []string
}

// mockCommandRunner tracks command invocations and returns configurable results.
type mockCommandRunner struct {
	invocations []commandInvocation
	results     []mockResult
	callIndex   int
}

type mockResult struct {
	stdout string
	err    error
}

func newMockCommandRunner(results ...mockResult) *mockCommandRunner {
	return &mockCommandRunner{results: results}
}

func (m *mockCommandRunner) Run(_ context.Context, workDir string, name string, args ...string) (string, error) {
	m.invocations = append(m.invocations, commandInvocation{
		WorkDir: workDir,
		Name:    name,
		Args:    args,
	})

	if m.callIndex < len(m.results) {
		r := m.results[m.callIndex]
		m.callIndex++
		return r.stdout, r.err
	}
	m.callIndex++
	return "", nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, nil))
}

func TestCloneRepo_Success(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: ""})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.CloneRepo(context.Background(), "owner/repo", "/tmp/target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.invocations) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(runner.invocations))
	}

	inv := runner.invocations[0]
	if inv.WorkDir != "" {
		t.Errorf("expected empty workDir, got %q", inv.WorkDir)
	}
	if inv.Name != "gh" {
		t.Errorf("expected command 'gh', got %q", inv.Name)
	}
	expectedArgs := []string{"repo", "clone", "owner/repo", "/tmp/target"}
	assertArgs(t, inv.Args, expectedArgs)
}

func TestCloneRepo_Error(t *testing.T) {
	runner := newMockCommandRunner(mockResult{err: fmt.Errorf("clone failed: permission denied")})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.CloneRepo(context.Background(), "owner/repo", "/tmp/target")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

func TestCreateBranch_ValidNames(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
	}{
		{"feature branch with story key", "feat/1-2-my-feature"},
		{"fix branch with story key", "fix/3-4-bug-fix"},
		{"feature with numbers", "feat/10-20-slug"},
		{"fix with long slug", "fix/1-2-long-slug-name-here"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newMockCommandRunner(mockResult{stdout: ""})
			adapter := NewGhCliAdapter(runner, testLogger())

			err := adapter.CreateBranch(context.Background(), "/work", tt.branchName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(runner.invocations) != 1 {
				t.Fatalf("expected 1 invocation, got %d", len(runner.invocations))
			}

			inv := runner.invocations[0]
			if inv.WorkDir != "/work" {
				t.Errorf("expected workDir '/work', got %q", inv.WorkDir)
			}
			if inv.Name != gitCommand {
				t.Errorf("expected command %q, got %q", gitCommand, inv.Name)
			}
			expectedArgs := []string{"checkout", "-b", tt.branchName}
			assertArgs(t, inv.Args, expectedArgs)
		})
	}
}

func TestCreateBranch_InvalidNames(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
	}{
		{"missing prefix", "1-2-my-feature"},
		{"wrong prefix", "feature/1-2-my-feature"},
		{"no slug after key", "feat/123"},
		{"empty string", ""},
		{"just prefix", "feat/"},
		{"no hyphen in key-slug", "feat/nokey"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := newMockCommandRunner()
			adapter := NewGhCliAdapter(runner, testLogger())

			err := adapter.CreateBranch(context.Background(), "/work", tt.branchName)
			if err == nil {
				t.Fatal("expected error for invalid branch name, got nil")
			}

			domainErr, ok := err.(*errors.DomainError)
			if !ok {
				t.Fatalf("expected *errors.DomainError, got %T", err)
			}
			if domainErr.Code != errors.ErrCodeInvalidInput {
				t.Errorf("expected code %q, got %q", errors.ErrCodeInvalidInput, domainErr.Code)
			}

			if len(runner.invocations) != 0 {
				t.Errorf("expected no command invocations for invalid branch name, got %d", len(runner.invocations))
			}
		})
	}
}

func TestCreateBranch_GitError(t *testing.T) {
	runner := newMockCommandRunner(mockResult{err: fmt.Errorf("branch already exists")})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.CreateBranch(context.Background(), "/work", "feat/1-2-slug")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

func TestPush_Success(t *testing.T) {
	runner := newMockCommandRunner(
		mockResult{stdout: ""},            // git add .
		mockResult{stdout: ""},            // git commit
		mockResult{stdout: "pushed ok\n"}, // git push
	)
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.Push(context.Background(), "/work", "feat(git): add clone support")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.invocations) != 3 {
		t.Fatalf("expected 3 invocations, got %d", len(runner.invocations))
	}

	// Verify git add .
	inv0 := runner.invocations[0]
	if inv0.Name != gitCommand {
		t.Errorf("invocation 0: expected %q, got %q", gitCommand, inv0.Name)
	}
	assertArgs(t, inv0.Args, []string{"add", "."})
	if inv0.WorkDir != "/work" {
		t.Errorf("invocation 0: expected workDir '/work', got %q", inv0.WorkDir)
	}

	// Verify git commit -m
	inv1 := runner.invocations[1]
	if inv1.Name != gitCommand {
		t.Errorf("invocation 1: expected %q, got %q", gitCommand, inv1.Name)
	}
	assertArgs(t, inv1.Args, []string{"commit", "-m", "feat(git): add clone support"})

	// Verify git push -u origin HEAD
	inv2 := runner.invocations[2]
	if inv2.Name != gitCommand {
		t.Errorf("invocation 2: expected %q, got %q", gitCommand, inv2.Name)
	}
	assertArgs(t, inv2.Args, []string{"push", "-u", "origin", "HEAD"})
}

func TestPush_AddError(t *testing.T) {
	runner := newMockCommandRunner(mockResult{err: fmt.Errorf("add failed")})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.Push(context.Background(), "/work", "feat(git): test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}

	// Only git add should have been called
	if len(runner.invocations) != 1 {
		t.Errorf("expected 1 invocation (add only), got %d", len(runner.invocations))
	}
}

func TestPush_CommitError(t *testing.T) {
	runner := newMockCommandRunner(
		mockResult{stdout: ""},                       // git add succeeds
		mockResult{err: fmt.Errorf("commit failed")}, // git commit fails
	)
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.Push(context.Background(), "/work", "feat(git): test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}

	// git add and git commit should have been called
	if len(runner.invocations) != 2 {
		t.Errorf("expected 2 invocations (add, commit), got %d", len(runner.invocations))
	}
}

func TestPush_PushError(t *testing.T) {
	runner := newMockCommandRunner(
		mockResult{stdout: ""},                     // git add succeeds
		mockResult{stdout: ""},                     // git commit succeeds
		mockResult{err: fmt.Errorf("push failed")}, // git push fails
	)
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.Push(context.Background(), "/work", "feat(git): test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}

	// All three commands should have been called
	if len(runner.invocations) != 3 {
		t.Errorf("expected 3 invocations, got %d", len(runner.invocations))
	}
}

func TestCloneRepo_FullURL(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: ""})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.CloneRepo(context.Background(), "https://github.com/owner/repo.git", "/tmp/target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := runner.invocations[0]
	assertArgs(t, inv.Args, []string{"repo", "clone", "https://github.com/owner/repo.git", "/tmp/target"})
}

// --- CreatePR Tests ---

func TestCreatePR_Success(t *testing.T) {
	prURL := "https://github.com/owner/repo/pull/42"
	runner := newMockCommandRunner(mockResult{stdout: prURL + "\n"})
	adapter := NewGhCliAdapter(runner, testLogger())

	url, err := adapter.CreatePR(context.Background(), "/work", "feat(api): add endpoint", "PR body here", "develop")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != prURL {
		t.Errorf("expected PR URL %q, got %q", prURL, url)
	}

	if len(runner.invocations) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(runner.invocations))
	}

	inv := runner.invocations[0]
	if inv.WorkDir != "/work" {
		t.Errorf("expected workDir '/work', got %q", inv.WorkDir)
	}
	if inv.Name != "gh" {
		t.Errorf("expected command 'gh', got %q", inv.Name)
	}
	assertArgs(t, inv.Args, []string{"pr", "create", "--title", "feat(api): add endpoint", "--body", "PR body here", "--base", "develop"})
}

func TestCreatePR_MultiLineOutput(t *testing.T) {
	// gh pr create sometimes outputs progress info before the URL
	output := "Creating pull request for feat/1-2-feature into develop in owner/repo\n\nhttps://github.com/owner/repo/pull/99\n"
	runner := newMockCommandRunner(mockResult{stdout: output})
	adapter := NewGhCliAdapter(runner, testLogger())

	url, err := adapter.CreatePR(context.Background(), "/work", "title", "body", "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://github.com/owner/repo/pull/99" {
		t.Errorf("expected parsed PR URL, got %q", url)
	}
}

func TestCreatePR_AuthFailure(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "authentication required\nPlease run gh auth login",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.CreatePR(context.Background(), "/work", "title", "body", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitAuthFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitAuthFailed, domainErr.Code)
	}
}

func TestCreatePR_LoginRequired(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "login required to access this resource",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.CreatePR(context.Background(), "/work", "title", "body", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitAuthFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitAuthFailed, domainErr.Code)
	}
}

func TestCreatePR_GenericError(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "some unexpected error",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.CreatePR(context.Background(), "/work", "title", "body", "main")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

func TestCreatePR_InvalidURLOutput(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: "not a URL at all\n"})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.CreatePR(context.Background(), "/work", "title", "body", "main")
	if err == nil {
		t.Fatal("expected error for invalid URL output, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

// --- MergePR Tests ---

func TestMergePR_Success(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: "Merged pull request #42\n"})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runner.invocations) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(runner.invocations))
	}

	inv := runner.invocations[0]
	if inv.WorkDir != "/work" {
		t.Errorf("expected workDir '/work', got %q", inv.WorkDir)
	}
	if inv.Name != "gh" {
		t.Errorf("expected command 'gh', got %q", inv.Name)
	}
	assertArgs(t, inv.Args, []string{"pr", "merge", "42", "--squash", "--delete-branch"})
}

func TestMergePR_WithURL(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: "Merged\n"})
	adapter := NewGhCliAdapter(runner, testLogger())

	prURL := "https://github.com/owner/repo/pull/42"
	err := adapter.MergePR(context.Background(), "/work", prURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inv := runner.invocations[0]
	assertArgs(t, inv.Args, []string{"pr", "merge", prURL, "--squash", "--delete-branch"})
}

func TestMergePR_MergeConflict(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "merge conflict detected in main.go",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "42")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeMergeConflict {
		t.Errorf("expected code %q, got %q", errors.ErrCodeMergeConflict, domainErr.Code)
	}
}

func TestMergePR_Conflicts(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "there are conflicts that need to be resolved",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "42")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeMergeConflict {
		t.Errorf("expected code %q, got %q", errors.ErrCodeMergeConflict, domainErr.Code)
	}
}

func TestMergePR_PRNotFound(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "no pull requests found for branch feat/test",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodePRNotFound {
		t.Errorf("expected code %q, got %q", errors.ErrCodePRNotFound, domainErr.Code)
	}
}

func TestMergePR_NotFoundMessage(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "pull request not found",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "999")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodePRNotFound {
		t.Errorf("expected code %q, got %q", errors.ErrCodePRNotFound, domainErr.Code)
	}
}

func TestMergePR_GenericError(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "unexpected error occurred",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	err := adapter.MergePR(context.Background(), "/work", "42")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

// --- GetCIStatus Tests ---

func TestGetCIStatus_AllPass(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"completed","conclusion":"success"},{"name":"Lint","state":"completed","conclusion":"success"}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "pass" {
		t.Errorf("expected status 'pass', got %q", status)
	}

	inv := runner.invocations[0]
	if inv.Name != "gh" {
		t.Errorf("expected command 'gh', got %q", inv.Name)
	}
	assertArgs(t, inv.Args, []string{"pr", "checks", "--json", "name,state,conclusion"})
}

func TestGetCIStatus_Fail(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"completed","conclusion":"success"},{"name":"Lint","state":"completed","conclusion":"failure"}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
}

func TestGetCIStatus_TimedOut(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"completed","conclusion":"timed_out"}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
}

func TestGetCIStatus_ActionRequired(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"completed","conclusion":"action_required"}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
}

func TestGetCIStatus_Pending(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"pending","conclusion":""},{"name":"Lint","state":"completed","conclusion":"success"}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status 'pending', got %q", status)
	}
}

func TestGetCIStatus_Queued(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"queued","conclusion":""}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status 'pending', got %q", status)
	}
}

func TestGetCIStatus_InProgress(t *testing.T) {
	checksJSON := `[{"name":"CI","state":"in_progress","conclusion":""}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected status 'pending', got %q", status)
	}
}

func TestGetCIStatus_NoChecks(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: "[]"})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "no_checks" {
		t.Errorf("expected status 'no_checks', got %q", status)
	}
}

func TestGetCIStatus_PRNotFound(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "no pull request found for branch",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.GetCIStatus(context.Background(), "/work")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodePRNotFound {
		t.Errorf("expected code %q, got %q", errors.ErrCodePRNotFound, domainErr.Code)
	}
}

func TestGetCIStatus_GenericError(t *testing.T) {
	runner := newMockCommandRunner(mockResult{
		stdout: "something went wrong",
		err:    fmt.Errorf("exit status 1"),
	})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.GetCIStatus(context.Background(), "/work")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

func TestGetCIStatus_InvalidJSON(t *testing.T) {
	runner := newMockCommandRunner(mockResult{stdout: "not json at all"})
	adapter := NewGhCliAdapter(runner, testLogger())

	_, err := adapter.GetCIStatus(context.Background(), "/work")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	domainErr, ok := err.(*errors.DomainError)
	if !ok {
		t.Fatalf("expected *errors.DomainError, got %T", err)
	}
	if domainErr.Code != errors.ErrCodeGitOperationFailed {
		t.Errorf("expected code %q, got %q", errors.ErrCodeGitOperationFailed, domainErr.Code)
	}
}

func TestGetCIStatus_FailBeforePending(t *testing.T) {
	// If one check fails and another is pending, should return "fail"
	checksJSON := `[{"name":"CI","state":"completed","conclusion":"failure"},{"name":"Deploy","state":"pending","conclusion":""}]`
	runner := newMockCommandRunner(mockResult{stdout: checksJSON})
	adapter := NewGhCliAdapter(runner, testLogger())

	status, err := adapter.GetCIStatus(context.Background(), "/work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "fail" {
		t.Errorf("expected status 'fail', got %q", status)
	}
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("expected %d args %v, got %d args %v", len(want), want, len(got), got)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("arg[%d]: expected %q, got %q", i, want[i], got[i])
		}
	}
}
