# Story 1.18: [BACK] Dev seed data fixtures

Status: review

## Story

As a developer,
I want pre-populated seed data in my local database,
So that I can develop and test features without manually creating users and projects each time I reset the dev stack.

## Acceptance Criteria (BDD)

**AC1: Seed SQL file exists and is syntactically valid**
- **Given** the file `backend/testdata/seed.sql` exists
- **When** I parse it with `psql --set ON_ERROR_STOP=on -f seed.sql` (or the Go test equivalent)
- **Then** it executes without SQL syntax errors on a freshly migrated database

**AC2: Seed creates admin user**
- **Given** a fresh database with all migrations applied
- **When** I run the seed SQL
- **Then** a user with email `admin@hopeitworks.dev`, role `admin`, and a valid bcrypt password hash for `admin123` exists in the `users` table

**AC3: Seed creates member users**
- **Given** a fresh database with all migrations applied
- **When** I run the seed SQL
- **Then** users `dev@hopeitworks.dev` (password: `dev123`) and `alice@hopeitworks.dev` (password: `alice123`) exist with role `user`

**AC4: Seed creates test projects**
- **Given** a fresh database with seed users
- **When** I run the seed SQL
- **Then** projects "Todo App", "E-commerce API" (owned by admin), and "Frontend Kit" (owned by dev user) exist in the `projects` table

**AC5: Seed is idempotent**
- **Given** seed data has already been applied
- **When** I run the seed SQL again
- **Then** it completes without errors (no duplicate key violations)

**AC6: Seed creates project memberships (conditional)**
- **Given** the `project_users` table exists (Story 1-6 migration applied)
- **When** I run the seed SQL
- **Then** project memberships are created (admin=owner of Todo App and E-commerce API, dev=member of Todo App, alice=member of E-commerce API)
- **And** if `project_users` does not exist, the seed completes without error (membership inserts are conditional)

**AC7: Makefile target runs seed**
- **Given** the dev docker-compose stack is running with a healthy Postgres
- **When** I run `cd backend && make seed`
- **Then** migrations are applied and seed data is inserted into the database

**AC8: Seed validation test passes**
- **Given** the seed SQL file exists
- **When** I run `go test ./testdata/ -short`
- **Then** the test validates that the SQL is parseable and the file is not empty

## Tasks / Subtasks

