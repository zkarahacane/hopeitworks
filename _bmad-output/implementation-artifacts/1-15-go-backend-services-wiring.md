# Story 1.15: Go Backend Services Wiring

Status: ready-for-dev

## Story

As a backend developer,
I want the Go application wired with configuration loading, structured logging, database connection, and HTTP server with health endpoints,
so that I have a running backend service ready for feature development.

## Acceptance Criteria (BDD)

**AC1: Configuration loads correctly**
- **Given** config.yaml contains settings
- **When** the API starts
- **Then** config is loaded from YAML with env var overrides and slog outputs JSON logs

**AC2: Health endpoint responds**
- **Given** the API service is running
- **When** I send GET /health
- **Then** I receive HTTP 200 with status ok

**AC3: Readiness endpoint checks database**
- **Given** the API service is running and Postgres is ready
- **When** I send GET /ready
- **Then** I receive HTTP 200 and readiness check pings database

**AC4: Logs are structured JSON**
- **Given** the API service is running
- **When** I check the application logs
- **Then** I see JSON formatted logs with timestamp, level, message, and context fields

**AC5: Complete stack runs end-to-end**
- **Given** docker-compose stack is up
- **When** I test all health endpoints and verify logs
- **Then** all checks pass and system is ready for feature development

## Tasks / Subtasks

