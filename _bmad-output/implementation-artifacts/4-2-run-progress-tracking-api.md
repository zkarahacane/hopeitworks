# Story 4.2: [BACK] Run Progress Tracking API

Status: ready-for-dev

## Story

As a frontend client, I want the Run API to return a computed `progress` field and support paginated project-level run listing, So that the UI can display accurate progress indicators without additional queries.

## Acceptance Criteria (BDD)

**AC1: `progress` field is present on Run responses**
- **Given** a run with 4 steps where 2 are `completed` and 2 are `pending`
- **When** `GET /api/v1/runs/{runId}` is called
- **Then** the response includes `"progress": 50` (integer, 0–100)
- **And** a run with zero steps returns `"progress": 0`
- **And** a run where all steps are `completed` returns `"progress": 100`

**AC2: `GetRun` handler returns full run with steps and progress**
- **Given** a valid `runId` that exists in the database
- **When** `GET /api/v1/runs/{runId}` is called by an authenticated user
- **Then** the response body is a `RunWithSteps` object including all step fields and the computed `progress`
- **And** HTTP 404 is returned if the run does not exist

**AC3: `ListRunsByProject` returns paginated runs with progress**
- **Given** a project with 35 runs
- **When** `GET /api/v1/projects/{projectId}/runs?page=2&per_page=20` is called
- **Then** the response contains 15 runs in `data[]` and `pagination.total = 35`, `pagination.page = 2`, `pagination.per_page = 20`
- **And** each run in `data[]` includes the `progress` field
- **And** HTTP 404 is returned if the project does not exist

**AC4: Progress is computed at service layer, not persisted**
- **Given** a run is updated (step status changes)
- **When** `GetRun` is called
- **Then** progress reflects the current step statuses fetched from `run_steps` at query time
- **And** there is no `progress` column in the database (computed only)

