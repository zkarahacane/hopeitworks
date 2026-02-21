# Story 7.3: [BACK] Pause/Resume for Story and Epic Runs

Status: ready-for-dev

## Story

As a user, I want to pause and resume pipeline runs for both individual stories and entire epics, So that I can halt execution at a safe boundary without losing progress, and continue from where execution left off.

## Acceptance Criteria (BDD)

**AC1: POST /runs/{runId}/pause transitions a running story run to paused**
- **Given** a run with status `running`
- **When** POST /api/v1/projects/{projectId}/runs/{runId}/pause is called
- **Then** the response is 200 with the updated run object where `status = "paused"`
- **And** no new steps are launched after the pause is applied (the current in-flight step completes normally)

**AC2: POST /runs/{runId}/resume transitions a paused story run back to running**
- **Given** a run with status `paused`
- **When** POST /api/v1/projects/{projectId}/runs/{runId}/resume is called
- **Then** the response is 200 with the updated run object where `status = "running"`
- **And** execution resumes from the next pending step after the last completed one

**AC3: Pausing a non-running run returns 409 Conflict**
- **Given** a run with status `pending`, `completed`, `failed`, or `cancelled`
- **When** POST /api/v1/projects/{projectId}/runs/{runId}/pause is called
- **Then** the response is 409 with error code `RUN_NOT_PAUSABLE`

**AC4: Resuming a non-paused run returns 409 Conflict**
- **Given** a run with status `running`, `pending`, `completed`, `failed`, or `cancelled`
- **When** POST /api/v1/projects/{projectId}/runs/{runId}/resume is called
- **Then** the response is 409 with error code `RUN_NOT_RESUMABLE`

**AC5: Pause event is published on run.paused**
- **Given** a running run
- **When** it is successfully paused
- **Then** event `run.paused` is published to the event bus with payload `{ "run_id": "...", "status": "paused" }`

**AC6: Resume event is published on run.resumed**
- **Given** a paused run
- **When** it is successfully resumed
- **Then** event `run.resumed` is published to the event bus with payload `{ "run_id": "...", "status": "running" }`

**AC7: POST /epic-runs/{epicRunId}/pause pauses all not-yet-started child runs**
- **Given** an epic run with some child runs pending and at least one currently running
- **When** POST /api/v1/projects/{projectId}/epic-runs/{epicRunId}/pause is called
- **Then** the response is 200 with the updated epic run object where `status = "paused"`
- **And** all child runs with status `pending` are transitioned to `paused`
- **And** child runs already `running` continue to completion without interruption

**AC8: POST /epic-runs/{epicRunId}/resume resumes all paused child runs**
- **Given** an epic run with status `paused` and multiple child runs in `paused` status
- **When** POST /api/v1/projects/{projectId}/epic-runs/{epicRunId}/resume is called
- **Then** the response is 200 with the updated epic run object where `status = "running"`
- **And** all child runs with status `paused` are transitioned back to `running` (or `pending` for not-yet-started ones)
- **And** execution of paused child story runs is re-enqueued

**AC9: Pausing a non-running epic run returns 409 Conflict**
- **Given** an epic run with status `pending`, `completed`, or `failed`
- **When** POST /api/v1/projects/{projectId}/epic-runs/{epicRunId}/pause is called
- **Then** the response is 409 with error code `EPIC_RUN_NOT_PAUSABLE`

