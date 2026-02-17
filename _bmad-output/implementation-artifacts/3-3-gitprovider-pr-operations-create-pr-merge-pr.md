# Story 3.3: [BACK] GitProvider PR operations (create PR, merge PR)

Status: ready-for-dev

## Story

As a backend developer, I want PR creation and merge operations via the GitProvider, so that the pipeline can complete the code review and merge cycle.

## Acceptance Criteria (BDD)

**AC1: GitProvider port interface defines CreatePR method**
- **Given** the GitProvider port interface in `backend/internal/domain/port/git_provider.go`
- **When** the interface is extended with PR operations
- **Then** it declares CreatePR(ctx, workDir, title, body, baseBranch) (prURL string, err error)
- **And** the method creates a PR via gh CLI and returns the PR URL
- **And** it wraps errors in DomainError with contextual information

**AC2: GitProvider port interface defines MergePR method**
- **Given** the GitProvider port interface in `backend/internal/domain/port/git_provider.go`
- **When** the interface is extended with merge operations
- **Then** it declares MergePR(ctx, workDir, prIdentifier) error
- **And** the method performs squash merge and deletes the source branch
- **And** it wraps errors in DomainError with contextual information

**AC3: GitProvider port interface defines GetCIStatus method**
- **Given** the GitProvider port interface in `backend/internal/domain/port/git_provider.go`
- **When** the interface is extended with CI status polling
- **Then** it declares GetCIStatus(ctx, workDir) (status string, err error)
- **And** the method returns "pass", "fail", "pending", or "no_checks"
- **And** it wraps errors in DomainError with contextual information

**AC4: gh CLI adapter implements CreatePR**
- **Given** a gh CLI adapter in `backend/internal/adapter/git/gh_cli_adapter.go`
- **When** CreatePR is called with title, body, and baseBranch
- **Then** it executes `gh pr create --title "..." --body "..." --base <baseBranch>` via CommandRunner
- **And** it parses the PR URL from stdout
- **And** it wraps errors in DomainError with command output context
- **And** authentication failures return GIT_AUTH_FAILED error code

**AC5: gh CLI adapter implements MergePR with squash merge**
- **Given** a PR exists on the remote repository
- **When** MergePR is called with prIdentifier (PR number or URL)
- **Then** it executes `gh pr merge <prIdentifier> --squash --delete-branch` via CommandRunner
- **And** it wraps errors in DomainError with command output context
- **And** merge conflicts return MERGE_CONFLICT error code with conflict details
- **And** PR not found returns PR_NOT_FOUND error code

**AC6: gh CLI adapter implements GetCIStatus with check polling**
- **Given** a PR exists with CI checks configured
- **When** GetCIStatus is called in the PR's working directory
- **Then** it executes `gh pr checks --json name,state,conclusion` via CommandRunner
- **And** it parses JSON output to determine overall status
- **And** all checks "success" returns "pass"
- **And** any check "failure" returns "fail"
- **And** any check "pending" or "queued" returns "pending"
- **And** no checks configured returns "no_checks"

**AC7: New error codes added to pkg/errors**
- **Given** error code constants in `backend/pkg/errors/codes.go`
- **When** the constants are reviewed
- **Then** MERGE_CONFLICT is defined for merge conflict scenarios
- **And** GIT_AUTH_FAILED is defined for authentication failures
- **And** PR_NOT_FOUND is defined for missing pull request errors

**AC8: Unit tests verify CreatePR with mock CommandRunner**
- **Given** unit tests in `backend/internal/adapter/git/gh_cli_adapter_test.go`
- **When** CreatePR tests are executed
- **Then** they verify correct gh pr create command arguments
- **And** they verify PR URL parsing from stdout
- **And** they verify error handling for auth failures
- **And** all tests use mock CommandRunner to avoid actual gh CLI operations

**AC9: Unit tests verify MergePR and GetCIStatus with mock CommandRunner**
- **Given** unit tests in `backend/internal/adapter/git/gh_cli_adapter_test.go`
- **When** MergePR and GetCIStatus tests are executed
- **Then** MergePR tests verify correct gh pr merge command arguments
- **And** MergePR tests verify merge conflict error detection
- **And** MergePR tests verify PR not found error detection
- **And** GetCIStatus tests verify JSON parsing and status mapping
- **And** all tests use mock CommandRunner to avoid actual gh CLI operations

