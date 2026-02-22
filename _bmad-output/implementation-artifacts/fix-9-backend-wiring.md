# Story fix-9: [BACK] Fix backend wiring — missing server.go delegations, nil pipeline executor dependencies, and missing role in auth response

Status: ready-for-dev

## Story

As a developer running the application end-to-end,
I want the backend to correctly wire all handler delegations, inject real dependencies into the pipeline executor, and return the `role` field in auth responses,
so that no endpoint silently returns 501, the pipeline executor can look up actions and publish events, and the authenticated user's role is available to the frontend.

## Context

An audit of the backend revealed three independent wiring bugs that collectively prevent the app from working correctly end-to-end:

1. **Six endpoints fall through to `Unimplemented` stubs** because `server.go` is missing their delegation methods. One of them (`ResetCircuitBreaker`) already has a fully implemented handler in `project_handler.go`. The other five will be implemented in fix-11 — their delegations must be added now so the router is ready.
2. **`PipelineExecutor` is constructed with `nil` for both `actionReg` and `eventPub`**. The action registry is populated at lines 187–220 of `main.go` but never passed back into the executor. Any run that reaches a step that calls `e.actionReg.Get(...)` or `e.eventPub.Publish(...)` will panic.
3. **`toUserResponse()` in `auth_handler.go` omits the `role` field**. The OpenAPI spec (`api/openapi.yaml`) marks `role` as required in the user schema. The frontend depends on this field to determine admin capabilities.

## Acceptance Criteria (BDD)

**AC1: `ResetCircuitBreaker` returns non-501 for an authenticated admin user**
- **Given** the backend is running
- **When** an admin user sends `POST /api/v1/projects/{id}/circuit-breaker/reset`
- **Then** the response is `204 No Content` (or `403 Forbidden` for non-admin), never `501 Not Implemented`

**AC2: `ListHITLRequests`, `GetHITLRequestByStep`, `GetProjectCostChart`, `GetProjectCostRuns`, `TestNotificationConfig` return non-501**
- **Given** the backend is running
- **When** any of those five endpoints are called
- **Then** the response is NOT `501 Not Implemented` (it may be `404`, `400`, or another code until fix-11 implements the handlers; the delegation wiring must exist)

**AC3: PipelineExecutor receives the populated actionRegistry and eventPublisher**
- **Given** a run is launched
- **When** the River worker calls `pipelineExecutor.ExecuteRun(ctx, runID)` and reaches a step
- **Then** the executor looks up the action from the real registry — no nil pointer panic on `e.actionReg.Get(step.Action)`

**AC4: PipelineExecutor publishes events correctly**
- **Given** a run transitions to `running` or `completed`
- **When** `publishEvent` is called inside the executor
- **Then** the event is forwarded to the real `eventPub` (Postgres NOTIFY) — no nil pointer panic on `e.eventPub.Publish(ctx, event)`

**AC5: `GET /api/v1/auth/me`, `POST /api/v1/auth/login`, and `POST /api/v1/auth/register` include `role` in the response**
- **Given** an authenticated user exists with role `admin` or `user`
- **When** any of those three endpoints is called
- **Then** the JSON response body contains `"role": "admin"` or `"role": "user"` as required by the OpenAPI spec

