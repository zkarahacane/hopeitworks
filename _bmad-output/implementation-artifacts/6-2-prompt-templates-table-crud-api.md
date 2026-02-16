# Story 6.2: [BACK] Prompt templates table + CRUD API

Status: ready-for-dev

## Story

As a backend developer,
I want a prompt_templates table with CRUD endpoints,
so that users can manage agent prompts per project via API.

## Acceptance Criteria (BDD)

**AC1: Migration creates prompt_templates table**
- **Given** migration 000010 exists
- **When** migrations are applied
- **Then** a `prompt_templates` table is created with: id (UUID PK), project_id (FK projects CASCADE), name (VARCHAR), template_content (TEXT), type (VARCHAR: implement/retry/review/merge/custom), created_at, updated_at

**AC2: sqlc generates PromptTemplate CRUD functions**
- **Given** sqlc queries are defined in `backend/queries/prompt_templates.sql`
- **When** I run `make generate`
- **Then** Go functions for CreatePromptTemplate, GetPromptTemplate, ListPromptTemplatesByProject, UpdatePromptTemplate, DeletePromptTemplate are generated

**AC3: Any authenticated user can list prompt templates for a project**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/templates
- **Then** I receive HTTP 200 with templates list and pagination metadata

**AC4: Any authenticated user can get a single prompt template**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/templates/{templateId}
- **Then** I receive HTTP 200 with template details

**AC5: Admin can create a prompt template**
- **Given** I am authenticated as admin
- **When** I POST /api/v1/projects/{projectId}/templates with valid payload
- **Then** I receive HTTP 201 with created template

**AC6: Admin can update a prompt template**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/projects/{projectId}/templates/{templateId} with valid payload
- **Then** I receive HTTP 200 with updated template

**AC7: Admin can delete a prompt template**
- **Given** I am authenticated as admin
- **When** I DELETE /api/v1/projects/{projectId}/templates/{templateId}
- **Then** I receive HTTP 204 and template is removed

