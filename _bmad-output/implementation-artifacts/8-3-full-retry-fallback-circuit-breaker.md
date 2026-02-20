# Story 8.3: [BACK] Full Retry Fallback + Circuit Breaker

Status: ready-for-dev

## Story

As a pipeline executor, I want a circuit breaker that opens after repeated failures and a full retry mechanism that re-runs the entire pipeline from scratch, so that transient failures get a second chance while persistent failures are isolated and don't waste resources.

## Acceptance Criteria (BDD)

**AC1: DB migration adds circuit breaker fields to stories**
- **Given** migration `000015_add_circuit_breaker_to_stories` is applied
- **When** the `stories` table is inspected
- **Then** it has columns: `circuit_breaker_state VARCHAR(10) NOT NULL DEFAULT 'closed' CHECK (circuit_breaker_state IN ('closed', 'open'))` and `consecutive_failures INT NOT NULL DEFAULT 0`
- **And** the `Story` domain model and sqlc queries reflect these new fields
- **And** `CircuitBreakerStateClosed = "closed"` and `CircuitBreakerStateOpen = "open"` constants are defined in the model package

**AC2: PipelineExecutor rejects run start when circuit breaker is open**
- **Given** a story has `circuit_breaker_state = 'open'`
- **When** `PipelineExecutor.ExecuteRun` is called for a run belonging to that story
- **Then** it returns an error with code `CIRCUIT_OPEN` without executing any step
- **And** the run is marked `failed` with `error_message = "CIRCUIT_OPEN: circuit breaker is open for story {storyKey}"`

**AC3: Consecutive failure counter increments on each failed run**
- **Given** a story has `circuit_breaker_state = 'closed'` and `consecutive_failures = N`
- **When** a run for that story completes with status `failed`
- **Then** `consecutive_failures` is incremented to `N + 1` via `StoryRepository.IncrementConsecutiveFailures`
- **And** if `N + 1 >= circuit_breaker_threshold` (default 3), `circuit_breaker_state` is set to `'open'`
- **And** a `circuit_breaker.opened` event is published via `EventPublisher`

**AC4: Consecutive failure counter resets on successful run**
- **Given** a story has `consecutive_failures > 0`
- **When** a run for that story completes with status `completed`
- **Then** `consecutive_failures` is reset to `0` and `circuit_breaker_state` remains `'closed'` via `StoryRepository.ResetConsecutiveFailures`

**AC5: Circuit breaker reset endpoint**
- **Given** the story exists and its `circuit_breaker_state = 'open'`
- **When** `POST /api/v1/projects/{projectId}/stories/{storyKey}/circuit-breaker/reset` is called by an authenticated admin user
- **Then** `circuit_breaker_state` is set to `'closed'` and `consecutive_failures` is reset to `0`
- **And** a `circuit_breaker.reset` event is published via `EventPublisher`
- **And** the response is `HTTP 200` with the updated story object
- **And** if the story does not exist, the response is `HTTP 404` with error code `STORY_NOT_FOUND`

**AC6: CircuitBreakerService encapsulates all circuit breaker logic**
- **Given** `CircuitBreakerService` is defined in `backend/internal/domain/service/circuit_breaker_service.go`
- **When** `PipelineExecutor` calls `CircuitBreakerService.Check(ctx, storyID)` before a run
- **Then** it returns `nil` if closed or an error with code `CIRCUIT_OPEN` if open
- **And** `CircuitBreakerService.RecordFailure(ctx, storyID)` increments counter and opens if threshold reached
- **And** `CircuitBreakerService.RecordSuccess(ctx, storyID)` resets counter
- **And** `CircuitBreakerService.Reset(ctx, storyID)` forcibly sets state to closed

**AC7: `circuit_breaker.opened` and `circuit_breaker.reset` events are published**
- **Given** the circuit breaker transitions state
- **When** `RecordFailure` crosses the threshold, a `circuit_breaker.opened` event is published with payload `{"story_id": "...", "story_key": "...", "consecutive_failures": N}`
- **And** when `Reset` completes, a `circuit_breaker.reset` event is published with payload `{"story_id": "...", "story_key": "..."}`

