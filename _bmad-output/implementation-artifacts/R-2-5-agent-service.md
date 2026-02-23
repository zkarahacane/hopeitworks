# Story R-2-5: [BACK] AgentService (replaces PromptTemplateService)

Status: ready-for-dev

## Story

As a **platform developer**,
I want the `PromptTemplateService` and its related layer renamed to `AgentService`,
so that the domain concept of "prompt templates" is replaced by the more accurate concept of "agents" that reflect the R-1-4 data model and support both global and project-scoped agents.

## Acceptance Criteria (BDD)

### Scenario 1: Files and types renamed

```gherkin
Given the current codebase
When Story R-2-5 is implemented
Then no file is named "prompt_template_service.go", "prompt_template_repo.go", or "prompt_template_handler.go"
  And the service struct is named "AgentService"
  And the repository adapter struct is named "AgentRepository" (implementing a renamed port interface)
  And the handler struct is named "AgentHandler"
```

### Scenario 2: Existing CRUD methods preserved

```gherkin
Given the renamed AgentService
When the existing CRUD operations are called (Create, GetByID, ListByProject, Update, Delete)
Then they behave identically to the previous PromptTemplateService methods
  And all existing tests pass without modification to their assertions
```

### Scenario 3: ListGlobal returns agents with no project_id

```gherkin
Given several agents in the database, some with project_id = NULL (global) and some with a project_id (project-scoped)
When AgentService.ListGlobal(ctx) is called
Then only agents with project_id = NULL are returned
  And agents with a project_id are excluded
```

### Scenario 4: ListByProjectMerged returns global + project agents

```gherkin
Given 2 global agents and 3 project-specific agents for project P1
When AgentService.ListByProjectMerged(ctx, P1_ID) is called
Then the result contains all 5 agents (2 global + 3 project-specific)
  And global agents are clearly identifiable (project_id = nil)
```

### Scenario 5: Scope validation enforced

```gherkin
Given an agent creation request with project_id = nil and scope "global"
When AgentService.Create is called
Then the agent is created successfully

Given an agent creation request with project_id = nil and scope "project"
When AgentService.Create is called
Then an error with code VALIDATION is returned explaining that project agents must have a project_id

Given an agent creation request with a valid project_id and scope "project"
When AgentService.Create is called
Then the agent is created successfully
```

### Scenario 6: Routes /agents are added, /templates routes continue to work

```gherkin
Given the updated router configuration
When GET /api/v1/projects/{projectId}/agents is called
Then the response is identical to GET /api/v1/projects/{projectId}/templates

When GET /api/v1/projects/{projectId}/templates is called
Then it still returns HTTP 200 with the agent list (backward compatibility for existing clients)
```

### Scenario 7: DI wiring compiles and starts

```gherkin
Given the renamed types in main.go / wire.go
When "go build ./..." is executed from backend/
Then it exits 0 with no compilation errors
```

### Scenario 8: Lint and tests pass

```gherkin
Given the full implementation
When "golangci-lint run ./..." is executed from backend/
Then it exits 0
When "go test ./... -short" is executed from backend/
Then it exits 0
```

## Tasks / Subtasks