- [ ] Task 2: Implement configuration loading system (AC: #1)
  - [ ] Create pkg/config/config.go with Config struct (server port, db connection, log level)
  - [ ] Create internal/config/loader.go to read config.yaml
  - [ ] Implement environment variable override logic (e.g., DB_HOST overrides yaml db.host)
  - [ ] Create backend/config.yaml with development defaults
  - [ ] Add validation for required config fields
  - [ ] Document config structure and environment variables in comments

- [ ] Task 3: Setup slog structured logging (AC: #4)
  - [ ] Create pkg/log/logger.go with slog JSON handler
  - [ ] Implement ScrubHandler wrapper to sanitize sensitive values (tokens, passwords)
  - [ ] Add WithLogger(ctx, logger) and LoggerFrom(ctx) helpers
  - [ ] Configure root logger in main.go with JSON output to stdout
  - [ ] Add log level configuration (debug, info, warn, error)
  - [ ] Test logging outputs valid JSON with structured fields

- [ ] Task 4: Setup Postgres connection with pgx/v5 (AC: #3)
  - [ ] Add pgx/v5 dependency: `go get github.com/jackc/pgx/v5`
  - [ ] Create internal/adapter/postgres/db.go with connection pool setup
  - [ ] Implement connection string building from config
  - [ ] Add connection timeout and pool size configuration
  - [ ] Create Ping() method for readiness checks
  - [ ] Add graceful shutdown for database pool

- [ ] Task 5: Implement health and readiness endpoints (AC: #2, #3)
  - [ ] Add chi router dependency: `go get github.com/go-chi/chi/v5`
  - [ ] Create internal/api/handler/health.go
  - [ ] Implement GET /health returning `{"status": "ok"}` (liveness)
  - [ ] Implement GET /ready with database ping check (readiness)
  - [ ] Return HTTP 200 if ready, HTTP 503 if not ready
  - [ ] Add basic request logging middleware

- [ ] Task 6: Wire up HTTP server in main.go (AC: #1-4)
  - [ ] Initialize config loader and logger
  - [ ] Initialize database connection pool
  - [ ] Setup chi router with health and ready endpoints
  - [ ] Configure HTTP server with timeouts and graceful shutdown
  - [ ] Listen on configured port (default: 8080)
  - [ ] Log startup information (port, environment, version)
  - [ ] Handle SIGTERM/SIGINT for graceful shutdown

- [ ] Task 11: Verify complete stack end-to-end (AC: #5)
  - [ ] Start stack: `docker compose -f deploy/docker-compose.yml up -d`
  - [ ] Wait for services to be healthy
  - [ ] Test GET /health returns 200
  - [ ] Test GET /ready returns 200 with database check
  - [ ] Verify logs show JSON structured output
  - [ ] Verify config loaded from yaml with env overrides
  - [ ] Stop stack gracefully
  - [ ] Document verification steps in README

## Dev Notes

This story wires the actual Go backend code: config loader, structured logger, DB connection, HTTP server, and health endpoints. It builds on the project shell created in Story 1.1.

### Dependencies on Story 1.1

**CRITICAL: Story 1.1 must be completed first. This story expects:**
- Go module exists at backend/ with go.mod
- docker-compose.yml exists at deploy/ with Postgres service configured
- Makefile exists at backend/ with build, docker-up, docker-down targets
- .env.example exists at backend/ with DB connection vars documented
- Project folder structure exists (internal/adapter/postgres/, internal/api/handler/, pkg/log/, pkg/config/, internal/config/)
- Minimal main.go exists at cmd/api/main.go (will be rewritten in this story)

### Architecture Requirements

**Services to Implement:**

1. **Configuration System** (pkg/config/ + internal/config/)
   - Config struct with nested sections (server, database, logging)
   - YAML file reader with viper or standard library
   - Environment variable override mechanism
   - Validation for required fields

2. **Structured Logging** (pkg/log/)
   - slog JSON handler for structured logs
   - ScrubHandler to sanitize passwords, tokens, secrets
   - Context helpers: WithLogger(ctx, logger), LoggerFrom(ctx)
   - Log levels: debug, info, warn, error

3. **Database Connection** (internal/adapter/postgres/)
   - pgxpool.Pool for connection pooling
   - Connection string builder from config
   - Ping() method for health checks
   - Graceful shutdown

4. **HTTP Server** (cmd/api/main.go + internal/api/handler/)
   - chi router for HTTP routing
   - Health endpoint: GET /health → {"status": "ok"}
   - Ready endpoint: GET /ready → {"status": "ready", "database": "connected"}
   - Request logging middleware
   - Graceful shutdown on SIGTERM/SIGINT

### Technical Specifications

**Dependencies (add these with `go get`):**
- `github.com/go-chi/chi/v5` - HTTP router (v5.1.0 or later)
- `github.com/jackc/pgx/v5` - Postgres driver (v5.5.5 or later)
- Standard library only for logging (log/slog - Go 1.21+)

**Configuration System:**
- Primary: config.yaml with development defaults
- Override: Environment variables take precedence
- Naming: Use UPPER_SNAKE_CASE for env vars, lowercase.dot.case for YAML
- Example: YAML `server.port: 8080` overridden by env `SERVER_PORT=9000`
- Required fields: server.port, db.host, db.port, db.name, db.user, db.password
- Optional fields: log.level (default: info), server.read_timeout (default: 15s), server.write_timeout (default: 15s)

**Logging Configuration:**
- Format: JSON (structured logging)
- Output: stdout
- Fields: timestamp, level, message, plus context fields (request_id, user_id, etc.)
- Levels: debug, info, warn, error
- ScrubHandler must wrap default handler to strip sensitive values
- Sensitive keys to scrub: password, token, secret, api_key, authorization

**Database Connection:**
- Use pgxpool for connection pooling
- Connection string format: `postgres://user:password@host:port/dbname?sslmode=disable`
- Pool config: max_conns (default: 25), min_conns (default: 5), max_conn_lifetime (default: 1h)
- Ping() with 5s timeout for readiness check
- Close pool on shutdown

**HTTP Server:**
- chi router for request routing
- Timeouts: read (15s), write (15s), idle (120s)
- Port: configurable via config (default: 8080)
- Graceful shutdown: 30s timeout for in-flight requests
- Request logging middleware logs: method, path, status, duration

### File Structure

**Files to create or modify in this story:**

1. **backend/pkg/config/config.go** - Config struct definitions
   - ServerConfig (port, read_timeout, write_timeout)
   - DatabaseConfig (host, port, name, user, password, sslmode, max_conns, min_conns)
   - LogConfig (level)
   - Config struct combining all sections

2. **backend/internal/config/loader.go** - Config loading logic
   - Load() function reads config.yaml
   - Override with environment variables
   - Validate required fields

3. **backend/pkg/log/logger.go** - Logging utilities
   - NewJSONLogger(level) returns slog.Logger
   - ScrubHandler wraps slog.Handler to sanitize sensitive fields
   - WithLogger(ctx, logger) adds logger to context
   - LoggerFrom(ctx) retrieves logger from context

4. **backend/internal/adapter/postgres/db.go** - Database connection
   - NewPool(config) returns *pgxpool.Pool
   - Ping() method for health checks
   - Close() for graceful shutdown

5. **backend/internal/api/handler/health.go** - Health endpoints
   - HandleHealth(w http.ResponseWriter, r *http.Request)
   - HandleReady(db *pgxpool.Pool) http.HandlerFunc

6. **backend/cmd/api/main.go** - Main entry point (REWRITE)
   - Load config
   - Initialize logger
   - Connect to database
   - Setup chi router with routes
   - Start HTTP server with graceful shutdown

7. **backend/config.yaml** - Development configuration
   - server.port: 8080
   - database.* (host, port, name, user, password, sslmode)
   - logging.level: debug

8. **backend/README.md** - Backend documentation (UPDATE)
   - Add setup instructions
   - Add development workflow
   - Add verification steps

### Testing Requirements

**Manual verification checklist (AC validation):**
1. Start stack: `make docker-up`
2. Check logs for JSON format: `make docker-logs | grep api`
3. Test health: `curl http://localhost:8080/health` → `{"status":"ok"}`
4. Test ready: `curl http://localhost:8080/ready` → `{"status":"ready","database":"connected"}`
5. Stop Postgres: `docker compose -f deploy/docker-compose.yml stop postgres`
6. Test ready fails: `curl http://localhost:8080/ready` → HTTP 503
7. Restart Postgres: `docker compose -f deploy/docker-compose.yml start postgres`
8. Test config override: `SERVER_PORT=9000 make run` → server listens on 9000
9. Test log level: `LOG_LEVEL=debug` → debug logs appear
10. Test graceful shutdown: Ctrl+C → logs show "shutting down gracefully"

**Expected log output examples:**
```json
{"time":"2026-02-16T10:00:00Z","level":"INFO","msg":"server starting","port":8080,"env":"development"}
{"time":"2026-02-16T10:00:01Z","level":"INFO","msg":"database connected","host":"postgres","database":"hopeitworks_dev"}
{"time":"2026-02-16T10:00:02Z","level":"INFO","msg":"server listening","port":8080}
```

### Configuration Best Practices

- Never commit .env files (use .env.example as template)
- All secrets via environment variables only
- YAML for structure, env vars for instance-specific overrides
- Document every config field in comments
- Validate required fields on startup (fail fast if missing)

### Code Examples

**Expected config.yaml structure:**
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

logging:
  level: debug
```

**Expected health endpoint response:**
```json
{"status": "ok"}
```

**Expected ready endpoint response (healthy):**
```json
{"status": "ready", "database": "connected"}
```

**Expected ready endpoint response (unhealthy):**
```json
{"status": "not ready", "database": "disconnected", "error": "connection refused"}
```

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#Backend Architecture — Hexagonal Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Logger: slog]
- [Source: _bmad-output/planning-artifacts/architecture.md#Database: PostgreSQL + pgx]
- [Source: _bmad-output/planning-artifacts/architecture.md#HTTP Router: chi]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 1: Project Foundation & Authentication]

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
