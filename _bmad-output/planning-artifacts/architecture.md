---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
lastStep: 8
status: 'complete'
completedAt: '2026-02-16'
inputDocuments:
  - '_bmad-output/planning-artifacts/prd.md'
workflowType: 'architecture'
project_name: 'hopeitworks'
user_name: 'Zakari'
date: '2026-02-15'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**
53 FRs across 10 domains: Project Management (5), Story Management (3), Pipeline Execution (10), Pipeline Configuration (4), HITL (4), Agent & Container Management (6), Prompt Management (4), Real-Time Monitoring (4), Cost & Observability (4), Auth (4), Git Operations (4), Test Environment (1). The pipeline execution domain (FR9-FR18) is the most architecturally significant — it encompasses the DAG scheduler, parallel execution, state machine transitions, CI polling, and incremental retry logic.

**Non-Functional Requirements:**
15 NFRs driving key architectural decisions:
- Performance: SSE < 1s latency, API < 500ms, DAG computation < 2s for 50+ stories, container startup < 30s
- Security: Container isolation via Docker, docker-socket-proxy for socket access, env-only secrets (never persisted), slog scrubbing, JWT with configurable secret/expiration
- Reliability: API crash tolerance (containers survive), full state persistence in Postgres (resumable), orphan container cleanup on startup, circuit breaker (default 3 failures), hard timeout per container (default 30min), evergreen test project baseline

**Scale & Complexity:**
- Primary domain: Backend-heavy full-stack (Go API core, Vue 3 for visibility/control)
- Complexity level: Medium-high
- Estimated architectural components: 8-10 major subsystems (API, Scheduler/DAG, Pipeline Engine, Container Manager, Event Bus, SSE Server, Git Provider, Agent Runtime, Prompt Engine, Auth)

### Technical Constraints & Dependencies

- **Go backend** with chi (router), pgx/v5 + sqlc (DB), oapi-codegen (OpenAPI-first) — API-first design
- **Postgres** as single data store + event bus (LISTEN/NOTIFY via pgxlisten) — no external message broker
- **Docker** as container runtime — docker-socket-proxy for security
- **Vue 3 + TypeScript** frontend — SPA consuming REST API + SSE
- **Solo developer + AI agents** — architecture must be agent-implementable (clear boundaries, simple interfaces, explicit contracts)
- **Dogfooding constraint** — the platform must be able to develop itself, meaning early pipeline stability is critical path

### Cross-Cutting Concerns Identified

1. **Event propagation**: Every state change (run started, step completed, CI result, HITL pending) must flow through Postgres LISTEN/NOTIFY → SSE → UI and → Notifier (Discord/webhook). Single event source, multiple consumers.
2. **Container lifecycle management**: Create → inject env → stream logs → monitor → cleanup. Must handle: normal completion, failure, timeout, API crash (orphans), circuit breaker halt.
3. **Cost tracking**: Token/cost data captured per step, aggregated per run, per story, per project. Budget measurement only for MVP (no enforcement).
4. **Error handling & retry strategy**: Incremental retry (diff + error → fix) with fallback to full retry after 2 failures. Affects pipeline engine, container manager, and prompt rendering.
5. **Security boundary**: Agent containers isolated (no host FS), secrets injected via env only, docker-socket-proxy allowlists operations, slog scrubs sensitive values. Zero trust between API and containers.
6. **Observability**: slog JSON structured logging on stdout, LGTM-compatible. No built-in dashboards for MVP, but structured enough for future Grafana integration.
7. **Multi-tenancy readiness**: project_id on all tables, no global state, but MVP ships with simple JWT admin/user roles.

## Starter Template Evaluation

### Primary Technology Domain

Backend-heavy full-stack — Go API as core orchestration engine, Vue 3 SPA for visibility/control. **Strict separation** between backend and frontend: different stories, different agents, shared API contract only.

### Project Structure Decision: Monorepo with Strict Boundaries

```
hopeitworks/
├── backend/                    # Go module — autonomous
│   ├── cmd/api/                # Entry point + wire.go (DI wiring)
│   ├── internal/
│   │   ├── domain/
│   │   │   ├── model/          # Entities: Story, Run, RunStep, Project, Epic, Event, PipelineConfig, User
│   │   │   ├── port/           # Interfaces: GitProvider, AgentRuntime, Repository, Notifier, EventPublisher, Transactor, Action, JobQueue
│   │   │   └── service/        # Business logic: PipelineService, SchedulerService, ActionRegistry
│   │   ├── adapter/
│   │   │   ├── action/         # Action implementations: agent_run, ci_poll, git_branch, git_pr, git_merge, hitl_gate, notify, script
│   │   │   ├── github/         # GitProvider implementation (via gh CLI)
│   │   │   ├── docker/         # AgentRuntime implementation
│   │   │   ├── postgres/       # All Repository impls (sqlc generated) + Transactor + EventPublisher
│   │   │   ├── discord/        # Notifier implementation (webhook)
│   │   │   ├── webhook/        # Generic webhook Notifier
│   │   │   └── river/          # JobQueue implementation (River)
│   │   ├── api/
│   │   │   ├── handler/        # oapi-codegen generated handlers + SSE handler
│   │   │   └── middleware/     # Auth JWT, CORS, request logging, error mapping (DomainError → HTTP)
│   │   ├── eventbus/           # pgxlisten wrapper: subscribe/publish on Postgres channels
│   │   ├── config/             # App config loading (YAML + env override)
│   │   └── testutil/           # Helpers: testcontainers setup, factories, assertions
│   ├── pkg/
│   │   ├── log/                # slog helpers: WithLogger, LoggerFrom, ScrubHandler
│   │   ├── errors/             # DomainError + categories + constructors
│   │   ├── exec/               # CommandRunner interface (for testable CLI calls)
│   │   └── config/             # Config struct definitions
│   ├── migrations/             # SQL migrations (golang-migrate)
│   ├── queries/                # SQL queries (sqlc source)
│   ├── testdata/               # Fixtures SQL + stories markdown de test
│   ├── sqlc.yaml
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   └── CLAUDE.md               # Agent instructions: backend only
├── frontend/                   # Vue 3 project — autonomous
│   ├── src/
│   │   ├── ui/                          # Atomic layer (shared only)
│   │   │   ├── primitives/              # PrimeVue wrappers, base components
│   │   │   ├── composed/                # Reusable combinations
│   │   │   └── layout/                  # Page structure (AppShell, PageHeader, SplitPanel)
│   │   ├── features/                    # By business domain
│   │   │   ├── projects/
│   │   │   ├── stories/
│   │   │   ├── runs/
│   │   │   ├── dag/
│   │   │   ├── approvals/
│   │   │   └── pipeline-editor/
│   │   ├── composables/                 # Shared functional (pure)
│   │   ├── stores/                      # Pinia stores
│   │   ├── api/                         # openapi-fetch client
│   │   ├── theme/                       # PrimeVue tokens + config
│   │   ├── assets/                      # main.css (@layer tailwind-base, primevue, tailwind-utilities)
│   │   ├── router/                      # Routes with auth guards
│   │   ├── views/                       # 1 view = 1 route, composes features
│   │   └── utils/                       # Pure functions (formatters, parsers)
│   ├── e2e/
│   │   ├── fixtures/           # Seed data for E2E
│   │   └── tests/              # Playwright specs
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── Dockerfile
│   └── CLAUDE.md               # Agent instructions: frontend only
├── api/
│   └── openapi.yaml            # Single source of truth — API contract
├── agent/                      # Agent runtime — what containers execute
│   ├── Dockerfile              # Project-specific agent image (extends base)
│   ├── Dockerfile.base         # Base image: Claude Code + git + gh CLI + common tools
│   ├── entrypoint.sh           # Container entry script
│   ├── scripts/
│   │   ├── clone-and-setup.sh      # Clone repo + checkout branch
│   │   ├── inject-claude-md.sh     # Compose CLAUDE.md from base + context-specific
│   │   ├── run-agent.sh            # Run Claude Code with rendered prompt
│   │   └── extract-results.sh      # Extract results (files changed, PR URL, errors)
│   ├── claude-md/              # CLAUDE.md templates injected per context
│   │   ├── base.md             # Common agent instructions
│   │   ├── backend.md          # Backend-specific agent instructions
│   │   ├── frontend.md         # Frontend-specific agent instructions
│   │   └── project.md          # Project-specific agent instructions
│   └── prompts/                # Handlebars prompt templates
│       ├── implement.hbs
│       ├── implement-retry.hbs
│       ├── review.hbs
│       ├── merge-conflict.hbs
│       └── custom/             # User-defined custom prompt templates
├── deploy/                     # Infrastructure / deployment
│   ├── docker-compose.yml      # Dev local (postgres, api, frontend, socket-proxy)
│   ├── docker-compose.prod.yml # Production overrides
│   ├── docker-compose.test.yml # Functional test stack
│   ├── postgres/
│   │   └── init.sql            # DB init + extensions
│   └── socket-proxy/
│       └── config.env          # Docker socket allowlist
├── test-project/               # Reference todo app (pipeline validation baseline)
│   └── CLAUDE.md
├── scripts/                    # Ops / bootstrap scripts
│   ├── dev-setup.sh
│   └── seed.sql
├── CLAUDE.md                   # Global project instructions
└── README.md
```

