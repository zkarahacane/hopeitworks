# Story 3-18: Retry failed step endpoint and UI button

Status: ready-for-dev

## Story

As a **platform user monitoring pipeline runs**,
I want to **manually retry a failed step from the run detail page**,
so that **I can recover a failed run without relaunching the entire pipeline from scratch**.

## Context

The domain model already supports retries (`RunStep.ParentStepID`, `RetryCount`, `RetryType`), and the `IncrementalRetryAction` (in `backend/internal/adapter/action/incremental_retry.go`) implements the full retry logic with tests. The `PipelineExecutor.handleStepFailure()` handles auto-retries based on `RetryPolicy`. However, there is no REST endpoint to trigger a manual retry, and the frontend `RetryStepEntry.vue` component displays retry history but cannot trigger new retries.

This story adds the missing manual retry path: API endpoint, backend service method, job enqueuing, and a frontend "Retry" button on failed steps.

## Acceptance Criteria (BDD)

### Scenario 1: Successfully retry a failed step

```gherkin
Given a run in "failed" status
  And the run has a step in "failed" status
  And the step has not exceeded max retries
When I POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry
Then the response status is 202 Accepted
  And the response body contains the new retry RunStep with:
    | field         | value                        |
    | status        | pending                      |
    | parent_step_id| {original stepId}            |
    | retry_count   | {previous retry_count + 1}   |
    | retry_type    | incremental or full           |
  And the run status transitions back to "running"
  And the run is re-enqueued in the River job queue
  And a "step.retried" SSE event is published
```

### Scenario 2: Retry blocked — step is not in failed status

```gherkin
Given a run with a step in "completed" status
When I POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry
Then the response status is 409 Conflict
  And the error code is "STEP_NOT_RETRYABLE"
  And the error message indicates the step must be in failed status
```

### Scenario 3: Retry blocked — max retries exceeded

```gherkin
Given a run with a failed step
  And the step's retry_count equals the pipeline config max_retries
When I POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry
Then the response status is 409 Conflict
  And the error code is "RETRY_MAX_EXCEEDED"
  And the error message includes the max retry count
```

### Scenario 4: Retry blocked — run has an active retry already in progress

```gherkin
Given a run in "running" status (retry already in progress)
When I POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry
Then the response status is 409 Conflict
  And the error code is "RUN_ALREADY_ACTIVE"
```

### Scenario 5: Step or run not found

```gherkin
Given a non-existent stepId
When I POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry
Then the response status is 404 Not Found
```

### Scenario 6: Frontend retry button visibility

```gherkin
Given I am on the run detail page
  And the run has a step with status "failed"
Then I see a "Retry" button next to the failed step
  And the button is not visible for steps with other statuses
```

### Scenario 7: Frontend retry button interaction

```gherkin
Given I am on the run detail page with a failed step
When I click the "Retry" button on the failed step
Then a confirmation dialog appears
When I confirm the retry
Then the API is called
  And a success toast appears: "Step retry queued"
  And the run detail refreshes (via SSE or refetch)
  And the retry button becomes disabled while the run is active
```

### Scenario 8: Frontend retry button — max retries reached

```gherkin
Given a failed step that has reached max retries
Then the "Retry" button is disabled
  And a tooltip says "Max retries reached"
```

## Technical Notes

### API Endpoint

Add to `api/openapi.yaml`:

```yaml
/projects/{projectId}/runs/{runId}/steps/{stepId}/retry:
  post:
    operationId: retryFailedStep
    summary: Retry a failed pipeline step
    description: >
      Creates a new retry step linked to the original failed step via parent_step_id.
      Transitions the run back to running and re-enqueues it for execution.
      Returns 409 if the step is not failed, max retries are exceeded, or the run is already active.
    tags: [runs]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/RunIdPath"
      - $ref: "#/components/parameters/StepIdPath"
    responses:
      "202":
        description: Retry step created and run re-enqueued
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/RunStep"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "404":
        $ref: "#/components/responses/NotFound"
      "409":
        $ref: "#/components/responses/Conflict"
```

### Backend Service — `RunService.RetryFailedStep()`

New method on `RunService` in `backend/internal/domain/service/run_service.go`:

```go
func (s *RunService) RetryFailedStep(ctx context.Context, projectID, runID, stepID uuid.UUID) (*model.RunStep, error)
```