- [ ] [BACK] Task 1: Rename service file and type (AC: #1, #2)
  - [ ] Rename `backend/internal/domain/service/prompt_template_service.go` → `agent_service.go`
  - [ ] Rename struct `PromptTemplateService` → `AgentService`
  - [ ] Rename constructor `NewPromptTemplateService` → `NewAgentService`
  - [ ] Rename param struct `CreatePromptTemplateParams` → `CreateAgentParams`
  - [ ] Rename param struct `UpdatePromptTemplateParams` → `UpdateAgentParams`
  - [ ] Rename result struct `PromptTemplateListResult` → `AgentListResult` with field `Agents []*model.Agent` (or keep `Templates []*model.PromptTemplate` if model rename is deferred — see Task 6)
  - [ ] Update all method signatures to use renamed types
  - [ ] Update all callers (handler, wire) to reference the new names

- [ ] [BACK] Task 2: Rename port interface (AC: #1, #2)
  - [ ] Rename `backend/internal/domain/port/prompt_template_repository.go` → `agent_repository.go`
  - [ ] Rename interface `PromptTemplateRepository` → `AgentRepository`
  - [ ] Keep existing methods: `Create`, `GetByID`, `GetByProjectAndName`, `ListByProject`, `CountByProject`, `Update`, `Delete`
  - [ ] Add new methods to the interface:
    - `ListGlobal(ctx context.Context, limit, offset int32) ([]*model.PromptTemplate, error)` (use `model.PromptTemplate` unless model rename is in scope)
    - `ListByProjectMerged(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.PromptTemplate, error)`
    - `CountGlobal(ctx context.Context) (int64, error)`
  - [ ] Update the compile-time check `var _ port.AgentRepository = (*AgentRepo)(nil)`

- [ ] [BACK] Task 3: Rename postgres adapter (AC: #1, #2, #3, #4)
  - [ ] Rename `backend/internal/adapter/postgres/prompt_template_repo.go` → `agent_repo.go`
  - [ ] Rename struct `PromptTemplateRepo` → `AgentRepo`
  - [ ] Rename constructor `NewPromptTemplateRepo` → `NewAgentRepo`
  - [ ] Implement new method `ListGlobal`:
    - Add sqlc query `-- name: ListGlobalAgents :many` in `backend/queries/prompt_templates.sql` (or rename file to `agents.sql`):
      ```sql
      SELECT * FROM prompt_templates WHERE project_id IS NULL ORDER BY name LIMIT $1 OFFSET $2;
      ```
    - Run `sqlc generate` to generate the query function
    - Implement `ListGlobal` calling the generated function
  - [ ] Implement new method `ListByProjectMerged`:
    - Add sqlc query `-- name: ListAgentsByProjectMerged :many`:
      ```sql
      SELECT * FROM prompt_templates WHERE project_id IS NULL OR project_id = $1 ORDER BY name LIMIT $2 OFFSET $3;
      ```
    - Run `sqlc generate`
    - Implement `ListByProjectMerged` calling the generated function
  - [ ] Implement `CountGlobal`:
    - Add sqlc query `-- name: CountGlobalAgents :one`:
      ```sql
      SELECT COUNT(*) FROM prompt_templates WHERE project_id IS NULL;
      ```
    - Run `sqlc generate`
    - Implement `CountGlobal`
  - [ ] Keep `toDomainPromptTemplate` mapper (rename to `toDomainAgent` if model is also renamed)

- [ ] [BACK] Task 4: Add new service methods (AC: #3, #4, #5)
  - [ ] Add `ListGlobal(ctx context.Context, page, perPage int) (*AgentListResult, error)` to `AgentService`
    - Calls `repo.ListGlobal(ctx, limit, offset)` and `repo.CountGlobal(ctx)`
  - [ ] Add `ListByProjectMerged(ctx context.Context, projectID uuid.UUID, page, perPage int) (*AgentListResult, error)`
    - Calls `repo.ListByProjectMerged(ctx, projectID, limit, offset)` and sum of CountByProject + CountGlobal
  - [ ] Add scope validation in `Create`:
    - If `params.ProjectID == uuid.Nil` (global): allow creation regardless of scope field
    - If scope field is `"project"` and `params.ProjectID == uuid.Nil`: return `errors.NewValidation("project_id", "project-scoped agents must have a project_id")`
    - If `params.ProjectID` is non-nil: allow creation as project agent

- [ ] [BACK] Task 5: Rename handler file and type (AC: #1, #6)
  - [ ] Rename `backend/internal/api/handler/prompt_template_handler.go` → `agent_handler.go`
  - [ ] Rename `backend/internal/api/handler/prompt_template_handler_test.go` → `agent_handler_test.go`
  - [ ] Rename struct `PromptTemplateHandler` → `AgentHandler`
  - [ ] Rename constructor `NewPromptTemplateHandler` → `NewAgentHandler`
  - [ ] Add handler methods for new routes: `ListGlobalAgents`, `ListProjectMergedAgents`
  - [ ] Register new routes in chi router:
    - `GET /api/v1/agents` → `AgentHandler.ListGlobalAgents`
    - `GET /api/v1/projects/{projectId}/agents` → `AgentHandler.ListProjectMergedAgents`
    - Keep existing `/templates` routes pointing to the same handlers for backward compatibility (AC: #6)
  - [ ] Update route registration in `main.go` or router setup file

- [ ] [BACK] Task 6: Update DI wiring (AC: #7)
  - [ ] In `backend/cmd/api/main.go` or `wire.go`, replace all references to:
    - `PromptTemplateService` → `AgentService`
    - `NewPromptTemplateService` → `NewAgentService`
    - `PromptTemplateRepository` → `AgentRepository`
    - `NewPromptTemplateRepo` → `NewAgentRepo`
    - `PromptTemplateHandler` → `AgentHandler`
    - `NewPromptTemplateHandler` → `NewAgentHandler`
  - [ ] Regenerate wire if using go-wire: `cd backend && wire ./cmd/api/`
  - [ ] Verify `go build ./...` exits 0

- [ ] [BACK] Task 7: Update all tests (AC: #2, #8)
  - [ ] Update service tests: rename test file, update type references, add tests for `ListGlobal`, `ListByProjectMerged`, and scope validation
  - [ ] Update handler tests: rename test file, update type references
  - [ ] Update repo tests if they exist: rename, update references
  - [ ] Run `go test ./... -short` — all must pass

- [ ] [BACK] Task 8: Run lint (AC: #8)
  - [ ] `cd backend && golangci-lint run ./...` — must pass before committing

## Dev Notes

### Dependencies

- **R-1-4 (Agent model + migration) — required:** This story assumes the `agents` (or `prompt_templates` with extended scope column) table/model exists from R-1-4. If R-1-4 renames the model from `PromptTemplate` to `Agent`, the model renames cascade into this story. If R-1-4 only adds DB schema without renaming the Go model, this story only renames the service/repo/handler layer.
- **No other story is blocked by this rename:** All consumers of `PromptTemplateService` are within the backend only (no frontend changes required — the OpenAPI spec still uses `template` or the new `agent` naming from R-1-1).

### Architecture Requirements

- This is a **pure rename + extension** story. No behavioral change to existing CRUD operations.
- The rename follows the Go module path: all files use `snake_case`, all types use `PascalCase`.
- `AgentService` depends on `port.AgentRepository` (interface). The adapter `AgentRepo` implements it. The handler `AgentHandler` depends on `*AgentService` (concrete service — consistent with the existing handler pattern in this codebase).
- New port methods (`ListGlobal`, `ListByProjectMerged`, `CountGlobal`) are added to the `AgentRepository` interface. The `AgentRepo` adapter implements them via new sqlc queries.
- Backward-compatible routes: the existing `/templates` URL prefix must continue to work. This is achieved by registering the same handler on both route prefixes in the chi router — no redirect, no deprecation header at MVP.
- `project_id IS NULL` in SQL means the agent is global; `project_id = $1` means it is project-scoped.

### Technical Specifications

**File renames:**

| Old file | New file |
|---|---|
| `backend/internal/domain/service/prompt_template_service.go` | `backend/internal/domain/service/agent_service.go` |
| `backend/internal/domain/port/prompt_template_repository.go` | `backend/internal/domain/port/agent_repository.go` |
| `backend/internal/adapter/postgres/prompt_template_repo.go` | `backend/internal/adapter/postgres/agent_repo.go` |
| `backend/internal/api/handler/prompt_template_handler.go` | `backend/internal/api/handler/agent_handler.go` |
| `backend/internal/api/handler/prompt_template_handler_test.go` | `backend/internal/api/handler/agent_handler_test.go` |
| `backend/queries/prompt_templates.sql` | `backend/queries/agents.sql` (or keep name if sqlc config references it) |

**Type renames:**

| Old name | New name |
|---|---|
| `PromptTemplateService` | `AgentService` |
| `NewPromptTemplateService` | `NewAgentService` |
| `PromptTemplateRepository` (port) | `AgentRepository` |
| `PromptTemplateRepo` (adapter) | `AgentRepo` |
| `NewPromptTemplateRepo` | `NewAgentRepo` |
| `PromptTemplateHandler` | `AgentHandler` |
| `NewPromptTemplateHandler` | `NewAgentHandler` |
| `CreatePromptTemplateParams` | `CreateAgentParams` |
| `UpdatePromptTemplateParams` | `UpdateAgentParams` |
| `PromptTemplateListResult` | `AgentListResult` |

**New sqlc queries (add to `backend/queries/agents.sql`):**

```sql
-- name: ListGlobalAgents :many
SELECT * FROM prompt_templates
WHERE project_id IS NULL
ORDER BY name
LIMIT $1 OFFSET $2;

-- name: CountGlobalAgents :one
SELECT COUNT(*) FROM prompt_templates
WHERE project_id IS NULL;

-- name: ListAgentsByProjectMerged :many
SELECT * FROM prompt_templates
WHERE project_id IS NULL OR project_id = $1
ORDER BY name
LIMIT $2 OFFSET $3;

-- name: CountAgentsByProjectMerged :one
SELECT COUNT(*) FROM prompt_templates
WHERE project_id IS NULL OR project_id = $1;
```

**New service methods:**

```go
// ListGlobal returns all global agents (project_id = NULL) with pagination.
func (s *AgentService) ListGlobal(ctx context.Context, page, perPage int) (*AgentListResult, error) {
    limit, offset := paginationToLimitOffset(page, perPage)
    agents, err := s.repo.ListGlobal(ctx, limit, offset)
    if err != nil {
        return nil, err
    }
    total, err := s.repo.CountGlobal(ctx)
    if err != nil {
        return nil, err
    }
    return &AgentListResult{Templates: agents, Total: total}, nil
}

// ListByProjectMerged returns global agents merged with project-specific agents.
// Global agents (project_id = NULL) and project agents (project_id = projectID)
// are returned together, ordered by name.
func (s *AgentService) ListByProjectMerged(ctx context.Context, projectID uuid.UUID, page, perPage int) (*AgentListResult, error) {
    limit, offset := paginationToLimitOffset(page, perPage)
    agents, err := s.repo.ListByProjectMerged(ctx, projectID, limit, offset)
    if err != nil {
        return nil, err
    }
    total, err := s.repo.CountByProjectMerged(ctx, projectID)
    if err != nil {
        return nil, err
    }
    return &AgentListResult{Templates: agents, Total: total}, nil
}
```

**New chi routes (add to router setup):**

```go
// Global agents
r.Get("/api/v1/agents", agentHandler.ListGlobalAgents)

// Project-merged agents (new route alias)
r.Get("/api/v1/projects/{projectId}/agents", agentHandler.ListProjectMergedAgents)

// Keep backward-compatible template routes
r.Get("/api/v1/projects/{projectId}/templates", agentHandler.ListPromptTemplates)
```

**Scope validation logic in `AgentService.Create`:**

```go
// if scope field is present and equals "project", project_id is mandatory
if params.Scope == "project" && params.ProjectID == uuid.Nil {
    return nil, errors.NewValidation("project_id", "project-scoped agents must have a project_id")
}
```

Note: `CreateAgentParams.Scope` should be added if the Agent model (from R-1-4) includes a `scope` field. If R-1-4 does not add scope to the model yet, omit this validation and add it in a follow-up.

### Testing Requirements

**Service tests (`backend/internal/domain/service/agent_service_test.go` — renamed from `prompt_template_service_test.go`):**

1. All existing `PromptTemplateService` tests renamed and updated to use new types.
2. **New: TestListGlobal** — mock repo returns 2 global agents, CountGlobal returns 2 → result has 2 agents.
3. **New: TestListByProjectMerged** — mock repo returns 5 agents (2 global + 3 project), CountByProjectMerged returns 5.
4. **New: TestCreate_ScopeValidation** — scope `"project"` with nil project_id → validation error; scope `"global"` with nil project_id → success; valid project_id → success.

**Handler tests (`backend/internal/api/handler/agent_handler_test.go`):**

1. All existing handler tests renamed and passing.
2. **New: TestListProjectMergedAgents** — GET `/api/v1/projects/{id}/agents` returns merged list.
3. **New: TestListGlobalAgents** — GET `/api/v1/agents` returns global-only list.

**Repo tests (if applicable):**

- Rename and verify `ListGlobal` and `ListByProjectMerged` with integration test against testcontainer DB.

Run `golangci-lint run ./...` and `go test ./... -short` before committing. Integration tests: `go test ./... -run Integration` (requires Docker).

### References

- `backend/internal/domain/service/prompt_template_service.go` — source file to rename
- `backend/internal/domain/port/prompt_template_repository.go` — port interface to rename and extend
- `backend/internal/adapter/postgres/prompt_template_repo.go` — adapter to rename and extend
- `backend/internal/api/handler/prompt_template_handler.go` — handler to rename
- `backend/queries/prompt_templates.sql` — sqlc queries to extend (add ListGlobal, ListByProjectMerged, Count variants)
- `backend/sqlc.yaml` — check if query file name needs updating
- `backend/cmd/api/main.go` — DI wiring to update
- Story R-1-4 — Agent model + migration (defines whether `model.PromptTemplate` is also renamed to `model.Agent`)
- `backend/.golangci.yml` — lint config
- `backend/migrations/000026_seed_merge_template.up.sql` — last migration; next migration number is `000027`

## Dev Agent Record

## Change Log

- 2026-02-23: Story created for Wave R implementation
