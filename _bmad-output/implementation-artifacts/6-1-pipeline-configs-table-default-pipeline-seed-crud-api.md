# Story 6.1: [BACK] Pipeline configs table + default pipeline seed + CRUD API

Status: ready-for-dev

## Story

As an admin,
I want a pipeline_configs table with default seeding and CRUD API,
so that I can customize execution steps per project.

## Acceptance Criteria (BDD)

**AC1: Migration creates pipeline_configs table**
- **Given** migration 000009 exists
- **When** migrations are applied
- **Then** a `pipeline_configs` table is created with: id (UUID PK), project_id (FK projects CASCADE), config_yaml (TEXT NOT NULL), version (INT default 1), created_at, updated_at

**AC2: Default pipeline config is seeded on project creation**
- **Given** a new project is created
- **When** the project creation triggers pipeline config initialization
- **Then** a default pipeline config is inserted with steps: agent_run, hitl_gate, git_create_pr, git_merge

**AC3: sqlc generates PipelineConfig queries**
- **Given** sqlc queries are defined in `backend/queries/pipeline_configs.sql`
- **When** I run `make generate`
- **Then** Go functions for GetPipelineConfig, UpsertPipelineConfig are generated

**AC4: Any authenticated user can get pipeline config for a project**
- **Given** I am authenticated and have access to a project
- **When** I GET /api/v1/projects/{projectId}/pipeline
- **Then** I receive HTTP 200 with pipeline config YAML

**AC5: Admin can update pipeline config**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/projects/{projectId}/pipeline with valid YAML
- **Then** I receive HTTP 200 with updated config and version incremented

**AC6: Non-admin cannot update pipeline config**
- **Given** I am authenticated as a non-admin user
- **When** I PUT /api/v1/projects/{projectId}/pipeline
- **Then** I receive HTTP 403

