# Project Context — Current State

## Project Overview

**hopeitworks v2** — AI agent orchestration platform for automated software development pipelines.

- **Current phase:** MVP implementation (Epics 1-4: Foundation, Story Board, Pipeline Execution, Agent Runtime)
- **Tech stack:** Go backend, Vue 3 frontend, Postgres, Docker
- **Architecture:** Hexagonal (backend), feature-based + atomic shared (frontend)
- **Development model:** Solo developer + AI agents with strict domain boundaries

## Project Structure

```
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
| Epic 2 | Story board & management | NOT STARTED |
| Epic 3 | Pipeline execution engine | NOT STARTED |
| Epic 4 | Agent runtime & container management | NOT STARTED |
| Epic 5 | DAG scheduler & epic runs | NOT STARTED |
| Epic 6 | HITL gates & approval workflow | NOT STARTED |
| Epic 7 | Pipeline configuration & templates | NOT STARTED |
| Epic 8 | Real-time monitoring & SSE | NOT STARTED |
| Epic 9 | Cost tracking & observability | NOT STARTED |
| Epic 10 | Reference project & validation | NOT STARTED |

### Completed

- Story 1-1: Go project scaffolding + docker-compose dev stack

### In Progress (Wave 1)

- Story 1-2: OpenAPI spec + code generation pipeline
- Story 1-7: Vue scaffolding + PrimeVue + Tailwind setup
- Story 1-14: CLAUDE.md files for agent scoping

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

```
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
