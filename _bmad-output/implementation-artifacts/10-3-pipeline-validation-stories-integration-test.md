# Story 10.3: [SHARED] Pipeline validation — stories + integration test

Status: ready-for-dev

## Story

As a platform developer, I want integration tests using the reference project, So that I can verify the entire pipeline works end-to-end.

## Acceptance Criteria (BDD)

**AC1: Reference project contains sample stories in frontmatter markdown format**
- **Given** the `test-project/stories/` directory
- **When** the directory is reviewed
- **Then** it contains 5 markdown files: `todo-s1.md`, `todo-s2.md`, `todo-s3.md`, `todo-s4.md`, `todo-s5.md`
- **And** each file has YAML frontmatter with fields: `key`, `title`, `epic_key`, `depends_on` (array), `scope`, `status`
- **And** todo-s1 has no dependencies, todo-s2 and todo-s3 depend on todo-s1, todo-s4 depends on todo-s2, todo-s5 depends on todo-s3 and todo-s4

**AC2: Story files define realistic backend and frontend tasks**
- **Given** a story file (e.g., `todo-s1.md`)
- **When** the story is reviewed
- **Then** it contains a user story, acceptance criteria, and tasks/subtasks
- **And** todo-s1, s2, s3 have `scope: backend`
- **And** todo-s4 has `scope: frontend`
- **And** todo-s5 has `scope: shared` (E2E test)
- **And** the description and tasks are realistic for a todo application (e.g., "Add todo endpoint", "Add list todos endpoint")

**AC3: Integration test suite exists in backend with proper structure**
- **Given** a test file at `backend/internal/integration/pipeline_validation_test.go`
- **When** the file is reviewed
- **Then** it is guarded with `//go:build integration` build tag
- **And** it contains test functions: `TestPipelineValidation_FullRun`, `TestPipelineValidation_DAGBuild`, `TestPipelineValidation_ContainerIntegration`
- **And** the test uses `testcontainers-go` to spin up ephemeral Postgres
- **And** the test uses a mock AgentRuntime instead of a real Claude agent

**AC4: Integration test validates story import and DAG build**
- **Given** the integration test with stories loaded
- **When** `TestPipelineValidation_DAGBuild` runs
- **Then** all 5 stories are imported from `test-project/stories/` into the test database
- **And** `SchedulerService.BuildDAG` is called successfully (no cycles)
- **And** DAG has 3 layers: layer 0 has S1, layer 1 has S2 and S3, layer 2 has S4, layer 3 has S5
- **And** the test validates the group structure matches expected dependencies

**AC5: Integration test validates container launch and mock agent recording**
- **Given** the integration test with a mock AgentRuntime
- **When** `TestPipelineValidation_ContainerIntegration` runs
- **Then** for each story, a `Run` is created and `PipelineExecutor.ExecuteRun` is invoked
- **And** the mock AgentRuntime records: container name, environment variables, network config, volumes mounted
- **And** the mock does NOT actually launch Claude — it records the call and returns success
- **And** the test verifies each recorded container has correct naming (e.g., `hopeitworks-story-todo-s1-...`)
- **And** the test verifies env vars include: `PROJECT_ID`, `STORY_ID`, `STORY_KEY`, `RUN_ID`, `PIPELINE_CONFIG`

**AC6: Integration test validates full pipeline from story import to completion**
- **Given** the integration test with all components wired
- **When** `TestPipelineValidation_FullRun` runs (this is the MVP acceptance gate)
- **Then** the test executes these phases in order:
  1. Story import: all 5 stories from `test-project/stories/` loaded into DB
  2. DAG build: SchedulerService validates DAG, no cycles, correct groups
  3. Epic run creation: EpicRunService launches an epic run for all stories
  4. Parallel execution: ParallelGroupExecutor runs layers sequentially, stories within a layer in parallel
  5. Container launch: Each story spawns a mocked agent container (no real Claude)
  6. Pipeline completion: All containers complete successfully
  7. Run records: All Run records transition to `completed`, events published
- **And** the test asserts: epic run status is `completed`, all story runs have status `completed`, total runtime is reasonable (no timeout after 30s)

**AC7: Mock AgentRuntime captures and validates wiring**
- **Given** a mock implementation of port.AgentRuntime
- **When** `LaunchContainer` is called during a story run
- **Then** the mock records: `ContainerName`, `Image`, `Env` (map), `Mounts` (volumes), `Networks`
- **And** the mock returns a fake container ID
- **And** a `WaitContainer` call on the fake ID returns: `exitCode: 0`, `logs: "<step output>"`, no error
- **And** the test can inspect recorded calls via a slice/channel: `mock.Calls()` or similar
- **And** the test validates correct project/story/run context passed in env vars

