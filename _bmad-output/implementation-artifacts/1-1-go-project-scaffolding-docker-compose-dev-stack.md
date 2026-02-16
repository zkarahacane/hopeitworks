# Story 1.1: Go project scaffolding + docker-compose dev stack

Status: ready-for-dev

## Story

As a backend developer,
I want a Go project scaffold with Docker development stack,
so that I have the project shell ready for implementing the backend services.

## Acceptance Criteria (BDD)

**AC1: Binary build succeeds**
- **Given** a fresh repository clone
- **When** I run `go build ./cmd/api`
- **Then** the build succeeds and a binary is produced

**AC2: Project structure is correct**
- **Given** the project structure is initialized
- **When** I examine the directory layout
- **Then** I see cmd/api/main.go, internal/, deploy/, and config.yaml

**AC3: Docker compose stack starts**
- **Given** docker-compose.yml is configured in deploy/
- **When** I run `docker compose -f deploy/docker-compose.yml up`
- **Then** Postgres and API containers start successfully

## Tasks / Subtasks

- [ ] Task 1: Initialize Go module and base project structure (AC: #1, #2)
  - [ ] Create backend/ directory
  - [ ] Initialize Go module: `go mod init github.com/zakari/hopeitworks/backend`
  - [ ] Create cmd/api/main.go entry point with minimal setup (just fmt.Println("Starting API server..."))
  - [ ] Create internal/ directory structure: domain/, adapter/, api/, config/
  - [ ] Create pkg/ directory structure: log/, errors/, config/, exec/
  - [ ] Create migrations/, queries/, testdata/ directories
  - [ ] Create .gitignore for Go (vendor/, *.exe, .env, etc.)
  - [ ] Verify `go build ./cmd/api` produces a binary

- [ ] Task 7: Create docker-compose.yml development stack (AC: #3)
  - [ ] Create deploy/docker-compose.yml
  - [ ] Define postgres service: official image, version 16
  - [ ] Configure postgres: database name, user, password via environment
  - [ ] Add postgres volume for data persistence
  - [ ] Add postgres healthcheck (pg_isready)
  - [ ] Define api service: build from backend/Dockerfile
  - [ ] Configure api environment variables (DB connection, log level)
  - [ ] Map api port 8080:8080
  - [ ] Add api depends_on postgres with condition: service_healthy
  - [ ] Create shared network for services
  - [ ] Add restart policies (on-failure)

- [ ] Task 8: Create backend Dockerfile (AC: #3)
  - [ ] Create backend/Dockerfile with multi-stage build
  - [ ] Stage 1: Builder - use golang:1.23-alpine
  - [ ] Copy go.mod, go.sum and download dependencies
  - [ ] Copy source code and build static binary
  - [ ] Stage 2: Runtime - use alpine:3.19
  - [ ] Copy binary from builder
  - [ ] Add ca-certificates for HTTPS
  - [ ] Create non-root user
  - [ ] Set ENTRYPOINT to binary
  - [ ] Document exposed port (8080)

- [ ] Task 9: Create .env.example and documentation (AC: #3)
  - [ ] Create backend/.env.example with all config variables documented
  - [ ] Include database connection variables
  - [ ] Include server configuration variables
  - [ ] Include logging configuration
  - [ ] Add comments explaining each variable
  - [ ] Create deploy/.env for local development defaults
  - [ ] Add .env to .gitignore

- [ ] Task 10: Create initial Makefile for common operations (AC: #1)
  - [ ] Create backend/Makefile
  - [ ] Add `make build` target (builds cmd/api)
  - [ ] Add `make run` target (runs local binary)
  - [ ] Add `make docker-up` target (docker compose up)
  - [ ] Add `make docker-down` target (docker compose down)
  - [ ] Add `make docker-logs` target (follows logs)
  - [ ] Add `make clean` target (removes binary, cleans cache)
  - [ ] Add help target with descriptions

## Dev Notes

This story creates the PROJECT SHELL: folder structure, Go module, docker-compose with Postgres/Redis, Dockerfile, env config, Makefile. No actual Go application code is implemented here - that comes in Story 1.15.

### Architecture Requirements

**Exact Project Structure:**
```
hopeitworks/
├── backend/                          # Go module root
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # Minimal entry point (just prints "Starting...")
│   ├── internal/                    # Private application code
│   │   ├── domain/
│   │   │   ├── model/               # Empty (created for future use)
│   │   │   ├── port/                # Empty (created for future use)
│   │   │   └── service/             # Empty (created for future use)
│   │   ├── adapter/
│   │   │   └── postgres/            # Empty (created for future use)
│   │   ├── api/
│   │   │   ├── handler/             # Empty (created for future use)
│   │   │   └── middleware/          # Empty (created for future use)
│   │   ├── eventbus/                # Empty (created for future use)
│   │   └── config/                  # Empty (created for future use)
│   ├── pkg/                         # Public shared utilities
│   │   ├── log/                     # Empty (created for future use)
│   │   ├── errors/                  # Empty (created for future use)
│   │   ├── exec/                    # Empty (created for future use)
│   │   └── config/                  # Empty (created for future use)
│   ├── migrations/                  # Empty (SQL migrations go here)
│   ├── queries/                     # Empty (sqlc queries go here)
│   ├── testdata/                    # Empty (test fixtures go here)
│   ├── .env.example                 # Environment variable template
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile                   # Multi-stage build
│   ├── Makefile                     # Common development commands
│   └── .gitignore
├── deploy/
│   ├── docker-compose.yml           # Local development stack
│   ├── .env                         # Local environment variables (gitignored)
│   └── postgres/                    # Empty (will contain init.sql)
└── .gitignore                       # Project-level gitignore
```

**Critical: Hexagonal Architecture Boundaries**
- `internal/domain/` contains business logic only - no external dependencies
- `internal/adapter/` contains implementations of domain ports - external integrations
- `internal/api/` contains HTTP handlers - entry point adapters
- `pkg/` contains reusable utilities with no domain coupling
- Empty directories must be created now to establish structure for future stories

### Technical Specifications

**Go Version & Module:**
- Go 1.23 (latest stable as of January 2025)
- Module name: `github.com/zakari/hopeitworks/backend`
- Use Go modules (go.mod, go.sum)
- No dependencies needed at this stage (Story 1.15 will add chi, pgx)

**Docker Compose Version:**
- Compose file version: 3.8 (no need to specify version key in modern Docker Compose)
- Services: postgres, api

**PostgreSQL Configuration:**
- Version: 16 (official Docker image: postgres:16-alpine)
- Database name: hopeitworks_dev
- Default user: hopeitworks
- Default password: hopeitworks_dev_password (change via .env)
- Port exposed: 5432 (host:5432 → container:5432)
- Healthcheck: `pg_isready -U hopeitworks`
- Volume: postgres_data (named volume for persistence)

**API Service Configuration:**
- Port: 8080 (configurable via SERVER_PORT env var)
- Build context: ../backend
- Dockerfile: backend/Dockerfile
- Depends on: postgres (service_healthy)
- Environment variables:
  - `APP_ENV=development`
  - `LOG_LEVEL=debug`
  - `DB_HOST=postgres` (Docker Compose service name)
  - `DB_PORT=5432`
  - `DB_NAME=hopeitworks_dev`
  - `DB_USER=hopeitworks`
  - `DB_PASSWORD=hopeitworks_dev_password`
  - `DB_SSLMODE=disable` (local dev only)

### File Structure

**Exact files to create with minimum viable content:**

1. **backend/go.mod** - Go module definition
2. **backend/cmd/api/main.go** - Minimal entry point (just `fmt.Println("Starting API server...")`)
3. **backend/.env.example** - Documented environment variables template
4. **backend/Dockerfile** - Multi-stage: builder (golang:1.23-alpine) + runtime (alpine:3.19)
5. **backend/Makefile** - build, run, docker-up, docker-down, docker-logs, clean, help targets
6. **backend/.gitignore** - Ignore: vendor/, *.exe, .env, coverage.out, *.log
7. **deploy/docker-compose.yml** - postgres + api services
8. **deploy/.env** - Local development environment variables (gitignored)

**Empty directories to create (critical for future structure):**
- backend/internal/domain/model/
- backend/internal/domain/port/
- backend/internal/domain/service/
- backend/internal/adapter/postgres/
- backend/internal/api/handler/
- backend/internal/api/middleware/
- backend/internal/eventbus/
- backend/internal/config/
- backend/pkg/log/
- backend/pkg/errors/
- backend/pkg/exec/
- backend/pkg/config/
- backend/migrations/
- backend/queries/
- backend/testdata/
- deploy/postgres/

### Testing Requirements

**Manual verification checklist:**
1. `go build ./cmd/api` succeeds
2. `docker compose -f deploy/docker-compose.yml up` starts both services
3. API container starts and prints "Starting API server..."
4. Postgres container is healthy via `docker compose ps`
5. All directory structure exists as specified

### Project Structure Notes

**Alignment with Architecture Document:**
- Follows hexagonal architecture layout exactly as specified
- `internal/domain/` for pure business logic (currently empty scaffolds)
- `internal/adapter/` for external integrations (currently empty scaffolds)
- `internal/api/` for HTTP layer (currently empty scaffolds)
- `pkg/` for reusable utilities (currently empty scaffolds)
- Strict boundary: domain never imports adapter or api

**Docker Best Practices:**
- Multi-stage builds to minimize image size
- Non-root user in runtime image
- Healthchecks for proper orchestration
- Named volumes for data persistence
- Explicit depends_on with service_healthy conditions

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#Project Structure Decision: Monorepo with Strict Boundaries]
- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture — Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Stack Decisions]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 1: Project Foundation & Authentication]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.1]

## Dev Agent Record

### Agent Model Used

_To be filled by the dev agent after implementation_

### Debug Log References

_To be filled by the dev agent after implementation_

### Completion Notes List

_To be filled by the dev agent after implementation. Include:_
- Any deviations from the spec and rationale
- Issues encountered and solutions
- Additional files created beyond the spec
- Recommendations for future stories

### File List

_To be filled by the dev agent after implementation. List all files created or modified with absolute paths._