**AC6: Backend compiles and lints clean**
- **Given** all three fixes are applied
- **When** `cd backend && golangci-lint run ./...` is run
- **Then** no lint errors are reported

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add missing delegations in `server.go` (AC: #1, #2)
  - [ ] Add `ResetCircuitBreaker` delegation pointing to `s.projects.ResetCircuitBreaker` (handler already exists)
  - [ ] Add `ListHITLRequests` delegation pointing to `s.hitl.ListHITLRequests` (stub to be implemented in fix-11)
  - [ ] Add `GetHITLRequestByStep` delegation pointing to `s.hitl.GetHITLRequestByStep` (stub to be implemented in fix-11)
  - [ ] Add `GetProjectCostChart` delegation pointing to `s.costs.GetProjectCostChart` (stub to be implemented in fix-11)
  - [ ] Add `GetProjectCostRuns` delegation pointing to `s.costs.GetProjectCostRuns` (stub to be implemented in fix-11)
  - [ ] Add `TestNotificationConfig` delegation pointing to `s.notifications.TestNotificationConfig` (stub to be implemented in fix-11)
  - [ ] Add stub methods on each respective handler struct so the code compiles (return `501` with a `{"error":{"code":"NOT_IMPLEMENTED","message":"not implemented"}}` body using `writeErrorResponse`)

- [ ] [BACK] Task 2: Fix `PipelineExecutor` wiring in `main.go` (AC: #3, #4)
  - [ ] Remove the early `service.NewPipelineExecutor(runRepo, nil, nil, logger)` call at line 153
  - [ ] Build `eventPublisher` from `eventRepo` before constructing `pipelineExecutor` (the `pgadapter.EventRepo` already implements `port.EventPublisher` — cast or use it directly)
  - [ ] Move `pipelineExecutor` construction to after `actionReg` is fully populated (after line 220) so the real `actionReg` and `eventPub` can be passed
  - [ ] Pass the real `actionReg` and `eventPub` to `service.NewPipelineExecutor(runRepo, actionReg, eventPublisher, logger)`
  - [ ] Verify that `pipelineExecutor.SetCircuitBreaker(circuitBreakerService)` is still called after the new construction site
  - [ ] Ensure `river.AddWorker(workers, riveradapter.NewExecuteRunWorker(pipelineExecutor))` still uses the same (now properly wired) instance

- [ ] [BACK] Task 3: Add `role` field to `userResponse` in `auth_handler.go` (AC: #5)
  - [ ] Add `Role string \`json:"role"\`` to the `userResponse` struct (lines 42–48)
  - [ ] Populate `Role: string(u.Role)` in `toUserResponse()` (line 238–246)

- [ ] [BACK] Task 4: Lint and compile check (AC: #6)
  - [ ] Run `cd backend && go build ./...` — must succeed with zero errors
  - [ ] Run `cd backend && golangci-lint run ./...` — must produce zero lint errors

## Dev Notes

### File Paths

| File | Change |
|------|--------|
| `backend/internal/api/handler/server.go` | Add 6 missing delegation methods |
| `backend/internal/api/handler/hitl_handler.go` | Add `ListHITLRequests` and `GetHITLRequestByStep` stub methods |
| `backend/internal/api/handler/cost_handler.go` | Add `GetProjectCostChart` and `GetProjectCostRuns` stub methods |
| `backend/internal/api/handler/notification_handler.go` | Add `TestNotificationConfig` stub method |
| `backend/cmd/api/main.go` | Restructure `PipelineExecutor` construction to receive real dependencies |
| `backend/internal/api/handler/auth_handler.go` | Add `Role` to `userResponse` and `toUserResponse()` |

### Task 1 — Missing `server.go` delegations

The generated interface in `gen_server.go` (lines 1289–1379) declares these six methods. `server.go` must override the `Unimplemented` stubs by adding explicit delegation methods.

**Signatures from `gen_server.go`:**

```go
// line 1289
ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams)
// line 1292
GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepId StepIdPath)
// line 1319
ResetCircuitBreaker(w http.ResponseWriter, r *http.Request, id IdPath)
// line 1325
GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, params GetProjectCostChartParams)
// line 1328
GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, params GetProjectCostRunsParams)
// line 1379
TestNotificationConfig(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, notificationId NotificationIdPath)
```

**Delegations to add in `server.go`:**

```go
// ResetCircuitBreaker delegates to ProjectHandler — handler fully implemented.
func (s *Server) ResetCircuitBreaker(w http.ResponseWriter, r *http.Request, id IdPath) {
    s.projects.ResetCircuitBreaker(w, r, id)
}

// ListHITLRequests delegates to HITLHandler — implementation deferred to fix-11.
func (s *Server) ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams) {
    s.hitl.ListHITLRequests(w, r, params)
}

// GetHITLRequestByStep delegates to HITLHandler — implementation deferred to fix-11.
func (s *Server) GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepId StepIdPath) {
    s.hitl.GetHITLRequestByStep(w, r, stepId)
}

// GetProjectCostChart delegates to CostHandler — implementation deferred to fix-11.
func (s *Server) GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, params GetProjectCostChartParams) {
    s.costs.GetProjectCostChart(w, r, projectId, params)
}

// GetProjectCostRuns delegates to CostHandler — implementation deferred to fix-11.
func (s *Server) GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, params GetProjectCostRunsParams) {
    s.costs.GetProjectCostRuns(w, r, projectId, params)
}

// TestNotificationConfig delegates to NotificationHandler — implementation deferred to fix-11.
func (s *Server) TestNotificationConfig(w http.ResponseWriter, r *http.Request, projectId ProjectIdPath, notificationId NotificationIdPath) {
    s.notifications.TestNotificationConfig(w, r, projectId, notificationId)
}
```

Each of the five handler structs (`HITLHandler`, `CostHandler`, `NotificationHandler`) needs the corresponding stub method so the code compiles. Pattern to follow:

```go
// ListHITLRequests — to be implemented in fix-11.
func (h *HITLHandler) ListHITLRequests(w http.ResponseWriter, _ *http.Request, _ ListHITLRequestsParams) {
    writeErrorResponse(w, errors.NewInternal("not implemented", fmt.Errorf("ListHITLRequests: not implemented")))
}
```

Use `http.StatusNotImplemented` directly via `writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")` if `writeErrorResponse` does not map to 501. Match the pattern used in the rest of the handler file.

### Task 2 — `PipelineExecutor` wiring in `main.go`

**Current (broken) code — lines 148–158:**

```go
// Run service and job queue
runRepo := pgadapter.NewRunRepo(queries)

// Pipeline executor (will be used by River workers)
// NOTE: event publisher and action registry wiring deferred to later story
pipelineExecutor := service.NewPipelineExecutor(runRepo, nil, nil, logger)
pipelineExecutor.SetCircuitBreaker(circuitBreakerService)

// River job queue for async pipeline execution
workers := river.NewWorkers()
river.AddWorker(workers, riveradapter.NewExecuteRunWorker(pipelineExecutor))
```

**The `actionReg` is populated starting at line 187:**

```go
actionReg := service.NewActionRegistry()
// ... agent_run and incremental_retry registered at lines 206-219 (conditionally on containerMgr != nil)
```

**The `eventRepo` (which implements `port.EventPublisher`) already exists at line 104:**

```go
eventRepo := pgadapter.NewEventRepo(queries)
```

`pgadapter.EventRepo` implements `port.EventPublisher` — confirm this in `backend/internal/adapter/postgres/event_repo.go`. If it does, pass `eventRepo` directly as the `eventPub` argument.

**Target structure — move executor construction to after line 221 (after `actionReg` is fully populated):**

```go
// --- after actionReg.Register(incrementalRetryAction) at line 219 ---

// Pipeline executor: now wired with the real action registry and event publisher.
pipelineExecutor := service.NewPipelineExecutor(runRepo, actionReg, eventRepo, logger)
pipelineExecutor.SetCircuitBreaker(circuitBreakerService)

// River job queue for async pipeline execution
workers := river.NewWorkers()
river.AddWorker(workers, riveradapter.NewExecuteRunWorker(pipelineExecutor))

jobQueue, err := riveradapter.NewJobQueue(pool, workers)
// ...
```

Note: `runService` and `parallelGroupExecutor` are constructed after the current `pipelineExecutor` position. Their construction order does not depend on `pipelineExecutor`'s position, but verify the overall dependency chain before moving code:

- `pipelineExecutor` depends on: `runRepo` (line 149), `actionReg` (line 187), `eventRepo` (line 104)
- `parallelGroupExecutor` (line 266) depends on `pipelineExecutor` — keep `pipelineExecutor` before it
- `jobQueue` depends on `workers` which depends on `pipelineExecutor` — keep them together

The safest approach: move the 5-line executor block (construct + SetCircuitBreaker + workers + AddWorker + jobQueue creation) to after line 221. Remove the original early construction block entirely.

### Task 3 — `role` field in `auth_handler.go`

**Current `userResponse` struct (lines 42–48):**

```go
type userResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

**Target:**

```go
type userResponse struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Name      string `json:"name"`
    Role      string `json:"role"`
    CreatedAt string `json:"created_at"`
    UpdatedAt string `json:"updated_at"`
}
```

**Current `toUserResponse()` (lines 238–246):**

```go
func toUserResponse(u *model.User) userResponse {
    return userResponse{
        ID:        u.ID.String(),
        Email:     u.Email,
        Name:      u.Name,
        CreatedAt: u.CreatedAt.Format(time.RFC3339),
        UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
    }
}
```

**Target:**

```go
func toUserResponse(u *model.User) userResponse {
    return userResponse{
        ID:        u.ID.String(),
        Email:     u.Email,
        Name:      u.Name,
        Role:      string(u.Role),
        CreatedAt: u.CreatedAt.Format(time.RFC3339),
        UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
    }
}
```

`model.User.Role` is of type `model.Role` (a `string` alias, values `"admin"` or `"user"`). Casting to `string` is sufficient.

### Architecture Notes

- **Why not use `SetActionRegistry` / `SetEventPublisher` setters?** The `PipelineExecutor` struct has no setters for these fields — they are set at construction time only (unlike `circuitBreaker` which has `SetCircuitBreaker`). Restructuring the construction order is the correct fix; adding setters would be a larger and unnecessary change.
- **`eventRepo` as `port.EventPublisher`:** The `EventRepo` adapter in `backend/internal/adapter/postgres/event_repo.go` implements `port.EventPublisher`. Passing it directly to `NewPipelineExecutor` satisfies the interface. No additional wrapping is needed.
- **Stub methods returning 501:** The five fix-11 stubs on the handler structs must compile correctly. Using `writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")` is preferred over `http.Error` to stay consistent with the project's error envelope format. The `writeError` helper is already defined in `auth_handler.go` and available package-wide.

### Testing Requirements

After applying the fix:

```bash
# Compile check
cd backend && go build ./...

# Lint check — must be zero errors
cd backend && golangci-lint run ./...

# Smoke test: verify ResetCircuitBreaker no longer returns 501
# (requires the dev stack running)
curl -s -o /dev/null -w "%{http_code}" -X POST \
  -H "Cookie: token=<valid-admin-token>" \
  http://localhost:8080/api/v1/projects/<project-id>/circuit-breaker/reset
# Expected: 204 or 403 (not 501)

# Smoke test: verify role field in /auth/me response
curl -s -H "Cookie: token=<valid-token>" http://localhost:8080/api/v1/auth/me | jq '.role'
# Expected: "admin" or "user" (not null, not absent)
```

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | story-writer | Initial story created |