**AC8: Unit tests cover all circuit breaker branches**
- **Given** unit tests in `backend/internal/domain/service/__tests__/circuit_breaker_service_test.go`
- **When** tests run with `go test -short`
- **Then** the following cases are covered: check when closed, check when open (CIRCUIT_OPEN error), record failure below threshold, record failure at threshold (opens + event), record success (reset counter), reset (force close + event), story not found on check
- **And** `golangci-lint run ./...` passes with zero errors

## Tasks / Subtasks

- [ ] [BACK] Task 1: DB migration 000015 — add circuit breaker fields to stories (AC: #1)
  - [ ] Create `backend/migrations/000015_add_circuit_breaker_to_stories.up.sql`
  - [ ] Add `circuit_breaker_state VARCHAR(10) NOT NULL DEFAULT 'closed' CHECK (circuit_breaker_state IN ('closed', 'open'))` to `stories`
  - [ ] Add `consecutive_failures INT NOT NULL DEFAULT 0` to `stories`
  - [ ] Create `backend/migrations/000015_add_circuit_breaker_to_stories.down.sql` (DROP COLUMN x2)

- [ ] [BACK] Task 2: Update Story domain model and sqlc queries (AC: #1, #3, #4, #5)
  - [ ] Add `CircuitBreakerState string` and `ConsecutiveFailures int` to `model.Story` in `backend/internal/domain/model/story.go`
  - [ ] Add constants `CircuitBreakerStateClosed = "closed"` and `CircuitBreakerStateOpen = "open"` to `model/story.go`
  - [ ] Add sqlc queries in `backend/queries/stories.sql`:
    - `GetStoryByID :one` — fetch story by UUID (if not already present)
    - `IncrementConsecutiveFailures :one` — `UPDATE stories SET consecutive_failures = consecutive_failures + 1 WHERE id = $1 RETURNING *`
    - `OpenCircuitBreaker :one` — `UPDATE stories SET circuit_breaker_state = 'open' WHERE id = $1 RETURNING *`
    - `ResetConsecutiveFailures :one` — `UPDATE stories SET consecutive_failures = 0, circuit_breaker_state = 'closed' WHERE id = $1 RETURNING *`
    - `ResetCircuitBreaker :one` — `UPDATE stories SET circuit_breaker_state = 'closed', consecutive_failures = 0 WHERE id = $1 RETURNING *`
  - [ ] Run `cd backend && sqlc generate`
  - [ ] Extend `StoryRepository` port in `backend/internal/domain/port/story_repository.go` with: `IncrementConsecutiveFailures(ctx, storyID)`, `OpenCircuitBreaker(ctx, storyID)`, `ResetConsecutiveFailures(ctx, storyID)`, `ResetCircuitBreaker(ctx, storyID)`
  - [ ] Implement new port methods in `backend/internal/adapter/postgres/story_repo.go`

- [ ] [BACK] Task 3: Implement `CircuitBreakerService` (AC: #6, #7)
  - [ ] Create `backend/internal/domain/service/circuit_breaker_service.go`
  - [ ] Define `CircuitBreakerService` struct with deps: `storyRepo port.StoryRepository`, `eventPub port.EventPublisher`, `threshold int`, `logger *slog.Logger`
  - [ ] Implement `NewCircuitBreakerService(storyRepo, eventPub, threshold, logger) *CircuitBreakerService`
  - [ ] Implement `Check(ctx, storyID uuid.UUID) error` — fetch story, return `errors.NewForbidden("circuit_breaker", "CIRCUIT_OPEN: circuit breaker is open for story {storyKey}")` if state is `open`
  - [ ] Implement `RecordFailure(ctx, storyID uuid.UUID) error` — increment counter; if `consecutive_failures >= threshold` call `OpenCircuitBreaker` and publish `circuit_breaker.opened` event
  - [ ] Implement `RecordSuccess(ctx, storyID uuid.UUID) error` — call `ResetConsecutiveFailures`
  - [ ] Implement `Reset(ctx, storyID uuid.UUID) error` — call `ResetCircuitBreaker`, publish `circuit_breaker.reset` event

- [ ] [BACK] Task 4: Integrate `CircuitBreakerService` into `PipelineExecutor` (AC: #2, #3, #4)
  - [ ] Add `circuitBreaker *CircuitBreakerService` field to `PipelineExecutor` struct in `backend/internal/domain/service/pipeline_executor.go`
  - [ ] Update `NewPipelineExecutor` constructor signature to accept `*CircuitBreakerService`
  - [ ] At the start of `ExecuteRun`, call `circuitBreaker.Check(ctx, run.StoryID)` — on error, mark run as failed with the error message and return
  - [ ] At the end of `ExecuteRun`, on run status `completed` call `circuitBreaker.RecordSuccess(ctx, run.StoryID)`; on run status `failed` call `circuitBreaker.RecordFailure(ctx, run.StoryID)`

- [ ] [BACK] Task 5: Add reset endpoint to OpenAPI spec and generate handler (AC: #5)
  - [ ] Add to `api/openapi.yaml` under `/projects/{projectId}/stories/{storyKey}/circuit-breaker/reset`:
    ```yaml
    post:
      operationId: resetCircuitBreaker
      summary: Reset circuit breaker for a story
      tags: [stories]
      parameters:
        - $ref: '#/components/parameters/projectId'
        - name: storyKey
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Circuit breaker reset
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Story'
        '404':
          $ref: '#/components/responses/NotFound'
    ```
  - [ ] Run `cd backend && make generate` to regenerate chi server interface
  - [ ] Implement `ResetCircuitBreaker(w http.ResponseWriter, r *http.Request, projectId string, storyKey string)` in `backend/internal/api/handler/story_handler.go` — delegate to `CircuitBreakerService.Reset`

- [ ] [BACK] Task 6: Wire `CircuitBreakerService` into DI (AC: #6)
  - [ ] In `backend/cmd/api/wire.go`: add `service.NewCircuitBreakerService` to the service provider set with default threshold of 3
  - [ ] Inject `CircuitBreakerService` into `PipelineExecutor` provider call
  - [ ] Inject `CircuitBreakerService` into the story handler (or expose via a thin adapter method on `StoryService`)
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] [BACK] Task 7: Write unit tests (AC: #8)
  - [ ] Create `backend/internal/domain/service/__tests__/circuit_breaker_service_test.go`
  - [ ] **Test: Check — circuit closed** — story has state `"closed"`, Check returns nil
  - [ ] **Test: Check — circuit open** — story has state `"open"`, Check returns CIRCUIT_OPEN error
  - [ ] **Test: Check — story not found** — StoryRepository returns not-found error, Check propagates it
  - [ ] **Test: RecordFailure — below threshold** — consecutive_failures incremented, state remains closed, no event published
  - [ ] **Test: RecordFailure — at threshold** — consecutive_failures reaches threshold, OpenCircuitBreaker called, `circuit_breaker.opened` event published
  - [ ] **Test: RecordSuccess** — ResetConsecutiveFailures called, no event published
  - [ ] **Test: Reset** — ResetCircuitBreaker called, `circuit_breaker.reset` event published
  - [ ] Hand-written mocks for `StoryRepository` and `EventPublisher`
  - [ ] Run `golangci-lint run ./...` — must pass

- [ ] [BACK] Task 8: Write unit tests for PipelineExecutor circuit breaker integration (AC: #2, #3, #4)
  - [ ] Extend `backend/internal/domain/service/__tests__/pipeline_executor_test.go`
  - [ ] **Test: ExecuteRun — CIRCUIT_OPEN** — CircuitBreakerService.Check returns error; no steps executed; run marked failed with CIRCUIT_OPEN message
  - [ ] **Test: ExecuteRun — success path calls RecordSuccess** — run completes normally; RecordSuccess called exactly once
  - [ ] **Test: ExecuteRun — failure path calls RecordFailure** — first step fails; RecordFailure called exactly once
  - [ ] Hand-written mock for `CircuitBreakerService` using a thin interface to allow injection

## Dev Notes

### Dependencies

**Story 8-2 (IncrementalRetryAction — wave 10, done):** `run_steps.retry_count`, `retry_type`, `parent_step_id` fields exist. After `retry_count >= max_incremental`, 8-2 creates a `retry_type = "full"` step using the `"implement"` template. Story 8-3 does NOT re-implement that logic — it adds the circuit breaker layer on top of the existing retry chain.

**Story 8-1 (CIPollAction — wave 10, done):** `CIPollAction` is available in the registry as `"ci_poll"`.

**Story 3-7 (PipelineExecutor — done):** `ExecuteRun` already tracks run status transitions. 8-3 hooks into the success/failure path at the end of `ExecuteRun` to call `CircuitBreakerService.RecordSuccess` or `RecordFailure`.

**Story 3-6 (EventPublisher — done):** `port.EventPublisher` with `Publish(ctx, model.Event) error` is available.

### Architecture Requirements

- `CircuitBreakerService` is a **domain service** in `backend/internal/domain/service/` — it only depends on ports (`StoryRepository`, `EventPublisher`), not on adapters
- The service is NOT an `Action` — it is invoked by `PipelineExecutor`, not by the action registry
- `PipelineExecutor` depends on `CircuitBreakerService` via direct struct composition (not a port interface) — this is acceptable since both are domain services; however, a thin `CircuitBreaker` interface can be introduced if needed for testability (recommended for mocking in PipelineExecutor tests)
- No direct DB access in `CircuitBreakerService` — all reads/writes go through `StoryRepository` port
- The circuit breaker threshold is configurable via `config.yaml` (field `pipeline.circuit_breaker_threshold`, default 3) — injected as `int` at wire time

### File Paths (exact)

```
backend/migrations/000015_add_circuit_breaker_to_stories.up.sql
backend/migrations/000015_add_circuit_breaker_to_stories.down.sql
backend/internal/domain/model/story.go                                  # Add CircuitBreakerState, ConsecutiveFailures + constants
backend/queries/stories.sql                                              # Add IncrementConsecutiveFailures, OpenCircuitBreaker, ResetConsecutiveFailures, ResetCircuitBreaker
backend/internal/domain/port/story_repository.go                        # Extend interface
backend/internal/adapter/postgres/story_repo.go                         # Implement new methods
backend/internal/domain/service/circuit_breaker_service.go              # New service
backend/internal/domain/service/__tests__/circuit_breaker_service_test.go
backend/internal/domain/service/pipeline_executor.go                    # Integrate CircuitBreakerService
backend/internal/domain/service/__tests__/pipeline_executor_test.go     # Extend with CB tests
backend/internal/api/handler/story_handler.go                           # Add ResetCircuitBreaker handler
api/openapi.yaml                                                         # Add reset endpoint
backend/cmd/api/wire.go                                                  # Wire CircuitBreakerService
```

### Technical Specifications

**Migration up (000015):**
```sql
ALTER TABLE stories
    ADD COLUMN circuit_breaker_state VARCHAR(10) NOT NULL DEFAULT 'closed'
        CHECK (circuit_breaker_state IN ('closed', 'open')),
    ADD COLUMN consecutive_failures INT NOT NULL DEFAULT 0;
```

**Migration down (000015):**
```sql
ALTER TABLE stories
    DROP COLUMN IF EXISTS consecutive_failures,
    DROP COLUMN IF EXISTS circuit_breaker_state;
```

**Updated Story model:**
```go
// Circuit breaker state constants.
const (
    CircuitBreakerStateClosed = "closed"
    CircuitBreakerStateOpen   = "open"
)

// Story represents a user story within a project.
type Story struct {
    ID                   uuid.UUID
    ProjectID            uuid.UUID
    EpicID               *uuid.UUID
    Key                  string
    Title                string
    Objective            *string
    TargetFiles          []string
    DependsOn            []string
    Scope                *string
    Status               string
    AcceptanceCriteria   *string
    CircuitBreakerState  string // "closed" | "open"
    ConsecutiveFailures  int
    CreatedAt            time.Time
    UpdatedAt            time.Time
}
```

**CircuitBreakerService:**
```go
// CircuitBreakerService manages per-story circuit breaker state.
// It opens after consecutive run failures and must be manually reset.
type CircuitBreakerService struct {
    storyRepo port.StoryRepository
    eventPub  port.EventPublisher
    threshold int // number of consecutive failures before opening
    logger    *slog.Logger
}

// NewCircuitBreakerService creates a new CircuitBreakerService.
func NewCircuitBreakerService(
    storyRepo port.StoryRepository,
    eventPub  port.EventPublisher,
    threshold int,
    logger    *slog.Logger,
) *CircuitBreakerService

// Check returns CIRCUIT_OPEN error if the circuit is open for the given story.
func (s *CircuitBreakerService) Check(ctx context.Context, storyID uuid.UUID) error

// RecordFailure increments the failure counter and opens the circuit if threshold is reached.
func (s *CircuitBreakerService) RecordFailure(ctx context.Context, storyID uuid.UUID) error

// RecordSuccess resets the failure counter to 0.
func (s *CircuitBreakerService) RecordSuccess(ctx context.Context, storyID uuid.UUID) error

// Reset forcibly closes the circuit and resets the counter (admin action).
func (s *CircuitBreakerService) Reset(ctx context.Context, storyID uuid.UUID) error
```

**CircuitBreakerService.Check implementation:**
```go
func (s *CircuitBreakerService) Check(ctx context.Context, storyID uuid.UUID) error {
    story, err := s.storyRepo.GetStoryByID(ctx, storyID)
    if err != nil {
        return fmt.Errorf("fetch story for circuit breaker check: %w", err)
    }
    if story.CircuitBreakerState == model.CircuitBreakerStateOpen {
        return errors.NewForbidden("circuit_breaker",
            fmt.Sprintf("CIRCUIT_OPEN: circuit breaker is open for story %s", story.Key))
    }
    return nil
}
```

**CircuitBreakerService.RecordFailure implementation:**
```go
func (s *CircuitBreakerService) RecordFailure(ctx context.Context, storyID uuid.UUID) error {
    updated, err := s.storyRepo.IncrementConsecutiveFailures(ctx, storyID)
    if err != nil {
        return fmt.Errorf("increment consecutive failures: %w", err)
    }
    if updated.ConsecutiveFailures >= s.threshold {
        opened, err := s.storyRepo.OpenCircuitBreaker(ctx, storyID)
        if err != nil {
            return fmt.Errorf("open circuit breaker: %w", err)
        }
        _ = s.eventPub.Publish(ctx, model.Event{
            EntityType: "circuit_breaker",
            Action:     "opened",
            Payload:    map[string]any{
                "story_id":             opened.ID,
                "story_key":            opened.Key,
                "consecutive_failures": opened.ConsecutiveFailures,
            },
        })
        s.logger.Warn("circuit breaker opened",
            "story_id", opened.ID,
            "story_key", opened.Key,
            "consecutive_failures", opened.ConsecutiveFailures,
        )
    }
    return nil
}
```

**PipelineExecutor integration (additions to ExecuteRun):**
```go
// At start of ExecuteRun, after fetching the run:
if err := e.circuitBreaker.Check(ctx, run.StoryID); err != nil {
    errMsg := err.Error()
    _ = e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusFailed, &errMsg)
    return err
}

// At end of ExecuteRun, after final status is determined:
if runStatus == model.RunStatusCompleted {
    if cbErr := e.circuitBreaker.RecordSuccess(ctx, run.StoryID); cbErr != nil {
        e.logger.Warn("circuit breaker RecordSuccess failed", "error", cbErr, "run_id", runID)
    }
} else if runStatus == model.RunStatusFailed {
    if cbErr := e.circuitBreaker.RecordFailure(ctx, run.StoryID); cbErr != nil {
        e.logger.Warn("circuit breaker RecordFailure failed", "error", cbErr, "run_id", runID)
    }
}
```

**CircuitBreaker interface for testability in PipelineExecutor:**
```go
// circuitBreaker is the interface used by PipelineExecutor to allow mock injection in tests.
type circuitBreaker interface {
    Check(ctx context.Context, storyID uuid.UUID) error
    RecordFailure(ctx context.Context, storyID uuid.UUID) error
    RecordSuccess(ctx context.Context, storyID uuid.UUID) error
}
// CircuitBreakerService satisfies this interface — no extra work needed.
```

**sqlc queries to add in `backend/queries/stories.sql`:**
```sql
-- name: IncrementConsecutiveFailures :one
UPDATE stories
SET consecutive_failures = consecutive_failures + 1,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: OpenCircuitBreaker :one
UPDATE stories
SET circuit_breaker_state = 'open',
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ResetConsecutiveFailures :one
UPDATE stories
SET consecutive_failures = 0,
    circuit_breaker_state = 'closed',
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ResetCircuitBreaker :one
UPDATE stories
SET circuit_breaker_state = 'closed',
    consecutive_failures = 0,
    updated_at = now()
WHERE id = $1
RETURNING *;
```

**Events published:**

| Entity type | Action | Trigger | Payload |
|---|---|---|---|
| `circuit_breaker` | `opened` | `consecutive_failures >= threshold` | `{story_id, story_key, consecutive_failures}` |
| `circuit_breaker` | `reset` | Admin calls reset endpoint | `{story_id, story_key}` |

**Error codes:**
- `CIRCUIT_OPEN` — circuit breaker is open for the story; new runs blocked
- `STORY_NOT_FOUND` — story does not exist (from existing pattern)

**Default config value:**
```yaml
# config.yaml
pipeline:
  circuit_breaker_threshold: 3  # open after 3 consecutive failed runs
```

**Mock StoryRepository (partial — for CircuitBreakerService tests):**
```go
type mockStoryRepo struct {
    GetStoryByIDFn              func(ctx context.Context, id uuid.UUID) (*model.Story, error)
    IncrementConsecutiveFailuresFn func(ctx context.Context, id uuid.UUID) (*model.Story, error)
    OpenCircuitBreakerFn        func(ctx context.Context, id uuid.UUID) (*model.Story, error)
    ResetConsecutiveFailuresFn  func(ctx context.Context, id uuid.UUID) (*model.Story, error)
    ResetCircuitBreakerFn       func(ctx context.Context, id uuid.UUID) (*model.Story, error)
}
```

**Mock EventPublisher:**
```go
type mockEventPublisher struct {
    Published []model.Event
}
func (m *mockEventPublisher) Publish(_ context.Context, e model.Event) error {
    m.Published = append(m.Published, e)
    return nil
}
```

### References

- `backend/internal/domain/model/story.go` — Story model (extend with CB fields)
- `backend/internal/domain/port/story_repository.go` — StoryRepository interface (extend)
- `backend/internal/adapter/postgres/story_repo.go` — postgres implementation (extend)
- `backend/internal/domain/service/pipeline_executor.go` — ExecuteRun (integrate CB check)
- `backend/internal/domain/service/__tests__/pipeline_executor_test.go` — existing tests (extend)
- `backend/internal/api/handler/story_handler.go` — story handler (add reset endpoint)
- `backend/internal/domain/service/run_service.go` — observe pattern for service composition
- `backend/cmd/api/wire.go` — DI wiring location
- `api/openapi.yaml` — add reset endpoint before regenerating
- `backend/.golangci.yml` — lint configuration
- `backend/pkg/errors/` — `NewForbidden`, `NewNotFound` constructors

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
