# Story 3-16: Cancel Run endpoint and UI

Status: ready-for-dev

## Story

As a project operator,
I want to cancel a running (or paused/pending) pipeline run,
so that I can stop wasted compute when a run is no longer needed and free the story for a new attempt.

## Acceptance Criteria (BDD)

**AC1: Cancel a running run**
- **Given** a run exists with status `running` and one step is currently executing in a Docker container
- **When** I POST to `/api/v1/projects/{projectId}/runs/{runId}/cancel`
- **Then** HTTP 200 is returned with the updated run object (status = `cancelled`)
- **And** the active agent container is stopped via `ContainerManager.Stop`
- **And** the currently running step is marked `cancelled` with `completed_at = now`
- **And** all remaining `pending` steps are marked `cancelled` with `completed_at = now`
- **And** the run is marked `cancelled` with `completed_at = now`
- **And** the associated story status is set back to `backlog`
- **And** SSE events are published: `step.cancelled` (for each affected step), `run.cancelled`, `story.status_updated`

**AC2: Cancel a paused run**
- **Given** a run exists with status `paused`
- **When** I POST to `/api/v1/projects/{projectId}/runs/{runId}/cancel`
- **Then** HTTP 200 is returned with the updated run object (status = `cancelled`)
- **And** all `pending` steps are marked `cancelled`
- **And** the run is marked `cancelled`
- **And** the associated story status is set back to `backlog`
- **And** SSE events are published for all state changes

**AC3: Cancel a pending run**
- **Given** a run exists with status `pending` (not yet picked up by the executor)
- **When** I POST to `/api/v1/projects/{projectId}/runs/{runId}/cancel`
- **Then** HTTP 200 is returned with the updated run object (status = `cancelled`)
- **And** all `pending` steps are marked `cancelled`
- **And** the run is marked `cancelled`

**AC4: Conflict - run already in terminal state**
- **Given** a run exists with status `completed`, `failed`, or `cancelled`
- **When** I POST to `/api/v1/projects/{projectId}/runs/{runId}/cancel`
- **Then** HTTP 409 is returned with error code `INVALID_STATE_TRANSITION`

**AC5: Not found - run does not exist or wrong project**
- **Given** the `runId` does not exist or does not belong to `projectId`
- **When** I POST to `/api/v1/projects/{projectId}/runs/{runId}/cancel`
- **Then** HTTP 404 is returned

**AC6: Cancel button in frontend**
- **Given** I am viewing the RunDetailView for a run with status `running`, `paused`, or `pending`
- **When** I click the "Cancel" button
- **Then** a confirmation dialog is shown
- **And** upon confirmation, the cancel API is called
- **And** a success toast is shown on success
- **And** the run status and step statuses are updated reactively in the UI

**AC7: Cancel button visibility**
- **Given** I am viewing the RunDetailView
- **When** the run is in a terminal state (`completed`, `failed`, `cancelled`)
- **Then** the "Cancel" button is NOT displayed

**AC8: Cancel an epic run**
- **Given** an epic run exists with status `running` or `paused`
- **When** I POST to `/api/v1/projects/{projectId}/epics/{epicId}/runs/{runId}/cancel`
- **Then** HTTP 200 is returned with the updated run (status = `cancelled`)
- **And** the same cancellation logic applies as AC1/AC2

## Technical Notes

### State Machine

The `cancelled` status and transitions are already defined in `backend/internal/domain/model/run.go`:

```go
// Valid run transitions (already exist):
RunStatusPending  -> RunStatusCancelled  // AC3
RunStatusRunning  -> RunStatusCancelled  // AC1
RunStatusPaused   -> RunStatusCancelled  // AC2

// Valid step transitions (already exist):
StepStatusPending -> StepStatusCancelled
StepStatusRunning -> StepStatusCancelled
```

No changes to the state machine model are needed.

### Cancel vs Pause Semantics

