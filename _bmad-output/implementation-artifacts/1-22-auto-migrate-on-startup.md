# Story 1.22: Auto-run database migrations on backend startup [BACK]

Status: ready-for-dev

## Story

As a developer,
I want the backend to automatically apply pending database migrations on startup,
so that I never have to run `make seed` or `migrate up` manually after pulling new code.

## Context

Currently, database migrations are applied manually via the `migrate` CLI tool (`make seed` or `migrate -path migrations/ -database $DATABASE_URL up`). This creates friction during development — forgetting to migrate after a `git pull` leads to cryptic runtime errors. The `golang-migrate` library supports programmatic usage with `pgx/v5`, so we can embed migration execution directly in the startup sequence.

Auto-migration is standard practice for development and MVP stages. The migration runs **before** any service initialization, ensuring the schema is always up-to-date when the app starts serving traffic.

A config flag `database.auto_migrate` (default: `true`) allows disabling auto-migration in environments where migrations are managed externally (e.g., CI pipelines, production with controlled rollouts).

## Acceptance Criteria (BDD)

**AC1: Migrations run automatically on startup**
- **Given** the backend starts with `database.auto_migrate: true` (default)
- **When** there are pending migrations in `backend/migrations/`
- **Then** all pending migrations are applied before the server starts listening
- **And** a log line confirms how many migrations were applied (e.g., `"migrations applied", "count", 3`)

**AC2: No-op when schema is up-to-date**
- **Given** the backend starts and all migrations have already been applied
- **When** the migration step runs
- **Then** no migrations are applied
- **And** a log line confirms the schema is current (e.g., `"database schema up to date"`)

**AC3: Startup fails on migration error**
- **Given** a migration file contains invalid SQL
- **When** the backend starts
- **Then** the process exits with a non-zero code and a clear error message indicating which migration failed

**AC4: Auto-migrate can be disabled via config**
- **Given** the config sets `database.auto_migrate: false`
- **When** the backend starts
- **Then** no migrations are executed
- **And** a log line confirms auto-migration is disabled

**AC5: Existing `make seed` and manual migrate still work**
- **Given** the new auto-migrate feature is in place
- **When** I run `make seed` or `migrate ... up` manually
- **Then** they still work as before (no conflict with the embedded migrator)

## Tasks / Subtasks

- [ ] Task 1: Add `golang-migrate` as a Go dependency (AC: #1)
  - [ ] `go get github.com/golang-migrate/migrate/v4`
  - [ ] `go get github.com/golang-migrate/migrate/v4/database/pgx/v5`
  - [ ] `go get github.com/golang-migrate/migrate/v4/source/iofs` (embed migrations)

- [ ] Task 2: Embed migration files and create migrator function (AC: #1, #2, #3)
  - [ ] Create `backend/internal/adapter/postgres/migrator.go`
  - [ ] Use `embed.FS` to embed `backend/migrations/*.sql`
  - [ ] Implement `RunMigrations(ctx, pool, logger) error` that:
    - Opens a `golang-migrate` instance with the iofs source and pgx driver
    - Calls `m.Up()` and handles `migrate.ErrNoChange` gracefully
    - Logs the result (applied count or "up to date")
    - Returns wrapped error on failure

- [ ] Task 3: Add config flag `database.auto_migrate` (AC: #4)
  - [ ] Add `AutoMigrate bool` field to `DatabaseConfig` struct (default: `true`)
  - [ ] Update `config.yaml` with `auto_migrate: true` under `database:`

- [ ] Task 4: Wire auto-migration into startup sequence (AC: #1, #2, #3, #4)
  - [ ] In `main.go`, after DB pool creation and before service initialization:
    - Check `cfg.Database.AutoMigrate`
    - If true, call `RunMigrations()`
    - If false, log skip message
  - [ ] On error, return immediately (prevents app from starting with stale schema)

- [ ] Task 5: Verify (AC: #1–#5)
  - [ ] Start backend fresh → migrations applied automatically
  - [ ] Restart backend → "up to date" logged
  - [ ] Set `auto_migrate: false` → migrations skipped
  - [ ] `make seed` still works alongside auto-migrate
  - [ ] Lint passes: `golangci-lint run ./...`

## Dev Notes

### File Paths

| File | Action |
|------|--------|
| `backend/internal/adapter/postgres/migrator.go` | CREATE |
| `backend/cmd/api/main.go` | MODIFY — add migration call after pool creation |
| `backend/internal/config/config.go` | MODIFY — add `AutoMigrate` field |
| `backend/config.yaml` | MODIFY — add `auto_migrate: true` |
| `backend/go.mod` | MODIFY — new dependency |

### Technical Specifications

```go
// backend/internal/adapter/postgres/migrator.go
package postgres

import (
    "embed"
    "fmt"
    "log/slog"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
// NOTE: embed directive must reference the migrations dir relative to this file.
// May need to use a separate package or pass FS from main.

func RunMigrations(dsn string, logger *slog.Logger) error {
    // Use iofs source with embedded migrations
    // Use pgx/v5 database driver
    // Call m.Up(), handle ErrNoChange
    // Log result
}
```

```go
// In main.go run(), after pool creation:
if cfg.Database.AutoMigrate {
    logger.Info("running database migrations")
    if err := pgadapter.RunMigrations(dsn, logger); err != nil {
        return fmt.Errorf("running migrations: %w", err)
    }
} else {
    logger.Info("auto-migration disabled, skipping")
}
```

### Architecture Notes

- `embed.FS` requires the `//go:embed` directive to be in a package that can see the `migrations/` directory. Since `migrations/` is at `backend/migrations/` and the migrator is at `backend/internal/adapter/postgres/`, the embed FS should be declared in a package closer to root (e.g., `backend/cmd/api/` or a dedicated `backend/migrations/` Go package) and passed to the migrator function. The dev agent should determine the cleanest approach.
- The `golang-migrate` pgx/v5 driver accepts a DSN string — reuse the same DSN constructed for the event bus.
- `migrate.ErrNoChange` is not an error — it means all migrations are already applied.
- The `schema_migrations` table created by `golang-migrate` is the same whether using the CLI or the library — no conflict.
- Integration tests in `testutil/testdb.go` use their own migration runner (manual SQL exec) — that can stay as-is for now.

### Dependencies

- None — this is a standalone improvement to the startup sequence.

### Testing Requirements

```bash
# Lint
cd backend && golangci-lint run ./...

# Start backend and check logs for migration output
cd deploy && docker compose up -d api
docker compose logs api | grep -i migrat
```

## Dev Agent Record

### Implementation Plan
### Completion Notes
## File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | zakari | Initial story created |