**AC8: test-project README documents the validation flow**
- **Given** a README at `test-project/README.md`
- **When** the file is reviewed
- **Then** it explains:
  - Purpose: reference todo app for pipeline validation
  - Directory structure: `stories/`, `backend/`, `frontend/`, `.gitignore`
  - Story format: YAML frontmatter markdown
  - How to run the integration test: `go test ./internal/integration/... -tags=integration`
  - What the test validates: story import, DAG build, parallel execution, mock agent recording
  - How to add new sample stories: template and instructions
  - Notes on the mock agent: why it's used (avoid real Claude costs in CI), how to switch to real agent for manual testing

**AC9: Integration test skips gracefully when testcontainers unavailable**
- **Given** a test environment where Docker/testcontainers is unavailable
- **When** `TestPipelineValidation_FullRun` runs
- **Then** the test checks if testcontainers is available via `os.LookupEnv("TESTCONTAINERS_DOCKER_SOCKET")` or similar
- **And** if unavailable, the test is skipped with a descriptive message: "testcontainers Docker not available"
- **And** the test is NOT marked as failed

**AC10: Unit tests for StoryParser verify frontmatter parsing**
- **Given** a StoryParser in `backend/internal/testutil/story_parser.go` (or domain service for frontmatter)
- **When** tests are executed
- **Then** there are tests in `backend/internal/testutil/story_parser_test.go` (or `backend/internal/domain/service/story_parser_test.go`)
- **And** tests validate: parsing valid frontmatter, handling missing fields, extracting depends_on array, scope normalization
- **And** tests verify error handling: malformed YAML, missing key/title, invalid scope

## Tasks / Subtasks

