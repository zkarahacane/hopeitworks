# Todo App - Reference Project

A minimal todo application used as a reference project for validating the hopeitworks pipeline end-to-end.

## Purpose

This project serves as a baseline for the hopeitworks CI polling and pipeline validation features. It provides:

- A simple REST API with CRUD operations for todos
- A static HTML frontend for managing todos
- A CI pipeline with build, lint, and test stages
- PostgreSQL as the database backend

## Tech Stack

- **Runtime:** Node.js 20+
- **Framework:** Express
- **Database:** PostgreSQL 16
- **Testing:** Node.js built-in test runner (`node --test`)
- **Linting:** ESLint 9 (flat config)
- **CI:** GitHub Actions

## Prerequisites

- Docker and Docker Compose v2
- Node.js 20+ (for local development without Docker)

## Getting Started

### Docker (Recommended)

```bash
# Start the full stack (Postgres + app)
docker compose up -d

# Verify it's running
curl http://localhost:3000/health

# Stop and clean up
docker compose down -v
```

### Local Development

Requires a running PostgreSQL instance.

```bash
# Install backend dependencies
cd backend && npm install

# Start the app (connects to Postgres via DATABASE_URL)
DATABASE_URL=postgres://todo:todo@localhost:5432/todo npm start
# App runs on http://localhost:3000
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/todos` | List all todos |
| GET | `/api/todos/:id` | Get a todo by ID |
| POST | `/api/todos` | Create a new todo |
| PUT | `/api/todos/:id` | Update a todo |
| DELETE | `/api/todos/:id` | Delete a todo |

### Request/Response Examples

**Create a todo:**
```bash
curl -X POST http://localhost:3000/api/todos \
  -H "Content-Type: application/json" \
  -d '{"title": "Buy groceries"}'
```

**List todos:**
```bash
curl http://localhost:3000/api/todos
```

## Testing

```bash
# Unit tests (uses mock pool, no Postgres required)
cd backend && npm test

# Integration tests against running stack
docker compose up -d
DATABASE_URL=postgres://todo:todo@localhost:5432/todo cd backend && npm test

# Lint
cd backend && npm run lint
```

## CI Pipeline

The GitHub Actions CI pipeline (`.github/workflows/ci.yml`) runs the following stages:

1. **Install** - Install npm dependencies
2. **Lint** - Run ESLint on source and test files
3. **Unit Tests** - Run test suite via `node --test`
4. **Build** - Build Docker image
5. **E2E Tests** - Start the app in Docker and run curl-based E2E tests

The pipeline triggers on:
- Push to `main`
- Pull requests targeting `main`
- Manual dispatch

## Database Schema

The `init.sql` file contains the PostgreSQL schema for the `todos` table. It is automatically applied when the Postgres container starts for the first time via Docker's `/docker-entrypoint-initdb.d/` mechanism.

---

## Pipeline Validation — Integration Test Reference

This directory also contains sample stories used to validate the hopeitworks pipeline end-to-end. It provides sample stories, seed data, and a known-good baseline for integration testing.

### Structure

```
test-project/
├── README.md             # This file
├── backend/              # Node.js + Express + pg (PostgreSQL)
│   ├── app.js            # Express routes
│   ├── server.js         # Entry point
│   └── test/             # Unit tests
├── init.sql              # PostgreSQL schema
├── docker-compose.yml    # Full stack: Postgres + app
├── Dockerfile            # Builds the backend app
└── stories/
    └── todo-stories.md   # 5 sample stories in frontmatter markdown format
```

### Sample Stories

The `stories/todo-stories.md` file contains 5 stories in standard frontmatter format:

| Key    | Title                               | Scope    | Dependencies     |
|--------|-------------------------------------|----------|------------------|
| TODO-1 | Add create todo endpoint            | backend  | none             |
| TODO-2 | Add list todos endpoint             | backend  | none             |
| TODO-3 | Add update todo endpoint            | backend  | TODO-1           |
| TODO-4 | Add delete todo endpoint            | backend  | TODO-1           |
| TODO-5 | Add todo list UI with toggle        | frontend | TODO-2, TODO-3   |

