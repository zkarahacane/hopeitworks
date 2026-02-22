# Story fix-11: Missing service methods for 501 endpoints

Status: done

## Story

As a backend developer,
I want to implement the 5 missing service methods, repository methods, sqlc queries, and handler methods that currently return 501,
so that the frontend can call `GET /hitl-requests`, `GET /hitl-requests/by-step/{stepId}`, `GET /projects/{projectId}/costs/chart`, `GET /projects/{projectId}/costs/runs`, and `POST /projects/{projectId}/notifications/{notificationId}/test` successfully.

## Acceptance Criteria (BDD)

**AC1: GET /hitl-requests returns paginated list with optional status filter**
- **Given** the caller sends `GET /hitl-requests` (no filter)
- **When** the handler is invoked
- **Then** all HITL requests across all projects are returned with pagination metadata (200 OK)

- **Given** the caller sends `GET /hitl-requests?status=pending`
- **When** the handler is invoked
- **Then** only HITL requests with status `pending` are returned

- **Given** the caller sends `GET /hitl-requests?status=approved&page=2&per_page=10`
- **When** the handler is invoked
- **Then** the response contains at most 10 items from page 2, with correct `total`, `page`, `per_page` in pagination

**AC2: GET /hitl-requests/by-step/{stepId} returns HITL request for a run step**
- **Given** a run step `stepId` that has a HITL request associated to it
- **When** the caller sends `GET /hitl-requests/by-step/{stepId}`
- **Then** the associated HITLRequest is returned (200 OK)

- **Given** a `stepId` with no HITL request
- **When** the caller sends `GET /hitl-requests/by-step/{stepId}`
- **Then** a 404 Not Found is returned

**AC3: GET /projects/{projectId}/costs/chart returns daily data points**
- **Given** a project with cost records over the last 7 days
- **When** the caller sends `GET /projects/{projectId}/costs/chart?period=7d`
- **Then** an array of `CostDataPoint` objects is returned, one per day, with `date` (YYYY-MM-DD) and `total_cost_usd`

- **Given** `period=30d`
- **Then** up to 30 days of data points are returned

**AC4: GET /projects/{projectId}/costs/runs returns paginated run-level cost rows**
- **Given** a project with runs that have cost records
- **When** the caller sends `GET /projects/{projectId}/costs/runs?period=7d`
- **Then** a paginated list of `RunCostRow` items is returned with `run_id`, `story_key`, `status`, `started_at`, `total_cost_usd`

- **Given** `page=1&per_page=5`
- **Then** pagination metadata is correct and at most 5 rows are returned

**AC5: POST /projects/{projectId}/notifications/{notificationId}/test sends a test notification**
- **Given** a notification config of channel type `discord` with a valid `url` in config
- **When** the caller sends `POST /projects/{projectId}/notifications/{notificationId}/test`
- **Then** a test event is dispatched to the Discord webhook and the handler responds 204 No Content

- **Given** a notification config of channel type `webhook`
- **Then** a test event is dispatched to the generic webhook URL and 204 is returned

- **Given** a `notificationId` that does not exist
- **Then** a 404 Not Found is returned

**AC6: All 5 endpoints return 200/204 (not 501)**
- **Given** the application is running with the new service methods wired into Wire DI
- **When** any of the 5 endpoints is called
- **Then** the response status is 200 or 204, never 501

## Tasks / Subtasks

