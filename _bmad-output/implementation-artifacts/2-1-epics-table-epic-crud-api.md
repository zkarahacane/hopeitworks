# Story 2.1: [BACK] Epics table + Epic CRUD API

Status: ready-for-dev

## Story

As a user,
I want an epics table and CRUD API to manage epics within a project,
so that I can organize stories by feature area.

## Acceptance Criteria (BDD)

**AC1: Migration creates epics table**
- **Given** migration 000005 exists
- **When** migrations are applied
- **Then** an `epics` table is created with: id (UUID PK), project_id (FK projects CASCADE), name, description, status (VARCHAR default 'backlog'), created_at, updated_at

**AC2: sqlc generates Epic CRUD functions**
- **Given** sqlc queries are defined in `backend/queries/epics.sql`
- **When** I run `make generate`
- **Then** Go functions for CreateEpic, GetEpic, ListEpicsByProject, CountEpicsByProject, UpdateEpic, DeleteEpic are generated

**AC3: Any authenticated user can list epics for a project**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/epics
- **Then** I receive HTTP 200 with epics list and pagination metadata

**AC4: Any authenticated user can get a single epic**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/epics/{epicId}
- **Then** I receive HTTP 200 with epic details

**AC5: Admin can create an epic**
- **Given** I am authenticated as admin
- **When** I POST /api/v1/projects/{projectId}/epics with valid payload
- **Then** I receive HTTP 201 with created epic

**AC6: Admin can update an epic**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/projects/{projectId}/epics/{epicId} with valid payload
- **Then** I receive HTTP 200 with updated epic

**AC7: Admin can delete an epic**
- **Given** I am authenticated as admin
- **When** I DELETE /api/v1/projects/{projectId}/epics/{epicId}
- **Then** I receive HTTP 204 and epic is removed

