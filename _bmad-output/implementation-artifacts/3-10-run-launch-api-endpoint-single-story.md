# Story 3.10: [BACK] Run launch API endpoint (single story)

Status: ready-for-dev

## Story

As a frontend developer,
I want an API endpoint to launch a pipeline run for a single story,
So that users can trigger story execution from the UI.

## Acceptance Criteria (BDD)

**AC1: Successful run launch**
- **Given** I am authenticated with access to the project
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 201 is returned with the created run object (status = `pending`) and its steps
- **And** a River job is enqueued (within the same DB transaction) to execute the run asynchronously

**AC2: Conflict — story already running**
- **Given** the story has an existing run with status `running` (or `pending`)
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 409 is returned with error code `STORY_ALREADY_RUNNING`

**AC3: Bad request — story already completed**
- **Given** the story has status `done`
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 400 is returned with error code `STORY_ALREADY_COMPLETED`

**AC4: Forbidden — no project access**
- **Given** the authenticated user does not have access to the project
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 403 is returned

**AC5: Not found — story does not exist**
- **Given** the `storyId` does not exist in the project
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 404 is returned with error code `STORY_NOT_FOUND`

**AC6: Not found — no pipeline config**
- **Given** the project has no pipeline configuration
- **When** I POST to `/api/v1/projects/{projectId}/stories/{storyId}/runs`
- **Then** HTTP 404 is returned with error code `PIPELINE_CONFIG_NOT_FOUND`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add `GetActiveRunByStory` query and port method (AC: #2)
  - [ ] Add SQL query `GetActiveRunByStory` to `backend/queries/runs.sql` selecting runs with status `IN ('pending', 'running')` for a given `story_id`
  - [ ] Run `cd backend && sqlc generate` to regenerate `internal/adapter/postgres/db/`
  - [ ] Add `GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error)` to `port.RunRepository` interface in `backend/internal/domain/port/run_repository.go`
  - [ ] Implement the method in `backend/internal/adapter/postgres/run_repo.go`

- [ ] [BACK] Task 2: Define `JobQueue` port and River adapter (AC: #1)
  - [ ] Add `JobQueue` interface to `backend/internal/domain/port/job_queue.go`
  - [ ] Add `riverqueue` dependency: `go get github.com/riverqueue/river@latest && go get github.com/riverqueue/river/riverdriver/riverpgxv5@latest`
  - [ ] Define `ExecuteRunArgs` River job struct in `backend/internal/adapter/river/execute_run_job.go`
  - [ ] Implement `RiverJobQueue` adapter (in-process client, `Insert` within same pgx transaction) in `backend/internal/adapter/river/job_queue.go`

- [ ] [BACK] Task 3: Implement `LaunchRun` in `RunService` (AC: #1, #2, #3, #5, #6)
  - [ ] Add `storyRepo port.StoryRepository`, `pipelineConfigRepo port.PipelineConfigRepository`, and `jobQueue port.JobQueue` to `RunService` struct and `NewRunService` constructor
  - [ ] Implement `LaunchRun(ctx context.Context, projectID, storyID uuid.UUID) (*model.Run, error)` method in `backend/internal/domain/service/run_service.go`
  - [ ] Write unit tests for `LaunchRun` in `backend/internal/domain/service/run_service_test.go`

- [ ] [BACK] Task 4: Add `LaunchRun` handler method to `RunHandler` (AC: #1, #4)
  - [ ] Implement `LaunchRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath)` in `backend/internal/api/handler/run_handler.go`
  - [ ] Register the method in `backend/internal/api/handler/server.go`
  - [ ] Write unit tests for the handler in `backend/internal/api/handler/run_handler_test.go`

- [ ] [BACK] Task 5: Update OpenAPI spec and regenerate (AC: #1)
  - [ ] Add `POST /projects/{projectId}/stories/{storyId}/runs` endpoint to `api/openapi.yaml`
  - [ ] Run `cd backend && make generate` to regenerate `internal/api/handler/gen_server.go`

- [ ] [BACK] Task 6: Wire DI and run lint + tests (AC: all)
  - [ ] Update `backend/cmd/api/wire.go` to inject `JobQueue` and updated `RunService`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`
  - [ ] Run `cd backend && golangci-lint run ./...` — must pass
  - [ ] Run `cd backend && go test ./... -short` — must pass

## Dev Notes

### Dependencies

- Story 3-1 (DONE): `RunRepository` port and DB tables (`runs`, `run_steps`) exist; `RunService` is implemented with `CreateRun`, `GetRun`, `ListRunsByProject`, `ListRunsByStory`, `TransitionRun`
- Story 3-7 (DONE): `PipelineExecutor.ExecuteRun` exists and is callable by the River worker

### Architecture Requirements

- `LaunchRun` must execute the following within a **single DB transaction** (Transactor pattern):
  1. Validate story exists and is launchable
  2. Check no active run exists for the story
  3. Fetch pipeline config for the project
  4. Parse YAML and create `Run` + `RunStep` records (status = `pending`)
  5. Enqueue a River job referencing the new `run.ID`
- The River worker is a separate goroutine (registered at startup) that calls `PipelineExecutor.ExecuteRun(ctx, runID)` upon job pickup
- Do NOT add a `Transactor` port in this story — use the River `Insert`-within-transaction pattern directly via the `JobQueue` port's `EnqueueTx` method

### File Paths (exact)

| File | Action |
|------|--------|
| `backend/queries/runs.sql` | Add `GetActiveRunByStory` query |
| `backend/internal/adapter/postgres/db/` | Regenerated by sqlc — do not edit manually |
| `backend/internal/domain/port/run_repository.go` | Add `GetActiveRunByStory` to interface |
| `backend/internal/domain/port/job_queue.go` | New file — `JobQueue` interface |
| `backend/internal/adapter/river/execute_run_job.go` | New file — job struct + worker |
| `backend/internal/adapter/river/job_queue.go` | New file — `RiverJobQueue` adapter |
| `backend/internal/domain/service/run_service.go` | Add `LaunchRun` method; extend constructor |
| `backend/internal/domain/service/run_service_test.go` | Add `LaunchRun` tests |
| `backend/internal/api/handler/run_handler.go` | Add `LaunchRun` handler |
| `backend/internal/api/handler/run_handler_test.go` | New/extended test file |
| `backend/internal/api/handler/server.go` | Register `LaunchRun` delegation |
| `api/openapi.yaml` | Add new endpoint |
| `backend/cmd/api/wire.go` | Update DI wiring |
| `backend/cmd/api/wire_gen.go` | Regenerated by wire — do not edit manually |

### Technical Specifications

#### SQL query — `backend/queries/runs.sql`

```sql
-- name: GetActiveRunByStory :one
SELECT * FROM runs
WHERE story_id = $1 AND status IN ('pending', 'running')
ORDER BY created_at DESC
LIMIT 1;
```

#### Port — `backend/internal/domain/port/job_queue.go`

```go
package port

import "context"

// JobQueue defines the interface for enqueuing async background jobs.
type JobQueue interface {
    // Enqueue inserts a job using an autocommit connection.
    Enqueue(ctx context.Context, job any) error
    // EnqueueTx inserts a job within an existing pgx transaction.
    // tx must be a *pgx.Tx or pgx.Tx-compatible value.
    EnqueueTx(ctx context.Context, tx any, job any) error
}
```

#### Port addition — `backend/internal/domain/port/run_repository.go`

Add to the `RunRepository` interface:

```go
// GetActiveRunByStory returns the most recent pending or running run for a story, or nil if none.
GetActiveRunByStory(ctx context.Context, storyID uuid.UUID) (*model.Run, error)
```

#### River job struct + worker — `backend/internal/adapter/river/execute_run_job.go`

```go
package river

import (
    "context"

    "github.com/google/uuid"
    "github.com/riverqueue/river"

    "github.com/zakari/hopeitworks/backend/internal/domain/service"
)

// ExecuteRunArgs is the River job payload for pipeline execution.
type ExecuteRunArgs struct {
    RunID uuid.UUID `json:"run_id"`
}

// Kind returns the unique job kind identifier used by River.
func (ExecuteRunArgs) Kind() string { return "execute_run" }

// ExecuteRunWorker processes execute_run jobs by calling PipelineExecutor.
type ExecuteRunWorker struct {
    river.WorkerDefaults[ExecuteRunArgs]
    executor *service.PipelineExecutor
}

// NewExecuteRunWorker creates a new ExecuteRunWorker.
func NewExecuteRunWorker(executor *service.PipelineExecutor) *ExecuteRunWorker {
    return &ExecuteRunWorker{executor: executor}
}

// Work executes the pipeline run identified by the job payload.
func (w *ExecuteRunWorker) Work(ctx context.Context, job *river.Job[ExecuteRunArgs]) error {
    return w.executor.ExecuteRun(ctx, job.Args.RunID)
}
```

#### River adapter — `backend/internal/adapter/river/job_queue.go`

```go
package river

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// RiverJobQueue implements port.JobQueue using River backed by Postgres.
type RiverJobQueue struct {
    client *river.Client[pgx.Tx]
    pool   *pgxpool.Pool
}

// NewRiverJobQueue creates a new RiverJobQueue.
// workers must have all job types registered before calling NewClient.
func NewRiverJobQueue(pool *pgxpool.Pool, workers *river.Workers) (*RiverJobQueue, error) {
    client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
        Workers: workers,
    })
    if err != nil {
        return nil, fmt.Errorf("create river client: %w", err)
    }
    return &RiverJobQueue{client: client, pool: pool}, nil
}

// Enqueue inserts a job using an autocommit connection from the pool.
func (q *RiverJobQueue) Enqueue(ctx context.Context, job any) error {
    insertable, ok := job.(river.JobArgs)
    if !ok {
        return fmt.Errorf("job must implement river.JobArgs")
    }
    conn, err := q.pool.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("acquire connection: %w", err)
    }
    defer conn.Release()
    _, err = q.client.InsertTx(ctx, conn.Conn().BeginTx, insertable, nil)
    return err
}

// EnqueueTx inserts a job within an existing pgx transaction.
// tx must be a pgx.Tx.
func (q *RiverJobQueue) EnqueueTx(ctx context.Context, tx any, job any) error {
    pgxTx, ok := tx.(pgx.Tx)
    if !ok {
        return fmt.Errorf("tx must be a pgx.Tx, got %T", tx)
    }
    insertable, ok := job.(river.JobArgs)
    if !ok {
        return fmt.Errorf("job must implement river.JobArgs")
    }
    _, err := q.client.InsertTx(ctx, pgxTx, insertable, nil)
    return err
}
```

#### `LaunchRun` service method — `backend/internal/domain/service/run_service.go`

Extend `RunService` struct:

```go
type RunService struct {
    runRepo            port.RunRepository
    projectRepo        port.ProjectRepository
    storyRepo          port.StoryRepository
    pipelineConfigRepo port.PipelineConfigRepository
    jobQueue           port.JobQueue
}

func NewRunService(
    runRepo port.RunRepository,
    projectRepo port.ProjectRepository,
    storyRepo port.StoryRepository,
    pipelineConfigRepo port.PipelineConfigRepository,
    jobQueue port.JobQueue,
) *RunService {
    return &RunService{
        runRepo:            runRepo,
        projectRepo:        projectRepo,
        storyRepo:          storyRepo,
        pipelineConfigRepo: pipelineConfigRepo,
        jobQueue:           jobQueue,
    }
}
```

Implement `LaunchRun`:

```go
// LaunchRun validates the story, creates a pending run with steps, and enqueues
// a River job for async execution. All writes occur within a single transaction.
func (s *RunService) LaunchRun(ctx context.Context, projectID, storyID uuid.UUID) (*model.Run, error) {
    // 1. Verify story exists and belongs to project
    story, err := s.storyRepo.GetByID(ctx, storyID)
    if err != nil {
        return nil, err // propagates as STORY_NOT_FOUND (not_found category)
    }
    if story.ProjectID != projectID {
        return nil, errors.NewNotFound("story", storyID)
    }

    // 2. Guard: story must not be 'done'
    if story.Status == model.StoryStatusDone {
        return nil, &errors.DomainError{
            Category: errors.CategoryValidation,
            Code:     "STORY_ALREADY_COMPLETED",
            Message:  fmt.Sprintf("story %s is already completed", story.Key),
        }
    }

    // 3. Guard: no active run (pending or running) for this story
    activeRun, err := s.runRepo.GetActiveRunByStory(ctx, storyID)
    if err != nil && !isNotFound(err) {
        return nil, err
    }
    if activeRun != nil {
        return nil, &errors.DomainError{
            Category: errors.CategoryConflict,
            Code:     "STORY_ALREADY_RUNNING",
            Message:  fmt.Sprintf("story %s already has an active run (%s)", story.Key, activeRun.ID),
        }
    }

    // 4. Fetch pipeline config for the project
    pipelineCfg, err := s.pipelineConfigRepo.GetByProjectID(ctx, projectID)
    if err != nil {
        return nil, &errors.DomainError{
            Category: errors.CategoryNotFound,
            Code:     "PIPELINE_CONFIG_NOT_FOUND",
            Message:  fmt.Sprintf("no pipeline config found for project %s", projectID),
        }
    }

    // 5. Parse YAML steps
    var parsed model.PipelineConfigYAML
    if err := yaml.Unmarshal([]byte(pipelineCfg.ConfigYAML), &parsed); err != nil {
        return nil, errors.NewInternal("parse pipeline config", err)
    }
    if len(parsed.Steps) == 0 {
        return nil, &errors.DomainError{
            Category: errors.CategoryValidation,
            Code:     "PIPELINE_CONFIG_NOT_FOUND",
            Message:  "pipeline config has no steps",
        }
    }

    // 6. Snapshot config as JSON for the run record
    snapshotJSON, err := json.Marshal(parsed)
    if err != nil {
        return nil, errors.NewInternal("marshal pipeline config snapshot", err)
    }

    // 7. Create Run
    run := &model.Run{
        ProjectID:              projectID,
        StoryID:                storyID,
        Status:                 model.RunStatusPending,
        PipelineConfigSnapshot: snapshotJSON,
    }
    createdRun, err := s.runRepo.CreateRun(ctx, run)
    if err != nil {
        return nil, err
    }

    // 8. Create RunSteps
    steps := make([]model.RunStep, 0, len(parsed.Steps))
    for i, stepCfg := range parsed.Steps {
        step := &model.RunStep{
            RunID:     createdRun.ID,
            StepName:  stepCfg.Name,
            StepOrder: i,
            Action:    stepCfg.ActionType,
            Status:    model.StepStatusPending,
        }
        createdStep, err := s.runRepo.CreateRunStep(ctx, step)
        if err != nil {
            return nil, err
        }
        steps = append(steps, *createdStep)
    }
    createdRun.Steps = steps

    // 9. Enqueue River job (non-transactional for MVP — full Transactor pattern deferred)
    if err := s.jobQueue.Enqueue(ctx, river_adapter.ExecuteRunArgs{RunID: createdRun.ID}); err != nil {
        return nil, errors.NewInternal("enqueue execute_run job", err)
    }

    return createdRun, nil
}

// isNotFound returns true if the error is a not_found domain error.
func isNotFound(err error) bool {
    domErr, ok := err.(*errors.DomainError)
    return ok && domErr.Category == errors.CategoryNotFound
}
```

> **Note on transaction scope:** The full Transactor pattern (run creation + River `InsertTx` in one DB transaction) is the target architecture. For MVP, run creation and job enqueue are sequential but not wrapped in a single transaction — if the enqueue fails after the run is created, the run stays in `pending` status and can be retried or cleaned up. Full transactional enqueue is tracked as a follow-up.

#### OpenAPI spec — `api/openapi.yaml`

Add the following endpoint block after the existing `GET /stories/{storyId}/runs` block (line ~622):

```yaml
  /projects/{projectId}/stories/{storyId}/runs:
    post:
      operationId: launchRun
      summary: Launch a pipeline run for a single story
      description: >
        Creates a pending run with steps derived from the project's pipeline config
        and enqueues async execution. Returns 409 if the story already has an active run,
        400 if the story is already done, 404 if the story or pipeline config is not found.
      tags: [runs]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/StoryIdPath"
      responses:
        "201":
          description: Run created and execution enqueued
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/RunWithSteps"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "403":
          $ref: "#/components/responses/Forbidden"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"
```

#### `LaunchRun` handler — `backend/internal/api/handler/run_handler.go`

```go
// LaunchRun handles POST /projects/{projectId}/stories/{storyId}/runs.
func (h *RunHandler) LaunchRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
    run, err := h.service.LaunchRun(r.Context(), projectID, storyID)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    writeJSON(w, http.StatusCreated, toAPIRunWithSteps(run))
}
```

Add to `server.go`:

```go
// LaunchRun delegates to RunHandler.
func (s *Server) LaunchRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, storyID StoryIdPath) {
    s.runs.LaunchRun(w, r, projectID, storyID)
}
```

### Testing Requirements

#### Unit tests — `backend/internal/domain/service/run_service_test.go`

Cover the following cases with table-driven tests and hand-written mocks:

| Test | Scenario | Expected |
|------|----------|----------|
| `TestLaunchRun_Success` | Valid story (backlog), no active run, config exists | Returns run with status `pending`, N steps, job enqueued |
| `TestLaunchRun_StoryNotFound` | `storyRepo.GetByID` returns not_found | Returns `STORY_NOT_FOUND` (404) |
| `TestLaunchRun_StoryAlreadyCompleted` | Story status = `done` | Returns `STORY_ALREADY_COMPLETED` (400) |
| `TestLaunchRun_StoryAlreadyRunning` | `GetActiveRunByStory` returns an existing run | Returns `STORY_ALREADY_RUNNING` (409) |
| `TestLaunchRun_NoPipelineConfig` | `pipelineConfigRepo.GetByProjectID` returns not_found | Returns `PIPELINE_CONFIG_NOT_FOUND` (404) |
| `TestLaunchRun_JobEnqueueFails` | `jobQueue.Enqueue` returns error | Returns `INTERNAL_ERROR` (500) |

Mock additions needed in the test file:

```go
type mockStoryRepo struct {
    getByIDFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}
