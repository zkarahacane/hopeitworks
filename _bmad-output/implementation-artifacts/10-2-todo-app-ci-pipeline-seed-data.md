# Story 10.2: [SHARED] Todo app CI pipeline + seed data

Status: review

## Story

As a developer,
I want the todo app to have a functioning CI pipeline and seed data,
So that I can validate CI polling in the main system.

## Acceptance Criteria (BDD)

**AC1: CI config file with build/test/lint stages**
- **Given** the todo app has a CI config file
- **When** I examine the config
- **Then** I see stages for build, test, and lint

**AC2: CI pipeline runs on PR creation**
- **Given** the CI pipeline is configured
- **When** a PR is created for the todo app
- **Then** the CI pipeline runs all stages

**AC3: Seed data exists with sample todos**
- **Given** seed data exists
- **When** I examine test-project/seed.sql
- **Then** I see 5-10 sample todos for testing

**AC4: E2E tests validate CRUD operations**
- **Given** E2E tests exist
- **When** I run the E2E tests
- **Then** CRUD operations are validated via curl-based tests

**AC5: CI pipeline is permanently green**
- **Given** the CI pipeline is functional
- **When** all tests pass
- **Then** the pipeline is permanently green (baseline for validation)

## Tasks / Subtasks

- [x] [SHARED] Task 1: Create test-project base structure (AC: #1, #2, #5)
  - [x] Create `test-project/` directory with Node.js todo API scaffolding
  - [x] Create `test-project/package.json` with dependencies and scripts
  - [x] Create Express-based REST API with CRUD endpoints for todos
  - [x] Create simple HTML frontend for managing todos
  - [x] Create `test-project/Dockerfile` for the app
  - [x] Create `test-project/docker-compose.yml` for local dev

- [x] [SHARED] Task 2: Add unit tests for the todo API (AC: #4, #5)
  - [x] Create test framework setup with Jest + supertest
  - [x] Write unit tests for CRUD operations (17 tests)
  - [x] Write tests for input validation and error handling

- [x] [SHARED] Task 3: Add ESLint configuration for linting (AC: #1, #5)
  - [x] Create `eslint.config.js` in test-project (ESLint 9 flat config)
  - [x] Ensure `npm run lint` passes with no errors

- [x] [SHARED] Task 4: Create seed.sql with sample todos (AC: #3)
  - [x] Create `test-project/seed.sql` with schema creation
  - [x] Add 8 sample todo items
  - [x] Ensure seed is idempotent (uses INSERT OR REPLACE)

- [x] [SHARED] Task 5: Create GitHub Actions CI pipeline (AC: #1, #2, #5)
  - [x] Create `test-project/.github/workflows/ci.yml`
  - [x] Configure build stage (npm ci + docker build)
  - [x] Configure test stage (npm test)
  - [x] Configure lint stage (npm run lint)
  - [x] Ensure pipeline triggers on PR creation and push to main

- [x] [SHARED] Task 6: Create E2E/integration tests (AC: #4, #5)
  - [x] Create curl-based E2E test script (20 assertions)
  - [x] Validate CRUD operations (create, read, update, delete todos)
  - [x] E2E tests configured to run in CI via Docker container

- [x] [SHARED] Task 7: Create README documentation
  - [x] Document project purpose and setup instructions
  - [x] Document CI pipeline stages
  - [x] Document seed data and how to use it

## Dev Notes

This story creates a reference todo application inside `test-project/` that serves as a validation baseline for the hopeitworks pipeline. The app is intentionally simple: a Node.js Express API with SQLite (for simplicity/portability), a static HTML frontend, and a CI pipeline.

### Dependencies

**Story 10-1 (backlog):** Todo app reference project structure. Since 10-1 has not been implemented, this story creates the base project structure as part of Task 1, then layers CI pipeline and seed data on top.

### Architecture Requirements

**File paths:**

```
test-project/
├── .github/
│   └── workflows/
│       └── ci.yml              # GitHub Actions CI pipeline
├── src/
│   ├── app.js                  # Express app setup
│   ├── routes/
│   │   └── todos.js            # CRUD routes
│   ├── db.js                   # SQLite database setup
│   └── public/
│       └── index.html          # Simple HTML frontend
├── __tests__/
│   ├── todos.test.js           # Unit tests (17 tests)
│   └── e2e.test.sh             # Curl-based E2E tests (20 assertions)
├── seed.sql                    # Seed data (8 todos)
├── package.json                # Dependencies and scripts
├── eslint.config.js            # ESLint 9 flat config
├── Dockerfile                  # Container build
├── docker-compose.yml          # Local dev stack
├── .gitignore                  # Git ignore rules
└── README.md                   # Documentation
```

### Technical Specifications

- **Runtime:** Node.js 20+ with Express
- **Database:** SQLite via better-sqlite3 (portable, no external dependencies for CI)
- **Testing:** Jest + supertest for unit tests (17 tests), bash + curl for E2E (20 assertions)
- **Linting:** ESLint 9 with flat config and recommended rules
- **CI:** GitHub Actions with install, lint, test, Docker build, and E2E stages

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 10.2]
- [Source: .github/workflows/ci.yml -- main project CI pipeline for reference]

## Dev Agent Record

### Implementation Plan

Created a minimal but complete Node.js todo application with:
1. Express REST API with CRUD endpoints (GET/POST/PUT/DELETE)
2. SQLite database for portability (better-sqlite3)
3. Unit tests with Jest + supertest (17 tests passing)
4. ESLint 9 for code quality (flat config, all passing)
5. GitHub Actions CI pipeline (install, lint, test, build, E2E)
6. Seed data with 8 sample todos (idempotent with INSERT OR REPLACE)
7. Curl-based E2E tests (20 assertions, all passing)

### Completion Notes

- All 17 unit tests pass (CRUD operations, input validation, error handling)
- All 20 E2E test assertions pass (health check, create, read, update, delete, error cases)
- ESLint passes with zero errors
- Docker build configured with health check
- Seed data contains 8 sample todos covering completed and incomplete states
- CI pipeline configured to trigger on push to main, PRs to main, and manual dispatch

## File List

- `test-project/package.json` (new) - Project dependencies and scripts
- `test-project/src/app.js` (new) - Express app entry point with health endpoint
- `test-project/src/db.js` (new) - SQLite database connection and schema
- `test-project/src/routes/todos.js` (new) - CRUD route handlers
- `test-project/src/public/index.html` (new) - Static HTML frontend
- `test-project/__tests__/todos.test.js` (new) - Jest unit tests (17 tests)
- `test-project/__tests__/e2e.test.sh` (new) - Curl-based E2E test script (20 assertions)
- `test-project/seed.sql` (new) - Database seed data (8 todos)
- `test-project/eslint.config.js` (new) - ESLint 9 flat configuration
- `test-project/Dockerfile` (new) - Docker container build
- `test-project/docker-compose.yml` (new) - Local dev stack
- `test-project/.gitignore` (new) - Git ignore rules
- `test-project/.github/workflows/ci.yml` (new) - GitHub Actions CI pipeline
- `test-project/README.md` (new) - Project documentation
- `_bmad-output/implementation-artifacts/10-2-todo-app-ci-pipeline-seed-data.md` (new) - Story file

## Change Log

- 2026-02-17: Created complete test-project reference todo application with CI pipeline, seed data, unit tests (17), E2E tests (20 assertions), ESLint config, Docker support, and documentation
