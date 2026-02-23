# Story R-2-1: [BACK] Backend action: git_branch

Status: ready-for-dev

## Story

As a **pipeline executor**,
I want a `git_branch` action that creates a feature branch from a configured pattern,
so that downstream steps (agent_run, git_pr) have a named branch to work on without coupling branch creation to the agent container.

## Acceptance Criteria (BDD)

### Scenario 1: Branch created with pattern rendering

```gherkin
Given a pipeline step with action_type "git_branch" and config:
  | branch_pattern | feat/{story_key}-{slug} |
  | base_branch    | main                    |
When the action executes with a RunContext for story key "S-03" and title "Add login page"
Then GitProvider.CreateBranch is called with branchName "feat/S-03-add-login-page"
  And RunContext.Metadata["branch_name"] is set to "feat/S-03-add-login-page"
  And the action returns nil (success)
```

### Scenario 2: Slug is derived from story title

```gherkin
Given a story title "Add login page (OAuth)"
When the branch name is rendered from pattern "feat/{story_key}-{slug}"
Then the slug is "add-login-page-oauth" (lowercased, special chars replaced by hyphen, no trailing hyphens)
```

### Scenario 3: Default base branch is main

```gherkin
Given a pipeline step with action_type "git_branch" and no "base_branch" config key
When the action executes
Then GitProvider.CreateBranch is called with the default base branch "main"
```

### Scenario 4: GitProvider failure is propagated

```gherkin
Given GitProvider.CreateBranch returns an error
When the action executes
Then the action returns an error wrapping the GitProvider error
  And RunContext.Metadata["branch_name"] is not set
```

### Scenario 5: Action is registered in ActionRegistry

```gherkin
Given the application has started
When ActionRegistry.Get("git_branch") is called
Then the GitBranchAction is returned without error
```

### Scenario 6: Unit tests pass lint

```gherkin
Given the implementation in backend/internal/adapter/action/git_branch.go
When "golangci-lint run ./..." is executed from the backend/ directory
Then the lint check exits 0
```

## Tasks / Subtasks