- **Pause** is soft: the current step completes, no new steps start, run can be resumed.
- **Cancel** is hard: the active container is killed, the current step is marked cancelled immediately, all pending steps are cancelled, story resets to `backlog`. The run cannot be resumed.

### Container Cleanup

When the run status is `running`, the service must find the currently executing step's `container_id` (stored on the `RunStep` record by the `agent_run` action) and call `ContainerManager.Stop(ctx, containerID)`. If the container is already gone (e.g., race condition), the error should be logged but not propagated to the caller.

### Executor Awareness

The `PipelineExecutor.ExecuteRun` loop already checks for `ctx.Done()` and for `paused` status between steps. For cancel, the approach is:
1. `CancelRun` updates the run status to `cancelled` in the DB.
2. The executor's next pause-check iteration (between steps) will see the cancelled status and stop.
3. For the currently executing step, the container is stopped externally by `CancelRun`, which causes the action to return an error, which the executor handles.

Since the executor uses a River job with its own context, and cancel operates via direct DB status update + container stop, no context cancellation propagation is needed. The executor naturally terminates when:
- The container it is waiting on is killed (returns non-zero exit code / error)
- The next DB status check sees `cancelled`

### Story Status Reset

On cancel, the story status should be set back to `backlog` to allow re-running. This differs from `failed` (which sets story to `failed`). The rationale: cancellation is a deliberate user action, not a failure, so the story should be immediately launchable again.

### File Paths (exact)

| File | Action |
|------|--------|
| `api/openapi.yaml` | Add `POST /projects/{projectId}/runs/{runId}/cancel` and `POST /projects/{projectId}/epics/{epicId}/runs/{runId}/cancel` endpoints |
| `backend/internal/api/handler/gen_server.go` | Regenerated by oapi-codegen -- do not edit manually |
| `backend/internal/domain/service/run_service.go` | Add `CancelRun` and `CancelEpicRun` methods |
| `backend/internal/domain/service/run_service_test.go` | Add `CancelRun` unit tests |
| `backend/internal/api/handler/run_handler.go` | Add `CancelRun` and `CancelEpicRun` handler methods |
| `backend/internal/api/handler/run_handler_test.go` | Add `CancelRun` handler tests |
| `backend/internal/api/handler/server.go` | Add `CancelRun` and `CancelEpicRun` delegation methods |
| `backend/cmd/api/wire.go` | Update `RunService` constructor if new deps added |
| `backend/cmd/api/wire_gen.go` | Regenerated by wire -- do not edit manually |
| `frontend/src/api/generated/schema.d.ts` | Regenerated by openapi-typescript -- do not edit manually |
| `frontend/src/stores/runs.ts` | Add `cancelRun` action and `isCancelling` state |
| `frontend/src/views/RunDetailView.vue` | Add Cancel button with confirmation dialog |

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add cancel endpoints to OpenAPI spec
  - [ ] Add `POST /projects/{projectId}/runs/{runId}/cancel` to `api/openapi.yaml` (copy pause endpoint pattern, change operationId to `cancelRun`, summary to "Cancel a pipeline run", description to explain hard cancellation semantics)
  - [ ] Add `POST /projects/{projectId}/epics/{epicId}/runs/{runId}/cancel` to `api/openapi.yaml` (operationId `cancelEpicRun`)
  - [ ] Run `cd backend && make generate` to regenerate `internal/api/handler/gen_server.go`
  - [ ] Run `cd frontend && npm run generate-api` to regenerate TypeScript types

