# Story fix-1: Fix migration numbering conflict and apply missing migrations

Status: done

## Story

As a developer,
I want all database migrations to apply cleanly,
so that the backend API doesn't return 500 errors on project routes.

## Bug

Two migration files share the `000013` prefix. The `000013_add_circuit_breaker_to_projects` migration was never applied. As a result, sqlc queries reference `circuit_breaker_count`, `circuit_breaker_active`, `circuit_breaker_max` columns that don't exist, causing all `/projects/*` endpoints to return 500.

## Acceptance Criteria (BDD)

**AC1: No duplicate migration numbers**
- **Given** the `backend/migrations/` directory
- **When** listing all migration files by their numeric prefix
- **Then** no two files share the same prefix number

**AC2: Migrations apply without error**
- **Given** a clean database
- **When** running `migrate -path migrations/ -database "$DATABASE_URL" up`
- **Then** all migrations apply without error

**AC3: Projects endpoint returns 200**
- **Given** migrations have been applied
- **When** sending `GET /api/v1/projects`
- **Then** the response status is 200 (not 500)

**AC4: circuit_breaker columns exist**
- **Given** migrations have been applied
- **When** inspecting the `projects` table schema
- **Then** columns `circuit_breaker_count`, `circuit_breaker_active`, and `circuit_breaker_max` exist

## Tasks / Subtasks

- [ ] Task 1: Audit migration files for numbering conflicts
  - [ ] List all files in `backend/migrations/` and identify duplicate numeric prefixes
- [ ] Task 2: Renumber the conflicting migration(s)
  - [ ] Assign a new unique prefix to the conflicting file (e.g., `000014` or next available)
  - [ ] Update both the `.up.sql` and `.down.sql` filenames consistently
- [ ] Task 3: Verify migration sequence
  - [ ] Confirm no gaps and no duplicates remain in the sequence
- [ ] Task 4: Test end-to-end
  - [ ] Run `make seed` and verify it completes without error after the migration fix

## Change Log

- Merged to develop via PR #98 (2026-02-21)