**Boundary Rules:**
- Backend agents NEVER touch `frontend/`
- Frontend agents NEVER touch `backend/`
- `api/openapi.yaml` is the shared contract — changes require coordination
- Each side has its own `CLAUDE.md` scoping agent behavior
- Each side has its own `Dockerfile` and independent build

### Starter Options Considered

**Backend (Go):**
No monolithic starter template. Code-gen-first approach:
- `oapi-codegen`: OpenAPI spec → chi handlers + types
- `sqlc`: SQL queries → type-safe Go functions
- `go-wire`: Provider sets → compile-time DI
- Standard Go layout (`cmd/`, `internal/`, `pkg/`)

**Frontend (Vue 3):**
`create-vue` with TypeScript, Vitest, ESLint, Prettier.

### Selected Approach: Manual Go + create-vue

**Rationale:**
- Code-gen-first philosophy drives project structure on both sides
- Strict separation enables parallel agent work with zero conflicts
- OpenAPI spec as single coupling point — both sides generate from it

**Initialization Commands:**

```bash
# Backend
cd backend && go mod init github.com/zakari/hopeitworks/backend

# Frontend
npm create vue@latest frontend -- --typescript

# API contract
mkdir api && touch api/openapi.yaml
```

### Stack Decisions (Refined from PRD)

**Router: chi**
Retained from PRD. Middleware composition and sub-routers needed for auth, SSE, CORS.

**DB Access: pgx/v5 + sqlc** (changed from sqlx)
sqlx wraps database/sql which does not support Postgres LISTEN/NOTIFY — the core event bus of this architecture. pgx/v5 provides native LISTEN/NOTIFY support. sqlc generates type-safe Go code from SQL queries, aligning with the code-gen-first philosophy (oapi-codegen for API, sqlc for DB).

**Event Bus: pgxlisten** (added)
Wrapper around pgx LISTEN/NOTIFY with auto-reconnection and per-channel dispatching. Dedicated connection for event listening.

**Job Queue: River** (added)
Postgres-based Go job queue. Jobs stored in same DB as domain data — transactional enqueue. No infrastructure addition (no Redis, no RabbitMQ). Built-in retry, scheduling, dead-letter. Go-native, active maintenance.

**Dependency Injection: go-wire (Google)** (added)
Compile-time DI via code generation. Aligns with code-gen-first philosophy (oapi-codegen, sqlc, go-wire).

**API Code Generation: oapi-codegen**
Generates chi-compatible server interfaces from OpenAPI 3.0 spec. Contract-first development.

**Code-Gen Philosophy:**

| Domain | Spec Source | Generator | Output |
|--------|-----------|-----------|--------|
| API handlers | `api/openapi.yaml` | oapi-codegen | chi server interfaces + types |
| API client | `api/openapi.yaml` | openapi-typescript + openapi-fetch | TypeScript typed fetch client |
| Database | `backend/queries/*.sql` | sqlc | type-safe Go functions |
| DI wiring | `wire.go` provider sets | go-wire | `wire_gen.go` auto-generated |
| Prompts | Handlebars templates | runtime rendering | agent prompts |

**Frontend:**
- Vue 3 + TypeScript via create-vue
- Vite dev server + production build
- Composition API + Pinia stores
- PrimeVue 4 (Aura preset, unstyled mode with CSS layers)
- Tailwind CSS v4 (layout utilities only)
- API client generated from shared OpenAPI spec via openapi-fetch

**Infrastructure:**
- Postgres (official Docker image)
- docker-socket-proxy (linuxserver.io or tecnativa)
- docker-compose.yml in `deploy/` as entry point

### Testing Strategy — 4 Levels

**Level 1 — Unit Tests (fast, isolated):**

| Domain | Tool | Scope |
|--------|------|-------|
| Backend | Go `testing` + table-driven tests | Pure business logic: DAG builder, state machine, prompt rendering, cost calculation |
| Frontend | Vitest + Vue Test Utils | Isolated composables, Pinia stores, utils |

**Level 2 — Integration Tests (real dependencies):**

| Domain | Tool | Scope |
|--------|------|-------|
| Backend | testcontainers-go + real Postgres | sqlc queries against real DB, migrations, LISTEN/NOTIFY, full repository layer |
| Backend | httptest + oapi-codegen client | API handlers with real DB, JWT auth, SSE streaming |
| Frontend | Vitest + MSW (Mock Service Worker) | API-connected components with mocked HTTP, realistic user flows |

**Level 3 — E2E Tests (real browser):**

| Tool | Scope |
|------|-------|
| Playwright | Critical UI flows from all 4 PRD journeys: login → view projects → launch story → follow SSE logs → approve HITL → view result. Every button clicked, every state verified. |

**Level 4 — Pipeline Functional Tests (full system):**

| Tool | Scope |
|------|-------|
| `test-project/` (todo app) + `deploy/docker-compose.test.yml` | Launch a real story on test-project, verify: container created → agent executes → PR opened → CI passes → review → merge. The ultimate smoke test. |

**Test Infrastructure:**

- `backend/testdata/`: SQL fixtures + test story markdown files
- `backend/internal/testutil/`: Shared helpers — testcontainers Postgres setup + migrations + seed, factories (`NewProject()`, `NewStory()`, `NewRun()`), custom business assertions
- `frontend/e2e/fixtures/`: Seed data for E2E (via API calls in setup)
- `frontend/e2e/tests/`: Playwright specs per critical flow (auth, story-run, hitl-approve, sse-streaming, pipeline-config)

**Testing Principles:**
- testcontainers-go: each integration test gets its own ephemeral Postgres. No shared test database.
- Factories over static fixtures in Go: `factory.NewStory(WithDeps("S-01", "S-02"))` — readable, composable
- Hand-written mocks implementing port interfaces (no mockgen)
- Playwright for critical E2E: not "click every button" but all 4 PRD user journey flows covered.
- Go `-short` flag: unit tests run in seconds, integration tests tagged separately.

**Note:** Project scaffolding (backend init + frontend init + deploy setup + OpenAPI stub) should be the first implementation stories, split by domain.

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**
- Data model + migrations (pgx/v5 + sqlc + golang-migrate)
- Auth flow (JWT via httpOnly cookie)
- SSE streaming architecture (pgxlisten → chi handler → EventSource)
- API contract design (OpenAPI 3.0, oapi-codegen)

**Important Decisions (Shape Architecture):**
- Error format standardization (DomainError pattern)
- Frontend component library (PrimeVue 4, unstyled + Aura)
- Config management (YAML + env override)
- Health check endpoints
- Dependency injection (go-wire, compile-time)
- Job queue (River, Postgres-native)

**Deferred Decisions (Post-MVP):**
- Caching layer (no Redis for MVP)
- Rate limiting (not needed at current scale)
- API versioning beyond v1
- Budget enforcement (MVP = measurement only)
- Visual pipeline editor (MVP = YAML editing)