- [ ] [BACK] Task 2: Implement `CancelRun` in `RunService`
  - [ ] Add `containerMgr port.ContainerManager` to `RunService` struct and `NewRunService` constructor (needed to stop the active container)
  - [ ] Implement `CancelRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error)` in `backend/internal/domain/service/run_service.go`:
    1. Fetch run by ID, verify `run.ProjectID == projectID` (return 404 if mismatch)
    2. Validate transition to `cancelled` via `model.ValidateRunTransition` (returns 409 if invalid)
    3. Fetch all steps via `runRepo.ListRunStepsByRun`
    4. Find the currently running step (if any) and stop its container via `containerMgr.Stop(ctx, *step.ContainerID)` -- log and continue on error
    5. Mark the running step as `cancelled` via `runRepo.UpdateRunStepStatus`
    6. Mark all `pending` steps as `cancelled` via `runRepo.UpdateRunStepStatus` in a loop
    7. Mark the run as `cancelled` via `runRepo.UpdateRunStatus` with `completedAt = now`
    8. Transition the story status to `backlog` via `storyRepo.Update` (best-effort, log errors)
    9. Publish SSE events: `step.cancelled` for each cancelled step, `run.cancelled`, `story.status_updated`
    10. Return the updated run
  - [ ] Implement `CancelEpicRun(ctx context.Context, projectID, epicID, runID uuid.UUID) (*model.Run, error)` as a thin wrapper that delegates to `CancelRun(ctx, projectID, runID)` (same pattern as `PauseEpicRun`)

- [ ] [BACK] Task 3: Add `CancelRun` and `CancelEpicRun` handler methods
  - [ ] Implement `CancelRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath)` in `backend/internal/api/handler/run_handler.go` (copy PauseRun pattern)
  - [ ] Implement `CancelEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath)` in `backend/internal/api/handler/run_handler.go`
  - [ ] Add `CancelRun` and `CancelEpicRun` delegation methods to `backend/internal/api/handler/server.go`

- [ ] [BACK] Task 4: Update DI wiring
  - [ ] Update `backend/cmd/api/wire.go` to inject `port.ContainerManager` into `NewRunService`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] [BACK] Task 5: Write unit tests for `CancelRun`
  - [ ] Add `containerMgr` mock to test file (or extend existing mocks):
    ```go
    type mockContainerManager struct {
        stopFn func(ctx context.Context, containerID string) error
    }
    func (m *mockContainerManager) Stop(ctx context.Context, containerID string) error {
        if m.stopFn != nil { return m.stopFn(ctx, containerID) }
        return nil
    }
    // Implement remaining interface methods with no-op stubs
    ```
  - [ ] Add table-driven tests in `backend/internal/domain/service/run_service_test.go`:

    | Test | Scenario | Expected |
    |------|----------|----------|
    | `TestCancelRun_RunningWithContainer` | Run is `running`, step has `container_id` | Container stopped, step cancelled, pending steps cancelled, run cancelled, story reset to backlog |
    | `TestCancelRun_RunningNoContainer` | Run is `running`, step has no `container_id` | Step cancelled, run cancelled (no container stop attempted) |
    | `TestCancelRun_Paused` | Run is `paused` | Pending steps cancelled, run cancelled, story reset to backlog |
    | `TestCancelRun_Pending` | Run is `pending`, all steps `pending` | All steps cancelled, run cancelled |
    | `TestCancelRun_AlreadyCompleted` | Run is `completed` | Returns `INVALID_STATE_TRANSITION` error |
    | `TestCancelRun_AlreadyFailed` | Run is `failed` | Returns `INVALID_STATE_TRANSITION` error |
    | `TestCancelRun_AlreadyCancelled` | Run is `cancelled` | Returns `INVALID_STATE_TRANSITION` error |
    | `TestCancelRun_NotFound` | Run does not exist | Returns `NOT_FOUND` error |
    | `TestCancelRun_WrongProject` | Run exists but `projectID` mismatch | Returns `NOT_FOUND` error |
    | `TestCancelRun_ContainerStopError` | Container stop fails | Run is still cancelled (error logged, not propagated) |

  - [ ] Add handler tests in `backend/internal/api/handler/run_handler_test.go`:

    | Test | Scenario | Expected HTTP |
    |------|----------|--------------|
    | `TestCancelRun_Success` | Service returns cancelled run | 200 + Run JSON |
    | `TestCancelRun_NotFound` | Service returns not_found | 404 |
    | `TestCancelRun_Conflict` | Service returns invalid_state_transition | 409 |

