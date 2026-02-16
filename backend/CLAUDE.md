# Backend Agent Instructions — Go Patterns & Conventions

You are working exclusively in the `backend/` directory. **NEVER** modify files outside `backend/` except when coordinating API contract changes to `api/openapi.yaml`.

## Technology Stack

- **Go 1.23+**
- **chi** — HTTP router
- **pgx/v5** — Postgres driver (native, not database/sql)
- **sqlc** — SQL query code generation
- **oapi-codegen** — OpenAPI to Go server interfaces
- **go-wire (Google)** — compile-time dependency injection
- **golang-migrate** — SQL migrations
- **River** — Postgres-based job queue
- **pgxlisten** — Postgres LISTEN/NOTIFY wrapper
- **golang-jwt/jwt/v5** — JWT authentication
- **slog** — structured logging (Go stdlib)

## Hexagonal Architecture

### Package Layout

```text
backend/
├── cmd/api/
│   ├── main.go              # Entry point
│   └── wire.go              # DI wiring (go-wire provider sets)
├── internal/
│   ├── domain/
│   │   ├── model/           # Entities: Story, Run, RunStep, Project, Epic, Event, PipelineConfig, User
│   │   ├── port/            # Interfaces: GitProvider, AgentRuntime, Repository, Notifier, EventPublisher, Transactor, Action, JobQueue
│   │   └── service/         # Business logic: PipelineService, SchedulerService, ActionRegistry
│   ├── adapter/
│   │   ├── action/          # Action implementations: agent_run, ci_poll, git_branch, git_pr, git_merge, hitl_gate, notify, script
│   │   ├── github/          # GitProvider implementation (via gh CLI)
│   │   ├── docker/          # AgentRuntime implementation
│   │   ├── postgres/        # All Repository impls (sqlc generated) + Transactor + EventPublisher
│   │   ├── discord/         # Notifier implementation (webhook)
│   │   ├── webhook/         # Generic webhook Notifier
│   │   └── river/           # JobQueue implementation (River)
│   ├── api/
│   │   ├── handler/         # oapi-codegen generated handlers + SSE handler
│   │   └── middleware/      # Auth JWT, CORS, request logging, error mapping (DomainError -> HTTP)
│   ├── eventbus/            # pgxlisten wrapper: subscribe/publish on Postgres channels
│   ├── config/              # App config loading (YAML + env override)
│   └── testutil/            # Helpers: testcontainers setup, factories, assertions
├── pkg/
│   ├── log/                 # slog helpers: WithLogger, LoggerFrom, ScrubHandler
│   ├── errors/              # DomainError + categories + constructors
│   ├── exec/                # CommandRunner interface (for testable CLI calls)
│   └── config/              # Config struct definitions
├── migrations/              # SQL migrations (golang-migrate)
├── queries/                 # SQL queries (sqlc source)
├── testdata/                # Fixtures SQL + test story markdown files
├── sqlc.yaml                # sqlc configuration
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

### Boundary Rules

- **Services depend on ports (interfaces), never on adapters**
- **Adapters implement ports** — each adapter is a concrete implementation of a port interface
- **No business logic in handlers or adapters** — handlers validate/parse HTTP, adapters translate external calls
- **Domain model has zero external dependencies** — no imports from adapter or api packages
- Import direction: `handler → service → port ← adapter`

## Chi Router Patterns

### Route Registration

```go
r := chi.NewRouter()

// Middleware order matters: RequestID → Logger → CORS → Auth
r.Use(middleware.RequestID)
r.Use(middleware.Logger)
r.Use(middleware.CORS)
r.Use(middleware.Auth)

r.Route("/api/v1/projects", func(r chi.Router) {
    r.Get("/", handler.ListProjects)
    r.Post("/", handler.CreateProject)
    r.Route("/{id}", func(r chi.Router) {
        r.Get("/", handler.GetProject)
        r.Put("/", handler.UpdateProject)
        r.Delete("/", handler.DeleteProject)
    })
})
```

### Middleware Order

1. `RequestID` — assigns unique request ID
2. `Logger` — structured request logging via slog
3. `CORS` — cross-origin handling
4. `Auth` — JWT extraction and validation

### URL Parameters

```go
id := chi.URLParam(r, "id")
```

## sqlc Conventions

### Configuration

`sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries/"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "internal/adapter/postgres/db"
        sql_package: "pgx/v5"
