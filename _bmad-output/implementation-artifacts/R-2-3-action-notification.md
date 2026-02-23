# Story R-2-3: [BACK] Backend action: notification

Status: ready-for-dev

## Story

As a **pipeline executor**,
I want a `notification` action that publishes a user-defined message as a platform event,
so that pipeline steps can surface progress updates to the SSE stream and optionally to external channels (Discord).

## Acceptance Criteria (BDD)

### Scenario 1: Notification event published on execute

```gherkin
Given a pipeline step with action_type "notification" and config:
  | message | "Story {story_key} is ready for review" |
And RunContext contains story_key "S-03"
When the action executes
Then EventPublisher.Publish is called with an event:
  | entity_type | "notification"                          |
  | action      | "sent"                                  |
  | payload.message | "Story S-03 is ready for review"    |
  | payload.step_id | <current run step UUID as string>  |
  | payload.run_id  | <current run UUID as string>       |
  And the action returns nil
```

### Scenario 2: Message template renders RunContext variables

```gherkin
Given a message template "Branch {branch_name} created for {story_key}"
And RunContext.Metadata["branch_name"] is "feat/S-03-login"
When the action executes
Then the published payload.message is "Branch feat/S-03-login created for S-03"
```

### Scenario 3: Missing message config results in a default message

```gherkin
Given a pipeline step with no "message" config key
When the action executes
Then EventPublisher.Publish is still called with a non-empty payload.message (e.g., "Step {step_name} completed")
  And the action returns nil
```

### Scenario 4: EventPublisher failure is logged but does not fail the step

```gherkin
Given EventPublisher.Publish returns an error
When the action executes
Then the error is logged at warn level with structured fields
  And the action returns nil (notification failures must not halt the pipeline)
```

### Scenario 5: Action is registered in ActionRegistry

```gherkin
Given the application has started
When ActionRegistry.Get("notification") is called
Then the NotificationAction is returned without error
```

### Scenario 6: Lint passes

```gherkin
Given the implementation in backend/internal/adapter/action/notification.go
When "golangci-lint run ./..." is executed from backend/
Then it exits 0
```

## Tasks / Subtasks