- [ ] [BACK] Task 6: Lint and test
  - [ ] Run `cd backend && golangci-lint run ./...` -- must pass
  - [ ] Run `cd backend && go test ./... -short` -- must pass

- [ ] [FRONT] Task 7: Add `cancelRun` action to runs store
  - [ ] Add `isCancelling` ref to `frontend/src/stores/runs.ts`
  - [ ] Add `cancelRun(projectId: string, runId: string)` async function following the `pauseRun` pattern:
    ```typescript
    async function cancelRun(projectId: string, runId: string) {
      isCancelling.value = true
      try {
        const { data, error } = await apiClient.POST(
          '/projects/{projectId}/runs/{runId}/cancel',
          { params: { path: { projectId, runId } } },
        )
        if (error) throw error
        if (data) {
          updateRunStatus(runId, data.status)
        }
        return data
      } finally {
        isCancelling.value = false
      }
    }
    ```
  - [ ] Export `isCancelling` and `cancelRun` from the store

- [ ] [FRONT] Task 8: Add Cancel button with confirmation dialog to RunDetailView
  - [ ] Add `canCancel` computed: `run.value?.status === 'running' || run.value?.status === 'paused' || run.value?.status === 'pending'`
  - [ ] Import `ConfirmDialog` from PrimeVue and `useConfirm` composable
  - [ ] Add Cancel button next to existing Pause/Resume buttons:
    ```vue
    <Button
      v-if="canCancel"
      label="Cancel"
      icon="pi pi-times-circle"
      severity="danger"
      :loading="runsStore.isCancelling"
      data-testid="cancel-run-btn"
      @click="confirmCancel"
    />
    ```
  - [ ] Implement `confirmCancel()` function that shows a PrimeVue `ConfirmDialog` with:
    - Header: "Cancel Run"
    - Message: "Are you sure you want to cancel this run? This action cannot be undone. The active container will be stopped and all pending steps will be cancelled."
    - Accept label: "Cancel Run" (severity danger)
    - Reject label: "Keep Running"
  - [ ] On accept, call `handleCancel()` which calls `runsStore.cancelRun(projectId, runId)` and shows toast on success/error
  - [ ] Add `<ConfirmDialog />` component to the template if not already present

- [ ] [FRONT] Task 9: Frontend lint and type check
  - [ ] Run `cd frontend && npm run lint` -- must pass
  - [ ] Run `cd frontend && npm run type-check` -- must pass

## Dev Notes

### Dependencies

- Story 3-1 (DONE): `RunRepository` port and DB tables exist; state machine transitions are defined
- Story 3-4 (DONE): `ContainerManager` port and Docker adapter exist with `Stop` method
- Story 3-7 (DONE): `PipelineExecutor` exists with `handleCancellation` method
- Story 3-10 (DONE): `LaunchRun` and Pause/Resume patterns established
- Story 3-11 (DONE): `RunDetailView` with Pause/Resume buttons exists

### Architecture Requirements

- `CancelRun` is a **synchronous** operation from the API perspective -- it updates DB records and stops the container, then returns. The executor's River job will terminate naturally when the container it is waiting on is killed.
- Container stop is **best-effort**: if the container is already gone (race condition with executor finishing), the error is logged but the cancellation still proceeds.
- Story status reset to `backlog` is **best-effort**: if the story update fails, the cancellation still succeeds. This mirrors the `updateStoryStatus` pattern in `PipelineExecutor`.
- No new sqlc queries are needed -- existing `UpdateRunStatus`, `UpdateRunStepStatus`, and `ListRunStepsByRun` cover all DB operations.
- No new DB migrations are needed -- `cancelled` status already exists in the enum.

### Technical Specifications

#### OpenAPI spec -- `api/openapi.yaml`

