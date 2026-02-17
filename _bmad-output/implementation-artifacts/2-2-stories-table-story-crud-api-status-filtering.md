# Story 2.2: [BACK] Stories table + Story CRUD API + status filtering

Status: ready-for-dev

## Story

As a user,
I want a stories table and CRUD API to manage stories and filter by status,
so that story data with frontmatter fields can be persisted and tracked.

## Acceptance Criteria (BDD)

**AC1: Migration creates stories table**
- **Given** migration 000006 exists
- **When** migrations are applied
- **Then** a `stories` table is created with: id (UUID PK), project_id (FK projects CASCADE), epic_id (FK epics SET NULL), key (unique per project), title, objective, target_files (JSONB), depends_on (JSONB), scope (VARCHAR), status (VARCHAR default 'backlog'), acceptance_criteria (TEXT), created_at, updated_at

**AC2: sqlc generates Story CRUD functions**
- **Given** sqlc queries are defined in `backend/queries/stories.sql`
- **When** I run `make generate`
- **Then** Go functions for CreateStory, GetStory, GetStoryByKey, ListStoriesByProject, ListStoriesByStatus, ListStoriesByEpic, CountStoriesByProject, UpdateStory, DeleteStory are generated

**AC3: Any authenticated user can list stories for a project with status filtering**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/stories?status=backlog,running
- **Then** I receive HTTP 200 with stories list filtered by status and pagination metadata

**AC4: Any authenticated user can get a story by ID**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/stories/{storyId}
- **Then** I receive HTTP 200 with story details including JSONB fields

**AC5: Any authenticated user can get a story by key**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/stories?key=S-14
- **Then** I receive HTTP 200 with story details

**AC6: Admin can create a story**
- **Given** I am authenticated as admin
- **When** I POST /api/v1/projects/{projectId}/stories with valid payload (including JSONB fields)
- **Then** I receive HTTP 201 with created story

**AC7: Admin can update a story**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/projects/{projectId}/stories/{storyId} with valid payload
- **Then** I receive HTTP 200 with updated story

**AC8: Admin can delete a story**
- **Given** I am authenticated as admin
- **When** I DELETE /api/v1/projects/{projectId}/stories/{storyId}
- **Then** I receive HTTP 204 and story is removed

**AC9: Duplicate key returns HTTP 409**
- **Given** I am authenticated as admin
- **When** I POST a story with a key that already exists in the project
- **Then** I receive HTTP 409 Conflict

