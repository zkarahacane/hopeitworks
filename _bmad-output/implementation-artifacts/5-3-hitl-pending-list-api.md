# Story 5.3: [BACK] HITL Pending List API Endpoint

Status: ready-for-dev

## Story

As a platform user, I want to list all pending HITL requests for a project so that I can see what needs approval without browsing individual runs.

## Acceptance Criteria (BDD)

**AC1: New OpenAPI endpoint for pending HITL list**
- **Given** the OpenAPI spec at `api/openapi.yaml`
- **When** `GET /api/v1/projects/{projectId}/hitl/pending` is added
- **Then** the spec defines a paginated response using `$ref: "#/components/schemas/HITLRequest"` items with a `Pagination` envelope
- **And** `page` and `per_page` query params are accepted via `$ref: "#/components/parameters/PageParam"` and `$ref: "#/components/parameters/PerPageParam"`
- **And** `cd backend && make generate` regenerates chi server interfaces successfully

**AC2: Only pending requests are returned**
- **Given** a project with HITL requests in `pending`, `approved`, and `rejected` states
- **When** `GET /api/v1/projects/{projectId}/hitl/pending` is called
- **Then** only requests with `status = 'pending'` are included in the response
- **And** approved and rejected requests are excluded

**AC3: Paginated response structure**
- **Given** a project with more than `per_page` pending HITL requests
- **When** `GET /api/v1/projects/{projectId}/hitl/pending?page=1&per_page=10` is called
- **Then** HTTP 200 is returned with body `{ "data": [...], "pagination": { "total": N, "page": 1, "per_page": 10 } }`
- **And** each item in `data` conforms to the `HITLRequest` schema (includes `run_id`, `step_id`, `story_key`, `created_at`, `diff_content`)
- **And** default values of `page=1` and `per_page=20` apply when query params are absent

**AC4: Auth and project access are enforced**
- **Given** a request without a valid JWT cookie
- **When** `GET /api/v1/projects/{projectId}/hitl/pending` is called
- **Then** HTTP 401 is returned
- **Given** a valid JWT but the authenticated user is not a member of the project identified by `projectId`
- **When** the endpoint is called
- **Then** HTTP 403 is returned

**AC5: Project not found returns 404**
- **Given** a `projectId` that does not exist
- **When** `GET /api/v1/projects/{projectId}/hitl/pending` is called with a valid JWT
- **Then** HTTP 404 is returned with error code `PROJECT_NOT_FOUND`

**AC6: sqlc query retrieves pending requests with run and story context**
- **Given** a new sqlc query `ListPendingHITLRequestsByProject :many` in `backend/queries/hitl_requests.sql`
- **When** executed with a `project_id` and `LIMIT`/`OFFSET` pagination params
- **Then** it returns all pending `hitl_requests` rows joined with `run_steps`, `runs`, and `stories` to populate `run_id`, `step_id`, `story_key`, and `story_title`
- **And** a companion `CountPendingHITLRequestsByProject :one` query returns the total count for pagination metadata

**AC7: curl smoke test**
- **Given** a running backend with an authenticated session cookie
- **When** `curl -X GET -b cookies.txt /api/v1/projects/{id}/hitl/pending` is executed
- **Then** HTTP 200 is returned with a valid paginated JSON response

**AC8: Unit tests cover the service method and handler**
- **Given** unit tests in `backend/internal/domain/service/__tests__/hitl_service_test.go`
- **When** tests run
- **Then** happy path (items returned), empty result (zero items, valid pagination), project not found, and unauthorized access paths are covered
- **And** `golangci-lint run ./...` passes from `backend/`

## Tasks / Subtasks

