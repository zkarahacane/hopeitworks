# Base Agent Instructions — Project-Wide Conventions

You are an AI agent working on the **hopeitworks** platform. Follow these conventions strictly for all code changes.

## Git Workflow

### Branch Naming

- Feature branches: `feat/{story-key}-{slug}` (e.g., `feat/1-14-claude-md-files`)
- Fix branches: `fix/{story-key}-{slug}` (e.g., `fix/1-3-ci-poller`)
- The branch name is provided to you — always work on the assigned branch

### Conventional Commits

Format: `type(scope): message`

Types:

- `feat` — new feature
- `fix` — bug fix
- `refactor` — code restructuring without behavior change
- `test` — adding or updating tests
- `docs` — documentation changes
- `chore` — build, CI, tooling changes

Scope matches the domain area:

- Backend: `pipeline`, `auth`, `api`, `dag`, `git`, `agent`, `event`, `cost`, `config`
- Frontend: `ui`, `stories`, `runs`, `approvals`, `dag`, `editor`, `auth`
- Shared: `api-spec`, `deploy`, `ci`

Rules:

- Message in imperative mood, lowercase, no period at end
- Body is optional — explains WHY, not WHAT
- One logical change per commit

Examples:

```text
feat(pipeline): add retry logic for failed steps
fix(auth): handle token expiry on page refresh
refactor(dag): extract cycle detection into helper
test(git): add integration tests for PR creation
chore(deploy): update docker-compose health checks
```

### Merge Strategy

- Squash merge by default
- Delete branch after merge
- PR title follows conventional commit format

## Commit Standards

- Scope matches domain (see list above)
- Message: imperative mood, lowercase, no period
- Body: optional, explains WHY not WHAT
- Footer: reference story key (e.g., `Refs: S-14`)

## Code Quality Standards

### Mandatory

- No commented-out code in commits
- No `console.log` or `fmt.Println` in production code (use structured logging)
- All exported functions and types must be documented
- Error messages must be actionable (include context: what failed, what was expected)
- No hardcoded secrets, tokens, or credentials — use environment variables
- No `TODO` or `FIXME` without a linked story key

### Formatting

- Backend: `gofmt` / `goimports` (enforced by golangci-lint)
- Frontend: Prettier + ESLint (enforced by lint scripts)
- Commit only formatted code

### Linting

- Backend: `cd backend && golangci-lint run ./...` — **MUST pass before committing**
- Frontend: `npm run lint`
- Configuration: `backend/.golangci.yml` (errcheck, staticcheck, gofmt, goimports, revive, goconst, etc.)
- golangci-lint is **enforced in CI** — PRs will fail if lint errors are present

## Testing Principles

- Every new feature has tests
- Tests must be deterministic — no flaky tests, no time-dependent assertions
- Use factories over static fixtures for test data
- Integration tests tagged or named separately from unit tests
- Test the behavior, not the implementation
- Aim for high coverage on business logic; skip trivial getters/setters

### Test Organization

- Tests co-located with source in `__tests__/` directories (both Go and Vue)
- Go: use `-short` flag to separate unit from integration tests
- Frontend: unit tests via Vitest, E2E tests via Playwright

## Documentation

- README updated for public API changes
- CHANGELOG.md follows Keep a Changelog format
- Code comments explain WHY, not WHAT
- Document non-obvious architectural decisions inline
- Generated code should never be manually edited — regenerate from source

## API Contract

- `api/openapi.yaml` is the single source of truth for the REST API
- All API changes start with updating the OpenAPI spec
- Both backend and frontend generate code from this spec
- Never manually write types that should be generated from the spec

## Naming Conventions

### Database

- Tables: `snake_case`, plural (`stories`, `run_steps`, `pipeline_configs`)
- Columns: `snake_case` (`created_at`, `project_id`, `retry_count`)
- Foreign keys: `{referenced_table_singular}_id` (`project_id`, `story_id`)
- Indexes: `idx_{table}_{columns}` (`idx_stories_project_id`, `idx_runs_status`)
- Constraints: `{table}_{type}_{columns}` (`runs_fk_story_id`, `stories_uq_key_project`)

### API

- Endpoints: plural nouns, kebab-case for multi-word (`/pipeline-configs`, `/run-steps`)
- Route params: `{id}` format (OpenAPI standard)
- Query params: `snake_case` (`project_id`, `per_page`, `sort_by`)
- JSON fields: `snake_case` (matches Go JSON tags and Postgres columns)
- Dates: ISO 8601 strings (`"2026-02-15T10:30:00Z"`)

### Events (SSE / Postgres NOTIFY)

- Format: `{entity}.{action}` dot-notation (`run.started`, `step.completed`, `hitl.pending`)
- Payload: JSON with `snake_case` fields

