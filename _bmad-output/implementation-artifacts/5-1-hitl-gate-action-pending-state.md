# Story 5.1: [BACK] HITL Gate Action + Pending State

Status: ready-for-dev

## Story

As a pipeline executor, I want a `hitl_gate` action that suspends a running step in a `waiting_approval` state and records a HITL request, so that a human reviewer can inspect the agent's work before the pipeline continues.

## Acceptance Criteria (BDD)

**AC1: New StepStatus constant and valid transition**
- **Given** the domain model defines step statuses
- **When** `StepStatusWaitingApproval` is added to `model/run.go`
- **Then** `ValidateStepTransition` allows `running → waiting_approval`
- **And** no other existing transitions are affected

**AC2: HITLRequest domain model**
- **Given** a HITL gate action is triggered
- **When** a `HITLRequest` is created
- **Then** it has fields: `ID`, `RunStepID`, `GateType` (default "approval"), `DiffContent` (optional), `Status` (pending/approved/rejected), `ResolvedAt`, `ResolvedBy`, `RejectionReason`, `CreatedAt`

**AC3: DB migration for hitl_requests table**
- **Given** migration `000013_create_hitl_requests_table.up.sql` is applied
- **When** a HITL request is inserted
- **Then** the `hitl_requests` table stores all fields with correct constraints (FK on `run_steps`, FK on `users`)
- **And** the down migration cleanly drops the table

**AC4: HITLRepository port and Postgres adapter**
- **Given** the `HITLRepository` port is defined
- **When** the postgres adapter implements it
- **Then** `Create`, `GetByRunStepID`, and `UpdateStatus` work correctly against the DB

**AC5: HITLGateAction implements model.Action**
- **Given** a pipeline step with action type `hitl_gate`
- **When** `Execute(ctx, runCtx)` is called
- **Then** it fetches the PR diff from `GitProvider` if `pr_url` is present in run metadata
- **And** it calls `HITLRepository.Create` to persist the HITL request with status `pending`
- **And** it calls `RunRepository.UpdateRunStepStatus` to set the step to `waiting_approval`
- **And** it publishes a `hitl_gate.pending` event with `run_id`, `step_id`, `story_key`, `hitl_request_id`
- **And** it returns `nil` (not an error — suspension is successful execution)

**AC6: Pipeline executor skips next step when step is waiting_approval**
- **Given** the pipeline executor checks step status after `action.Execute()` returns nil
- **When** the step status is `waiting_approval`
- **Then** the executor does NOT enqueue the next step
- **And** it logs the suspension at info level with `run_id` and `step_id`

**AC7: ActionRegistry wiring**
- **Given** the application starts
- **When** `HITLGateAction` is registered in the `ActionRegistry`
- **Then** `actionReg.Get("hitl_gate")` returns the action without error

**AC8: Unit tests cover the action and repository**
- **Given** unit tests in `backend/internal/adapter/action/__tests__/hitl_gate_test.go`
- **When** tests run
- **Then** happy path, missing PR URL (no diff), and event publish failure paths are covered
- **And** all mocks are hand-written

## Tasks / Subtasks