These stories exercise:
- Multiple scopes (backend, frontend)
- Dependency chains (linear and diamond)
- Standard YAML frontmatter parsing

### End-to-End with hopeitworks

To run a real pipeline on the todo app:

1. Ensure `hopeitworks/agent:latest` is built (runtime-1)
2. Ensure actions are wired in the action registry (runtime-2)
3. Start the hopeitworks dev stack: `cd deploy && docker compose up -d`
4. Register `test-project/` as a project in hopeitworks (via UI or API) using the GitHub URL of the hopeitworks repo
5. Import a story from `test-project/stories/todo-stories.md` (e.g., `TODO-1`)
6. Launch a run via UI or `POST /api/v1/projects/{id}/stories/{story_id}/runs`
7. Observe the pipeline: implement step -> container spawns -> agent clones repo -> runs claude -> pushes branch -> CI polls -> HITL gate -> merge

### Pipeline Validation Flow

The integration test suite in `backend/internal/integration/` validates the full pipeline using these stories:

#### Test: Story Import (`TestIntegration_PipelineValidation_StoryImport`)

1. Reads `test-project/stories/todo-stories.md`
2. Parses markdown into story blocks via the markdown adapter
3. Imports stories into Postgres via `StoryService.Import()`
4. Verifies all 5 stories created with correct keys, scopes, dependencies
5. Verifies re-import updates existing stories (idempotent)

#### Test: Run Creation (`TestIntegration_PipelineValidation_RunCreation`)

1. Creates a project with a 3-step pipeline config (implement, review, merge)
2. Creates a story and launches a run via `RunService.LaunchRun()`
3. Verifies run created in `pending` status with 3 steps in correct order
4. Verifies pipeline config snapshot persisted as JSON
5. Verifies duplicate launch is blocked for active runs

#### Test: Pipeline Execution (`TestIntegration_PipelineValidation_Execution`)

1. Sets up project, story, pipeline config, and run
2. Registers noop actions in the `ActionRegistry`
3. Executes the pipeline via `PipelineExecutor.ExecuteRun()`
4. Verifies run transitions: `pending` -> `running` -> `completed`
5. Verifies all steps transition: `pending` -> `running` -> `completed`
6. Verifies events published to the events table (run.started, step.started/completed, run.completed)

#### Test: Full Flow (`TestIntegration_PipelineValidation_FullFlow`)

End-to-end validation combining all the above:
1. Creates a project with pipeline config
2. Imports all 5 stories from test-project markdown
3. Picks TODO-1 (no dependencies) and launches a run
4. Executes the pipeline with noop actions
5. Verifies final state: run completed, all steps completed, events generated

### Running the Tests

```bash
# From the backend directory:

# Run only pipeline validation integration tests
go test ./internal/integration/ -v -run TestIntegration_PipelineValidation

# Run all integration tests (skip unit tests)
go test ./... -run TestIntegration

# Run unit tests only (skips integration tests)
go test ./... -short
```

### Test Infrastructure

The integration tests use:
- **testcontainers-go**: Ephemeral Postgres 16 containers
- **Real migrations**: All `.up.sql` files applied automatically
- **Real adapters**: Postgres repositories, event publisher
- **Mock actions**: Noop actions that succeed immediately (no Docker/agent containers)
- **Shared testutil**: `backend/internal/testutil/` provides `SetupTestDB`, `CreateProject`, and other factories

### Design Decisions

- **Noop actions instead of real containers**: Integration tests validate the
  pipeline orchestration logic (state machine, events, step ordering) against
  real Postgres without requiring Docker-in-Docker for agent containers.
- **testcontainers for isolation**: Each test gets a fresh Postgres instance
  with all migrations applied, ensuring tests are independent and deterministic.
- **Shared testutil package**: Test helpers extracted to `backend/internal/testutil/`
  for reuse across integration test suites.