## API Response Format

### Success (single resource)

Direct object, HTTP 200/201:

```json
{ "id": "...", "summary": "...", "status": "..." }
```

### Success (list)

Array with pagination metadata, HTTP 200:

```json
{
  "data": [...],
  "pagination": { "total": 42, "page": 1, "per_page": 20 }
}
```

### Error

Consistent error envelope:

```json
{
  "error": {
    "code": "STORY_NOT_FOUND",
    "message": "Story S-03 not found in project X",
    "details": {}
  }
}
```

### Async Operations

Async operations return 202 Accepted:

```json
{ "epic_run_id": "...", "status": "scheduling", "stories_count": 5 }
```

## Error Handling Philosophy

- Errors are values — handle them explicitly, never ignore
- Wrap errors with context as they propagate up the call stack
- Error codes are `UPPER_SNAKE_CASE`
- Error messages are human-readable and actionable
- See "API Response Format > Error" above for the standard error envelope

## Code Generation Philosophy

This project follows a **code-gen-first** approach. Never manually write code that should be generated from a spec.

| Domain | Spec Source | Generator | Output |
|--------|-----------|-----------|--------|
| API handlers | `api/openapi.yaml` | oapi-codegen | chi server interfaces + types |
| API client | `api/openapi.yaml` | openapi-typescript + openapi-fetch | TypeScript typed fetch client |
| Database queries | `backend/queries/*.sql` | sqlc | type-safe Go functions |
| DI wiring | `wire.go` provider sets | go-wire | `wire_gen.go` auto-generated |
| Prompts | Handlebars templates | runtime rendering | agent prompts |

Generated files (e.g., `wire_gen.go`, `db/` for sqlc, `frontend/src/api/generated/`) must NEVER be manually edited — always regenerate from source.

## Security

- Never log secrets, tokens, or API keys
- Use environment variables for all sensitive configuration
- Validate all external input at system boundaries
- Agent containers have no host filesystem access

## CI Pipeline

### Backend CI

```bash
golangci-lint run ./...              # Lint
go test ./... -short                 # Unit tests (fast, no containers)
go test ./... -run Integration       # Integration tests (testcontainers)
```

### Frontend CI

```bash
npm run lint                         # ESLint
npm run type-check                   # tsc --noEmit
npm run test:unit                    # Vitest
npm run test:e2e                     # Playwright (against docker-compose.test.yml)
```

## Infrastructure

- **Docker Compose** in `deploy/` for local dev stack
- Health checks: `GET /health` (liveness) and `GET /ready` (readiness: DB + Docker socket)
- Config: `config.yaml` + env var override, resolved at startup
- No hot-reload for MVP — restart to apply config changes

## Local Dev Reset

Use `scripts/reset-dev.sh` to reset the local dev environment to a clean state:

```bash
./scripts/reset-dev.sh
```

This script:
1. Drops and recreates the DB schema
2. Restarts the API container (triggers migrations)
3. Creates all test data **via API calls** (no SQL seed)

After reset, the environment contains:
- 6 users (3 admin, 3 user) — login: `admin@hopeitworks.dev` / `admin1234`
- 1 project (Todo App) pointing to local Gitea
- 1 epic (MVP) with 3 stories in backlog
- 4 agents (2 global, 2 project-scoped)
- Pipeline config with 4 groups and 7 preconfigured steps
- Zero runs (clean slate)

## Story Implementation Pipeline

Stories are implemented via Docker containers running Claude Code agents. **Never implement stories directly in the local repo** — always use the pipeline scripts.

### Scripts

| Script | Purpose |
|--------|---------|
| `scripts/bmad-dev.sh` | Launch dev agent containers (clone mode or interactive) |
| `scripts/pipeline.sh` | Runs inside container: dev-story → code-review → merge-story |

### Default models per phase

| Phase | Default model | Override flag |
|-------|--------------|---------------|
| `dev-story` | `opus` | `--dev=MODEL` |
| `code-review` | `sonnet` | `--review=MODEL` |
| `merge-story` | `opus` | `--merge=MODEL` |

### Usage

```bash
# Full pipeline for a single story (dev → review → merge)
./scripts/bmad-dev.sh --story <story-key> --pipeline

# Full pipeline for an entire wave (all stories in parallel)
./scripts/bmad-dev.sh --wave <N> --pipeline

# Setup wave branch first (if using wave-based merging)
./scripts/bmad-dev.sh --wave <N> --setup

# Single phase on a story
./scripts/bmad-dev.sh --story <story-key> --phase dev-story
./scripts/bmad-dev.sh --story <story-key> --phase code-review
./scripts/bmad-dev.sh --story <story-key> --phase merge-story

# Override models per phase (reduce cost or test with lighter models)
./scripts/bmad-dev.sh --story <story-key> --pipeline --dev=sonnet
./scripts/bmad-dev.sh --wave <N> --pipeline --dev=sonnet --review=haiku --merge=sonnet

# Monitor running containers
./scripts/bmad-dev.sh --status
docker logs -f bmad-dev-<story-key>-pipeline
```

