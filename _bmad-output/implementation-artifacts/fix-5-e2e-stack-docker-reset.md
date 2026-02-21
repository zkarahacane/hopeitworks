# Story fix-5: Fix e2e-stack.sh to reset database via Docker exec

Status: ready-for-dev

## Story

As a developer,
I want `./scripts/e2e-stack.sh reset` to work without local PostgreSQL tools installed,
so that I can reset the test database using only Docker.

## Bug

The `reset` command calls `make reset-db` which uses `dropdb` and `createdb` CLI tools. These are not installed on all dev machines. Since Postgres runs in Docker, the reset should use `docker exec` to run commands inside the container.

## Acceptance Criteria (BDD)

**AC1: reset works without local psql tools**
- **Given** a machine without `dropdb` or `createdb` installed
- **When** running `./scripts/e2e-stack.sh reset`
- **Then** the command completes successfully

**AC2: reset uses docker exec**
- **Given** Postgres is running in the `hopeitworks-postgres` container
- **When** the reset command executes
- **Then** it uses `docker exec hopeitworks-postgres` to run psql commands inside the container

**AC3: Seed data is present after reset**
- **Given** `./scripts/e2e-stack.sh reset` has completed
- **When** querying the database
- **Then** seed data is present (3 users, 3 projects, epics, stories)

**AC4: Full lifecycle works without error**
- **Given** a stopped stack
- **When** running `./scripts/e2e-stack.sh up` followed by `reset` then `status`
- **Then** all three commands complete without error

## Tasks / Subtasks

- [ ] Task 1: Modify cmd_reset() to use docker exec
  - [ ] In `scripts/e2e-stack.sh`, replace the `make reset-db` call with `docker exec hopeitworks-postgres psql -U hopeitworks -d postgres` commands for drop/recreate
- [ ] Task 2: Handle migrations inside Docker
  - [ ] Run migrations via `docker exec` on the API container, or mount and run migration files directly
  - [ ] Alternatively call `docker exec hopeitworks-api /app/api migrate` if the binary supports it
- [ ] Task 3: Ensure seed data is applied after reset
  - [ ] Verify the existing seed mechanism is triggered after migrations run
- [ ] Task 4: Test the full cycle
  - [ ] Test `up` → `reset` → `status` end-to-end on a machine without local psql tools