### Data Architecture

**Database:** Postgres (single instance via docker-compose)

**Schema Design:**
Core entities: `projects`, `epics`, `stories`, `runs`, `run_steps`, `events`, `users`, `prompt_templates`, `pipeline_configs`
- All tables include `project_id` FK for multi-tenancy readiness
- `events` table as append-only event log, source for LISTEN/NOTIFY triggers
- Timestamps: `created_at`, `updated_at` on all tables (UTC)

**Migration Tool:** golang-migrate
- Pure SQL migration files in `backend/migrations/`
- Numbered sequential: `000001_init_schema.up.sql` / `.down.sql`
- Run automatically on API startup (or via CLI flag)

**Query Layer:** sqlc
- SQL queries in `backend/queries/*.sql`
- Generated type-safe Go code in `backend/internal/adapter/postgres/`
- Config: `sqlc.yaml` with `sql_package: "pgx/v5"`

**Caching:** None for MVP. Postgres is the single source of truth. Volume does not justify Redis. Revisit in Phase 2 if needed.

### Authentication & Security

**JWT Library:** golang-jwt/jwt/v5

**Token Flow:**
- Login endpoint returns JWT in `httpOnly` secure cookie
- Frontend does not handle tokens directly — browser sends cookie automatically
- Token contains: `user_id`, `role` (admin/user), `exp`
- Configurable secret and expiration via config

**CORS:** chi middleware
- Dev: `localhost:*`
- Prod: specific domain(s) from config

**Secrets Management:**
- Environment variables only (.env file in dev, injected by docker-compose)
- Never persisted in database or logs
- slog scrubbing middleware strips tokens/keys from structured log output

**Docker Socket Security:**
- docker-socket-proxy with allowlisted operations only
- Agent containers: no host filesystem access, no privileged mode
- Network isolation: dedicated Docker network for agent containers

### API & Communication Patterns

**API Design:** REST, OpenAPI 3.0 spec as single source of truth
- Generated chi handlers via oapi-codegen
- Prefix: `/api/v1/`
- Contract-first: spec written before implementation

**Error Format:**
```json
{
  "error": {
    "code": "STORY_NOT_FOUND",
    "message": "Story S-03 not found in project X",
    "details": {}
  }
}
```
Error types defined in OpenAPI spec, generated by oapi-codegen. Backend uses DomainError pattern (see Backend Architecture — Foundations).

**SSE Streaming:**
- Endpoint: `GET /api/v1/events/stream?project_id={id}`
- Content-Type: `text/event-stream`
- chi handler with per-client goroutine
- Fed by eventbus (pgxlisten subscriptions)
- Event types: `run.started`, `step.completed`, `step.failed`, `hitl.pending`, `run.completed`, `log.line`
- Client reconnection via `Last-Event-ID` header

**Rate Limiting:** Deferred. Not needed at MVP scale (solo + few colleagues).

### Frontend Architecture (Summary)

**Component Library:** PrimeVue 4 (Aura preset, unstyled mode with CSS layers)
- Tailwind CSS v4 for layout utilities only
- CSS layer order: `tailwind-base, primevue, tailwind-utilities`
- Design tokens at 3 levels (primitive → semantic → component)

**State Management:** Pinia
- Stores per domain: `auth`, `projects`, `stories`, `runs`, `approvals`, `templates`
- SSE events update stores reactively

**Routing:** Vue Router
- Route structure by feature: `/projects`, `/projects/:id/stories`, `/runs/:id`
- Navigation guards for auth

**SSE Client:** `useSSE` composable wrapping EventSource
- Auto-reconnect on disconnect
- Dispatches events to Pinia stores
- Cleanup on component unmount

**API Client:** Generated from `api/openapi.yaml` via openapi-typescript + openapi-fetch
- Type-safe fetch calls with `credentials: 'include'`
- Shared types between spec and frontend

### Infrastructure & Deployment

**Health Checks:**
- `GET /health` — liveness (API is running)
- `GET /ready` — readiness (DB connected + Docker socket accessible)
- Used in docker-compose healthcheck directives

**Logging:** slog (Go stdlib)
- JSON format on stdout
- Structured fields: `request_id`, `user_id`, `project_id`, `run_id`, `step_id`
- ScrubHandler middleware strips sensitive values (tokens, API keys)
- LGTM-compatible for future Grafana integration

**Config Management:**
- Single `config.yaml` read at boot
- Environment variables override any YAML value
- Resolved into typed Go config struct at startup
- No hot-reload for MVP — restart to apply changes

**Container Networking:**
- Dedicated Docker network for API ↔ agent containers
- API communicates with agents via Docker API (through socket-proxy), not direct network calls
- Agent containers get: cloned repo, env vars (secrets, CLAUDE.md path), network access to git remote only

### Decision Impact Analysis

**Implementation Sequence:**
1. OpenAPI spec (`api/openapi.yaml`) — defines the contract first
2. Database schema + migrations — data model foundation
3. sqlc queries + generated code — data access layer
4. oapi-codegen handlers — API skeleton
5. Core backend services (eventbus, pipeline, scheduler)
6. Frontend scaffolding + PrimeVue setup
7. Frontend views consuming API + SSE
8. Agent runtime (Dockerfile, entrypoint, prompts)
9. Deploy stack (docker-compose)
10. Test infrastructure (all 4 levels)

**Cross-Component Dependencies:**
- OpenAPI spec → backend handlers + frontend API client (both generated)
- sqlc queries → backend services (data access)
- eventbus (pgxlisten) → SSE handler → frontend stores (real-time chain)
- Pipeline engine → container manager → agent runtime (execution chain)
- Auth middleware → all API endpoints + frontend route guards

---

## Backend Architecture — Foundations

### Dependency Injection: go-wire (Google)

- Compile-time DI via code generation (not runtime like go-fx/Uber)
- Aligns with code-gen-first philosophy (oapi-codegen, sqlc, go-wire)
- `wire.go` files define provider sets, `wire_gen.go` is auto-generated
- Each domain service gets its own provider set
- Main wiring in `cmd/api/wire.go`

### Logger: slog (Go stdlib)

- Chosen for open-source friendliness — standard library, no external dep
- JSON output on stdout, LGTM-compatible
- Context enrichment pattern: `pkg/log/WithLogger(ctx, logger)` and `pkg/log/LoggerFrom(ctx)`
- ScrubHandler wrapping: sanitizes sensitive values (tokens, API keys) before output
- Structured fields: request_id, user_id, project_id, run_id, step_id

### Error Handling: DomainError Pattern

- stdlib `errors` package only, no external deps
- Custom DomainError struct with categories:

```go
type ErrorCategory string
const (
    NotFound     ErrorCategory = "not_found"
    Validation   ErrorCategory = "validation"
    Conflict     ErrorCategory = "conflict"
    Unauthorized ErrorCategory = "unauthorized"
    Forbidden    ErrorCategory = "forbidden"
    Internal     ErrorCategory = "internal"
)

type DomainError struct {
    Category ErrorCategory
    Code     string    // e.g. "STORY_NOT_FOUND"
    Message  string
    Cause    error
}
```

- Constructors: `errors.NewNotFound("story", storyID)`, `errors.NewValidation("field", "reason")`
- API middleware maps DomainError.Category → HTTP status code
- Services return DomainError, adapters wrap external errors into DomainError

### Transaction Management: Transactor Pattern

- Transaction lives in context, services orchestrate, repos extract tx from context

