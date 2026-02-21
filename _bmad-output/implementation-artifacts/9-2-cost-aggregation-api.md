# Story 9.2: [BACK] Cost Aggregation API

Status: ready-for-dev

## Story

As a platform operator,
I want API endpoints that aggregate cost data by project, run, and story,
So that the cost dashboard can display spending breakdowns and trends.

## Acceptance Criteria (BDD)

**AC1: OpenAPI spec defines the four cost endpoints**
- **Given** `api/openapi.yaml` is updated
- **When** `cd backend && make generate` is run
- **Then** oapi-codegen produces server interface methods: `GetProjectCostsSummary`, `ListProjectCostsByRun`, `ListProjectCostsByStory`, `GetRunCosts`
- **And** `cd frontend && npm run generate-api` produces matching TypeScript types

**AC2: GET /projects/{projectId}/costs/summary returns aggregated totals**
- **Given** I am authenticated and the project exists with cost_records
- **When** I GET `/api/v1/projects/{projectId}/costs/summary?period=30d`
- **Then** I receive HTTP 200 with `{ "total_cost_usd": 12.50, "tokens_input": 1000000, "tokens_output": 500000, "runs_count": 42, "period_days": 30 }`
- **And** `period` accepts values `7d`, `30d`, `90d` (default `30d`)
- **And** an invalid `period` value returns HTTP 400 with error code `INVALID_PERIOD`

**AC3: GET /projects/{projectId}/costs/by-run returns paginated per-run cost rows**
- **Given** I am authenticated and the project has completed runs with cost records
- **When** I GET `/api/v1/projects/{projectId}/costs/by-run?limit=20&offset=0`
- **Then** I receive HTTP 200 with a paginated envelope `{ "data": [...], "pagination": { "total": N, "page": 1, "per_page": 20 } }`
- **And** each item contains: `run_id`, `story_key`, `cost_usd`, `tokens_input`, `tokens_output`, `model`, `completed_at`
- **And** `limit` is capped at 100; values above 100 return HTTP 400

**AC4: GET /projects/{projectId}/costs/by-story returns per-story cost aggregates**
- **Given** I am authenticated and stories have multiple runs with cost records
- **When** I GET `/api/v1/projects/{projectId}/costs/by-story`
- **Then** I receive HTTP 200 with a list of `{ "story_key", "story_summary", "total_cost_usd", "run_count" }` ordered by `total_cost_usd DESC`

**AC5: GET /projects/{projectId}/runs/{runId}/costs returns step-level cost detail**
- **Given** I am authenticated and the run exists with cost records per step
- **When** I GET `/api/v1/projects/{projectId}/runs/{runId}/costs`
- **Then** I receive HTTP 200 with `{ "run_id", "total_cost_usd", "steps": [{ "step_name", "cost_usd", "tokens_input", "tokens_output", "model" }] }`
- **And** a run that belongs to a different project returns HTTP 404

**AC6: Project membership is enforced on all cost endpoints**
- **Given** I am authenticated as a user without access to the project
- **When** I call any cost endpoint for that project
- **Then** I receive HTTP 403 (enforced by existing project-scope middleware)

**AC7: New sqlc queries compile and generate correct Go types**
- **Given** new queries are added to `backend/queries/cost_records.sql`
- **When** `cd backend && sqlc generate` runs
- **Then** `ListCostsByProject`, `CountCostsByProject`, `ListCostsByStory`, `ListCostsByRun` are generated without errors
- **And** the generated types match the JOIN columns returned by each query