### Parallel execution rules

- Stories in the same wave can run in parallel (each in its own Docker container)
- Each container clones the repo independently — no git conflicts during dev
- **Never run parallel local agents on the same working directory** — use `git worktree` if needed
- Merge conflicts are resolved at merge time (sequential merge order matters)
- **Verify CI is green on develop before launching pipelines** — agents wait for CI green to merge

### Required env vars

- `CLAUDE_CODE_OAUTH_TOKEN` — OAuth token for Claude Code
- `GITHUB_TOKEN` — GitHub token for gh CLI

# Project Context — Current State

## Project Overview

**hopeitworks v2** — AI agent orchestration platform for automated software development pipelines.

- **Current phase:** MVP implementation (Epics 1-4: Foundation, Story Board, Pipeline Execution, Agent Runtime)
- **Tech stack:** Go backend, Vue 3 frontend, Postgres, Docker
- **Architecture:** Hexagonal (backend), feature-based + atomic shared (frontend)
- **Development model:** Solo developer + AI agents with strict domain boundaries

## Project Structure

```text
hopeitworks/
├── backend/                    # Go module — autonomous
│   ├── cmd/api/                # Entry point + wire.go
│   ├── internal/               # Domain, adapters, API, config
│   ├── pkg/                    # Shared utilities (log, errors, exec, config)
│   ├── migrations/             # SQL migrations (golang-migrate)
│   ├── queries/                # SQL queries (sqlc source)
│   ├── Makefile
│   └── Dockerfile
├── frontend/                   # Vue 3 project — autonomous
│   ├── src/                    # ui/, features/, composables/, stores/, api/, views/
│   ├── e2e/                    # Playwright tests
│   └── Dockerfile
├── api/
│   └── openapi.yaml            # Single source of truth — API contract
├── agent/                      # Agent runtime
│   ├── Dockerfile              # Project-specific agent image
│   ├── Dockerfile.base         # Base image: Claude Code + git + gh CLI
│   ├── entrypoint.sh           # Container entry script
│   ├── scripts/                # Runtime scripts (clone, inject CLAUDE.md, run agent, extract results)
│   ├── claude-md/              # CLAUDE.md templates (this directory)
│   └── prompts/                # Handlebars prompt templates
├── deploy/                     # Infrastructure
│   ├── docker-compose.yml      # Dev local stack
│   └── postgres/               # DB setup
├── test-project/               # Reference todo app (pipeline validation baseline)
└── scripts/                    # Ops / bootstrap scripts
```

## Key File Paths

| Purpose | Path |
|---------|------|
| API contract | `api/openapi.yaml` — single source of truth for REST API |
| Backend entry point | `backend/cmd/api/main.go` |
| Backend domain models | `backend/internal/domain/model/` |
| Backend port interfaces | `backend/internal/domain/port/` |
| Backend services | `backend/internal/domain/service/` |
| Backend adapters | `backend/internal/adapter/` |
| Backend API handlers | `backend/internal/api/handler/` |
| Backend middleware | `backend/internal/api/middleware/` |
| Backend migrations | `backend/migrations/*.sql` |
| Backend sqlc queries | `backend/queries/*.sql` |
| Backend test utilities | `backend/internal/testutil/` |
| Backend Go module | `backend/go.mod` |
| Frontend API client | `frontend/src/api/client.ts` — generated from openapi.yaml |
| Frontend shared UI | `frontend/src/ui/` |
| Frontend features | `frontend/src/features/` |
| Frontend composables | `frontend/src/composables/` |
| Frontend stores | `frontend/src/stores/` |
| Frontend views | `frontend/src/views/` |
| Frontend E2E tests | `frontend/e2e/tests/` |
| Docker Compose (dev) | `deploy/docker-compose.yml` |
| Agent scripts | `agent/scripts/` |
| Agent prompts | `agent/prompts/` |
| Architecture doc | `_bmad-output/planning-artifacts/architecture.md` |

## Shared API Contract

All API changes follow this workflow:

1. Update `api/openapi.yaml` (the single source of truth)
2. Regenerate backend: `cd backend && make generate` (oapi-codegen)
3. Regenerate frontend: `cd frontend && npm run generate-api` (openapi-typescript + openapi-fetch)
4. Implement handlers (backend) and views (frontend) — can run in parallel after spec merge

Both sides generate types and clients from the same OpenAPI spec. Never manually write types that should be generated.

## Current Implementation Status

