# Story 3.2: [BACK] GitProvider port + gh CLI adapter (clone, branch, push)

Status: ready-for-dev

## Story

As a backend developer, I want a GitProvider port with gh CLI implementation for repository operations, so that the pipeline can interact with Git repositories through a testable interface.

## Acceptance Criteria (BDD)

**AC1: GitProvider port interface defines clone, branch, and push operations**
- **Given** a GitProvider port interface in `backend/internal/domain/port/git_provider.go`
- **When** the interface is reviewed
- **Then** it declares CloneRepo(ctx, repoURL, targetDir) error
- **And** it declares CreateBranch(ctx, workDir, branchName) error
- **And** it declares Push(ctx, workDir, commitMsg) error
- **And** all methods return domain errors with contextual information

**AC2: CommandRunner port interface wraps shell command execution**
- **Given** a CommandRunner port interface in `backend/internal/domain/port/command_runner.go`
- **When** the interface is reviewed
- **Then** it declares Run(ctx, workDir, name, args...) (stdout string, err error)
- **And** it supports context cancellation
- **And** it includes working directory for command execution

**AC3: Real CommandRunner implementation executes shell commands**
- **Given** a CommandRunner implementation in `backend/pkg/exec/command_runner.go`
- **When** Run is called with valid command parameters
- **Then** it executes the command using exec.CommandContext
- **And** it captures stdout and stderr
- **And** it respects context cancellation
- **And** it returns combined output on error

**AC4: gh CLI adapter implements GitProvider using CommandRunner**
- **Given** a gh CLI adapter in `backend/internal/adapter/git/gh_cli_adapter.go`
- **When** CloneRepo is called with a repository URL and target directory
- **Then** it executes `gh repo clone <repoURL> <targetDir>` via CommandRunner
- **And** it wraps errors in DomainError with command output context

**AC5: gh CLI adapter creates feature branches with conventional naming**
- **Given** a git repository cloned via gh CLI adapter
- **When** CreateBranch is called with branchName matching `feat/{story-key}-{slug}`
- **Then** it executes `git checkout -b <branchName>` via CommandRunner
- **And** it validates branch name format before execution
- **And** it wraps errors in DomainError with command output context

**AC6: gh CLI adapter pushes commits with conventional commit messages**
- **Given** a git repository with uncommitted changes
- **When** Push is called with a conventional commit message
- **Then** it executes `git add .` via CommandRunner
- **And** it executes `git commit -m "<commitMsg>"` via CommandRunner
- **And** it executes `git push -u origin HEAD` via CommandRunner
- **And** it wraps errors in DomainError with command output context

**AC7: Unit tests verify gh CLI adapter behavior with mock CommandRunner**
- **Given** unit tests in `backend/internal/adapter/git/gh_cli_adapter_test.go`
- **When** tests are executed
- **Then** CloneRepo tests verify correct gh repo clone command arguments
- **And** CreateBranch tests verify correct git checkout command arguments
- **And** Push tests verify git add, commit, and push command sequence
- **And** all tests use mock CommandRunner to avoid actual git operations
- **And** error handling tests verify DomainError wrapping with command output

**AC8: Integration test verifies real gh CLI and git operations**
- **Given** an integration test with a real CommandRunner
- **When** the test executes CloneRepo, CreateBranch, and Push sequence
- **Then** it clones a test repository to a temporary directory
- **And** it creates a new branch with conventional naming
- **And** it pushes a commit with conventional commit message
- **And** it cleans up the temporary directory after test completion

## Tasks / Subtasks