**AC8: Non-admin cannot mutate prompt templates**
- **Given** I am authenticated as a non-admin user
- **When** I POST/PUT/DELETE /api/v1/projects/{projectId}/templates
- **Then** I receive HTTP 403

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000010 for prompt_templates table (AC: #1)
  - [ ] Create `backend/migrations/000010_create_prompt_templates_table.up.sql`
  - [ ] Create `backend/migrations/000010_create_prompt_templates_table.down.sql`
  - [ ] Define prompt_templates table: id (UUID PK DEFAULT gen_random_uuid()), project_id (UUID NOT NULL REFERENCES projects CASCADE), name (VARCHAR(255) NOT NULL), template_content (TEXT NOT NULL), type (VARCHAR(50) NOT NULL CHECK IN ('implement','retry','review','merge','custom')), created_at (TIMESTAMPTZ NOT NULL DEFAULT now()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
  - [ ] Add unique constraint on (project_id, name)
  - [ ] Add index on project_id for foreign key lookup
  - [ ] Down migration drops the prompt_templates table

- [ ] [BACK] Task 2: Create sqlc queries for PromptTemplate CRUD (AC: #2)
  - [ ] Create `backend/queries/prompt_templates.sql`
  - [ ] Write `CreatePromptTemplate` query (INSERT ... RETURNING *)
  - [ ] Write `GetPromptTemplate` query (SELECT by id)
  - [ ] Write `ListPromptTemplatesByProject` query (SELECT with project_id filter, LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `CountPromptTemplatesByProject` query (SELECT COUNT(*) with project_id filter for pagination)
  - [ ] Write `UpdatePromptTemplate` query (UPDATE ... RETURNING *)
  - [ ] Write `DeletePromptTemplate` query (DELETE by id)
  - [ ] Run `make generate` and verify generated code compiles

- [ ] [BACK] Task 3: Create PromptTemplate domain model and port interface (AC: #3-8)
  - [ ] Create `backend/internal/domain/model/prompt_template.go` with PromptTemplate struct and TemplateType constants
  - [ ] Create `backend/internal/domain/port/prompt_template_repository.go` with PromptTemplateRepository interface
  - [ ] Define methods: Create, GetByID, ListByProject, CountByProject, Update, Delete
  - [ ] Use domain types (uuid.UUID, time.Time) not sqlc-generated types

- [ ] [BACK] Task 4: Implement Postgres adapter for PromptTemplateRepository (AC: #2)
  - [ ] Create `backend/internal/adapter/postgres/prompt_template_repo.go`
  - [ ] Implement PromptTemplateRepository interface using sqlc-generated Queries
  - [ ] Map between domain model and sqlc-generated types
  - [ ] Handle pgx errors (not found -> domain error, unique constraint -> domain error)

- [ ] [BACK] Task 5: Create PromptTemplateService in domain layer (AC: #3-8)
  - [ ] Create `backend/internal/domain/service/prompt_template_service.go`
  - [ ] Implement Create, GetByID, ListByProject (with pagination params), Update, Delete
  - [ ] Validate inputs (name required, name length, template_content required, type valid)
  - [ ] Service depends only on PromptTemplateRepository port (no adapter import)

- [ ] [BACK] Task 6: Create PromptTemplateHandler with RBAC-protected routes (AC: #3-8)
  - [ ] Create `backend/internal/api/handler/prompt_template_handler.go`
  - [ ] Implement GET /api/v1/projects/{projectId}/templates (any authenticated user) -> HTTP 200 with pagination
  - [ ] Implement GET /api/v1/projects/{projectId}/templates/{templateId} (any authenticated user) -> HTTP 200
  - [ ] Implement POST /api/v1/projects/{projectId}/templates (admin only) -> HTTP 201
  - [ ] Implement PUT /api/v1/projects/{projectId}/templates/{templateId} (admin only) -> HTTP 200
  - [ ] Implement DELETE /api/v1/projects/{projectId}/templates/{templateId} (admin only) -> HTTP 204
  - [ ] Use auth middleware from Story 1-3 for authentication
  - [ ] Use admin check from auth context for RBAC (POST/PUT/DELETE -> 403 if not admin)
  - [ ] Parse pagination query params (page, per_page with defaults)
  - [ ] Register routes on chi router under `/api/v1/projects/{projectId}/templates`

- [ ] [BACK] Task 7: Add unit tests for PromptTemplateService (AC: #3-8)
  - [ ] Create `backend/internal/domain/service/prompt_template_service_test.go`
  - [ ] Test validation: name required, template_content required, type valid
  - [ ] Test Create, GetByID, ListByProject, Update, Delete
  - [ ] Test error handling (not found, conflict, validation)
  - [ ] Use mock repository

- [ ] [BACK] Task 8: Add unit tests for PromptTemplateHandler (AC: #3-8)
  - [ ] Create `backend/internal/api/handler/prompt_template_handler_test.go`
  - [ ] Test RBAC: admin can mutate, non-admin returns 403
  - [ ] Test pagination response format
  - [ ] Test error responses (not found, validation, conflict)
  - [ ] Use mock service

- [ ] [BACK] Task 9: Wire PromptTemplateHandler into main.go and verify (AC: #1-8)
  - [ ] Instantiate PromptTemplateRepository, PromptTemplateService, PromptTemplateHandler in main.go (or DI wiring)
  - [ ] Mount prompt template routes on the chi router
  - [ ] Run migration 000010 against dev database
  - [ ] Manual test: admin POST/GET/PUT/DELETE prompt templates
  - [ ] Manual test: non-admin POST -> 403, GET -> 200
  - [ ] Verify pagination response format matches OpenAPI spec

## Dev Notes

This story upgrades prompt storage from file-based (agent/prompts/*.hbs) to DB-backed. It follows the exact same hexagonal pattern as Story 1-5 (projects) and Story 2-1 (epics): database table, sqlc queries, hexagonal layers (model, port, adapter, service, handler), and RBAC-protected HTTP endpoints nested under projects.

### Dependencies

**Story 1-5 (Projects table):** PromptTemplate endpoints are nested under `/api/v1/projects/{projectId}/templates`. The projects table must exist for the foreign key constraint. Template creation requires a valid project_id.

**Story 1-3 (Users + Auth):** Auth middleware must exist for JWT cookie validation and user context extraction. The handler reads user role from auth context to enforce RBAC.

**Story 1-2 (OpenAPI spec):** PromptTemplate endpoints and schemas need to be defined in `api/openapi.yaml`. If not present, add them following the same pattern as Project and Epic endpoints.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000010_create_prompt_templates_table.up.sql
│   └── 000010_create_prompt_templates_table.down.sql
├── queries/
│   └── prompt_templates.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── prompt_template.go              # PromptTemplate struct (domain model)
│   │   ├── port/
│   │   │   └── prompt_template_repository.go   # PromptTemplateRepository interface
│   │   └── service/
│   │       ├── prompt_template_service.go      # PromptTemplateService (business logic)
│   │       └── prompt_template_service_test.go # Unit tests
│   ├── adapter/
│   │   └── postgres/
│   │       └── prompt_template_repo.go         # PromptTemplateRepository impl (uses sqlc)
│   └── api/
│       └── handler/
│           ├── prompt_template_handler.go      # HTTP handlers + route registration
│           └── prompt_template_handler_test.go # Unit tests
└── cmd/
    └── api/
        └── main.go                             # Updated wiring
```

**Strict boundaries:**
- `domain/model/` and `domain/port/` import NOTHING from adapter/ or api/
- `domain/service/` depends only on `domain/port/` interfaces
- `adapter/postgres/` implements `domain/port/` interfaces, imports sqlc-generated code
- `api/handler/` depends on `domain/service/`, never directly on adapter/

### File Paths (exact)

- Migration: `backend/migrations/000010_create_prompt_templates_table.{up,down}.sql`
- sqlc queries: `backend/queries/prompt_templates.sql`
- Domain model: `backend/internal/domain/model/prompt_template.go`
- Port interface: `backend/internal/domain/port/prompt_template_repository.go`
- Service: `backend/internal/domain/service/prompt_template_service.go`
- Service tests: `backend/internal/domain/service/prompt_template_service_test.go`
- Postgres adapter: `backend/internal/adapter/postgres/prompt_template_repo.go`
- Handler: `backend/internal/api/handler/prompt_template_handler.go`
- Handler tests: `backend/internal/api/handler/prompt_template_handler_test.go`

### Technical Specifications

**Migration 000010 SQL:**
```sql
-- 000010_create_prompt_templates_table.up.sql
CREATE TABLE prompt_templates (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    template_content TEXT NOT NULL,
    type             VARCHAR(50) NOT NULL CHECK (type IN ('implement', 'retry', 'review', 'merge', 'custom')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT prompt_templates_uq_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_prompt_templates_project_id ON prompt_templates(project_id);

-- 000010_create_prompt_templates_table.down.sql
DROP TABLE IF EXISTS prompt_templates;
```

**sqlc query signatures (`backend/queries/prompt_templates.sql`):**
```sql
-- name: CreatePromptTemplate :one
INSERT INTO prompt_templates (project_id, name, template_content, type)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPromptTemplate :one
SELECT * FROM prompt_templates WHERE id = $1;

-- name: ListPromptTemplatesByProject :many
SELECT * FROM prompt_templates
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPromptTemplatesByProject :one
SELECT COUNT(*) FROM prompt_templates WHERE project_id = $1;

-- name: UpdatePromptTemplate :one
UPDATE prompt_templates
SET name = COALESCE(sqlc.narg('name'), name),
    template_content = COALESCE(sqlc.narg('template_content'), template_content),
    type = COALESCE(sqlc.narg('type'), type),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeletePromptTemplate :exec
DELETE FROM prompt_templates WHERE id = $1;
```

**Domain model (`backend/internal/domain/model/prompt_template.go`):**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type TemplateType string

const (
    TemplateTypeImplement TemplateType = "implement"
    TemplateTypeRetry     TemplateType = "retry"
    TemplateTypeReview    TemplateType = "review"
    TemplateTypeMerge     TemplateType = "merge"
    TemplateTypeCustom    TemplateType = "custom"
)

type PromptTemplate struct {
    ID              uuid.UUID
    ProjectID       uuid.UUID
    Name            string
    TemplateContent string
    Type            TemplateType
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**Port interface (`backend/internal/domain/port/prompt_template_repository.go`):**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type PromptTemplateRepository interface {
    Create(ctx context.Context, template *model.PromptTemplate) (*model.PromptTemplate, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.PromptTemplate, error)
    ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.PromptTemplate, error)
    CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
    Update(ctx context.Context, template *model.PromptTemplate) (*model.PromptTemplate, error)
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
    "code": "TEMPLATE_NOT_FOUND",
    "message": "Prompt template not found"
  }
}
```

**Error codes used:**
- `TEMPLATE_NOT_FOUND` — template not found (404)
- `TEMPLATE_NAME_CONFLICT` — duplicate name in project (409)
- `VALIDATION_ERROR` — missing/invalid fields (400)
- `FORBIDDEN` — non-admin trying to mutate (403)

### Testing Requirements

**Manual verification checklist:**
1. Run migration: `migrate -path migrations/ -database $DB_URL up`
2. Verify table: `\d prompt_templates` shows all columns with correct types and defaults
3. Run `make generate` -- sqlc generates prompt template query functions
4. `go build ./...` compiles successfully
5. `golangci-lint run ./...` passes with no errors
6. Admin POST `/api/v1/projects/{projectId}/templates` with valid payload -> 201
7. Admin GET `/api/v1/projects/{projectId}/templates` -> 200 with data[] and pagination
8. Admin GET `/api/v1/projects/{projectId}/templates/{templateId}` -> 200 with template
9. Admin PUT `/api/v1/projects/{projectId}/templates/{templateId}` with valid payload -> 200
10. Admin DELETE `/api/v1/projects/{projectId}/templates/{templateId}` -> 204
11. Non-admin POST `/api/v1/projects/{projectId}/templates` -> 403
12. Non-admin GET `/api/v1/projects/{projectId}/templates` -> 200 (read is allowed)
13. Duplicate name POST -> 409 with error envelope
14. Invalid type POST -> 400 with validation error
15. Invalid project_id POST -> 400/404 with error envelope

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.2]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#API Design]
- [Source: Story 1-5 (projects CRUD) — exact same pattern]
- [Source: Story 2-1 (epics CRUD) — exact same pattern]

## Dev Agent Record

## Change Log