```go
type Transactor interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

- Postgres adapter injects `pgx.Tx` into context
- Repository methods extract tx from context, fallback to pool if no tx
- Services call `transactor.WithinTransaction(ctx, func(ctx) { repo.Save(ctx, ...) })`

---

## Backend Architecture — Hexagonal Structure

### Package Layout

```
backend/internal/
├── domain/
│   ├── model/          # Entities: Story, Run, RunStep, Project, Epic, Event, PipelineConfig, User
│   ├── port/           # Interfaces: GitProvider, AgentRuntime, Repository, Notifier, EventPublisher, Transactor, Action, JobQueue
│   └── service/        # Business logic: PipelineService, SchedulerService, ActionRegistry
├── adapter/
│   ├── action/         # Action implementations: agent_run, ci_poll, git_branch, git_pr, git_merge, hitl_gate, notify, script
│   ├── github/         # GitProvider implementation (via gh CLI)
│   ├── docker/         # AgentRuntime implementation
│   ├── postgres/       # All Repository impls (sqlc generated) + Transactor + EventPublisher
│   ├── discord/        # Notifier implementation (webhook)
│   ├── webhook/        # Generic webhook Notifier
│   └── river/          # JobQueue implementation (River)
├── api/
│   ├── handler/        # oapi-codegen generated handlers + SSE handler
│   └── middleware/     # Auth JWT, CORS, request logging, error mapping (DomainError → HTTP)
├── eventbus/           # pgxlisten wrapper: subscribe/publish on Postgres channels
└── config/             # App config loading (YAML + env override)

backend/pkg/
├── log/                # slog helpers: WithLogger, LoggerFrom, ScrubHandler
├── errors/             # DomainError + categories + constructors
├── exec/               # CommandRunner interface (for testable CLI calls)
└── config/             # Config struct definitions
```

### Key Interfaces (Ports)

```go
type Action interface {
    Name() string
    Execute(ctx context.Context, params ActionParams) (*ActionResult, error)
}

type GitProvider interface {
    CreateBranch(ctx context.Context, repo, base, branch string) error
    CreatePR(ctx context.Context, repo string, pr PRRequest) (*PRResponse, error)
    MergePR(ctx context.Context, repo string, prNumber int, strategy string) error
    GetPRDiff(ctx context.Context, repo string, prNumber int) (string, error)
    GetCIStatus(ctx context.Context, repo, ref string) (*CIStatus, error)
}

type AgentRuntime interface {
    Run(ctx context.Context, params AgentParams) (<-chan AgentEvent, error)
    Stop(ctx context.Context, containerID string) error
}

type Transactor interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type JobQueue interface {
    Enqueue(ctx context.Context, job any) error
    EnqueueTx(ctx context.Context, tx any, job any) error
}

type EventPublisher interface {
    Publish(ctx context.Context, eventType string, payload any) error
}
```

---

## Backend Architecture — Domain Services

### Domain 1: PipelineService (State Machine + River)

The pipeline engine is **event-driven**, not a blocking loop. Uses **River** (Postgres-based Go job queue) for step orchestration.

**Flow:**
1. User triggers story run → PipelineService creates Run + first RunStep
2. First step job enqueued in River (within same transaction)
3. River worker picks up job → executes Action via ActionRegistry
4. On step completion → worker enqueues next step job (or pauses for HITL)
5. HITL steps: no new job enqueued, pipeline pauses naturally. User approval → enqueues next step.

**State Machine:** `pending → running → completed|failed|cancelled`
- RunStep states: `pending → running → completed|failed|skipped|waiting_approval`
- Transitions validated by PipelineService (no illegal state jumps)

**Why River over external broker:**
- Postgres-native: jobs stored in same DB as domain data → transactional enqueue
- No infrastructure addition (no Redis, no RabbitMQ)
- Built-in retry, scheduling, dead-letter
- Go-native, active maintenance

**Retry Strategy:**
- `incremental`: on failure, new prompt includes diff + error → agent fixes
- After 2 incremental failures → `full`: clean retry from scratch
- Max retries configurable in pipeline YAML `on_failure.max_retries`

### Domain 2: SchedulerService (DAG Builder)

Builds execution DAG for epic runs (multiple stories in parallel where possible).

**Dependency Resolution:**
1. Explicit: `depends_on: [S-01, S-03]` in story frontmatter
2. Implicit via provides/requires:

```yaml
# Story S-03 frontmatter
provides: ["port.GitProvider", "adapter/github/provider.go"]
requires: ["model.Project", "port.Repository"]

# Story S-07 frontmatter
requires: ["port.GitProvider.GetCIStatus"]  # → implicitly depends on S-03
```

3. DAG builder resolves: explicit deps + provides/requires matching → topological sort
4. Stories with no dependency between them run in parallel

**Max concurrency:** Configurable per project (`max_parallel_runs`), prevents resource exhaustion.

**File conflict detection:** The DAG builder performs file scope overlap detection to prevent parallel execution of stories with overlapping target files. Stories declare their target files in frontmatter (`provides: ["path/to/file.go"]`). If two stories target the same file, they are scheduled sequentially (dependency edge added automatically). This is prevention at the scheduling level, not merge conflict resolution.

**Merge conflicts:** If conflicts still occur (e.g., unexpected file overlap, concurrent edits to shared imports), a dedicated `conflict_resolve` agent step handles it post-merge-failure. This is reactive conflict resolution, not prevention.

### Domain 3: Composable Actions + Pipeline YAML

Actions are **atomic primitives**. Pipelines are **YAML compositions** of actions.

**Built-in Actions:**

| Action | Description |
|--------|-------------|
| `agent_run` | Launch Claude Code agent in container |
| `ci_poll` | Poll CI status at interval until pass/fail/timeout |
| `git_create_branch` | Create branch via gh CLI |
| `git_create_pr` | Create PR via gh CLI |
| `git_merge` | Merge PR (squash/merge/rebase) |
| `hitl_gate` | Pause pipeline, notify user, wait for approval |
| `notify` | Send notification (Discord, SSE, webhook) |
| `script` | Run arbitrary script in container |

**ActionRegistry:** Central registry mapping action names → Action implementations. New actions added by implementing the Action interface and registering.

**Pipeline YAML Example:**

```yaml
name: default
on_failure:
  max_retries: 2
  retry_strategy: incremental
  notify: [discord, sse]
steps:
  - name: create-branch
    action: git_create_branch
    params: { base: develop, pattern: "feat/{{story.key}}-{{story.slug}}" }
  - name: implement
    action: agent_run
    params: { prompt: implement.hbs, model: sonnet, timeout: 30m }
  - name: create-pr
    action: git_create_pr
    params: { title: "feat({{story.scope}}): {{story.title}}" }
  - name: ci
    action: ci_poll
    params: { interval: 30s, timeout: 20m }
  - name: review
    action: agent_run
    params: { prompt: review.hbs, model: sonnet }
  - name: approve
    action: hitl_gate
    params: { notify: [discord, sse] }
  - name: merge
    action: git_merge
    params: { strategy: squash, delete_branch: true }
  - name: done
    action: notify
    params: { channels: [discord, sse], message: "{{story.key}} merged" }
