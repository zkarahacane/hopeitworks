# Story 3.1: [BACK] Runs & Run Steps Tables + Run Creation API + State Machine

Status: ready-for-dev

## Story

As a backend developer, I want database schemas for runs/run_steps and a run creation service with state machine transitions, so that the system can track pipeline execution state persistently.

## Acceptance Criteria (BDD)

**AC1: Runs table schema supports all pipeline execution lifecycle states**
- **Given** the database is initialized
- **When** I query the schema for the runs table
- **Then** it contains: id (UUID PK), project_id (FK projects CASCADE), story_id (UUID NOT NULL), status (VARCHAR NOT NULL DEFAULT 'pending' CHECK IN pending/running/completed/failed/cancelled), pipeline_config_snapshot (JSONB), started_at (TIMESTAMPTZ), completed_at (TIMESTAMPTZ), error_message (TEXT), created_at, updated_at
- **And** indexes exist on (project_id, status) and (story_id)
- **And** story_id does NOT have FK constraint (stories table not created yet)

**AC2: Run steps table schema tracks individual step execution**
- **Given** the database is initialized
- **When** I query the schema for the run_steps table
- **Then** it contains: id (UUID PK), run_id (FK runs CASCADE), step_name (VARCHAR NOT NULL), step_order (INT NOT NULL), action (VARCHAR NOT NULL), status (VARCHAR NOT NULL DEFAULT 'pending' CHECK IN pending/running/completed/failed/cancelled), started_at (TIMESTAMPTZ), completed_at (TIMESTAMPTZ), error_message (TEXT), container_id (VARCHAR), log_tail (TEXT), created_at (TIMESTAMPTZ NOT NULL DEFAULT now())
- **And** index exists on (run_id, step_order)
- **And** run_id FK cascades deletes from runs table

**AC3: State machine enforces valid run status transitions**
- **Given** a run with status "pending"
- **When** I transition to "running"
- **Then** the transition succeeds
- **When** I transition "running" → "completed"
- **Then** the transition succeeds
- **When** I attempt invalid transition "completed" → "running"
- **Then** DomainError INVALID_STATE_TRANSITION is returned

**AC4: State machine enforces valid run step status transitions**
- **Given** a run step with status "pending"
- **When** I transition to "running"
- **Then** the transition succeeds
- **When** I transition "running" → "failed"
- **Then** the transition succeeds
- **When** I attempt invalid transition "failed" → "pending"
- **Then** DomainError INVALID_STATE_TRANSITION is returned

**AC5: CreateRun service creates run with steps from pipeline config**
- **Given** a project exists with pipeline config containing 3 steps
- **When** I call RunService.CreateRun(ctx, projectID, storyID)
- **Then** a new run is created with status "pending"
- **And** 3 run_steps are created with step_order 0, 1, 2
- **And** each step has status "pending"
- **And** pipeline_config_snapshot contains the full pipeline config JSON

**AC6: List runs by project returns paginated results**
- **Given** a project with 15 runs
- **When** I call GET /projects/{id}/runs?page=1&per_page=10
- **Then** response contains data array with 10 runs
- **And** pagination metadata shows total=15, page=1, per_page=10
- **And** runs are ordered by created_at DESC

