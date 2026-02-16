# Story 1.15: Go backend services wiring

Status: dev-done

## Story

As a backend developer,
I want the Go application wired with chi router, pgx connection pool, config loading, structured logging, health endpoint, and Wire DI,
so that I have a running backend service ready for feature development.

## Acceptance Criteria (BDD)

**AC1: Config loads from YAML with env override**
- **Given** config.yaml exists with development defaults
- **When** the API starts
- **Then** config is loaded from YAML and any `APP_*` / `DB_*` / `LOG_*` env vars override the YAML values

**AC2: Health endpoint responds**
- **Given** the API service is running
- **When** I send GET /healthz
- **Then** I receive HTTP 200 with `{"status":"ok"}`

**AC3: chi router serves requests with middleware chain**
- **Given** the API is running
- **When** I send any HTTP request
- **Then** the request passes through logging, CORS, and recovery middleware

**AC4: pgx pool connects to Postgres**
- **Given** Postgres is running and config is correct
- **When** the API starts
- **Then** a pgxpool.Pool is created and Ping() succeeds

**AC5: Graceful shutdown works**
- **Given** the API is running with active connections
- **When** SIGTERM or SIGINT is received
- **Then** the HTTP server stops accepting new connections, drains in-flight requests, closes the DB pool, and exits cleanly

## Tasks / Subtasks