**AC7: Pipeline config validation on update**
- **Given** I am authenticated as admin
- **When** I PUT /api/v1/projects/{projectId}/pipeline with invalid action names
- **Then** I receive HTTP 400 with validation error

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration 000009 for pipeline_configs table (AC: #1)
  - [ ] Create `backend/migrations/000009_create_pipeline_configs_table.up.sql`
  - [ ] Create `backend/migrations/000009_create_pipeline_configs_table.down.sql`
  - [ ] Define pipeline_configs table: id (UUID PK DEFAULT gen_random_uuid()), project_id (UUID UNIQUE NOT NULL REFERENCES projects CASCADE), config_yaml (TEXT NOT NULL), version (INT NOT NULL DEFAULT 1), created_at (TIMESTAMPTZ NOT NULL DEFAULT now()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
  - [ ] Add index on project_id (unique constraint enforces one config per project)
  - [ ] Down migration drops the pipeline_configs table

- [ ] [BACK] Task 2: Create sqlc queries for PipelineConfig CRUD (AC: #3)
  - [ ] Create `backend/queries/pipeline_configs.sql`
  - [ ] Write `GetPipelineConfig` query (SELECT by project_id)
  - [ ] Write `UpsertPipelineConfig` query (INSERT ... ON CONFLICT (project_id) DO UPDATE, incrementing version, RETURNING *)
  - [ ] Run `make generate` and verify generated code compiles

- [ ] [BACK] Task 3: Create PipelineConfig domain model and port interface (AC: #4-7)
  - [ ] Create `backend/internal/domain/model/pipeline_config.go` with PipelineConfig struct
  - [ ] Create `backend/internal/domain/port/pipeline_config_repository.go` with PipelineConfigRepository interface
  - [ ] Define methods: GetByProjectID, Upsert
  - [ ] Use domain types (uuid.UUID, time.Time) not sqlc-generated types

- [ ] [BACK] Task 4: Implement Postgres adapter for PipelineConfigRepository (AC: #3)
  - [ ] Create `backend/internal/adapter/postgres/pipeline_config_repo.go`
  - [ ] Implement PipelineConfigRepository interface using sqlc-generated Queries
  - [ ] Map between domain model and sqlc-generated types
  - [ ] Handle pgx errors (not found -> domain error)

- [ ] [BACK] Task 5: Create PipelineConfigService with validation logic (AC: #4-7)
  - [ ] Create `backend/internal/domain/service/pipeline_config_service.go`
  - [ ] Implement GetByProjectID, Upsert
  - [ ] Add YAML validation (parse config_yaml, validate step action names against ActionRegistry)
  - [ ] Service depends on PipelineConfigRepository port + ActionRegistry (validation only)
  - [ ] Generate default pipeline config YAML for new projects

- [ ] [BACK] Task 6: Create PipelineConfigHandler with RBAC-protected routes (AC: #4-6)
  - [ ] Create `backend/internal/api/handler/pipeline_config_handler.go`
  - [ ] Implement GET /api/v1/projects/{projectId}/pipeline (any authenticated user) -> HTTP 200
  - [ ] Implement PUT /api/v1/projects/{projectId}/pipeline (admin only) -> HTTP 200
  - [ ] Use auth middleware from Story 1-3 for authentication
  - [ ] Use admin check from auth context for RBAC (PUT -> 403 if not admin)
  - [ ] Register routes on chi router under `/api/v1/projects/{projectId}/pipeline`

- [ ] [BACK] Task 7: Add default pipeline config seeding logic (AC: #2)
  - [ ] Update ProjectService.Create to call PipelineConfigService.Upsert with default config after project creation
  - [ ] Default config YAML includes steps: agent_run, hitl_gate, git_create_pr, git_merge
  - [ ] Each step has: name, action, model (if agent_run), auto_approve (if hitl_gate), retry_policy

- [ ] [BACK] Task 8: Add unit tests and wire into main.go (AC: #1-7)
  - [ ] Create `backend/internal/domain/service/pipeline_config_service_test.go`
  - [ ] Test validation: invalid action names return error
  - [ ] Test GetByProjectID, Upsert
  - [ ] Create `backend/internal/api/handler/pipeline_config_handler_test.go`
  - [ ] Test RBAC: admin can update, non-admin returns 403
  - [ ] Instantiate PipelineConfigRepository, PipelineConfigService, PipelineConfigHandler in main.go
  - [ ] Mount pipeline config routes on the chi router
  - [ ] Run migration 000009 against dev database
  - [ ] Manual test: create project -> verify default pipeline config exists
  - [ ] Manual test: admin PUT pipeline config with valid YAML -> 200
  - [ ] Manual test: admin PUT with invalid action name -> 400

## Dev Notes

This story adds pipeline configuration management: database table, sqlc queries, hexagonal layers (model, port, adapter, service, handler), RBAC-protected HTTP endpoints, and automatic seeding of default pipeline config on project creation.

### Dependencies

**Story 1-5 (Projects table):** PipelineConfig endpoints are nested under `/api/v1/projects/{projectId}/pipeline`. The projects table must exist for the foreign key constraint. Default pipeline config is created when a project is created.

**Story 1-3 (Users + Auth):** Auth middleware must exist for JWT cookie validation and user context extraction. The handler reads user role from auth context to enforce RBAC.

**Story 3-3 (ActionRegistry):** PipelineConfigService validates action names against the ActionRegistry to ensure only valid actions are configured. If ActionRegistry is not yet implemented, defer validation to a later story and add a TODO comment.

### Architecture Requirements

**Hexagonal Architecture - Exact file paths:**

```
backend/
├── migrations/
│   ├── 000009_create_pipeline_configs_table.up.sql
│   └── 000009_create_pipeline_configs_table.down.sql
├── queries/
│   └── pipeline_configs.sql
├── internal/
│   ├── domain/
│   │   ├── model/
│   │   │   └── pipeline_config.go              # PipelineConfig struct (domain model)
│   │   ├── port/
│   │   │   └── pipeline_config_repository.go   # PipelineConfigRepository interface
│   │   └── service/
│   │       ├── pipeline_config_service.go      # PipelineConfigService (business logic)
│   │       └── pipeline_config_service_test.go # Unit tests
│   ├── adapter/
│   │   └── postgres/
│   │       └── pipeline_config_repo.go         # PipelineConfigRepository impl (uses sqlc)
│   └── api/
│       └── handler/
│           ├── pipeline_config_handler.go      # HTTP handlers + route registration
│           └── pipeline_config_handler_test.go # Unit tests
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

- Migration: `backend/migrations/000009_create_pipeline_configs_table.{up,down}.sql`
- sqlc queries: `backend/queries/pipeline_configs.sql`
- Domain model: `backend/internal/domain/model/pipeline_config.go`
- Port interface: `backend/internal/domain/port/pipeline_config_repository.go`
- Service: `backend/internal/domain/service/pipeline_config_service.go`
- Service tests: `backend/internal/domain/service/pipeline_config_service_test.go`
- Postgres adapter: `backend/internal/adapter/postgres/pipeline_config_repo.go`
- Handler: `backend/internal/api/handler/pipeline_config_handler.go`
- Handler tests: `backend/internal/api/handler/pipeline_config_handler_test.go`

### Technical Specifications

**Migration 000009 SQL:**
```sql
-- 000009_create_pipeline_configs_table.up.sql
CREATE TABLE pipeline_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID UNIQUE NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    config_yaml TEXT NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipeline_configs_project_id ON pipeline_configs(project_id);

-- 000009_create_pipeline_configs_table.down.sql
DROP TABLE IF EXISTS pipeline_configs;
```

**sqlc query signatures (`backend/queries/pipeline_configs.sql`):**
```sql
-- name: GetPipelineConfig :one
SELECT * FROM pipeline_configs WHERE project_id = $1;

-- name: UpsertPipelineConfig :one
INSERT INTO pipeline_configs (project_id, config_yaml, version)
VALUES ($1, $2, 1)
ON CONFLICT (project_id) DO UPDATE
SET config_yaml = EXCLUDED.config_yaml,
    version = pipeline_configs.version + 1,
    updated_at = now()
RETURNING *;
```

**Domain model (`backend/internal/domain/model/pipeline_config.go`):**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type PipelineConfig struct {
    ID         uuid.UUID
    ProjectID  uuid.UUID
    ConfigYAML string
    Version    int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// PipelineStep represents a single step in the pipeline YAML
type PipelineStep struct {
    Name        string                 `yaml:"name"`
    Action      string                 `yaml:"action"`
    Model       *string                `yaml:"model,omitempty"`
    AutoApprove *bool                  `yaml:"auto_approve,omitempty"`
    RetryPolicy *RetryPolicy           `yaml:"retry_policy,omitempty"`
    Params      map[string]interface{} `yaml:"params,omitempty"`
}

type RetryPolicy struct {
    MaxRetries int    `yaml:"max_retries"`
    Strategy   string `yaml:"strategy"` // fixed, exponential
}

// PipelineConfigYAML represents the parsed YAML structure
type PipelineConfigYAML struct {
    Steps []PipelineStep `yaml:"steps"`
}
```

**Port interface (`backend/internal/domain/port/pipeline_config_repository.go`):**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

type PipelineConfigRepository interface {
    GetByProjectID(ctx context.Context, projectID uuid.UUID) (*model.PipelineConfig, error)
    Upsert(ctx context.Context, config *model.PipelineConfig) (*model.PipelineConfig, error)
}
```

**Default pipeline config YAML:**
```yaml
steps:
  - name: agent_run
    action: agent_run
    model: claude-opus-4-6
    retry_policy:
      max_retries: 3
      strategy: exponential
  - name: hitl_gate
    action: hitl_gate
    auto_approve: false
  - name: git_create_pr
    action: git_create_pr
  - name: git_merge
    action: git_merge
```

**RBAC logic:**
- Extract user from auth context (set by Story 1-3 auth middleware)
- PUT: check `user.Role == "admin"`, return 403 if not
- GET: allow any authenticated user

**Error responses (match OpenAPI error envelope):**
```json
{
  "error": {
    "code": "INVALID_PIPELINE_CONFIG",
    "message": "Invalid action name: invalid_action"
  }
}
```

**Error codes used:**
- `PIPELINE_CONFIG_NOT_FOUND` — config not found for project (404)
- `INVALID_PIPELINE_CONFIG` — invalid YAML or action names (400)
- `VALIDATION_ERROR` — missing/invalid fields (400)
- `FORBIDDEN` — non-admin trying to mutate (403)

### Testing Requirements

**Manual verification checklist:**
1. Run migration: `migrate -path migrations/ -database $DB_URL up`
2. Verify table: `\d pipeline_configs` shows all columns with correct types and defaults
3. Run `make generate` -- sqlc generates pipeline config query functions
4. `go build ./...` compiles successfully
5. `golangci-lint run ./...` passes with no errors
6. Create project via API -> verify default pipeline config exists in DB
7. Admin GET `/api/v1/projects/{projectId}/pipeline` -> 200 with YAML
8. Admin PUT `/api/v1/projects/{projectId}/pipeline` with valid YAML -> 200 with version incremented
9. Admin PUT with invalid action name -> 400 with validation error
10. Non-admin PUT `/api/v1/projects/{projectId}/pipeline` -> 403
11. Non-admin GET `/api/v1/projects/{projectId}/pipeline` -> 200 (read is allowed)
12. Verify version increments on each update

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture -- Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Pipeline Configuration & Templates]
- [Source: Story 1-5 (projects CRUD) — same hexagonal pattern]

## Dev Agent Record

## Change Log