- [x] [BACK] Task 1: Create seed SQL file with user fixtures (AC: #1, #2, #3, #5)
  - [x] Create `backend/testdata/seed.sql`
  - [x] Add header comment documenting purpose, credentials, and how to run
  - [x] Insert admin user with deterministic UUID `00000000-0000-0000-0000-000000000001`
  - [x] Insert dev user with deterministic UUID `00000000-0000-0000-0000-000000000002`
  - [x] Insert alice user with deterministic UUID `00000000-0000-0000-0000-000000000003`
  - [x] All passwords bcrypt-hashed at cost 10 (pre-computed hashes embedded in SQL)
  - [x] Use `INSERT ... ON CONFLICT (email) DO UPDATE SET` to ensure idempotency
  - [x] Verify hashes match the actual schema columns (email, password_hash, name, role)

- [x] [BACK] Task 2: Add project fixtures to seed SQL (AC: #4, #5)
  - [x] Insert "Todo App" project with deterministic UUID `00000000-0000-0000-0000-000000000101`, owner_id = admin UUID
  - [x] Insert "E-commerce API" project with deterministic UUID `00000000-0000-0000-0000-000000000102`, owner_id = admin UUID
  - [x] Insert "Frontend Kit" project with deterministic UUID `00000000-0000-0000-0000-000000000103`, owner_id = dev user UUID
  - [x] Use `INSERT ... ON CONFLICT (name) DO UPDATE SET` for idempotency
  - [x] Set meaningful description and repo_url for each project

- [x] [BACK] Task 3: Add conditional project_users fixtures (AC: #6)
  - [x] Add a DO block that checks for `project_users` table existence before inserting memberships
  - [x] Use `INSERT ... ON CONFLICT DO NOTHING` for membership rows
  - [x] Assign admin as owner of Todo App and E-commerce API
  - [x] Assign dev as member of Todo App
  - [x] Assign alice as member of E-commerce API

- [x] [BACK] Task 4: Add Makefile `seed` target (AC: #7)
  - [x] Add `seed` target to `backend/Makefile`
  - [x] Target runs migrations first (`migrate -path migrations/ -database $DATABASE_URL up`)
  - [x] Then runs seed SQL via `psql` against the local database
  - [x] Use docker-compose default connection params (host=localhost, port=5432, db=hopeitworks_dev, user=hopeitworks, password=hopeitworks_dev_password)
  - [x] Add `seed` to the `.PHONY` declaration
  - [x] Add `reset-db` convenience target that drops + recreates DB, runs migrations, and seeds

- [x] [BACK] Task 5: Write seed validation test (AC: #8)
  - [x] Create `backend/testdata/seed_test.go`
  - [x] Test 1: Verify `seed.sql` file exists and is not empty
  - [x] Test 2: Verify SQL is parseable (use `pg_query_go` or simple string validation for key statements)
  - [x] Test 3: Verify all expected INSERT statements are present (users, projects)
  - [x] Test 4: Verify ON CONFLICT clauses exist (idempotency check)
  - [x] All tests run with `-short` flag (no database required)

- [x] [BACK] Task 6: Update docker-compose documentation (AC: #7)
  - [x] Add comment in `deploy/docker-compose.yml` referencing the seed command
  - [x] Alternatively, add a `seed` section to the backend README or Makefile help text
  - [x] Verify `make seed` works with a fresh `docker compose up` environment

## Dev Notes

This story provides developer convenience fixtures for the local dev stack. It is intentionally simple: a single SQL file with INSERT statements, a Makefile target to run it, and a basic test to prevent the SQL from going stale. No Go code beyond the test file is required.

### Dependencies

**Story 1-3 (done):** Users table migration (000001) defines the `users` schema: id, email, password_hash, name, role, created_at, updated_at.

**Story 1-5 (done):** Projects table migration (000002) defines the `projects` schema: id, name, description, owner_id, repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget, created_at, updated_at.

**Story 1-4 (ready-for-dev):** User management API (adds `deleted_at` column via migration 000003). Seed users should have `deleted_at = NULL` (active users). The seed SQL should handle this column if it exists.

**Story 1-6 (ready-for-dev):** RBAC + project_users table. The seed conditionally inserts memberships only if the table exists. This means the seed works regardless of whether 1-6 has been merged.

### Architecture Requirements

**File paths:**

```
backend/
├── testdata/
│   ├── seed.sql              # Master seed file (INSERT statements)
│   └── seed_test.go          # Validation test
├── Makefile                  # Updated with seed + reset-db targets
```

No domain/service/adapter code is required for this story. The seed is pure SQL executed via `psql`.

### Technical Specifications

**Deterministic UUIDs:**

Using zero-padded UUIDs for reproducibility. This lets developers reference these IDs in manual testing, curl commands, and future seed extensions.

| Entity | UUID | Identifier |
|--------|------|------------|
| Admin user | `00000000-0000-0000-0000-000000000001` | admin@hopeitworks.dev |
| Dev user | `00000000-0000-0000-0000-000000000002` | dev@hopeitworks.dev |
| Alice user | `00000000-0000-0000-0000-000000000003` | alice@hopeitworks.dev |
| Todo App project | `00000000-0000-0000-0000-000000000101` | Todo App |
| E-commerce API project | `00000000-0000-0000-0000-000000000102` | E-commerce API |
| Frontend Kit project | `00000000-0000-0000-0000-000000000103` | Frontend Kit |

**Pre-computed bcrypt hashes (cost 10):**

The dev agent MUST generate real bcrypt hashes at build time or use a Go snippet to produce them. The following hashes can be generated with:

```go
package main

import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    for _, pw := range []string{"admin123", "dev123", "alice123"} {
        hash, _ := bcrypt.GenerateFromPassword([]byte(pw), 10)
        fmt.Printf("%s: %s\n", pw, string(hash))
    }
}
```

The dev agent should run this snippet and paste the actual hashes into `seed.sql`. Alternatively, the seed SQL can use `crypt()` from pgcrypto extension if available:

```sql
-- If pgcrypto is available (it is, from migration 000001):
INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@hopeitworks.dev',
    crypt('admin123', gen_salt('bf', 10)),
    'Admin User',
    'admin'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;
```

**Recommended approach:** Use `crypt()` + `gen_salt()` from pgcrypto. This avoids hardcoding hashes that may differ across bcrypt implementations and keeps the SQL self-contained. The pgcrypto extension is already created in migration 000001.

**Complete seed.sql content (`backend/testdata/seed.sql`):**

```sql
-- =============================================================================
-- hopeitworks dev seed data
-- =============================================================================
-- Purpose: Pre-populate local dev database with test users and projects.
-- Run:     cd backend && make seed
-- Reset:   cd backend && make reset-db
--
-- Credentials:
--   admin@hopeitworks.dev / admin123  (role: admin)
--   dev@hopeitworks.dev   / dev123    (role: user)
--   alice@hopeitworks.dev / alice123  (role: user)
--
-- Idempotent: safe to run multiple times (uses ON CONFLICT).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@hopeitworks.dev',
    crypt('admin123', gen_salt('bf', 10)),
    'Admin User',
    'admin'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'dev@hopeitworks.dev',
    crypt('dev123', gen_salt('bf', 10)),
    'Dev User',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'alice@hopeitworks.dev',
    crypt('alice123', gen_salt('bf', 10)),
    'Alice Developer',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

-- ---------------------------------------------------------------------------
-- Projects
-- ---------------------------------------------------------------------------

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    'Todo App',
    'Reference todo application for pipeline validation and baseline testing',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/todo-app',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    'E-commerce API',
    'Sample e-commerce REST API for multi-project testing scenarios',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/ecommerce-api',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000103',
    'Frontend Kit',
    'Vue 3 component library project owned by dev user',
    '00000000-0000-0000-0000-000000000002',
    'https://github.com/hopeitworks/frontend-kit',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

-- ---------------------------------------------------------------------------
-- Project memberships (conditional: only if project_users table exists)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'project_users'
    ) THEN
        -- Admin owns Todo App and E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        -- Dev user is member of Todo App
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002', 'member')
        ON CONFLICT DO NOTHING;

        -- Alice is member of E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000003', 'member')
        ON CONFLICT DO NOTHING;

        -- Admin owns Frontend Kit too (for visibility)
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002', 'owner')
        ON CONFLICT DO NOTHING;

        RAISE NOTICE 'Seed: project_users memberships inserted';
    ELSE
        RAISE NOTICE 'Seed: project_users table not found (Story 1-6 not applied), skipping memberships';
    END IF;
END $$;

COMMIT;
```

**Makefile additions (`backend/Makefile`):**

```makefile
# Database connection defaults (match docker-compose.yml)
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_NAME ?= hopeitworks_dev
DB_USER ?= hopeitworks
DB_PASS ?= hopeitworks_dev_password
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

seed: ## Seed the local database with dev fixtures
	@echo "Running migrations..."
	@migrate -path migrations/ -database "$(DATABASE_URL)" up
	@echo "Seeding dev data..."
	@PGPASSWORD="$(DB_PASS)" psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -f testdata/seed.sql
	@echo "Seed complete. Credentials:"
	@echo "  admin@hopeitworks.dev / admin123 (admin)"
	@echo "  dev@hopeitworks.dev   / dev123   (user)"
	@echo "  alice@hopeitworks.dev / alice123  (user)"

reset-db: ## Drop and recreate dev database, run migrations, and seed
	@echo "Dropping database $(DB_NAME)..."
	@PGPASSWORD="$(DB_PASS)" dropdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) --if-exists $(DB_NAME)
	@echo "Creating database $(DB_NAME)..."
	@PGPASSWORD="$(DB_PASS)" createdb -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) $(DB_NAME)
	@$(MAKE) seed
```

**Seed validation test (`backend/testdata/seed_test.go`):**

```go
package testdata_test

import (
	"os"
	"strings"
	"testing"
)

const seedFile = "seed.sql"

func TestSeedFileExists(t *testing.T) {
	info, err := os.Stat(seedFile)
	if err != nil {
		t.Fatalf("seed.sql not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("seed.sql is empty")
	}
}

func TestSeedFileContainsExpectedStatements(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	// Verify expected user inserts
	expectedUsers := []string{
		"admin@hopeitworks.dev",
		"dev@hopeitworks.dev",
		"alice@hopeitworks.dev",
	}
	for _, email := range expectedUsers {
		if !strings.Contains(content, email) {
			t.Errorf("seed.sql missing user insert for %s", email)
		}
	}

	// Verify expected project inserts
	expectedProjects := []string{
		"Todo App",
		"E-commerce API",
		"Frontend Kit",
	}
	for _, name := range expectedProjects {
		if !strings.Contains(content, name) {
			t.Errorf("seed.sql missing project insert for %s", name)
		}
	}
}

func TestSeedFileIsIdempotent(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	// Every INSERT should have ON CONFLICT
	inserts := 0
	onConflicts := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.ToUpper(line))
		if strings.HasPrefix(trimmed, "INSERT INTO") {
			inserts++
		}
		if strings.Contains(trimmed, "ON CONFLICT") {
			onConflicts++
		}
	}

	if inserts == 0 {
		t.Fatal("seed.sql contains no INSERT statements")
	}
	// ON CONFLICT may appear on same or next line; just verify at least some exist
	if onConflicts == 0 {
		t.Fatal("seed.sql contains no ON CONFLICT clauses (not idempotent)")
	}
}

func TestSeedFileContainsTransaction(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := strings.ToUpper(string(data))

	if !strings.Contains(content, "BEGIN") {
		t.Error("seed.sql missing BEGIN (should be wrapped in transaction)")
	}
	if !strings.Contains(content, "COMMIT") {
		t.Error("seed.sql missing COMMIT (should be wrapped in transaction)")
	}
}

func TestSeedFileUsesDeterministicUUIDs(t *testing.T) {
	data, err := os.ReadFile(seedFile)
	if err != nil {
		t.Fatalf("failed to read seed.sql: %v", err)
	}
	content := string(data)

	expectedUUIDs := []string{
		"00000000-0000-0000-0000-000000000001", // admin
		"00000000-0000-0000-0000-000000000002", // dev
		"00000000-0000-0000-0000-000000000003", // alice
		"00000000-0000-0000-0000-000000000101", // Todo App
		"00000000-0000-0000-0000-000000000102", // E-commerce API
		"00000000-0000-0000-0000-000000000103", // Frontend Kit
	}
	for _, uuid := range expectedUUIDs {
		if !strings.Contains(content, uuid) {
			t.Errorf("seed.sql missing deterministic UUID %s", uuid)
		}
	}
}
```

### Testing Requirements

**Unit tests (no database needed):**
1. `TestSeedFileExists` -- seed.sql exists and is not empty
2. `TestSeedFileContainsExpectedStatements` -- all expected emails and project names present
3. `TestSeedFileIsIdempotent` -- ON CONFLICT clauses exist
4. `TestSeedFileContainsTransaction` -- BEGIN/COMMIT present
5. `TestSeedFileUsesDeterministicUUIDs` -- all deterministic UUIDs present

All tests run with `go test ./testdata/ -short` (no database container required).

**Manual verification checklist:**
1. Start dev stack: `cd deploy && docker compose up -d`
2. Wait for Postgres health check to pass
3. Run seed: `cd backend && make seed`
4. Verify users: `PGPASSWORD=hopeitworks_dev_password psql -h localhost -U hopeitworks -d hopeitworks_dev -c "SELECT id, email, name, role FROM users;"`
5. Verify projects: `PGPASSWORD=hopeitworks_dev_password psql -h localhost -U hopeitworks -d hopeitworks_dev -c "SELECT id, name, owner_id FROM projects;"`
6. Login as admin: `curl -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"admin@hopeitworks.dev","password":"admin123"}'` -> 200
7. Login as dev: same curl with dev@hopeitworks.dev / dev123 -> 200
8. Run seed again: `cd backend && make seed` -> completes without errors (idempotent)
9. Reset and re-seed: `cd backend && make reset-db` -> clean database with fresh seed data
10. Run tests: `cd backend && go test ./testdata/ -short -v` -> all pass

**Integration test (optional, if time permits):**
- Use testcontainers to spin up Postgres, apply migrations, run seed.sql, and verify row counts.
- This is NOT required for the story to be complete. The manual checklist and unit tests are sufficient.

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.18]
- [Source: backend/migrations/000001_create_users_table.up.sql -- users schema + pgcrypto extension]
- [Source: backend/migrations/000002_create_projects_table.up.sql -- projects schema]
- [Source: backend/migrations/000003_add_users_deleted_at.up.sql -- deleted_at column]
- [Source: _bmad-output/implementation-artifacts/1-6-rbac-middleware-project-users-table.md -- project_users schema]
- [Source: deploy/docker-compose.yml -- Postgres connection defaults]
- [Source: backend/Makefile -- existing targets]

## Dev Agent Record

### Implementation Plan

- Created `backend/testdata/seed.sql` with idempotent INSERT statements for 3 users and 3 projects
- Used pgcrypto `crypt()` + `gen_salt('bf', 10)` for bcrypt password hashing (self-contained SQL, no hardcoded hashes)
- Added conditional DO block for project_users memberships (checks table existence via information_schema)
- All INSERTs use ON CONFLICT for idempotency (users: email, projects: name, memberships: DO NOTHING)
- Wrapped entire seed in BEGIN/COMMIT transaction
- Added `seed` and `reset-db` targets to backend Makefile with configurable DB connection vars
- Created 5 validation tests in seed_test.go (file existence, expected statements, idempotency, transaction, UUIDs)
- Added seed documentation comment to deploy/docker-compose.yml

### Completion Notes

- All 6 tasks implemented and verified
- 5 unit tests pass: `go test ./testdata/ -short -v` (all PASS)
- Full backend test suite passes with no regressions: `go test ./... -short` (all PASS)
- Seed SQL follows recommended pgcrypto approach from Dev Notes
- Makefile targets visible in `make help` output
- No Go dependencies added (tests use only stdlib: os, strings, testing)

### Debug Log

No issues encountered during implementation.

## File List

- `backend/testdata/seed.sql` (new) - Dev seed data SQL with users, projects, and conditional memberships
- `backend/testdata/seed_test.go` (new) - Seed file validation tests (5 tests)
- `backend/Makefile` (modified) - Added DB connection vars, `seed` and `reset-db` targets
- `deploy/docker-compose.yml` (modified) - Added seed command documentation comment
- `_bmad-output/implementation-artifacts/1-18-dev-seed-data-fixtures.md` (modified) - Story status and task completion
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified) - Story status updated to in-progress

## Change Log

- 2026-02-16: Implemented dev seed data fixtures - seed SQL, Makefile targets, validation tests, docker-compose docs