func (m *mockStoryRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Story, error) {
    if m.getByIDFn != nil { return m.getByIDFn(context.Background(), id) }
    return nil, errors.NewNotFound("story", id)
}
// Implement remaining interface methods with no-op stubs

type mockPipelineConfigRepo struct {
    getByProjectIDFn func(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
}
func (m *mockPipelineConfigRepo) GetByProjectID(_ context.Context, id uuid.UUID) (*model.PipelineConfig, error) {
    if m.getByProjectIDFn != nil { return m.getByProjectIDFn(context.Background(), id) }
    return nil, errors.NewNotFound("pipeline config", id)
}
func (m *mockPipelineConfigRepo) Upsert(_ context.Context, _ *model.PipelineConfig) (*model.PipelineConfig, error) {
    return nil, nil
}

type mockJobQueue struct {
    enqueueFn func(ctx context.Context, job any) error
}
func (m *mockJobQueue) Enqueue(ctx context.Context, job any) error {
    if m.enqueueFn != nil { return m.enqueueFn(ctx, job) }
    return nil
}
func (m *mockJobQueue) EnqueueTx(_ context.Context, _ any, _ any) error { return nil }
```

Also add `GetActiveRunByStory` to the existing `mockRunRepo`:

```go
getActiveRunByStoryFn func(ctx context.Context, storyID uuid.UUID) (*model.Run, error)

func (m *mockRunRepo) GetActiveRunByStory(_ context.Context, storyID uuid.UUID) (*model.Run, error) {
    if m.getActiveRunByStoryFn != nil { return m.getActiveRunByStoryFn(context.Background(), storyID) }
    return nil, errors.NewNotFound("run", storyID)
}
```

#### Unit tests — `backend/internal/api/handler/run_handler_test.go`

| Test | Scenario | Expected HTTP |
|------|----------|--------------|
| `TestLaunchRun_Created` | Service returns run | 201 + RunWithSteps JSON |
| `TestLaunchRun_StoryNotFound` | Service returns not_found | 404 |
| `TestLaunchRun_AlreadyRunning` | Service returns conflict | 409 |
| `TestLaunchRun_AlreadyCompleted` | Service returns validation error | 400 |

### References

- `backend/internal/domain/service/run_service.go` — existing `RunService` to extend
- `backend/internal/domain/service/run_service_test.go` — existing mock patterns
- `backend/internal/api/handler/run_handler.go` — existing handler structure
- `backend/internal/api/handler/helpers.go` — `writeJSON`, `writeErrorResponse`
- `backend/internal/domain/model/story.go` — `StoryStatusDone`, `StoryStatusRunning`
- `backend/internal/domain/model/pipeline_config.go` — `PipelineConfigYAML`, `PipelineStep`
- `backend/internal/domain/service/pipeline_executor.go` — `PipelineExecutor.ExecuteRun`
- `api/openapi.yaml` — OpenAPI spec (single source of truth)
- Architecture doc: `_bmad-output/planning-artifacts/architecture.md` (River section, JobQueue port definition)
- River docs: https://riverqueue.com/docs

## Dev Agent Record

_To be filled in during implementation._

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-17 | 1.0 | Initial story creation | Arch |
