# Story 9.1: [BACK] Cost Tracking Table + Per-Step Cost Recording

Status: ready-for-dev

## Story

As a platform operator, I want per-step cost records persisted after each agent run step completes, So that cost data is available for dashboards and budget enforcement.

## Acceptance Criteria (BDD)

**AC1: cost_records table exists with correct schema**
- **Given** the backend starts with migrations applied
- **When** a migration is executed
- **Then** a `cost_records` table exists with columns: `id` (UUID PK), `run_step_id` (FK run_steps CASCADE), `project_id` (FK projects), `tokens_input` (BIGINT NOT NULL), `tokens_output` (BIGINT NOT NULL), `cost_usd` (DECIMAL(10,6) NOT NULL DEFAULT 0), `model` (VARCHAR NOT NULL), `created_at` (TIMESTAMPTZ DEFAULT now())
- **And** indexes exist on `(run_step_id)` and `(project_id, created_at DESC)`

**AC2: NDJSON cost events are parsed during log streaming**
- **Given** an agent container emits NDJSON lines including `{"type":"cost","input_tokens":1000,"output_tokens":500,"model":"claude-opus-4-6"}`
- **When** the AgentRunAction streams logs
- **Then** cost events are detected by `type == "cost"` and accumulated in memory
- **And** non-cost lines are forwarded to the log event pipeline unmodified

**AC3: Cost records are inserted on step completion**
- **Given** one or more cost events were accumulated during a step's execution
- **When** the step completes (success or failure)
- **Then** a single `cost_records` row is inserted aggregating all cost events for that step
- **And** `cost_usd` is computed from model pricing constants

**AC4: Model pricing is applied correctly**
- **Given** a cost event with a known model name
- **When** cost_usd is computed
- **Then** the following rates apply (per million tokens):
  - `claude-opus-4-6`: input $15, output $75
  - `claude-sonnet-4-5`: input $3, output $15
  - `claude-haiku-4-3`: input $0.25, output $1.25
- **And** an unknown model results in `cost_usd = 0` and a `slog.Warn` log line including `"unknown_model"` field

**AC5: CostRepository and CostService ports are wired**
- **Given** the application starts
- **When** DI wiring runs
- **Then** `CostRepository`, `CostService`, and their sqlc-backed implementations are instantiated without error

## Tasks / Subtasks