Add after the existing `/projects/{projectId}/runs/{runId}/resume` block:

```yaml
  /projects/{projectId}/runs/{runId}/cancel:
    post:
      operationId: cancelRun
      summary: Cancel a pipeline run
      description: >
        Hard-cancels a run. If the run is currently executing, the active agent
        container is stopped (SIGTERM + 10s timeout + SIGKILL). The currently
        running step and all pending steps are marked as cancelled. The
        associated story is reset to backlog status. Returns 409 if the run is
        already in a terminal state (completed, failed, or cancelled).
      tags: [runs]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/RunIdPath"
      responses:
        "200":
          description: Run cancelled successfully
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
```

Add after the existing `/projects/{projectId}/epics/{epicId}/runs/{runId}/resume` block:

```yaml
  /projects/{projectId}/epics/{epicId}/runs/{runId}/cancel:
    post:
      operationId: cancelEpicRun
      summary: Cancel an in-progress epic run
      description: >
        Cancels an epic run. When story 7-2 implements the epic run model with
        parent/child relationships, this method will also cancel pending child runs.
      tags: [epic-runs]
      parameters:
        - $ref: "#/components/parameters/ProjectIdPath"
        - $ref: "#/components/parameters/EpicIdPath"
        - $ref: "#/components/parameters/RunIdPath"
      responses:
        "200":
          description: Epic run cancelled successfully
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
```

#### `CancelRun` service method -- `backend/internal/domain/service/run_service.go`

```go
// CancelRun hard-cancels a run: stops the active container, marks running/pending steps
// as cancelled, transitions the run to cancelled, and resets the story to backlog.
func (s *RunService) CancelRun(ctx context.Context, projectID, runID uuid.UUID) (*model.Run, error) {
    run, err := s.runRepo.GetRun(ctx, runID)
    if err != nil {
        return nil, err
    }

    if run.ProjectID != projectID {
        return nil, errors.NewNotFound("run", runID)
    }

    if err := model.ValidateRunTransition(run.Status, model.RunStatusCancelled); err != nil {
        return nil, err
    }

    // Fetch all steps
    steps, err := s.runRepo.ListRunStepsByRun(ctx, runID)
    if err != nil {
        return nil, err
    }

    now := time.Now()
    cancelMsg := "cancelled by user"

    // Stop active container (if any running step has a container_id)
    for _, step := range steps {
        if step.Status == model.StepStatusRunning && step.ContainerID != nil && *step.ContainerID != "" {
            if s.containerMgr != nil {
                if stopErr := s.containerMgr.Stop(ctx, *step.ContainerID); stopErr != nil {
                    // Log but do not fail -- container may already be gone
                    // (use structured logging in real implementation)
                }
            }
        }
    }

    // Cancel running and pending steps
    for _, step := range steps {
        if step.Status == model.StepStatusRunning || step.Status == model.StepStatusPending ||
           step.Status == model.StepStatusWaitingApproval {
            if _, err := s.runRepo.UpdateRunStepStatus(ctx, step.ID, model.StepStatusCancelled, nil, &now, &cancelMsg); err != nil {
                return nil, err
            }

            s.publishRunEvent(ctx, run.ProjectID, step.ID, "cancelled", map[string]any{
                "run_id":    runID.String(),
                "step_id":   step.ID.String(),
                "step_name": step.StepName,
                "status":    string(model.StepStatusCancelled),
            })
        }
    }

    // Cancel the run
    updated, err := s.runRepo.UpdateRunStatus(ctx, runID, model.RunStatusCancelled, nil, &now, nil, &cancelMsg)
    if err != nil {
        return nil, err
    }

    s.publishRunEvent(ctx, updated.ProjectID, updated.ID, "cancelled", map[string]any{
        "run_id":       runID.String(),
        "status":       string(model.RunStatusCancelled),
        "cancelled_at": now.Format(time.RFC3339),
    })

    // Reset story to backlog (best-effort)
    s.resetStoryToBacklog(ctx, run)

    return updated, nil
}

// CancelEpicRun cancels an epic run. Placeholder that delegates to CancelRun.
// When story 7-2 implements parent/child relationships, this will also cancel child runs.
func (s *RunService) CancelEpicRun(ctx context.Context, projectID, _ uuid.UUID, runID uuid.UUID) (*model.Run, error) {
    return s.CancelRun(ctx, projectID, runID)
}

// resetStoryToBacklog resets the story associated with a run back to backlog status.
// Errors are logged without propagating -- this is best-effort.
func (s *RunService) resetStoryToBacklog(ctx context.Context, run *model.Run) {
    if run.StoryID == uuid.Nil {
        return
    }

    story, err := s.storyRepo.GetByID(ctx, run.StoryID)
    if err != nil {
        return
    }

    story.Status = model.StoryStatusBacklog
    if _, err := s.storyRepo.Update(ctx, story); err != nil {
        return
    }

    s.publishRunEvent(ctx, run.ProjectID, run.StoryID, "status_updated", map[string]any{
        "story_id": run.StoryID.String(),
        "run_id":   run.ID.String(),
        "status":   model.StoryStatusBacklog,
    })
}
```

