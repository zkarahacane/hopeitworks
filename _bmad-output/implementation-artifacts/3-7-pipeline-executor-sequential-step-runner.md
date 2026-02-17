# Story 3.7: [BACK] Pipeline executor — sequential step runner

Status: ready-for-dev

## Story

As a backend developer, I want a pipeline executor that runs steps sequentially for a story, so that each pipeline step executes in order with proper state tracking.

## Acceptance Criteria (BDD)

**AC1: Action interface defines executable step behavior**
- **Given** an Action interface in `backend/internal/domain/model/action.go`
- **When** the interface is reviewed
- **Then** it declares Name() string method returning action identifier
- **And** it declares Execute(ctx context.Context, runCtx *RunContext) error method
- **And** the interface can be implemented by different action types (agent_run, git_create_pr, hitl_gate)

**AC2: RunContext model provides execution context for actions**
- **Given** a RunContext struct in `backend/internal/domain/model/run_context.go`
- **When** the struct is reviewed
- **Then** it includes Run *Run field with full run details
- **And** it includes RunStep *RunStep field with current step details
- **And** it includes ProjectID uuid.UUID field
- **And** it includes StoryID uuid.UUID field
- **And** it includes Metadata map[string]any field for inter-step data sharing

**AC3: ActionRegistry port interface manages action registration and lookup**
- **Given** an ActionRegistry port in `backend/internal/domain/port/action_registry.go`
- **When** the interface is reviewed
- **Then** it declares Register(action model.Action) method
- **And** it declares Get(name string) (model.Action, error) method
- **And** Get returns ACTION_NOT_FOUND error if action not registered

**AC4: PipelineExecutor executes steps sequentially in order**
- **Given** a run with 3 steps in pending status
- **When** PipelineExecutor.ExecuteRun is called
- **Then** run transitions to "running" and run.started event is published
- **And** step 1 transitions to "running", executes, transitions to "completed", publishes step.started and step.completed
- **And** step 2 transitions to "running", executes, transitions to "completed", publishes step.started and step.completed
- **And** step 3 transitions to "running", executes, transitions to "completed", publishes step.started and step.completed
- **And** run transitions to "completed" and run.completed event is published
- **And** steps execute in step_order sequence (0, 1, 2)

**AC5: PipelineExecutor updates step timestamps during execution**
- **Given** a step with status "pending"
- **When** PipelineExecutor transitions step to "running"
- **Then** step.started_at is set to current timestamp
- **When** step execution completes successfully
- **Then** step.completed_at is set to current timestamp
- **And** step status is "completed"

**AC6: PipelineExecutor stops on first step failure**
- **Given** a run with 3 steps where step 2 will fail
- **When** PipelineExecutor.ExecuteRun is called
- **Then** step 1 completes successfully
- **And** step 2 transitions to "running", executes, transitions to "failed" with error_message
- **And** step.failed event is published with error details
- **And** run transitions to "failed" with error_message from step 2
- **And** run.failed event is published
- **And** step 3 remains in "pending" status (not executed)
- **And** ExecuteRun returns the error from step 2

**AC7: PipelineExecutor handles cancellation gracefully**
- **Given** a run with 3 steps in execution
- **When** context is cancelled during step 2 execution
- **Then** step 2 execution stops and transitions to "cancelled"
- **And** step.cancelled event is published
- **And** run transitions to "cancelled"
- **And** run.cancelled event is published
- **And** remaining steps stay in "pending" status
- **And** ExecuteRun returns context cancellation error

**AC8: PipelineExecutor publishes events at each state transition**
- **Given** a run executing successfully
- **When** execution progresses through each step
- **Then** events are published in order: run.started, step.started (step 1), step.completed (step 1), step.started (step 2), step.completed (step 2), ..., run.completed
- **And** each event payload includes run_id, step_id (for step events), status, timestamp
- **And** failure events include error_message in payload