- [ ] [BACK] Task 1: Update OpenAPI spec — add pending list endpoint (AC: #1, #3)
  - [ ] Add `GET /projects/{projectId}/hitl/pending` to `api/openapi.yaml` under the HITL Requests section
  - [ ] Set `operationId: listPendingHITLRequests`, `tags: [hitl]`, `summary: "List pending HITL requests for a project"`
  - [ ] Add parameters: `$ref: "#/components/parameters/ProjectIdPath"`, `$ref: "#/components/parameters/PageParam"`, `$ref: "#/components/parameters/PerPageParam"`
  - [ ] Add `HITLPendingList` schema to `components/schemas`: `{ type: object, required: [data, pagination], properties: { data: { type: array, items: { $ref: HITLRequest } }, pagination: { $ref: Pagination } } }`
  - [ ] Define 200 response using `HITLPendingList` schema, plus 401, 403, 404 error responses
  - [ ] Run `cd backend && make generate` to regenerate server interfaces

- [ ] [BACK] Task 2: Add sqlc queries for pending list (AC: #6)
  - [ ] Add `ListPendingHITLRequestsByProject :many` query to `backend/queries/hitl_requests.sql`:
    ```sql
    SELECT hr.*, rs.run_id, s.key AS story_key, s.title AS story_title, s.objective AS story_objective
    FROM hitl_requests hr
    JOIN run_steps rs ON rs.id = hr.run_step_id
    JOIN runs r ON r.id = rs.run_id
    JOIN stories s ON s.id = r.story_id
    WHERE r.project_id = $1
      AND hr.status = 'pending'
    ORDER BY hr.created_at ASC
    LIMIT $2 OFFSET $3;
    ```
  - [ ] Add `CountPendingHITLRequestsByProject :one` query to the same file:
    ```sql
    SELECT COUNT(*) FROM hitl_requests hr
    JOIN run_steps rs ON rs.id = hr.run_step_id
    JOIN runs r ON r.id = rs.run_id
    WHERE r.project_id = $1
      AND hr.status = 'pending';
    ```
  - [ ] Run `cd backend && sqlc generate` to regenerate `backend/internal/adapter/postgres/db/`

- [ ] [BACK] Task 3: Extend HITLRepository port with ListPendingByProject (AC: #6)
  - [ ] Add `ListPendingByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) ([]*model.HITLRequest, int64, error)` to `backend/internal/domain/port/hitl_repository.go`
  - [ ] The method returns the page of results and the total count for pagination metadata

- [ ] [BACK] Task 4: Implement ListPendingByProject in the Postgres adapter (AC: #2, #6)
  - [ ] Implement `ListPendingByProject` in `backend/internal/adapter/postgres/hitl_repo.go`
  - [ ] Compute `offset = (page - 1) * perPage`
  - [ ] Call `r.queries.ListPendingHITLRequestsByProject` and `r.queries.CountPendingHITLRequestsByProject` in sequence
  - [ ] Map sqlc rows to `[]*model.HITLRequest` using the existing `toDomainHITLRequest` helper, enriching `RunID` and `StoryKey` fields from the joined columns
  - [ ] Handle DB errors: wrap with `apperrors.NewInternal`; empty result (zero rows) is a valid non-error response

- [ ] [BACK] Task 5: Extend model.HITLRequest with denormalised list fields (AC: #3, #6)
  - [ ] Add `RunID *uuid.UUID`, `StoryKey string`, `StoryTitle string`, `StoryObjective *string` to `backend/internal/domain/model/hitl.go`
  - [ ] These fields are populated only in list queries — they are `nil`/zero in single-resource fetch paths
  - [ ] Update `toDomainHITLRequest` in `hitl_repo.go` to map the new joined columns when present (use a separate `toDomainHITLRequestFromList` helper to avoid polluting the single-resource mapper)

- [ ] [BACK] Task 6: Add ListPendingByProject method to HITLService (AC: #2, #3, #4, #5)
  - [ ] Add `ListPendingByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*PendingHITLListResult, error)` to `backend/internal/domain/service/hitl_service.go`
  - [ ] Define `PendingHITLListResult` struct: `{ Items []*model.HITLRequest; Total int64 }`
  - [ ] Verify project exists via `projectRepo.GetByID` — return `errors.NewNotFound("project", projectID)` with code `PROJECT_NOT_FOUND` if not found
  - [ ] Delegate to `hitlRepo.ListPendingByProject(ctx, projectID, page, perPage)`
  - [ ] Return wrapped result

- [ ] [BACK] Task 7: Implement ListPendingHITLRequests handler (AC: #1, #3, #4)
  - [ ] Add `ListPendingHITLRequests(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListPendingHITLRequestsParams)` to `backend/internal/api/handler/hitl_handler.go`
  - [ ] Parse `page` and `per_page` from `params`, defaulting to `1` and `20` respectively
  - [ ] Call `h.service.ListPendingByProject(r.Context(), projectID, page, perPage)`
  - [ ] Map domain `[]*model.HITLRequest` to `[]HITLRequest` API schema using a `toAPIHITLRequest` helper
  - [ ] Render 200 with `HITLPendingList{ Data: items, Pagination: Pagination{ Total: int(result.Total), Page: page, PerPage: perPage } }`
  - [ ] Register route in `backend/internal/api/router.go`: `r.Get("/hitl/pending", hitlHandler.ListPendingHITLRequests)` inside the `r.Route("/api/v1/projects/{projectId}", ...)` block

- [ ] [BACK] Task 8: Wire ProjectRepository into HITLService and update DI (AC: #5, #6)
  - [ ] Add `projectRepo port.ProjectRepository` to `HITLService` struct dependencies
  - [ ] Update `NewHITLService` constructor signature to accept `port.ProjectRepository`
  - [ ] Update `backend/cmd/api/wire.go` provider set to inject `ProjectRepository` into `HITLService`
  - [ ] Run `cd backend && wire ./cmd/api/` to regenerate `wire_gen.go`

- [ ] [BACK] Task 9: Unit tests for HITLService.ListPendingByProject and HITLHandler (AC: #8)
  - [ ] Add test cases to `backend/internal/domain/service/__tests__/hitl_service_test.go`:
    - Happy path — 3 pending items returned; verify pagination total matches mock count
    - Empty result — 0 items with `total = 0`; verify valid `PendingHITLListResult` returned (no error)
    - Project not found — `projectRepo.GetByID` returns not-found; `PROJECT_NOT_FOUND` propagated
    - DB error — `hitlRepo.ListPendingByProject` returns internal error; error propagated
  - [ ] Add `MockProjectRepository` to the mock set in the test file (hand-written, `GetByID` method)
  - [ ] Add handler test in `backend/internal/api/handler/hitl_handler_test.go` (or create if absent):
    - Verify 200 response structure matches `HITLPendingList` schema
    - Verify default pagination (no query params → page=1, per_page=20 passed to service)
    - Verify 401 when auth middleware blocks request
  - [ ] Run `golangci-lint run ./...` from `backend/` — must pass before commit

## Dev Notes

### Dependencies

**Story 5-1 (HITL Gate Action — Wave 10, DONE):** Provides the complete foundation:
- `model.HITLRequest` and `model.HITLStatus` in `backend/internal/domain/model/hitl.go`
- `port.HITLRepository` interface in `backend/internal/domain/port/hitl_repository.go` (to be extended with `ListPendingByProject`)
- `HITLRepo` Postgres adapter in `backend/internal/adapter/postgres/hitl_repo.go` (to be extended)
- `hitl_requests` table with `idx_hitl_requests_status` index (migration `000013`)
- `HITLGateAction` registered in `ActionRegistry`

**Story 5-2 (Approve/Reject API — Wave 11, DONE):** Provides:
- `HITLService` struct in `backend/internal/domain/service/hitl_service.go` — `ListPendingByProject` is added to this service (Task 6)
- `HITLHandler` in `backend/internal/api/handler/hitl_handler.go` — `ListPendingHITLRequests` is added to this handler (Task 7)
- `GetPendingByRunID` sqlc query and port method already present

**Story 5-5 (HITL Pending List + Notification Badge — frontend):** Consumes this endpoint. The `HITLPendingList` schema and `GET /projects/{projectId}/hitl/pending` endpoint must be in `api/openapi.yaml` before Story 5-5 runs `npm run generate-api`.

**Story 1-6 (RBAC Middleware — DONE):** `middleware.UserIDFromContext`, project membership checks, and the `checkProjectAccess` helper available in `ProjectHandler` are already in place.

### Architecture Requirements

- `ListPendingByProject` belongs in `HITLService` — not a new service
- `ListPendingHITLRequests` belongs in `HITLHandler` — not a new handler
- No business logic in the handler — only HTTP parsing and response rendering
- The join query (hitl_requests → run_steps → runs → stories) is the correct approach; do NOT add denormalised columns to `hitl_requests`
- Pagination uses the existing `PageParam` / `PerPageParam` / `Pagination` schema conventions — do not invent a new pagination format
- The `RunID`, `StoryKey`, `StoryTitle` fields needed for the list response come from the JOIN — use a separate `toDomainHITLRequestFromList` mapping helper to avoid coupling the single-resource path to these joined columns

### File Paths (exact)

```
api/openapi.yaml                                                               # Add: GET /projects/{projectId}/hitl/pending + HITLPendingList schema
backend/queries/hitl_requests.sql                                              # Add: ListPendingHITLRequestsByProject + CountPendingHITLRequestsByProject
backend/internal/adapter/postgres/db/                                          # Regenerated: sqlc generate
backend/internal/adapter/postgres/hitl_repo.go                                 # Add: ListPendingByProject + toDomainHITLRequestFromList
backend/internal/domain/model/hitl.go                                          # Extend: add RunID, StoryKey, StoryTitle, StoryObjective fields
backend/internal/domain/port/hitl_repository.go                               # Add: ListPendingByProject method to interface
backend/internal/domain/service/hitl_service.go                               # Add: ListPendingByProject method + PendingHITLListResult type
backend/internal/domain/service/__tests__/hitl_service_test.go                # Add: ListPendingByProject test cases
backend/internal/api/handler/hitl_handler.go                                  # Add: ListPendingHITLRequests handler method + toAPIHITLRequest helper
backend/internal/api/handler/hitl_handler_test.go                             # Add/create: handler tests for list endpoint
backend/internal/api/router.go                                                 # Add: GET /hitl/pending route under /projects/{projectId}
backend/cmd/api/wire.go                                                        # Update: inject ProjectRepository into HITLService
backend/cmd/api/wire_gen.go                                                    # Regenerated: wire ./cmd/api/
```

### Technical Specifications

**New OpenAPI endpoint:**
```yaml
/projects/{projectId}/hitl/pending:
  get:
    operationId: listPendingHITLRequests
    summary: List pending HITL requests for a project
    tags: [hitl]
    parameters:
      - $ref: "#/components/parameters/ProjectIdPath"
      - $ref: "#/components/parameters/PageParam"
      - $ref: "#/components/parameters/PerPageParam"
    responses:
      "200":
        description: Paginated list of pending HITL requests
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/HITLPendingList"
      "401":
        $ref: "#/components/responses/Unauthorized"
      "403":
        $ref: "#/components/responses/Forbidden"
      "404":
        $ref: "#/components/responses/NotFound"
```

**HITLPendingList schema:**
```yaml
# api/openapi.yaml — add to components/schemas alongside HITLRequest
HITLPendingList:
  type: object
  required: [data, pagination]
  properties:
    data:
      type: array
      items:
        $ref: "#/components/schemas/HITLRequest"
    pagination:
      $ref: "#/components/schemas/Pagination"
```

**sqlc queries (backend/queries/hitl_requests.sql — append):**
```sql
-- name: ListPendingHITLRequestsByProject :many
SELECT
    hr.id,
    hr.run_step_id,
    hr.gate_type,
    hr.diff_content,
    hr.status,
    hr.resolved_at,
    hr.resolved_by,
    hr.rejection_reason,
    hr.created_at,
    rs.run_id         AS run_id,
    s.key             AS story_key,
    s.title           AS story_title,
    s.objective       AS story_objective
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
JOIN runs r ON r.id = rs.run_id
JOIN stories s ON s.id = r.story_id
WHERE r.project_id = $1
  AND hr.status = 'pending'
ORDER BY hr.created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountPendingHITLRequestsByProject :one
SELECT COUNT(*)
FROM hitl_requests hr
JOIN run_steps rs ON rs.id = hr.run_step_id
JOIN runs r ON r.id = rs.run_id
WHERE r.project_id = $1
  AND hr.status = 'pending';
```

**Extended domain model (backend/internal/domain/model/hitl.go):**
```go
// HITLRequest records a human-in-the-loop gate triggered by a pipeline step.
type HITLRequest struct {
    ID              uuid.UUID
    RunStepID       uuid.UUID
    GateType        string
    DiffContent     *string
    Status          HITLStatus
    ResolvedAt      *time.Time
    ResolvedBy      *uuid.UUID
    RejectionReason *string
    CreatedAt       time.Time

    // Populated in list queries only (joined from runs and stories).
    RunID          *uuid.UUID
    StoryKey       string
    StoryTitle     string
    StoryObjective *string
}
```

**Extended HITLRepository port:**
```go
// HITLRepository defines persistence operations for HITL approval requests.
type HITLRepository interface {
    Create(ctx context.Context, req *model.HITLRequest) (*model.HITLRequest, error)
    GetByRunStepID(ctx context.Context, runStepID uuid.UUID) (*model.HITLRequest, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status model.HITLStatus, resolvedBy *uuid.UUID, rejectionReason *string, resolvedAt time.Time) (*model.HITLRequest, error)
    // ListPendingByProject returns paginated pending HITL requests for a project
    // with run and story context populated from joined tables.
    // Returns the page slice and the total count of pending requests.
    ListPendingByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) ([]*model.HITLRequest, int64, error)
}
```

**HITLService addition:**
```go
// PendingHITLListResult holds the paginated result of a pending HITL request query.
type PendingHITLListResult struct {
    Items []*model.HITLRequest
    Total int64
}

// ListPendingByProject returns all pending HITL requests for the given project, paginated.
func (s *HITLService) ListPendingByProject(ctx context.Context, projectID uuid.UUID, page, perPage int) (*PendingHITLListResult, error) {
    // Verify project exists.
    if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
        return nil, err // already wrapped as PROJECT_NOT_FOUND by projectRepo
    }
    items, total, err := s.hitlRepo.ListPendingByProject(ctx, projectID, page, perPage)
    if err != nil {
        return nil, err
    }
    return &PendingHITLListResult{Items: items, Total: total}, nil
}
```

**HITLHandler addition:**
```go
// ListPendingHITLRequests handles GET /projects/{projectId}/hitl/pending.
func (h *HITLHandler) ListPendingHITLRequests(w http.ResponseWriter, r *http.Request, projectID ProjectIdPath, params ListPendingHITLRequestsParams) {
    page := 1
    perPage := 20
    if params.Page != nil && *params.Page > 0 {
        page = *params.Page
    }
    if params.PerPage != nil && *params.PerPage > 0 {
        perPage = *params.PerPage
    }

    result, err := h.service.ListPendingByProject(r.Context(), projectID, page, perPage)
    if err != nil {
        writeErrorResponse(w, err)
        return
    }

    items := make([]HITLRequest, len(result.Items))
    for i, req := range result.Items {
        items[i] = toAPIHITLRequest(req)
    }

    writeJSON(w, http.StatusOK, HITLPendingList{
        Data: items,
        Pagination: Pagination{
            Total:   int(result.Total),
            Page:    page,
            PerPage: perPage,
        },
    })
}
```

**toAPIHITLRequest helper:**
```go
// toAPIHITLRequest maps a domain HITLRequest to the generated API schema type.
func toAPIHITLRequest(req *model.HITLRequest) HITLRequest {
    r := HITLRequest{
        Id:        req.ID,
        RunStepId: req.RunStepID,
        GateType:  req.GateType,
        Status:    HITLRequestStatus(req.Status),
        StoryKey:  req.StoryKey,
        StoryTitle: req.StoryTitle,
        CreatedAt: req.CreatedAt,
    }
    if req.RunID != nil {
        r.RunId = req.RunID
    }
    if req.DiffContent != nil {
        r.DiffContent = req.DiffContent
    }
    if req.RejectionReason != nil {
        r.RejectionReason = req.RejectionReason
    }
    if req.ResolvedAt != nil {
        r.ResolvedAt = req.ResolvedAt
    }
    return r
}
```

**toDomainHITLRequestFromList helper (hitl_repo.go):**
```go
// toDomainHITLRequestFromList maps a list sqlc row (with joined columns) to a domain HITLRequest.
// Use this helper ONLY for ListPendingHITLRequestsByProject rows; use toDomainHITLRequest for
// single-resource queries that do not carry the joined context.
func toDomainHITLRequestFromList(r ListPendingHITLRequestsByProjectRow) *model.HITLRequest {
    req := toDomainHITLRequest(HitlRequest{ /* map common fields */ })
    req.RunID = &r.RunID
    req.StoryKey = r.StoryKey
    req.StoryTitle = r.StoryTitle
    if r.StoryObjective.Valid {
        req.StoryObjective = &r.StoryObjective.String
    }
    return req
}
```

**Router registration (backend/internal/api/router.go):**
```go
// Inside r.Route("/api/v1/projects/{projectId}", func(r chi.Router) { ... })
r.Get("/hitl/pending", hitlHandler.ListPendingHITLRequests)
```

**Pagination offset formula:**
```go
offset := (page - 1) * perPage
```

**Error codes:**
- `PROJECT_NOT_FOUND` — project does not exist (404) — already used by `projectRepo`
- No new error codes introduced

**Index availability:** `idx_hitl_requests_status` on `hitl_requests(status)` was created in migration `000013` (Story 5-1). Postgres will use this index filtered by `status = 'pending'`; the JOIN to `runs` uses the FK index on `run_steps.run_id` and `runs.project_id`. No additional migrations are required.

### Testing Requirements

**Unit tests additions to `hitl_service_test.go`:**

Mock additions needed: `MockProjectRepository` with `GetByID` method.

Table-driven cases for `ListPendingByProject`:
1. **Happy path** — `projectRepo.GetByID` succeeds, `hitlRepo.ListPendingByProject` returns 2 items and total=5; verify `PendingHITLListResult.Items` has length 2 and `Total == 5`
2. **Empty result** — `hitlRepo.ListPendingByProject` returns empty slice and total=0; verify no error, `Total == 0`, `Items` is empty (not nil)
3. **Project not found** — `projectRepo.GetByID` returns `errors.NewNotFound("project", id)`; verify error propagated, `hitlRepo` not called
4. **DB error** — `hitlRepo.ListPendingByProject` returns internal error; verify error propagated

**Handler test additions to `hitl_handler_test.go`:**
1. **Default pagination** — call with no query params; verify service called with `page=1, perPage=20`
2. **Custom pagination** — `?page=2&per_page=5`; verify service called with `page=2, perPage=5`
3. **200 response shape** — verify response body has `data` array and `pagination` object with `total`, `page`, `per_page` keys
4. **Service error** — service returns `PROJECT_NOT_FOUND`; verify `writeErrorResponse` renders 404

**golangci-lint:** Run `golangci-lint run ./...` from `backend/` — must pass before commit. CI enforces this.

### References

- `backend/internal/domain/port/hitl_repository.go` — port to extend
- `backend/internal/adapter/postgres/hitl_repo.go` — adapter to extend
- `backend/internal/domain/service/hitl_service.go` — service to extend (from Story 5-2)
- `backend/internal/api/handler/hitl_handler.go` — handler to extend (from Story 5-2)
- `backend/internal/api/handler/run_handler.go` — pagination pattern to follow (`ListRunsByProject`)
- `backend/queries/hitl_requests.sql` — existing HITL sqlc queries
- `backend/queries/runs.sql` — JOIN pattern reference
- `api/openapi.yaml` — `HITLRequest` schema, `Pagination` schema, `PageParam`/`PerPageParam` parameter refs
- `backend/migrations/000013_create_hitl_requests_table.up.sql` — table schema and index definitions

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-18 | Claude Sonnet 4.6 | Initial story creation |