**AC5: OpenAPI spec is updated with the `progress` field**
- **Given** the `Run` and `RunWithSteps` schemas in `api/openapi.yaml`
- **When** the spec is reviewed
- **Then** both schemas include `progress: integer (0–100)` as a required field
- **And** backend and frontend generated types are regenerated from the updated spec

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add `progress` field to `Run` and `RunWithSteps` schemas in `api/openapi.yaml` (AC: #5)
  - [ ] Add `progress: { type: integer, minimum: 0, maximum: 100 }` as required to both `Run` and `RunWithSteps` schemas
  - [ ] Run `cd backend && make generate` to regenerate oapi-codegen types
  - [ ] Run `cd frontend && npm run generate-api` to regenerate TypeScript types

- [ ] [BACK] Task 2: Add `Progress` field to `model.Run` and implement `ComputeProgress` (AC: #1, #4)
  - [ ] Add `Progress int` field to `model.Run` struct in `backend/internal/domain/model/run.go`
  - [ ] Add method `ComputeProgress(steps []RunStep) int` on `Run` — counts steps with `status == completed`, divides by total, multiplies by 100, returns 0 if len(steps)==0
  - [ ] Write unit tests in `backend/internal/domain/model/run_test.go` covering: zero steps, partial completion, full completion, rounding

- [ ] [BACK] Task 3: Populate `Progress` in `RunService.GetRun` (AC: #1, #2, #4)
  - [ ] After fetching steps in `service.GetRun`, call `run.Progress = run.ComputeProgress(run.Steps)` before returning
  - [ ] `ListRunsByProject` and `ListRunsByStory` — for list endpoints, fetch steps per run and compute progress (accept N+1 for MVP; document the tradeoff)

- [ ] [BACK] Task 4: Expose `Progress` in `toAPIRun` and `toAPIRunWithSteps` converters (AC: #1, #2, #3)
  - [ ] In `backend/internal/api/handler/run_handler.go`, update `toAPIRun` to set `Progress: r.Progress`
  - [ ] Update `toAPIRunWithSteps` to set `Progress: r.Progress`
  - [ ] The `Progress` field is now part of the generated oapi-codegen `Run` and `RunWithSteps` types

- [ ] [BACK] Task 5: Verify `ListRunsByProject` handler returns correct pagination (AC: #3)
  - [ ] The handler already exists and delegates to `service.ListRunsByProject` — verify the `page`/`per_page` defaults (page=1, per_page=20) and max cap (per_page=100) are applied
  - [ ] Confirm `CountRunsByProject` is called and wired into the `Pagination` response object
  - [ ] Add validation: if `projectID` is not a valid UUID, return 400 via `writeErrorResponse`

- [ ] [BACK] Task 6: Add `ListRunStepsByRun` batch query for progress in list endpoints (AC: #3, #4)
  - [ ] For `ListRunsByProject`, loop over returned runs and call `runRepo.ListRunStepsByRun` per run, then call `run.ComputeProgress`
  - [ ] Accept N+1 for MVP — document with comment `// TODO(perf): batch fetch steps in single query for large lists`
  - [ ] If `ListRunsByStory` already follows the same pattern, apply consistently

- [ ] [BACK] Task 7: Write unit tests for progress computation in RunService (AC: #1, #2, #3, #4)
  - [ ] File: `backend/internal/domain/service/run_service_test.go`
  - [ ] Test `GetRun`: run with 3 steps (2 completed, 1 running) returns `Progress = 66`
  - [ ] Test `GetRun`: run with 0 steps returns `Progress = 0`
  - [ ] Test `ListRunsByProject`: progress is populated for each run in the result
  - [ ] Use hand-written mock `RunRepository` implementing `port.RunRepository`

- [ ] [BACK] Task 8: Write integration test for `GetRun` and `ListRunsByProject` (AC: #2, #3)
  - [ ] File: `backend/internal/adapter/postgres/run_repo_integration_test.go`
  - [ ] Tag with `if testing.Short() { t.Skip() }` guard
  - [ ] Test `GetRun` with real DB: create run + steps, call handler via httptest, assert 200 + `progress` field in JSON
  - [ ] Test `ListRunsByProject`: create 3 runs, assert pagination metadata and `progress` field in each item

## Dev Notes

### Dependencies

- Story 3.1 (runs/run_steps tables and RunRepository) — DONE. `port.RunRepository` has `GetRun`, `ListRunStepsByRun`, `ListRunsByProject`, `CountRunsByProject`.
- `run_handler.go` already implements `GetRun`, `ListRunsByProject`, `CreateRun`, `LaunchRun`, `ListRunsByStory` — no new handlers needed, only enhancements.
- OpenAPI spec update is required first (Task 1) since it drives code generation for both sides.

### Architecture Requirements

- `Progress` is a computed field, never persisted. It lives on `model.Run` as a transient value populated by the service layer after fetching steps.
- Do NOT add a `progress` column to the database. Do NOT add it to sqlc queries.
- The N+1 query pattern for list endpoints is acceptable for MVP. At expected scale (< 50 runs per project page), it adds negligible latency.
- `ComputeProgress` belongs on the domain model, not the service, to keep the computation testable in isolation.

### File Paths (exact)

```
api/openapi.yaml                                              (add progress to Run + RunWithSteps schemas)
backend/internal/domain/model/run.go                         (add Progress field + ComputeProgress method)
backend/internal/domain/model/run_test.go                    (new — unit tests for ComputeProgress)
backend/internal/domain/service/run_service.go               (populate Progress in GetRun + list methods)
backend/internal/domain/service/run_service_test.go          (new or extend — progress unit tests)
backend/internal/api/handler/run_handler.go                  (update toAPIRun + toAPIRunWithSteps)
backend/internal/adapter/postgres/run_repo_integration_test.go (new or extend — integration tests)
```

### Technical Specifications

**`ComputeProgress` method on `model.Run`:**
```go
// ComputeProgress computes the run progress as a percentage (0–100)
// based on the number of completed steps. Returns 0 if there are no steps.
func (r *Run) ComputeProgress(steps []RunStep) int {
    if len(steps) == 0 {
        return 0
    }
    completed := 0
    for _, s := range steps {
        if s.Status == StepStatusCompleted {
            completed++
        }
    }
    return int(float64(completed) / float64(len(steps)) * 100)
}
```

**Updated `model.Run` struct (add field):**
```go
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
    Progress               int  // computed, not persisted
}
```

**`GetRun` service method — updated tail:**
```go
run.Steps = make([]model.RunStep, len(steps))
for i, step := range steps {
    run.Steps[i] = *step
}
run.Progress = run.ComputeProgress(run.Steps)
return run, nil
```

**`ListRunsByProject` service method — updated loop:**
```go
for _, r := range runs {
    steps, err := s.runRepo.ListRunStepsByRun(ctx, r.ID)
    if err != nil {
        return nil, err
    }
    r.Steps = make([]model.RunStep, len(steps))
    for i, s := range steps {
        r.Steps[i] = *s
    }
    r.Progress = r.ComputeProgress(r.Steps)
    // TODO(perf): batch fetch steps in single query for large lists
}
```

**OpenAPI schema addition (in `api/openapi.yaml`, inside Run and RunWithSteps schemas):**
```yaml
progress:
  type: integer
  minimum: 0
  maximum: 100
  description: Percentage of completed steps (0–100), computed at query time.
```
Add `progress` to the `required` array for both schemas.

**`toAPIRun` update:**
```go
func toAPIRun(r *model.Run) Run {
    run := Run{
        // ... existing fields ...
        Progress: r.Progress,
    }
    // ...
    return run
}
```

### Testing Requirements

**`run_test.go` (model unit tests):**
- `ComputeProgress` with 0 steps → 0
- `ComputeProgress` with 3 steps, 0 completed → 0
- `ComputeProgress` with 3 steps, 2 completed → 66
- `ComputeProgress` with 3 steps, 3 completed → 100
- `ComputeProgress` with 1 step, 1 completed → 100

**`run_service_test.go` (service unit tests):**
- Mock `RunRepository.GetRun` returns a run; mock `ListRunStepsByRun` returns 2 steps (1 completed)
- Assert `GetRun` result has `Progress == 50`
- Assert `ListRunsByProject` result runs each have `Progress` populated

**Integration test guards:**
```go
if testing.Short() {
    t.Skip("skipping integration test")
}
```

### References

- `backend/internal/domain/model/run.go` — existing `Run` and `RunStep` structs
- `backend/internal/domain/service/run_service.go` — existing `GetRun`, `ListRunsByProject`
- `backend/internal/api/handler/run_handler.go` — existing `toAPIRun`, `toAPIRunWithSteps`
- `backend/internal/domain/port/run_repository.go` — `ListRunStepsByRun` signature
- `api/openapi.yaml` — `Run` and `RunWithSteps` component schemas to update

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