**AC9: Unit tests verify PipelineExecutor behavior**
- **Given** unit tests in `backend/internal/domain/service/pipeline_executor_test.go`
- **When** tests are executed
- **Then** happy path test verifies all steps execute in order, run completes
- **And** step failure test verifies execution stops, run fails, remaining steps skipped
- **And** cancellation test verifies context cancellation stops execution, marks step/run cancelled
- **And** event publishing test verifies all events published in correct order
- **And** timestamp test verifies started_at and completed_at set correctly
- **And** all tests use mock RunRepository, ActionRegistry, EventPublisher

## Tasks / Subtasks

- [ ] [BACK] Task 1: Define Action interface and RunContext model (AC: #1, #2)
  - [ ] Create `backend/internal/domain/model/action.go` with Action interface
  - [ ] Define Name() string method for action identifier
  - [ ] Define Execute(ctx context.Context, runCtx *RunContext) error method
  - [ ] Create `backend/internal/domain/model/run_context.go` with RunContext struct
  - [ ] Add fields: Run, RunStep, ProjectID, StoryID, Metadata
  - [ ] Document interfaces and structs with godoc comments

- [ ] [BACK] Task 2: Create ActionRegistry port interface (AC: #3)
  - [ ] Create `backend/internal/domain/port/action_registry.go`
  - [ ] Define ActionRegistry interface with Register and Get methods
  - [ ] Document expected behavior for Register (idempotent, overwrites existing)
  - [ ] Document expected error for Get when action not found (ACTION_NOT_FOUND)
  - [ ] Add godoc comments for interface and methods

- [ ] [BACK] Task 3: Implement PipelineExecutor service structure (AC: #4)
  - [ ] Create `backend/internal/domain/service/pipeline_executor.go`
  - [ ] Define PipelineExecutor struct with dependencies: RunRepository, ActionRegistry, EventPublisher, logger
  - [ ] Implement NewPipelineExecutor constructor
  - [ ] Add ExecuteRun(ctx context.Context, runID uuid.UUID) error method signature

- [ ] [BACK] Task 4: Implement run-level state transitions and event publishing (AC: #4, #8)
  - [ ] In ExecuteRun: fetch run with steps from RunRepository
  - [ ] Transition run to "running", set started_at timestamp
  - [ ] Publish run.started event with payload: run_id, status, started_at
  - [ ] On success: transition run to "completed", set completed_at, publish run.completed
  - [ ] On failure: transition run to "failed", set error_message, publish run.failed
  - [ ] On cancellation: transition run to "cancelled", publish run.cancelled

- [ ] [BACK] Task 5: Implement step-level execution loop (AC: #4, #5)
  - [ ] Sort steps by step_order (ascending)
  - [ ] For each step: transition to "running", set started_at, publish step.started
  - [ ] Lookup action from ActionRegistry by step.action name
  - [ ] Build RunContext with Run, RunStep, ProjectID, StoryID, Metadata
  - [ ] Execute action.Execute(ctx, runCtx)
  - [ ] On success: transition step to "completed", set completed_at, publish step.completed
  - [ ] Update run Metadata with any results from step execution (if action returns data)

- [ ] [BACK] Task 6: Implement step failure handling (AC: #6, #8)
  - [ ] On step action.Execute error: transition step to "failed", set error_message
  - [ ] Publish step.failed event with error_message in payload
  - [ ] Transition run to "failed", set error_message from step
  - [ ] Publish run.failed event
  - [ ] Return error immediately (do not execute remaining steps)
  - [ ] Ensure remaining steps stay in "pending" status

- [ ] [BACK] Task 7: Implement cancellation support (AC: #7, #8)
  - [ ] Check ctx.Done() before each step execution
  - [ ] On ctx.Done(): transition current step to "cancelled", publish step.cancelled
  - [ ] Transition run to "cancelled", publish run.cancelled
  - [ ] Return context cancellation error
  - [ ] Ensure partial execution is tracked (completed steps stay completed)

- [ ] [BACK] Task 8: Write unit tests for PipelineExecutor (AC: #9)
  - [ ] Create `backend/internal/domain/service/pipeline_executor_test.go`
  - [ ] Test happy path: 3 steps all succeed, run completes, events published in order
  - [ ] Test step failure: step 2 fails, run fails, step 3 not executed
  - [ ] Test cancellation: context cancelled during step 2, step/run marked cancelled
  - [ ] Test event publishing: verify all events (run.started, step.started, step.completed, run.completed)
  - [ ] Test timestamps: verify started_at and completed_at set correctly
  - [ ] Use mock RunRepository, mock ActionRegistry, mock EventPublisher
  - [ ] Verify no remaining steps executed after failure or cancellation

- [ ] [BACK] Task 9: Write unit tests for ActionRegistry (AC: #3)
  - [ ] Create mock ActionRegistry implementation for testing
  - [ ] Test Register: registers action, allows lookup by name
  - [ ] Test Get: returns registered action
  - [ ] Test Get with unknown action: returns ACTION_NOT_FOUND error
  - [ ] Test Register overwrites existing action with same name
  - [ ] Verify mock can be used in PipelineExecutor tests

## Dev Notes

### Dependencies

**Story 3-1 (Runs & RunSteps tables + state machine):** PipelineExecutor depends on Run and RunStep domain models, RunRepository port, and state machine transition validation (ValidateRunTransition, ValidateStepTransition). The executor must use these to ensure valid state transitions.

**Story 3-4 (Docker container lifecycle):** ContainerManager port exists but is NOT used directly by PipelineExecutor. Instead, concrete Action implementations (e.g., AgentRunAction in future story) will use ContainerManager. PipelineExecutor only knows about the Action interface.

**Story 3-6 (Events table + event bus):** EventPublisher port must exist for publishing run/step lifecycle events (run.started, step.started, step.completed, step.failed, run.completed, run.failed, run.cancelled, step.cancelled).

**Future stories:** Story 3-8 will implement concrete Action types (AgentRunAction, GitCreatePRAction, etc.). This story only defines the Action interface and PipelineExecutor framework.

### Architecture Requirements

**Hexagonal architecture:**
- PipelineExecutor is a domain service in `backend/internal/domain/service/`
- Depends ONLY on ports: RunRepository, ActionRegistry, EventPublisher
- NO imports from adapter/ or api/ layers
- Action is a domain interface — concrete implementations live in domain/action/ (future story)
- ActionRegistry is a port — concrete in-memory implementation in adapter/registry/ (future story)

**State machine integration:**
- PipelineExecutor uses state transition validation from Story 3-1
- All status changes go through RunRepository which validates transitions
- Invalid transitions return INVALID_STATE_TRANSITION error

**Event publishing:**
- Events published AFTER successful state transition (not before)
- Event payload includes all relevant context (run_id, step_id, status, timestamp, error_message)
- Failed event publishing logged but does NOT fail execution (fire-and-forget)

### File Paths (exact)

```
backend/internal/domain/model/action.go               # Action interface
backend/internal/domain/model/run_context.go          # RunContext struct
backend/internal/domain/port/action_registry.go       # ActionRegistry port interface
backend/internal/domain/service/pipeline_executor.go  # PipelineExecutor service
backend/internal/domain/service/pipeline_executor_test.go # Unit tests
```

### Technical Specifications

**Action interface:**
```go
// backend/internal/domain/model/action.go
package model

import "context"

// Action represents a pipeline step action (e.g., agent_run, git_create_pr, hitl_gate).
// Concrete implementations handle specific action types.
type Action interface {
    // Name returns the action identifier matching pipeline config step action field.
    // Examples: "agent_run", "git_create_pr", "hitl_gate"
    Name() string

    // Execute runs the action with the given run context.
    // Returns nil on success, error on failure.
    // The error will be stored in run_step.error_message and cause run failure.
    Execute(ctx context.Context, runCtx *RunContext) error
}
```

**RunContext model:**
```go
// backend/internal/domain/model/run_context.go
package model

import "github.com/google/uuid"

// RunContext provides context for action execution.
// It carries the current run, step, and shared metadata across the pipeline.
type RunContext struct {
    // Run is the current pipeline run
    Run *Run

    // RunStep is the current step being executed
    RunStep *RunStep

    // ProjectID is the ID of the project owning this run
    ProjectID uuid.UUID

    // StoryID is the ID of the story being processed
    StoryID uuid.UUID

    // Metadata holds inter-step data (e.g., branch name, PR URL)
    // Previous steps can write to this map, later steps can read from it
    Metadata map[string]any
}
```

**ActionRegistry port:**
```go
// backend/internal/domain/port/action_registry.go
package port

import "github.com/hopeitworks/backend/internal/domain/model"

// ActionRegistry manages registration and lookup of pipeline step actions.
type ActionRegistry interface {
    // Register registers an action by its name.
    // If an action with the same name exists, it is overwritten.
    Register(action model.Action)

    // Get retrieves an action by name.
    // Returns ACTION_NOT_FOUND error if action is not registered.
    Get(name string) (model.Action, error)
}
```

**PipelineExecutor service:**
```go
// backend/internal/domain/service/pipeline_executor.go
package service

import (
    "context"
    "fmt"
    "log/slog"
    "sort"
    "time"

    "github.com/google/uuid"

    "github.com/hopeitworks/backend/internal/domain/model"
    "github.com/hopeitworks/backend/internal/domain/port"
    "github.com/hopeitworks/backend/pkg/errors"
)

// PipelineExecutor orchestrates sequential execution of pipeline steps.
type PipelineExecutor struct {
    runRepo   port.RunRepository
    actionReg port.ActionRegistry
    eventPub  port.EventPublisher
    logger    *slog.Logger
}

// NewPipelineExecutor creates a new pipeline executor.
func NewPipelineExecutor(
    runRepo port.RunRepository,
    actionReg port.ActionRegistry,
    eventPub port.EventPublisher,
    logger *slog.Logger,
) *PipelineExecutor {
    return &PipelineExecutor{
        runRepo:   runRepo,
        actionReg: actionReg,
        eventPub:  eventPub,
        logger:    logger,
    }
}

// ExecuteRun executes all steps of a run sequentially.
// Steps execute in step_order sequence. Execution stops on first failure or cancellation.
func (e *PipelineExecutor) ExecuteRun(ctx context.Context, runID uuid.UUID) error {
    // 1. Get run with steps
    run, err := e.runRepo.GetRun(ctx, runID)
    if err != nil {
        return err
    }

    steps, err := e.runRepo.ListRunStepsByRun(ctx, runID)
    if err != nil {
        return err
    }

    // Sort steps by step_order
    sort.Slice(steps, func(i, j int) bool {
        return steps[i].StepOrder < steps[j].StepOrder
    })

    // 2. Transition run to "running", publish run.started
    now := time.Now()
    if err := e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusRunning, nil); err != nil {
        return err
    }
    run.Status = model.RunStatusRunning
    run.StartedAt = &now

    e.publishEvent(ctx, "run.started", map[string]any{
        "run_id":     runID.String(),
        "status":     string(model.RunStatusRunning),
        "started_at": now,
    })

    // 3. Execute each step in order
    metadata := make(map[string]any)
    for _, step := range steps {
        select {
        case <-ctx.Done():
            // Context cancelled — mark current step and run as cancelled
            e.handleCancellation(ctx, run, step)
            return ctx.Err()
        default:
            // Continue execution
        }

        if err := e.executeStep(ctx, run, step, metadata); err != nil {
            // Step failed — mark run as failed and stop
            e.handleStepFailure(ctx, run, step, err)
            return err
        }
    }

    // 4. All steps completed — mark run as completed
    completedAt := time.Now()
    if err := e.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusCompleted, nil); err != nil {
        return err
    }

    e.publishEvent(ctx, "run.completed", map[string]any{
        "run_id":       runID.String(),
        "status":       string(model.RunStatusCompleted),
        "completed_at": completedAt,
    })

    return nil
}

// executeStep executes a single pipeline step.
func (e *PipelineExecutor) executeStep(ctx context.Context, run *model.Run, step *model.RunStep, metadata map[string]any) error {
    // Transition step to "running"
    startedAt := time.Now()
    if err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusRunning, nil); err != nil {
        return err
    }
    step.Status = model.StepStatusRunning
    step.StartedAt = &startedAt

    e.publishEvent(ctx, "step.started", map[string]any{
        "run_id":     run.ID.String(),
        "step_id":    step.ID.String(),
        "step_name":  step.StepName,
        "action":     step.Action,
        "status":     string(model.StepStatusRunning),
        "started_at": startedAt,
    })

    // Lookup action
    action, err := e.actionReg.Get(step.Action)
    if err != nil {
        return fmt.Errorf("action not found: %s: %w", step.Action, err)
    }

    // Build run context
    runCtx := &model.RunContext{
        Run:       run,
        RunStep:   step,
        ProjectID: run.ProjectID,
        StoryID:   run.StoryID,
        Metadata:  metadata,
    }

    // Execute action
    if err := action.Execute(ctx, runCtx); err != nil {
        return err
    }

    // Transition step to "completed"
    completedAt := time.Now()
    if err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCompleted, nil); err != nil {
        return err
    }
    step.Status = model.StepStatusCompleted
    step.CompletedAt = &completedAt

    e.publishEvent(ctx, "step.completed", map[string]any{
        "run_id":       run.ID.String(),
        "step_id":      step.ID.String(),
        "step_name":    step.StepName,
        "status":       string(model.StepStatusCompleted),
        "completed_at": completedAt,
    })

    return nil
}

// handleStepFailure marks step and run as failed, publishes events.
func (e *PipelineExecutor) handleStepFailure(ctx context.Context, run *model.Run, step *model.RunStep, stepErr error) {
    errMsg := stepErr.Error()

    // Mark step as failed
    if err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusFailed, &errMsg); err != nil {
        e.logger.Error("failed to update step status to failed", "step_id", step.ID, "error", err)
    }

    e.publishEvent(ctx, "step.failed", map[string]any{
        "run_id":        run.ID.String(),
        "step_id":       step.ID.String(),
        "step_name":     step.StepName,
        "status":        string(model.StepStatusFailed),
        "error_message": errMsg,
    })

    // Mark run as failed
    if err := e.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusFailed, &errMsg); err != nil {
        e.logger.Error("failed to update run status to failed", "run_id", run.ID, "error", err)
    }

    e.publishEvent(ctx, "run.failed", map[string]any{
        "run_id":        run.ID.String(),
        "status":        string(model.RunStatusFailed),
        "error_message": errMsg,
    })
}

// handleCancellation marks step and run as cancelled, publishes events.
func (e *PipelineExecutor) handleCancellation(ctx context.Context, run *model.Run, step *model.RunStep) {
    cancelMsg := "execution cancelled"

    // Mark step as cancelled
    if err := e.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCancelled, &cancelMsg); err != nil {
        e.logger.Error("failed to update step status to cancelled", "step_id", step.ID, "error", err)
    }

    e.publishEvent(ctx, "step.cancelled", map[string]any{
        "run_id":    run.ID.String(),
        "step_id":   step.ID.String(),
        "step_name": step.StepName,
        "status":    string(model.StepStatusCancelled),
    })

    // Mark run as cancelled
    if err := e.runRepo.UpdateRunStatus(ctx, run.ID, model.RunStatusCancelled, &cancelMsg); err != nil {
        e.logger.Error("failed to update run status to cancelled", "run_id", run.ID, "error", err)
    }

    e.publishEvent(ctx, "run.cancelled", map[string]any{
        "run_id": run.ID.String(),
        "status": string(model.RunStatusCancelled),
    })
}

// publishEvent publishes an event, logging errors without failing execution.
func (e *PipelineExecutor) publishEvent(ctx context.Context, eventType string, payload map[string]any) {
    if err := e.eventPub.Publish(ctx, eventType, payload); err != nil {
        e.logger.Error("failed to publish event", "event_type", eventType, "error", err)
    }
}
```

**Error codes:**
- `ACTION_NOT_FOUND` — action not registered in ActionRegistry
- `INVALID_STATE_TRANSITION` — from Story 3-1 state machine (reused)
- `RUN_NOT_FOUND` — run doesn't exist (from RunRepository)

**Event types (dot notation):**
- `run.started` — run transitioned to "running"
- `run.completed` — run transitioned to "completed" (all steps succeeded)
- `run.failed` — run transitioned to "failed" (step failed)
- `run.cancelled` — run transitioned to "cancelled" (context cancelled)
- `step.started` — step transitioned to "running"
- `step.completed` — step transitioned to "completed"
- `step.failed` — step transitioned to "failed"
- `step.cancelled` — step transitioned to "cancelled"

**Event payload schema:**
```json
{
  "run_id": "uuid",
  "step_id": "uuid",         // Only in step events
  "step_name": "string",     // Only in step events
  "action": "string",        // Only in step.started
  "status": "string",
  "started_at": "timestamp", // Only in .started events
  "completed_at": "timestamp", // Only in .completed events
  "error_message": "string"  // Only in .failed events
}
```

### Testing Requirements

**Unit tests (backend/internal/domain/service/pipeline_executor_test.go):**

1. **Happy path test:**
   - Run with 3 steps, all actions succeed
   - Verify steps execute in step_order (0, 1, 2)
   - Verify run transitions: pending → running → completed
   - Verify each step transitions: pending → running → completed
   - Verify started_at and completed_at timestamps set
   - Verify events published in order: run.started, step.started (0), step.completed (0), step.started (1), step.completed (1), step.started (2), step.completed (2), run.completed

2. **Step failure test:**
   - Run with 3 steps, step 1 (index 1) action returns error
   - Verify step 0 completes successfully
   - Verify step 1 transitions: pending → running → failed
   - Verify step 1 error_message set
   - Verify run transitions to "failed" with error_message from step 1
   - Verify step 2 stays in "pending" (not executed)
   - Verify events: run.started, step.started (0), step.completed (0), step.started (1), step.failed (1), run.failed

3. **Cancellation test:**
   - Run with 3 steps, cancel context during step 1 execution
   - Verify step 0 completes successfully
   - Verify step 1 transitions to "cancelled"
   - Verify run transitions to "cancelled"
   - Verify step 2 stays in "pending" (not executed)
   - Verify events: run.started, step.started (0), step.completed (0), step.started (1), step.cancelled (1), run.cancelled

4. **Event publishing test:**
   - Verify all events published in correct order
   - Verify event payloads contain correct fields (run_id, step_id, status, timestamps)
   - Verify error_message included in .failed events

5. **Timestamp test:**
   - Verify started_at set when transitioning to "running"
   - Verify completed_at set when transitioning to "completed"
   - Verify timestamps are non-nil and reasonable (within test execution window)

6. **ActionRegistry test:**
   - Test Register and Get
   - Test Get with unknown action returns ACTION_NOT_FOUND error
   - Test Register overwrites existing action

**Mocks used:**
- Mock RunRepository: tracks UpdateRunStatus and UpdateRunStepStatus calls
- Mock ActionRegistry: returns configured actions by name
- Mock EventPublisher: tracks Publish calls for verification
- Mock Action: configurable Name() and Execute() behavior (success, error, delay)

**Linting:**
- Run `golangci-lint run ./...` — must pass before commit

### References

- Story 3-1: Runs & RunSteps tables + state machine (Run, RunStep models, RunRepository, state transitions)
- Story 3-4: Docker container lifecycle (ContainerManager port — used by future Action implementations)
- Story 3-6: Events table + event bus (EventPublisher port)
- Epic 3 Planning: Pipeline execution architecture
- `backend/.golangci.yml`: Linting rules
- `backend/pkg/errors/domain.go`: DomainError implementation

## Dev Agent Record

(To be filled during implementation)

## Change Log

- 2026-02-17: Story created for Wave 5 pipeline execution engine
