# Story R-2-4: [BACK] Backend action: human (extends HITL)

Status: ready-for-dev

## Story

As a **pipeline executor**,
I want a `human` action that suspends a pipeline step pending explicit human approval,
so that pipelines can gate arbitrary steps on human review — not just code review gates.

## Acceptance Criteria (BDD)

### Scenario 1: Human action creates HITLRequest and suspends step

```gherkin
Given a pipeline step with action_type "human" and config:
  | message      | "Please review the generated plan for {story_key}" |
  | instructions | "Check that all acceptance criteria are addressed"  |
And RunContext contains story_key "S-05"
When the action executes
Then HITLRepository.Create is called with a HITLRequest where:
  | run_step_id | <current run step UUID>       |
  | gate_type   | "human"                       |
  | status      | "pending"                     |
  And the rendered message is stored as a custom field or in RejectionReason placeholder (see technical spec)
  And RunRepository.UpdateRunStepStatus is called with status "waiting_approval"
  And EventPublisher.Publish is called with entity_type "human", action "pending"
  And the action returns nil
```

### Scenario 2: Message template rendered with RunContext variables

```gherkin
Given config message "Approve work on {story_key} — branch: {branch_name}"
And RunContext.Metadata["branch_name"] is "feat/S-05-plan"
When the action executes
Then the HITL request is created with the rendered message stored in the payload
```

### Scenario 3: Default message when not configured

```gherkin
Given a pipeline step with no "message" config key
When the action executes
Then HITLRepository.Create is called with a default message "Human approval required for step {step_name}"
```

### Scenario 4: Action is registered in ActionRegistry

```gherkin
Given the application has started
When ActionRegistry.Get("human") is called
Then the HumanAction is returned without error
```

### Scenario 5: HITLRepository failure returns error (step not suspended)

```gherkin
Given HITLRepository.Create returns an error
When the action executes
Then the action returns the error
  And RunRepository.UpdateRunStepStatus is NOT called
```

### Scenario 6: Lint passes

```gherkin
Given the implementation in backend/internal/adapter/action/human.go
When "golangci-lint run ./..." is executed from backend/
Then it exits 0
```

## Tasks / Subtasks

