# Story 5.2: [BACK] HITL Approval/Rejection API

Status: ready-for-dev

## Story

As an authorized reviewer, I want to approve or reject a pending HITL gate via API, so that the pipeline can resume or fail based on human decision.

## Acceptance Criteria (BDD)

**AC1: OpenAPI spec defines approve and reject endpoints under the project/run scope**
- **Given** the OpenAPI spec at `api/openapi.yaml` already has stub endpoints `/hitl-requests/{hitlRequestId}/approve` and `/hitl-requests/{hitlRequestId}/reject`
- **When** Story 5-2 is implemented
- **Then** those endpoints are updated to include a `projectId` and `runId` path parameter scope, changing to `POST /api/v1/projects/{projectId}/runs/{runId}/hitl/approve` and `POST /api/v1/projects/{projectId}/runs/{runId}/hitl/reject`
- **And** the reject body schema `RejectHITLRequest` has `reason` as optional (minLength removed, not required)
- **And** both endpoints return `202 Accepted` with a body of `{ "run_id": "...", "status": "...", "hitl_request_id": "..." }` using a new `HITLActionResponse` schema
- **And** `cd backend && make generate` regenerates chi server interfaces successfully

**AC2: Approve endpoint resolves a pending HITL request and resumes the pipeline**
- **Given** a run with status `running` that has a step in `waiting_approval` status
- **And** a `HITLRequest` record with `status = "pending"` linked to that step
- **When** `POST /api/v1/projects/{projectId}/runs/{runId}/hitl/approve` is called with a valid JWT
- **Then** HTTP 202 is returned
- **And** `HITLRepository.UpdateStatus` is called with `HITLStatusApproved`, the authenticated user's ID as `resolvedBy`, and `time.Now()` as `resolvedAt`
- **And** the pending step is transitioned from `waiting_approval` to `running`
- **And** a `hitl_gate.approved` event is published with `run_id`, `step_id`, `hitl_request_id`, `project_id`
- **And** the pipeline executor is re-triggered by enqueuing a `ResumeRunJob` via the `JobQueue` for this run and step

**AC3: Reject endpoint fails the run with HITL_REJECTED error**
- **Given** a run with a step in `waiting_approval` status and a pending HITL request
- **When** `POST /api/v1/projects/{projectId}/runs/{runId}/hitl/reject` is called with an optional JSON body `{"reason": "string"}`
- **Then** HTTP 202 is returned
- **And** `HITLRepository.UpdateStatus` is called with `HITLStatusRejected`, the user's ID as `resolvedBy`, and the rejection reason stored in `rejection_reason`
- **And** the pending step is transitioned to `failed` with `error_message = "HITL_REJECTED: <reason>"`
- **And** the run is transitioned to `failed` with `error_message = "HITL_REJECTED: <reason>"`
- **And** a `hitl_gate.rejected` event is published with `run_id`, `step_id`, `hitl_request_id`, `project_id`, `reason`

**AC4: Auth and project access are enforced**
- **Given** a request without a valid JWT cookie
- **When** either endpoint is called
- **Then** HTTP 401 is returned
- **Given** a valid JWT but the authenticated user is not a member of the project identified by `projectId`
- **When** either endpoint is called
- **Then** HTTP 403 is returned

**AC5: Idempotency guard — only pending HITL requests can be resolved**
- **Given** a HITL request that is already `approved` or `rejected`
- **When** approve or reject is called again
- **Then** HTTP 409 Conflict is returned with error code `HITL_ALREADY_RESOLVED`
- **And** no state is modified

**AC6: Run and step must exist and belong to the project**
- **Given** a `runId` that does not exist or does not belong to the project identified by `projectId`
- **When** either endpoint is called
- **Then** HTTP 404 is returned with error code `RUN_NOT_FOUND`
- **Given** a run with no HITL request pending (no `waiting_approval` step)
- **When** either endpoint is called
- **Then** HTTP 404 is returned with error code `HITL_REQUEST_NOT_FOUND`