- [x] Task 1 [BACK]: Config struct + YAML loader (AC: #1)
  - [x]Create `backend/pkg/config/config.go` — Config, ServerConfig, DatabaseConfig, LogConfig structs
  - [x]Create `backend/internal/config/loader.go` — `Load(path string) (*config.Config, error)` reads YAML, applies env overrides
  - [x]Create `backend/config.yaml` with development defaults
  - [x]Validate required fields (fail fast on missing db.host, db.name, db.user, db.password)

- [x] Task 2 [BACK]: slog JSON structured logging (AC: #3)
  - [x]Create `backend/pkg/log/logger.go` — `New(level string) *slog.Logger` with JSON handler on stdout
  - [x]Implement `ScrubHandler` wrapping slog.Handler to redact keys: password, token, secret, api_key, authorization
  - [x]Add `WithLogger(ctx, logger)` and `FromContext(ctx)` context helpers

- [x] Task 3 [BACK]: pgx/v5 connection pool (AC: #4)
  - [x]Create `backend/internal/adapter/postgres/pool.go` — `NewPool(ctx, cfg DatabaseConfig) (*pgxpool.Pool, error)`
  - [x]Build DSN from config: `postgres://user:pass@host:port/dbname?sslmode=X`
  - [x]Configure pool: max_conns, min_conns, max_conn_lifetime from config
  - [x]Ping with 5s timeout on creation

- [x] Task 4 [BACK]: chi router + middleware + /healthz handler (AC: #2, #3)
  - [x]Create `backend/internal/api/router.go` — `NewRouter(pool, logger) chi.Router`
  - [x]Wire middleware chain: chi/middleware.Recoverer, chi/middleware.RequestID, CORS, slog request logger
  - [x]Create `backend/internal/api/handler/health.go` — `HandleHealthz` returning `{"status":"ok"}`
  - [x]Mount GET /healthz

- [x] Task 5 [BACK]: Wire DI + main.go + graceful shutdown (AC: #1-5)
  - [x]Create `backend/cmd/api/providers.go` — Wire provider sets for config, logger, pool, router
  - [x]Create `backend/cmd/api/wire.go` — Wire injector function `InitializeApp() (*App, error)`
  - [x]Rewrite `backend/cmd/api/main.go` — load config, init logger, connect DB, build router, start HTTP server
  - [x]Implement graceful shutdown: SIGTERM/SIGINT → server.Shutdown(30s ctx) → pool.Close()

- [x] Task 6 [BACK]: Verify end-to-end (AC: #1-5)
  - [x]`docker compose -f deploy/docker-compose.yml up -d`
  - [x]`curl http://localhost:8080/healthz` → 200 `{"status":"ok"}`
  - [x]Verify JSON structured logs on stdout
  - [x]Verify env var override: `SERVER_PORT=9000` changes listen port
  - [x]Send SIGTERM, verify clean shutdown in logs

## Dev Notes

This story wires the actual Go backend services onto the project shell from Story 1.1. After this story, the backend is a running HTTP server connected to Postgres, ready to receive feature handlers.

### Dependencies on Story 1.1

Story 1.1 must be completed first. This story expects:
- Go module at `backend/` with `go.mod`
- `deploy/docker-compose.yml` with Postgres service
- `backend/Makefile` with build/run/docker targets
- Directory structure: `internal/adapter/postgres/`, `internal/api/handler/`, `internal/api/middleware/`, `internal/config/`, `pkg/log/`, `pkg/config/`, `cmd/api/`
- Minimal `cmd/api/main.go` (will be rewritten)

### Go Dependencies to Add

```bash
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/jackc/pgx/v5
go get gopkg.in/yaml.v3
go get github.com/google/wire
```

### File Structure

**Files to create or modify:**

1. **backend/pkg/config/config.go** — Config struct definitions
   ```go
   type Config struct {
       Server   ServerConfig   `yaml:"server"`
       Database DatabaseConfig `yaml:"database"`
       Log      LogConfig      `yaml:"logging"`
   }
   type ServerConfig struct {
       Port         int           `yaml:"port"`          // default: 8080
       ReadTimeout  time.Duration `yaml:"read_timeout"`  // default: 15s
       WriteTimeout time.Duration `yaml:"write_timeout"` // default: 15s
   }
   type DatabaseConfig struct {
       Host            string `yaml:"host"`
       Port            int    `yaml:"port"`
       Name            string `yaml:"name"`
       User            string `yaml:"user"`
       Password        string `yaml:"password"`
       SSLMode         string `yaml:"sslmode"`          // default: disable
       MaxConns        int32  `yaml:"max_conns"`        // default: 25
       MinConns        int32  `yaml:"min_conns"`        // default: 5
       MaxConnLifetime string `yaml:"max_conn_lifetime"` // default: 1h
   }
   type LogConfig struct {
       Level string `yaml:"level"` // debug, info, warn, error
   }
   ```

2. **backend/internal/config/loader.go** — Config loading
   - `Load(path string) (*config.Config, error)`
   - Read YAML file, unmarshal into Config struct
   - Override with env vars: `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`, `SERVER_PORT`, `LOG_LEVEL`
   - Validate required: host, name, user, password

3. **backend/pkg/log/logger.go** — slog utilities
   - `New(level string) *slog.Logger` — JSON handler on os.Stdout
   - `ScrubHandler` wraps any slog.Handler, redacts sensitive attrs
   - `WithLogger(ctx, *slog.Logger) context.Context`
   - `FromContext(ctx) *slog.Logger`

4. **backend/internal/adapter/postgres/pool.go** — pgx pool
   - `NewPool(ctx, cfg) (*pgxpool.Pool, error)`
   - DSN: `postgres://user:pass@host:port/name?sslmode=X`
   - Pool config from DatabaseConfig fields
   - Ping on creation with 5s timeout

5. **backend/internal/api/router.go** — chi router factory
   - `NewRouter(pool *pgxpool.Pool, logger *slog.Logger) chi.Router`
   - Middleware: Recoverer, RequestID, CORS (dev: allow all origins), slog request logging
   - Mount `/healthz`

6. **backend/internal/api/handler/health.go** — Health handler
   - `HandleHealthz(w, r)` → `{"status":"ok"}` HTTP 200

7. **backend/cmd/api/wire.go** — Wire injector
   - Provider sets: ConfigSet, LogSet, PostgresSet, RouterSet
   - `InitializeApp() (*App, error)` — generates `wire_gen.go`

8. **backend/cmd/api/main.go** — Entry point (REWRITE)
   - Call `InitializeApp()` or manual wiring if Wire not yet generated
   - `http.Server` with configured timeouts
   - Listen on `cfg.Server.Port`
   - Graceful shutdown on SIGTERM/SIGINT (30s drain)
   - Log: "server starting", "server listening", "shutting down"

9. **backend/config.yaml** — Development defaults
   ```yaml
   server:
     port: 8080
     read_timeout: 15s
     write_timeout: 15s

   database:
     host: localhost
     port: 5432
     name: hopeitworks_dev
     user: hopeitworks
     password: hopeitworks_dev_password
     sslmode: disable
     max_conns: 25
     min_conns: 5
     max_conn_lifetime: 1h

   logging:
     level: debug
   ```

### Wire Provider Sets

```go
// cmd/api/providers.go
var ConfigSet = wire.NewSet(config.Load)
var LogSet = wire.NewSet(log.New)
var PostgresSet = wire.NewSet(postgres.NewPool)
var RouterSet = wire.NewSet(api.NewRouter)
```

### Testing Requirements

**Manual verification checklist:**
1. `make build` succeeds
2. `make docker-up` starts Postgres + API
3. `curl http://localhost:8080/healthz` → `{"status":"ok"}`
4. Logs are JSON: `{"time":"...","level":"INFO","msg":"server listening","port":8080}`
5. `SERVER_PORT=9000 make run` → listens on 9000
6. `LOG_LEVEL=debug make run` → debug logs appear
7. Ctrl+C → logs show "shutting down gracefully"

### References

- [Source: architecture.md#Backend Architecture — Foundations — Logger: slog]
- [Source: architecture.md#Backend Architecture — Foundations — Dependency Injection: go-wire]
- [Source: architecture.md#Backend Architecture — Hexagonal Structure]
- [Source: architecture.md#Infrastructure & Deployment — Health Checks]
- [Source: architecture.md#Infrastructure & Deployment — Config Management]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- All unit tests pass: `go test ./...` — 10 tests across 4 packages
- `go vet ./...` — no issues
- `go build ./cmd/api` — compiles successfully

### Completion Notes List

- AC1: Config loads from YAML with env override — implemented in `internal/config/loader.go` with env vars: DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD, DB_SSLMODE, SERVER_PORT, LOG_LEVEL. Validated with unit tests.
- AC2: Health endpoint responds — `GET /healthz` returns HTTP 200 with `{"status":"ok"}`. Validated with unit test.
- AC3: chi router with middleware chain — Recoverer, RequestID, CORS, slog request logger. Validated with router tests.
- AC4: pgx pool connects to Postgres — `NewPool()` builds DSN from config, configures pool params, pings with 5s timeout.
- AC5: Graceful shutdown — SIGTERM/SIGINT triggers `server.Shutdown(30s)` then `pool.Close()` with structured log messages.
- Wire DI provider sets defined in `cmd/api/providers.go` with injector in `cmd/api/wire.go`. Main uses manual wiring directly for simplicity.
- Go version upgraded from 1.23 to 1.24 (required by pgx/v5 v5.8.0).
- Dockerfile updated to Go 1.24 builder and copies config.yaml to runtime image.
- ScrubHandler redacts sensitive log keys: password, token, secret, api_key, authorization.

### File List

- `backend/pkg/config/config.go` — Config, ServerConfig, DatabaseConfig, LogConfig structs
- `backend/internal/config/loader.go` — YAML loader with env overrides and validation
- `backend/internal/config/loader_test.go` — Unit tests for config loading
- `backend/config.yaml` — Development defaults
- `backend/pkg/log/logger.go` — slog JSON logger with ScrubHandler and context helpers
- `backend/pkg/log/logger_test.go` — Unit tests for logger and scrub handler
- `backend/internal/adapter/postgres/pool.go` — pgxpool.Pool creation with DSN and ping
- `backend/internal/api/router.go` — chi router factory with middleware chain
- `backend/internal/api/router_test.go` — Router integration tests
- `backend/internal/api/handler/health.go` — /healthz handler
- `backend/internal/api/handler/health_test.go` — Health handler unit test
- `backend/cmd/api/main.go` — Entry point with graceful shutdown (rewritten)
- `backend/cmd/api/providers.go` — Wire provider sets
- `backend/cmd/api/wire.go` — Wire injector definition
- `backend/Dockerfile` — Updated Go version and config.yaml copy
- `backend/go.mod` — Updated dependencies
- `backend/go.sum` — Updated dependency checksums

## Change Log

- 2026-02-16: Initial implementation of all acceptance criteria (AC1-AC5)