- [ ] [BACK] Task 1: Extend domain model — add StepStatusWaitingApproval and HITLRequest (AC: #1, #2)
  - [ ] Add `StepStatusWaitingApproval StepStatus = "waiting_approval"` to `backend/internal/domain/model/run.go`
  - [ ] Update `validStepTransitions` to allow `StepStatusRunning → StepStatusWaitingApproval`
  - [ ] Create `backend/internal/domain/model/hitl.go` with `HITLRequest` struct and `HITLStatus` constants (`HITLStatusPending`, `HITLStatusApproved`, `HITLStatusRejected`)

- [ ] [BACK] Task 2: DB migration for hitl_requests table (AC: #3)
  - [ ] Create `backend/migrations/000013_create_hitl_requests_table.up.sql`
  - [ ] Create `backend/migrations/000013_create_hitl_requests_table.down.sql`
  - [ ] Schema: `id UUID PK`, `run_step_id UUID NOT NULL REFERENCES run_steps(id) ON DELETE CASCADE`, `gate_type VARCHAR(50) NOT NULL DEFAULT 'approval'`, `diff_content TEXT`, `status VARCHAR(50) NOT NULL DEFAULT 'pending'`, `resolved_at TIMESTAMPTZ`, `resolved_by UUID REFERENCES users(id)`, `rejection_reason TEXT`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
  - [ ] Add index: `idx_hitl_requests_run_step_id` on `run_step_id`, and `idx_hitl_requests_status` on `status`

- [ ] [BACK] Task 3: sqlc queries and HITLRepository port (AC: #4)
  - [ ] Create `backend/queries/hitl_requests.sql` with `CreateHITLRequest :one`, `GetHITLRequestByRunStepID :one`, `UpdateHITLRequestStatus :one`
  - [ ] Run `cd backend && sqlc generate` to generate `internal/adapter/postgres/db/` code
  - [ ] Define `HITLRepository` port interface in `backend/internal/domain/port/hitl_repository.go`

- [ ] [BACK] Task 4: Postgres adapter implementing HITLRepository (AC: #4)
  - [ ] Create `backend/internal/adapter/postgres/hitl_repo.go` implementing `port.HITLRepository`
  - [ ] Map sqlc rows to `model.HITLRequest` in a `toDomainHITLRequest` helper
  - [ ] Handle `pgx.ErrNoRows` → `errors.NewNotFound`; all other DB errors → `errors.NewInternal`

- [ ] [BACK] Task 5: Implement HITLGateAction (AC: #5)
  - [ ] Create `backend/internal/adapter/action/hitl_gate.go`
  - [ ] Define `HITLGateAction` struct with deps: `hitlRepo port.HITLRepository`, `runRepo port.RunRepository`, `gitProvider port.GitProvider`, `eventPub port.EventPublisher`, `storyRepo port.StoryRepository`, `logger *slog.Logger`
  - [ ] Implement `Name() string` returning `"hitl_gate"`
  - [ ] Implement `Execute`: fetch story key, attempt diff fetch if `pr_url` in metadata, call `hitlRepo.Create`, update step status to `waiting_approval`, publish `hitl_gate.pending` event, return `nil`

- [ ] [BACK] Task 6: Extend GitProvider port with GetDiff (AC: #5)
  - [ ] Add `GetPRDiff(ctx context.Context, prURL string) (string, error)` to `backend/internal/domain/port/git_provider.go`
  - [ ] Implement in `backend/internal/adapter/github/git_provider.go` using `gh pr diff <prURL>` via `CommandRunner`
  - [ ] HITLGateAction calls this and stores result in `diff_content`; if the call fails, log at warn level and proceed with empty diff (non-fatal)

- [ ] [BACK] Task 7: Patch PipelineExecutor to detect waiting_approval after Execute (AC: #6)
  - [ ] In `backend/internal/domain/service/pipeline_executor.go`, after `action.Execute()` returns nil, re-fetch the step from `RunRepository`
  - [ ] If `step.Status == StepStatusWaitingApproval`, log the suspension and return without enqueuing the next step
  - [ ] Add unit test for this behaviour in the pipeline executor test file

- [ ] [BACK] Task 8: Wire HITLGateAction into ActionRegistry and DI (AC: #7)
  - [ ] Instantiate `HITLGateAction` in `backend/cmd/api/main.go` with its dependencies
  - [ ] Register with `actionReg.Register(hitlGateAction)` after the existing `agent_run` registration
  - [ ] Instantiate `HITLRepository` postgres adapter and inject it

- [ ] [BACK] Task 9: Update OpenAPI spec with HITL schemas (AC: #2, #8)
  - [ ] Add `HITLRequest` schema to `api/openapi.yaml` (fields: `id`, `run_step_id`, `gate_type`, `diff_content`, `status`, `resolved_at`, `resolved_by`, `rejection_reason`, `created_at`)
  - [ ] Add stub endpoints (for Story 5.2 implementation): `GET /api/v1/hitl-requests/{id}`, `POST /api/v1/hitl-requests/{id}/approve`, `POST /api/v1/hitl-requests/{id}/reject`
  - [ ] Run `cd backend && make generate` to regenerate server interfaces

- [ ] [BACK] Task 10: Unit tests for HITLGateAction (AC: #8)
  - [ ] Create `backend/internal/adapter/action/__tests__/hitl_gate_test.go`
  - [ ] Test: happy path with PR URL — diff fetched, HITL request created, step status updated, event published, nil returned
  - [ ] Test: no PR URL in metadata — HITL request created with empty diff, no git call made
  - [ ] Test: GitProvider.GetPRDiff fails — proceeds with empty diff (warn logged), no error returned
  - [ ] Test: HITLRepository.Create fails — returns wrapped error
  - [ ] Run `golangci-lint run ./...` — must pass

## Dev Notes

### Dependencies

**Story 3.7 (Pipeline executor — DONE):** Provides `Action` interface, `RunContext`, `PipelineExecutor`. Task 7 patches the executor post-Execute check. The executor's `execute step` loop must re-fetch step status after a successful `action.Execute()` to detect suspension.

**Story 3.10 (Run launch API — DONE):** Run and RunStep persistence already in place. `RunRepository.UpdateRunStepStatus` is available.

**Story 3.6 (Event bus — DONE):** `EventPublisher` port is available for `hitl_gate.pending` events.

**Story 5.2 (Approve/Reject API — Wave 11):** The OpenAPI spec stub endpoints added in Task 9 are required for Story 5.4 (frontend). The backend handlers are NOT implemented in this story — only the spec is updated and code-generated.

### Architecture Requirements

- `HITLGateAction` is in `backend/internal/adapter/action/` — implements `model.Action`
- `HITLRepository` is a new port in `backend/internal/domain/port/hitl_repository.go`
- `HITLRequest` domain model lives in `backend/internal/domain/model/hitl.go`
- No business logic in the adapter; decisions (e.g. diff-fetch is non-fatal) are encoded in the action
- `GetPRDiff` added to `GitProvider` port — the github adapter uses `gh pr diff <url>` via `CommandRunner`

### File Paths (exact)

```
backend/internal/domain/model/run.go                                          # Add StepStatusWaitingApproval + transition
backend/internal/domain/model/hitl.go                                         # New: HITLRequest model + HITLStatus constants
backend/internal/domain/port/hitl_repository.go                               # New: HITLRepository port interface
backend/internal/domain/port/git_provider.go                                  # Add GetPRDiff method
backend/internal/adapter/action/hitl_gate.go                                  # New: HITLGateAction implementation
backend/internal/adapter/action/__tests__/hitl_gate_test.go                   # New: unit tests
backend/internal/adapter/postgres/hitl_repo.go                                # New: HITLRepository postgres adapter
backend/internal/adapter/github/git_provider.go                               # Add GetPRDiff implementation
backend/internal/domain/service/pipeline_executor.go                          # Patch: detect waiting_approval post-Execute
backend/migrations/000013_create_hitl_requests_table.up.sql                   # New: migration up
backend/migrations/000013_create_hitl_requests_table.down.sql                 # New: migration down
backend/queries/hitl_requests.sql                                              # New: sqlc queries
api/openapi.yaml                                                               # Add HITLRequest schema + stub endpoints
```

### Technical Specifications

**HITLStatus constants and HITLRequest model:**
```go
// backend/internal/domain/model/hitl.go
package model

import (
    "time"
    "github.com/google/uuid"
)

// HITLStatus represents the approval state of a HITL request.
type HITLStatus string

const (
    HITLStatusPending  HITLStatus = "pending"
    HITLStatusApproved HITLStatus = "approved"
    HITLStatusRejected HITLStatus = "rejected"
)

// HITLRequest records a human-in-the-loop gate triggered by a pipeline step.
type HITLRequest struct {
    ID              uuid.UUID
    RunStepID       uuid.UUID
    GateType        string      // default "approval"
    DiffContent     *string     // PR diff fetched from GitProvider; nil if unavailable
    Status          HITLStatus
    ResolvedAt      *time.Time
    ResolvedBy      *uuid.UUID  // user ID who resolved
    RejectionReason *string
    CreatedAt       time.Time
}
```

**HITLRepository port:**
```go
// backend/internal/domain/port/hitl_repository.go
package port

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// HITLRepository defines persistence operations for HITL approval requests.
type HITLRepository interface {
    // Create persists a new HITL request with status "pending".
    Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
    // GetByRunStepID returns the HITL request for the given run step.
    GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error)
    // UpdateStatus transitions the HITL request to approved or rejected.
    UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error)
}
```

**HITLGateAction — Execute flow:**
```go
// backend/internal/adapter/action/hitl_gate.go
func (a *HITLGateAction) Execute(ctx context.Context, runCtx *model.RunContext) error {
    // 1. Fetch story for context (story key for event payload)
    story, err := a.storyRepo.GetByID(ctx, runCtx.StoryID)
    if err != nil {
        return fmt.Errorf("fetch story: %w", err)
    }

    // 2. Attempt to fetch PR diff (non-fatal)
    var diffContent *string
    if prURL, ok := runCtx.Metadata["pr_url"].(string); ok && prURL != "" {
        diff, diffErr := a.gitProvider.GetPRDiff(ctx, prURL)
        if diffErr != nil {
            a.logger.Warn("failed to fetch PR diff, proceeding without diff",
                "pr_url", prURL, "error", diffErr)
        } else {
            diffContent = &diff
        }
    }

    // 3. Create HITL request
    req := &model.HITLRequest{
        ID:          uuid.New(),
        RunStepID:   runCtx.RunStep.ID,
        GateType:    "approval",
        DiffContent: diffContent,
        Status:      model.HITLStatusPending,
        CreatedAt:   time.Now(),
    }
    created, err := a.hitlRepo.Create(ctx, req)
    if err != nil {
        return fmt.Errorf("create HITL request: %w", err)
    }

    // 4. Transition step to waiting_approval
    now := time.Now()
    if _, err := a.runRepo.UpdateRunStepStatus(ctx, runCtx.RunStep.ID,
        model.StepStatusWaitingApproval, &now, nil, nil); err != nil {
        return fmt.Errorf("update step to waiting_approval: %w", err)
    }

    // 5. Publish hitl_gate.pending event
    a.publishPendingEvent(ctx, runCtx, story.Key, created.ID)

    return nil
}
```

**PipelineExecutor patch (post-Execute check):**
```go
// In pipeline_executor.go, after action.Execute() returns nil:
if err := action.Execute(ctx, runCtx); err != nil {
    // ... existing error handling
}

// Re-fetch step to detect suspension
updatedStep, fetchErr := s.runRepo.GetRunStep(ctx, step.ID)
if fetchErr != nil {
    s.logger.Warn("failed to re-fetch step after execute", "step_id", step.ID, "error", fetchErr)
} else if updatedStep.Status == model.StepStatusWaitingApproval {
    s.logger.Info("pipeline step suspended for approval",
        "run_id", run.ID, "step_id", step.ID)
    return // do NOT enqueue next step
}
// ... existing: enqueue next step
```

**Event payload for hitl_gate.pending:**
```go
type HITLPendingPayload struct {
    RunID         string `json:"run_id"`
    StepID        string `json:"step_id"`
    StoryKey      string `json:"story_key"`
    HITLRequestID string `json:"hitl_request_id"`
}
```

**sqlc queries (backend/queries/hitl_requests.sql):**
```sql
-- name: CreateHITLRequest :one
INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, status, created_at)
VALUES ($1, $2, $3, $4, $5, now())
RETURNING *;

-- name: GetHITLRequestByRunStepID :one
SELECT * FROM hitl_requests WHERE run_step_id = $1 LIMIT 1;

-- name: UpdateHITLRequestStatus :one
UPDATE hitl_requests
SET status = $2, resolved_at = $3, resolved_by = $4, rejection_reason = $5
WHERE id = $1
RETURNING *;
```

**GetPRDiff — github adapter:**
```go
// Runs: gh pr diff <prURL>
func (g *GitHubProvider) GetPRDiff(ctx context.Context, prURL string) (string, error) {
    out, err := g.runner.Run(ctx, "gh", "pr", "diff", prURL)
    if err != nil {
        return "", fmt.Errorf("gh pr diff %q: %w", prURL, err)
    }
    return out, nil
}
```

**Migration 000013 up:**
```sql
CREATE TABLE hitl_requests (
    id              UUID PRIMARY KEY,
    run_step_id     UUID NOT NULL REFERENCES run_steps(id) ON DELETE CASCADE,
    gate_type       VARCHAR(50) NOT NULL DEFAULT 'approval',
    diff_content    TEXT,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    resolved_at     TIMESTAMPTZ,
    resolved_by     UUID REFERENCES users(id),
    rejection_reason TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hitl_requests_run_step_id ON hitl_requests(run_step_id);
CREATE INDEX idx_hitl_requests_status ON hitl_requests(status);
```

**Next migration number:** Last is `000012` — use `000013`.

**Error codes:**
- `HITL_REQUEST_NOT_FOUND` — no HITL request for given run step
- `HITL_REQUEST_CREATE_FAILED` — DB insert failed

### Testing Requirements

**Unit tests (hitl_gate_test.go):**

Mock interfaces needed: `MockHITLRepository`, `MockRunRepository`, `MockGitProvider`, `MockEventPublisher`, `MockStoryRepository`.

1. **Happy path with PR URL:** metadata has `pr_url`, GetPRDiff returns diff string, Create called with non-nil DiffContent, UpdateRunStepStatus called with `waiting_approval`, event published, nil returned.
2. **No PR URL:** GetPRDiff NOT called, Create called with nil DiffContent, nil returned.
3. **GetPRDiff fails:** warn logged, Create called with nil DiffContent, nil returned (non-fatal).
4. **HITLRepository.Create fails:** error wraps "create HITL request".

**Pipeline executor patch test:** Add a table-driven case to the existing `pipeline_executor_test.go` — when action.Execute returns nil but step is re-fetched with status `waiting_approval`, verify no next-step enqueue call is made.

**Lint:** Run `golangci-lint run ./...` from `backend/` — must pass before commit.

### References

- `backend/internal/domain/model/run.go` — existing StepStatus constants and ValidateStepTransition
- `backend/internal/adapter/action/agent_run.go` — Action interface pattern to follow
- `backend/internal/domain/service/pipeline_executor.go` — execution loop to patch
- `backend/internal/domain/port/git_provider.go` — GitProvider port to extend
- `backend/internal/adapter/github/git_provider.go` — gh CLI command pattern
- `backend/migrations/000012_seed_default_prompt_templates.up.sql` — last migration for reference
- `api/openapi.yaml` — add HITLRequest schema + stub endpoints before Story 5.4 starts

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
