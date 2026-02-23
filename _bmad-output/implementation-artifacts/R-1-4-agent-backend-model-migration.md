# Story R-1-4: [BACK] Agent model + DB migration (rename prompt_templates → agents)

Status: ready-for-dev

## Story

As a **platform developer**,
I want the `prompt_templates` table renamed to `agents` with additional columns, and the domain model updated accordingly,
so that the Agent entity is a first-class concept in the data model with support for global and project-scoped agents.

## Acceptance Criteria (BDD)

### Scenario 1: Database migration runs successfully

```gherkin
Given the database is on the migration before this one
When the up migration is applied
Then the "agents" table exists (renamed from prompt_templates)
  And the "agents" table has a "scope" column of type VARCHAR(10) NOT NULL DEFAULT 'project'
  And the "agents" table has a "model" column of type VARCHAR(100) nullable
  And the "agents" table has an "image" column of type VARCHAR(255) nullable
  And the "project_id" column is nullable (was NOT NULL)
  And all existing prompt_template rows are preserved with scope = 'project'
```

### Scenario 2: Down migration restores the original schema

```gherkin
Given the up migration has been applied
When the down migration is applied
Then the "prompt_templates" table is restored
  And the "scope", "model", "image" columns are removed
  And the "project_id" column is restored as NOT NULL
```

### Scenario 3: Agent domain model compiles and is correct

```gherkin
Given the backend compiles successfully
When I inspect backend/internal/domain/model/agent.go
Then an Agent struct exists with fields:
  | field            | type          | notes                         |
  | ID               | uuid.UUID     |                               |
  | Name             | string        |                               |
  | Model            | string        | e.g. "claude-opus-4-6"        |
  | Image            | string        | Docker image reference        |
  | TemplateContent  | string        |                               |
  | Scope            | string        | "global" or "project"         |
  | ProjectID        | *uuid.UUID    | nullable pointer              |
  | CreatedAt        | time.Time     |                               |
  | UpdatedAt        | time.Time     |                               |
```

### Scenario 4: AgentRepository port interface exists

```gherkin
Given the backend compiles successfully
When I inspect backend/internal/domain/port/
Then an AgentRepository interface exists with methods:
  | method                      | signature summary                                      |
  | CreateAgent                 | (ctx, agent) (*model.Agent, error)                    |
  | GetAgent                    | (ctx, id uuid.UUID) (*model.Agent, error)             |
  | ListAgentsByProject         | (ctx, projectID uuid.UUID) ([]*model.Agent, error)    |
  | ListGlobalAgents            | (ctx) ([]*model.Agent, error)                         |
  | ListAgentsByProjectMerged   | (ctx, projectID uuid.UUID) ([]*model.Agent, error)    |
  | UpdateAgent                 | (ctx, agent *model.Agent) (*model.Agent, error)       |
  | DeleteAgent                 | (ctx, id uuid.UUID) error                             |
```

### Scenario 5: sqlc queries are renamed and regenerated

```gherkin
Given the backend compiles successfully after "make generate"
When I inspect backend/queries/agents.sql
Then it contains query definitions named:
  | query name                  | type    |
  | CreateAgent                 | :one    |
  | GetAgent                    | :one    |
  | ListAgentsByProject         | :many   |
  | ListGlobalAgents            | :many   |
  | ListAgentsByProjectMerged   | :many   |
  | UpdateAgent                 | :one    |
  | DeleteAgent                 | :exec   |
```

### Scenario 6: ListGlobalAgents returns only global-scoped agents

```gherkin
Given agents exist with scope "global" and scope "project"
When ListGlobalAgents is called
Then only agents with scope = 'global' are returned
```

### Scenario 7: ListAgentsByProjectMerged returns project + global agents

```gherkin
Given a project with ID "proj-1"
  And agents exist: two project-scoped for "proj-1", one project-scoped for "proj-2", two global
When ListAgentsByProjectMerged is called with projectID = "proj-1"
Then 4 agents are returned: the two project-scoped agents for "proj-1" plus the two global agents
  And the project-scoped agent for "proj-2" is not returned
```

### Scenario 8: Old PromptTemplate references are removed

```gherkin
Given the backend compiles successfully
When I search the backend source for "PromptTemplate"
Then no references remain (except inside comments documenting the rename)
  And "PromptTemplateRepository" interface no longer exists in backend/internal/domain/port/
```

## Technical Notes

### Database Migration

**File:** `backend/migrations/000028_rename_prompt_templates_to_agents.up.sql`

```sql
-- Rename table
ALTER TABLE prompt_templates RENAME TO agents;

-- Add new columns
ALTER TABLE agents
    ADD COLUMN scope  VARCHAR(10)  NOT NULL DEFAULT 'project',
    ADD COLUMN model  VARCHAR(100),
    ADD COLUMN image  VARCHAR(255);

-- Make project_id nullable (global agents have no project)
ALTER TABLE agents ALTER COLUMN project_id DROP NOT NULL;

-- Rename constraints and indexes to match new table name
ALTER INDEX IF EXISTS idx_prompt_templates_project_id RENAME TO idx_agents_project_id;
ALTER TABLE agents RENAME CONSTRAINT IF EXISTS prompt_templates_pkey TO agents_pkey;
-- Rename any FK constraints referencing this table (adjust names to match actual schema)
```

