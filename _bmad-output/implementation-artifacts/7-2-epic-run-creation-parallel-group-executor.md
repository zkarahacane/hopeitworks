# Story 7.2: [BACK] Epic Run Creation + Parallel Group Executor

Status: ready-for-dev

## Story

As a user, I want to launch all stories of an epic as a batch run with parallel execution within DAG layers, So that independent stories run simultaneously while dependencies are respected.

## Acceptance Criteria (BDD)

**AC1: POST /api/v1/projects/{projectId}/epics/{epicId}/runs returns 202 with scheduling payload**
- **Given** an authenticated user with project access and an epic with at least one story
- **When** POST /api/v1/projects/{projectId}/epics/{epicId}/runs is called (no body required)
- **Then** the response is 202 Accepted with body `{ "epic_run_id": "<uuid>", "status": "scheduling", "stories_count": N }` where N is the count of stories in the epic

**AC2: EpicRun record is created in the database with status pending**
- **Given** a valid launch request
- **When** the handler processes the request
- **Then** a row is inserted into `epic_runs` with `project_id`, `epic_id`, `status = 'pending'`, `created_at` set; `completed_at` is null
- **And** one row per story is inserted into `epic_run_stories` with the correct `group_index` from the DAG computation, `status = 'pending'`

**AC3: DAG cycle in epic stories returns 422 and no epic run is created**
- **Given** an epic whose stories contain a dependency cycle
- **When** POST /api/v1/projects/{projectId}/epics/{epicId}/runs is called
- **Then** the response is 422 with error code `DAG_CYCLE_DETECTED`
- **And** no `epic_runs` row is inserted

**AC4: ParallelGroupExecutor processes DAG layers sequentially, stories within a layer in parallel**
- **Given** an epic with 3 stories: S-01 (layer 0), S-02 and S-03 both depending on S-01 (layer 1)
- **When** the epic run is executed
- **Then** S-01 is executed first; only after S-01 completes do S-02 and S-03 start concurrently
- **And** all three `Run` records are created via `RunService.CreateRun` before the executor returns

**AC5: Fail-fast mode — any story failure in a layer aborts the epic run**
- **Given** an epic run in progress where one story in a parallel layer fails
- **When** the failed story's run completes with status `failed`
- **Then** the epic run transitions to `failed` immediately (remaining parallel stories in the layer are allowed to finish but the next layer does not start)
- **And** the `epic_run.failed` event is published

**AC6: All layers complete successfully — epic run transitions to completed**
- **Given** all stories across all DAG layers complete successfully
- **When** the last layer's stories finish
- **Then** the epic run status transitions to `completed` and `completed_at` is set
- **And** the `epic_run.completed` event is published

**AC7: Events are published at each execution milestone**
- **Given** an epic run in progress
- **When** the executor starts, begins a layer, and each story completes
- **Then** the following events are published in order:
  - `epic_run.started` — when execution begins
  - `epic_run.group.started` — once per layer, payload includes `group_index`
  - `epic_run.story.completed` — once per story completion, payload includes `story_id`, `run_id`, `status`
  - `epic_run.completed` or `epic_run.failed` — at the end

**AC8: GET /api/v1/projects/{projectId}/epic-runs/{epicRunId} returns epic run with per-story status**
- **Given** an existing epic run
- **When** GET /api/v1/projects/{projectId}/epic-runs/{epicRunId} is called
- **Then** the response is 200 with the epic run object including `id`, `project_id`, `epic_id`, `status`, `created_at`, `completed_at`, and a `stories` array where each element has `story_id`, `run_id`, `group_index`, `status`