**AC8: Unit and integration tests pass under golangci-lint**
- **Given** all implementation files are written
- **When** `cd backend && golangci-lint run ./...` and `go test ./... -short` are run
- **Then** both pass with zero errors or warnings

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec with four cost endpoints (AC: #1)
  - [ ] Add tag `costs` to the tags section of `api/openapi.yaml`
  - [ ] Add `GET /projects/{projectId}/costs/summary` with query param `period` (enum: 7d, 30d, 90d) and `ProjectCostSummary` response schema
  - [ ] Add `GET /projects/{projectId}/costs/by-run` with `limit` (int, max 100) and `offset` query params and `ProjectCostsByRunResponse` (paginated) response schema
  - [ ] Add `GET /projects/{projectId}/costs/by-story` with `ProjectCostsByStoryResponse` response schema
  - [ ] Add `GET /projects/{projectId}/runs/{runId}/costs` with `RunCostDetail` response schema
  - [ ] Define component schemas: `ProjectCostSummary`, `RunCostItem`, `StoryCostItem`, `RunCostDetail`, `RunStepCostItem`
  - [ ] Run `cd backend && make generate` and `cd frontend && npm run generate-api` to verify codegen succeeds

- [ ] [BACK] Task 2: Add sqlc queries for cost aggregation (AC: #7)
  - [ ] Extend `backend/queries/cost_records.sql` with:
    - `ListCostsByProject :many` — paginated, JOIN runs + stories to get story_key and completed_at
    - `CountCostsByProject :one` — count for pagination metadata
    - `ListCostsByStory :many` — GROUP BY story, SUM cost_usd and COUNT runs, ORDER BY total_cost_usd DESC
    - `ListCostsByRun :many` — JOIN run_steps to get step_name, grouped by step
  - [ ] Run `cd backend && sqlc generate` and verify the generated `internal/adapter/postgres/db/` compiles

- [ ] [BACK] Task 3: Extend CostRepository port with aggregation methods (AC: #2, #3, #4, #5)
  - [ ] Add to `backend/internal/domain/port/cost_repository.go`:
    - `CountCostsByProject(ctx, projectID uuid.UUID, since time.Time) (int64, error)`
    - `ListCostsByProject(ctx, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostItem, error)`
    - `ListCostsByStory(ctx, projectID uuid.UUID) ([]model.StoryCostItem, error)`
    - `ListCostsByRun(ctx, runID uuid.UUID) ([]model.RunStepCostItem, error)`
  - [ ] Add domain model structs to `backend/internal/domain/model/cost_record.go`:
    - `RunCostItem` — `RunID`, `StoryKey`, `CostUSD`, `TokensInput`, `TokensOutput`, `Model`, `CompletedAt`
    - `StoryCostItem` — `StoryKey`, `StorySummary`, `TotalCostUSD`, `RunCount`
    - `RunStepCostItem` — `StepName`, `CostUSD`, `TokensInput`, `TokensOutput`, `Model`

- [ ] [BACK] Task 4: Implement new repository methods in postgres adapter (AC: #3, #4, #5, #7)
  - [ ] Extend `backend/internal/adapter/postgres/cost_repository.go` with implementations for the four new port methods
  - [ ] Map sqlc-generated rows to domain model structs
  - [ ] Handle `pgx.ErrNoRows` as empty slice (not an error) for list queries
  - [ ] Wrap unexpected errors with `domainerrors.NewInternal`

- [ ] [BACK] Task 5: Create CostService aggregation methods (AC: #2, #3, #4, #5)
  - [ ] Extend `backend/internal/domain/service/cost_service.go` with:
    - `GetProjectSummary(ctx, projectID uuid.UUID, periodDays int) (*model.ProjectCostSummary, error)`
    - `ListByRun(ctx, projectID uuid.UUID, limit, offset int32) ([]model.RunCostItem, int64, error)`
    - `ListByStory(ctx, projectID uuid.UUID) ([]model.StoryCostItem, error)`
    - `GetRunDetail(ctx, projectID, runID uuid.UUID) (*model.RunCostDetail, error)`
  - [ ] `GetProjectSummary` validates `periodDays` in {7, 30, 90}, returns `domainerrors.NewValidation` otherwise
  - [ ] `ListByRun` caps `limit` at 100, returns validation error if exceeded
  - [ ] `GetRunDetail` verifies the run belongs to the project (join via run_steps) or returns `domainerrors.NewNotFound`
  - [ ] Add `ProjectCostSummary` and `RunCostDetail` model structs to `backend/internal/domain/model/cost_record.go`

- [ ] [BACK] Task 6: Implement CostHandler with four HTTP handlers (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Create `backend/internal/api/handler/cost_handler.go`
  - [ ] Implement the oapi-codegen generated interface methods: `GetProjectCostsSummary`, `ListProjectCostsByRun`, `ListProjectCostsByStory`, `GetRunCosts`
  - [ ] Parse and validate query params (`period`, `limit`, `offset`) from request
  - [ ] Call the appropriate `CostService` method
  - [ ] Render responses with `renderJSON` helper (200 on success)
  - [ ] Propagate `DomainError` via `renderError` for middleware mapping

- [ ] [BACK] Task 7: Register cost routes in chi router and wire DI (AC: #1, #6)
  - [ ] Register routes under the existing project-scoped group in the router setup:
    ```
    GET /api/v1/projects/{projectId}/costs/summary
    GET /api/v1/projects/{projectId}/costs/by-run
    GET /api/v1/projects/{projectId}/costs/by-story
    GET /api/v1/projects/{projectId}/runs/{runId}/costs
    ```
  - [ ] Add `handler.NewCostHandler` provider to `backend/cmd/api/wire.go`
  - [ ] Inject `CostService` into `CostHandler` constructor
  - [ ] Run `cd backend && wire ./cmd/api/` and verify `wire_gen.go` regenerates cleanly

- [ ] [BACK] Task 8: Write unit and integration tests (AC: #8)
  - [ ] Create `backend/internal/domain/service/cost_service_aggregation_test.go` with table-driven unit tests:
    - `GetProjectSummary` with invalid period values (`1d`, `0`, `365d`) returns validation error
    - `GetProjectSummary` with valid period calls repo with correct `since` time
    - `ListByRun` with `limit=101` returns validation error
    - `GetRunDetail` when run not found returns `NotFound` error
  - [ ] Create `backend/internal/api/handler/cost_handler_test.go` with handler-level unit tests using mock service:
    - `GET /costs/summary?period=30d` returns 200 with correct JSON shape
    - `GET /costs/summary?period=invalid` returns 400
    - `GET /costs/by-run?limit=200` returns 400
    - Unauthenticated request returns 401 (middleware test)
  - [ ] Run `cd backend && golangci-lint run ./...` — must be clean
  - [ ] Run `cd backend && go test ./... -short` — must pass

## Dev Notes

### Dependencies

- **Story 9-1 (DONE):** `cost_records` table, `CostRepository` port (with `InsertCostRecord`, `GetCostByRunStep`, `SumCostByProject`, `SumCostByRun`), `CostService.RecordStepCost`, sqlc queries in `backend/queries/cost_records.sql`
- **No new migration needed:** all queries run against the existing `cost_records` table from 9-1
- **Last migration number:** `000013` (from 9-1) — do NOT create a new migration

### Architecture Requirements

Port/adapter boundaries:

```
CostHandler (api/handler)
    └─ injects CostService (domain/service)
              └─ injects CostRepository (domain/port)
                          └─ implemented by postgres.CostRepository (adapter/postgres)
                                      └─ uses sqlc-generated db.Queries
```

`CostService` must depend only on `CostRepository` (port interface). `CostHandler` must depend only on `CostService` (concrete service struct or interface — follow existing handler pattern in the codebase).

### File Paths (exact)

```
api/openapi.yaml                                                     (extend: add costs tag + 4 endpoints + 5 schemas)
backend/queries/cost_records.sql                                     (extend: add 4 new queries)
backend/internal/domain/model/cost_record.go                        (extend: add RunCostItem, StoryCostItem, RunStepCostItem, ProjectCostSummary, RunCostDetail)
backend/internal/domain/port/cost_repository.go                     (extend: add 4 new interface methods)
backend/internal/domain/service/cost_service.go                     (extend: add GetProjectSummary, ListByRun, ListByStory, GetRunDetail)
backend/internal/domain/service/cost_service_aggregation_test.go    (new)
backend/internal/adapter/postgres/cost_repository.go                (extend: implement 4 new port methods)
backend/internal/api/handler/cost_handler.go                        (new)
backend/internal/api/handler/cost_handler_test.go                   (new)
backend/cmd/api/wire.go                                              (extend: add CostHandler provider)
```

### Technical Specifications

**New sqlc queries (append to `backend/queries/cost_records.sql`):**

```sql
-- name: ListCostsByProject :many
SELECT
    cr.run_step_id,
    rs.run_id,
    s.key            AS story_key,
    cr.cost_usd,
    cr.tokens_input,
    cr.tokens_output,
    cr.model,
    r.completed_at
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
JOIN runs r       ON r.id  = rs.run_id
JOIN stories s    ON s.id  = r.story_id
WHERE cr.project_id = $1
  AND cr.created_at >= $2
ORDER BY cr.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCostsByProject :one
SELECT COUNT(*)
FROM cost_records
WHERE project_id = $1
  AND created_at >= $2;

-- name: ListCostsByStory :many
SELECT
    s.key            AS story_key,
    s.summary        AS story_summary,
    COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost_usd,
    COUNT(DISTINCT r.id)                           AS run_count
FROM stories s
JOIN runs r        ON r.story_id  = s.id
JOIN run_steps rs  ON rs.run_id   = r.id
JOIN cost_records cr ON cr.run_step_id = rs.id
WHERE s.project_id = $1
GROUP BY s.id, s.key, s.summary
ORDER BY total_cost_usd DESC;

-- name: ListCostsByRun :many
SELECT
    rs.step_name,
    cr.cost_usd,
    cr.tokens_input,
    cr.tokens_output,
    cr.model
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
WHERE rs.run_id = $1
ORDER BY rs.step_order ASC;
```

**New domain model structs (in `cost_record.go`):**

```go
// ProjectCostSummary aggregates cost totals for a project over a time window.
type ProjectCostSummary struct {
    TotalCostUSD  float64
    TokensInput   int64
    TokensOutput  int64
    RunsCount     int64
    PeriodDays    int
}

// RunCostItem is a single row in the per-run cost breakdown list.
type RunCostItem struct {
    RunID        uuid.UUID
    StoryKey     string
    CostUSD      float64
    TokensInput  int64
    TokensOutput int64
    Model        string
    CompletedAt  *time.Time
}

// StoryCostItem is a single row in the per-story cost aggregate list.
type StoryCostItem struct {
    StoryKey      string
    StorySummary  string
    TotalCostUSD  float64
    RunCount      int64
}

// RunCostDetail is the full cost breakdown for one run.
type RunCostDetail struct {
    RunID        uuid.UUID
    TotalCostUSD float64
    Steps        []RunStepCostItem
}

// RunStepCostItem is the cost record for a single pipeline step within a run.
type RunStepCostItem struct {
    StepName     string
    CostUSD      float64
    TokensInput  int64
    TokensOutput int64
    Model        string
}
```

**New CostRepository port methods:**

```go
// CountCostsByProject returns the total number of cost records for a project since the given time.
CountCostsByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error)

// ListCostsByProject returns paginated per-step cost rows for a project since the given time.
ListCostsByProject(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostItem, error)

// ListCostsByStory returns per-story cost aggregates for a project, ordered by total cost descending.
ListCostsByStory(ctx context.Context, projectID uuid.UUID) ([]model.StoryCostItem, error)

// ListCostsByRun returns per-step cost rows for a specific run.
ListCostsByRun(ctx context.Context, runID uuid.UUID) ([]model.RunStepCostItem, error)
```

**Period-to-duration mapping in CostService:**

```go
var validPeriods = map[int]bool{7: true, 30: true, 90: true}

func periodSince(days int) (time.Time, error) {
    if !validPeriods[days] {
        return time.Time{}, errors.NewValidation("period", "must be one of 7, 30, 90")
    }
    return time.Now().UTC().AddDate(0, 0, -days), nil
}
```

**OpenAPI response schemas (summary):**

```yaml
ProjectCostSummary:
  type: object
  required: [total_cost_usd, tokens_input, tokens_output, runs_count, period_days]
  properties:
    total_cost_usd:  { type: number, format: double }
    tokens_input:    { type: integer, format: int64 }
    tokens_output:   { type: integer, format: int64 }
    runs_count:      { type: integer, format: int64 }
    period_days:     { type: integer }

RunCostItem:
  type: object
  required: [run_id, story_key, cost_usd, tokens_input, tokens_output, model]
  properties:
    run_id:       { type: string, format: uuid }
    story_key:    { type: string }
    cost_usd:     { type: number, format: double }
    tokens_input: { type: integer, format: int64 }
    tokens_output: { type: integer, format: int64 }
    model:        { type: string }
    completed_at: { type: string, format: date-time, nullable: true }

StoryCostItem:
  type: object
  required: [story_key, story_summary, total_cost_usd, run_count]
  properties:
    story_key:     { type: string }
    story_summary: { type: string }
    total_cost_usd: { type: number, format: double }
    run_count:     { type: integer, format: int64 }

RunCostDetail:
  type: object
  required: [run_id, total_cost_usd, steps]
  properties:
    run_id:        { type: string, format: uuid }
    total_cost_usd: { type: number, format: double }
    steps:
      type: array
      items: { $ref: '#/components/schemas/RunStepCostItem' }

RunStepCostItem:
  type: object
  required: [step_name, cost_usd, tokens_input, tokens_output, model]
  properties:
    step_name:    { type: string }
    cost_usd:     { type: number, format: double }
    tokens_input: { type: integer, format: int64 }
    tokens_output: { type: integer, format: int64 }
    model:        { type: string }
```

**GetRunDetail cross-project validation pattern:**

`GetRunDetail` must verify that the run belongs to the requested project. Since `ListCostsByRun` only takes `runID`, the service should first call `SumCostByRun` (which joins via run_steps but does not validate project), then verify ownership via an existing `RunRepository.GetRun` call. If `run.ProjectID != projectID`, return `domainerrors.NewNotFound("run", runID.String())` to avoid leaking existence.

Inject `RunRepository` into `CostService` for this check. Follow the existing service constructor pattern: add it as a constructor parameter and bind in wire.go.

### Testing Requirements

**Unit tests (cost_service_aggregation_test.go) — table-driven:**

- `GetProjectSummary("7d")` → calls repo with `since = now - 7 days`, returns correct struct
- `GetProjectSummary("1d")` → returns `ValidationError` with code containing `"INVALID_PERIOD"` or `"period"`
- `ListByRun(limit=100)` → succeeds
- `ListByRun(limit=101)` → returns `ValidationError`
- `GetRunDetail` with run from different project → returns `NotFoundError`

**Handler unit tests (cost_handler_test.go):**

- Use a mock `CostService` struct with function fields (hand-written, no mockgen)
- Test HTTP status codes and JSON response field presence
- Test query parameter parsing edge cases (`period=90d`, `limit=0`, missing `limit`)

### References

- Existing handler pattern: `backend/internal/api/handler/story_handler.go` (or equivalent)
- Existing service pattern: `backend/internal/domain/service/pipeline_service.go`
- Pagination pattern: `backend/internal/api/handler/` list endpoints (look for `renderJSON` + pagination envelope)
- Port interface pattern: `backend/internal/domain/port/cost_repository.go` (from 9-1)
- Wire provider pattern: `backend/cmd/api/wire.go` (existing CostRepository + CostService providers added in 9-1)

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
