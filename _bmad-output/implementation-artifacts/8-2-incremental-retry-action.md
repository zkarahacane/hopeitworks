# Story 8.2: [BACK] Incremental Retry Action

Status: ready-for-dev

## Story

As a pipeline executor, I want an `incremental_retry` action that re-runs a failed agent step with error context injected into the prompt, so that the agent can fix its own mistakes without restarting from scratch.

## Acceptance Criteria (BDD)

**AC1: DB migration adds retry fields to run_steps**
- **Given** migration `000013_add_retry_fields_to_run_steps` is applied
- **When** the `run_steps` table is inspected
- **Then** it has columns: `retry_count INT NOT NULL DEFAULT 0`, `retry_type VARCHAR(50) NULL`, `parent_step_id UUID NULL REFERENCES run_steps(id)`
- **And** an index `idx_run_steps_parent_step_id` exists on `parent_step_id`
- **And** `RunStep` domain model and sqlc queries reflect these new fields

**AC2: Incremental retry creates a new RunStep with error context**
- **Given** a failed `RunStep` with non-empty `ErrorMessage` and `LogTail`
- **When** the `incremental_retry` action executes
- **Then** a new `RunStep` is created with `retry_count = parent.retry_count + 1`, `retry_type = "incremental"`, `parent_step_id = parent.ID`
- **And** the new step's metadata carries `error_context` (from parent `ErrorMessage`) and `log_tail` (from parent `LogTail`)

**AC3: Incremental retry uses the `implement-retry` Handlebars template**
- **Given** the new retry RunStep is being executed
- **When** the `TemplateService.RenderForStory` is called
- **Then** it is called with template name `"implement-retry"`
- **And** the rendered prompt includes `error_context` and `log_tail` via `TemplateContext.ErrorContext` and `TemplateContext.LogTail`

**AC4: After 2 incremental failures, fall back to full retry**
- **Given** `parent.retry_count >= 2` and `parent.retry_type = "incremental"`
- **When** the coordinator evaluates the retry policy
- **Then** it creates a new step with `retry_type = "full"` and uses the `"implement"` template (no error context injection)
- **And** the full retry step's `parent_step_id` points to the original (root) failed step

**AC5: Max retries enforced from pipeline config**
- **Given** the pipeline step's `retry_policy.max_retries` is set (default 3)
- **When** `parent.retry_count >= max_retries`
- **Then** the action returns an error with code `RETRY_MAX_EXCEEDED` without creating a new step

**AC6: ActionRegistry registration**
- **Given** the `IncrementalRetryAction` is implemented
- **When** the application starts
- **Then** the action is registered in `ActionRegistry` with name `"incremental_retry"`