**File:** `backend/migrations/000028_rename_prompt_templates_to_agents.down.sql`

```sql
-- Reverse column changes
ALTER TABLE agents ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE agents
    DROP COLUMN IF EXISTS scope,
    DROP COLUMN IF EXISTS model,
    DROP COLUMN IF EXISTS image;

-- Rename back
ALTER TABLE agents RENAME TO prompt_templates;

-- Restore index/constraint names
ALTER INDEX IF EXISTS idx_agents_project_id RENAME TO idx_prompt_templates_project_id;
ALTER TABLE prompt_templates RENAME CONSTRAINT IF EXISTS agents_pkey TO prompt_templates_pkey;
```

### Domain Model

**File:** `backend/internal/domain/model/agent.go` (new file, replaces `prompt_template.go`)

```go
package model

import (
    "time"

    "github.com/google/uuid"
)

// Agent represents an AI agent definition with its runtime configuration and prompt template.
// Agents can be scoped globally (available to all projects) or to a specific project.
type Agent struct {
    ID              uuid.UUID  `json:"id"`
    Name            string     `json:"name"`
    Model           string     `json:"model"`
    Image           string     `json:"image"`
    TemplateContent string     `json:"template_content"`
    Scope           string     `json:"scope"` // "global" or "project"
    ProjectID       *uuid.UUID `json:"project_id"`
    CreatedAt       time.Time  `json:"created_at"`
    UpdatedAt       time.Time  `json:"updated_at"`
}
```

Delete `backend/internal/domain/model/prompt_template.go`.

### Port Interface

**File:** `backend/internal/domain/port/agent_repository.go` (new file, replaces `prompt_template_repository.go`)

```go
package port

import (
    "context"

    "github.com/google/uuid"
    "github.com/yourorg/hopeitworks/backend/internal/domain/model"
)

// AgentRepository defines persistence operations for Agent entities.
type AgentRepository interface {
    CreateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
    GetAgent(ctx context.Context, id uuid.UUID) (*model.Agent, error)
    ListAgentsByProject(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
    // ListGlobalAgents returns all agents with scope = "global".
    ListGlobalAgents(ctx context.Context) ([]*model.Agent, error)
    // ListAgentsByProjectMerged returns all agents scoped to projectID plus all global agents.
    ListAgentsByProjectMerged(ctx context.Context, projectID uuid.UUID) ([]*model.Agent, error)
    UpdateAgent(ctx context.Context, agent *model.Agent) (*model.Agent, error)
    DeleteAgent(ctx context.Context, id uuid.UUID) error
}
```

Delete `backend/internal/domain/port/prompt_template_repository.go`.

### sqlc Queries

**File:** `backend/queries/agents.sql` (renamed from `prompt_templates.sql`)

```sql
-- name: CreateAgent :one
INSERT INTO agents (id, name, model, image, template_content, scope, project_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
RETURNING *;

-- name: GetAgent :one
SELECT * FROM agents WHERE id = $1 LIMIT 1;

-- name: ListAgentsByProject :many
SELECT * FROM agents WHERE project_id = $1 ORDER BY name ASC;

-- name: ListGlobalAgents :many
SELECT * FROM agents WHERE scope = 'global' ORDER BY name ASC;

-- name: ListAgentsByProjectMerged :many
SELECT * FROM agents
WHERE project_id = $1 OR scope = 'global'
ORDER BY scope DESC, name ASC;

-- name: UpdateAgent :one
UPDATE agents
SET name = $2, model = $3, image = $4, template_content = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAgent :exec
DELETE FROM agents WHERE id = $1;
```

Delete `backend/queries/prompt_templates.sql`.

### Code Generation

After applying model, port, and query changes:

```bash
cd backend && make generate
```

This runs sqlc (regenerates `backend/internal/adapter/postgres/db/` query files) and oapi-codegen (regenerates handler interfaces from the updated `api/openapi.yaml` from story R-1-2).

### Postgres Adapter

**File:** `backend/internal/adapter/postgres/agent_repository.go` (new file, replaces `prompt_template_repository.go`)

Implement `AgentRepository` interface using the sqlc-generated queries. Each method maps the sqlc-generated `db.Agent` struct to `model.Agent`.

### Handler Stubs

If R-1-2 introduced new oapi-codegen server interface methods for agent endpoints, add full implementations in `backend/internal/api/handler/agent_handler.go`. The handler delegates to a new `AgentService` or directly to `AgentRepository` via a service layer following hexagonal conventions.

### Wire / DI