**AC7: HITLService encapsulates all approval/rejection business logic**
- **Given** a `HITLService` struct in `backend/internal/domain/service/hitl_service.go`
- **When** `Approve(ctx, projectID, runID, reviewerID)` is called
- **Then** it orchestrates: fetch run, verify project ownership, fetch HITL request, guard idempotency, update HITL status, transition step to `running`, publish event, enqueue resume job
- **When** `Reject(ctx, projectID, runID, reviewerID, reason)` is called
- **Then** it orchestrates: fetch run, verify project ownership, fetch HITL request, guard idempotency, update HITL status, transition step to `failed`, transition run to `failed`, publish event

**AC8: Unit tests cover the service and handler**
- **Given** unit tests in `backend/internal/domain/service/__tests__/hitl_service_test.go`
- **When** tests run
- **Then** happy path for approve and reject, idempotency conflict, run not found, and HITL request not found are covered
- **And** `golangci-lint run ./...` passes from `backend/`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec — replace stub endpoints with project-scoped routes (AC: #1)
  - [ ] Remove stubs `/hitl-requests/{hitlRequestId}/approve` and `/hitl-requests/{hitlRequestId}/reject` from `api/openapi.yaml`
  - [ ] Add `POST /projects/{projectId}/runs/{runId}/hitl/approve` with operationId `approveHITLGate`, tags `[hitl]`, requiring `ProjectIdPath` and `RunIdPath` parameters, returning 202 with `HITLActionResponse` schema
  - [ ] Add `POST /projects/{projectId}/runs/{runId}/hitl/reject` with operationId `rejectHITLGate`, tags `[hitl]`, requiring `ProjectIdPath` and `RunIdPath` parameters, optional request body `RejectHITLRequest` (make `reason` field optional, remove `required` constraint), returning 202 with `HITLActionResponse` schema
  - [ ] Add `HITLActionResponse` schema to components: `{ run_id: uuid, hitl_request_id: uuid, status: string }` with `required: [run_id, hitl_request_id, status]`
  - [ ] Update `RejectHITLRequest` schema to make `reason` optional (remove from `required` array, remove `minLength`)
  - [ ] Add 401, 403, 404, 409 error responses to both new endpoints
  - [ ] Run `cd backend && make generate` to regenerate server interfaces

- [ ] [BACK] Task 2: Extend HITLRepository port with GetByRunID (AC: #7)
  - [ ] Add `GetPendingByRunID(ctx context.Context, runID uuid.UUID) (*model.HITLRequest, error)` to `backend/internal/domain/port/hitl_repository.go`
  - [ ] This method fetches the HITL request for the `waiting_approval` step of a given run by joining on `run_steps`

- [ ] [BACK] Task 3: Implement GetPendingByRunID in Postgres adapter (AC: #6, #7)
  - [ ] Add sqlc query `GetPendingHITLRequestByRunID :one` in `backend/queries/hitl_requests.sql`:
    ```sql
    SELECT hr.* FROM hitl_requests hr
    JOIN run_steps rs ON rs.id = hr.run_step_id
    WHERE rs.run_id = $1 AND hr.status = 'pending'
    LIMIT 1;
    ```
  - [ ] Run `cd backend && sqlc generate` to regenerate `backend/internal/adapter/postgres/db/`
  - [ ] Implement `GetPendingByRunID` in `backend/internal/adapter/postgres/hitl_repo.go`
  - [ ] Handle `pgx.ErrNoRows` → `errors.NewNotFound("hitl_request", runID)` with code `HITL_REQUEST_NOT_FOUND`

- [ ] [BACK] Task 4: Implement HITLService with Approve and Reject methods (AC: #2, #3, #5, #7)
  - [ ] Create `backend/internal/domain/service/hitl_service.go`
  - [ ] Define `HITLService` struct with deps: `hitlRepo port.HITLRepository`, `runRepo port.RunRepository`, `eventPub port.EventPublisher`, `jobQueue port.JobQueue`, `logger *slog.Logger`
  - [ ] Implement `Approve(ctx context.Context, projectID, runID, reviewerID uuid.UUID) (*ApproveResult, error)`:
    - Fetch run via `runRepo.GetRun` — return `RUN_NOT_FOUND` if missing
    - Verify `run.ProjectID == projectID` — return `errors.NewNotFound` if mismatch (avoids leaking existence)
    - Fetch HITL request via `hitlRepo.GetPendingByRunID(ctx, runID)` — return `HITL_REQUEST_NOT_FOUND` if missing
    - Guard: if `hitlRequest.Status != HITLStatusPending` → return `errors.NewConflict("HITL_ALREADY_RESOLVED", ...)`
    - Call `hitlRepo.UpdateStatus(ctx, hitlRequest.ID, HITLStatusApproved, &reviewerID, nil, time.Now())`
    - Transition step from `waiting_approval` to `running` via `runRepo.UpdateRunStepStatus`
    - Publish `hitl_gate.approved` event
    - Enqueue `ResumeRunJob{RunID: runID, StepID: hitlRequest.RunStepID}` via `jobQueue.Enqueue`
    - Return `&ApproveResult{RunID: runID, HITLRequestID: hitlRequest.ID, Status: run.Status.String()}`
  - [ ] Implement `Reject(ctx context.Context, projectID, runID, reviewerID uuid.UUID, reason string) (*RejectResult, error)`:
    - Same fetch + project ownership check as Approve
    - Fetch HITL request via `hitlRepo.GetPendingByRunID`; guard idempotency
    - Build `errorMsg := "HITL_REJECTED"` + append reason if non-empty
    - Call `hitlRepo.UpdateStatus(ctx, hitlRequest.ID, HITLStatusRejected, &reviewerID, &reason, time.Now())`
    - Transition step to `failed` with `error_message = errorMsg` via `runRepo.UpdateRunStepStatus`
    - Transition run to `failed` with `error_message = errorMsg` via `runRepo.UpdateRunStatus`
    - Publish `hitl_gate.rejected` event with reason in payload
    - Return `&RejectResult{RunID: runID, HITLRequestID: hitlRequest.ID, Status: "failed"}`
  - [ ] Define `ApproveResult` and `RejectResult` structs in the same file

- [ ] [BACK] Task 5: Implement HITLHandler with ApproveHITLGate and RejectHITLGate (AC: #1, #2, #3, #4)
  - [ ] Create `backend/internal/api/handler/hitl_handler.go`
  - [ ] Define `HITLHandler` struct with deps: `service *service.HITLService`, `userService *service.ProjectUserService`
  - [ ] Implement `ApproveHITLGate(w http.ResponseWriter, r *http.Request)`:
    - Parse `projectId` and `runId` from chi URL params
    - Call `h.checkProjectAccess(r, projectID)` (same pattern as `ProjectHandler`)
    - Extract `reviewerID` from auth context via `middleware.UserIDFromContext`
    - Call `h.service.Approve(r.Context(), projectID, runID, reviewerID)`
    - Render 202 with `HITLActionResponse` JSON
  - [ ] Implement `RejectHITLGate(w http.ResponseWriter, r *http.Request)`:
    - Same auth + access check
    - Decode optional JSON body `{"reason": "string"}` — if body absent or empty, use `reason = ""`
    - Call `h.service.Reject(r.Context(), projectID, runID, reviewerID, reason)`
    - Render 202 with `HITLActionResponse` JSON
  - [ ] Register both routes in `backend/internal/api/router.go` under `r.Route("/api/v1/projects/{projectId}/runs/{runId}", ...)`:
    - `r.Post("/hitl/approve", hitlHandler.ApproveHITLGate)`
    - `r.Post("/hitl/reject", hitlHandler.RejectHITLGate)`

- [ ] [BACK] Task 6: Publish hitl_gate events from HITLService (AC: #2, #3)
  - [ ] In `hitl_service.go`, add private `publishApprovedEvent` and `publishRejectedEvent` helpers following the same pattern as `hitl_gate.go`'s `publishPendingEvent`
  - [ ] `hitl_gate.approved` event payload: `{ "run_id": "...", "step_id": "...", "hitl_request_id": "...", "project_id": "..." }`
  - [ ] `hitl_gate.rejected` event payload: `{ "run_id": "...", "step_id": "...", "hitl_request_id": "...", "project_id": "...", "reason": "..." }`
  - [ ] Event publish failures must be logged at error level but must NOT fail the HTTP request (non-fatal, same policy as `hitl_gate.go`)

- [ ] [BACK] Task 7: Wire HITLService and HITLHandler into DI (AC: #7)
  - [ ] Add `NewHITLService` to `backend/cmd/api/wire.go` provider set
  - [ ] Add `NewHITLHandler` to `backend/cmd/api/wire.go` provider set
  - [ ] Inject `HITLHandler` into `NewRouter` (add parameter; register routes)
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] [BACK] Task 8: Unit tests for HITLService (AC: #8)
  - [ ] Create `backend/internal/domain/service/__tests__/hitl_service_test.go`
  - [ ] Hand-written mocks: `MockHITLRepository`, `MockRunRepository`, `MockEventPublisher`, `MockJobQueue`
  - [ ] Test table for `Approve`:
    - Happy path: HITL request updated, step transitioned to `running`, event published, job enqueued, 202 result returned
    - Run not found: `GetRun` returns not-found error → propagates as `RUN_NOT_FOUND`
    - Run project mismatch: `run.ProjectID != projectID` → 404 not found (not 403, to avoid existence leakage)
    - HITL request not found: `GetPendingByRunID` returns not-found → `HITL_REQUEST_NOT_FOUND`
    - Already resolved: `hitlRequest.Status == HITLStatusApproved` → conflict `HITL_ALREADY_RESOLVED`
  - [ ] Test table for `Reject`:
    - Happy path: HITL updated, step set to `failed`, run set to `failed`, event published, 202 returned
    - Reject with empty reason: `error_message = "HITL_REJECTED"` (no colon/suffix)
    - Reject with reason: `error_message = "HITL_REJECTED: code review required"`
    - Already resolved: `hitlRequest.Status == HITLStatusRejected` → conflict `HITL_ALREADY_RESOLVED`
  - [ ] Run `golangci-lint run ./...` from `backend/` — must pass

## Dev Notes

### Dependencies

**Story 5-1 (HITL Gate Action — DONE):** Provides the complete foundation this story builds on:
- `model.HITLRequest` and `model.HITLStatus` constants in `backend/internal/domain/model/hitl.go`
- `port.HITLRepository` interface in `backend/internal/domain/port/hitl_repository.go`
- `port.HITLRepository` Postgres adapter in `backend/internal/adapter/postgres/hitl_repo.go`
- `StepStatusWaitingApproval` in `backend/internal/domain/model/run.go`
- State machine allows `waiting_approval → running`, `waiting_approval → failed`, `waiting_approval → completed`, `waiting_approval → cancelled`
- `hitl_requests` table (migration `000013`) with `Create`, `GetByRunStepID`, `UpdateStatus` queries already in `backend/queries/hitl_requests.sql`
- `HITLGateAction` registered in `ActionRegistry`

**Story 3-7 (Pipeline Executor — DONE):** `ResumeRunJob` or equivalent mechanism for re-triggering execution from a specific step. If no `ResumeRunJob` type exists yet, Task 4 must define it in `backend/internal/domain/port/job_queue.go` or inline as a River job payload struct.

**Story 3-6 (Event Bus — DONE):** `EventPublisher` port available.

**Story 1-6 (RBAC Middleware — DONE):** `middleware.UserIDFromContext` and `middleware.IsAdmin` available.

### Architecture Requirements

- `HITLService` lives in `backend/internal/domain/service/` — it is a domain service, not a handler
- No business logic in `HITLHandler` — only HTTP parsing, auth extraction, and rendering
- `HITLHandler` uses the same project access check pattern as `ProjectHandler.checkProjectAccess`
- Event publish failures are **non-fatal** — log at error, do not abort the response
- The idempotency guard (`HITL_ALREADY_RESOLVED`) is checked before any state mutation
- Run project ownership is verified via `run.ProjectID == projectID` — not via a separate query
- The `reason` field in `RejectHITLRequest` is optional in the updated spec — handlers must handle an absent body gracefully

### File Paths (exact)

```
api/openapi.yaml                                                               # Update: replace stub endpoints with project-scoped routes, add HITLActionResponse schema
backend/queries/hitl_requests.sql                                              # Add: GetPendingHITLRequestByRunID query
backend/internal/adapter/postgres/db/                                          # Regenerated: sqlc generate
backend/internal/adapter/postgres/hitl_repo.go                                 # Add: GetPendingByRunID implementation
backend/internal/domain/port/hitl_repository.go                               # Add: GetPendingByRunID method to interface
backend/internal/domain/service/hitl_service.go                               # New: HITLService with Approve and Reject
backend/internal/domain/service/__tests__/hitl_service_test.go                # New: unit tests
backend/internal/api/handler/hitl_handler.go                                  # New: HITLHandler with ApproveHITLGate and RejectHITLGate
backend/internal/api/router.go                                                 # Update: register HITL routes
backend/cmd/api/wire.go                                                        # Update: add HITLService and HITLHandler providers
backend/cmd/api/wire_gen.go                                                    # Regenerated: wire ./cmd/api/
```

### Technical Specifications

**HITLActionResponse (API response schema):**
```yaml
# api/openapi.yaml — add to components/schemas
HITLActionResponse:
  type: object
  required: [run_id, hitl_request_id, status]
  properties:
    run_id:
      type: string
      format: uuid
    hitl_request_id:
      type: string
      format: uuid
    status:
      type: string
      description: Current run status after the action
      example: running
```

**New OpenAPI endpoints (replacing stubs):**
```yaml
/projects/{projectId}/runs/{runId}/hitl/approve:
  post:
    operationId: approveHITLGate
    summary: Approve a pending HITL gate, resuming pipeline execution
    tags: [hitl]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/RunIdPath"
    responses:
      "202":
        description: HITL gate approved, pipeline resuming
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HITLActionResponse"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "403":
        $ref: "#/components/responses/Forbidden"
      "404":
        $ref: "#/components/responses/NotFound"
      "409":
        $ref: "#/components/responses/Conflict"

/projects/{projectId}/runs/{runId}/hitl/reject:
  post:
    operationId: rejectHITLGate
    summary: Reject a pending HITL gate, failing the pipeline run
    tags: [hitl]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/RunIdPath"
    requestBody:
      required: false
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/RejectHITLRequest"
    responses:
      "202":
        description: HITL gate rejected, run marked as failed
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HITLActionResponse"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "403":
        $ref: "#/components/responses/Forbidden"
      "404":
        $ref: "#/components/responses/NotFound"
      "409":
        $ref: "#/components/responses/Conflict"
```

**HITLService struct:**
```go
// backend/internal/domain/service/hitl_service.go
package service

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
    "github.com/zakari/hopeitworks/backend/internal/domain/port"
    "github.com/zakari/hopeitworks/backend/pkg/errors"
)

// HITLService provides business logic for approving and rejecting HITL gates.
type HITLService struct {
    hitlRepo port.HITLRepository
    runRepo  port.RunRepository
    eventPub port.EventPublisher
    jobQueue port.JobQueue
    logger   *slog.Logger
}

// NewHITLService creates a new HITLService.
func NewHITLService(
    hitlRepo port.HITLRepository,
    runRepo port.RunRepository,
    eventPub port.EventPublisher,
    jobQueue port.JobQueue,
    logger *slog.Logger,
) *HITLService { ... }

// ApproveResult holds the data returned by a successful approval.
type ApproveResult struct {
    RunID         uuid.UUID
    HITLRequestID uuid.UUID
    Status        string
}

// RejectResult holds the data returned by a successful rejection.
type RejectResult struct {
    RunID         uuid.UUID
    HITLRequestID uuid.UUID
    Status        string
}

// Approve approves a pending HITL gate for the given run, resuming pipeline execution.
func (s *HITLService) Approve(ctx context.Context, projectID, runID, reviewerID uuid.UUID) (*ApproveResult, error) { ... }

// Reject rejects a pending HITL gate for the given run, failing the pipeline.
func (s *HITLService) Reject(ctx context.Context, projectID, runID, reviewerID uuid.UUID, reason string) (*RejectResult, error) { ... }
```

**Error codes introduced:**
- `HITL_ALREADY_RESOLVED` — attempt to approve/reject a non-pending HITL request (409 Conflict)
- `RUN_NOT_FOUND` — run does not exist or does not belong to the project (404 Not Found)
- `HITL_REQUEST_NOT_FOUND` — no pending HITL request found for the given run (404 Not Found)

**GetPendingHITLRequestByRunID sqlc query:**
```sql
-- name: GetPendingHITLRequestByRunID :one
SELECT hr.*
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
WHERE rs.run_id = $1
  AND hr.status = 'pending'
LIMIT 1;
```

**Event payloads:**

`hitl_gate.approved`:
```json
{
  "run_id": "...",
  "step_id": "...",
  "hitl_request_id": "...",
  "project_id": "..."
}
```

`hitl_gate.rejected`:
```json
{
  "run_id": "...",
  "step_id": "...",
  "hitl_request_id": "...",
  "project_id": "...",
  "reason": "code review required"
}
```

**State transitions triggered by Approve:**
- `run_steps.status`: `waiting_approval → running` (valid per `validStepTransitions` in `model/run.go`)
- `hitl_requests.status`: `pending → approved`

**State transitions triggered by Reject:**
- `run_steps.status`: `waiting_approval → failed` (valid per `validStepTransitions`)
- `runs.status`: `running → failed` (valid per `validRunTransitions`)
- `hitl_requests.status`: `pending → rejected`

**ResumeRunJob:** The job enqueued on approval must trigger the pipeline executor to continue from the suspended step. If `ResumeRunJob` does not yet exist as a River job type, define it in `backend/internal/adapter/river/` alongside the existing job types. The job payload must include `RunID uuid.UUID` and `StepID uuid.UUID`. The pipeline executor's River worker must handle this job type by fetching the run and continuing from the given step.

**Idempotency check (Go):**
```go
if hitlReq.Status != model.HITLStatusPending {
    return nil, errors.NewConflict(
        fmt.Sprintf("HITL request %s is already %s", hitlReq.ID, hitlReq.Status),
    )
}
```

**Handler — optional body decoding for reject:**
```go
var body struct {
    Reason string `json:"reason"`
}
if r.ContentLength > 0 {
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        renderError(w, errors.NewValidation("body", "invalid JSON"))
        return
    }
}
// body.Reason is "" if not provided — HITLService handles this correctly
```

**Linting rules to watch:**
- Use `_ = json.NewEncoder(w).Encode(...)` or `renderJSON` helper if encode errors are intentionally ignored
- Rename unused params in mock methods to `_`
- `errcheck` is enforced: never silently drop `error` return values

### Testing Requirements

**Unit tests in `hitl_service_test.go`:**

Mock interfaces needed (hand-written):
- `MockHITLRepository` — `Create`, `GetByRunStepID`, `GetPendingByRunID`, `UpdateStatus`
- `MockRunRepository` — `GetRun`, `UpdateRunStatus`, `UpdateRunStepStatus`
- `MockEventPublisher` — `Publish`
- `MockJobQueue` — `Enqueue`

Table-driven test cases for `Approve`:
1. Happy path — all mocks succeed; verify `UpdateStatus` called with `HITLStatusApproved`, `UpdateRunStepStatus` called with `StepStatusRunning`, event published, job enqueued
2. Run not found — `GetRun` returns not-found; error propagated, no further calls
3. Project mismatch — `run.ProjectID != projectID`; 404 returned
4. No pending HITL request — `GetPendingByRunID` returns not-found; `HITL_REQUEST_NOT_FOUND` returned
5. Already approved — `hitlReq.Status == HITLStatusApproved`; 409 conflict, no mutation

Table-driven test cases for `Reject`:
1. Happy path with reason — `UpdateStatus` called with `HITLStatusRejected` and reason, step set to `failed`, run set to `failed`, event published with reason in payload
2. Happy path without reason — `error_message = "HITL_REJECTED"` (no suffix), event payload has empty reason
3. Already rejected — 409 conflict, no mutation
4. Run project mismatch — 404 returned

**golangci-lint:** Run `golangci-lint run ./...` from `backend/` — must pass before commit. CI enforces this.

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
