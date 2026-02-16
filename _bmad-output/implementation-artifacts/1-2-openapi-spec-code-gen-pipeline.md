# Story 1.2: OpenAPI spec + code-gen pipeline

Status: ready-for-dev

## Story

As a backend developer,
I want a contract-first development workflow with automated code generation,
so that API contracts are the source of truth.

## Acceptance Criteria (BDD)

**AC1: OpenAPI spec structure**
- **Given** api/openapi.yaml exists
- **When** I examine the spec
- **Then** I see OpenAPI 3.0 with auth, user, and project endpoints defined

**AC2: Code generation execution**
- **Given** oapi-codegen and sqlc configs exist
- **When** I run `make generate`
- **Then** Go server interfaces, types, and DB query functions are generated without errors

**AC3: Spec validation**
- **Given** api/openapi.yaml is valid
- **When** I run openapi-lint validation
- **Then** the spec passes with no errors

## Tasks / Subtasks

- [ ] Task 1: Create OpenAPI 3.0 specification (AC: #1)
  - [ ] Create `api/` directory in project root
  - [ ] Create `api/openapi.yaml` with OpenAPI 3.0 structure
  - [ ] Define info section (title, version, description)
  - [ ] Define servers section (API base path `/api/v1`)
  - [ ] Define security schemes (JWT bearer via httpOnly cookie)
  - [ ] Define shared components (schemas, responses, parameters)
  - [ ] Define auth endpoints (`POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`)
  - [ ] Define user endpoints (`GET /users`, `GET /users/{id}`, `PUT /users/{id}`, `DELETE /users/{id}`)
  - [ ] Define project endpoints (`GET /projects`, `POST /projects`, `GET /projects/{id}`, `PUT /projects/{id}`, `DELETE /projects/{id}`)
  - [ ] Define error response schemas (error envelope pattern)

- [ ] Task 2: Configure oapi-codegen (AC: #2)
  - [ ] Create `.oapi-codegen.yaml` in backend directory
  - [ ] Configure server generation (chi-compatible server interfaces)
  - [ ] Configure types generation (request/response structs)
  - [ ] Set package names (`backend/internal/api/handler` for handlers, `backend/internal/api/types` for types)
  - [ ] Set output paths for generated code
  - [ ] Configure strict validation and type generation

- [ ] Task 3: Configure sqlc (AC: #2)
  - [ ] Create `sqlc.yaml` in backend directory
  - [ ] Configure database engine (`pgx/v5`)
  - [ ] Set queries directory (`backend/queries/`)
  - [ ] Set output directory (`backend/internal/adapter/postgres/`)
  - [ ] Configure type overrides for UUIDs and timestamps
  - [ ] Set package name for generated code

- [ ] Task 4: Create Makefile with generate target (AC: #2)
  - [ ] Create `Makefile` in backend directory
  - [ ] Add `generate` target that runs oapi-codegen
  - [ ] Add `generate` target that runs sqlc (depends on migrations existing)
  - [ ] Add `lint-api` target for OpenAPI spec validation
  - [ ] Add `help` target documenting all commands
  - [ ] Ensure generate target is idempotent

- [ ] Task 5: Install and configure code-gen tools (AC: #2)
  - [ ] Document oapi-codegen version in `backend/README.md` or `backend/tools.go`
  - [ ] Document sqlc version in `backend/README.md` or `backend/tools.go`
  - [ ] Create `backend/tools.go` with tool dependencies (Go tools management pattern)
  - [ ] Add installation instructions to backend README

- [ ] Task 6: Validate OpenAPI spec (AC: #3)
  - [ ] Install openapi-lint or similar validator
  - [ ] Run validation on `api/openapi.yaml`
  - [ ] Fix any validation errors
  - [ ] Document validation command in Makefile

- [ ] Task 7: Test code generation (AC: #2)
  - [ ] Run `make generate` from backend directory
  - [ ] Verify generated files exist in expected locations
  - [ ] Verify generated Go code compiles (`go build ./...`)
  - [ ] Commit generated code to `.gitignore` (generated code should NOT be committed)
  - [ ] Add `.gitignore` entries for generated code directories

## Dev Notes

### Architecture Requirements

**Contract-First Philosophy:**
- `api/openapi.yaml` is the **single source of truth** for the API contract
- Backend handlers are generated from the spec (oapi-codegen)
- Frontend client will be generated from the same spec (openapi-typescript + openapi-fetch)
- Changes to the API ALWAYS start with updating the spec first

**Code Generation Approach:**
- Code-gen-first architecture per Architecture Decision Document
- Three code-gen tools in the stack:
  1. `oapi-codegen`: OpenAPI spec → chi handlers + Go types
  2. `sqlc`: SQL queries → type-safe Go functions
  3. `go-wire`: Provider sets → compile-time DI (comes in later stories)

**OpenAPI Spec Location:**
- Path: `api/openapi.yaml` (project root, not inside backend/)
- Rationale: Shared contract between backend and frontend

**Generated Code Output Paths:**
- oapi-codegen server interfaces → `backend/internal/api/handler/`
- oapi-codegen types → `backend/internal/api/types/` or inline with handlers
- sqlc queries → `backend/internal/adapter/postgres/`

**API Versioning:**
- Base path: `/api/v1`
- All endpoints prefixed with `/api/v1`
- Version in path enables future v2 without breaking existing clients

**Endpoint Conventions (from Architecture):**
- Plural nouns: `/users`, `/projects`, `/runs`
- Kebab-case for multi-word: `/pipeline-configs`, `/run-steps`
- Route params: `{id}` format (OpenAPI standard)
- Query params: `snake_case` (`project_id`, `per_page`, `sort_by`)

### Technical Specifications

**OpenAPI Version:**
- OpenAPI 3.0.3 (stable, widely supported)
- NOT 3.1 (less tooling support as of 2026)

**oapi-codegen Configuration:**
- Version: `v2.4.1` or latest stable
- Package: `github.com/deepmap/oapi-codegen/v2`
- Flags/Options:
  - `--package=handler` (for server interfaces)
  - `--generate=chi-server,types`
  - `--output=internal/api/handler/generated.go`
  - Chi server interface generation (not Gin, Echo, etc.)
- Config file: `.oapi-codegen.yaml` in backend directory

**sqlc Configuration:**
- Version: `v1.28.0` or latest stable
- Package: `github.com/sqlc-dev/sqlc`
- Config: `sqlc.yaml` in backend directory
- SQL package: `pgx/v5` (NOT database/sql or pgx/v4)
- Output: `internal/adapter/postgres/` with package name `postgres`

**Makefile Targets:**
```makefile
.PHONY: generate
generate: ## Generate code from OpenAPI spec and SQL queries
	@echo "Generating API handlers and types from OpenAPI spec..."
	oapi-codegen -config .oapi-codegen.yaml ../api/openapi.yaml
	@echo "Generating database query functions from SQL..."
	sqlc generate

.PHONY: lint-api
lint-api: ## Validate OpenAPI spec
	@echo "Validating OpenAPI spec..."
	# Use redocly, spectral, or openapi-generator validate
	npx @redocly/cli lint ../api/openapi.yaml

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
```

**Authentication Scheme (OpenAPI):**
- JWT via httpOnly secure cookie (NOT Bearer token header)
- Security scheme name: `cookieAuth`
- Type: `apiKey`
- In: `cookie`
- Name: `token` (cookie name)

**Error Response Format (from Architecture):**
```yaml
components:
  schemas:
    Error:
      type: object
      required:
        - error
      properties:
        error:
          type: object
          required:
            - code
            - message
          properties:
            code:
              type: string
              example: "USER_NOT_FOUND"
            message:
              type: string
              example: "User with ID abc123 not found"
            details:
              type: object
              additionalProperties: true
```

**Response Format (from Architecture):**
- Single resource: Direct object, HTTP 200/201
- List: Array with pagination metadata, HTTP 200
```yaml
components:
  schemas:
    PaginatedResponse:
      type: object
      properties:
        data:
          type: array
          items: {}
        pagination:
          type: object
          properties:
            total:
              type: integer
            page:
              type: integer
            per_page:
              type: integer
```

### File Structure

**Files to create:**

1. `api/openapi.yaml` — OpenAPI 3.0 specification
2. `backend/.oapi-codegen.yaml` — oapi-codegen configuration
3. `backend/sqlc.yaml` — sqlc configuration
4. `backend/Makefile` — Build automation with generate target
5. `backend/tools.go` — Go tools dependency management
6. `backend/.gitignore` — Ignore generated code directories
7. `backend/README.md` — Update with code-gen instructions

**Directory structure after this story:**
```
hopeitworks/
├── api/
│   └── openapi.yaml              # API contract - single source of truth
├── backend/
│   ├── .oapi-codegen.yaml        # oapi-codegen config
│   ├── sqlc.yaml                 # sqlc config
│   ├── Makefile                  # Build targets including 'generate'
│   ├── tools.go                  # Go tool dependencies
│   ├── .gitignore                # Ignore generated code
│   ├── go.mod                    # From Story 1.1
│   ├── go.sum                    # From Story 1.1
│   ├── README.md                 # Update with codegen instructions
│   ├── internal/
│   │   ├── api/
│   │   │   └── handler/
│   │   │       └── generated.go  # Generated (gitignored)
│   │   └── adapter/
│   │       └── postgres/
│   │           └── *.sql.go      # Generated (gitignored)
│   └── queries/                  # SQL queries (created in later stories)
└── frontend/                     # Not touched in this story
```

### Testing Requirements

**OpenAPI Spec Validation:**
- Use `npx @redocly/cli lint` or `spectral lint`
- Spec must pass validation with zero errors
- Validation should be part of CI pipeline (future story)

**Generated Code Compilation:**
- After running `make generate`, run `go build ./...` from backend directory
- Verify no compilation errors
- Generated code should compile even if not yet used in application

**Manual Verification Checklist:**
- [ ] `api/openapi.yaml` exists and is valid OpenAPI 3.0
- [ ] Running `make generate` from `backend/` succeeds
- [ ] Generated files appear in `backend/internal/api/handler/`
- [ ] Generated files appear in `backend/internal/adapter/postgres/` (once queries exist)
- [ ] `go build ./...` from `backend/` succeeds
- [ ] `.gitignore` excludes generated code directories
- [ ] `make help` shows all available targets

**Note on sqlc generation:**
- sqlc will only generate successfully once SQL query files exist in `backend/queries/`
- This story sets up the configuration; actual query generation happens in Story 1.3+ (migrations and queries)
- For this story, sqlc generation may fail gracefully or be skipped if no queries exist yet

### Dependencies on Story 1.1

**Must exist from Story 1-1:**
- `backend/go.mod` with module name `github.com/zakari/hopeitworks/backend` (or similar)
- `backend/go.sum` (dependencies installed)
- Go module initialized and buildable
- Project directory structure: `backend/`, `frontend/`, `api/`

**Expected from Story 1-1:**
- `backend/cmd/api/main.go` stub (minimal, may just be package main with empty main())
- `backend/internal/` directory structure started
- Basic project scaffolding complete

**What this story adds:**
- API contract specification
- Code generation tooling and configuration
- Makefile automation
- Generated code output directories (gitignored)

### Project Structure Notes

**Alignment with Architecture:**
- OpenAPI spec location (`api/openapi.yaml`) matches Architecture Decision Document
- Backend package structure (`internal/api/handler/`, `internal/adapter/postgres/`) follows hexagonal architecture
- Code-gen-first approach is core architectural principle
- Naming conventions (snake_case in API, PascalCase in Go) per Architecture naming patterns

**Generated Code Policy:**
- Generated code should NOT be committed to git
- Add to `.gitignore`: `internal/api/handler/generated.go`, `internal/adapter/postgres/*.sql.go`
- Generated code is reproducible from source specs
- CI/CD regenerates code on each build

**Tool Management (tools.go pattern):**
```go
//go:build tools
// +build tools

package tools

import (
	_ "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
```
- Ensures tool versions are tracked in go.mod
- Enables `go install` to fetch exact versions
- Standard Go tooling practice

### References

- [Source: architecture.md#API Design] — OpenAPI 3.0 spec as single source of truth, oapi-codegen generates chi handlers
- [Source: architecture.md#Stack Decisions] — Code-gen philosophy: oapi-codegen for API, sqlc for DB
- [Source: architecture.md#Naming Patterns] — API endpoints use kebab-case, JSON fields use snake_case
- [Source: architecture.md#API & Communication Patterns] — Error format, response format, endpoint conventions
- [Source: architecture.md#Decision Impact Analysis] — Implementation sequence: OpenAPI spec first, then DB schema, then code generation
- [Source: epics.md#Story 1.2] — Acceptance criteria for OpenAPI spec and code generation
- [Source: prd.md#Technical Architecture] — Abstraction-first design, code-gen-first approach

## Dev Agent Record

### Agent Model Used

_To be filled by implementation agent_

### Debug Log References

_To be filled by implementation agent_

### Completion Notes List

_To be filled by implementation agent_

### File List

_To be filled by implementation agent_