- [ ] [BACK] Task 1: Implement NotificationAction (AC: #1, #2, #3, #4)
  - [ ] Create `backend/internal/adapter/action/notification.go`
  - [ ] Define `NotificationAction` struct with fields: `eventPub port.EventPublisher`, `storyRepo port.StoryRepository`, `logger *slog.Logger`
  - [ ] Implement `NewNotificationAction(eventPub port.EventPublisher, storyRepo port.StoryRepository, logger *slog.Logger) *NotificationAction`
  - [ ] Implement `Name() string` returning `"notification"`
  - [ ] Implement `Execute(ctx context.Context, runCtx *model.RunContext) error`:
    - Read `message` from `runCtx.RunStep.Config`; default to `"Pipeline step {step_name} completed"` if absent
    - Fetch story from `storyRepo.GetByID(ctx, runCtx.StoryID)` to obtain `story.Key`
    - Build template variable map: `story_key`, `step_name` (from `runCtx.RunStep.StepName`), `run_id`, `branch_name` (from metadata, may be empty), `pr_url` (from metadata, may be empty)
    - Render message using the private `renderTemplate` helper (shared with git_pr, or duplicated if not yet in helpers.go)
    - Build event payload: `map[string]string{"message": renderedMsg, "step_id": stepID, "run_id": runID, "story_key": storyKey}`
    - Marshal payload to JSON
    - Call `eventPub.Publish(ctx, event)` with entity_type `"notification"`, action `"sent"`, entity_id = `runCtx.RunStep.ID`, project_id = `runCtx.ProjectID`
    - If `eventPub.Publish` returns error: log at Warn level with `"story_key"`, `"run_id"`, `"error"` fields — do NOT return the error
    - Return nil

- [ ] [BACK] Task 2: Register NotificationAction in ActionRegistry (AC: #5)
  - [ ] In `backend/cmd/api/main.go` or DI wire file, instantiate `NewNotificationAction` and call `actionRegistry.Register(notificationAction)`

- [ ] [BACK] Task 3: Write unit tests (AC: #1–#4, #6)
  - [ ] Create `backend/internal/adapter/action/__tests__/notification_test.go`
  - [ ] **Test: happy path** — config has `message` template, mock EventPublisher captures event, verify payload fields
  - [ ] **Test: template rendering** — `{branch_name}` and `{story_key}` replaced from context
  - [ ] **Test: missing message** — no config key → default message used, EventPublisher still called
  - [ ] **Test: EventPublisher failure** — Publish returns error → action returns nil (non-fatal), error logged at warn
  - [ ] **Test: story not found** — storyRepo returns error → decide: return error (story lookup is required for story_key variable) or degrade gracefully with empty story_key; implement consistently
  - [ ] All mocks hand-written
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

- **R-1-3 (action_types + step config plumbing) — required:** `notification` must be in the action type enum and `RunStep.Config` populated.
- **Story 8-1 (SSE event infrastructure) — DONE:** `EventPublisher.Publish` is available and wired.
- **Story 3-7 (PipelineExecutor) — DONE:** `model.Action` interface and `RunContext` are in place.

### Architecture Requirements

- `NotificationAction` is an adapter in `backend/internal/adapter/action/`. It implements `model.Action`.
- **Notification failure must never fail the pipeline step.** External notification channels are best-effort. The `eventPub.Publish` error is logged but swallowed.
- If the story lookup fails, the action should still attempt to publish with an empty `story_key` — story lookup failure should be logged at Warn, not returned as an error. This ensures the pipeline is not blocked by a DB read for a notification step.
- The `renderTemplate` helper should be in `backend/internal/adapter/action/helpers.go` (shared with `git_pr.go`). If it does not exist yet, create it there.
- This action does NOT directly call any Discord/webhook adapter. It only publishes to the event bus. The `NotificationDispatcher` service (already implemented) will route the event to configured notifiers.

### Technical Specifications

**File:** `backend/internal/adapter/action/notification.go`

```go
package action

import (
    "context"
    "encoding/json"
    "log/slog"

    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
)

// NotificationAction implements model.Action for publishing a notification event.
// It renders a configurable message template with RunContext variables and publishes
// a "notification.sent" event via EventPublisher. Failures are non-fatal.
type NotificationAction struct {
    eventPub  port.EventPublisher
    storyRepo port.StoryRepository
    logger    *slog.Logger
}

func NewNotificationAction(
    eventPub port.EventPublisher,
    storyRepo port.StoryRepository,
    logger *slog.Logger,
) *NotificationAction {
    return &NotificationAction{eventPub: eventPub, storyRepo: storyRepo, logger: logger}
}

func (a *NotificationAction) Name() string { return "notification" }

func (a *NotificationAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    cfg := runCtx.RunStep.Config

    msgTemplate, ok := cfg["message"]
    if !ok || msgTemplate == "" {
        msgTemplate = "Pipeline step {step_name} completed"
    }

    storyKey := ""
    story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
    if err != nil {
        a.logger.Warn("notification: failed to fetch story, proceeding with empty story_key",
            "story_id", runCtx.StoryID, "error", err)
    } else {
        storyKey = story.Key
    }

    branchName, _ := runCtx.Metadata["branch_name"].(string)
    prURL, _ := runCtx.Metadata["pr_url"].(string)

    message := renderTemplate(msgTemplate, map[string]string{
        "story_key":   storyKey,
        "step_name":   runCtx.RunStep.StepName,
        "run_id":      runCtx.Run.ID.String(),
        "branch_name": branchName,
        "pr_url":      prURL,
    })

    payload := map[string]string{
        "message":   message,
        "step_id":   runCtx.RunStep.ID.String(),
        "run_id":    runCtx.Run.ID.String(),
        "story_key": storyKey,
    }
    payloadJSON, marshalErr := json.Marshal(payload)
    if marshalErr != nil {
        a.logger.Warn("notification: failed to marshal payload", "error", marshalErr)
        return nil // non-fatal
    }

    event := model.Event{
        ID:         uuid.New(),
        ProjectID:  runCtx.ProjectID,
        EntityType: "notification",
        EntityID:   runCtx.RunStep.ID,
        Action:     "sent",
        Payload:    payloadJSON,
    }

    if pubErr := a.eventPub.Publish(ctx, event); pubErr != nil {
        a.logger.Warn("notification: failed to publish event",
            "story_key", storyKey,
            "run_id", runCtx.Run.ID,
            "error", pubErr,
        )
    }

    return nil
}
```

**Event schema (notification.sent):**
- `entity_type`: `"notification"`
- `action`: `"sent"`
- `entity_id`: run step UUID
- `project_id`: from RunContext
- `payload`: `{ "message": "...", "step_id": "...", "run_id": "...", "story_key": "..." }`

**Template variables available:**
| Variable | Source |
|---|---|
| `{story_key}` | `story.Key` |
| `{step_name}` | `runCtx.RunStep.StepName` |
| `{run_id}` | `runCtx.Run.ID.String()` |
| `{branch_name}` | `runCtx.Metadata["branch_name"]` |
| `{pr_url}` | `runCtx.Metadata["pr_url"]` |

**Shared helper** — `renderTemplate` should live in `backend/internal/adapter/action/helpers.go`. If R-2-2 already created it, reuse it. Otherwise, define it there:

```go
// renderTemplate replaces {key} placeholders in tpl with values from vars.
func renderTemplate(tpl string, vars map[string]string) string {
    result := tpl
    for k, v := range vars {
        result = strings.ReplaceAll(result, "{"+k+"}", v)
    }
    return result
}
```

### Testing Requirements

File: `backend/internal/adapter/action/__tests__/notification_test.go`

**Tests:**

1. **Happy path** — config `message = "Story {story_key} done"`, mock storyRepo returns key `"S-01"`, mock EventPublisher captures event → verify `payload.message = "Story S-01 done"`, `payload.story_key = "S-01"`, `payload.run_id` non-empty, action returns nil.
2. **Template with metadata vars** — `message = "Branch {branch_name} ready"`, `Metadata["branch_name"] = "feat/S-01-foo"` → verify rendered message.
3. **Missing message config** — no key in config → default message used, EventPublisher called, action returns nil.
4. **EventPublisher failure** — Publish returns error → action returns nil, slog.Warn called (use a capturing slog handler or verify by not panicking).
5. **Story lookup failure** — storyRepo returns not-found → story_key is empty string in payload, EventPublisher still called, action returns nil.

```go
type MockEventPublisher struct {
    PublishFn    func(ctx context.Context, event model.Event) error
    PublishedEvents []model.Event
}
func (m *MockEventPublisher) Publish(_ context.Context, event model.Event) error {
    m.PublishedEvents = append(m.PublishedEvents, event)
    if m.PublishFn != nil {
        return m.PublishFn(context.Background(), event)
    }
    return nil
}
```

Run `golangci-lint run ./...` before committing.

### References

- `backend/internal/domain/port/event_publisher.go` — `EventPublisher.Publish` signature
- `backend/internal/domain/model/event.go` — `model.Event` struct
- `backend/internal/domain/model/run_context.go` — `RunContext`
- `backend/internal/adapter/action/hitl_gate.go` — reference `publishPendingEvent` pattern
- `backend/internal/domain/service/notification_dispatcher.go` — the dispatcher that routes `notification.sent` events to notifiers
- Story R-2-2 — defines `renderTemplate` helper in helpers.go (shared)

## Dev Agent Record

## Change Log

- 2026-02-23: Story created for Wave R implementation