**AC9: Wire and DI — EpicRunHandler and ParallelGroupExecutor are wired via go-wire**
- **Given** the application starts
- **When** the DI graph is initialized
- **Then** `EpicRunHandler`, `ParallelGroupExecutor`, `EpicRunService`, and `EpicRunRepository` are all resolvable without manual initialization

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec with epic run endpoints (AC: #1, #8)
  - [ ] Add path `POST /api/v1/projects/{projectId}/epics/{epicId}/runs` with operationId `launchEpicRun`; request body is empty (no schema required); response 202 schema `EpicRunScheduled`: `{ epic_run_id: string (uuid), status: string, stories_count: integer }`
  - [ ] Add path `GET /api/v1/projects/{projectId}/epic-runs/{epicRunId}` with operationId `getEpicRun`; response 200 schema `EpicRunDetail`: `{ id, project_id, epic_id, status, created_at, completed_at, stories: EpicRunStory[] }`
  - [ ] Add `EpicRunStory` schema: `{ story_id: string (uuid), run_id: string (uuid) | null, group_index: integer, status: string }`
  - [ ] Remove the stub `launchEpicRun` added in story 7-1 if present (replace with full schema)
  - [ ] Regenerate backend types: `cd backend && make generate` — verify `LaunchEpicRun` and `GetEpicRun` appear in `gen_server.go`

- [ ] [BACK] Task 2: Add DB migrations for epic_runs and epic_run_stories (AC: #2)
  - [ ] Create `backend/migrations/000014_create_epic_runs_table.up.sql`:
    ```sql
    CREATE TYPE epic_run_status AS ENUM ('pending', 'running', 'completed', 'failed', 'paused');

    CREATE TABLE epic_runs (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
        epic_id     UUID NOT NULL REFERENCES epics(id) ON DELETE CASCADE,
        status      epic_run_status NOT NULL DEFAULT 'pending',
        created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
        completed_at TIMESTAMPTZ
    );

    CREATE INDEX idx_epic_runs_project_id ON epic_runs(project_id);
    CREATE INDEX idx_epic_runs_epic_id ON epic_runs(epic_id);
    CREATE INDEX idx_epic_runs_status ON epic_runs(status);
    ```
  - [ ] Create `backend/migrations/000014_create_epic_runs_table.down.sql`: `DROP TABLE IF EXISTS epic_runs; DROP TYPE IF EXISTS epic_run_status;`
  - [ ] Create `backend/migrations/000015_create_epic_run_stories_table.up.sql`:
    ```sql
    CREATE TABLE epic_run_stories (
        epic_run_id UUID NOT NULL REFERENCES epic_runs(id) ON DELETE CASCADE,
        story_id    UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
        run_id      UUID REFERENCES runs(id) ON DELETE SET NULL,
        group_index INTEGER NOT NULL,
        status      TEXT NOT NULL DEFAULT 'pending',
        PRIMARY KEY (epic_run_id, story_id)
    );

    CREATE INDEX idx_epic_run_stories_epic_run_id ON epic_run_stories(epic_run_id);
    CREATE INDEX idx_epic_run_stories_run_id ON epic_run_stories(run_id);
    ```
  - [ ] Create `backend/migrations/000015_create_epic_run_stories_table.down.sql`: `DROP TABLE IF EXISTS epic_run_stories;`

- [ ] [BACK] Task 3: Add sqlc queries and domain model for EpicRun (AC: #2, #8)
  - [ ] Create `backend/queries/epic_runs.sql` with queries:
    - `CreateEpicRun :one` — INSERT INTO epic_runs RETURNING *
    - `GetEpicRun :one` — SELECT * FROM epic_runs WHERE id = $1
    - `UpdateEpicRunStatus :one` — UPDATE epic_runs SET status=$1, completed_at=$2 WHERE id=$3 RETURNING *
    - `InsertEpicRunStory :exec` — INSERT INTO epic_run_stories(epic_run_id, story_id, run_id, group_index, status) VALUES(...)
    - `UpdateEpicRunStoryStatus :exec` — UPDATE epic_run_stories SET status=$1, run_id=$2 WHERE epic_run_id=$3 AND story_id=$4
    - `ListEpicRunStories :many` — SELECT * FROM epic_run_stories WHERE epic_run_id=$1 ORDER BY group_index, story_id
  - [ ] Run `cd backend && sqlc generate` to generate `internal/adapter/postgres/db/epic_runs.sql.go`
  - [ ] Create `backend/internal/domain/model/epic_run.go`:
    - `EpicRunStatus` string type with constants: `EpicRunStatusPending`, `EpicRunStatusRunning`, `EpicRunStatusCompleted`, `EpicRunStatusFailed`, `EpicRunStatusPaused`
    - `EpicRun` struct: `ID uuid.UUID`, `ProjectID uuid.UUID`, `EpicID uuid.UUID`, `Status EpicRunStatus`, `CreatedAt time.Time`, `CompletedAt *time.Time`, `Stories []EpicRunStory`
    - `EpicRunStory` struct: `EpicRunID uuid.UUID`, `StoryID uuid.UUID`, `RunID *uuid.UUID`, `GroupIndex int`, `Status string`
    - `ValidateEpicRunTransition(from, to EpicRunStatus) error` — valid transitions: `pending→running`, `running→completed`, `running→failed`, `running→paused`

- [ ] [BACK] Task 4: Add EpicRunRepository port and postgres adapter (AC: #2, #8)
  - [ ] Create `backend/internal/domain/port/epic_run_repository.go`:
    ```go
    type EpicRunRepository interface {
        CreateEpicRun(ctx context.Context, run *model.EpicRun) (*model.EpicRun, error)
        GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error)
        UpdateEpicRunStatus(ctx context.Context, id uuid.UUID, status model.EpicRunStatus, completedAt *time.Time) (*model.EpicRun, error)
        InsertEpicRunStory(ctx context.Context, story model.EpicRunStory) error
        UpdateEpicRunStoryStatus(ctx context.Context, epicRunID, storyID uuid.UUID, status string, runID *uuid.UUID) error
        ListEpicRunStories(ctx context.Context, epicRunID uuid.UUID) ([]model.EpicRunStory, error)
    }
    ```
  - [ ] Create `backend/internal/adapter/postgres/epic_run_repository.go` implementing the port using generated sqlc queries
  - [ ] Map `pgx.ErrNoRows` to `errors.NewNotFound("epic_run", id.String())` in GetEpicRun

- [ ] [BACK] Task 5: Implement EpicRunService (AC: #1, #2, #3)
  - [ ] Create `backend/internal/domain/service/epic_run_service.go`
  - [ ] `EpicRunService` struct depends on: `port.EpicRunRepository`, `port.StoryRepository`, `port.EpicRepository` (for validation), `*SchedulerService`, `*ParallelGroupExecutor`, `port.EventPublisher`
  - [ ] `LaunchEpicRun(ctx context.Context, projectID, epicID uuid.UUID) (*model.EpicRun, error)`:
    1. Verify epic exists and belongs to projectID (return `errors.NewNotFound` or `errors.NewForbidden` as appropriate)
    2. Fetch all stories for the epic via `storyRepo.ListByEpic`
    3. Call `scheduler.BuildDAG(stories)` — if error code is `DAG_CYCLE_DETECTED`, propagate as-is (422 via middleware mapping of `validation` category)
    4. Create `EpicRun` record with status `pending` via `epicRunRepo.CreateEpicRun`
    5. Insert one `EpicRunStory` row per story with correct `group_index` from DAG groups
    6. Launch `go executor.Execute(context.Background(), epicRun, dag)` in a goroutine (fire-and-forget; use detached context to survive HTTP request lifecycle)
    7. Return the created `EpicRun` immediately (handler responds 202)
  - [ ] `GetEpicRun(ctx context.Context, id uuid.UUID) (*model.EpicRun, error)` — delegates to repo, populates `Stories` field

- [ ] [BACK] Task 6: Implement ParallelGroupExecutor (AC: #4, #5, #6, #7)
  - [ ] Create `backend/internal/domain/service/parallel_group_executor.go`
  - [ ] `ParallelGroupExecutor` struct depends on: `port.EpicRunRepository`, `*RunService`, `*PipelineExecutor`, `port.EventPublisher`, `*slog.Logger`
  - [ ] `Execute(ctx context.Context, epicRun *model.EpicRun, dag model.DAGResult) error`:
    1. Transition epic run to `running`, publish `epic_run.started` event
    2. Iterate `dag.Groups` by index (layer):
       a. Publish `epic_run.group.started` event with payload `{ "group_index": i, "story_count": len(group) }`
       b. Use `golang.org/x/sync/errgroup` to launch one goroutine per story in the group:
          - Call `runSvc.CreateRun(ctx, projectID, story.ID, pipelineConfig)` to get a `*model.Run`
          - Update `epic_run_stories` row: set `run_id`, status `running`
          - Call `executor.ExecuteRun(ctx, run.ID)`
          - On success: update `epic_run_stories` status to `completed`, publish `epic_run.story.completed` event
          - On error: update `epic_run_stories` status to `failed`, publish `epic_run.story.completed` event with `status: failed`
       c. Wait for all goroutines via `eg.Wait()` — if any returned an error, transition epic run to `failed`, publish `epic_run.failed`, return
    3. After all layers complete: transition epic run to `completed`, set `completed_at`, publish `epic_run.completed`
  - [ ] Import `golang.org/x/sync/errgroup` — verify it is already in `go.mod`; if not, run `go get golang.org/x/sync`
  - [ ] Use `log/slog` structured logging: log each layer start/end and each story result with `epic_run_id`, `group_index`, `story_id`, `run_id`

- [ ] [BACK] Task 7: Implement EpicRunHandler and wire it up (AC: #1, #8, #9)
  - [ ] Create `backend/internal/api/handler/epic_run_handler.go`:
    - `EpicRunHandler` struct with `*service.EpicRunService`
    - `LaunchEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath)`:
      - Call `svc.LaunchEpicRun(ctx, projectID, epicID)`
      - Write 202 with `EpicRunScheduled{ EpicRunId: run.ID, Status: "scheduling", StoriesCount: len(run.Stories) }`
    - `GetEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicRunID EpicRunIdPath)`:
      - Call `svc.GetEpicRun(ctx, epicRunID)`, write 200 with `EpicRunDetail`
  - [ ] Register handler in `backend/internal/api/handler/server.go`: add `EpicRunHandler` to `Server` struct, wire the two new routes via the oapi-codegen generated interface
  - [ ] Update `backend/cmd/api/wire.go`: add `NewEpicRunRepository`, `NewEpicRunService`, `NewParallelGroupExecutor`, `NewEpicRunHandler` to the appropriate provider sets
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] [BACK] Task 8: Write unit tests for EpicRunService and ParallelGroupExecutor (AC: #1–#7)
  - [ ] Create `backend/internal/domain/service/epic_run_service_test.go`:
    - Test `LaunchEpicRun` with valid epic: assert EpicRun created with pending status, correct stories_count returned
    - Test `LaunchEpicRun` with cyclic epic: assert `DAG_CYCLE_DETECTED` error propagated, no EpicRun inserted
    - Test `LaunchEpicRun` with non-existent epic: assert `not_found` error
    - Use hand-written mocks for `port.EpicRunRepository`, `port.StoryRepository`
  - [ ] Create `backend/internal/domain/service/parallel_group_executor_test.go`:
    - Test sequential layer execution (mock `PipelineExecutor.ExecuteRun` with captured call order)
    - Test fail-fast: first layer succeeds, second layer has one failure — epic run transitions to `failed`
    - Test all success: epic run transitions to `completed`, `completed_at` set
    - Test events published: verify `EventPublisher.Publish` called with expected `entity_type` and `action` values in correct order
    - All tests use `-short` compatible mocks (no testcontainers)

- [ ] [BACK] Task 9: Write integration test for epic run DB operations (AC: #2, #8)
  - [ ] Create `backend/internal/adapter/postgres/epic_run_repository_integration_test.go`
  - [ ] Guard with `testing.Short()` skip
  - [ ] Use `testutil.NewTestDB(t)` to get ephemeral Postgres with migrations applied
  - [ ] Test: create epic run → list stories → update story status → get epic run (verify stories populated)
  - [ ] Test: `GetEpicRun` with unknown UUID returns `not_found` DomainError

- [ ] [BACK] Task 10: Lint and validate (AC: all)
  - [ ] Run `cd backend && golangci-lint run ./...` — fix all reported issues before committing
  - [ ] Run `cd backend && go test ./... -short` — all unit tests must pass
  - [ ] Verify `go vet ./...` is clean
  - [ ] Ensure no `console.log`, `fmt.Println`, `TODO` without story key, or hardcoded secrets are introduced

## Dev Notes

### Dependencies

- Story 7-1 (done): `SchedulerService.BuildDAG` and `model.DAGResult` are available; `model.DAGResult.Groups` is `[][]model.Story`
- Story 3-1 (done): `RunService.CreateRun`, `RunRepository`, `model.Run` state machine available
- Story 3-7 (done): `PipelineExecutor.ExecuteRun(ctx, runID uuid.UUID) error` executes a single story's steps

### Architecture Requirements

- `EpicRunService` and `ParallelGroupExecutor` are domain services — they depend on ports only, never on adapters or handlers directly
- `ParallelGroupExecutor.Execute` is called in a detached goroutine; the HTTP handler must NOT wait for it — use `context.Background()` or a detached context derived from `context.WithoutCancel` (Go 1.21+) so the goroutine outlives the HTTP request
- The `errgroup` context must NOT be used for the fire-and-forget outer goroutine; only use it internally within a single layer's parallel stories
- Events follow the established `model.Event` struct: `EntityType = "epic_run"`, `EntityID = epicRun.ID`, `Action = "started"` etc.; for group events use `EntityType = "epic_run_group"` and include `group_index` in `Payload`
- `EpicRunRepository` adapter lives in `backend/internal/adapter/postgres/epic_run_repository.go` — same pattern as `run_repository.go`

### File Paths (exact)

| Purpose | Path |
|---------|------|
| OpenAPI spec | `api/openapi.yaml` |
| Domain model | `backend/internal/domain/model/epic_run.go` |
| Port interface | `backend/internal/domain/port/epic_run_repository.go` |
| Service — launch + get | `backend/internal/domain/service/epic_run_service.go` |
| Service — executor | `backend/internal/domain/service/parallel_group_executor.go` |
| Postgres adapter | `backend/internal/adapter/postgres/epic_run_repository.go` |
| Handler | `backend/internal/api/handler/epic_run_handler.go` |
| sqlc queries | `backend/queries/epic_runs.sql` |
| Migration 14 up | `backend/migrations/000014_create_epic_runs_table.up.sql` |
| Migration 14 down | `backend/migrations/000014_create_epic_runs_table.down.sql` |
| Migration 15 up | `backend/migrations/000015_create_epic_run_stories_table.up.sql` |
| Migration 15 down | `backend/migrations/000015_create_epic_run_stories_table.down.sql` |
| Unit tests — service | `backend/internal/domain/service/epic_run_service_test.go` |
| Unit tests — executor | `backend/internal/domain/service/parallel_group_executor_test.go` |
| Integration tests | `backend/internal/adapter/postgres/epic_run_repository_integration_test.go` |
| DI wiring | `backend/cmd/api/wire.go` |

### Technical Specifications

**errgroup usage pattern:**
```go
eg, egCtx := errgroup.WithContext(ctx)
for _, story := range group {
    story := story // capture loop var (Go < 1.22)
    eg.Go(func() error {
        return e.runStory(egCtx, epicRun, story, groupIndex)
    })
}
if err := eg.Wait(); err != nil {
    // fail-fast: mark epic run failed, do not start next layer
    return err
}
```

**Detached goroutine (fire-and-forget) pattern:**
```go
// In EpicRunService.LaunchEpicRun:
go func() {
    detachedCtx := context.WithoutCancel(ctx) // Go 1.21+
    if err := s.executor.Execute(detachedCtx, epicRun, dag); err != nil {
        s.logger.Error("epic run failed", "epic_run_id", epicRun.ID, "error", err)
    }
}()
```

**Event payload examples:**
```go
// epic_run.started
model.Event{ProjectID: projectID, EntityType: "epic_run", EntityID: epicRun.ID, Action: "started", Payload: json.RawMessage(`{}`)}

// epic_run.group.started
Payload: json.RawMessage(fmt.Sprintf(`{"group_index":%d,"story_count":%d}`, i, len(group)))

// epic_run.story.completed
Payload: json.RawMessage(fmt.Sprintf(`{"story_id":%q,"run_id":%q,"status":%q}`, storyID, runID, status))
```

**HTTP 422 mapping:** `DAG_CYCLE_DETECTED` must map to HTTP 422. If the existing middleware maps `validation` category errors to 400, add a dedicated `unprocessable` category or override in the middleware for `DAG_CYCLE_DETECTED` code. Alternatively, keep 400 if spec allows — align with what `api/openapi.yaml` documents for this endpoint.

**golangci-lint compliance:**
- Rename unused goroutine closure parameters to `_`
- All exported types and functions must have godoc comments
- No `fmt.Println` — use `s.logger.Error(...)` / `s.logger.Info(...)`
- `errcheck`: do not ignore errors from `eventPub.Publish` — log on error but do not abort execution

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