**AC10: Integration test verifies PR workflow**
- **Given** an integration test with a real CommandRunner
- **When** the test executes CreatePR, GetCIStatus, and MergePR sequence
- **Then** it creates a test PR on a real repository
- **And** it polls CI status until completion or timeout
- **And** it merges the PR with squash merge
- **And** it verifies the branch is deleted after merge

## Tasks / Subtasks

- [ ] [BACK] Task 1: Extend GitProvider port interface with PR operations (AC: #1, #2, #3)
  - [ ] Add CreatePR method to `backend/internal/domain/port/git_provider.go`
  - [ ] Add MergePR method to `backend/internal/domain/port/git_provider.go`
  - [ ] Add GetCIStatus method to `backend/internal/domain/port/git_provider.go`
  - [ ] Document all new methods with godoc comments

- [ ] [BACK] Task 2: Implement CreatePR in GhCliAdapter (AC: #4)
  - [ ] Add CreatePR method to `backend/internal/adapter/git/gh_cli_adapter.go`
  - [ ] Execute `gh pr create --title --body --base` via CommandRunner
  - [ ] Parse PR URL from stdout using regex or string parsing
  - [ ] Detect auth failures in stderr and wrap in GIT_AUTH_FAILED error

- [ ] [BACK] Task 3: Implement MergePR in GhCliAdapter (AC: #5)
  - [ ] Add MergePR method to `backend/internal/adapter/git/gh_cli_adapter.go`
  - [ ] Execute `gh pr merge --squash --delete-branch` via CommandRunner
  - [ ] Detect merge conflicts in stderr and wrap in MERGE_CONFLICT error
  - [ ] Detect PR not found in stderr and wrap in PR_NOT_FOUND error

- [ ] [BACK] Task 4: Implement GetCIStatus in GhCliAdapter (AC: #6)
  - [ ] Add GetCIStatus method to `backend/internal/adapter/git/gh_cli_adapter.go`
  - [ ] Execute `gh pr checks --json name,state,conclusion` via CommandRunner
  - [ ] Parse JSON output into struct for check results
  - [ ] Map check conclusions to status: "pass", "fail", "pending", "no_checks"

- [ ] [BACK] Task 5: Add new error codes to pkg/errors (AC: #7)
  - [ ] Add MERGE_CONFLICT constant to `backend/pkg/errors/codes.go`
  - [ ] Add GIT_AUTH_FAILED constant to `backend/pkg/errors/codes.go`
  - [ ] Add PR_NOT_FOUND constant to `backend/pkg/errors/codes.go`

- [ ] [BACK] Task 6: Write unit tests for CreatePR (AC: #8)
  - [ ] Test CreatePR with valid title, body, and baseBranch
  - [ ] Test PR URL parsing from gh CLI stdout
  - [ ] Test auth failure detection and error wrapping
  - [ ] Verify command arguments passed to mock CommandRunner

- [ ] [BACK] Task 7: Write unit tests for MergePR and GetCIStatus (AC: #9)
  - [ ] Test MergePR with valid prIdentifier
  - [ ] Test merge conflict detection and error wrapping
  - [ ] Test PR not found detection and error wrapping
  - [ ] Test GetCIStatus JSON parsing with various check states
  - [ ] Test GetCIStatus status mapping logic

- [ ] [BACK] Task 8: Write integration test for PR workflow (AC: #10)
  - [ ] Create integration test file with `//go:build integration` tag
  - [ ] Use real CommandRunner implementation
  - [ ] Create PR on test repository with CreatePR
  - [ ] Poll CI status with GetCIStatus (with timeout)
  - [ ] Merge PR with MergePR after CI passes
  - [ ] Clean up test resources

## Dev Notes

### Dependencies
- Story 3-2: GitProvider port + gh CLI adapter (Wave 5, provides base interface and adapter)
- Story 1-1: Go project scaffolding (provides base project structure)
- gh CLI must be installed in agent container (Dockerfile.base)
- Git must be installed in agent container (already present)

### Architecture Requirements
- **Extends existing port:** This story adds methods to the existing GitProvider interface from Story 3-2
- **Hexagonal architecture:** New methods follow same pattern as existing GitProvider methods
- **Testability:** Uses existing CommandRunner abstraction from Story 3-2
- **Error handling:** All adapter errors wrapped in DomainError via pkg/errors
- **Structured logging:** Use slog to log commands before execution (debug level)

### File Paths (exact)
```
backend/internal/domain/port/git_provider.go       # EXTEND with CreatePR, MergePR, GetCIStatus
backend/internal/adapter/git/gh_cli_adapter.go     # ADD CreatePR, MergePR, GetCIStatus implementations
backend/internal/adapter/git/gh_cli_adapter_test.go # ADD unit tests for new methods
backend/pkg/errors/codes.go                        # ADD MERGE_CONFLICT, GIT_AUTH_FAILED, PR_NOT_FOUND
```

### Technical Specifications

**GitProvider port interface additions:**
```go
package port

import "context"

// GitProvider abstracts Git repository operations.
// (Existing methods: CloneRepo, CreateBranch, Push from Story 3-2)
type GitProvider interface {
    // ... existing methods from Story 3-2 ...

    // CreatePR creates a pull request and returns the PR URL.
    // title: PR title (should follow conventional commit format for squash merge)
    // body: PR description/body
    // baseBranch: target branch (typically "main" or "develop")
    // Returns: PR URL (e.g., "https://github.com/owner/repo/pull/123")
    CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (prURL string, err error)

    // MergePR squash-merges a pull request and deletes the source branch.
    // prIdentifier: PR number (e.g., "123") or PR URL
    // Performs squash merge to maintain clean commit history.
    MergePR(ctx context.Context, workDir string, prIdentifier string) error

    // GetCIStatus returns the CI check status for the current branch's PR.
    // Returns: "pass" (all checks successful), "fail" (any check failed),
    //          "pending" (checks running), "no_checks" (no CI configured)
    GetCIStatus(ctx context.Context, workDir string) (status string, err error)
}
```

**CreatePR implementation:**
```go
func (a *GhCliAdapter) CreatePR(ctx context.Context, workDir string, title string, body string, baseBranch string) (string, error) {
    stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "create", "--title", title, "--body", body, "--base", baseBranch)
    if err != nil {
        // Check for auth failures
        if strings.Contains(stdout, "authentication") || strings.Contains(stdout, "login required") {
            return "", errors.NewDomainError(
                errors.ErrCodeGitAuthFailed,
                fmt.Sprintf("GitHub authentication failed: %v", err),
                map[string]any{"work_dir": workDir, "output": stdout},
            )
        }

        return "", errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to create PR: %v", err),
            map[string]any{"work_dir": workDir, "title": title, "base_branch": baseBranch, "output": stdout},
        )
    }

    // Parse PR URL from stdout (gh pr create returns URL on last line)
    lines := strings.Split(strings.TrimSpace(stdout), "\n")
    prURL := strings.TrimSpace(lines[len(lines)-1])

    if !strings.HasPrefix(prURL, "http") {
        return "", errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            "failed to parse PR URL from gh CLI output",
            map[string]any{"output": stdout},
        )
    }

    return prURL, nil
}
```

**MergePR implementation:**
```go
func (a *GhCliAdapter) MergePR(ctx context.Context, workDir string, prIdentifier string) error {
    stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "merge", prIdentifier, "--squash", "--delete-branch")
    if err != nil {
        // Check for merge conflicts
        if strings.Contains(stdout, "merge conflict") || strings.Contains(stdout, "conflicts") {
            return errors.NewDomainError(
                errors.ErrCodeMergeConflict,
                fmt.Sprintf("merge conflict detected for PR %s: %v", prIdentifier, err),
                map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
            )
        }

        // Check for PR not found
        if strings.Contains(stdout, "no pull requests found") || strings.Contains(stdout, "not found") {
            return errors.NewDomainError(
                errors.ErrCodePRNotFound,
                fmt.Sprintf("pull request not found: %s", prIdentifier),
                map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
            )
        }

        return errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to merge PR %s: %v", prIdentifier, err),
            map[string]any{"pr_identifier": prIdentifier, "work_dir": workDir, "output": stdout},
        )
    }

    return nil
}
```

**GetCIStatus implementation:**
```go
type prCheck struct {
    Name       string `json:"name"`
    State      string `json:"state"`
    Conclusion string `json:"conclusion"`
}

func (a *GhCliAdapter) GetCIStatus(ctx context.Context, workDir string) (string, error) {
    stdout, err := a.runner.Run(ctx, workDir, "gh", "pr", "checks", "--json", "name,state,conclusion")
    if err != nil {
        // No PR found for current branch
        if strings.Contains(stdout, "no pull request") {
            return "", errors.NewDomainError(
                errors.ErrCodePRNotFound,
                "no pull request found for current branch",
                map[string]any{"work_dir": workDir, "output": stdout},
            )
        }

        return "", errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to get CI status: %v", err),
            map[string]any{"work_dir": workDir, "output": stdout},
        )
    }

    // Parse JSON output
    var checks []prCheck
    if err := json.Unmarshal([]byte(stdout), &checks); err != nil {
        return "", errors.NewDomainError(
            errors.ErrCodeGitOperationFailed,
            fmt.Sprintf("failed to parse CI check JSON: %v", err),
            map[string]any{"output": stdout},
        )
    }

    // No checks configured
    if len(checks) == 0 {
        return "no_checks", nil
    }

    // Determine overall status
    hasPending := false
    for _, check := range checks {
        // Check is still running
        if check.State == "pending" || check.State == "queued" || check.State == "in_progress" {
            hasPending = true
            continue
        }

        // Check failed
        if check.Conclusion == "failure" || check.Conclusion == "timed_out" || check.Conclusion == "action_required" {
            return "fail", nil
        }
    }

    // Some checks still pending
    if hasPending {
        return "pending", nil
    }

    // All checks passed
    return "pass", nil
}
```

**Error codes to add to pkg/errors/codes.go:**
```go
const (
    // ... existing codes from previous stories ...

    // Git provider errors (Story 3-3)
    ErrCodeMergeConflict  = "MERGE_CONFLICT"
    ErrCodeGitAuthFailed  = "GIT_AUTH_FAILED"
    ErrCodePRNotFound     = "PR_NOT_FOUND"
)
```

**gh CLI commands used:**
```bash
# CreatePR
gh pr create --title "feat(scope): add feature" --body "Description here" --base main
# Output: https://github.com/owner/repo/pull/123

# MergePR
gh pr merge 123 --squash --delete-branch
# Output: Merged pull request #123 (deleted branch feat/1-2-example)

# GetCIStatus
gh pr checks --json name,state,conclusion
# Output: [{"name":"CI","state":"completed","conclusion":"success"}]
```

### Testing Requirements

**Unit tests (gh_cli_adapter_test.go):**
- Mock CommandRunner tracks all command invocations
- Test CreatePR with valid title, body, baseBranch
- Test CreatePR parses PR URL correctly from stdout
- Test CreatePR detects auth failures and returns GIT_AUTH_FAILED
- Test MergePR with valid prIdentifier
- Test MergePR detects merge conflicts and returns MERGE_CONFLICT
- Test MergePR detects PR not found and returns PR_NOT_FOUND
- Test GetCIStatus parses JSON output correctly
- Test GetCIStatus returns "pass" when all checks succeed
- Test GetCIStatus returns "fail" when any check fails
- Test GetCIStatus returns "pending" when checks are running
- Test GetCIStatus returns "no_checks" when no checks configured
- No actual gh CLI commands executed in unit tests

**Integration tests:**
- Tag with `//go:build integration`
- Use real CommandRunner
- Create test PR on test repository (e.g., hopeitworks test branch)
- Poll CI status with timeout (max 5 minutes)
- Merge PR after CI passes
- Verify branch deletion
- Clean up test resources
- Skip if GITHUB_TOKEN not available

**Mock CommandRunner updates:**
- Support multiple sequential commands in single test
- Allow configuring different stdout/error for each command
- Track command sequence for verification

### References
- Story 3-2: GitProvider port + gh CLI adapter (base implementation)
- Story 1-1: Go project scaffolding
- Story 3-4: merge-story phase uses MergePR (Wave 6, depends on this story)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md`
- CLAUDE.md: Conventional commit format, PR workflow conventions
- gh CLI docs: https://cli.github.com/manual/gh_pr_create, gh_pr_merge, gh_pr_checks

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 5 backend infrastructure