**AC10: Non-admin cannot mutate stories**
- **Given** I am authenticated as a non-admin user
- **When** I POST/PUT/DELETE /api/v1/projects/{projectId}/stories
- **Then** I receive HTTP 403

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000006 for stories table (AC: #1)
  - [ ] Create `backend/migrations/000006_create_stories_table.up.sql`
  - [ ] Create `backend/migrations/000006_create_stories_table.down.sql`
  - [ ] Define stories table: id (UUID PK DEFAULT gen_random_uuid()), project_id (UUID FK REFERENCES projects CASCADE), epic_id (UUID NULL FK REFERENCES epics SET NULL), key (VARCHAR(50) NOT NULL), title (VARCHAR(255) NOT NULL), objective (TEXT), target_files (JSONB), depends_on (JSONB), scope (VARCHAR(50)), status (VARCHAR(50) NOT NULL DEFAULT 'backlog'), acceptance_criteria (TEXT), created_at (TIMESTAMPTZ NOT NULL DEFAULT now()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
  - [ ] Add unique constraint on (project_id, key)
  - [ ] Add indexes on project_id, epic_id, status for foreign key lookup and filtering
  - [ ] Down migration drops the stories table

- [ ] [BACK] Task 2: Create sqlc queries for Story CRUD (AC: #2)
  - [ ] Create `backend/queries/stories.sql`
  - [ ] Write `CreateStory` query (INSERT ... RETURNING *)
  - [ ] Write `GetStory` query (SELECT by id)
  - [ ] Write `GetStoryByKey` query (SELECT by project_id + key)
  - [ ] Write `ListStoriesByProject` query (SELECT with project_id filter, LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `ListStoriesByStatus` query (SELECT with project_id + status IN (...) filter, LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `ListStoriesByEpic` query (SELECT with epic_id filter, LIMIT/OFFSET, ORDER BY created_at DESC)
  - [ ] Write `CountStoriesByProject` query (SELECT COUNT(*) with project_id filter for pagination)
  - [ ] Write `CountStoriesByStatus` query (SELECT COUNT(*) with project_id + status IN (...) filter for pagination)
  - [ ] Write `UpdateStory` query (UPDATE ... RETURNING *)
  - [ ] Write `DeleteStory` query (DELETE by id)
  - [ ] Run `make generate` and verify generated code compiles

- [ ] [BACK] Task 3: Create Story domain model and port interface (AC: #3-10)
  - [ ] Create `backend/internal/domain/model/story.go` with Story struct (including JSONB fields as Go types)
  - [ ] Create `backend/internal/domain/port/story_repository.go` with StoryRepository interface
  - [ ] Define methods: Create, GetByID, GetByKey, ListByProject, ListByStatus, ListByEpic, CountByProject, CountByStatus, Update, Delete
  - [ ] Use domain types (uuid.UUID, time.Time, []string for JSONB arrays) not sqlc-generated types

- [ ] [BACK] Task 4: Implement Postgres adapter for StoryRepository (AC: #2)
  - [ ] Create `backend/internal/adapter/postgres/story_repo.go`
  - [ ] Implement StoryRepository interface using sqlc-generated Queries
  - [ ] Map between domain model and sqlc-generated types (handle JSONB marshaling/unmarshaling)
  - [ ] Handle pgx errors (not found -> domain error, unique constraint -> domain error)

- [ ] [BACK] Task 5: Create StoryService in domain layer (AC: #3-10)
  - [ ] Create `backend/internal/domain/service/story_service.go`
  - [ ] Implement Create, GetByID, GetByKey, ListByProject, ListByStatus, Update, Delete
  - [ ] Validate inputs (key required, key format, title required, scope enum, status enum)
  - [ ] Service depends only on StoryRepository port (no adapter import)

- [ ] [BACK] Task 6: Create StoryHandler with RBAC-protected routes + status filtering (AC: #3-10)
  - [ ] Create `backend/internal/api/handler/story_handler.go`
  - [ ] Implement GET /api/v1/projects/{projectId}/stories (any authenticated user) -> HTTP 200 with pagination and status filtering (?status=backlog,running)
  - [ ] Implement GET /api/v1/projects/{projectId}/stories?key={key} (any authenticated user) -> HTTP 200 (single story by key)
  - [ ] Implement GET /api/v1/projects/{projectId}/stories/{storyId} (any authenticated user) -> HTTP 200
  - [ ] Implement POST /api/v1/projects/{projectId}/stories (admin only) -> HTTP 201
  - [ ] Implement PUT /api/v1/projects/{projectId}/stories/{storyId} (admin only) -> HTTP 200
  - [ ] Implement DELETE /api/v1/projects/{projectId}/stories/{storyId} (admin only) -> HTTP 204
  - [ ] Use auth middleware from Story 1-3 for authentication
  - [ ] Use admin check from auth context for RBAC (POST/PUT/DELETE -> 403 if not admin)
  - [ ] Parse pagination query params (page, per_page with defaults)
  - [ ] Parse status query param (comma-separated list, e.g., ?status=backlog,running)
  - [ ] Register routes on chi router under `/api/v1/projects/{projectId}/stories`

- [ ] [BACK] Task 7: Add unit tests for StoryService (AC: #3-10)
  - [ ] Create `backend/internal/domain/service/story_service_test.go`
  - [ ] Test validation: key required, key format, title required, scope enum, status enum
  - [ ] Test Create, GetByID, GetByKey, ListByProject, ListByStatus, Update, Delete
  - [ ] Test error handling (not found, conflict, validation)
  - [ ] Test JSONB field handling (target_files, depends_on)
  - [ ] Use mock repository

- [ ] [BACK] Task 8: Add unit tests for StoryHandler (AC: #3-10)
  - [ ] Create `backend/internal/api/handler/story_handler_test.go`
  - [ ] Test RBAC: admin can mutate, non-admin returns 403
  - [ ] Test status filtering: ?status=backlog,running returns only matching stories
  - [ ] Test key lookup: ?key=S-14 returns single story
  - [ ] Test pagination response format
  - [ ] Test error responses (not found, validation, conflict)
  - [ ] Use mock service

- [ ] [BACK] Task 9: Wire StoryHandler into main.go and verify (AC: #1-10)
  - [ ] Instantiate StoryRepository, StoryService, StoryHandler in main.go (or DI wiring)
  - [ ] Mount story routes on the chi router
  - [ ] Run migration 000006 against dev database
  - [ ] Manual test: admin POST/GET/PUT/DELETE stories with JSONB fields
  - [ ] Manual test: non-admin POST -> 403, GET -> 200
  - [ ] Manual test: status filtering (?status=backlog,running)
  - [ ] Manual test: key lookup (?key=S-14)
  - [ ] Manual test: duplicate key -> 409
  - [ ] Verify pagination response format matches OpenAPI spec

## Dev Notes

This story adds the stories domain following the exact same hexagonal pattern as Story 2-1 (epics): database table, sqlc queries, hexagonal layers (model, port, adapter, service, handler), and RBAC-protected HTTP endpoints nested under projects.

**Key differences from Story 2-1 (epics):**
- Stories have JSONB fields (target_files, depends_on) requiring marshaling/unmarshaling
- Stories have status filtering on list endpoint (?status=backlog,running — comma-separated)
- Stories have GetStoryByKey query (unique per project)
- Stories have epic_id FK (nullable, SET NULL on epic delete)
- Stories have scope field (backend/frontend/shared)
- Unique constraint on (project_id, key) instead of (project_id, name)

### Dependencies

**Story 2-1 (Epics table - wave 5):** epic_id FK references epics table. The epics table must exist for the foreign key constraint. Stories can optionally belong to an epic.

**Story 1-5 (Projects table):** Story endpoints are nested under `/api/v1/projects/{projectId}/stories`. The projects table must exist for the foreign key constraint. Story creation requires a valid project_id.

**Story 1-3 (Users + Auth):** Auth middleware must exist for JWT cookie validation and user context extraction. The handler reads user role from auth context to enforce RBAC.

**Story 1-2 (OpenAPI spec):** Story endpoints and schemas need to be defined in `api/openapi.yaml`. If not present, add them following the same pattern as Epic endpoints.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000006_create_stories_table.up.sql
│   └── 000006_create_stories_table.down.sql
├── queries/
│   └── stories.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── story.go                 # Story struct (domain model)
│   │   ├── port/
│   │   │   └── story_repository.go      # StoryRepository interface
│   │   └── service/
│   │       ├── story_service.go         # StoryService (business logic)
│   │       └── story_service_test.go    # Unit tests
│   ├── adapter/
│   │   └── postgres/
│   │       └── story_repo.go            # StoryRepository impl (uses sqlc)
│   └── api/
│       └── handler/
│           ├── story_handler.go         # HTTP handlers + route registration
│           └── story_handler_test.go    # Unit tests
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

- Migration: `backend/migrations/000006_create_stories_table.{up,down}.sql`
- sqlc queries: `backend/queries/stories.sql`
- Domain model: `backend/internal/domain/model/story.go`
- Port interface: `backend/internal/domain/port/story_repository.go`
- Service: `backend/internal/domain/service/story_service.go`
- Service tests: `backend/internal/domain/service/story_service_test.go`
- Postgres adapter: `backend/internal/adapter/postgres/story_repo.go`
- Handler: `backend/internal/api/handler/story_handler.go`
- Handler tests: `backend/internal/api/handler/story_handler_test.go`

### Technical Specifications

**Migration 000006 SQL:**
```sql
-- 000006_create_stories_table.up.sql
CREATE TABLE stories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    epic_id             UUID NULL REFERENCES epics(id) ON DELETE SET NULL,
    key                 VARCHAR(50) NOT NULL,
    title               VARCHAR(255) NOT NULL,
    objective           TEXT,
    target_files        JSONB,
    depends_on          JSONB,
    scope               VARCHAR(50),
    status              VARCHAR(50) NOT NULL DEFAULT 'backlog',
    acceptance_criteria TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stories_uq_project_key UNIQUE (project_id, key)
);

CREATE INDEX idx_stories_project_id ON stories(project_id);
CREATE INDEX idx_stories_epic_id ON stories(epic_id);
CREATE INDEX idx_stories_status ON stories(status);

-- 000006_create_stories_table.down.sql
DROP TABLE IF EXISTS stories;
```

**sqlc query signatures (`backend/queries/stories.sql`):**
```sql
-- name: CreateStory :one
INSERT INTO stories (project_id, epic_id, key, title, objective, target_files, depends_on, scope, status, acceptance_criteria)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetStory :one
SELECT * FROM stories WHERE id = $1;

-- name: GetStoryByKey :one
SELECT * FROM stories WHERE project_id = $1 AND key = $2;

-- name: ListStoriesByProject :many
SELECT * FROM stories
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListStoriesByStatus :many
SELECT * FROM stories
WHERE project_id = $1 AND status = ANY($2::text[])
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ListStoriesByEpic :many
SELECT * FROM stories
WHERE epic_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountStoriesByProject :one
SELECT COUNT(*) FROM stories WHERE project_id = $1;

-- name: CountStoriesByStatus :one
SELECT COUNT(*) FROM stories WHERE project_id = $1 AND status = ANY($2::text[]);

-- name: UpdateStory :one
UPDATE stories
SET title = COALESCE(sqlc.narg('title'), title),
    objective = COALESCE(sqlc.narg('objective'), objective),
    target_files = COALESCE(sqlc.narg('target_files'), target_files),
    depends_on = COALESCE(sqlc.narg('depends_on'), depends_on),
    scope = COALESCE(sqlc.narg('scope'), scope),
    status = COALESCE(sqlc.narg('status'), status),
    acceptance_criteria = COALESCE(sqlc.narg('acceptance_criteria'), acceptance_criteria),
    epic_id = COALESCE(sqlc.narg('epic_id'), epic_id),
    updated_at = now()
WHERE id = @id
RETURNING *;

-- name: DeleteStory :exec
DELETE FROM stories WHERE id = $1;
```

**Domain model (`backend/internal/domain/model/story.go`):**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type Story struct {
    ID                 uuid.UUID
    ProjectID          uuid.UUID
    EpicID             *uuid.UUID  // nullable FK to epics
    Key                string
    Title              string
    Objective          *string
    TargetFiles        []string    // JSONB -> Go slice
    DependsOn          []string    // JSONB -> Go slice
    Scope              *string     // backend/frontend/shared
    Status             string      // backlog/running/done/failed
    AcceptanceCriteria *string
    CreatedAt          time.Time
    UpdatedAt          time.Time
}
```

**Port interface (`backend/internal/domain/port/story_repository.go`):**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type StoryRepository interface {
    Create(ctx context.Context, story *model.Story) (*model.Story, error)
    GetByID(ctx context.Context, id uuid.UUID) (*model.Story, error)
    GetByKey(ctx context.Context, projectID uuid.UUID, key string) (*model.Story, error)
    ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]*model.Story, error)
    ListByStatus(ctx context.Context, projectID uuid.UUID, statuses []string, limit, offset int32) ([]*model.Story, error)
    ListByEpic(ctx context.Context, epicID uuid.UUID, limit, offset int32) ([]*model.Story, error)
    CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error)
    CountByStatus(ctx context.Context, projectID uuid.UUID, statuses []string) (int64, error)
    Update(ctx context.Context, story *model.Story) (*model.Story, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

**Status filtering query param:**
- Format: `?status=backlog,running` (comma-separated list)
- Handler parses comma-separated string into []string
- Pass to ListByStatus and CountByStatus queries
- Empty or missing status param = list all stories (use ListByProject)

**Key lookup query param:**
- Format: `?key=S-14` (single key lookup)
- Handler detects key param and calls GetByKey instead of ListByProject
- Returns single story object (not array)
- HTTP 404 if not found

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
- GET (list, single, by key): allow any authenticated user

**Error responses (match OpenAPI error envelope):**
```json
{
  "error": {
    "code": "STORY_NOT_FOUND",
    "message": "Story not found"
  }
}
```

**Error codes used:**
- `STORY_NOT_FOUND` — story not found (404)
- `STORY_KEY_CONFLICT` — duplicate key in project (409)
- `VALIDATION_ERROR` — missing/invalid fields (400)
- `FORBIDDEN` — non-admin trying to mutate (403)

**JSONB handling in adapter:**
- `target_files` and `depends_on` are stored as JSONB arrays in Postgres
- Adapter marshals Go []string to JSONB on insert/update
- Adapter unmarshals JSONB to Go []string on select
- Use `encoding/json` or `pgtype.JSONB` for conversion

**Validation rules:**
- key: required, format: `[A-Z0-9]+-\d+` (e.g., S-14, STORY-123)
- title: required, max 255 chars
- scope: optional, enum: backend/frontend/shared
- status: required, default: backlog, enum: backlog/running/done/failed
- epic_id: optional, must reference existing epic if provided
- project_id: required, must reference existing project

### Testing Requirements

**Manual verification checklist:**
1. Run migration: `migrate -path migrations/ -database $DB_URL up`
2. Verify table: `\d stories` shows all columns including JSONB fields with correct types and defaults
3. Run `make generate` -- sqlc generates story query functions
4. `go build ./...` compiles successfully
5. `golangci-lint run ./...` passes with no errors
6. Admin POST `/api/v1/projects/{projectId}/stories` with `{"key": "S-01", "title": "First story", "target_files": ["backend/main.go"], "depends_on": [], "scope": "backend"}` -> 201
7. Admin GET `/api/v1/projects/{projectId}/stories` -> 200 with data[] and pagination
8. Admin GET `/api/v1/projects/{projectId}/stories?status=backlog` -> 200 with filtered stories
9. Admin GET `/api/v1/projects/{projectId}/stories?status=backlog,running` -> 200 with filtered stories
10. Admin GET `/api/v1/projects/{projectId}/stories?key=S-01` -> 200 with single story
11. Admin GET `/api/v1/projects/{projectId}/stories/{storyId}` -> 200 with story including JSONB fields
12. Admin PUT `/api/v1/projects/{projectId}/stories/{storyId}` with `{"title": "Renamed Story", "status": "running"}` -> 200
13. Admin DELETE `/api/v1/projects/{projectId}/stories/{storyId}` -> 204
14. Non-admin POST `/api/v1/projects/{projectId}/stories` -> 403
15. Non-admin GET `/api/v1/projects/{projectId}/stories` -> 200 (read is allowed)
16. Duplicate key POST -> 409 with error envelope
17. Invalid project_id POST -> 400/404 with error envelope
18. Invalid epic_id POST -> 400/404 with error envelope
19. Delete epic -> stories.epic_id SET NULL (verify CASCADE behavior)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.2]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#API Design]
- [Source: Story 2-1 (epics CRUD) — exact same pattern]
- [Source: Story 1-5 (projects CRUD) — hexagonal architecture reference]

## Dev Agent Record

## Change Log
