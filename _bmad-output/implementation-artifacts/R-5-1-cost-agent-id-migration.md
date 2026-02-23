# Story R-5-1: [BACK] Add agent_id to cost_records + queries by agent

Status: ready-for-dev

## Story

As a **platform developer**,
I want the `cost_records` table to track which agent incurred each cost,
so that the platform can aggregate and report costs broken down by agent identity.

## Acceptance Criteria (BDD)

### Scenario 1: cost_records table has agent_id column

```gherkin
Given the migration is applied to the database
When I inspect the cost_records table schema
Then an "agent_id" column is present of type UUID
  And it has a foreign key to agents(id) with ON DELETE SET NULL
  And an index idx_cost_records_agent_id exists on this column
```

### Scenario 2: CostRecord model carries AgentID field

```gherkin
Given the updated CostRecord Go model
When I read a cost record from the database
Then the AgentID field is populated with the agent's UUID if one was set
  And the field is *uuid.UUID (nullable pointer)
```

### Scenario 3: ListByProjectByAgent query returns aggregated breakdown

```gherkin
Given multiple cost_records exist for a project with different agent_ids
When I call CostRepository.ListByProjectByAgent(ctx, projectID)
Then I receive a []AgentCostBreakdown
  And each entry contains: AgentID, AgentName, TokensInput, TokensOutput, CostUSD, RunsCount
  And the values are correctly aggregated (SUM for tokens/cost, COUNT DISTINCT for runs)
```

### Scenario 4: Backend code generation succeeds

```gherkin
Given the updated queries/cost_records.sql and model
When I run "cd backend && make generate"
Then the command exits with code 0
  And the generated sqlc code includes the ListCostsByProjectByAgent query function
```

### Scenario 5: Lint and tests pass

```gherkin
Given all code changes are in place
When I run golangci-lint and go test -short
Then no lint errors are reported
  And all existing tests pass
```

## Tasks / Subtasks

- [ ] **1.1** [BACK] Create DB migration (next number after 000026): `ALTER TABLE cost_records ADD COLUMN agent_id UUID REFERENCES agents(id) ON DELETE SET NULL` (AC: #1)
  - [ ] Create `000027_add_agent_id_to_cost_records.up.sql` with ALTER TABLE and CREATE INDEX statements
  - [ ] Create `000027_add_agent_id_to_cost_records.down.sql` with DROP INDEX and ALTER TABLE DROP COLUMN

- [ ] **1.2** [BACK] Add `AgentCostBreakdown` model struct to `backend/internal/domain/model/cost_record.go` (AC: #3)
  - [ ] Fields: `AgentID uuid.UUID`, `AgentName string`, `TokensInput int64`, `TokensOutput int64`, `CostUSD float64`, `RunsCount int32`

- [ ] **1.3** [BACK] Update `CostRecord` model in `backend/internal/domain/model/cost_record.go` to add `AgentID *uuid.UUID` field (AC: #2)

- [ ] **1.4** [BACK] Add sqlc query `ListCostsByProjectByAgent` to `backend/queries/cost_records.sql` (AC: #3)
  - [ ] Query: aggregate SUM(tokens_input), SUM(tokens_output), SUM(cost_usd), COUNT(DISTINCT run_id) grouped by agent_id
  - [ ] JOIN on agents table to retrieve agent name
  - [ ] Filter by project_id via JOIN on run_steps/runs
  - [ ] Return nulls gracefully (LEFT JOIN, COALESCE for agent_name)

- [ ] **1.5** [BACK] Add `ListByProjectByAgent(ctx context.Context, projectID uuid.UUID) ([]AgentCostBreakdown, error)` to `CostRepository` port in `backend/internal/domain/port/cost_repository.go` (AC: #3)

- [ ] **1.6** [BACK] Run `cd backend && make generate` and fix any compilation errors (AC: #4)

- [ ] **1.7** [BACK] Lint and test (AC: #5)
  - [ ] `cd backend && golangci-lint run ./...`
  - [ ] `cd backend && go test ./... -short`

## Dev Notes

### Dependencies

- **R-1-4** — agents table must exist before this migration can add the FK reference. This story's migration must run after R-1-4's migration.
- Migration number: 000027 (after 000026_seed_merge_template).

### Architecture Requirements

- Follow hexagonal architecture: model change in `domain/model/`, port update in `domain/port/`, sqlc queries in `queries/`
- Generated sqlc code goes into `backend/internal/adapter/postgres/db/` — never edit manually
- The `agent_id` column is nullable (SET NULL on cascade) so existing cost records without an agent are unaffected

### Technical Specifications

**Migration up (`000027_add_agent_id_to_cost_records.up.sql`):**

```sql
ALTER TABLE cost_records ADD COLUMN agent_id UUID REFERENCES agents(id) ON DELETE SET NULL;
CREATE INDEX idx_cost_records_agent_id ON cost_records(agent_id);
```

**Migration down (`000027_add_agent_id_to_cost_records.down.sql`):**

```sql
DROP INDEX IF EXISTS idx_cost_records_agent_id;
ALTER TABLE cost_records DROP COLUMN IF EXISTS agent_id;
```

**sqlc query (`ListCostsByProjectByAgent`):**

```sql
-- name: ListCostsByProjectByAgent :many
SELECT
  cr.agent_id,
  COALESCE(a.name, 'Unknown') AS agent_name,
  SUM(cr.tokens_input)::bigint AS tokens_input,
  SUM(cr.tokens_output)::bigint AS tokens_output,
  SUM(cr.cost_usd) AS cost_usd,
  COUNT(DISTINCT cr.run_id)::int AS runs_count
FROM cost_records cr
LEFT JOIN agents a ON a.id = cr.agent_id
JOIN runs r ON r.id = cr.run_id
WHERE r.project_id = $1
  AND cr.agent_id IS NOT NULL
GROUP BY cr.agent_id, a.name
ORDER BY cost_usd DESC;
```

**AgentCostBreakdown model:**

```go
// AgentCostBreakdown aggregates cost data per agent for a project.
type AgentCostBreakdown struct {
    AgentID     uuid.UUID
    AgentName   string
    TokensInput  int64
    TokensOutput int64
    CostUSD     float64
    RunsCount   int32
}
```

### Testing Requirements

- Existing cost record tests must still pass (agent_id field is nullable, no breaking change)
- Add a unit test for the sqlc-generated function using testcontainers (or mock) verifying aggregation across multiple cost records with different agent_ids

### References

- `backend/internal/domain/model/cost_record.go` — add AgentID field and AgentCostBreakdown struct
- `backend/internal/domain/port/cost_repository.go` — add ListByProjectByAgent method
- `backend/queries/cost_records.sql` — add ListCostsByProjectByAgent query
- `backend/migrations/` — add 000027 migration files
- `backend/internal/adapter/postgres/db/` — generated by `make generate`, do not edit

## Dev Agent Record

## Change Log