Logic:
1. **Fetch run** via `runRepo.GetRun(ctx, runID)`. Verify `run.ProjectID == projectID`.
2. **Guard: run must be "failed"**. If `run.Status` is `running` or `pending`, return `RUN_ALREADY_ACTIVE` conflict. If `completed` or `cancelled`, return `RUN_NOT_RETRYABLE`.
3. **Fetch step** via `runRepo.GetRunStep(ctx, stepID)`. Verify `step.RunID == runID`.
4. **Guard: step must be "failed"**. Return `STEP_NOT_RETRYABLE` if not.
5. **Resolve retry policy** from `run.PipelineConfigSnapshot`. Parse the snapshot JSON, find the matching step by `step_order`, extract `retry_policy.max_retries` (default 3) and `retry_policy.max_incremental` (default 2, implied from `RetryPolicy.RetryType`).
6. **Guard: max retries**. If `step.RetryCount >= maxRetries`, return `RETRY_MAX_EXCEEDED`.
7. **Determine retry type**: if `step.RetryCount >= maxIncremental` then `"full"`, else `"incremental"`.
8. **Create new RunStep** via `runRepo.CreateRetryRunStep()`:
   - `ID`: new UUID
   - `RunID`: same run
   - `StepName`, `StepOrder`, `Action`: copied from original step
   - `Status`: `pending`
   - `RetryCount`: `step.RetryCount + 1`
   - `RetryType`: computed above
   - `ParentStepID`: original `stepID`
9. **Transition run to "running"**: Update via `runRepo.UpdateRunStatus()` with `startedAt = now`.
10. **Enqueue River job**: `jobQueue.EnqueueExecuteRun(ctx, runID)`.
11. **Publish event**: `step.retried` with `{run_id, step_id, retry_step_id, retry_count, retry_type}`.
12. Return the new retry step.

**Important**: The `PipelineExecutor.ExecuteRun()` already skips completed steps and handles retry steps naturally since the new step will be `pending` and the executor iterates by `step_order`. However, the executor needs to handle the case where multiple steps exist with the same `step_order` (original + retries). The executor should pick the latest pending step for each `step_order`, skipping failed ones. This may require a minor adjustment to `ExecuteRun()` step filtering (see Task 2.3).

### Backend Handler

Add `RetryFailedStep` handler method to `RunHandler` in `backend/internal/api/handler/run_handler.go`:

```go
func (h *RunHandler) RetryFailedStep(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, runID RunIdPath, stepID StepIdPath) {
    step, err := h.service.RetryFailedStep(r.Context(), projectID, runID, stepID)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    writeJSON(w, http.StatusAccepted, toAPIRunStep(step))
}
```

### Executor Adjustment

When the run is re-enqueued after a retry, `PipelineExecutor.ExecuteRun()` fetches all steps via `ListRunStepsByRun()`. The list will now contain the original failed step AND the new pending retry step (both at the same `step_order`). The executor must:

- Skip steps with status `failed` or `cancelled` (already done for `completed`, extend to failed/cancelled).
- Process the new `pending` retry step.
- Continue with subsequent steps after the retry step completes.

Check that `ListRunStepsByRun` returns steps ordered by `(step_order, created_at)` so the retry step comes after the original. The current query orders by `step_order ASC` only; add `created_at ASC` as a tiebreaker.

### Frontend Changes

#### 1. Store — `stores/runs.ts`

Add `retryStep` action and `isRetrying` ref:

```typescript
const isRetrying = ref(false)

async function retryStep(projectId: string, runId: string, stepId: string) {
  isRetrying.value = true
  try {
    const { data, error } = await apiClient.POST(
      '/projects/{projectId}/runs/{runId}/steps/{stepId}/retry',
      { params: { path: { projectId, runId, stepId } } },
    )
    if (error) throw error
    return data
  } finally {
    isRetrying.value = false
  }
}
```

#### 2. Run Detail View — `views/RunDetailView.vue`

Add a "Retry" button inside the step timeline `#content` slot for failed steps:

```vue
<Button
  v-if="(item as RunStep).status === 'failed'"
  label="Retry"
  icon="pi pi-refresh"
  severity="warn"
  size="small"
  :loading="runsStore.isRetrying"
  :disabled="isRetryDisabled(item as RunStep)"
  v-tooltip="retryTooltip(item as RunStep)"
  data-testid="retry-step-btn"
  @click="handleRetryStep(item as RunStep)"
/>
```

Helper functions:
- `isRetryDisabled(step)`: returns true if `run.status === 'running'` or if the step has reached max retries (derive from pipeline config snapshot).
- `retryTooltip(step)`: returns `"Max retries reached"` or `"Run is already active"` accordingly.
- `handleRetryStep(step)`: shows a confirm dialog, then calls `runsStore.retryStep()`, shows toast on success/error.

#### 3. SSE Event Handling