**AC8: Non-admin cannot mutate epics**
- **Given** I am authenticated as a non-admin user
- **When** I POST/PUT/DELETE /api/v1/projects/{projectId}/epics
- **Then** I receive HTTP 403

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000005 for epics table (AC: #1)
  - [ ] Create `backend/migrations/000005_create_epics_table.up.sql`
  - [ ] Create `backend/migrations/000005_create_epics_table.down.sql`
  - [ ] Define epics table: id (UUID PK DEFAULT gen_random_uuid()), project_id (UUID FK REFERENCES projects CASCADE), name (VARCHAR(255) NOT NULL), description (TEXT), status (VARCHAR(50) NOT NULL DEFAULT 'backlog'), created_at (TIMESTAMPTZ NOT NULL DEFAULT now()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
  - [ ] Add unique constraint on (project_id, name)
  - [ ] Add index on project_id for foreign key lookup
  - [ ] Down migration drops the epics table

- [ ] [BACK] Task 2: Create sqlc queries for Epic CRUD (AC: #2)
  - [ ] Create `backend/queries/epics.sql`
  - [ ] Write `CreateEpic` query (INSERT ... RETURNING *)
  - [ ] Write `GetEpic` query (SELECT by id)
  - [ ] Write `ListEpicsByProject` query (SELECT with project_id filter, LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `CountEpicsByProject` query (SELECT COUNT(*) with project_id filter for pagination)
  - [ ] Write `UpdateEpic` query (UPDATE ... RETURNING *)
  - [ ] Write `DeleteEpic` query (DELETE by id)
  - [ ] Run `make generate` and verify generated code compiles

- [ ] [BACK] Task 3: Create Epic domain model and port interface (AC: #3-8)
  - [ ] Create `backend/internal/domain/model/epic.go` with Epic struct
  - [ ] Create `backend/internal/domain/port/epic_repository.go` with EpicRepository interface
  - [ ] Define methods: Create, GetByID, ListByProject, CountByProject, Update, Delete
  - [ ] Use domain types (uuid.UUID, time.Time) not sqlc-generated types

- [ ] [BACK] Task 4: Implement Postgres adapter for EpicRepository (AC: #2)
  - [ ] Create `backend/internal/adapter/postgres/epic_repo.go`
  - [ ] Implement EpicRepository interface using sqlc-generated Queries
  - [ ] Map between domain model and sqlc-generated types
  - [ ] Handle pgx errors (not found -> domain error, unique constraint -> domain error)

- [ ] [BACK] Task 5: Create EpicService in domain layer (AC: #3-8)
  - [ ] Create `backend/internal/domain/service/epic_service.go`
  - [ ] Implement Create, GetByID, ListByProject (with pagination params), Update, Delete
  - [ ] Validate inputs (name required, name length, project_id required)
  - [ ] Service depends only on EpicRepository port (no adapter import)

- [ ] [BACK] Task 6: Create EpicHandler with RBAC-protected routes (AC: #3-8)
  - [ ] Create `backend/internal/api/handler/epic_handler.go`
  - [ ] Implement GET /api/v1/projects/{projectId}/epics (any authenticated user) -> HTTP 200 with pagination
  - [ ] Implement GET /api/v1/projects/{projectId}/epics/{epicId} (any authenticated user) -> HTTP 200
  - [ ] Implement POST /api/v1/projects/{projectId}/epics (admin only) -> HTTP 201
  - [ ] Implement PUT /api/v1/projects/{projectId}/epics/{epicId} (admin only) -> HTTP 200
  - [ ] Implement DELETE /api/v1/projects/{projectId}/epics/{epicId} (admin only) -> HTTP 204
  - [ ] Use auth middleware from Story 1-3 for authentication
  - [ ] Use admin check from auth context for RBAC (POST/PUT/DELETE -> 403 if not admin)
  - [ ] Parse pagination query params (page, per_page with defaults)
  - [ ] Register routes on chi router under `/api/v1/projects/{projectId}/epics`

- [ ] [BACK] Task 7: Add unit tests for EpicService (AC: #3-8)
  - [ ] Create `backend/internal/domain/service/epic_service_test.go`
  - [ ] Test validation: name required, name max length
  - [ ] Test Create, GetByID, ListByProject, Update, Delete
  - [ ] Test error handling (not found, conflict, validation)
  - [ ] Use mock repository

- [ ] [BACK] Task 8: Add unit tests for EpicHandler (AC: #3-8)
  - [ ] Create `backend/internal/api/handler/epic_handler_test.go`
  - [ ] Test RBAC: admin can mutate, non-admin returns 403
  - [ ] Test pagination response format
  - [ ] Test error responses (not found, validation, conflict)
  - [ ] Use mock service

- [ ] [BACK] Task 9: Wire EpicHandler into main.go and verify (AC: #1-8)
  - [ ] Instantiate EpicRepository, EpicService, EpicHandler in main.go (or DI wiring)
  - [ ] Mount epic routes on the chi router
  - [ ] Run migration 000005 against dev database
  - [ ] Manual test: admin POST/GET/PUT/DELETE epics
  - [ ] Manual test: non-admin POST -> 403, GET -> 200
  - [ ] Verify pagination response format matches OpenAPI spec

## Dev Notes

This story adds the epics domain following the exact same hexagonal pattern as Story 1-5 (projects): database table, sqlc queries, hexagonal layers (model, port, adapter, service, handler), and RBAC-protected HTTP endpoints nested under projects.

### Dependencies

**Story 1-5 (Projects table):** Epic endpoints are nested under `/api/v1/projects/{projectId}/epics`. The projects table must exist for the foreign key constraint. Epic creation requires a valid project_id.

**Story 1-3 (Users + Auth):** Auth middleware must exist for JWT cookie validation and user context extraction. The handler reads user role from auth context to enforce RBAC.

**Story 1-2 (OpenAPI spec):** Epic endpoints and schemas need to be defined in `api/openapi.yaml`. If not present, add them following the same pattern as Project endpoints.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000005_create_epics_table.up.sql
│   └── 000005_create_epics_table.down.sql
├── queries/
│   └── epics.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── epic.go                 # Epic struct (domain model)
│   │   ├── port/
│   │   │   └── epic_repository.go      # EpicRepository interface
│   │   └── service/
│   │       ├── epic_service.go         # EpicService (business logic)
│   │       └── epic_service_test.go    # Unit tests
│   ├── adapter/
│   │   └── postgres/
│   │       └── epic_repo.go            # EpicRepository impl (uses sqlc)
│   └── api/
│       └── handler/
│           ├── epic_handler.go         # HTTP handlers + route registration
│           └── epic_handler_test.go    # Unit tests
└── cmd/
    └── api/
        └── main.go                     # Updated wiring
```

**Strict boundaries:**
- `domain/model/` and `domain/port/` import NOTHING from adapter/ or api/
- `domain/service/` depends only on `domain/port/` interfaces
- `adapter/postgres/` implements `domain/port/` interfaces, imports sqlc-generated code
- `api/handler/` depends on `domain/service/`, never directly on adapter/

### File Paths (exact)

- Migration: `backend/migrations/000005_create_epics_table.{up,down}.sql`
- sqlc queries: `backend/queries/epics.sql`
- Domain model: `backend/internal/domain/model/epic.go`
- Port interface: `backend/internal/domain/port/epic_repository.go`
- Service: `backend/internal/domain/service/epic_service.go`
- Service tests: `backend/internal/domain/service/epic_service_test.go`
- Postgres adapter: `backend/internal/adapter/postgres/epic_repo.go`
- Handler: `backend/internal/api/handler/epic_handler.go`
- Handler tests: `backend/internal/api/handler/epic_handler_test.go`

### Technical Specifications

**Migration 000005 SQL:**
```sql
-- 000005_create_epics_table.up.sql
CREATE TABLE epics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      VARCHAR(50) NOT NULL DEFAULT 'backlog',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT epics_uq_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_epics_project_id ON epics(project_id);

-- 000005_create_epics_table.down.sql
DROP TABLE IF EXISTS epics;
```

**sqlc query signatures (`backend/queries/epics.sql`):**
```sql
-- name: CreateEpic :one
INSERT INTO epics (project_id, name, description, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetEpic :one
SELECT * FROM epics WHERE id = $1;

-- name: ListEpicsByProject :many
SELECT * FROM epics
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountEpicsByProject :one
SELECT COUNT(*) FROM epics WHERE project_id = $1;

-- name: UpdateEpic :one
UPDATE epics
SET name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteEpic :exec
DELETE FROM epics WHERE id = $1;
```

**Domain model (`backend/internal/domain/model/epic.go`):**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Epic struct {
    ID          uuid.UUID
    ProjectID   uuid.UUID
    Name        string
    Description *string
    Status      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**Port interface (`backend/internal/domain/port/epic_repository.go`):**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type EpicRepository interface {
    Create(ctx context.Context, epic *model.Epic) (*model.Epic, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Epic, error)
    ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Epic, error)
    CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
    Update(ctx context.Context, epic *model.Epic) (*model.Epic, error)
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
    "code": "EPIC_NOT_FOUND",
    "message": "Epic not found"
  }
}
```

**Error codes used:**
- `EPIC_NOT_FOUND` — epic not found (404)
- `EPIC_NAME_CONFLICT` — duplicate name in project (409)
- `VALIDATION_ERROR` — missing/invalid fields (400)
- `FORBIDDEN` — non-admin trying to mutate (403)

### Testing Requirements

**Manual verification checklist:**
1. Run migration: `migrate -path migrations/ -database $DB_URL up`
2. Verify table: `\d epics` shows all columns with correct types and defaults
3. Run `make generate` -- sqlc generates epic query functions
4. `go build ./...` compiles successfully
5. `golangci-lint run ./...` passes with no errors
6. Admin POST `/api/v1/projects/{projectId}/epics` with `{"name": "Epic 1", "description": "First epic"}` -> 201
7. Admin GET `/api/v1/projects/{projectId}/epics` -> 200 with data[] and pagination
8. Admin GET `/api/v1/projects/{projectId}/epics/{epicId}` -> 200 with epic
9. Admin PUT `/api/v1/projects/{projectId}/epics/{epicId}` with `{"name": "Renamed Epic"}` -> 200
10. Admin DELETE `/api/v1/projects/{projectId}/epics/{epicId}` -> 204
11. Non-admin POST `/api/v1/projects/{projectId}/epics` -> 403
12. Non-admin GET `/api/v1/projects/{projectId}/epics` -> 200 (read is allowed)
13. Duplicate name POST -> 409 with error envelope
14. Invalid project_id POST -> 400/404 with error envelope

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#API Design]
- [Source: Story 1-5 (projects CRUD) — exact same pattern]

## Dev Agent Record

## Change Log