- [ ] [BACK] Task 1: DB migration for cost_records table (AC: #1)
  - [ ] Create `backend/migrations/000013_create_cost_records_table.up.sql` with table + indexes
  - [ ] Create `backend/migrations/000013_create_cost_records_table.down.sql` with `DROP TABLE cost_records`

- [ ] [BACK] Task 2: sqlc queries for cost data (AC: #1, #5)
  - [ ] Create `backend/queries/cost_records.sql` with: `InsertCostRecord :one`, `GetCostByRunStep :one`, `SumCostByProject :one`, `SumCostByRun :one`
  - [ ] Run `cd backend && sqlc generate` to regenerate `internal/adapter/postgres/db/`

- [ ] [BACK] Task 3: CostRecord domain model (AC: #3, #4)
  - [ ] Create `backend/internal/domain/model/cost_record.go`
  - [ ] Define `CostRecord` struct: `ID`, `RunStepID`, `ProjectID`, `TokensInput`, `TokensOutput`, `CostUSD`, `Model`, `CreatedAt`
  - [ ] Define pricing constants: `ModelPricing` map from model name to `{InputPerMTok, OutputPerMTok float64}`
  - [ ] Export `ComputeCostUSD(model string, inputTokens, outputTokens int64) (float64, bool)` pure function

- [ ] [BACK] Task 4: CostRepository port + postgres adapter (AC: #1, #5)
  - [ ] Create `backend/internal/domain/port/cost_repository.go` with interface: `InsertCostRecord`, `GetCostByRunStep`, `SumCostByProject`, `SumCostByRun`
  - [ ] Create `backend/internal/adapter/postgres/cost_repository.go` implementing the port using sqlc queries

- [ ] [BACK] Task 5: CostService with RecordStepCost business logic (AC: #3, #4)
  - [ ] Create `backend/internal/domain/service/cost_service.go`
  - [ ] `RecordStepCost(ctx, stepID, projectID uuid.UUID, events []CostEvent) error`: aggregates tokens, calls `ComputeCostUSD`, inserts via `CostRepository`
  - [ ] Emit `slog.Warn` for unknown models with `model` field
  - [ ] Unit test: `backend/internal/domain/service/cost_service_test.go` (table-driven, mock repo)

- [ ] [BACK] Task 6: Extend LogEvent model with cost fields (AC: #2)
  - [ ] Add to `backend/internal/domain/model/log_event.go`: `Type string`, `InputTokens int64`, `OutputTokens int64`, `Model string`
  - [ ] `Type` is populated from `data["type"]` when `IsJSON == true`

- [ ] [BACK] Task 7: Parse + accumulate cost events in AgentRunAction (AC: #2, #3)
  - [ ] In `backend/internal/adapter/action/agent_run.go`: inject `CostService` dependency
  - [ ] During log streaming loop: detect `logEvent.Type == "cost"`, extract `input_tokens`, `output_tokens`, `model` from `logEvent.Data`, append to step-local `[]model.CostEvent` slice
  - [ ] On step completion (before return): call `costSvc.RecordStepCost(ctx, stepID, projectID, costEvents)` if `len(costEvents) > 0`

- [ ] [BACK] Task 8: Wire CostRepository and CostService into DI (AC: #5)
  - [ ] Add providers to `backend/cmd/api/wire.go`: `postgres.NewCostRepository`, `service.NewCostService`
  - [ ] Inject `CostService` into `AgentRunAction` constructor
  - [ ] Regenerate: `cd backend && wire ./cmd/api/`

- [ ] [BACK] Task 9: Integration test for cost recording (AC: #3, #4)
  - [ ] Create `backend/internal/adapter/postgres/cost_repository_integration_test.go`
  - [ ] Test `InsertCostRecord` and `SumCostByProject` against ephemeral Postgres via testcontainers

- [ ] [BACK] Task 10: Lint validation (AC: #5)
  - [ ] Run `cd backend && golangci-lint run ./...` — must pass clean
  - [ ] Run `cd backend && go test ./... -short` — unit tests must pass

## Dev Notes

### Dependencies

- **Story 3.5 (DONE):** NDJSON log streaming in `AgentRunAction` — the streaming loop exists in `agent_run.go`
- **Migration sequence:** Last migration is `000012_seed_default_prompt_templates` — use `000013`

### Architecture Requirements

Port/adapter boundaries:

```
AgentRunAction (adapter/action)
    └─ injects CostService (domain/service)
              └─ injects CostRepository (domain/port)
                          └─ implemented by postgres.CostRepository (adapter/postgres)
                                      └─ uses sqlc-generated db.Queries
```

`CostService` must depend only on `CostRepository` (port interface), never on the postgres adapter directly.

### File Paths (exact)

```
backend/migrations/000013_create_cost_records_table.up.sql     (new)
backend/migrations/000013_create_cost_records_table.down.sql   (new)
backend/queries/cost_records.sql                               (new)
backend/internal/domain/model/cost_record.go                   (new)
backend/internal/domain/model/log_event.go                     (extend: Type, InputTokens, OutputTokens, Model)
backend/internal/domain/port/cost_repository.go                (new)
backend/internal/domain/service/cost_service.go                (new)
backend/internal/domain/service/cost_service_test.go           (new)
backend/internal/adapter/postgres/cost_repository.go           (new)
backend/internal/adapter/postgres/cost_repository_integration_test.go  (new)
backend/internal/adapter/action/agent_run.go                   (extend: inject CostService, accumulate/record cost events)
backend/cmd/api/wire.go                                        (extend: add CostRepository + CostService providers)
```

### Technical Specifications

**Migration up (000013):**
```sql
CREATE TABLE cost_records (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_step_id UUID NOT NULL REFERENCES run_steps(id) ON DELETE CASCADE,
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    tokens_input  BIGINT NOT NULL DEFAULT 0,
    tokens_output BIGINT NOT NULL DEFAULT 0,
    cost_usd    DECIMAL(10,6) NOT NULL DEFAULT 0,
    model       VARCHAR NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cost_records_run_step_id ON cost_records(run_step_id);
CREATE INDEX idx_cost_records_project_created ON cost_records(project_id, created_at DESC);
```

**sqlc queries (cost_records.sql):**
```sql
-- name: InsertCostRecord :one
INSERT INTO cost_records (run_step_id, project_id, tokens_input, tokens_output, cost_usd, model)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetCostByRunStep :one
SELECT * FROM cost_records WHERE run_step_id = $1 LIMIT 1;

-- name: SumCostByProject :one
SELECT COALESCE(SUM(cost_usd), 0)::DECIMAL(10,6) AS total_cost,
       COALESCE(SUM(tokens_input), 0)             AS total_input,
       COALESCE(SUM(tokens_output), 0)            AS total_output
FROM cost_records
WHERE project_id = $1 AND created_at >= $2;

-- name: SumCostByRun :one
SELECT COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost
FROM cost_records cr
JOIN run_steps rs ON rs.id = cr.run_step_id
WHERE rs.run_id = $1;
```

**CostRecord model + pricing:**
```go
// Model pricing: USD per million tokens
var modelPricing = map[string][2]float64{
    "claude-opus-4-6":   {15.0, 75.0},  // input, output
    "claude-sonnet-4-5": {3.0, 15.0},
    "claude-haiku-4-3":  {0.25, 1.25},
}

// ComputeCostUSD returns (costUSD, known). known=false for unrecognized models.
func ComputeCostUSD(model string, inputTokens, outputTokens int64) (float64, bool) {
    pricing, ok := modelPricing[model]
    if !ok {
        return 0, false
    }
    cost := (float64(inputTokens)/1_000_000)*pricing[0] +
            (float64(outputTokens)/1_000_000)*pricing[1]
    return cost, true
}
```

**CostEvent internal type (in model package):**
```go
// CostEvent is an intermediate accumulation type parsed from agent NDJSON output.
type CostEvent struct {
    InputTokens  int64
    OutputTokens int64
    Model        string
}
```

**Cost accumulation in agent_run.go (pseudo-code pattern):**
```go
var costEvents []model.CostEvent

// inside log streaming loop:
if logEvent.Type == "cost" {
    costEvents = append(costEvents, model.CostEvent{
        InputTokens:  logEvent.InputTokens,
        OutputTokens: logEvent.OutputTokens,
        Model:        logEvent.Model,
    })
}

// after step completion:
if len(costEvents) > 0 {
    if err := a.costSvc.RecordStepCost(ctx, stepID, projectID, costEvents); err != nil {
        a.logger.Warn("failed to record step cost", "step_id", stepID, "err", err)
        // non-fatal: cost recording failure must not fail the step
    }
}
```

**CostRepository port:**
```go
type CostRepository interface {
    InsertCostRecord(ctx context.Context, record *model.CostRecord) (*model.CostRecord, error)
    GetCostByRunStep(ctx context.Context, runStepID uuid.UUID) (*model.CostRecord, error)
    SumCostByProject(ctx context.Context, projectID uuid.UUID, since time.Time) (totalCost float64, totalInput, totalOutput int64, err error)
    SumCostByRun(ctx context.Context, runID uuid.UUID) (float64, error)
}
```

### Testing Requirements

**Unit tests (cost_service_test.go):**
- Known model → correct cost_usd computed (table-driven: opus, sonnet, haiku)
- Unknown model → cost_usd = 0, repo called with 0, no error returned
- Multiple cost events aggregated before insert (single InsertCostRecord call)
- RecordStepCost with empty events slice → no-op (no repo call)
- Repo returns error → error propagated

**Integration tests (cost_repository_integration_test.go):**
- `InsertCostRecord` roundtrip: insert, fetch by run_step_id, verify fields
- `SumCostByProject` with multiple records in time range

### References

- Existing migration pattern: `backend/migrations/000011_create_stories_table.up.sql`
- sqlc query pattern: `backend/queries/run_steps.sql`
- Port interface pattern: `backend/internal/domain/port/run_repository.go`
- Service pattern: `backend/internal/domain/service/` (existing services)
- AgentRunAction constructor: `backend/internal/adapter/action/agent_run.go` lines 49-73

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