- [ ] [BACK] Task 1: Implement GitBranchAction (AC: #1, #2, #3, #4)
  - [ ] Create `backend/internal/adapter/action/git_branch.go`
  - [ ] Define `GitBranchAction` struct with fields: `gitProvider port.GitProvider`, `logger *slog.Logger`
  - [ ] Implement `NewGitBranchAction(gitProvider port.GitProvider, logger *slog.Logger) *GitBranchAction`
  - [ ] Implement `Name() string` returning `"git_branch"`
  - [ ] Implement `Execute(ctx context.Context, runCtx *model.RunContext) error`:
    - Read `branch_pattern` from `runCtx.RunStep` config (via step config map); default to `"feat/{story_key}-{slug}"` if absent
    - Read `base_branch` from step config; default to `"main"` if absent
    - Read `work_dir` from step config or `runCtx.Metadata["work_dir"]`; error if missing
    - Fetch story from `storyRepo.GetByID(ctx, runCtx.StoryID)` to get `story.Key` and `story.Title`
    - Derive slug: lowercase `story.Title`, replace non-alphanumeric chars with `-`, collapse repeated `-`, trim leading/trailing `-`
    - Render branch name by replacing `{story_key}` and `{slug}` in pattern
    - Call `gitProvider.CreateBranch(ctx, workDir, branchName)`
    - On success: set `runCtx.Metadata["branch_name"] = branchName` and return nil
    - On error: return `fmt.Errorf("create branch %q: %w", branchName, err)`
  - [ ] Add `storyRepo port.StoryRepository` dependency to struct and constructor

- [ ] [BACK] Task 2: Implement slug helper (AC: #2)
  - [ ] Create private func `slugify(title string) string` in the same file
  - [ ] Uses `strings.ToLower`, `regexp.MustCompile("[^a-z0-9]+").ReplaceAllString`, `strings.Trim`
  - [ ] Compile regex at package level with `var nonAlphanumeric = regexp.MustCompile("[^a-z0-9]+")`

- [ ] [BACK] Task 3: Extend step config access (AC: #1, #3)
  - [ ] The step config is stored in `runCtx.RunStep.Action` (action type string) and the config map is available on the `PipelineStep` embedded in `RunContext` — check where step config is currently accessible (likely via `runCtx.RunStep` extended fields or a `Config map[string]string` on `RunStep`); add if missing
  - [ ] If `RunStep` does not carry a `Config map[string]string`, add it to `model.RunStep` and populate it in `PipelineExecutor` from `PipelineStep.Config`

- [ ] [BACK] Task 4: Register GitBranchAction in ActionRegistry (AC: #5)
  - [ ] In `backend/cmd/api/main.go` (or the wire setup file), instantiate `NewGitBranchAction` with required deps
  - [ ] Call `actionRegistry.Register(gitBranchAction)`

- [ ] [BACK] Task 5: Write unit tests (AC: #1, #2, #3, #4, #6)
  - [ ] Create `backend/internal/adapter/action/__tests__/git_branch_test.go`
  - [ ] **Test: happy path** — mock GitProvider.CreateBranch succeeds, verify `runCtx.Metadata["branch_name"]` is set correctly
  - [ ] **Test: slug derivation** — table-driven tests for various story titles (spaces, special chars, unicode-like)
  - [ ] **Test: default base branch** — no `base_branch` config key, verify GitProvider called with `"main"`
  - [ ] **Test: custom pattern** — `fix/{story_key}-{slug}`, verify rendered branch name
  - [ ] **Test: GitProvider failure** — verify error wrapping, metadata not set
  - [ ] All mocks hand-written implementing `port.GitProvider` and `port.StoryRepository`
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

- **R-1-3 (new action_types validated) — required:** The `git_branch` action type must be present in the `PipelineStep.ActionType` enum and the config map (`PipelineStep.Config`) must be plumbed through to `RunContext`/`RunStep` by the pipeline executor.
- **Story 3-2 (GitProvider port + gh CLI adapter) — DONE:** `GitProvider.CreateBranch(ctx, workDir, branchName)` is already implemented and available.
- **Story 3-7 (Pipeline executor) — DONE:** `model.Action` interface, `RunContext`, and `ActionRegistry` are in place.

### Architecture Requirements

- `GitBranchAction` lives in `backend/internal/adapter/action/` — it is an adapter implementing the domain `model.Action` interface.
- No business logic outside of the adapter: slug derivation is a pure string transformation, not domain logic.
- Import direction: `adapter/action → domain/port ← adapter/github`. No cross-adapter imports.
- The `workDir` is the cloned repository path set earlier in the pipeline (by the agent container setup or a dedicated clone step). It is passed via `runCtx.Metadata["work_dir"]` or via step config.
- Metadata key `branch_name` must be set on success so that downstream `git_pr` and `agent_run` actions can read it.

### Technical Specifications

**File:** `backend/internal/adapter/action/git_branch.go`

```go
package action

import (
    "context"
    "fmt"
    "log/slog"
    "regexp"
    "strings"

    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// GitBranchAction implements model.Action for creating a Git feature branch.
// It renders the branch name from a configurable pattern using RunContext variables,
// delegates creation to GitProvider, and stores the result in RunContext.Metadata.
type GitBranchAction struct {
    gitProvider port.GitProvider
    storyRepo   port.StoryRepository
    logger      *slog.Logger
}

// NewGitBranchAction creates a new GitBranchAction.
func NewGitBranchAction(gitProvider port.GitProvider, storyRepo port.StoryRepository, logger *slog.Logger) *GitBranchAction {
    return &GitBranchAction{
        gitProvider: gitProvider,
        storyRepo:   storyRepo,
        logger:      logger,
    }
}

// Name returns the action identifier.
func (a *GitBranchAction) Name() string { return "git_branch" }

func (a *GitBranchAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    cfg := runCtx.RunStep.Config // map[string]string

    pattern, ok := cfg["branch_pattern"]
    if !ok || pattern == "" {
        pattern = "feat/{story_key}-{slug}"
    }
    baseBranch, ok := cfg["base_branch"]
    if !ok || baseBranch == "" {
        baseBranch = "main"
    }
    workDir, ok := cfg["work_dir"]
    if !ok || workDir == "" {
        if wd, wdOK := runCtx.Metadata["work_dir"].(string); wdOK && wd != "" {
            workDir = wd
        } else {
            return fmt.Errorf("git_branch: work_dir not configured and not in metadata")
        }
    }

    story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
    if err != nil {
        return fmt.Errorf("fetch story: %w", err)
    }

    slug := slugify(story.Title)
    branchName := strings.ReplaceAll(pattern, "{story_key}", story.Key)
    branchName = strings.ReplaceAll(branchName, "{slug}", slug)

    a.logger.Info("creating branch", "branch", branchName, "base", baseBranch, "story_key", story.Key)

    if err := a.gitProvider.CreateBranch(ctx, workDir, branchName); err != nil {
        return fmt.Errorf("create branch %q: %w", branchName, err)
    }

    runCtx.Metadata["branch_name"] = branchName
    return nil
}

// slugify converts a story title to a URL-safe lowercase slug.
func slugify(title string) string {
    lower := strings.ToLower(title)
    slug := nonAlphanumeric.ReplaceAllString(lower, "-")
    return strings.Trim(slug, "-")
}
```

**Step config access:** `RunStep.Config` must be a `map[string]string` populated by the pipeline executor from `PipelineStep.Config` before calling `action.Execute`. If this field does not yet exist on `model.RunStep`, add it and populate it in `PipelineExecutor.executeStep`.

**Metadata contract:**
- Reads: `work_dir` (optional, fallback to step config)
- Writes: `branch_name` (string — the rendered branch name)

**Error codes produced:**
- `STORY_NOT_FOUND` — story not in DB (from StoryRepository)
- `GIT_BRANCH_FAILED` — GitProvider returned error (wrapped)

### Testing Requirements

File: `backend/internal/adapter/action/__tests__/git_branch_test.go`

**Tests:**

1. **Happy path** — `branch_pattern = "feat/{story_key}-{slug}"`, story key `"S-03"`, title `"Add login page"` → branch `"feat/S-03-add-login-page"`, `Metadata["branch_name"]` set.
2. **Slug with special chars** — table-driven: `"Add login (OAuth)"` → `"add-login-oauth"`, `"Hello World!"` → `"hello-world"`, `"  spaces  "` → `"spaces"`.
3. **Default base branch** — no `base_branch` in config → GitProvider called (verify no error; base branch used is internal to `CreateBranch` args).
4. **Custom fix pattern** — `branch_pattern = "fix/{story_key}-{slug}"` → branch starts with `"fix/"`.
5. **GitProvider failure** — `CreateBranch` returns error → action returns error, `Metadata["branch_name"]` not set.
6. **Story not found** — `storyRepo.GetByID` returns not-found error → action returns error, no GitProvider call.
7. **Missing work_dir** — neither in config nor metadata → action returns descriptive error.

```go
type MockGitProvider struct {
    CreateBranchFn func(ctx context.Context, workDir, branchName string) error
}
func (m *MockGitProvider) CreateBranch(_ context.Context, workDir, branchName string) error {
    return m.CreateBranchFn(context.Background(), workDir, branchName)
}
// Implement remaining GitProvider methods as stubs returning nil/zero values.
```

Run `golangci-lint run ./...` before committing — must pass.

### References

- `backend/internal/domain/port/git_provider.go` — `GitProvider.CreateBranch` signature
- `backend/internal/domain/model/run_context.go` — `RunContext.Metadata`
- `backend/internal/domain/model/run.go` — `RunStep` struct (add `Config map[string]string` if missing)
- `backend/internal/adapter/action/hitl_gate.go` — reference action pattern
- `backend/internal/domain/service/action_registry.go` — `Register(action)` method
- Story R-1-3 — wires step config through pipeline executor to RunStep

## Dev Agent Record

## Change Log

- 2026-02-23: Story created for Wave R implementation