Add `step.retried` to the `RUN_REFRESH_EVENTS` set in `useRunDetail.ts` so the view auto-refetches when a retry step is created.

#### 4. RetryStepEntry Integration

The existing `RetryStepEntry.vue` component already displays retry history (retry label, error context, log tail). After the retry endpoint is wired, the step timeline should render `RetryStepEntry` sub-entries for steps that have `parent_step_id` set. Group retry steps under their parent in the timeline rendering.

### Database

No migration needed. The `run_steps` table already has `retry_count`, `retry_type`, and `parent_step_id` columns. The `CreateRetryRunStep` sqlc query already exists.

### SQL Query Update

Update `ListRunStepsByRun` in `backend/queries/run_steps.sql` to add `created_at` as tiebreaker:

```sql
-- name: ListRunStepsByRun :many
SELECT * FROM run_steps
WHERE run_id = $1
ORDER BY step_order ASC, created_at ASC;
```

### Error Codes

| Code | HTTP | When |
|------|------|------|
| `STEP_NOT_RETRYABLE` | 409 | Step is not in `failed` status |
| `RETRY_MAX_EXCEEDED` | 409 | Step retry count >= max_retries from pipeline config |
| `RUN_ALREADY_ACTIVE` | 409 | Run is in `running` or `pending` status |
| `RUN_NOT_RETRYABLE` | 409 | Run is in `completed` or `cancelled` status |

### Events

Publish `step.retried` event via `EventPublisher`:

```json
{
  "run_id": "...",
  "step_id": "...",
  "retry_step_id": "...",
  "retry_count": 2,
  "retry_type": "incremental"
}
```

## Tasks / Subtasks

### 1. API Contract

- [ ] **1.1** Add `POST /projects/{projectId}/runs/{runId}/steps/{stepId}/retry` endpoint to `api/openapi.yaml`
- [ ] **1.2** Run `cd backend && make generate` to regenerate Go server interface
- [ ] **1.3** Run `cd frontend && npm run generate-api` to regenerate TypeScript types

### 2. Backend — Service Layer

- [ ] **2.1** Add `RetryFailedStep(ctx, projectID, runID, stepID)` method to `RunService` in `backend/internal/domain/service/run_service.go`
- [ ] **2.2** Implement validation guards: run status, step status, step ownership, max retries
- [ ] **2.3** Update `PipelineExecutor.ExecuteRun()` to skip `failed`/`cancelled` steps when iterating (so retry resumes from the new pending step)
- [ ] **2.4** Update `ListRunStepsByRun` SQL query to order by `(step_order ASC, created_at ASC)` for deterministic retry step ordering

### 3. Backend — Handler

- [ ] **3.1** Add `RetryFailedStep` handler method to `RunHandler` in `backend/internal/api/handler/run_handler.go`
- [ ] **3.2** Wire handler to the generated server interface in `wire.go` / router registration

### 4. Backend — Tests

- [ ] **4.1** Unit tests for `RunService.RetryFailedStep()`: happy path, step not failed, max retries exceeded, run already active, run not found, step not found, step not belonging to run
- [ ] **4.2** Unit test for updated `PipelineExecutor` skip-failed-steps behavior
- [ ] **4.3** Integration test: full retry flow (create run -> fail step -> retry -> verify new step created and run re-enqueued)

### 5. Frontend — Store

- [ ] **5.1** Add `retryStep()` action and `isRetrying` ref to `stores/runs.ts`

### 6. Frontend — Run Detail View

- [ ] **6.1** Add "Retry" button to failed steps in `RunDetailView.vue` step timeline
- [ ] **6.2** Add confirmation dialog before triggering retry
- [ ] **6.3** Add `isRetryDisabled()` and `retryTooltip()` helper functions
- [ ] **6.4** Add toast notifications for success/error
- [ ] **6.5** Add `step.retried` to SSE refresh events in `useRunDetail.ts`

### 7. Frontend — Retry History Display

- [ ] **7.1** Group retry steps under their parent step in the timeline (render `RetryStepEntry.vue` as sub-items for steps with `parent_step_id`)

### 8. Frontend — Tests

- [ ] **8.1** Unit test for `retryStep()` store action
- [ ] **8.2** Unit test for retry button visibility logic (only on failed steps, disabled when max retries reached)
- [ ] **8.3** E2E test: click retry on failed step, verify toast and run status change

### 9. Lint & Verify

- [ ] **9.1** `cd backend && golangci-lint run ./...`
- [ ] **9.2** `cd frontend && npm run lint && npm run type-check`
- [ ] **9.3** `cd backend && go test ./... -short`
- [ ] **9.4** `cd frontend && npm run test:unit`
