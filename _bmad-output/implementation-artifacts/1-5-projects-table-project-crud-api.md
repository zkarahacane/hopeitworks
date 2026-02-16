# Story 1.5: Projects table + Project CRUD API

Status: dev-done

## Story

As an admin,
I want a projects table and CRUD API to create and configure projects with Git connections,
so that the platform can orchestrate agents per project.

## Acceptance Criteria (BDD)

**AC1: Migration creates projects table**
- **Given** migration 000002 exists
- **When** migrations are applied
- **Then** a `projects` table is created with: id (UUID PK), name, repo_url, git_provider (default 'github'), git_token_env, agent_runtime (default 'docker'), default_model, max_budget, created_at, updated_at

**AC2: sqlc generates Project CRUD functions**
- **Given** sqlc queries are defined in `backend/queries/projects.sql`
- **When** I run `make generate`
- **Then** Go functions for CreateProject, GetProject, ListProjects, UpdateProject, DeleteProject are generated

**AC3: Admin can create a project**
- **Given** I am authenticated as admin
- **When** I POST /api/v1/projects with valid payload
- **Then** I receive HTTP 201 with created project

**AC4: Admin can list all projects with pagination**
- **Given** I am authenticated as admin
- **When** I GET /api/v1/projects
- **Then** I receive all projects with pagination metadata

**AC5: Regular user can list projects (MVP: all projects)**
- **Given** I am authenticated as a regular user
- **When** I GET /api/v1/projects
- **Then** I receive all projects (MVP simplification; project_users filtering comes in Story 1.6)