> **Note:** The `publishRunEvent` helper currently publishes with `entityType = "run"`. For step events, the implementation should use a dedicated `publishStepEvent` helper or generalize the `publishRunEvent` to accept `entityType`. See the `PipelineExecutor.publishEvent` method for the generalized pattern. The actual implementation should use the same event publishing pattern as `PipelineExecutor`.

#### `CancelRun` handler -- `backend/internal/api/handler/run_handler.go`

```go
// CancelRun handles POST /projects/{projectId}/runs/{runId}/cancel.
func (h *RunHandler) CancelRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
    run, err := h.service.CancelRun(r.Context(), projectID, runID)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }

    writeJSON(w, http.StatusOK, toAPIRun(run))
}

// CancelEpicRun handles POST /projects/{projectId}/epics/{epicId}/runs/{runId}/cancel.
func (h *RunHandler) CancelEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
    run, err := h.service.CancelEpicRun(r.Context(), projectID, epicID, runID)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }

    writeJSON(w, http.StatusOK, toAPIRun(run))
}
```

Add to `server.go`:

```go
// CancelRun delegates to RunHandler.
func (s *Server) CancelRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath) {
    s.runs.CancelRun(w, r, projectID, runID)
}

// CancelEpicRun delegates to RunHandler.
func (s *Server) CancelEpicRun(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, epicID EpicIdPath, runID RunIdPath) {
    s.runs.CancelEpicRun(w, r, projectID, epicID, runID)
}
```

### References

- `backend/internal/domain/model/run.go` -- state machine transitions (cancel already valid)
- `backend/internal/domain/service/run_service.go` -- `PauseRun`/`ResumeRun` as pattern reference
- `backend/internal/domain/service/pipeline_executor.go` -- `handleCancellation` method shows the executor-side cancel pattern
- `backend/internal/api/handler/run_handler.go` -- existing handler patterns
- `backend/internal/api/handler/server.go` -- delegation pattern
- `backend/internal/domain/port/container_manager.go` -- `ContainerManager.Stop` interface
- `backend/internal/adapter/docker/container_manager.go` -- Docker `Stop` implementation (SIGTERM + 10s + SIGKILL)
- `backend/internal/domain/port/run_repository.go` -- `UpdateRunStatus`, `UpdateRunStepStatus`, `ListRunStepsByRun`
- `frontend/src/views/RunDetailView.vue` -- existing Pause/Resume button area
- `frontend/src/stores/runs.ts` -- existing `pauseRun`/`resumeRun` store actions
- `api/openapi.yaml` -- existing pause/resume endpoint definitions as template

## Dev Agent Record

_To be filled in during implementation._

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-22 | 1.0 | Initial story creation | Arch |