- [ ] [BACK] Task 1: Define GitProvider and CommandRunner port interfaces (AC: #1, #2)
  - [ ] Create `backend/internal/domain/port/git_provider.go` with CloneRepo, CreateBranch, Push methods
  - [ ] Create `backend/internal/domain/port/command_runner.go` with Run method
  - [ ] Document all interface methods with godoc comments
  - [ ] Add context.Context as first parameter for all methods

- [ ] [BACK] Task 2: Implement real CommandRunner in pkg/exec (AC: #3)
  - [ ] Create `backend/pkg/exec/command_runner.go` with RealCommandRunner struct
  - [ ] Implement Run method using exec.CommandContext
  - [ ] Capture and combine stdout/stderr on error
  - [ ] Create `backend/pkg/exec/command_runner_test.go` with basic execution tests
  - [ ] Test context cancellation behavior

- [ ] [BACK] Task 3: Implement gh CLI adapter for GitProvider (AC: #4)
  - [ ] Create `backend/internal/adapter/git/gh_cli_adapter.go`
  - [ ] Add GhCliAdapter struct with CommandRunner dependency
  - [ ] Implement CloneRepo using `gh repo clone` command
  - [ ] Wrap all errors in DomainError with command output context

- [ ] [BACK] Task 4: Implement CreateBranch with validation (AC: #5)
  - [ ] Add CreateBranch method to GhCliAdapter
  - [ ] Validate branch name format: `feat/{story-key}-{slug}` or `fix/{story-key}-{slug}`
  - [ ] Execute `git checkout -b <branchName>` via CommandRunner
  - [ ] Wrap errors in DomainError with command output

- [ ] [BACK] Task 5: Implement Push with conventional commit workflow (AC: #6)
  - [ ] Add Push method to GhCliAdapter
  - [ ] Execute `git add .` via CommandRunner
  - [ ] Execute `git commit -m "<commitMsg>"` via CommandRunner
  - [ ] Execute `git push -u origin HEAD` via CommandRunner
  - [ ] Wrap errors in DomainError with command output at each step

- [ ] [BACK] Task 6: Create mock CommandRunner for unit tests (AC: #7)
  - [ ] Create mock CommandRunner in `gh_cli_adapter_test.go`
  - [ ] Track command invocations (name, args) for verification
  - [ ] Support configurable return values (stdout, error)

- [ ] [BACK] Task 7: Write unit tests for gh CLI adapter (AC: #7)
  - [ ] Test CloneRepo with valid repoURL and targetDir
  - [ ] Test CreateBranch with valid and invalid branch names
  - [ ] Test Push command sequence (add, commit, push)
  - [ ] Test error handling and DomainError wrapping
  - [ ] Verify command arguments passed to mock CommandRunner

- [ ] [BACK] Task 8: Write integration test with real git operations (AC: #8)
  - [ ] Create integration test file with `//go:build integration` tag
  - [ ] Use real CommandRunner implementation
  - [ ] Clone a test repository (e.g., hopeitworks itself) to temp directory
  - [ ] Create branch and push commit to verify full workflow
  - [ ] Clean up temp directory in test teardown

## Dev Notes

### Dependencies
- Story 1-1: Go project scaffolding (provides base project structure)
- gh CLI must be installed in agent container (Dockerfile.base)
- Git must be installed in agent container (already present)

### Architecture Requirements
- **Hexagonal architecture:** GitProvider is a port in domain/port, gh CLI adapter in adapter/git
- **Testability:** CommandRunner abstraction allows mocking shell commands in unit tests
- **Error handling:** All adapter errors wrapped in DomainError via pkg/errors
- **Structured logging:** Use slog to log commands before execution (debug level)

### File Paths (exact)
```
backend/internal/domain/port/git_provider.go       # GitProvider port interface
backend/internal/domain/port/command_runner.go     # CommandRunner port interface
backend/internal/adapter/git/gh_cli_adapter.go     # gh CLI implementation
backend/internal/adapter/git/gh_cli_adapter_test.go # Unit tests with mock CommandRunner
backend/pkg/exec/command_runner.go                 # Real CommandRunner implementation
backend/pkg/exec/command_runner_test.go            # CommandRunner tests
```

### Technical Specifications

**GitProvider port interface:**
```go
package port

import "context"

// GitProvider abstracts Git repository operations.
// Implementations must use conventional branch naming and commit messages.
type GitProvider interface {
    // CloneRepo clones a repository to the target directory using gh CLI.
    // repoURL format: "owner/repo" or full HTTPS URL
    CloneRepo(ctx context.Context, repoURL string, targetDir string) error

    // CreateBranch creates and checks out a new branch.
    // branchName must follow convention: feat/{story-key}-{slug} or fix/{story-key}-{slug}
    CreateBranch(ctx context.Context, workDir string, branchName string) error

    // Push stages all changes, commits with the given message, and pushes to origin.
    // commitMsg must follow conventional commit format: type(scope): message
    Push(ctx context.Context, workDir string, commitMsg string) error
}
```

**CommandRunner port interface:**
```go
package port

import "context"

// CommandRunner abstracts shell command execution for testability.
type CommandRunner interface {
    // Run executes a command in the specified working directory.
    // Returns stdout on success, error with combined output on failure.
    Run(ctx context.Context, workDir string, name string, args ...string) (stdout string, err error)
}
```

**Real CommandRunner implementation:**
```go
package exec

import (
    "bytes"
    "context"
    "os/exec"
)

type RealCommandRunner struct{}

func NewRealCommandRunner() *RealCommandRunner {
    return &RealCommandRunner{}
}

func (r *RealCommandRunner) Run(ctx context.Context, workDir string, name string, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Dir = workDir

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    if err != nil {
        // Combine stdout and stderr for error context
        combined := stdout.String() + stderr.String()
        return "", fmt.Errorf("%w: %s", err, combined)
    }

    return stdout.String(), nil
}
```

**gh CLI adapter:**
```go
package git

import (
    "context"
    "fmt"
    "regexp"

    "hopeitworks/backend/internal/domain/port"
    "hopeitworks/backend/pkg/errors"
)

type GhCliAdapter struct {
    runner port.CommandRunner
}

func NewGhCliAdapter(runner port.CommandRunner) *GhCliAdapter {
    return &GhCliAdapter{runner: runner}
}

func (a *GhCliAdapter) CloneRepo(ctx context.Context, repoURL string, targetDir string) error {
    stdout, err := a.runner.Run(ctx, "", "gh", "repo", "clone", repoURL, targetDir)
    if err != nil {
        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to clone repository %s: %v", repoURL, err),
            map[string]any{"repo_url": repoURL, "target_dir": targetDir, "output": stdout},
        )
    }
    return nil
}

func (a *GhCliAdapter) CreateBranch(ctx context.Context, workDir string, branchName string) error {
    // Validate branch name format
    validBranchName := regexp.MustCompile(`^(feat|fix)/[a-zA-Z0-9]+-[a-zA-Z0-9-]+$`)
    if !validBranchName.MatchString(branchName) {
        return errors.NewDomainError(
            errors.ErrCodeInvalidInput,
            fmt.Sprintf("invalid branch name format: %s (expected feat/{story-key}-{slug})", branchName),
            map[string]any{"branch_name": branchName},
        )
    }

    stdout, err := a.runner.Run(ctx, workDir, "git", "checkout", "-b", branchName)
    if err != nil {
        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to create branch %s: %v", branchName, err),
            map[string]any{"branch_name": branchName, "work_dir": workDir, "output": stdout},
        )
    }
    return nil
}

func (a *GhCliAdapter) Push(ctx context.Context, workDir string, commitMsg string) error {
    // Stage all changes
    if _, err := a.runner.Run(ctx, workDir, "git", "add", "."); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to stage changes: %v", err),
            map[string]any{"work_dir": workDir},
        )
    }

    // Commit with conventional message
    if _, err := a.runner.Run(ctx, workDir, "git", "commit", "-m", commitMsg); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to commit changes: %v", err),
            map[string]any{"work_dir": workDir, "commit_msg": commitMsg},
        )
    }

    // Push to origin
    if _, err := a.runner.Run(ctx, workDir, "git", "push", "-u", "origin", "HEAD"); err != nil {
        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to push changes: %v", err),
            map[string]any{"work_dir": workDir},
        )
    }

    return nil
}
```

**Error codes to add to pkg/errors:**
```go
const (
    ErrCodeGitOperationFailed = "GIT_OPERATION_FAILED"
    ErrCodeInvalidInput       = "INVALID_INPUT"
)
```

### Testing Requirements

**Unit tests (gh_cli_adapter_test.go):**
- Mock CommandRunner tracks all command invocations
- Test CloneRepo with valid repoURL and targetDir
- Test CreateBranch with valid branch names (feat/1-2-slug, fix/3-4-slug)
- Test CreateBranch rejects invalid branch names (missing prefix, wrong format)
- Test Push executes correct sequence: add, commit, push
- Test error handling wraps errors in DomainError with command output
- No actual git/gh CLI commands executed in unit tests

**Integration tests:**
- Tag with `//go:build integration`
- Use real CommandRunner
- Clone test repository to temp directory
- Create feature branch
- Make trivial change (e.g., create test file)
- Push commit
- Verify branch exists on remote (optional, requires gh CLI query)
- Clean up temp directory

**CommandRunner tests:**
- Test successful command execution returns stdout
- Test failed command returns error with combined output
- Test context cancellation interrupts command
- Test working directory is respected

### References
- Story 1-1: Go project scaffolding
- Story 3-3: CreatePR, MergePR, GetCIStatus (Wave 6, not in this story)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md`
- CLAUDE.md: Conventional commit format, branch naming conventions

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 5 backend infrastructure
