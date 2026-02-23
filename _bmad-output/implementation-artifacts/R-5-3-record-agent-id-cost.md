# Story R-5-3: [BACK] Record agent_id when calling RecordStepCost

Status: ready-for-dev

## Story

As a **platform developer**,
I want the agent ID to be persisted in cost_records when a pipeline step completes,
so that cost data is properly attributed to the agent that ran the step and agent-level cost breakdowns are accurate.

## Acceptance Criteria (BDD)

### Scenario 1: RecordStepCost persists agent_id when provided

```gherkin
Given a pipeline step that has an agent_id in its configuration
When the AgentRunAction completes and calls RecordStepCost
Then the resulting cost_record row in the database has agent_id set to the step's agent UUID
```

### Scenario 2: RecordStepCost works without agent_id (backward compat)

```gherkin
Given a pipeline step that has no agent_id in its configuration
When the AgentRunAction completes and calls RecordStepCost
Then the resulting cost_record row has agent_id = NULL
  And no error is returned
```

### Scenario 3: CostService.RecordStepCost signature accepts optional agentID

```gherkin
Given the updated CostService
When I call RecordStepCost with agentID = nil
Then the cost record is saved with agent_id = NULL
When I call RecordStepCost with agentID = &someUUID
Then the cost record is saved with agent_id = someUUID
```

### Scenario 4: sqlc INSERT query includes agent_id

```gherkin
Given the updated cost_records.sql and regenerated sqlc code
When I run "cd backend && make generate"
Then the command exits with code 0
  And the generated CreateCostRecord function accepts an agent_id parameter
```

### Scenario 5: Lint and tests pass

```gherkin
Given all code changes including updated tests
When I run golangci-lint and go test -short
Then no lint errors are reported
  And all tests pass including updated RecordStepCost tests
```

## Tasks / Subtasks

- [ ] **1.1** [BACK] Update sqlc INSERT query in `backend/queries/cost_records.sql` to include `agent_id` column (AC: #4)
  - [ ] Add `agent_id` to the INSERT column list and `$N` parameter
  - [ ] Annotate parameter as nullable UUID

- [ ] **1.2** [BACK] Run `cd backend && make generate` to regenerate sqlc code (AC: #4)

- [ ] **1.3** [BACK] Update `CostService.RecordStepCost` signature in `backend/internal/domain/service/cost_service.go` to accept `agentID *uuid.UUID` (AC: #3)
  - [ ] Pass agentID through to the repository/sqlc call
  - [ ] Keep existing callers compiling by updating all call sites

- [ ] **1.4** [BACK] Update `AgentRunAction` in `backend/internal/adapter/action/agent_run.go` to extract the agent_id from the step config and pass it to RecordStepCost (AC: #1, #2)
  - [ ] Look up agent_id from `RunStep.Config` or `RunContext` metadata
  - [ ] Pass `nil` if no agent_id is available (AC: #2)

- [ ] **1.5** [BACK] Update `CostRepository` port `CreateCostRecord` method (if it exists as a named method) or the relevant sqlc params struct to include `AgentID *uuid.UUID` (AC: #3)

- [ ] **1.6** [BACK] Update existing unit tests for `RecordStepCost` to pass the new agentID parameter (AC: #5)
  - [ ] Add test case: RecordStepCost with agentID set
  - [ ] Add test case: RecordStepCost with agentID nil

- [ ] **1.7** [BACK] Lint and test (AC: #5)
  - [ ] `cd backend && golangci-lint run ./...`
  - [ ] `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

- **R-5-1** — the `agent_id` column must exist in `cost_records` before the INSERT can include it. The updated sqlc query will fail to generate until the migration has been applied.
- **R-2-5** — `AgentService` must exist and agents must be resolvable so the AgentRunAction can look up the agent ID. If AgentService is not yet available, fall back to reading agent_id directly from RunStep config/metadata.

### Architecture Requirements

- `AgentRunAction` is in `backend/internal/adapter/action/agent_run.go` — this is where CostEvents are accumulated and `RecordStepCost` is called after the container exits
- The agent_id for a step is available via the step's pipeline configuration (the agent reference linked to the step)
- If no agent_id is configured for the step, pass `nil` — do not fail the run

### Technical Specifications

**Updated RecordStepCost signature:**

```go
// RecordStepCost records the cost for a completed run step, optionally attributed to an agent.
func (s *CostService) RecordStepCost(
    ctx context.Context,
    runID uuid.UUID,
    stepID uuid.UUID,
    tokens int64,
    tokensInput int64,
    tokensOutput int64,
    costUSD float64,
    agentID *uuid.UUID,
) error
```

**AgentRunAction — extracting agent_id:**

The `RunContext` carries `RunStep` which has a `Config` map (or agent reference). Look for the agent_id in:
1. `runCtx.RunStep.AgentID` if the field exists on the RunStep model
2. `runCtx.RunStep.Config["agent_id"]` as a UUID string fallback
3. `nil` if neither is available

**Updated sqlc query fragment:**

```sql
-- name: CreateCostRecord :one
INSERT INTO cost_records (
  id, run_id, step_id, tokens_input, tokens_output, cost_usd, agent_id, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, now()
)
RETURNING *;
```

### Testing Requirements

- Update all existing call sites of `RecordStepCost` that currently pass fewer parameters
- Add two focused unit tests in cost_service_test.go:
  - `TestRecordStepCost_WithAgentID` — verifies agent_id is persisted
  - `TestRecordStepCost_WithoutAgentID` — verifies nil agent_id is accepted
- Use mock repository; no testcontainers required for unit tests

### References

- `backend/internal/adapter/action/agent_run.go` — AgentRunAction, update CostService call
- `backend/internal/domain/service/cost_service.go` — update RecordStepCost signature
- `backend/queries/cost_records.sql` — update INSERT query
- `backend/internal/domain/model/cost_record.go` — CostRecord model (AgentID field added in R-5-1)
- `backend/internal/domain/port/cost_repository.go` — repository port

## Dev Agent Record

## Change Log