| Epic | Description | Status |
|------|-------------|--------|
| Epic 1 | Project scaffolding & foundation | IN PROGRESS |
| Epic 2 | Story board & management | IN PROGRESS |
| Epic 3 | Pipeline execution engine | IN PROGRESS |
| Epic 4 | Agent runtime & container management | IN PROGRESS |
| Epic 5 | DAG scheduler & epic runs | NOT STARTED |
| Epic 6 | HITL gates & approval workflow | NOT STARTED |
| Epic 7 | Pipeline configuration & templates | NOT STARTED |
| Epic 8 | Real-time monitoring & SSE | IN PROGRESS |
| Epic 9 | Cost tracking & observability | IN PROGRESS |
| Epic 10 | Reference project & validation | NOT STARTED |

### Completed

- Story 1-1: Go project scaffolding + docker-compose dev stack
- Story 1-2: OpenAPI 3.0 spec + code generation pipeline
- Story 1-7: Vue 3 scaffolding + PrimeVue 4 + Tailwind CSS v4
- Story 1-14: CLAUDE.md files for agent scoping
- Story 2-1: Epic CRUD API and board view
- Story 2-2: Stories CRUD API with status filtering
- Story 3-10: Run launch API endpoint (single story)
- Story 3-11: Pipeline executor with step sequencing
- Story 4-1: Docker agent runtime with container lifecycle
- Story 4-2: Agent entrypoint with Claude Code integration
- Story 8-1: SSE event infrastructure (Postgres NOTIFY + SSE handler)
- Story 8-2: Run/step status SSE events and frontend wiring
- Auth: Login, logout, forgot/reset password, user profile
- Admin: User management CRUD
- Pipeline runtime fixes: River timeout, OAuth auth, healthcheck, template mount
- Pipeline wiring: story_key in Run API, story status transitions on run complete

## Known Constraints

- **Backend agents** work ONLY in the `backend/` directory
- **Frontend agents** work ONLY in the `frontend/` directory
- API contract changes require coordination between both sides
- MVP = measurement, not enforcement (cost tracking tracks but does not halt)
- Docker mode for MVP (Kubernetes deferred to Phase 2)
- No caching layer for MVP (no Redis) — Postgres is single source of truth
- No rate limiting for MVP — not needed at current scale
- No hot-reload for backend config — restart to apply changes
- Budget enforcement deferred to Phase 2 — MVP only tracks cost

## Code Generation Pipeline

| Domain | Spec Source | Generator | Output |
|--------|-----------|-----------|--------|
| API handlers | `api/openapi.yaml` | oapi-codegen | chi server interfaces + types |
| API client | `api/openapi.yaml` | openapi-typescript + openapi-fetch | TypeScript typed fetch client |
| Database queries | `backend/queries/*.sql` | sqlc | type-safe Go functions |
| DI wiring | `wire.go` provider sets | go-wire | `wire_gen.go` auto-generated |
| Prompts | Handlebars templates | runtime rendering | agent prompts |

## Architectural Boundaries

```text
┌─────────────┐     openapi.yaml      ┌──────────────┐
│   Frontend   │◄────────────────────►│   Backend    │
│   (Vue 3)    │   HTTP + SSE         │   (Go API)   │
└─────────────┘                       └──────┬───────┘
                                              │
                                   ┌──────────┼──────────┐
                                   │          │          │
                              Docker API   Postgres   gh CLI
                                   │          │          │
                              ┌────▼───┐ ┌───▼────┐ ┌──▼───┐
                              │ Agent  │ │  DB +  │ │GitHub│
                              │Contain.│ │ River  │ │ API  │
                              └────────┘ │+ Event │ └──────┘
                                         └────────┘
```

1. **Frontend <-> Backend**: REST API + SSE. Contract = `api/openapi.yaml`. No direct coupling.
2. **Backend <-> Postgres**: sqlc queries + pgxlisten + River jobs. Everything goes through ports.
3. **Backend <-> Docker**: Docker API via socket-proxy. AgentRuntime port.
4. **Backend <-> GitHub**: `gh` CLI via CommandRunner. GitProvider port.

## E2E Testing

### Commands
- `./scripts/e2e-stack.sh up|down|reset|status` — lifecycle stack de test
- `./scripts/e2e-smoke.sh` — lance la suite smoke complète avec rapport
- `npm run test:e2e:real` (dans frontend/) — lance uniquement les tests Playwright

### Delegation rules
When testing the app:
1. **ALWAYS delegate** test execution to a Task agent (Sonnet or Haiku)
2. **ALWAYS delegate** log/report analysis to a Task agent
3. **ALWAYS delegate** Playwright MCP exploration to a Task agent
4. Main thread only coordinates and synthesizes results
5. If an agent finds bugs, launch a fix agent per bug (Task Sonnet)