```

**MVP:** Pre-built pipeline templates (default, review-only, no-ci). User can edit YAML.
**Post-MVP:** Visual pipeline editor in frontend.

### Domain 4: Agent Runtime

**Container Lifecycle:**
1. Docker creates container from agent image
2. `entrypoint.sh` runs: clone repo → checkout branch → inject CLAUDE.md (composed from base + context-specific) → run Claude Code with rendered prompt
3. Agent outputs NDJSON to stdout (streamed via Docker logs API)
4. On completion: extract results (files changed, PR URL, errors) → cleanup container

**NDJSON Event Format:**
```json
{"type": "log", "level": "info", "message": "Starting implementation..."}
{"type": "file_changed", "path": "internal/service/foo.go", "action": "create"}
{"type": "cost", "input_tokens": 1500, "output_tokens": 800, "model": "sonnet"}
{"type": "result", "status": "success", "files_changed": 5}
```

**Agent Image:**
- `Dockerfile.base`: Claude Code + git + gh CLI + common tools
- Project can extend via `agent/Dockerfile` (additional deps) or `agent/setup.sh` (runtime install)
- CLAUDE.md composed at runtime: `base.md` + `backend.md` or `frontend.md` + project-specific `project.md`

**Prompt Rendering:** Handlebars templates. Variables: story metadata, project config, previous attempt diff/error (for retry), PR diff (for review).

### Domain 5: Remaining Services

**GitProvider (adapter/github/):**
- Wraps `gh` CLI via CommandRunner interface (for testability)
- Operations: create-branch, create-pr, merge-pr, get-pr-diff, get-ci-status
- CommandRunner interface in `pkg/exec/` allows mocking in tests

**EventBus (eventbus/):**
- pgxlisten wrapping Postgres LISTEN/NOTIFY
- Dedicated connection (separate from query pool)
- Channels per event type: `run.started`, `step.completed`, `step.failed`, `hitl.pending`, etc.
- SSE handler subscribes to eventbus, fans out to connected clients
- River also publishes events on job completion

**Auth (api/middleware/):**
- JWT in httpOnly secure cookie
- golang-jwt/jwt/v5
- Middleware extracts user from token, injects in context
- Two roles: admin, user (MVP)
- Configurable secret + expiration

**PromptRenderer:**
- Handlebars templates in `agent/prompts/`
- Built-in templates: implement.hbs, implement-retry.hbs, review.hbs, merge-conflict.hbs
- Custom templates supported in `agent/prompts/custom/`
- Variables injected from story metadata + run context

**Notifier (adapter/discord/ + adapter/webhook/):**
- Dispatcher pattern: event → routing rules → channel(s)
- MVP: Discord webhook + SSE (built-in)
- Discord: webhook URL per project in config
- Events with urgency levels: info (step complete), warning (retry), critical (HITL needed, failure)

**CostTracker:**
- Passive tracking only (MVP = measure, not enforce)
- NDJSON `cost` events from agent → stored per RunStep
- SQL aggregation for reporting: per step, per run, per story, per project
- Frontend displays cost breakdown

---

## API Design

### REST Endpoints

**Auth:**
- `POST /api/v1/auth/login` — returns JWT in httpOnly cookie
- `POST /api/v1/auth/logout` — clears cookie
- `GET /api/v1/auth/me` — current user info

**Projects:**
- `GET /api/v1/projects` — list projects
- `POST /api/v1/projects` — create project
- `GET /api/v1/projects/{id}` — project detail
- `PUT /api/v1/projects/{id}` — update project
- `DELETE /api/v1/projects/{id}` — delete project

**Pipeline Config:**
- `GET /api/v1/projects/{id}/pipeline-config` — get pipeline YAML
- `PUT /api/v1/projects/{id}/pipeline-config` — update pipeline YAML
- `GET /api/v1/templates` — list pipeline templates
- `GET /api/v1/templates/{name}` — get template detail

**Epics & Stories:**
- `GET /api/v1/projects/{id}/epics` — list epics
- `POST /api/v1/projects/{id}/epics` — create epic
- `GET /api/v1/projects/{id}/stories` — list stories (filterable by epic, status)
- `POST /api/v1/projects/{id}/stories` — create story
- `POST /api/v1/projects/{id}/stories/sync` — sync stories from markdown files (upserts)
- `GET /api/v1/stories/{id}` — story detail
- `PUT /api/v1/stories/{id}` — update story

**Runs:**
- `POST /api/v1/stories/{id}/runs` — launch story run → returns 201 with run
- `POST /api/v1/epics/{id}/runs` — launch epic run → returns 202 with epic_run (trackable entity)
- `GET /api/v1/runs` — list runs (filterable by project, status)
- `GET /api/v1/runs/{id}` — run detail with steps
- `GET /api/v1/runs/{id}/steps/{step_id}/logs` — step logs

**HITL Approvals:**
- `GET /api/v1/approvals` — list pending approvals
- `POST /api/v1/runs/{id}/approve` — approve HITL gate
- `POST /api/v1/runs/{id}/reject` — reject HITL gate (with reason)

**Scheduler:**
- `GET /api/v1/epics/{id}/dag` — get DAG visualization data
- `POST /api/v1/epics/{id}/dag/preview` — preview DAG without running

**SSE:**
- `GET /api/v1/events/stream?project_id={id}` — SSE stream

**Test (APP_ENV=test only):**
- `POST /api/v1/test/seed` — seed test data
- `POST /api/v1/test/reset` — reset test data

### Response Format

**Success (single):** Direct object, HTTP 200/201
```json
{ "id": "...", "summary": "...", "status": "..." }
```

**Success (list):** Array with pagination metadata, HTTP 200
```json
{
  "data": [...],
  "pagination": { "total": 42, "page": 1, "per_page": 20 }
}
```

**Error:** Consistent error envelope
```json
{
  "error": {
    "code": "STORY_NOT_FOUND",
    "message": "Story S-03 not found in project X",
    "details": {}
  }
}
```

**Epic run:** Returns 202 Accepted (async operation)
```json
{ "epic_run_id": "...", "status": "scheduling", "stories_count": 5 }
```

### API Contract

- `api/openapi.yaml` is the **single source of truth**
- Backend: `oapi-codegen` → Go server interfaces + request/response types
- Frontend: `openapi-typescript` + `openapi-fetch` → typed TypeScript client
- Changes to API require updating the spec FIRST, then regenerating both sides

### Git Flow

- Branch naming: `feat/S-01-setup-project`, `fix/S-03-ci-poller`
- Conventional commits: `feat(pipeline): add retry logic`, `fix(auth): token expiry`
- PR workflow via `gh` CLI (create, review, merge)
- Squash merge by default, delete branch after merge

---

## Frontend Architecture

### Stack

- Vue 3 + TypeScript (Composition API exclusively)
- PrimeVue 4 (Aura preset, unstyled mode with CSS layers)
- Tailwind CSS v4 (layout utilities only — AI agents work well with it)
- Pinia (state management)
- Vue Router (with auth guards)
- openapi-fetch (generated typed API client)
- Vitest (unit tests)
- Playwright (E2E tests)

### Hybrid Structure: Feature + Atomic for Shared

```
frontend/src/
├── ui/                          # Atomic layer (shared only)
│   ├── primitives/              # PrimeVue wrappers, base components
│   │   ├── StatusBadge.vue      # Badge with status → color mapping
│   │   ├── CodeBlock.vue        # Code/log display
│   │   └── EmptyState.vue
│   ├── composed/                # Reusable combinations
│   │   ├── DataTable.vue        # Table + pagination + loading + empty
│   │   ├── ConfirmDialog.vue    # Dialog + standardized actions
│   │   ├── LogViewer.vue        # Stream logs with ANSI support
│   │   └── TimelineStep.vue     # Visual step with state
│   └── layout/                  # Page structure
│       ├── AppShell.vue         # Sidebar + header + content
│       ├── PageHeader.vue       # Title + breadcrumb + actions
│       └── SplitPanel.vue       # Resizable panel
│
├── features/                    # By business domain
│   ├── projects/
│   │   ├── ProjectList.vue
│   │   ├── ProjectSettings.vue
│   │   └── composables/useProjectForm.ts
│   ├── stories/
│   │   ├── StoryBoard.vue       # Kanban view
│   │   ├── StoryDetail.vue
│   │   ├── StoryEditor.vue
│   │   └── composables/useStoryFilters.ts
│   ├── runs/
│   │   ├── RunTimeline.vue      # Step timeline
│   │   ├── RunDetail.vue
│   │   ├── StepOutput.vue       # NDJSON logs for a step
│   │   └── composables/useRunPolling.ts
│   ├── dag/
│   │   ├── DagGraph.vue         # DAG visualization
│   │   ├── DagControls.vue
│   │   └── composables/useDagLayout.ts
│   ├── approvals/
│   │   ├── ApprovalQueue.vue
│   │   ├── DiffViewer.vue
│   │   └── composables/useApprovalActions.ts
│   └── pipeline-editor/
│       ├── PipelineCanvas.vue   # Visual YAML editor
│       ├── ActionPalette.vue    # Available actions list
│       ├── StepConfigForm.vue
│       └── composables/usePipelineValidation.ts
│
├── composables/                 # Shared functional (pure)
│   ├── useSSE.ts                # EventSource + dispatch to stores
│   ├── useAuth.ts               # JWT lifecycle
│   ├── usePagination.ts         # Generic pagination logic
│   ├── useAsyncAction.ts        # loading + error + execute pattern
│   └── useKeyboard.ts           # Keyboard shortcuts
│
├── stores/                      # Pinia stores
│   ├── auth.ts
│   ├── projects.ts
│   ├── stories.ts
│   ├── runs.ts
│   ├── approvals.ts
│   └── templates.ts
│
├── api/                         # openapi-fetch client
│   └── client.ts                # createClient<paths>({ baseUrl: '/api/v1', credentials: 'include' })
│
├── theme/
│   ├── tokens.ts                # HopeTheme = definePreset(Aura, {...})
│   └── index.ts                 # PrimeVue config export
│
├── assets/
│   └── main.css                 # @layer tailwind-base, primevue, tailwind-utilities
│
├── router/
│   └── index.ts                 # Routes with auth guards
│
├── views/                       # 1 view = 1 route, composes features
│   ├── LoginView.vue
│   ├── DashboardView.vue
│   ├── ProjectsView.vue
│   ├── ProjectDetailView.vue
│   ├── StoriesView.vue
│   ├── RunDetailView.vue
│   ├── ApprovalsView.vue
│   ├── DagView.vue
│   └── PipelineEditorView.vue
│
└── utils/                       # Pure functions (formatters, parsers)
```

**Component rule:** If used by 2+ features → `ui/`. Otherwise stays in its feature.

### PrimeVue Setup

- **Mode:** Unstyled with Aura preset
- **CSS Layers order:** `tailwind-base, primevue, tailwind-utilities` (Tailwind utilities override PrimeVue)
- **Theming:** Design tokens at 3 levels (primitive → semantic → component)
- **Dark mode:** `.dark` class on `<html>`, persisted in localStorage via `useTheme()` composable

### Style Conventions (Agent Rules)

1. **PrimeVue first** — use PrimeVue components for everything they provide (Button, DataTable, Dialog, Toast, Menu, Tag, Badge, InputText, etc.)
2. **Tailwind for layout** — flex, grid, gap, padding, margin. Override PrimeVue colors/sizes via design tokens, NOT Tailwind
3. **Zero custom CSS** except complex animations or SVG. No `<style scoped>` blocks
4. **No inline styles** — use PrimeVue severity props: `severity="danger"` not `:style="{ color: 'red' }"`

### Key Frontend Libraries

| Library | Usage |
|---------|-------|
| `@vue-flow/core` | DAG visualization |
| `@guolao/vue-monaco-editor` | YAML pipeline editor |
| `ansi-to-html` | Agent log rendering |
| `diff2html` | PR diff rendering |
| `vee-validate` + `zod` | Form validation |
| `@vueuse/core` | Utility composables (useLocalStorage, useEventSource, useDebounceFn) |
| `date-fns` | Date formatting (tree-shakeable) |

### Functional Patterns

- **Components are visual assemblers** — zero business logic in `.vue` files
- **Composables are the logic layer** — all reactive state management, API calls, event handling
- **`useAsyncAction`** pattern wraps every async operation (loading + error + execute)
- **Props down, events up** strictly enforced
- **Stores handle SSE dispatch** — `useSSE` composable dispatches events to relevant Pinia stores

### PrimeVue Component Mapping

| Project Need | PrimeVue Component |
|---|---|
| Story kanban | DataView + custom template |
| Run step timeline | Timeline |
| Agent logs | Custom LogViewer (ANSI support) |
| DAG visualization | Custom DagGraph (@vue-flow/core) |
| Pipeline YAML editor | Custom (Monaco editor) |
| Diff review | Custom DiffViewer (diff2html) |
| Toast notifications | Toast service |
| Forms | InputText, Select, Textarea, FloatLabel |
| Confirmation | ConfirmDialog service |
| Navigation | Menubar + PanelMenu (sidebar) |
| Status display | Tag with severity mapping |

---

## Testing Strategy — Detailed

### Backend Testing

**Unit Tests (Go `testing` + table-driven):**
- Hand-written mocks (no mockgen) — implements port interfaces directly
- Factories with options pattern: `factory.NewStory(WithEpic("E-01"), WithDeps("S-01"))`
- Table-driven tests for state machine transitions, DAG builder, action registry

**Per-Domain Testing:**

| Domain | Unit Test Focus | Integration Test Focus |
|--------|----------------|----------------------|
| PipelineService | State transitions, retry logic, action dispatch | River job enqueue/process with real Postgres |
| SchedulerService | DAG building, dependency resolution, cycle detection | Real stories with provides/requires in Postgres |
| ActionRegistry | Action lookup, param validation | N/A (tested via Pipeline integration) |
| Actions (agent_run, ci_poll, etc.) | Command building, output parsing, error handling | Docker container lifecycle (testcontainers) |
| GitProvider | Command construction, response parsing | gh CLI integration (optional, CI-only) |
| EventBus | N/A | LISTEN/NOTIFY with real Postgres |
| Auth | Token generation, validation, expiry | Full HTTP flow with JWT cookies |
| Repositories | N/A (generated by sqlc) | All queries against real Postgres |

**Testability Patterns:**
- `CommandRunner` interface in `pkg/exec/` — wraps `exec.Command`, mockable for GitProvider/Docker tests
- Separated parsing: command output → parse function (pure, unit-testable) → domain model
- SSE testing: httptest server + EventSource client, verify event stream format

**Integration Tests:**
- testcontainers-go: each test gets ephemeral Postgres with migrations applied
- Test helpers in `internal/testutil/`: `testdb.New()`, `factory.NewProject()`, `factory.NewRun()`
- Tagged with build tags or test name prefix for CI separation

**Coverage Target:** 85% initial → 90% with CommandRunner + separated parsing techniques

### Frontend Testing

**Unit Tests (Vitest):**

| Target | What to test | Coverage target |
|--------|-------------|----------------|
| Composables | Reactive logic, edge cases, error paths | 95%+ |
| Pinia stores | Actions, mutations, SSE event handlers | 90%+ |
| Utils | Formatters, parsers, validators (pure functions) | 100% |
| Zod schemas | Validation rules | 100% |

**Component tests:** ONLY for components with complex conditional logic. NOT for testing PrimeVue renders a button. Uses `@vue/test-utils` mount.

**Tests live co-located:** `__tests__/` directory next to the source file.

**E2E Tests (Playwright):**

Critical scenarios:
1. Auth: login → redirect → session expire → re-login
2. Project CRUD: create → configure pipeline → see settings
3. Story board: see kanban → drag status → story detail
4. Run lifecycle: launch → see SSE timeline → steps progress
5. HITL: notification → review diff → approve/reject → run continues
6. DAG: see graph → dependencies visible → launch epic run
7. Pipeline editor: open YAML → modify → save → validate
8. Logs: open step → streamed logs → ANSI colors → auto scroll
9. Dark mode: toggle → persisted → all components coherent
10. Responsive: sidebar collapse → mobile nav → table scroll

**E2E Backend:** `docker-compose.test.yml` with real backend + seeded Postgres.
Test-only endpoints `POST /api/v1/test/seed` and `POST /api/v1/test/reset` (enabled only when `APP_ENV=test`).

### CI Pipeline

```bash
# Backend CI
golangci-lint run ./...              # Lint
go test ./... -short                 # Unit tests (fast, no containers)
go test ./... -run Integration       # Integration tests (testcontainers)