```

### Query Files

Queries in `backend/queries/*.sql`:

```sql
-- name: GetStoryByKey :one
SELECT * FROM stories WHERE project_id = $1 AND key = $2 LIMIT 1;

-- name: ListStoriesByStatus :many
SELECT * FROM stories WHERE project_id = $1 AND status = ANY($2::text[]);

-- name: CreateStory :one
INSERT INTO stories (project_id, key, title, status, epic_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateStoryStatus :exec
UPDATE stories SET status = $1, updated_at = now() WHERE id = $2;
```

### Generated Code Usage

```go
story, err := q.GetStoryByKey(ctx, db.GetStoryByKeyParams{
    ProjectID: projectID,
    Key:       storyKey,
})
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, domainerrors.NewNotFound("story", storyKey)
    }
    return nil, domainerrors.NewInternal("query story", err)
}
```

### Regeneration

```bash
cd backend && sqlc generate
```

## DomainError Pattern

### Error Categories

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
```

### Error Struct

```go
type DomainError struct {
    Category ErrorCategory
    Code     string  // e.g., "STORY_NOT_FOUND"
    Message  string
    Cause    error
}
```

### Constructors

```go
import "github.com/zakari/hopeitworks/backend/pkg/errors"

// In service layer
if story == nil {
    return nil, errors.NewNotFound("story", storyKey)
}
if !valid {
    return nil, errors.NewValidation("field", "reason")
}
if exists {
    return nil, errors.NewConflict("story", storyKey)
}
```

### HTTP Mapping

API middleware maps `DomainError.Category` to HTTP status codes:

| Category | HTTP Status |
|----------|-------------|
| `not_found` | 404 |
| `validation` | 400 |
| `conflict` | 409 |
| `unauthorized` | 401 |
| `forbidden` | 403 |
| `internal` | 500 |

Services return `DomainError`. Adapters wrap external errors into `DomainError`.

## slog Structured Logging

### Setup

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

### Context Enrichment

```go
import "github.com/zakari/hopeitworks/backend/pkg/log"

// Inject logger into context
ctx = log.WithLogger(ctx, logger)

// Retrieve logger from context
log.LoggerFrom(ctx).Info("processing run",
    "run_id", runID,
    "story_key", story.Key,
)
```

### Structured Fields

Always include relevant context:

- `request_id` — from middleware
- `user_id` — from auth context
- `project_id` — from request scope
- `run_id` — from pipeline context
- `step_id` — from action context

### ScrubHandler

Sensitive values are automatically scrubbed by `ScrubHandler` wrapping the JSON handler. Fields containing `token`, `secret`, `password`, `key`, `authorization` are replaced with `[REDACTED]`.

### Rules

- Use `slog.Info` for normal operations
- Use `slog.Warn` for recoverable issues
- Use `slog.Error` for failures requiring attention
- Never use `fmt.Println` or `log.Println` — always use slog
- JSON output on stdout — LGTM-compatible for future Grafana integration

## Testing Patterns

### Unit Tests (Table-Driven)

```go
func TestBuildDAG(t *testing.T) {
    tests := []struct {
        name    string
        stories []model.Story
        wantDAG model.DAG
        wantErr bool
    }{
        {
            name:    "linear dependency chain",
            stories: []model.Story{...},
            wantDAG: model.DAG{...},
        },
        {
            name:    "cycle detected",
            stories: []model.Story{...},
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := scheduler.BuildDAG(tt.stories)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.wantDAG, got)
        })
    }
}
```

### Integration Tests (testcontainers)

```go
func TestRepositoryWithDB(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    db := testutil.NewTestDB(t) // spins up ephemeral Postgres container
    defer db.Close()

    repo := postgres.NewStoryRepository(db.Pool)
    // test against real DB with migrations applied
}
```

Each integration test gets its own ephemeral Postgres instance via testcontainers-go. No shared test database.

### Factories (Options Pattern)

```go
story := testutil.NewStory(
    testutil.WithKey("S-01"),
    testutil.WithDeps("S-02", "S-03"),
    testutil.WithEpic("E-01"),
    testutil.WithStatus("backlog"),
)

project := testutil.NewProject(
    testutil.WithName("test-project"),
)

run := testutil.NewRun(
    testutil.WithStory(story),
    testutil.WithStatus("running"),
)
```

Use factories over static fixtures — they are readable, composable, and maintainable.

### Mocks

Hand-written mocks implementing port interfaces directly. No mockgen.

```go
type MockGitProvider struct {
    CreateBranchFn func(ctx context.Context, repo, base, branch string) error
    // ...
}

func (m *MockGitProvider) CreateBranch(ctx context.Context, repo, base, branch string) error {
    return m.CreateBranchFn(ctx, repo, base, branch)
}
```

**Mock lint rules (golangci-lint revive):**
- Rename unused parameters to `_` — e.g., `func (m *mock) GetByID(_ context.Context, id uuid.UUID)`
- If a parameter is passed to a callback field (like `createFn`), it IS used — keep the name
- Always check `errcheck`: use `_ = json.NewEncoder(w).Encode(...)` when ignoring return values

### Test Commands

```bash
# Unit tests only (fast, no containers)
go test ./... -short

# All tests including integration
go test ./...

# Integration tests only
go test ./... -run Integration

# With verbose output
go test ./... -v

# Lint — MUST pass before committing (enforced in CI)
golangci-lint run ./...
```

**IMPORTANT:** Always run `golangci-lint run ./...` before committing Go code. CI will reject PRs with lint errors. Configuration is in `backend/.golangci.yml`.

## go-wire Dependency Injection

### Provider Sets

```go
var ServiceSet = wire.NewSet(
    service.NewPipelineService,
    service.NewSchedulerService,
    service.NewActionRegistry,
)

var AdapterSet = wire.NewSet(
    postgres.NewStoryRepository,
    postgres.NewRunRepository,
    github.NewGitProvider,
    docker.NewAgentRuntime,
)
```

### Wire File

`cmd/api/wire.go`:

```go
//go:build wireinject
// +build wireinject

func InitializeApp(cfg config.Config) (*App, error) {
    wire.Build(ServiceSet, AdapterSet, HandlerSet, NewApp)
    return nil, nil
}
```

### Generation

```bash
cd backend && wire ./cmd/api/
```

This generates `wire_gen.go` — never edit this file manually.

## pgx/v5 Transaction Management

### Transactor Pattern

```go
type Transactor interface {
    WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### Implementation

- Postgres adapter injects `pgx.Tx` into context
- Repository methods extract tx from context, fall back to pool if no tx present

### Usage in Services

```go
err := transactor.WithinTransaction(ctx, func(ctx context.Context) error {
    // All repo calls within this closure share the same transaction
    if err := storyRepo.UpdateStatus(ctx, storyID, "running"); err != nil {
        return err
    }
    if err := runRepo.Create(ctx, run); err != nil {
        return err
    }
    // River job enqueued in same transaction
    return jobQueue.EnqueueTx(ctx, tx, job)
})
```

## oapi-codegen (OpenAPI Code Generation)

### Workflow

1. Update `api/openapi.yaml` (the single source of truth)
2. Regenerate Go server interfaces: `cd backend && make generate`
3. Implement the generated interface methods in handler package

### Generated Output

oapi-codegen produces:

- Server interface (chi-compatible)
- Request/response types
- Parameter types
- Validation helpers

### Handler Implementation

```go
// Implement the generated interface
type StoryHandler struct {
    service *service.StoryService
}

func (h *StoryHandler) GetStory(w http.ResponseWriter, r *http.Request, id string) {
    story, err := h.service.GetByID(r.Context(), id)
    if err != nil {
        // Error middleware handles DomainError mapping
        renderError(w, err)
        return
    }
    renderJSON(w, http.StatusOK, story)
}
```

## Go Naming Conventions

- Files: `snake_case.go` (`pipeline_service.go`, `run_step.go`)
- Packages: single lowercase word where possible (`model`, `port`, `service`)
- Types: `PascalCase` (`PipelineService`, `RunStep`, `ActionResult`)
- Interfaces: descriptive noun (`GitProvider`, `Transactor`, `JobQueue`) — NOT `IGitProvider`
- Methods: `PascalCase` for exported, `camelCase` for private
- Variables: `camelCase` (`storyID`, `runStep`, `maxRetries`)
- Constants: `PascalCase` for exported, `camelCase` for private
- Errors: `ErrXxx` pattern for sentinel errors (`ErrNotFound`, `ErrUnauthorized`)

## Go Module

```text
module github.com/zakari/hopeitworks/backend
```

## Build & Run

```bash
# Build
cd backend && make build

# Run locally
cd backend && make run

# Docker compose (dev stack)
cd deploy && docker compose up -d

# View logs
cd deploy && docker compose logs -f api

# Stop
cd deploy && docker compose down
```

## Config Management

- Single `config.yaml` read at boot
- Environment variables override any YAML value
- Resolved into typed Go config struct at startup via `internal/config/`
- `.env.example` documents all available configuration variables
- No hot-reload for MVP — restart to apply changes

## Migration Management

```bash
# Create new migration
migrate create -ext sql -dir backend/migrations -seq <name>

# Run migrations
migrate -database $DATABASE_URL -path backend/migrations up

# Rollback last migration
migrate -database $DATABASE_URL -path backend/migrations down 1
```

Migrations are numbered sequentially: `000001_init_schema.up.sql` / `000001_init_schema.down.sql`