- [ ] [BACK] Task 1: Implement HumanAction (AC: #1, #2, #3, #5)
  - [ ] Create `backend/internal/adapter/action/human.go`
  - [ ] Define `HumanAction` struct with fields:
    - `hitlRepo port.HITLRepository`
    - `runRepo port.RunRepository`
    - `storyRepo port.StoryRepository`
    - `eventPub port.EventPublisher`
    - `logger *slog.Logger`
  - [ ] Implement `NewHumanAction(hitlRepo, runRepo, storyRepo, eventPub, logger) *HumanAction`
  - [ ] Implement `Name() string` returning `"human"`
  - [ ] Implement `Execute(ctx context.Context, runCtx *model.RunContext) error`:
    - Read `message` from `runCtx.RunStep.Config`; default to `"Human approval required for step {step_name}"`
    - Read `instructions` from config (optional, may be empty)
    - Fetch story with `storyRepo.GetByID(ctx, runCtx.StoryID)` for event payload
    - Build template vars: `story_key`, `step_name`, `branch_name` (from metadata), `pr_url` (from metadata)
    - Render both `message` and `instructions` using `renderTemplate`
    - Create `HITLRequest`:
      - `ID`: `uuid.New()`
      - `RunStepID`: `runCtx.RunStep.ID`
      - `GateType`: `"human"`
      - `DiffContent`: nil (human gate does not fetch PR diff)
      - `Status`: `model.HITLStatusPending`
      - `CreatedAt`: `time.Now()`
    - Call `hitlRepo.Create(ctx, req)` — return error on failure (step 5)
    - Call `runRepo.UpdateRunStepStatus(ctx, runCtx.RunStep.ID, model.StepStatusWaitingApproval, &now, nil, nil)` — return error on failure
    - Publish `human.pending` event via `publishHumanPendingEvent`
    - Return nil

- [ ] [BACK] Task 2: Implement publishHumanPendingEvent (AC: #1)
  - [ ] Private method on `HumanAction`
  - [ ] Payload: `{ "run_id": "...", "step_id": "...", "story_key": "...", "hitl_request_id": "...", "message": "...", "instructions": "..." }`
  - [ ] Event: `entity_type = "human"`, `action = "pending"`, `entity_id = runCtx.RunStep.ID`
  - [ ] Log event publish errors at Warn (non-fatal for the step suspension itself)

- [ ] [BACK] Task 3: Extend HITLRequest model if needed (AC: #1, #2)
  - [ ] Check if `model.HITLRequest` has a `Message` field; if not, check if the rendered message should be stored in `RejectionReason` (repurposed as a generic notes field) or if a new `Message *string` field must be added to the model and corresponding DB migration
  - [ ] If a new field is needed, add it to `model.HITLRequest`, create a migration `backend/migrations/<next>_add_hitl_message.up.sql`, add the column, and update the sqlc query for `HITLRepository.Create`
  - [ ] Preferred approach: add `Message *string` to `HITLRequest` so the human gate message is first-class

- [ ] [BACK] Task 4: Register HumanAction in ActionRegistry (AC: #4)
  - [ ] In `backend/cmd/api/main.go` or DI wire file, instantiate `NewHumanAction` and call `actionRegistry.Register(humanAction)`

- [ ] [BACK] Task 5: Write unit tests (AC: #1–#3, #5, #6)
  - [ ] Create `backend/internal/adapter/action/__tests__/human_test.go`
  - [ ] **Test: happy path** — config has `message` and `instructions`, mocks all succeed, verify HITLRequest created with GateType `"human"`, step transitioned to `waiting_approval`, event published
  - [ ] **Test: message template rendering** — `{story_key}` and `{branch_name}` replaced from context
  - [ ] **Test: default message** — no message config key → default used, HITL created
  - [ ] **Test: HITLRepository failure** — Create returns error → action returns error, UpdateRunStepStatus NOT called
  - [ ] **Test: UpdateRunStepStatus failure** — returns error → action returns error
  - [ ] **Test: EventPublisher failure** — Publish fails → action still returns nil (event failure non-fatal)
  - [ ] All mocks hand-written, unused params renamed to `_`
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

- **R-1-3 (action_types + step config plumbing) — required:** `human` must be in the action type enum.
- **Story 6-1 (HITL gates — HITLRepository, HITLRequest model) — DONE:** `HITLRepository.Create`, `HITLRepository.GetByRunStep`, `model.HITLRequest`, `model.HITLStatusPending` are all in place.
- **Story 3-7 (PipelineExecutor) — DONE:** `model.Action`, `RunContext`, `ActionRegistry` in place.
- `hitl_gate.go` — existing HITL action; `HumanAction` reuses the same mechanism with `GateType = "human"` instead of `"approval"`.

### Architecture Requirements

- `HumanAction` is an adapter in `backend/internal/adapter/action/`. It mirrors `HITLGateAction` closely.
- Key difference from `HITLGateAction`: no PR diff fetch, custom message, `GateType = "human"`, event type `"human"` (not `"hitl_gate"`).
- The existing HITL approval/rejection API endpoints (from Story 6-x) are reused to resume a human-gated step — no new API endpoints needed.
- The `waiting_approval` step status transition is shared with `hitl_gate`. The HITL request with `GateType = "human"` distinguishes which type of gate is pending.
- Do not import `hitl_gate.go` — duplicate the struct and logic to keep the two actions independently evolvable.

### Technical Specifications

**File:** `backend/internal/adapter/action/human.go`

```go
package action

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// HumanAction implements model.Action for suspending a pipeline step
// pending explicit human approval. Unlike HITLGateAction, it does not
// fetch a PR diff — it presents a configurable message and optional instructions
// to the reviewer.
type HumanAction struct {
    hitlRepo  port.HITLRepository
    runRepo   port.RunRepository
    storyRepo port.StoryRepository
    eventPub  port.EventPublisher
    logger    *slog.Logger
}

func NewHumanAction(
    hitlRepo port.HITLRepository,
    runRepo port.RunRepository,
    storyRepo port.StoryRepository,
    eventPub port.EventPublisher,
    logger *slog.Logger,
) *HumanAction {
    return &HumanAction{
        hitlRepo:  hitlRepo,
        runRepo:   runRepo,
        storyRepo: storyRepo,
        eventPub:  eventPub,
        logger:    logger,
    }
}

func (a *HumanAction) Name() string { return "human" }

func (a *HumanAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    cfg := runCtx.RunStep.Config

    msgTemplate, ok := cfg["message"]
    if !ok || msgTemplate == "" {
        msgTemplate = "Human approval required for step {step_name}"
    }
    instructions := cfg["instructions"]

    story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
    if err != nil {
        return fmt.Errorf("fetch story: %w", err)
    }

    vars := map[string]string{
        "story_key":   story.Key,
        "step_name":   runCtx.RunStep.StepName,
        "branch_name": func() string { s, _ := runCtx.Metadata["branch_name"].(string); return s }(),
        "pr_url":      func() string { s, _ := runCtx.Metadata["pr_url"].(string); return s }(),
    }

    message := renderTemplate(msgTemplate, vars)
    renderedInstructions := renderTemplate(instructions, vars)

    req := &model.HITLRequest{
        ID:          uuid.New(),
        RunStepID:   runCtx.RunStep.ID,
        GateType:    "human",
        DiffContent: nil,
        Status:      model.HITLStatusPending,
        CreatedAt:   time.Now(),
    }
    // Store rendered message in Message field (requires Task 3 model extension).
    // If model.HITLRequest.Message does not exist yet, add it before this action.
    req.Message = &message

    created, err := a.hitlRepo.Create(ctx, req)
    if err != nil {
        return fmt.Errorf("create HITL request: %w", err)
    }

    now := time.Now()
    if _, err := a.runRepo.UpdateRunStepStatus(ctx, runCtx.RunStep.ID,
        model.StepStatusWaitingApproval, &now, nil, nil); err != nil {
        return fmt.Errorf("update step to waiting_approval: %w", err)
    }

    a.publishHumanPendingEvent(ctx, runCtx, story.Key, created.ID, message, renderedInstructions)

    return nil
}

func (a *HumanAction) publishHumanPendingEvent(
    ctx context.Context,
    runCtx *model.RunContext,
    storyKey string,
    hitlRequestID uuid.UUID,
    message, instructions string,
) {
    payload := map[string]string{
        "run_id":          runCtx.Run.ID.String(),
        "step_id":         runCtx.RunStep.ID.String(),
        "story_key":       storyKey,
        "hitl_request_id": hitlRequestID.String(),
        "message":         message,
        "instructions":    instructions,
    }
    payloadJSON, err := json.Marshal(payload)
    if err != nil {
        a.logger.Error("failed to marshal human.pending payload", "error", err)
        return
    }

    event := model.Event{
        ID:         uuid.New(),
        ProjectID:  runCtx.ProjectID,
        EntityType: "human",
        EntityID:   runCtx.RunStep.ID,
        Action:     "pending",
        Payload:    payloadJSON,
    }

    if err := a.eventPub.Publish(ctx, event); err != nil {
        a.logger.Error("failed to publish human.pending event", "error", err)
    }
}
```

**Model extension — `HITLRequest.Message`:**

If not already present, add to `backend/internal/domain/model/hitl.go`:

```go
// HITLRequest records a human-in-the-loop gate triggered by a pipeline step.
type HITLRequest struct {
    // ...existing fields...
    Message *string // optional human-readable message for the reviewer
}
```

Create migration `backend/migrations/<next>_add_hitl_request_message.up.sql`:

```sql
ALTER TABLE hitl_requests ADD COLUMN message text;
```

Down migration:

```sql
ALTER TABLE hitl_requests DROP COLUMN message;
```

Update `backend/queries/hitl.sql` to include the `message` column in the INSERT query, then regenerate with `sqlc generate`.

**Event schema (human.pending):**
- `entity_type`: `"human"`
- `action`: `"pending"`
- `entity_id`: run step UUID
- `payload`: `{ "run_id": "...", "step_id": "...", "story_key": "...", "hitl_request_id": "...", "message": "...", "instructions": "..." }`

**Resuming a human-gated step:** The existing HITL approval/rejection endpoints (POST `/hitl-requests/{id}/approve` and `/reject`) are reused unchanged. When the HITL request is approved, the pipeline executor resumes the step (transitions to `running` or `completed` depending on existing HITL resolution logic).

### Testing Requirements

File: `backend/internal/adapter/action/__tests__/human_test.go`

**Tests:**

1. **Happy path** — all mocks succeed, verify: HITLRequest has GateType `"human"`, `Status = "pending"`, step transitioned to `"waiting_approval"`, `human.pending` event published with correct payload.
2. **Message rendering** — `{story_key}` and `{branch_name}` replaced; verify `Message` field on created HITLRequest.
3. **Default message** — no message in config → default `"Human approval required for step {step_name}"` used.
4. **HITLRepository.Create failure** — returns error → action error, `UpdateRunStepStatus` NOT called, event NOT published.
5. **UpdateRunStepStatus failure** — returns error → action error.
6. **EventPublisher failure** — action still returns nil.
7. **Story not found** — storyRepo returns error → action returns error (story key is required for event payload).

Mock pattern mirrors `hitl_gate_test.go` if it exists. Run `golangci-lint run ./...` before committing.

### References

- `backend/internal/adapter/action/hitl_gate.go` — reference implementation (same pattern, different GateType)
- `backend/internal/domain/model/hitl.go` — `HITLRequest`, `HITLStatus` constants
- `backend/internal/domain/port/hitl_repository.go` — `HITLRepository.Create`
- `backend/internal/domain/port/run_repository.go` — `RunRepository.UpdateRunStepStatus`
- `backend/internal/domain/port/event_publisher.go` — `EventPublisher.Publish`
- `backend/internal/domain/model/run_context.go` — `RunContext`
- `backend/migrations/` — next available migration number for adding `hitl_requests.message`
- Story 6-1 — HITLRepository and approval/rejection flow (already implemented)

## Dev Agent Record

## Change Log

- 2026-02-23: Story created for Wave R implementation