# Frontend CI
npm run lint                         # ESLint
npm run type-check                   # tsc --noEmit
npm run test:unit                    # Vitest
npm run test:e2e                     # Playwright (against docker-compose.test.yml)

# Functional (nightly or pre-release)
docker compose -f deploy/docker-compose.test.yml up -d
# Launch story on test-project, verify full flow
```

---

## Implementation Patterns & Consistency Rules

### Naming Patterns

**Database:**
- Tables: `snake_case`, plural (`stories`, `run_steps`, `pipeline_configs`)
- Columns: `snake_case` (`created_at`, `project_id`, `retry_count`)
- Foreign keys: `{referenced_table_singular}_id` (`project_id`, `story_id`)
- Indexes: `idx_{table}_{columns}` (`idx_stories_project_id`, `idx_runs_status`)
- Constraints: `{table}_{type}_{columns}` (`runs_fk_story_id`, `stories_uq_key_project`)

**API:**
- Endpoints: plural nouns, kebab-case for multi-word (`/pipeline-configs`, `/run-steps`)
- Route params: `{id}` format (OpenAPI standard)
- Query params: `snake_case` (`project_id`, `per_page`, `sort_by`)
- JSON fields: `snake_case` (matches Go JSON tags and Postgres columns)
- Dates: ISO 8601 strings (`"2026-02-15T10:30:00Z"`)

**Go Code:**
- Files: `snake_case.go` (`pipeline_service.go`, `run_step.go`)
- Packages: single lowercase word where possible (`model`, `port`, `service`)
- Types: `PascalCase` (`PipelineService`, `RunStep`, `ActionResult`)
- Interfaces: descriptive noun (`GitProvider`, `Transactor`, `JobQueue`) — NOT `IGitProvider`
- Methods: `PascalCase` for exported, `camelCase` for private
- Variables: `camelCase` (`storyID`, `runStep`, `maxRetries`)

**Vue/TypeScript:**
- Components: `PascalCase.vue` (`RunTimeline.vue`, `StoryBoard.vue`)
- Composables: `use` prefix, `camelCase` (`useSSE.ts`, `useAsyncAction.ts`)
- Stores: domain noun (`auth.ts`, `runs.ts`, `stories.ts`)
- Utils: `camelCase.ts` (`formatDate.ts`, `parseNdjson.ts`)
- Types/interfaces: `PascalCase` (`Run`, `Story`, `SSEEvent`)

**Events (SSE / Postgres NOTIFY):**
- Format: `{entity}.{action}` dot-notation (`run.started`, `step.completed`, `hitl.pending`)
- Payload: JSON with snake_case fields

**Git:**
- Branches: `feat/S-{key}-{slug}`, `fix/S-{key}-{slug}`
- Commits: conventional commits (`feat(pipeline): add retry logic`)

### Structure Patterns

- Tests co-located in `__tests__/` directories (both Go and Vue)
- Go: `internal/` for private packages, `pkg/` for shared utilities
- Vue: shared UI → `ui/`, domain-specific → `features/`, shared logic → `composables/`
- Config files at project root or dedicated `config/` directories
- Generated code in gitignored directories or clearly marked (`wire_gen.go`, `db/` for sqlc)

### Process Patterns

**Loading States:**
- Every async operation uses `useAsyncAction` pattern (isLoading, error, data, execute)
- Stores expose `isLoading` per domain
- UI shows PrimeVue Skeleton or ProgressSpinner during load

**Error Recovery:**
- API errors → DomainError → HTTP status + error envelope → frontend error ref
- Toast for transient errors (network, 500)
- Inline error display for validation (400)
- Redirect to login on 401

**Enforcement:**
- Linters: golangci-lint (Go), ESLint (Vue/TS)
- Type checking: Go compiler + tsc --noEmit
- Generated code keeps implementations in sync with specs
- CLAUDE.md files per directory scope agent behavior

## Project Structure & Boundaries

### Requirements to Architecture Mapping

#### Backend-Only Stories (agent backend, CLAUDE.md backend)

| FR | Description | Package Target |
|---|---|---|
| FR1-FR2 | Project CRUD + config | `adapter/postgres/`, `api/handler/` |
| FR6-FR8 | Story parsing + markdown import | `domain/service/`, `adapter/postgres/` |
| FR9-FR18 | Pipeline execution (state machine, DAG, retry, pause/resume) | `domain/service/`, `adapter/river/`, `adapter/action/*` |
| FR19-FR22 | Pipeline config CRUD (YAML storage) | `adapter/postgres/`, `api/handler/` |
| FR23, FR26 | HITL backend logic (pause, approve/reject via API) | `adapter/action/hitl_gate`, `domain/service/` |
| FR27-FR32 | Agent & container lifecycle | `adapter/docker/`, `adapter/action/agent_run` |
| FR33-FR36 | Prompt rendering + template CRUD | `domain/service/PromptRenderer`, `api/handler/` |
| FR40 | Notifications Discord + webhook | `adapter/discord/`, `adapter/webhook/` |
| FR41-FR44 | Cost tracking + SQL aggregation | `adapter/postgres/` |
| FR45-FR48 | Auth JWT + user management | `api/middleware/auth`, `adapter/postgres/` |
| FR49-FR52 | Git operations (gh CLI) | `adapter/github/` |
| FR53 | Test project + seed | `internal/testutil/`, `test-project/` |

~40 FRs are purely backend.

#### Frontend-Only Stories (agent frontend, CLAUDE.md frontend)

| FR | Description | Feature / Composable Target |
|---|---|---|
| FR3 | Project list UI | `features/projects/ProjectList.vue` |
| FR4-FR5 | Stories by epic view + status display | `features/stories/StoryBoard.vue` |
| FR6 | Story detail UI | `features/stories/StoryDetail.vue` |
| FR9-FR10 | "Run Story" / "Run Epic" buttons + confirmation | `features/runs/` (UI trigger) |
| FR17-FR18 | Pause / Resume buttons | `features/runs/RunDetail.vue` |
| FR21 | Pipeline config view (read-only) | `features/pipeline-editor/` (read mode) |
| FR24-FR25 | Approve / Reject HITL via web UI | `features/approvals/ApprovalQueue.vue`, `DiffViewer.vue` |
| FR37, FR39 | Live SSE logs + real-time progress | `composables/useSSE`, `features/runs/StepOutput.vue` |
| FR42 | Cost breakdown display | `features/runs/` (cost panel) |
| FR45 | Login page + auth flow | `composables/useAuth`, `views/LoginView.vue` |

~15 FRs have a frontend component.

#### Coordinated Stories (API Contract Changes)

These FRs require an `api/openapi.yaml` change BEFORE frontend and backend can work:

| FR | Coordination Required | Spec Change |
|---|---|---|
| FR9-FR10 | `POST /runs` endpoint + response shape | OpenAPI: runs resource |
| FR23-FR25 | Approve/reject endpoint + approval list | OpenAPI: approvals resource |
| FR37 | SSE event types + payload format | OpenAPI: SSE event schemas |
| FR19-FR20 | Pipeline config CRUD endpoints | OpenAPI: pipeline-config resource |
| FR41-FR42 | Cost response format | OpenAPI: cost fields in run/step |

**Coordination Workflow:**
1. Spec story: Add endpoints to `openapi.yaml`
2. Regenerate: `oapi-codegen` (backend) + `openapi-typescript` (frontend)
3. Backend story: Implement handlers + services
4. Frontend story: Implement views + composables
   (Steps 3 and 4 can run in parallel once spec is merged)

#### Story Breakdown Summary

| Category | Estimated Stories | Agent |
|---|---|---|
| **Backend pure** | ~25-30 | Backend agent (Go) |
| **Frontend pure** | ~12-15 | Frontend agent (Vue) |
| **API spec** | ~5-8 | Manual or dedicated agent |
| **Infra / deploy** | ~3-5 | Manual |
| **E2E tests** | ~3-5 | Frontend agent (Playwright) |

### Architectural Boundaries

**4 strict boundaries:**

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

1. **Frontend ↔ Backend**: REST API + SSE. Contract = `api/openapi.yaml`. No direct coupling.
2. **Backend ↔ Postgres**: sqlc queries + pgxlisten + River jobs. Everything goes through ports.
3. **Backend ↔ Docker**: Docker API via socket-proxy. AgentRuntime port.
4. **Backend ↔ GitHub**: `gh` CLI via CommandRunner. GitProvider port.

### Data Flow — Story Run

```
User click "Run" → API handler → PipelineService.StartRun()
  → [tx] Create Run + RunStep + Enqueue River job
  → River worker picks job → ActionRegistry.Get("git_create_branch")
  → Action.Execute() → gh CLI → branch created
  → Worker enqueues next job → ActionRegistry.Get("agent_run")
  → DockerAdapter.Run() → container starts → NDJSON stream
  → Events → Postgres NOTIFY → pgxlisten → SSE handler → Browser
  → Step complete → next job → ... → hitl_gate → pause
  → User approve → API → PipelineService.Approve() → enqueue next
  → ... → merge → notify Discord + SSE → Run completed
```

### Cross-Cutting Data Flow

| Concern | Source | Transport | Consumers |
|---|---|---|---|
| Real-time updates | River worker / Actions | Postgres NOTIFY → pgxlisten | SSE handler → Frontend stores |
| Notifications | PipelineService events | EventBus → Notifier dispatcher | Discord webhook, SSE |
| Cost tracking | Agent NDJSON `cost` events | RunStep update in Postgres | Frontend cost display, SQL aggregation |
| Logging | All services via slog | stdout JSON | Docker logs / future LGTM stack |
| Auth context | JWT cookie | Middleware → context | All handlers, audit log |

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**
All technology choices are mutually compatible with no conflicts:
- pgx/v5 + sqlc + pgxlisten: same native Postgres driver throughout
- River: uses pgx/v5 natively, jobs in same DB enabling transactional enqueue
- oapi-codegen + chi: native compatibility (chi server interface generation)
- go-wire: compile-time DI, no runtime conflicts with other libraries
- openapi-fetch (frontend) consumes the same `openapi.yaml` as oapi-codegen: single contract
- PrimeVue 4 + Tailwind v4: CSS layers manage priority, no style conflicts

**Pattern Consistency:**
- Naming conventions consistent: snake_case in DB/API/JSON, PascalCase in Go types/Vue components
- Code-gen-first philosophy applied uniformly: oapi-codegen, sqlc, go-wire, openapi-typescript
- Hexagonal architecture enforced: all external dependencies behind port interfaces
- Error handling consistent: DomainError at service layer → HTTP mapping at middleware layer

**Structure Alignment:**
- Project structure directly supports hexagonal boundaries (domain/, adapter/, api/)
- Frontend hybrid structure (ui/ + features/) aligns with component reuse rules
- Test co-location pattern consistent across backend and frontend
- CLAUDE.md scoping per directory enables safe parallel agent work

### Requirements Coverage ✅

**Functional Requirements (53 FRs):**
All 53 FRs have architectural support. Coverage verified per domain:
- Project Management (FR1-FR5): ✅ CRUD + API + UI
- Story Management (FR6-FR8): ✅ Parser + storage + board UI
- Pipeline Execution (FR9-FR18): ✅ PipelineService + River + ActionRegistry + DAG
- Pipeline Configuration (FR19-FR22): ✅ YAML storage + CRUD + editor UI
- HITL (FR23-FR26): ✅ hitl_gate action + ApprovalQueue UI + Discord
- Agent & Container (FR27-FR32): ✅ DockerAdapter + NDJSON + timeout + circuit breaker
- Prompt Management (FR33-FR36): ✅ Handlebars renderer + CRUD API
- Real-Time Monitoring (FR37-FR40): ✅ pgxlisten → SSE → stores + Discord
- Cost & Observability (FR41-FR44): ✅ tracking + aggregation (enforcement deferred to Phase 2)
- Auth (FR45-FR48): ✅ JWT httpOnly + roles
- Git Operations (FR49-FR52): ✅ GitProvider via gh CLI
- Test Environment (FR53): ✅ test-project + docker-compose.test.yml

**Non-Functional Requirements (15 NFRs):**
- NFR1 (SSE < 1s): pgxlisten → SSE direct, no polling ✅
- NFR2 (API < 500ms): sqlc (no ORM overhead), chi lightweight ✅
- NFR3 (DAG < 2s for 50+ stories): in-memory topological sort ✅
- NFR4 (Container startup < 30s): pre-built Docker image ✅
- NFR5-8 (Security): socket-proxy, env-only secrets, slog scrub, JWT ✅
- NFR9-12 (Reliability): River retry, state in Postgres, orphan cleanup, circuit breaker ✅

### Gap Analysis

**Critical Gaps: 0** 🎯

**Important Gaps (documented, not blocking):**

1. **FR43-FR44 (Budget halt):** PRD specifies "halt execution when budget exceeded". Architecture decision: measurement and tracking only for MVP. Budget enforcement (automatic halt) deferred to Phase 2. This is a conscious scope decision, not a gap.

2. **FR38 (CLI logs):** "Users can view live agent logs via CLI" — no dedicated CLI client in architecture. For MVP, `curl` on the SSE endpoint suffices. Dedicated CLI client deferred to Phase 2.

3. **FR7 (Story import mechanism):** Story sync implemented via dedicated API endpoint:
   - Endpoint: `POST /api/v1/projects/{projectId}/stories/sync`
   - Reads markdown files with frontmatter from configurable repo path (default: `.hopeitworks/stories/*.md`)
   - Parses frontmatter (key, epic, depends_on, scope, status) and body content
   - Upserts stories in database (creates new, updates existing by key)
   - Returns sync result (created count, updated count, errors array)
   - Can be triggered manually via UI or via webhook (GitHub push event)

**Nice-to-Have Gaps (deferred):**
- Rate limiting (not needed at MVP scale)
- Infrastructure monitoring/alerting (slog + LGTM-ready, no built-in dashboards)
- Postgres backup strategy (ops responsibility, outside app architecture scope)

### Architecture Completeness Checklist

**✅ Requirements Analysis**
- [x] Project context thoroughly analyzed (53 FRs, 15 NFRs)
- [x] Scale and complexity assessed (medium-high)
- [x] Technical constraints identified (Go, Postgres, Docker, solo dev + AI agents)
- [x] Cross-cutting concerns mapped (7 concerns)

**✅ Architectural Decisions**
- [x] Critical decisions documented (pgx/v5, sqlc, River, go-wire, slog)
- [x] Technology stack fully specified with rationale for each choice
- [x] Integration patterns defined (hexagonal ports, event-driven pipeline)
- [x] Performance considerations addressed (all NFRs covered)

**✅ Implementation Patterns**
- [x] Naming conventions established (DB, API, Go, Vue/TS, events, git)
- [x] Structure patterns defined (hexagonal backend, hybrid frontend)
- [x] Communication patterns specified (SSE, Postgres NOTIFY, NDJSON)
- [x] Process patterns documented (error handling, loading states, auth flow)

**✅ Project Structure**
- [x] Complete directory structure defined for all modules
- [x] Component boundaries established (4 strict boundaries)
- [x] Integration points mapped (data flow diagrams)
- [x] Requirements to structure mapping complete (FR → package/feature)

**✅ Testing Strategy**
- [x] 4-level testing defined (unit, integration, E2E, pipeline functional)
- [x] Per-domain test focus documented
- [x] Coverage targets set (85-90% backend, 90-100% frontend logic)
- [x] CI pipeline defined

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** HIGH

**Key Strengths:**
- Code-gen-first eliminates drift between spec and implementation (OpenAPI, sqlc, go-wire)
- Hexagonal architecture ensures maximum testability via port interfaces
- River in Postgres means zero additional infrastructure
- Strict frontend/backend separation enables safe parallel agent work
- Composable action system allows fully configurable pipelines without code changes
- Event-driven pipeline (River) naturally handles HITL pauses without blocking

**Deferred to Phase 2:**
- Budget enforcement (FR43-FR44) — MVP only tracks cost, no automatic halt
- CLI client for log viewing (FR38) — MVP uses `curl` on SSE endpoint
- Visual pipeline editor (frontend)
- Rate limiting
- Agent builder (custom agent creation UI)

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all architectural decisions exactly as documented
- Use implementation patterns consistently across all components
- Respect project structure and boundaries (backend agents: `backend/` only, frontend agents: `frontend/` only)
- Refer to this document for all architectural questions
- Changes to `api/openapi.yaml` require coordination between frontend and backend stories

**Implementation Priority:**
1. `api/openapi.yaml` — API contract (single source of truth)
2. Database schema + migrations — data model foundation
3. sqlc queries + generated code — data access layer
4. oapi-codegen handlers — API skeleton
5. Core backend services (eventbus, pipeline, scheduler, River setup)
6. Frontend scaffolding (create-vue + PrimeVue + Tailwind + theme)
7. Frontend views consuming API + SSE
8. Agent runtime (Dockerfile, entrypoint, prompts)
9. Deploy stack (docker-compose)
10. Test infrastructure (all 4 levels)