- [ ] Task 1 [BACK]: sqlc query — `ListHITLRequestsFiltered` with optional status, pagination (AC: #1)
  - [ ] Add to `backend/queries/hitl_requests.sql`:
    ```sql
    -- name: ListHITLRequestsFiltered :many
    -- name: CountHITLRequestsFiltered :one
    ```
  - [ ] Run `cd backend && sqlc generate` to regenerate `internal/adapter/postgres/db/`

- [ ] Task 2 [BACK]: sqlc query — `ListDailyCostsByProject` for chart data (AC: #3)
  - [ ] Add to `backend/queries/cost_records.sql`:
    ```sql
    -- name: ListDailyCostsByProject :many
    ```
  - [ ] Run `cd backend && sqlc generate`

- [ ] Task 3 [BACK]: sqlc query — `ListCostsByProjectByRunPaginated` for runs tab (AC: #4)
  - [ ] Add to `backend/queries/cost_records.sql`:
    ```sql
    -- name: ListCostsByProjectByRunPaginated :many
    -- name: CountCostsByProjectByRun :one
    ```
  - [ ] Run `cd backend && sqlc generate`

- [ ] Task 4 [BACK]: Extend `port.HITLRepository` with `ListFiltered` (AC: #1)
  - [ ] Add to `backend/internal/domain/port/hitl_repository.go`:
    ```go
    ListFiltered(ctx context.Context, status *string, limit, offset int32) ([]*model.HITLRequest, error)
    CountFiltered(ctx context.Context, status *string) (int64, error)
    ```

- [ ] Task 5 [BACK]: Extend `port.CostRepository` with chart + paginated runs (AC: #3, #4)
  - [ ] Add to `backend/internal/domain/port/cost_repository.go`:
    ```go
    ListDailyCostsByProject(ctx context.Context, projectID uuid.UUID, since time.Time) ([]model.CostDataPoint, error)
    ListCostsByProjectByRunPaginated(ctx context.Context, projectID uuid.UUID, since time.Time, limit, offset int32) ([]model.RunCostRow, error)
    CountCostsByProjectByRun(ctx context.Context, projectID uuid.UUID, since time.Time) (int64, error)
    ```

- [ ] Task 6 [BACK]: Add domain model types `CostDataPoint` and `RunCostRow` (AC: #3, #4)
  - [ ] Add to `backend/internal/domain/model/cost_record.go`:
    ```go
    type CostDataPoint struct {
        Date        string  // "YYYY-MM-DD"
        TotalCostUSD float64
    }
    type RunCostRow struct {
        RunID        uuid.UUID
        StoryKey     string
        Status       string
        StartedAt    time.Time
        TotalCostUSD float64
    }
    ```

- [ ] Task 7 [BACK]: Implement `HITLRepo.ListFiltered` and `HITLRepo.CountFiltered` in postgres adapter (AC: #1)
  - [ ] Add methods to `backend/internal/adapter/postgres/hitl_repo.go` calling new sqlc queries

- [ ] Task 8 [BACK]: Implement `CostRepo.ListDailyCostsByProject`, `CostRepo.ListCostsByProjectByRunPaginated`, `CostRepo.CountCostsByProjectByRun` (AC: #3, #4)
  - [ ] Add methods to `backend/internal/adapter/postgres/cost_repo.go`

- [ ] Task 9 [BACK]: `HITLService.ListAll` — new service method (AC: #1)
  - [ ] Add to `backend/internal/domain/service/hitl_service.go`:
    ```go
    func (s *HITLService) ListAll(ctx context.Context, status *string, page, perPage int) ([]*model.HITLRequest, int64, error)
    ```

- [ ] Task 10 [BACK]: `HITLService.GetByStepID` — new service method (AC: #2)
  - [ ] Add to `backend/internal/domain/service/hitl_service.go`:
    ```go
    func (s *HITLService) GetByStepID(ctx context.Context, stepID uuid.UUID) (*model.HITLRequest, error)
    ```
  - [ ] Delegates to `s.hitlRepo.GetByRunStepID` (already exists in port and adapter)

- [ ] Task 11 [BACK]: `CostService.GetProjectCostChart` — new service method (AC: #3)
  - [ ] Add to `backend/internal/domain/service/cost_service.go`:
    ```go
    func (s *CostService) GetProjectCostChart(ctx context.Context, projectID uuid.UUID, period string) ([]model.CostDataPoint, error)
    ```

- [ ] Task 12 [BACK]: `CostService.GetProjectCostRuns` — new service method (AC: #4)
  - [ ] Add to `backend/internal/domain/service/cost_service.go`:
    ```go
    func (s *CostService) GetProjectCostRuns(ctx context.Context, projectID uuid.UUID, period string, page, perPage int) ([]model.RunCostRow, int64, error)
    ```

- [ ] Task 13 [BACK]: `NotificationConfigService.Test` — new service method (AC: #5)
  - [ ] Add to `backend/internal/domain/service/notification_config_service.go`:
    ```go
    func (s *NotificationConfigService) Test(ctx context.Context, projectID, notifID uuid.UUID, discordNotifier port.Notifier, webhookNotifier port.Notifier) error
    ```
  - [ ] Fetch config by ID, verify it belongs to `projectID`, construct a synthetic `model.Event`, dispatch to the correct notifier based on `ChannelType`

- [ ] Task 14 [BACK]: `HITLHandler.ListHITLRequests` — implement handler (AC: #1)
  - [ ] Add to `backend/internal/api/handler/hitl_handler.go`:
    ```go
    func (h *HITLHandler) ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams)
    ```

- [ ] Task 15 [BACK]: `HITLHandler.GetHITLRequestByStep` — implement handler (AC: #2)
  - [ ] Add to `backend/internal/api/handler/hitl_handler.go`:
    ```go
    func (h *HITLHandler) GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepID StepIdPath)
    ```

- [ ] Task 16 [BACK]: `CostHandler.GetProjectCostChart` — implement handler (AC: #3)
  - [ ] Add to `backend/internal/api/handler/cost_handler.go`:
    ```go
    func (h *CostHandler) GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostChartParams)
    ```

- [ ] Task 17 [BACK]: `CostHandler.GetProjectCostRuns` — implement handler (AC: #4)
  - [ ] Add to `backend/internal/api/handler/cost_handler.go`:
    ```go
    func (h *CostHandler) GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostRunsParams)
    ```

- [ ] Task 18 [BACK]: `NotificationHandler.TestNotificationConfig` — implement handler (AC: #5)
  - [ ] Update `NotificationHandler` struct to hold both `discordNotifier` and `webhookNotifier`
  - [ ] Update `NewNotificationHandler` constructor
  - [ ] Add to `backend/internal/api/handler/notification_handler.go`:
    ```go
    func (h *NotificationHandler) TestNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, notificationID NotificationIdPath)
    ```

- [ ] Task 19 [BACK]: Update Wire DI providers (AC: #6)
  - [ ] Update `backend/cmd/api/wire.go` / `providers.go` to inject `discordNotifier` and `webhookNotifier` into `NotificationHandler`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] Task 20 [BACK]: Unit tests (AC: #1–#5)
  - [ ] `HITLService.ListAll` — test with nil status, status="pending", pagination math
  - [ ] `HITLService.GetByStepID` — test found + not found
  - [ ] `CostService.GetProjectCostChart` — test 7d and 30d, verify date string format
  - [ ] `CostService.GetProjectCostRuns` — test pagination
  - [ ] `NotificationConfigService.Test` — test discord dispatch, webhook dispatch, wrong project 404

- [ ] Task 21 [BACK]: Lint check
  - [ ] `cd backend && golangci-lint run ./...` — must pass before committing

## Dev Notes

### Context

5 endpoints are registered in the generated `ServerInterface` (from `api/openapi.yaml` via oapi-codegen) and have `Unimplemented` stubs returning HTTP 501. The handler files exist but the methods are absent — they must be added to satisfy the interface. The oapi-codegen `Unimplemented` struct provides the fallback, so the concrete handler structs (HITLHandler, CostHandler, NotificationHandler) each need their specific method added.

### Existing infrastructure that can be reused

- `port.HITLRepository.GetByRunStepID` already exists in the port and is implemented in `postgres/hitl_repo.go` — `GetByStepID` in the service is a thin delegation
- `ListCostsByProjectByRun` already exists in the port and adapter for 7d/30d breakdowns — the paginated variant is a new query on top of the same data
- `discord.Notifier` and `webhook.Notifier` both implement `port.Notifier` — the test endpoint only needs to pick the right one based on `ChannelType`

### Exact Go signatures to implement

**hitl_service.go — new methods:**
```go
// ListAll returns a paginated list of HITL requests, optionally filtered by status.
func (s *HITLService) ListAll(ctx context.Context, status *string, page, perPage int) ([]*model.HITLRequest, int64, error) {
    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }
    offset := int32((page - 1) * perPage)
    limit := int32(perPage)
    items, err := s.hitlRepo.ListFiltered(ctx, status, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    total, err := s.hitlRepo.CountFiltered(ctx, status)
    if err != nil {
        return nil, 0, err
    }
    return items, total, nil
}

// GetByStepID returns the HITL request associated with a run step.
func (s *HITLService) GetByStepID(ctx context.Context, stepID uuid.UUID) (*model.HITLRequest, error) {
    return s.hitlRepo.GetByRunStepID(ctx, stepID)
}
```

**cost_service.go — new methods:**
```go
// GetProjectCostChart returns daily cost data points for chart rendering.
func (s *CostService) GetProjectCostChart(ctx context.Context, projectID uuid.UUID, period string) ([]model.CostDataPoint, error) {
    if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
        return nil, err
    }
    if period == "" {
        period = "7d"
    }
    since, err := parsePeriod(period)
    if err != nil {
        return nil, err
    }
    return s.costRepo.ListDailyCostsByProject(ctx, projectID, since)
}

// GetProjectCostRuns returns a paginated run-level cost breakdown for a project.
func (s *CostService) GetProjectCostRuns(ctx context.Context, projectID uuid.UUID, period string, page, perPage int) ([]model.RunCostRow, int64, error) {
    if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
        return nil, 0, err
    }
    if period == "" {
        period = "7d"
    }
    since, err := parsePeriod(period)
    if err != nil {
        return nil, 0, err
    }
    if page < 1 {
        page = 1
    }
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }
    offset := int32((page - 1) * perPage)
    limit := int32(perPage)
    rows, err := s.costRepo.ListCostsByProjectByRunPaginated(ctx, projectID, since, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    total, err := s.costRepo.CountCostsByProjectByRun(ctx, projectID, since)
    if err != nil {
        return nil, 0, err
    }
    return rows, total, nil
}
```

**notification_config_service.go — new method:**
```go
// Test dispatches a synthetic test notification for the given config.
func (s *NotificationConfigService) Test(ctx context.Context, projectID, notifID uuid.UUID, discordNotifier port.Notifier, webhookNotifier port.Notifier) error {
    cfg, err := s.repo.Get(ctx, notifID)
    if err != nil {
        return err
    }
    if cfg.ProjectID != projectID {
        return errors.NewNotFound("notification_config", notifID)
    }
    if !cfg.Enabled {
        return errors.NewValidation("enabled", "notification config is disabled")
    }
    testEvent := model.Event{
        ID:         uuid.New(),
        ProjectID:  projectID,
        EntityType: "notification",
        EntityID:   notifID,
        Action:     "test",
        Payload:    []byte(`{"message":"test notification"}`),
        CreatedAt:  time.Now(),
    }
    switch cfg.ChannelType {
    case model.ChannelTypeDiscord:
        return discordNotifier.Send(ctx, testEvent, cfg.Config)
    case model.ChannelTypeWebhook:
        return webhookNotifier.Send(ctx, testEvent, cfg.Config)
    default:
        return errors.NewValidation("channel_type", fmt.Sprintf("unsupported channel type: %s", cfg.ChannelType))
    }
}
```

### sqlc queries to add

**backend/queries/hitl_requests.sql — append:**
```sql
-- name: ListHITLRequestsFiltered :many
SELECT * FROM hitl_requests
WHERE ($1::text IS NULL OR status = $1::text)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountHITLRequestsFiltered :one
SELECT COUNT(*) FROM hitl_requests
WHERE ($1::text IS NULL OR status = $1::text);
```

**backend/queries/cost_records.sql — append:**
```sql
-- name: ListDailyCostsByProject :many
SELECT
    DATE(cr.created_at)::text AS date,
    COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost_usd
FROM cost_records cr
WHERE cr.project_id = $1
  AND cr.created_at >= $2
GROUP BY DATE(cr.created_at)
ORDER BY date ASC;

-- name: ListCostsByProjectByRunPaginated :many
SELECT rs2.run_id,
       s.key    AS story_key,
       r.status,
       r.created_at AS started_at,
       COALESCE(SUM(cr.cost_usd), 0)::DECIMAL(10,6) AS total_cost_usd
FROM cost_records cr
JOIN run_steps rs2 ON rs2.id = cr.run_step_id
JOIN runs r ON r.id = rs2.run_id
JOIN stories s ON s.id = r.story_id
WHERE cr.project_id = $1 AND cr.created_at >= $2
GROUP BY rs2.run_id, s.key, r.status, r.created_at
ORDER BY r.created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCostsByProjectByRun :one
SELECT COUNT(DISTINCT rs2.run_id)
FROM cost_records cr
JOIN run_steps rs2 ON rs2.id = cr.run_step_id
WHERE cr.project_id = $1 AND cr.created_at >= $2;
```

### Handler patterns to follow

**hitl_handler.go — ListHITLRequests:**
```go
func (h *HITLHandler) ListHITLRequests(w http.ResponseWriter, r *http.Request, params ListHITLRequestsParams) {
    var status *string
    if params.Status != nil {
        s := string(*params.Status)
        status = &s
    }
    page, perPage := paginationParams(params.Page, params.PerPage)
    items, total, err := h.service.ListAll(r.Context(), status, page, perPage)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    data := make([]HITLRequest, len(items))
    for i, req := range items {
        data[i] = toAPIHITLRequest(req)
    }
    writeJSON(w, http.StatusOK, HITLRequestList{
        Data: data,
        Pagination: Pagination{
            Total:   int(total),
            Page:    page,
            PerPage: perPage,
        },
    })
}
```

**hitl_handler.go — GetHITLRequestByStep:**
```go
func (h *HITLHandler) GetHITLRequestByStep(w http.ResponseWriter, r *http.Request, stepID StepIdPath) {
    req, err := h.service.GetByStepID(r.Context(), stepID)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    writeJSON(w, http.StatusOK, toAPIHITLRequest(req))
}
```

**cost_handler.go — GetProjectCostChart:**
```go
func (h *CostHandler) GetProjectCostChart(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostChartParams) {
    period := "7d"
    if params.Period != nil {
        period = string(*params.Period)
    }
    points, err := h.service.GetProjectCostChart(r.Context(), projectID, period)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    resp := make([]CostDataPoint, len(points))
    for i, p := range points {
        resp[i] = CostDataPoint{
            Date:         p.Date,
            TotalCostUsd: p.TotalCostUSD,
        }
    }
    writeJSON(w, http.StatusOK, resp)
}
```

**cost_handler.go — GetProjectCostRuns:**
```go
func (h *CostHandler) GetProjectCostRuns(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params GetProjectCostRunsParams) {
    period := "7d"
    if params.Period != nil {
        period = string(*params.Period)
    }
    page, perPage := paginationParams(params.Page, params.PerPage)
    rows, total, err := h.service.GetProjectCostRuns(r.Context(), projectID, period, page, perPage)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }
    data := make([]RunCostRow, len(rows))
    for i, row := range rows {
        data[i] = RunCostRow{
            RunId:        row.RunID,
            StoryKey:     row.StoryKey,
            Status:       row.Status,
            StartedAt:    row.StartedAt,
            TotalCostUsd: row.TotalCostUSD,
        }
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "data": data,
        "pagination": Pagination{
            Total:   int(total),
            Page:    page,
            PerPage: perPage,
        },
    })
}
```

**notification_handler.go — TestNotificationConfig:**
```go
// NotificationHandler must be updated to carry both notifiers:
type NotificationHandler struct {
    service         *service.NotificationConfigService
    discordNotifier port.Notifier
    webhookNotifier port.Notifier
}

func NewNotificationHandler(svc *service.NotificationConfigService, discord port.Notifier, webhook port.Notifier) *NotificationHandler {
    return &NotificationHandler{service: svc, discordNotifier: discord, webhookNotifier: webhook}
}

func (h *NotificationHandler) TestNotificationConfig(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, notificationID NotificationIdPath) {
    if err := h.service.Test(r.Context(), projectID, uuid.UUID(notificationID), h.discordNotifier, h.webhookNotifier); err != nil {
        writeErrorResponse(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

### Pagination helper

Check if a `paginationParams` helper already exists in the handler package (look for it in `backend/internal/api/handler/`). If absent, add a package-level helper:
```go
// paginationParams extracts page and perPage from optional query params with defaults.
func paginationParams(page, perPage *int) (int, int) {
    p, pp := 1, 20
    if page != nil && *page > 0 {
        p = *page
    }
    if perPage != nil && *perPage > 0 && *perPage <= 100 {
        pp = *perPage
    }
    return p, pp
}
```

### Wire DI changes

`NotificationHandler` gains two new dependencies (`discord.Notifier`, `webhook.Notifier`). Both already exist as singletons in the app. Update the Wire provider set (in `cmd/api/wire.go` or `providers.go`) to pass them to `NewNotificationHandler`. Regenerate with `wire ./cmd/api/`.

### parsePeriod note for GetProjectCostChart

`parsePeriod` in `cost_service.go` accepts `7d`, `30d`, `90d`. The chart endpoint only exposes `7d` and `30d` (OpenAPI enum), so `parsePeriod` already covers both valid values.

### File list to modify

- `backend/queries/hitl_requests.sql` — add 2 queries
- `backend/queries/cost_records.sql` — add 3 queries
- `backend/internal/domain/port/hitl_repository.go` — add 2 methods to interface
- `backend/internal/domain/port/cost_repository.go` — add 3 methods to interface
- `backend/internal/domain/model/cost_record.go` — add `CostDataPoint` and `RunCostRow` types
- `backend/internal/domain/service/hitl_service.go` — add `ListAll`, `GetByStepID`
- `backend/internal/domain/service/cost_service.go` — add `GetProjectCostChart`, `GetProjectCostRuns`
- `backend/internal/domain/service/notification_config_service.go` — add `Test`; needs `port` import
- `backend/internal/adapter/postgres/hitl_repo.go` — implement `ListFiltered`, `CountFiltered`
- `backend/internal/adapter/postgres/cost_repo.go` — implement 3 new repo methods
- `backend/internal/api/handler/hitl_handler.go` — add `ListHITLRequests`, `GetHITLRequestByStep`
- `backend/internal/api/handler/cost_handler.go` — add `GetProjectCostChart`, `GetProjectCostRuns`
- `backend/internal/api/handler/notification_handler.go` — update struct + constructor, add `TestNotificationConfig`
- `backend/cmd/api/wire.go` (or `providers.go`) — update `NotificationHandler` provider
- `backend/cmd/api/wire_gen.go` — regenerated (never edit manually)
- `backend/internal/adapter/postgres/db/hitl_requests.sql.go` — regenerated by sqlc
- `backend/internal/adapter/postgres/db/cost_records.sql.go` — regenerated by sqlc

### Do not touch

- `backend/internal/api/handler/gen_server.go` — fully generated by oapi-codegen, never edit
- `api/openapi.yaml` — no changes needed; all 5 endpoints are already defined