**AC7: List runs by story returns all runs for that story**
- **Given** a story with 3 runs
- **When** I call GET /stories/{id}/runs
- **Then** response contains data array with 3 runs
- **And** all runs have matching story_id

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migrations for runs and run_steps tables (AC: #1, #2)
  - [ ] Write 000006_create_runs_table.up.sql with schema, indexes, CHECK constraint
  - [ ] Write 000006_create_runs_table.down.sql
  - [ ] Write 000007_create_run_steps_table.up.sql with schema, indexes, FK CASCADE
  - [ ] Write 000007_create_run_steps_table.down.sql
  - [ ] Test migrations: up, down, up cycle validates schema

- [ ] [BACK] Task 2: Create sqlc queries for runs and run_steps (AC: #5, #6, #7)
  - [ ] Write backend/queries/runs.sql: CreateRun, GetRun, ListRunsByProject, ListRunsByStory, UpdateRunStatus, CountRunsByProject
  - [ ] Write backend/queries/run_steps.sql: CreateRunStep, GetRunStep, ListRunStepsByRun, UpdateRunStepStatus
  - [ ] Run `make generate` to generate sqlc types

- [ ] [BACK] Task 3: Implement domain models and state machine logic (AC: #3, #4)
  - [ ] Create backend/internal/domain/model/run.go with Run, RunStep, RunStatus, StepStatus types
  - [ ] Implement ValidateRunTransition(from, to RunStatus) error with transition map
  - [ ] Implement ValidateStepTransition(from, to StepStatus) error with transition map
  - [ ] Return errors.NewDomain(errors.CategoryInvalidState, "INVALID_STATE_TRANSITION", msg) on invalid transitions

- [ ] [BACK] Task 4: Define port interfaces (AC: #5, #6, #7)
  - [ ] Create backend/internal/domain/port/run_repository.go with RunRepository interface
  - [ ] Methods: CreateRun, GetRun, ListRunsByProject, ListRunsByStory, UpdateRunStatus, CreateRunStep, GetRunStep, ListRunStepsByRun, UpdateRunStepStatus, CountRunsByProject

- [ ] [BACK] Task 5: Implement Postgres adapter (AC: #1, #2, #5)
  - [ ] Create backend/internal/adapter/postgres/run_repo.go implementing RunRepository
  - [ ] Wrap sqlc calls, map DB errors to DomainErrors
  - [ ] CreateRun: insert run + insert steps in transaction
  - [ ] UpdateRunStatus: validate transition before UPDATE
  - [ ] UpdateRunStepStatus: validate transition before UPDATE

- [ ] [BACK] Task 6: Implement RunService with create + transitions (AC: #3, #4, #5)
  - [ ] Create backend/internal/domain/service/run_service.go
  - [ ] CreateRun(ctx, projectID, storyID): fetch project pipeline config, create run + steps from config
  - [ ] TransitionRun(ctx, runID, newStatus): validate transition, update status, set started_at/completed_at timestamps
  - [ ] TransitionRunStep(ctx, stepID, newStatus): validate transition, update status, set timestamps

- [ ] [BACK] Task 7: Implement API handlers for read endpoints (AC: #6, #7)
  - [ ] Create backend/internal/api/handler/run_handler.go
  - [ ] GET /projects/{id}/runs: ListRunsByProject with pagination
  - [ ] GET /stories/{id}/runs: ListRunsByStory with pagination
  - [ ] GET /runs/{id}: GetRun with nested steps
  - [ ] Map domain errors to HTTP status codes via error middleware

- [ ] [BACK] Task 8: Write unit tests for service and state machine (AC: #3, #4, #5)
  - [ ] Test ValidateRunTransition: all valid transitions pass, invalid transitions fail
  - [ ] Test ValidateStepTransition: all valid transitions pass, invalid transitions fail
  - [ ] Test RunService.CreateRun: creates run + steps from config, handles missing project error
  - [ ] Test RunService.TransitionRun: validates transitions, sets timestamps, propagates repo errors

- [ ] [BACK] Task 9: Wire dependencies and integration test (AC: #1-#7)
  - [ ] Add RunRepository and RunService to wire.go provider sets
  - [ ] Run `go generate ./cmd/api` to regenerate wire_gen.go
  - [ ] Write integration test: create project → create run → list runs → transition run → verify state
  - [ ] Verify migrations apply cleanly in testcontainer

## Dev Notes

### Dependencies
- **Story 1-5:** projects table must exist (FK project_id)
- **Story 1-1:** Go scaffolding, pgx/v5, docker-compose dev stack
- **Future:** Story 2-2 will add FK constraint from runs.story_id → stories.id (not in this story)

### Architecture Requirements
- Hexagonal architecture: domain/model → domain/port → domain/service → adapter/postgres → api/handler
- State machine logic in domain/model, enforced in adapter before DB update
- RunService orchestrates CreateRun transaction (run + steps)
- sqlc for all DB queries
- DomainError for all business logic errors

### File Paths (exact)

```
backend/migrations/000006_create_runs_table.up.sql
backend/migrations/000006_create_runs_table.down.sql
backend/migrations/000007_create_run_steps_table.up.sql
backend/migrations/000007_create_run_steps_table.down.sql
backend/queries/runs.sql
backend/queries/run_steps.sql
backend/internal/domain/model/run.go
backend/internal/domain/port/run_repository.go
backend/internal/domain/service/run_service.go
backend/internal/adapter/postgres/run_repo.go
backend/internal/api/handler/run_handler.go
backend/cmd/api/wire.go                              # Add providers
backend/cmd/api/wire_gen.go                          # Auto-generated
```

### Technical Specifications

#### Domain Model

```go
// backend/internal/domain/model/run.go
package model

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

type RunStatus string

const (
    RunStatusPending   RunStatus = "pending"
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCancelled RunStatus = "cancelled"
)

type StepStatus string

const (
    StepStatusPending   StepStatus = "pending"
    StepStatusRunning   StepStatus = "running"
    StepStatusCompleted StepStatus = "completed"
    StepStatusFailed    StepStatus = "failed"
    StepStatusCancelled StepStatus = "cancelled"
)

type Run struct {
    ID                     uuid.UUID
    ProjectID              uuid.UUID
    StoryID                uuid.UUID
    Status                 RunStatus
    PipelineConfigSnapshot json.RawMessage
    StartedAt              *time.Time
    CompletedAt            *time.Time
    ErrorMessage           *string
    CreatedAt              time.Time
    UpdatedAt              time.Time
    Steps                  []RunStep
}

type RunStep struct {
    ID           uuid.UUID
    RunID        uuid.UUID
    StepName     string
    StepOrder    int
    Action       string
    Status       StepStatus
    StartedAt    *time.Time
    CompletedAt  *time.Time
    ErrorMessage *string
    ContainerID  *string
    LogTail      *string
    CreatedAt    time.Time
}

// State machine transition maps
var validRunTransitions = map[RunStatus][]RunStatus{
    RunStatusPending: {RunStatusRunning, RunStatusCancelled},
    RunStatusRunning: {RunStatusCompleted, RunStatusFailed, RunStatusCancelled},
}

var validStepTransitions = map[StepStatus][]StepStatus{
    StepStatusPending: {StepStatusRunning, StepStatusCancelled},
    StepStatusRunning: {StepStatusCompleted, StepStatusFailed, StepStatusCancelled},
}

// ValidateRunTransition checks if status transition is valid
func ValidateRunTransition(from, to RunStatus) error {
    allowed, ok := validRunTransitions[from]
    if !ok {
        return errors.NewDomain(errors.CategoryInvalidState, "INVALID_STATE_TRANSITION",
            "no transitions allowed from status: %s", from)
    }
    for _, valid := range allowed {
        if valid == to {
            return nil
        }
    }
    return errors.NewDomain(errors.CategoryInvalidState, "INVALID_STATE_TRANSITION",
        "cannot transition from %s to %s", from, to)
}

// ValidateStepTransition checks if step status transition is valid
func ValidateStepTransition(from, to StepStatus) error {
    allowed, ok := validStepTransitions[from]
    if !ok {
        return errors.NewDomain(errors.CategoryInvalidState, "INVALID_STATE_TRANSITION",
            "no transitions allowed from status: %s", from)
    }
    for _, valid := range allowed {
        if valid == to {
            return nil
        }
    }
    return errors.NewDomain(errors.CategoryInvalidState, "INVALID_STATE_TRANSITION",
        "cannot transition from %s to %s", from, to)
}
```

#### Migration Schema

```sql
-- 000006_create_runs_table.up.sql
CREATE TABLE runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    story_id UUID NOT NULL,  -- No FK constraint yet, stories table created in Wave 6
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    pipeline_config_snapshot JSONB,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_runs_project_id_status ON runs(project_id, status);
CREATE INDEX idx_runs_story_id ON runs(story_id);

-- 000007_create_run_steps_table.up.sql
CREATE TABLE run_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    step_order INT NOT NULL,
    action VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    container_id VARCHAR(255),
    log_tail TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_run_steps_run_id_order ON run_steps(run_id, step_order);
```

#### Port Interface

```go
// backend/internal/domain/port/run_repository.go
package port

import (
    "context"
    "github.com/google/uuid"
    "hopeitworks/backend/internal/domain/model"
)

type RunRepository interface {
    CreateRun(ctx context.Context, run *model.Run) error
    GetRun(ctx context.Context, id uuid.UUID) (*model.Run, error)
    ListRunsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*model.Run, error)
    ListRunsByStory(ctx context.Context, storyID uuid.UUID, limit, offset int) ([]*model.Run, error)
    UpdateRunStatus(ctx context.Context, id uuid.UUID, status model.RunStatus, errorMsg *string) error
    CountRunsByProject(ctx context.Context, projectID uuid.UUID) (int, error)

    CreateRunStep(ctx context.Context, step *model.RunStep) error
    GetRunStep(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
    ListRunStepsByRun(ctx context.Context, runID uuid.UUID) ([]*model.RunStep, error)
    UpdateRunStepStatus(ctx context.Context, id uuid.UUID, status model.StepStatus, errorMsg *string) error
}
```

#### Service Interface

```go
// backend/internal/domain/service/run_service.go
package service

type RunService struct {
    runRepo     port.RunRepository
    projectRepo port.ProjectRepository
    logger      *slog.Logger
}

func NewRunService(runRepo port.RunRepository, projectRepo port.ProjectRepository, logger *slog.Logger) *RunService {
    return &RunService{runRepo: runRepo, projectRepo: projectRepo, logger: logger}
}

// CreateRun creates a new run with steps from project pipeline config
func (s *RunService) CreateRun(ctx context.Context, projectID, storyID uuid.UUID) (*model.Run, error) {
    // Fetch project to get pipeline config
    project, err := s.projectRepo.GetProject(ctx, projectID)
    if err != nil {
        return nil, err
    }

    // Parse pipeline config to extract steps
    var pipelineConfig struct {
        Steps []struct {
            Name   string `json:"name"`
            Action string `json:"action"`
        } `json:"steps"`
    }
    if err := json.Unmarshal(project.PipelineConfig, &pipelineConfig); err != nil {
        return nil, errors.NewDomain(errors.CategoryInvalidInput, "INVALID_PIPELINE_CONFIG",
            "failed to parse pipeline config: %v", err)
    }

    // Create run
    run := &model.Run{
        ID:                     uuid.New(),
        ProjectID:              projectID,
        StoryID:                storyID,
        Status:                 model.RunStatusPending,
        PipelineConfigSnapshot: project.PipelineConfig,
        CreatedAt:              time.Now(),
        UpdatedAt:              time.Now(),
    }

    if err := s.runRepo.CreateRun(ctx, run); err != nil {
        return nil, err
    }

    // Create steps
    for i, stepConfig := range pipelineConfig.Steps {
        step := &model.RunStep{
            ID:        uuid.New(),
            RunID:     run.ID,
            StepName:  stepConfig.Name,
            StepOrder: i,
            Action:    stepConfig.Action,
            Status:    model.StepStatusPending,
            CreatedAt: time.Now(),
        }
        if err := s.runRepo.CreateRunStep(ctx, step); err != nil {
            return nil, err
        }
    }

    return run, nil
}

// TransitionRun validates and transitions run to new status
func (s *RunService) TransitionRun(ctx context.Context, runID uuid.UUID, newStatus model.RunStatus) error {
    run, err := s.runRepo.GetRun(ctx, runID)
    if err != nil {
        return err
    }

    if err := model.ValidateRunTransition(run.Status, newStatus); err != nil {
        return err
    }

    return s.runRepo.UpdateRunStatus(ctx, runID, newStatus, nil)
}
```

### Testing Requirements

1. **Unit Tests (backend/internal/domain/model/run_test.go)**
   - ValidateRunTransition: all valid paths, all invalid paths
   - ValidateStepTransition: all valid paths, all invalid paths

2. **Unit Tests (backend/internal/domain/service/run_service_test.go)**
   - CreateRun: success with 3 steps, handles missing project, handles invalid config JSON
   - TransitionRun: success for valid transitions, fails for invalid transitions

3. **Integration Test (backend/internal/adapter/postgres/run_repo_test.go)**
   - CreateRun + CreateRunStep in transaction
   - UpdateRunStatus validates transition
   - ListRunsByProject pagination
   - Cascade delete: delete project → runs + steps deleted

4. **Linting**
   - Run `golangci-lint run ./...` — must pass before commit

### References

- Story 1-1: Go scaffolding, pgx/v5, docker-compose
- Story 1-5: projects table schema
- Epic 3 Planning: Run state machine design
- `backend/.golangci.yml`: Linting rules
- `backend/pkg/errors/domain.go`: DomainError implementation

## Dev Agent Record

## Change Log