**AC7: Unit tests cover all retry branches**
- **Given** unit tests in `backend/internal/adapter/action/__tests__/incremental_retry_test.go`
- **When** tests run
- **Then** happy path (first retry), second incremental failure → full retry, max retries exceeded, missing parent step, and template render failure are all tested

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add DB migration for retry fields (AC: #1)
  - [ ] Create `backend/migrations/000013_add_retry_fields_to_run_steps.up.sql`
  - [ ] Add `retry_count INT NOT NULL DEFAULT 0` to `run_steps`
  - [ ] Add `retry_type VARCHAR(50) NULL CHECK (retry_type IN ('incremental', 'full'))` to `run_steps`
  - [ ] Add `parent_step_id UUID NULL REFERENCES run_steps(id) ON DELETE SET NULL` to `run_steps`
  - [ ] Add `CREATE INDEX idx_run_steps_parent_step_id ON run_steps(parent_step_id)`
  - [ ] Create `backend/migrations/000013_add_retry_fields_to_run_steps.down.sql` (DROP INDEX + DROP COLUMN x3)

- [ ] [BACK] Task 2: Update domain model and sqlc queries (AC: #1)
  - [ ] Add `RetryCount int`, `RetryType *string`, `ParentStepID *uuid.UUID` to `model.RunStep` in `backend/internal/domain/model/run.go`
  - [ ] Add sqlc queries in `backend/queries/run_steps.sql`:
    - `GetRunStepWithParent :one` — fetch step joining parent step data (or separate GetRunStep)
    - `CreateRetryRunStep :one` — INSERT with retry_count, retry_type, parent_step_id
    - `ListRetryStepsByParent :many` — SELECT WHERE parent_step_id = $1 ORDER BY retry_count
  - [ ] Run `cd backend && sqlc generate` and `make generate`
  - [ ] Update `RunRepository` port with `CreateRetryRunStep` and `ListRetryStepsByParent` methods
  - [ ] Update postgres adapter to implement the new port methods

- [ ] [BACK] Task 3: Extend `TemplateContext` with retry fields (AC: #3)
  - [ ] Add `ErrorContext string` and `LogTail string` to `model.TemplateContext` in `backend/internal/domain/model/`
  - [ ] Ensure the Handlebars `implement-retry` template (seeded by Story 6-3) can access `{{errorContext}}` and `{{logTail}}` — verify template exists in `backend/migrations/000012_seed_default_prompt_templates.up.sql`; add it if missing

- [ ] [BACK] Task 4: Implement `IncrementalRetryAction` struct and retry policy resolution (AC: #4, #5, #6)
  - [ ] Create `backend/internal/adapter/action/incremental_retry.go`
  - [ ] Define `IncrementalRetryAction` struct with deps: `runRepo port.RunRepository`, `templateSvc *service.TemplateService`, `agentRunAction *AgentRunAction`, `logger *slog.Logger`
  - [ ] Implement `Name() string` returning `"incremental_retry"`
  - [ ] Implement `resolveRetryPolicy(runCtx)` helper: reads `retry_policy.max_retries` from `runCtx.Metadata` (default 3) and `retry_policy.max_incremental` (default 2)

- [ ] [BACK] Task 5: Implement `Execute` — fetch parent step, evaluate policy, create retry step (AC: #2, #3, #4, #5)
  - [ ] Fetch parent RunStep ID from `runCtx.Metadata["parent_step_id"].(string)` — return `RETRY_MISSING_PARENT` if absent
  - [ ] Fetch parent step via `runRepo.GetRunStep(ctx, parentStepID)`
  - [ ] Check `parent.RetryCount >= maxRetries` → return `errors.NewValidation("retry", "RETRY_MAX_EXCEEDED")`
  - [ ] Determine retry type: if `parent.RetryCount < maxIncremental` → `"incremental"` else `"full"`
  - [ ] Build new `RunStep` with `retry_count = parent.RetryCount + 1`, `retry_type`, `parent_step_id = parent.ID`
  - [ ] Persist via `runRepo.CreateRetryRunStep(ctx, newStep)`
  - [ ] Build `RunContext` for the new step with updated metadata (`template_name`, `error_context`, `log_tail`)
  - [ ] Delegate execution to `agentRunAction.Execute(ctx, newRunCtx)` using the appropriate template

- [ ] [BACK] Task 6: Wire `IncrementalRetryAction` into `ActionRegistry` (AC: #6)
  - [ ] In `backend/cmd/api/wire.go`: add `NewIncrementalRetryAction` to the adapter provider set
  - [ ] Register with `actionRegistry.Register(incrementalRetryAction)` after `AgentRunAction` is registered
  - [ ] Verify `PipelineExecutor` can look it up by name `"incremental_retry"`

- [ ] [BACK] Task 7: Write unit tests (AC: #7)
  - [ ] Create `backend/internal/adapter/action/__tests__/incremental_retry_test.go`
  - [ ] **Test: first incremental retry** — parent has retry_count=0; new step created with retry_type="incremental", implement-retry template used, error_context injected
  - [ ] **Test: second failure → full retry** — parent has retry_count=2, retry_type="incremental"; new step created with retry_type="full", "implement" template used
  - [ ] **Test: max retries exceeded** — parent retry_count >= max_retries; RETRY_MAX_EXCEEDED error, no new step created
  - [ ] **Test: missing parent step ID in metadata** — RETRY_MISSING_PARENT error
  - [ ] **Test: template render failure** — TemplateService returns error; no step persisted
  - [ ] Hand-written mocks for `RunRepository`, `TemplateService`, `AgentRunAction`
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

**Story 6-3 (Handlebars rendering engine — DONE):** `TemplateService.RenderForStory` is available. The `implement-retry` template must exist in the DB (seeded by migration 000012 or added in Task 3 of this story). Template name constant: use `service.TemplateNameImplementRetry = "implement-retry"`.

**Story 3-8 (AgentRunAction — DONE):** `AgentRunAction.Execute` is the delegate that actually runs the agent container. `IncrementalRetryAction` does not re-implement container logic — it constructs a new `RunContext` and calls through to `AgentRunAction`.

**Story 3-7 (PipelineExecutor — DONE):** `model.RunContext` carries `Metadata map[string]any`, `Run *model.Run`, `RunStep *model.RunStep`, `StoryID`, `ProjectID`.

### Architecture Requirements

- `IncrementalRetryAction` is an adapter in `backend/internal/adapter/action/`
- It composes `AgentRunAction` (not inherits) — dependency injection, not embedding
- Separation of concerns: retry coordination (this action) vs. agent execution (AgentRunAction)
- The new RunStep record must be created before delegation — it acts as the audit trail
- `model.RunStep` is extended with retry fields; the DB model and domain model must stay in sync via sqlc

### File Paths (exact)

```
backend/migrations/000013_add_retry_fields_to_run_steps.up.sql
backend/migrations/000013_add_retry_fields_to_run_steps.down.sql
backend/internal/domain/model/run.go                              # Add RetryCount, RetryType, ParentStepID
backend/queries/run_steps.sql                                     # Add CreateRetryRunStep, ListRetryStepsByParent
backend/internal/domain/port/run_repository.go                   # Extend interface
backend/internal/adapter/postgres/run_repo.go                    # Implement new methods
backend/internal/adapter/action/incremental_retry.go             # New action
backend/internal/adapter/action/__tests__/incremental_retry_test.go
backend/cmd/api/wire.go                                          # DI wiring
```

### Technical Specifications

**Migration up (000013):**
```sql
ALTER TABLE run_steps
    ADD COLUMN retry_count      INT          NOT NULL DEFAULT 0,
    ADD COLUMN retry_type       VARCHAR(50)  NULL
        CHECK (retry_type IN ('incremental', 'full')),
    ADD COLUMN parent_step_id   UUID         NULL
        REFERENCES run_steps(id) ON DELETE SET NULL;

CREATE INDEX idx_run_steps_parent_step_id ON run_steps(parent_step_id);
```

**Migration down (000013):**
```sql
DROP INDEX IF EXISTS idx_run_steps_parent_step_id;
ALTER TABLE run_steps
    DROP COLUMN IF EXISTS parent_step_id,
    DROP COLUMN IF EXISTS retry_type,
    DROP COLUMN IF EXISTS retry_count;
```

**Updated RunStep model:**
```go
// RunStep represents an individual step within a pipeline run.
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
    RetryCount   int       // number of retries attempted from this step
    RetryType    *string   // "incremental" | "full" | nil (original)
    ParentStepID *uuid.UUID // nil for original steps
    CreatedAt    time.Time
}
```

**TemplateContext extension:**
```go
// In backend/internal/domain/model/ (template_context.go or similar)
type TemplateContext struct {
    // ... existing fields ...
    ErrorContext string // error message from the failed parent step
    LogTail      string // last N log lines from the failed parent step
}
```

**IncrementalRetryAction:**
```go
// IncrementalRetryAction coordinates retry logic for failed agent steps.
// It creates a new RunStep record and delegates execution to AgentRunAction.
type IncrementalRetryAction struct {
    runRepo      port.RunRepository
    templateSvc  *service.TemplateService
    agentRun     *AgentRunAction
    logger       *slog.Logger
}

func NewIncrementalRetryAction(
    runRepo port.RunRepository,
    templateSvc *service.TemplateService,
    agentRun *AgentRunAction,
    logger *slog.Logger,
) *IncrementalRetryAction

func (a *IncrementalRetryAction) Name() string { return "incremental_retry" }
```

**Execute flow:**
```go
func (a *IncrementalRetryAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    // 1. Extract parent step ID from metadata
    parentStepIDStr, _ := runCtx.Metadata["parent_step_id"].(string)
    if parentStepIDStr == "" {
        return errors.NewValidation("parent_step_id", "missing required metadata key parent_step_id")
    }
    parentStepID, err := uuid.Parse(parentStepIDStr)
    if err != nil {
        return errors.NewValidation("parent_step_id", "invalid UUID format")
    }

    // 2. Fetch parent step
    parent, err := a.runRepo.GetRunStep(ctx, parentStepID)
    if err != nil {
        return fmt.Errorf("fetch parent step: %w", err)
    }

    // 3. Resolve retry policy from metadata
    maxRetries := a.intFromMetadata(runCtx.Metadata, "retry_policy.max_retries", 3)
    maxIncremental := a.intFromMetadata(runCtx.Metadata, "retry_policy.max_incremental", 2)

    // 4. Check max retries
    if parent.RetryCount >= maxRetries {
        return errors.NewValidation("retry_count",
            fmt.Sprintf("RETRY_MAX_EXCEEDED: max %d retries reached for step %s", maxRetries, parent.ID))
    }

    // 5. Determine retry type and template
    retryType := "incremental"
    templateName := service.TemplateNameImplementRetry
    if parent.RetryCount >= maxIncremental {
        retryType = "full"
        templateName = service.TemplateNameImplement
    }

    // 6. Create new RunStep
    newStep := &model.RunStep{
        ID:           uuid.New(),
        RunID:        parent.RunID,
        StepName:     parent.StepName,
        StepOrder:    parent.StepOrder,
        Action:       parent.Action,
        Status:       model.StepStatusPending,
        RetryCount:   parent.RetryCount + 1,
        RetryType:    &retryType,
        ParentStepID: &parent.ID,
    }
    created, err := a.runRepo.CreateRetryRunStep(ctx, newStep)
    if err != nil {
        return fmt.Errorf("create retry step: %w", err)
    }

    // 7. Build new RunContext with retry metadata
    errorContext := ""
    if parent.ErrorMessage != nil {
        errorContext = *parent.ErrorMessage
    }
    logTail := ""
    if parent.LogTail != nil {
        logTail = *parent.LogTail
    }
    newMetadata := make(map[string]any, len(runCtx.Metadata))
    for k, v := range runCtx.Metadata {
        newMetadata[k] = v
    }
    newMetadata["template_name"] = templateName
    newMetadata["error_context"] = errorContext
    newMetadata["log_tail"] = logTail

    newRunCtx := &model.RunContext{
        Run:       runCtx.Run,
        RunStep:   created,
        StoryID:   runCtx.StoryID,
        ProjectID: runCtx.ProjectID,
        Metadata:  newMetadata,
    }

    // 8. Delegate to AgentRunAction
    return a.agentRun.Execute(ctx, newRunCtx)
}
```

**Metadata keys (read):**
- `parent_step_id` (required) — UUID of the failed step to retry
- `retry_policy.max_retries` (optional, default 3) — total maximum retries allowed
- `retry_policy.max_incremental` (optional, default 2) — incremental retries before switching to full

**Metadata keys (written to child RunContext):**
- `template_name` — overridden to `"implement-retry"` or `"implement"` based on retry type
- `error_context` — error message from the failed parent step
- `log_tail` — last N log lines from the failed parent step

**Error codes:**
- `RETRY_MISSING_PARENT` — `parent_step_id` metadata key absent or empty
- `RETRY_MAX_EXCEEDED` — `parent.retry_count >= max_retries`
- Propagated from `AgentRunAction.Execute` (e.g., `AGENT_FAILED`, `CONTAINER_CREATE_FAILED`)

**Template name constants (add to service package):**
```go
// backend/internal/domain/service/template_service.go (or constants file)
const (
    TemplateNameImplement      = "implement"
    TemplateNameImplementRetry = "implement-retry"
    TemplateNameReview         = "code-review"
    TemplateNameMerge          = "merge-story"
)
```

**sqlc queries to add in `backend/queries/run_steps.sql`:**
```sql
-- name: CreateRetryRunStep :one
INSERT INTO run_steps (
    id, run_id, step_name, step_order, action, status,
    retry_count, retry_type, parent_step_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: ListRetryStepsByParent :many
SELECT * FROM run_steps
WHERE parent_step_id = $1
ORDER BY retry_count ASC;
```

### Testing Requirements

**Mock RunRepository (partial — only methods used by IncrementalRetryAction):**
```go
type MockRunRepo struct {
    GetRunStepFn         func(ctx context.Context, id uuid.UUID) (*model.RunStep, error)
    CreateRetryRunStepFn func(ctx context.Context, step *model.RunStep) (*model.RunStep, error)
}
```

**Mock AgentRunAction wrapper** — create a thin `AgentRunExecutor` interface:
```go
// AgentRunExecutor is an interface to allow mocking AgentRunAction in tests.
type AgentRunExecutor interface {
    Execute(ctx context.Context, runCtx *model.RunContext) error
}
// Use this interface as the field type in IncrementalRetryAction instead of *AgentRunAction
// to allow injection of a mock in tests.
```

Test the full incremental→full retry transition by setting `parent.RetryCount = 2` and `retry_policy.max_incremental = 2`.

### References

- `backend/internal/adapter/action/agent_run.go` — delegate target
- `backend/internal/domain/model/run.go` — RunStep model (to be extended)
- `backend/internal/domain/port/run_repository.go` — RunRepository interface (to be extended)
- `backend/queries/run_steps.sql` — sqlc queries (to be extended)
- `backend/migrations/000012_seed_default_prompt_templates.up.sql` — verify `implement-retry` template exists
- `backend/internal/domain/service/template_service.go` — `RenderForStory`, `TemplateNameImplement`
- `backend/internal/domain/service/action_registry.go` — `Register` pattern
- `backend/cmd/api/wire.go` — DI wiring location

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