Update `backend/cmd/api/wire.go`:
- Remove `PromptTemplateRepository` / `PromptTemplateService` provider bindings
- Add `AgentRepository` binding (postgres adapter)
- Regenerate `wire_gen.go`: `cd backend && wire ./cmd/api/`

## Tasks / Subtasks

### 1. Database Migration

- [ ] **1.1** Determine the next available migration number (check `backend/migrations/` for the highest `0000XX` prefix) (AC: #1)
- [ ] **1.2** Create `backend/migrations/000028_rename_prompt_templates_to_agents.up.sql` (AC: #1)
- [ ] **1.3** Create `backend/migrations/000028_rename_prompt_templates_to_agents.down.sql` (AC: #2)
- [ ] **1.4** Apply migration locally and verify table structure: `psql` inspect columns and constraints (AC: #1)

### 2. Domain Model

- [ ] **2.1** Create `backend/internal/domain/model/agent.go` with `Agent` struct (AC: #3)
- [ ] **2.2** Delete `backend/internal/domain/model/prompt_template.go` (AC: #8)

### 3. Port Interface

- [ ] **3.1** Create `backend/internal/domain/port/agent_repository.go` with `AgentRepository` interface (AC: #4)
- [ ] **3.2** Delete `backend/internal/domain/port/prompt_template_repository.go` (AC: #8)

### 4. sqlc Queries

- [ ] **4.1** Create `backend/queries/agents.sql` with all 7 named queries (AC: #5)
- [ ] **4.2** Delete `backend/queries/prompt_templates.sql` (AC: #8)
- [ ] **4.3** Run `cd backend && make generate` to regenerate sqlc Go code (AC: #5)

### 5. Postgres Adapter

- [ ] **5.1** Create `backend/internal/adapter/postgres/agent_repository.go` implementing `AgentRepository` using sqlc-generated queries (AC: #4, #6, #7)
- [ ] **5.2** Delete `backend/internal/adapter/postgres/prompt_template_repository.go` (AC: #8)
- [ ] **5.3** Map sqlc `db.Agent` ↔ `model.Agent` in the adapter

### 6. Handler

- [ ] **6.1** Create `backend/internal/api/handler/agent_handler.go` implementing the oapi-codegen server interface methods for agent CRUD (AC: #4)
- [ ] **6.2** Delete or rename `backend/internal/api/handler/prompt_template_handler.go` (AC: #8)

### 7. Wire / DI

- [ ] **7.1** Update `backend/cmd/api/wire.go`: remove old PromptTemplate bindings, add AgentRepository and AgentHandler providers (AC: #3)
- [ ] **7.2** Regenerate `wire_gen.go`: `cd backend && wire ./cmd/api/` (AC: #3)

### 8. Tests

- [ ] **8.1** Unit tests for `AgentRepository` adapter: CreateAgent, GetAgent, ListAgentsByProject, ListGlobalAgents, ListAgentsByProjectMerged, UpdateAgent, DeleteAgent (AC: #6, #7)
- [ ] **8.2** Verify `ListGlobalAgents` only returns scope = 'global' rows (AC: #6)
- [ ] **8.3** Verify `ListAgentsByProjectMerged` returns project + global agents, excludes other projects (AC: #7)
- [ ] **8.4** Integration test: run up migration, insert data with both scopes, verify queries return correct results (AC: #1, #6, #7)

### 9. Lint & Verify

- [ ] **9.1** `cd backend && golangci-lint run ./...`
- [ ] **9.2** `cd backend && go test ./... -short`
- [ ] **9.3** `cd backend && go test ./... -run Integration` (integration tests with testcontainers)

## Dev Notes

### Dependencies

**Depends on:** R-1-2 (the Agent schema and endpoints must exist in `api/openapi.yaml` and code must be regenerated before this story can wire up the handler to the generated server interface)

### Architecture Requirements

- Hexagonal boundary: `AgentRepository` interface lives in `port/`, implementation in `adapter/postgres/`
- Services depend on ports (interfaces), never on adapters directly
- sqlc-generated code in `backend/internal/adapter/postgres/db/` is auto-generated — never edit manually
- `wire_gen.go` is auto-generated — always regenerate via `wire ./cmd/api/`

### Migration Number

Check the actual highest migration number in `backend/migrations/` before creating the file. Adjust `000028` if a higher number already exists.

```bash
ls backend/migrations/*.up.sql | sort | tail -3
```

### References

- `backend/internal/domain/model/prompt_template.go` — file to replace
- `backend/internal/domain/port/prompt_template_repository.go` — file to replace
- `backend/queries/prompt_templates.sql` — file to rename/replace
- `backend/internal/adapter/postgres/prompt_template_repository.go` — file to replace
- `backend/internal/api/handler/prompt_template_handler.go` — file to replace
- `backend/cmd/api/wire.go` — DI wiring to update
- Story R-1-2 — OpenAPI spec (prerequisite)

## Dev Agent Record

## Change Log
