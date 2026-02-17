# Test Project — Pipeline Validation Reference

This directory contains a reference todo application used to validate the
hopeitworks pipeline end-to-end. It provides sample stories, seed data, and
a known-good baseline for integration testing.

## Structure

```
test-project/
├── README.md             # This file
└── stories/
    └── todo-stories.md   # 5 sample stories in frontmatter markdown format
```

## Sample Stories

The `stories/todo-stories.md` file contains 5 stories in standard frontmatter
format:

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

## Pipeline Validation Flow

The integration test suite in `backend/internal/integration/` validates the
full pipeline using these stories:

### Test: Story Import (`TestIntegration_PipelineValidation_StoryImport`)

1. Reads `test-project/stories/todo-stories.md`
2. Parses markdown into story blocks via the markdown adapter
3. Imports stories into Postgres via `StoryService.Import()`
4. Verifies all 5 stories created with correct keys, scopes, dependencies
5. Verifies re-import updates existing stories (idempotent)

### Test: Run Creation (`TestIntegration_PipelineValidation_RunCreation`)

1. Creates a project with a 3-step pipeline config (implement, review, merge)
2. Creates a story and launches a run via `RunService.LaunchRun()`
3. Verifies run created in `pending` status with 3 steps in correct order
4. Verifies pipeline config snapshot persisted as JSON
5. Verifies duplicate launch is blocked for active runs

### Test: Pipeline Execution (`TestIntegration_PipelineValidation_Execution`)

1. Sets up project, story, pipeline config, and run
2. Registers noop actions in the `ActionRegistry`
3. Executes the pipeline via `PipelineExecutor.ExecuteRun()`
4. Verifies run transitions: `pending` -> `running` -> `completed`
5. Verifies all steps transition: `pending` -> `running` -> `completed`
6. Verifies events published to the events table (run.started, step.started/completed, run.completed)

### Test: Full Flow (`TestIntegration_PipelineValidation_FullFlow`)

End-to-end validation combining all the above:
1. Creates a project with pipeline config
2. Imports all 5 stories from test-project markdown
3. Picks TODO-1 (no dependencies) and launches a run
4. Executes the pipeline with noop actions
5. Verifies final state: run completed, all steps completed, events generated

## Running the Tests

```bash
# From the backend directory:

# Run only pipeline validation integration tests
go test ./internal/integration/ -v -run TestIntegration_PipelineValidation

# Run all integration tests (skip unit tests)
go test ./... -run TestIntegration

# Run unit tests only (skips integration tests)
go test ./... -short
```

## Test Infrastructure

The integration tests use:
- **testcontainers-go**: Ephemeral Postgres 16 containers
- **Real migrations**: All `.up.sql` files applied automatically
- **Real adapters**: Postgres repositories, event publisher
- **Mock actions**: Noop actions that succeed immediately (no Docker/agent containers)
- **Shared testutil**: `backend/internal/testutil/` provides `SetupTestDB`, `CreateProject`, and other factories

## Design Decisions

- **Noop actions instead of real containers**: Integration tests validate the
  pipeline orchestration logic (state machine, events, step ordering) against
  real Postgres without requiring Docker-in-Docker for agent containers.
- **testcontainers for isolation**: Each test gets a fresh Postgres instance
  with all migrations applied, ensuring tests are independent and deterministic.
- **Shared testutil package**: Test helpers extracted to `backend/internal/testutil/`
  for reuse across integration test suites.