- [ ] [SHARED] Task 1: Create sample stories in test-project/stories/ (AC: #1, #2)
  - [ ] Create `test-project/stories/todo-s1.md`:
    - Key: `S-1`, Epic: `10`, Scope: `backend`, Status: `ready-for-dev`, No deps
    - Title: "Add todo endpoint"
    - User story: "As an API user, I want to create a new todo item via POST /todos, so I can track tasks"
    - Acceptance criteria: Endpoint accepts POST with JSON body, returns 201 with created todo, validates title field
    - Tasks: Define Todo model, implement POST /todos handler, add unit tests
  - [ ] Create `test-project/stories/todo-s2.md`:
    - Key: `S-2`, Epic: `10`, Scope: `backend`, Status: `ready-for-dev`, Depends: `[S-1]`
    - Title: "Add list todos endpoint"
    - Similar format, accepts GET /todos, returns 200 with array
  - [ ] Create `test-project/stories/todo-s3.md`:
    - Key: `S-3`, Epic: `10`, Scope: `backend`, Status: `ready-for-dev`, Depends: `[S-1]`
    - Title: "Add delete todo endpoint"
    - DELETE /todos/{id}
  - [ ] Create `test-project/stories/todo-s4.md`:
    - Key: `S-4`, Epic: `10`, Scope: `frontend`, Status: `ready-for-dev`, Depends: `[S-2]`
    - Title: "Frontend todo list component"
    - Vue 3 component to display todos from API
  - [ ] Create `test-project/stories/todo-s5.md`:
    - Key: `S-5`, Epic: `10`, Scope: `shared`, Status: `ready-for-dev`, Depends: `[S-3, S-4]`
    - Title: "E2E smoke test"
    - Playwright test: create todo, list todos, delete todo, verify UI updates

- [ ] [BACK] Task 2: Create StoryParser utility for frontmatter parsing (AC: #1, #10)
  - [ ] Create `backend/internal/testutil/story_parser.go`:
    - `ParseStoryFile(filePath string) (*model.Story, error)` — reads markdown, extracts frontmatter, parses YAML, populates Story model
    - `LoadStoriesFromDirectory(dirPath string) ([]*model.Story, error)` — walks directory, calls ParseStoryFile for each .md file
    - Handle error cases: missing file, invalid YAML, missing required fields (key, title, scope)
    - Return `DomainError` with code `INVALID_STORY_FORMAT` for parsing errors
  - [ ] Create `backend/internal/testutil/story_parser_test.go`:
    - Test valid story file parsing
    - Test missing frontmatter
    - Test missing required fields (key, title, epic_key, scope)
    - Test depends_on array parsing (single dep, multiple deps, empty)
    - Test invalid YAML in frontmatter
    - Verify Story model fields populated correctly

- [ ] [BACK] Task 3: Create MockAgentRuntime for testing (AC: #5, #7)
  - [ ] Create `backend/internal/testutil/mock_agent_runtime.go`:
    - Implement `port.AgentRuntime` interface
    - `LaunchContainer(ctx context.Context, config *model.ContainerConfig) (*model.Container, error)`:
      - Record call: `ContainerName`, `Image`, `Env`, `Mounts`, `Networks`
      - Return fake container ID (e.g., UUID or "mock-container-{counter}")
      - Do NOT call Docker or testcontainers
    - `WaitContainer(ctx context.Context, containerID string) (*model.ContainerResult, error)`:
      - Return `exitCode: 0`, `logs: "mocked output"`, no error
    - `CleanupContainer(ctx context.Context, containerID string) error`:
      - Return nil
    - `Calls() []ContainerCall` — return slice of recorded calls for test inspection
    - `Reset()` — clear recorded calls
  - [ ] Define `ContainerCall` struct: `ID`, `Name`, `Image`, `Env map[string]string`, `Mounts`, `Networks`, `Timestamp`

- [ ] [BACK] Task 4: Set up integration test infrastructure (AC: #3, #6, #9)
  - [ ] Create `backend/internal/integration/pipeline_validation_test.go`:
    - Add `//go:build integration` build tag at top
    - Import: `testing`, `context`, `testcontainers` (testcontainers-go), `internal/testutil`, `internal/domain/service`, `internal/adapter/postgres`, `internal/domain/model`
    - `setupTestDB(t *testing.T) (*sql.DB, func())` — uses testcontainers-go to spin up Postgres, runs migrations, returns DB and cleanup func
    - Check Docker availability; if unavailable, skip tests with `t.Skip("testcontainers Docker not available")`
    - All tests use `-short` tag but tagged with `//go:build integration` so excluded from default runs

- [ ] [BACK] Task 5: Implement TestPipelineValidation_DAGBuild (AC: #4)
  - [ ] Load 5 sample stories from embedded or hardcoded test fixtures (or use `testutil.LoadStoriesFromDirectory` pointing to a test fixtures directory)
  - [ ] Insert stories into test DB via `StoryRepository.CreateStory` or batch insert
  - [ ] Call `SchedulerService.BuildDAG(stories)` — verify no error
  - [ ] Assert DAG structure:
    - `dag.Groups[0]` contains only S1
    - `dag.Groups[1]` contains S2 and S3 (no specific order)
    - `dag.Groups[2]` contains S4
    - `dag.Groups[3]` contains S5
  - [ ] Log results for debugging

- [ ] [BACK] Task 6: Implement TestPipelineValidation_ContainerIntegration (AC: #5, #7)
  - [ ] Use mock AgentRuntime (NOT real testcontainers)
  - [ ] Load stories, create an EpicRun via EpicRunService
  - [ ] Call `ParallelGroupExecutor.Execute` with mock runtime
  - [ ] Inspect mock runtime calls:
    - Assert 5 calls total (one per story)
    - For each call, verify: ContainerName starts with `hopeitworks-story-`, Image is set, Env includes `PROJECT_ID`, `STORY_ID`, `STORY_KEY`, `RUN_ID`
    - Verify networks include correct bridge/overlay for containerized runs

- [ ] [BACK] Task 7: Implement TestPipelineValidation_FullRun (AC: #6)
  - [ ] Full MVP acceptance gate test — validates entire pipeline end-to-end
  - [ ] Steps:
    1. Setup test DB with testcontainers (or skip if unavailable)
    2. Load 5 sample stories into DB
    3. Create EpicRun via EpicRunService.LaunchEpicRun
    4. Wire all services: StoryRepository, EpicRunRepository, RunRepository, SchedulerService, PipelineExecutor, ParallelGroupExecutor
    5. Wire mock AgentRuntime
    6. Call ParallelGroupExecutor.Execute with the epic run
    7. Wait for completion (30s timeout)
    8. Assert epic run status is `completed`
    9. Assert all Run records have status `completed`
    10. Assert all events published (check EventPublisher mock or DB event log)
    11. Assert mock runtime recorded 5 container launches
    12. Log full trace for debugging

- [ ] [BACK] Task 8: Add integration test helpers and fixtures (AC: #3, #4)
  - [ ] Create `backend/internal/testutil/fixtures.go`:
    - `LoadTestStories()` — returns [5]*model.Story with todo-s1...s5 populated (can be embedded YAML or Go structs)
    - `BuildEpicForStories(stories ...*model.Story) *model.Epic` — creates an Epic with given stories linked
  - [ ] Create `backend/internal/testutil/test_db.go` (if not already exists):
    - `NewTestDB(t *testing.T) (*sql.DB, func())` — testcontainers Postgres + migrations, cleanup
    - Must handle testcontainers availability check gracefully

- [ ] [BACK] Task 9: Update test-project/README.md (AC: #8)
  - [ ] Create or update `test-project/README.md`:
    - Section: Overview — purpose, what the pipeline validates
    - Section: Directory Structure — stories/, backend/, frontend/, config
    - Section: Story Format — explain YAML frontmatter (key, title, epic_key, depends_on, scope, status)
    - Section: Running the Integration Test:
      ```bash
      cd backend
      go test ./internal/integration/... -tags=integration
      ```
    - Section: What the Test Validates — story import, DAG build, parallel execution, agent recording
    - Section: Adding New Stories — template + step-by-step
    - Section: Using Real Claude Agent — notes on switching from mock to real (requires CLAUDE_CODE_OAUTH_TOKEN)
    - Section: Troubleshooting — testcontainers setup, Docker socket access

- [ ] [BACK] Task 10: Lint, test, and validate (AC: all)
  - [ ] Run `cd backend && golangci-lint run ./...` — fix all errors
  - [ ] Run `cd backend && go test ./... -short` — all unit tests pass
  - [ ] Run `cd backend && go test ./internal/integration/... -tags=integration` — integration tests pass (may skip if Docker unavailable)
  - [ ] Verify no `fmt.Println`, hardcoded secrets, commented code
  - [ ] Verify all new types have godoc comments
  - [ ] Ensure testcontainers gracefully skips on Docker unavailable

## Dev Notes

### Dependencies

**Story 10-2 (todo app CI):** The reference project's CI/CD configuration and build pipeline must be set up in story 10-2. This story assumes the project builds and its CI passes.

**Story 3-1 (Runs & RunSteps):** Run and RunStep models, RunRepository, state machine transitions available.

**Story 3-7 (PipelineExecutor):** `PipelineExecutor.ExecuteRun(ctx, runID uuid.UUID) error` available.

**Story 7-1 (SchedulerService):** `SchedulerService.BuildDAG(stories []*Story) (*DAGResult, error)` available — detects cycles, returns layer groups.

**Story 7-2 (EpicRunService & ParallelGroupExecutor):** `EpicRunService.LaunchEpicRun`, `ParallelGroupExecutor.Execute` available.

**testcontainers-go:** Must be in `backend/go.mod`. If not, run `go get github.com/testcontainers/testcontainers-go@latest`.

### Architecture Requirements

- Integration tests are **acceptance gates** — if they pass, the platform works end-to-end
- Mock AgentRuntime is used in tests to **avoid real Claude costs in CI**. For manual validation, can swap to real agent by setting `CLAUDE_CODE_OAUTH_TOKEN` and updating wire.go to use real ContainerManager + AgentRuntime
- `StoryParser` is a testutil — not a production service (production story parsing would be different)
- Mock runtime records calls in-memory; test inspects calls for validation
- Tests are tagged `//go:build integration` — not run by default, only with `-tags=integration`
- Tests skip gracefully if testcontainers Docker is unavailable (CI may have it, local dev may not)

### File Paths (exact)

```
test-project/README.md                                            # Validation docs
test-project/stories/todo-s1.md                                   # Sample story 1
test-project/stories/todo-s2.md                                   # Sample story 2
test-project/stories/todo-s3.md                                   # Sample story 3
test-project/stories/todo-s4.md                                   # Sample story 4
test-project/stories/todo-s5.md                                   # Sample story 5
backend/internal/testutil/story_parser.go                         # Frontmatter parser
backend/internal/testutil/story_parser_test.go                    # Parser tests
backend/internal/testutil/mock_agent_runtime.go                   # Mock runtime
backend/internal/testutil/fixtures.go                             # Test fixtures
backend/internal/testutil/test_db.go                              # Test DB setup (may already exist)
backend/internal/integration/pipeline_validation_test.go          # Integration test suite
```

### Story Frontmatter Template

All stories in `test-project/stories/` follow this format:

```markdown
---
key: S-N
title: "Story Title"
epic_key: "10"
depends_on: ["S-M"]  # or [] for no deps
scope: "backend|frontend|shared"
status: "ready-for-dev"
---

## Story

As a [role], I want [capability], so that [benefit].

## Acceptance Criteria

[Standard AC format]

## Tasks

- [ ] Task 1
- [ ] Task 2
```

### Mock AgentRuntime Pattern

```go
// backend/internal/testutil/mock_agent_runtime.go
type MockAgentRuntime struct {
    calls []ContainerCall
    mu    sync.Mutex
}

func (m *MockAgentRuntime) LaunchContainer(ctx context.Context, cfg *model.ContainerConfig) (*model.Container, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    call := ContainerCall{
        Name:      cfg.Name,
        Image:     cfg.Image,
        Env:       cfg.Env,
        Mounts:    cfg.Mounts,
        Networks:  cfg.Networks,
        Timestamp: time.Now(),
    }
    m.calls = append(m.calls, call)

    return &model.Container{
        ID:   fmt.Sprintf("mock-container-%d", len(m.calls)),
        Name: cfg.Name,
    }, nil
}

func (m *MockAgentRuntime) WaitContainer(ctx context.Context, containerID string) (*model.ContainerResult, error) {
    return &model.ContainerResult{ExitCode: 0, Logs: "mocked output"}, nil
}

func (m *MockAgentRuntime) CleanupContainer(ctx context.Context, containerID string) error {
    return nil
}

func (m *MockAgentRuntime) Calls() []ContainerCall {
    m.mu.Lock()
    defer m.mu.Unlock()
    return append([]ContainerCall{}, m.calls...)
}

func (m *MockAgentRuntime) Reset() {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.calls = nil
}
```

### Integration Test Structure

```go
// backend/internal/integration/pipeline_validation_test.go
//go:build integration

package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestPipelineValidation_DAGBuild(t *testing.T) {
    // Setup test DB
    db, cleanup := setupTestDB(t)
    defer cleanup()

    // Load stories, insert into DB
    // Call SchedulerService.BuildDAG
    // Assert DAG structure
}

func TestPipelineValidation_ContainerIntegration(t *testing.T) {
    // Setup test DB
    // Setup mock AgentRuntime
    // Load stories, create epic run
    // Call ParallelGroupExecutor.Execute with mock
    // Inspect mock calls
}

func TestPipelineValidation_FullRun(t *testing.T) {
    // MVP acceptance gate
    // Complete end-to-end flow
}

func setupTestDB(t *testing.T) (*sql.DB, func()) {
    // testcontainers Postgres + migrations
    // Skip if Docker unavailable
}
```

### Testing Requirements

**Integration test (backend/internal/integration/pipeline_validation_test.go):**

1. **DAGBuild test:**
   - Load 5 stories with correct dependencies
   - Build DAG, verify 4 layers
   - Assert no cycles

2. **ContainerIntegration test:**
   - Mock runtime records calls
   - Verify 5 containers launched
   - Verify env vars and naming correct

3. **FullRun test (MVP acceptance gate):**
   - Complete pipeline: import → DAG → epic run → execute → complete
   - Verify epic run status is `completed`
   - Verify all Run records completed
   - Verify events published
   - All within 30s timeout

**Unit tests (backend/internal/testutil/story_parser_test.go):**

1. Valid story file parsing
2. Missing fields
3. depends_on array
4. Invalid YAML
5. Error handling

### Validation Checklist (MVP Acceptance)

If this test passes (TestPipelineValidation_FullRun), the platform works end-to-end:

- [ ] 5 sample stories loaded from `test-project/stories/`
- [ ] DAG built with correct layer structure (no cycles)
- [ ] EpicRun created for all stories
- [ ] ParallelGroupExecutor runs layers sequentially, stories in parallel
- [ ] Mock agent records all 5 container launches
- [ ] Each container wired with correct project/story/run context
- [ ] All Run records transition to `completed`
- [ ] All events published in correct order
- [ ] Epic run transitions to `completed`
- [ ] No timeout or cancellation

### References

- Story 3-1: Runs & RunSteps tables + state machine
- Story 3-7: PipelineExecutor (sequential step runner)
- Story 7-1: SchedulerService (DAG builder)
- Story 7-2: EpicRunService & ParallelGroupExecutor
- Story 10-2: Reference todo app CI
- testcontainers-go: https://golang.testcontainers.org/
- Epic 10 Planning: Reference project & validation

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-18: Story created for Wave 15 reference validation