**AC6: Non-admin cannot mutate projects**
- **Given** I am authenticated as a non-admin user
- **When** I POST/PUT/DELETE /api/v1/projects
- **Then** I receive HTTP 403

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000002 for projects table (AC: #1)
  - [ ] Create `backend/migrations/000002_create_projects_table.up.sql`
  - [ ] Create `backend/migrations/000002_create_projects_table.down.sql`
  - [ ] Define projects table: id (UUID PK DEFAULT gen_random_uuid()), name (VARCHAR(255) NOT NULL), repo_url (TEXT), git_provider (VARCHAR(50) NOT NULL DEFAULT 'github'), git_token_env (VARCHAR(255)), agent_runtime (VARCHAR(50) NOT NULL DEFAULT 'docker'), default_model (VARCHAR(100)), max_budget (NUMERIC(10,2)), created_at (TIMESTAMPTZ NOT NULL DEFAULT now()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
  - [ ] Add unique constraint on name
  - [ ] Add index on created_at for default sort order
  - [ ] Down migration drops the projects table

- [ ] [BACK] Task 2: Create sqlc queries for Project CRUD (AC: #2)
  - [ ] Create `backend/queries/projects.sql`
  - [ ] Write `CreateProject` query (INSERT ... RETURNING *)
  - [ ] Write `GetProject` query (SELECT by id)
  - [ ] Write `ListProjects` query (SELECT with LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `CountProjects` query (SELECT COUNT(*) for pagination metadata)
  - [ ] Write `UpdateProject` query (UPDATE ... RETURNING *)
  - [ ] Write `DeleteProject` query (DELETE by id)
  - [ ] Run `make generate` and verify generated code compiles

- [ ] [BACK] Task 3: Create Project domain model and port interface (AC: #3, #4, #5, #6)
  - [ ] Create `backend/internal/domain/model/project.go` with Project struct
  - [ ] Create `backend/internal/domain/port/project_repository.go` with ProjectRepository interface
  - [ ] Define methods: Create, GetByID, List, Count, Update, Delete
  - [ ] Use domain types (uuid.UUID, time.Time) not sqlc-generated types

- [ ] [BACK] Task 4: Implement Postgres adapter for ProjectRepository (AC: #2, #3, #4)
  - [ ] Create `backend/internal/adapter/postgres/project_repo.go`
  - [ ] Implement ProjectRepository interface using sqlc-generated Queries
  - [ ] Map between domain model and sqlc-generated types
  - [ ] Handle pgx errors (not found -> domain error, unique constraint -> domain error)

- [ ] [BACK] Task 5: Create ProjectService in domain layer (AC: #3, #4, #5, #6)
  - [ ] Create `backend/internal/domain/service/project_service.go`
  - [ ] Implement Create, GetByID, List (with pagination params), Update, Delete
  - [ ] Validate inputs (name required, name length, budget >= 0)
  - [ ] Service depends only on ProjectRepository port (no adapter import)

- [ ] [BACK] Task 6: Create ProjectHandler with RBAC-protected routes (AC: #3, #4, #5, #6)
  - [ ] Create `backend/internal/api/handler/project_handler.go`
  - [ ] Implement POST /api/v1/projects (admin only) -> HTTP 201
  - [ ] Implement GET /api/v1/projects (any authenticated user) -> HTTP 200 with pagination
  - [ ] Implement GET /api/v1/projects/{id} (any authenticated user) -> HTTP 200
  - [ ] Implement PUT /api/v1/projects/{id} (admin only) -> HTTP 200
  - [ ] Implement DELETE /api/v1/projects/{id} (admin only) -> HTTP 204
  - [ ] Use auth middleware from Story 1-3 for authentication
  - [ ] Use admin check from auth context for RBAC (POST/PUT/DELETE -> 403 if not admin)
  - [ ] Parse pagination query params (page, per_page with defaults)
  - [ ] Register routes on chi router under `/api/v1/projects`

- [ ] [BACK] Task 7: Wire ProjectHandler into main.go and verify (AC: #1-6)
  - [ ] Instantiate ProjectRepository, ProjectService, ProjectHandler in main.go (or DI wiring)
  - [ ] Mount project routes on the chi router
  - [ ] Run migration 000002 against dev database
  - [ ] Manual test: admin POST/GET/PUT/DELETE projects
  - [ ] Manual test: non-admin POST -> 403, GET -> 200
  - [ ] Verify pagination response format matches OpenAPI spec

## Dev Notes

This story adds the projects domain to the backend: database table, sqlc queries, hexagonal layers (model, port, adapter, service, handler), and RBAC-protected HTTP endpoints. It follows the same hexagonal pattern established in Story 1-3 for users/auth.

### Dependencies

**Story 1-2 (OpenAPI spec):** Project endpoints and schemas are already defined in `api/openapi.yaml`. The handler must match the contract:
- `Project` schema: id, name, description, owner_id, created_at, updated_at
- `CreateProjectRequest`: name (required), description
- `UpdateProjectRequest`: name, description
- `ProjectList`: data[] + pagination (total, page, per_page)

**Story 1-3 (Users + Auth):** Auth middleware must exist for JWT cookie validation and user context extraction. The handler reads user role from auth context to enforce RBAC.

**IMPORTANT: OpenAPI vs Epic schema mismatch.** The OpenAPI spec (`api/openapi.yaml`) defines a simplified Project (name, description, owner_id). The epic defines a richer schema (repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget). The **database migration should include ALL epic fields** since they will be needed later. The **API handler should follow the OpenAPI contract** for now (name, description, owner_id). Additional fields can be exposed when the OpenAPI spec is updated. The dev agent should update the OpenAPI spec to include the additional fields from the epic, or note the discrepancy.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000002_create_projects_table.up.sql
│   └── 000002_create_projects_table.down.sql
├── queries/
│   └── projects.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── project.go              # Project struct (domain model)
│   │   ├── port/
│   │   │   └── project_repository.go   # ProjectRepository interface
│   │   └── service/
│   │       └── project_service.go      # ProjectService (business logic)
│   ├── adapter/
│   │   └── postgres/
│   │       └── project_repo.go         # ProjectRepository impl (uses sqlc)
│   └── api/
│       └── handler/
│           └── project_handler.go      # HTTP handlers + route registration
└── cmd/
    └── api/
        └── main.go                     # Updated wiring
```

**Strict boundaries:**
- `domain/model/` and `domain/port/` import NOTHING from adapter/ or api/
- `domain/service/` depends only on `domain/port/` interfaces
- `adapter/postgres/` implements `domain/port/` interfaces, imports sqlc-generated code
- `api/handler/` depends on `domain/service/`, never directly on adapter/

### Technical Specifications

**Migration 000002 SQL:**
```sql
-- 000002_create_projects_table.up.sql
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    repo_url    TEXT,
    git_provider VARCHAR(50) NOT NULL DEFAULT 'github',
    git_token_env VARCHAR(255),
    agent_runtime VARCHAR(50) NOT NULL DEFAULT 'docker',
    default_model VARCHAR(100),
    max_budget  NUMERIC(10,2),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_created_at ON projects(created_at DESC);

-- 000002_create_projects_table.down.sql
DROP TABLE IF EXISTS projects;
```

**sqlc query signatures (`backend/queries/projects.sql`):**
```sql
-- name: CreateProject :one
INSERT INTO projects (name, repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountProjects :one
SELECT COUNT(*) FROM projects;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE(sqlc.narg('name'), name),
    repo_url = COALESCE(sqlc.narg('repo_url'), repo_url),
    git_provider = COALESCE(sqlc.narg('git_provider'), git_provider),
    git_token_env = COALESCE(sqlc.narg('git_token_env'), git_token_env),
    agent_runtime = COALESCE(sqlc.narg('agent_runtime'), agent_runtime),
    default_model = COALESCE(sqlc.narg('default_model'), default_model),
    max_budget = COALESCE(sqlc.narg('max_budget'), max_budget),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
```

**Domain model (`backend/internal/domain/model/project.go`):**
```go
type Project struct {
    ID           uuid.UUID
    Name         string
    RepoURL      *string
    GitProvider  string
    GitTokenEnv  *string
    AgentRuntime string
    DefaultModel *string
    MaxBudget    *float64
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Port interface (`backend/internal/domain/port/project_repository.go`):**
```go
type ProjectRepository interface {
    Create(ctx context.Context, project *model.Project) (*model.Project, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error)
    List(ctx context.Context, limit, offset int32) ([]*model.Project, error)
    Count(ctx context.Context) (int64, error)
    Update(ctx context.Context, project *model.Project) (*model.Project, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

**Pagination query params:**
- `page` (default: 1, min: 1)
- `per_page` (default: 20, min: 1, max: 100)
- Offset calculated as: `(page - 1) * per_page`

**Pagination response format (matches OpenAPI):**
```json
{
  "data": [...],
  "pagination": {
    "total": 42,
    "page": 1,
    "per_page": 20
  }
}
```

**RBAC logic:**
- Extract user from auth context (set by Story 1-3 auth middleware)
- POST, PUT, DELETE: check `user.Role == "admin"`, return 403 if not
- GET (list, single): allow any authenticated user

**Error responses (match OpenAPI error envelope):**
```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "Admin access required"
  }
}
```

### Testing Requirements

**Manual verification checklist:**
1. Run migration: `migrate -path migrations/ -database $DB_URL up`
2. Verify table: `\d projects` shows all columns with correct types and defaults
3. Run `make generate` -- sqlc generates project query functions
4. `go build ./...` compiles successfully
5. Admin POST `/api/v1/projects` with `{"name": "test-project"}` -> 201
6. Admin GET `/api/v1/projects` -> 200 with data[] and pagination
7. Admin GET `/api/v1/projects/{id}` -> 200 with project
8. Admin PUT `/api/v1/projects/{id}` with `{"name": "renamed"}` -> 200
9. Admin DELETE `/api/v1/projects/{id}` -> 204
10. Non-admin POST `/api/v1/projects` -> 403
11. Non-admin GET `/api/v1/projects` -> 200 (read is allowed)
12. Duplicate name POST -> 400/409 with error envelope

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.5]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#API Design]
- [Source: api/openapi.yaml#/projects endpoints and Project schema]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

No debug issues encountered. Build and all tests pass cleanly.

### Completion Notes List

- **OpenAPI vs Epic schema reconciliation:** The database migration includes ALL fields from the epic (repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget) plus the OpenAPI fields (name, description, owner_id). The API handler exposes only the OpenAPI contract fields (id, name, description, owner_id, created_at, updated_at). Additional fields can be exposed when the OpenAPI spec is updated.
- **Auth middleware placeholder:** Created a minimal auth middleware with context helpers (SetUserContext, UserIDFromContext, RoleFromContext, IsAdmin). Full JWT validation will be implemented in Story 1-3. The RBAC checks in the handler are fully functional via context inspection.
- **Unimplemented server pattern:** Created a `Server` struct that embeds the generated `Unimplemented` type and delegates project endpoints to `ProjectHandler`. Auth/user endpoints return 501 Not Implemented until Story 1-3.
- **pkg/errors package:** Created the `DomainError` type with category-based HTTP status mapping as specified in the architecture docs.
- **Unit tests:** Added comprehensive unit tests for both `ProjectService` and `ProjectHandler` covering validation, RBAC, CRUD operations, and error handling.
- **main.go updated:** Full server wiring with pgxpool, graceful shutdown, health check endpoint, and chi middleware stack.

### File List

**Created:**
- `backend/migrations/000002_create_projects_table.up.sql`
- `backend/migrations/000002_create_projects_table.down.sql`
- `backend/queries/projects.sql`
- `backend/internal/adapter/postgres/db.go` (generated by sqlc)
- `backend/internal/adapter/postgres/models.go` (generated by sqlc)
- `backend/internal/adapter/postgres/projects.sql.go` (generated by sqlc)
- `backend/internal/adapter/postgres/project_repo.go`
- `backend/internal/api/handler/gen_server.go` (generated by oapi-codegen)
- `backend/internal/api/handler/project_handler.go`
- `backend/internal/api/handler/project_handler_test.go`
- `backend/internal/api/handler/helpers.go`
- `backend/internal/api/handler/server.go`
- `backend/internal/api/middleware/auth.go`
- `backend/internal/domain/model/project.go`
- `backend/internal/domain/port/project_repository.go`
- `backend/internal/domain/service/project_service.go`
- `backend/internal/domain/service/project_service_test.go`
- `backend/pkg/errors/errors.go`

**Modified:**
- `backend/cmd/api/main.go`

## Change Log

- 2026-02-16: Initial implementation by Claude Opus 4.6