**AC10: Resuming a non-paused epic run returns 409 Conflict**
- **Given** an epic run with status `running`, `pending`, `completed`, or `failed`
- **When** POST /api/v1/projects/{projectId}/epic-runs/{epicRunId}/resume is called
- **Then** the response is 409 with error code `EPIC_RUN_NOT_RESUMABLE`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add `paused` status to runs table CHECK constraint (AC: #1, #2, #3, #4)
  - [ ] Create `backend/migrations/000016_add_paused_status_to_runs.up.sql`:
    ```sql
    ALTER TABLE runs DROP CONSTRAINT IF EXISTS runs_status_check;
    ALTER TABLE runs ADD CONSTRAINT runs_status_check
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled', 'paused'));
    ```
  - [ ] Create `backend/migrations/000016_add_paused_status_to_runs.down.sql`:
    ```sql
    ALTER TABLE runs DROP CONSTRAINT IF EXISTS runs_status_check;
    ALTER TABLE runs ADD CONSTRAINT runs_status_check
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'));
    ```

- [ ] [BACK] Task 2: Add `RunStatusPaused` constant and update state machine (AC: #1, #2, #3, #4)
  - [ ] Edit `backend/internal/domain/model/run.go`:
    - Add `RunStatusPaused RunStatus = "paused"` constant
    - Add `running → paused` and `paused → running` transitions to `validRunTransitions`
  - [ ] Edit `backend/internal/domain/model/run_test.go`: add table cases for `running→paused` (valid), `paused→running` (valid), `pending→paused` (invalid), `completed→paused` (invalid)

- [ ] [BACK] Task 3: Update OpenAPI spec with pause/resume endpoints (AC: #1, #2, #7, #8)
  - [ ] Add `paused` to the `Run.status` enum in `api/openapi.yaml` (currently `[pending, running, completed, failed, cancelled]`)
  - [ ] Add endpoint `POST /projects/{projectId}/runs/{runId}/pause` with operationId `pauseRun`; no request body; response 200 schema `$ref: Run`, response 409 `$ref: Conflict`
  - [ ] Add endpoint `POST /projects/{projectId}/runs/{runId}/resume` with operationId `resumeRun`; no request body; response 200 schema `$ref: Run`, response 409 `$ref: Conflict`
  - [ ] Add `EpicRunIdPath` parameter to `components/parameters` (if not already present): `name: epicRunId, in: path, required: true, schema: { type: string, format: uuid }`
  - [ ] Add endpoint `POST /projects/{projectId}/epic-runs/{epicRunId}/pause` with operationId `pauseEpicRun`; no request body; response 200 schema `$ref: EpicRunDetail`, response 409 `$ref: Conflict`
  - [ ] Add endpoint `POST /projects/{projectId}/epic-runs/{epicRunId}/resume` with operationId `resumeEpicRun`; no request body; response 200 schema `$ref: EpicRunDetail`, response 409 `$ref: Conflict`
  - [ ] Regenerate backend types: `cd backend && make generate`
  - [ ] Verify `PauseRun`, `ResumeRun`, `PauseEpicRun`, `ResumeEpicRun` methods appear in `gen_server.go`

- [ ] [BACK] Task 4: Implement `PauseRun` and `ResumeRun` in `RunService` (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Edit `backend/internal/domain/service/run_service.go`:
    - Add `PauseRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error)`:
      1. Fetch run; verify `run.ProjectID == projectID` (return `errors.NewNotFound` otherwise)
      2. If `run.Status != RunStatusRunning`, return `errors.NewConflict` with code `RUN_NOT_PAUSABLE` and message `"run <id> is not in running state (current: <status>)"`
      3. Call `runRepo.UpdateRunStatus(ctx, runID, RunStatusPaused, nil, nil, nil)` — do not set `completed_at`
      4. Publish `run.paused` event with payload `{ "run_id": run.ID, "status": "paused" }`
      5. Return updated run
    - Add `ResumeRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error)`:
      1. Fetch run; verify `run.ProjectID == projectID`
      2. If `run.Status != RunStatusPaused`, return `errors.NewConflict` with code `RUN_NOT_RESUMABLE` and message `"run <id> is not in paused state (current: <status>)"`
      3. Call `runRepo.UpdateRunStatus(ctx, runID, RunStatusRunning, nil, nil, nil)`
      4. Publish `run.resumed` event with payload `{ "run_id": run.ID, "status": "running" }`
      5. Re-enqueue execution via `jobQueue.Enqueue(ctx, jobs.ExecuteRun{RunID: runID})` so `PipelineExecutor.ExecuteRun` restarts from next pending step
      6. Return updated run

- [ ] [BACK] Task 5: Update `PipelineExecutor` to check for paused state before each step (AC: #1)
  - [ ] Edit `backend/internal/domain/service/pipeline_executor.go`, inside the step loop (`for i := range steps`):
    - After the existing cancellation check, add a paused check:
      ```go
      currentRun, err := e.runRepo.GetRun(ctx, runID)
      if err != nil {
          return err
      }
      if currentRun.Status == model.RunStatusPaused {
          e.logger.Info("pipeline execution paused before step",
              "run_id", runID, "step_order", step.StepOrder)
          return nil
      }
      ```
    - Only skip steps with status `pending` — already-running steps complete normally (the check fires before `executeStep` is called)

- [ ] [BACK] Task 6: Implement `PauseEpicRun` and `ResumeEpicRun` in `EpicRunService` (AC: #7, #8, #9, #10)
  - [ ] Edit `backend/internal/domain/service/epic_run_service.go`:
    - Add `PauseEpicRun(ctx context.Context, projectID, epicRunID uuid.UUID) (*model.EpicRun, error)`:
      1. Fetch epic run; verify `epicRun.ProjectID == projectID`
      2. If `epicRun.Status != EpicRunStatusRunning`, return conflict error with code `EPIC_RUN_NOT_PAUSABLE`
      3. Update epic run status to `paused` via `epicRunRepo.UpdateEpicRunStatus`
      4. List all `epic_run_stories` for this epic run; for each with `status = "pending"`, transition the associated run (if `run_id` is set) to `paused` via `runService.PauseRun`; for entries where `run_id` is nil (not yet started), update `epic_run_stories.status` to `"paused"` directly
      5. Publish `epic_run.paused` event
      6. Return updated epic run
    - Add `ResumeEpicRun(ctx context.Context, projectID, epicRunID uuid.UUID) (*model.EpicRun, error)`:
      1. Fetch epic run; verify `epicRun.ProjectID == projectID`
      2. If `epicRun.Status != EpicRunStatusPaused`, return conflict error with code `EPIC_RUN_NOT_RESUMABLE`
      3. Update epic run status to `running`
      4. List all `epic_run_stories`; for each with `status = "paused"`: if `run_id` is set, call `runService.ResumeRun`; if `run_id` is nil (not-yet-started), reset `epic_run_stories.status` to `"pending"` so the `ParallelGroupExecutor` picks it up again
      5. Publish `epic_run.resumed` event
      6. Return updated epic run

- [ ] [BACK] Task 7: Add `paused → running` transition to `ValidateEpicRunTransition` (AC: #7, #8)
  - [ ] Edit `backend/internal/domain/model/epic_run.go`:
    - Add `running → paused` and `paused → running` to the valid transitions in `ValidateEpicRunTransition`
    - Verify `EpicRunStatusPaused` constant already exists (added in story 7-2 migration); if not, add it

- [ ] [BACK] Task 8: Implement `PauseRun` and `ResumeRun` handlers in `RunHandler` (AC: #1, #2, #3, #4)
  - [ ] Edit `backend/internal/api/handler/run_handler.go`:
    - Add `PauseRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath)`:
      - Call `svc.PauseRun(r.Context(), projectID, runID)`
      - On `conflict` category error: `renderError(w, err)` (middleware maps to 409)
      - On success: `renderJSON(w, http.StatusOK, toRunResponse(run))`
    - Add `ResumeRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath)`:
      - Call `svc.ResumeRun(r.Context(), projectID, runID)`
      - Same error handling pattern

- [ ] [BACK] Task 9: Implement `PauseEpicRun` and `ResumeEpicRun` handlers in `EpicRunHandler` (AC: #7, #8, #9, #10)
  - [ ] Edit `backend/internal/api/handler/epic_run_handler.go`:
    - Add `PauseEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicRunID EpicRunIdPath)`:
      - Call `svc.PauseEpicRun(r.Context(), projectID, epicRunID)`
      - On success: `renderJSON(w, http.StatusOK, toEpicRunDetailResponse(epicRun))`
    - Add `ResumeEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicRunID EpicRunIdPath)`:
      - Call `svc.ResumeEpicRun(r.Context(), projectID, epicRunID)`

- [ ] [BACK] Task 10: Register new routes in `server.go` (AC: #1, #2, #7, #8)
  - [ ] Edit `backend/internal/api/handler/server.go`:
    - Add `PauseRun` and `ResumeRun` delegation to `RunHandler`
    - Add `PauseEpicRun` and `ResumeEpicRun` delegation to `EpicRunHandler`
  - [ ] Verify all four new generated interface methods are satisfied

- [ ] [BACK] Task 11: Update DI wiring (AC: all)
  - [ ] Edit `backend/cmd/api/wire.go` if `RunService` does not yet receive `EventPublisher` and `JobQueue` — add them if missing for the new pause/resume methods
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`
  - [ ] Verify `cd backend && go build ./...` compiles cleanly

- [ ] [BACK] Task 12: Write unit tests for `RunService` pause/resume (AC: #1–#6)
  - [ ] Edit `backend/internal/domain/service/run_service_test.go`:
    - Test `PauseRun` with running run → status transitions to `paused`, event `run.paused` published
    - Test `PauseRun` with non-running run (e.g., `completed`) → `RUN_NOT_PAUSABLE` conflict error returned
    - Test `PauseRun` with run belonging to different project → `not_found` error returned
    - Test `ResumeRun` with paused run → status transitions to `running`, event `run.resumed` published, job enqueued
    - Test `ResumeRun` with non-paused run → `RUN_NOT_RESUMABLE` conflict error returned
  - [ ] Use hand-written mocks for `port.RunRepository`, `port.EventPublisher`, `port.JobQueue`

- [ ] [BACK] Task 13: Write unit tests for `EpicRunService` pause/resume (AC: #7–#10)
  - [ ] Edit `backend/internal/domain/service/epic_run_service_test.go`:
    - Test `PauseEpicRun` with running epic run containing 3 pending child runs → all 3 child runs paused, epic run status = `paused`
    - Test `PauseEpicRun` with non-running epic run → `EPIC_RUN_NOT_PAUSABLE` conflict error
    - Test `ResumeEpicRun` with paused epic run → all paused child runs resumed, epic run status = `running`
    - Test `ResumeEpicRun` with non-paused epic run → `EPIC_RUN_NOT_RESUMABLE` conflict error

- [ ] [BACK] Task 14: Write unit tests for `PipelineExecutor` paused-state check (AC: #1)
  - [ ] Edit `backend/internal/domain/service/pipeline_executor_test.go`:
    - Test: run transitions to `paused` between step 1 and step 2 → executor exits after step 1 without executing step 2, no error returned

- [ ] [BACK] Task 15: Lint and validate (AC: all)
  - [ ] Run `cd backend && golangci-lint run ./...` — must pass with zero errors
  - [ ] Run `cd backend && go test ./... -short` — all unit tests green
  - [ ] Ensure no `fmt.Println`, `TODO` without story key, or hardcoded secrets introduced

## Dev Notes

### Dependencies

- Story 3-1 (done): `model.Run`, `RunStatus`, `ValidateRunTransition`, `RunRepository`, `RunService.TransitionRun` are available
- Story 3-7 (done): `PipelineExecutor.ExecuteRun` executes steps sequentially — the paused-state check in Task 5 inserts into its existing step loop
- Story 7-2 (done): `EpicRunService`, `ParallelGroupExecutor`, `epic_runs` table with `paused` status in the ENUM, `model.EpicRun`, `EpicRunRepository` are available
- The `epic_run_status` Postgres ENUM defined in story 7-2 already includes the `paused` value — no migration change needed for `epic_runs`
- The `runs` table uses a VARCHAR CHECK constraint (not an ENUM), so `paused` must be added via migration (Task 1)

### Architecture Requirements

**State machine additions (`model/run.go`):**

```go
const (
    RunStatusPaused RunStatus = "paused"
    // ... existing constants
)

var validRunTransitions = map[RunStatus][]RunStatus{
    RunStatusPending:  {RunStatusRunning, RunStatusCancelled},
    RunStatusRunning:  {RunStatusCompleted, RunStatusFailed, RunStatusCancelled, RunStatusPaused},
    RunStatusPaused:   {RunStatusRunning},
}
```

**Conflict error pattern for 409 responses:**

The existing error middleware maps `errors.CategoryConflict` to HTTP 409. Use `errors.NewConflict` for all pause/resume guard errors:

```go
// In RunService.PauseRun:
if run.Status != model.RunStatusRunning {
    return nil, &errors.DomainError{
        Category: errors.CategoryConflict,
        Code:     "RUN_NOT_PAUSABLE",
        Message:  fmt.Sprintf("run %s is not in running state (current: %s)", runID, run.Status),
    }
}

// In RunService.ResumeRun:
if run.Status != model.RunStatusPaused {
    return nil, &errors.DomainError{
        Category: errors.CategoryConflict,
        Code:     "RUN_NOT_RESUMABLE",
        Message:  fmt.Sprintf("run %s is not in paused state (current: %s)", runID, run.Status),
    }
}
```

**PipelineExecutor paused-state check (Task 5):**

The check must be placed inside the step loop, after the cancellation `select` and before `executeStep` is called. It re-fetches the run from the DB to pick up external status changes (the pause was written by the HTTP handler in a separate goroutine):

```go
// Inside ExecuteRun, for i := range steps loop:
select {
case <-ctx.Done():
    e.handleCancellation(run, step)
    return ctx.Err()
default:
}

// Paused-state check (re-read from DB to pick up external pause)
currentRun, err := e.runRepo.GetRun(ctx, runID)
if err != nil {
    return err
}
if currentRun.Status == model.RunStatusPaused {
    e.logger.Info("pipeline execution paused before step",
        "run_id", runID, "step_order", step.StepOrder)
    return nil // exit cleanly; resume will re-enqueue ExecuteRun
}
```

The current in-flight step (if `executeStep` is already executing) will complete normally. The paused check only gates the *next* step.

**Resume re-enqueue pattern:**

On `ResumeRun`, execution is re-launched by enqueuing a new `ExecuteRun` job. `PipelineExecutor.ExecuteRun` already skips steps that are not in `pending` status because it iterates steps sorted by `step_order` and each `executeStep` checks/transitions step status — steps already `completed` or `failed` are implicitly skipped (the transition from `pending → running` would fail for non-pending steps). Verify this skip behavior exists in `executeStep`; if not, add a guard:

```go
// In executeStep or in the step loop before calling executeStep:
if step.Status != model.StepStatusPending {
    continue // skip already-processed steps
}
```

**Epic run pause semantics:**

- "Pending" child runs whose `epic_run_stories.run_id` is nil (run not yet created) are handled by setting `epic_run_stories.status = "paused"` directly. The `ParallelGroupExecutor` must check this flag before launching a new run for a story entry.
- "Pending" child runs whose `run_id` is set (run created but queued) should call `runService.PauseRun` to set the run's DB status to `paused` before the executor picks it up.
- The `ParallelGroupExecutor.Execute` layer loop in story 7-2 must be updated to check for paused epic run status at the start of each layer (similar to the cancellation check), returning early if the epic run is paused. This prevents the executor from starting the next layer after one completes.

```go
// In ParallelGroupExecutor.Execute, at the top of the layer loop:
epicRun, err = e.epicRunRepo.GetEpicRun(ctx, epicRun.ID)
if err != nil {
    return err
}
if epicRun.Status == model.EpicRunStatusPaused {
    e.logger.Info("epic run paused, stopping layer execution",
        "epic_run_id", epicRun.ID, "group_index", i)
    return nil
}
```

### File Paths (exact)

| Purpose | Path |
|---------|------|
| OpenAPI spec | `api/openapi.yaml` |
| Migration 16 up | `backend/migrations/000016_add_paused_status_to_runs.up.sql` |
| Migration 16 down | `backend/migrations/000016_add_paused_status_to_runs.down.sql` |
| Run domain model | `backend/internal/domain/model/run.go` |
| Run model tests | `backend/internal/domain/model/run_test.go` |
| Epic run model | `backend/internal/domain/model/epic_run.go` |
| Run service | `backend/internal/domain/service/run_service.go` |
| Run service tests | `backend/internal/domain/service/run_service_test.go` |
| Epic run service | `backend/internal/domain/service/epic_run_service.go` |
| Epic run service tests | `backend/internal/domain/service/epic_run_service_test.go` |
| Pipeline executor | `backend/internal/domain/service/pipeline_executor.go` |
| Pipeline executor tests | `backend/internal/domain/service/pipeline_executor_test.go` |
| Run handler | `backend/internal/api/handler/run_handler.go` |
| Run handler tests | `backend/internal/api/handler/run_handler_test.go` |
| Epic run handler | `backend/internal/api/handler/epic_run_handler.go` |
| Server (route delegation) | `backend/internal/api/handler/server.go` |
| Generated server interface | `backend/internal/api/handler/gen_server.go` (regenerated — do not edit) |
| DI wiring | `backend/cmd/api/wire.go` |
| DI generated | `backend/cmd/api/wire_gen.go` (regenerated — do not edit) |

### Technical Specifications

**OpenAPI additions to `api/openapi.yaml`:**

```yaml
  /projects/{projectId}/runs/{runId}/pause:
    post:
      operationId: pauseRun
      summary: Pause a running pipeline run
      description: >
        Transitions the run to 'paused'. The current in-flight step completes normally;
        no new steps are launched until the run is resumed. Returns 409 if the run is
        not currently in 'running' state.
      tags: [runs]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/RunIdPath"
      responses:
        "200":
          description: Run paused
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Run"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"

  /projects/{projectId}/runs/{runId}/resume:
    post:
      operationId: resumeRun
      summary: Resume a paused pipeline run
      description: >
        Transitions the run back to 'running' and re-enqueues execution from the last
        completed step. Returns 409 if the run is not currently in 'paused' state.
      tags: [runs]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/RunIdPath"
      responses:
        "200":
          description: Run resumed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Run"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"

  /projects/{projectId}/epic-runs/{epicRunId}/pause:
    post:
      operationId: pauseEpicRun
      summary: Pause a running epic run
      description: >
        Transitions the epic run to 'paused'. All child runs not yet started are paused;
        any child run currently executing continues to completion. Returns 409 if the
        epic run is not currently in 'running' state.
      tags: [epics]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/EpicRunIdPath"
      responses:
        "200":
          description: Epic run paused
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/EpicRunDetail"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"

  /projects/{projectId}/epic-runs/{epicRunId}/resume:
    post:
      operationId: resumeEpicRun
      summary: Resume a paused epic run
      description: >
        Transitions the epic run back to 'running'. All paused child runs are resumed.
        Returns 409 if the epic run is not currently in 'paused' state.
      tags: [epics]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/EpicRunIdPath"
      responses:
        "200":
          description: Epic run resumed
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/EpicRunDetail"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "409":
          $ref: "#/components/responses/Conflict"
```

Also add to `components/parameters`:

```yaml
    EpicRunIdPath:
      name: epicRunId
      in: path
      required: true
      schema:
        type: string
        format: uuid
      description: Epic run UUID
```

And update `Run.status` enum:

```yaml
        status:
          type: string
          enum: [pending, running, completed, failed, cancelled, paused]
```

**Event payload pattern (consistent with story 7-2):**

```go
// run.paused
model.Event{
    ProjectID:  run.ProjectID,
    EntityType: "run",
    EntityID:   run.ID,
    Action:     "paused",
    Payload:    json.RawMessage(fmt.Sprintf(`{"run_id":%q,"status":"paused"}`, run.ID)),
}

// run.resumed
model.Event{
    ProjectID:  run.ProjectID,
    EntityType: "run",
    EntityID:   run.ID,
    Action:     "resumed",
    Payload:    json.RawMessage(fmt.Sprintf(`{"run_id":%q,"status":"running"}`, run.ID)),
}

// epic_run.paused
model.Event{
    ProjectID:  epicRun.ProjectID,
    EntityType: "epic_run",
    EntityID:   epicRun.ID,
    Action:     "paused",
    Payload:    json.RawMessage(fmt.Sprintf(`{"epic_run_id":%q,"status":"paused"}`, epicRun.ID)),
}
```

**golangci-lint compliance notes:**

- All new exported methods must have godoc comments
- `errcheck`: do not silently ignore `eventPub.Publish` errors — log on error with `slog.Error` but do not abort the pause/resume operation (publishing is best-effort)
- No `fmt.Println` — use structured slog logging throughout
- Rename unused mock parameters to `_` (revive)

### Testing Requirements

**`run_service_test.go` new cases:**

| Test name | Setup | Expected |
|-----------|-------|----------|
| `PauseRun/running run` | run.Status = running | status → paused, event run.paused published |
| `PauseRun/non-running run` | run.Status = completed | DomainError{Category: conflict, Code: RUN_NOT_PAUSABLE} |
| `PauseRun/wrong project` | run.ProjectID ≠ projectID | DomainError{Category: not_found} |
| `ResumeRun/paused run` | run.Status = paused | status → running, event run.resumed published, job enqueued |
| `ResumeRun/non-paused run` | run.Status = running | DomainError{Category: conflict, Code: RUN_NOT_RESUMABLE} |

**`pipeline_executor_test.go` new case:**

- Setup: 3-step run; mock `runRepo.GetRun` to return `status = paused` on the second call (i.e., between step 1 and step 2)
- Expected: `ExecuteRun` returns `nil` after step 1 completes; step 2 and step 3 `executeStep` are never called

**`run_handler_test.go` new cases:**

- `POST /pause` with running run → 200 with updated run object
- `POST /pause` with completed run → 409 with `"code": "RUN_NOT_PAUSABLE"`
- `POST /resume` with paused run → 200 with updated run object
- `POST /resume` with running run → 409 with `"code": "RUN_NOT_RESUMABLE"`

## Dependencies

- Story 3-1 (done): `model.Run`, `RunStatus`, state machine, `RunRepository`, `RunService`
- Story 3-7 (done): `PipelineExecutor.ExecuteRun` step loop
- Story 7-2 (must be merged first): `EpicRunService`, `ParallelGroupExecutor`, `epic_runs` table with `paused` in the Postgres ENUM, `EpicRunRepository`, `EpicRunHandler`

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-18 | Claude Sonnet 4.6 | Initial story creation |
