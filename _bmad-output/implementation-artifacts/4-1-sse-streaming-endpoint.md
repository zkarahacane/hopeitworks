# Story 4.1: [BACK] SSE Streaming Endpoint

Status: ready-for-dev

## Story

As a frontend client, I want a persistent SSE connection to receive project events in real time, So that the UI can update without polling.

## Acceptance Criteria (BDD)

**AC1: SSE endpoint streams events to connected clients**
- **Given** an authenticated user with access to project P connects to `GET /api/v1/events/stream?project_id={P}`
- **When** any event is published for project P via the EventBus
- **Then** the client receives an SSE message in the format `event: {entity_type}.{action}\ndata: {json_payload}\nid: {event_id}\n\n` within 1 second

**AC2: Last-Event-ID replay delivers missed events**
- **Given** a client reconnects with header `Last-Event-ID: {uuid}`
- **When** the SSE handler processes the connection
- **Then** it queries `GetEventsSince(ctx, projectID, lastEventID)` and streams each missed event before subscribing to live notifications
- **And** events are replayed in ascending `created_at` order

**AC3: Keepalive heartbeat prevents proxy timeouts**
- **Given** a connected SSE client with no new events
- **When** 30 seconds elapse without an event
- **Then** the handler writes `: keepalive\n\n` and flushes the response writer

**AC4: Auth and project access are enforced**
- **Given** a request without a valid JWT cookie
- **When** the SSE endpoint is called
- **Then** the connection is rejected with HTTP 401
- **Given** a valid JWT but the user is not a member of project P
- **When** the SSE endpoint is called with `project_id={P}`
- **Then** the connection is rejected with HTTP 403

**AC5: Client disconnect terminates the subscription cleanly**
- **Given** a connected SSE client
- **When** the client disconnects (closes browser tab or network drops)
- **Then** `r.Context().Done()` triggers, the cleanup function is called, and the goroutine exits without leaking

## Tasks / Subtasks

- [ ] [BACK] Task 1: Add `GetEventsSince` sqlc query and regenerate (AC: #2)
  - [ ] In `backend/queries/events.sql`, add query `GetEventsSince :many` — selects events where `project_id = $1 AND created_at > (SELECT created_at FROM events WHERE id = $2)` ordered by `created_at ASC`
  - [ ] Run `cd backend && sqlc generate` to generate `db.GetEventsSince`

- [ ] [BACK] Task 2: Add `EventRepository` port with `GetEventsSince` (AC: #2)
  - [ ] Create `backend/internal/domain/port/event_repository.go`
  - [ ] Define `EventRepository` interface with `GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error)`

- [ ] [BACK] Task 3: Implement `EventRepository` in the postgres adapter (AC: #2)
  - [ ] Add method `GetEventsSince` to `backend/internal/adapter/postgres/event_repo.go` implementing the `EventRepository` port
  - [ ] Map `pgx.ErrNoRows` for the anchor event (unknown `afterEventID`) → return empty slice (idempotent replay)

- [ ] [BACK] Task 4: Create `SSEHandler` struct with constructor (AC: #1, #3, #4, #5)
  - [ ] Create `backend/internal/api/handler/sse_handler.go`
  - [ ] `SSEHandler` holds `eventSub port.EventSubscriber`, `eventRepo port.EventRepository`, `projectUserRepo port.ProjectUserRepository`, `authService *service.AuthService`
  - [ ] Constructor `NewSSEHandler(...)` wires all dependencies

- [ ] [BACK] Task 5: Implement `ServeHTTP` — auth, project access, Last-Event-ID validation (AC: #1, #2, #4)
  - [ ] Parse `project_id` query param; reject with 400 if missing or not a valid UUID
  - [ ] Extract user from context via `middleware.UserIDFromContext`; reject 401 if absent
  - [ ] Check project membership via `projectUserRepo.IsMember(ctx, projectID, userID)`; reject 403 if not member
  - [ ] Parse `Last-Event-ID` header; if present and valid UUID, call `eventRepo.GetEventsSince` and flush replay events first

- [ ] [BACK] Task 6: Implement SSE streaming loop with keepalive (AC: #1, #3, #5)
  - [ ] Set response headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`
  - [ ] Assert `http.Flusher` support; return 500 if not available
  - [ ] Call `eventSub.Subscribe(r.Context(), projectID)` to get the event channel and cleanup func; defer cleanup
  - [ ] Loop: `select` on event channel → write SSE frame; 30s ticker → write `: keepalive\n\n`; `r.Context().Done()` → return
  - [ ] Helper `writeSSEEvent(w http.Flusher, event model.Event)` writes the full SSE frame and flushes

- [ ] [BACK] Task 7: Register SSE route manually in `main.go` (AC: #4)
  - [ ] Instantiate `SSEHandler` in `run()` after all deps are wired
  - [ ] Register route outside oapi-codegen mux: `r.With(authmw.Auth(authService)).Get("/api/v1/events/stream", sseHandler.ServeHTTP)`
  - [ ] Ensure the route is mounted before `handler.HandlerFromMuxWithBaseURL`

- [ ] [BACK] Task 8: Write unit tests for SSEHandler (AC: #1, #2, #4, #5)
  - [ ] File: `backend/internal/api/handler/sse_handler_test.go`
  - [ ] Test: missing `project_id` → 400
  - [ ] Test: unauthenticated (no user in context) → 401
  - [ ] Test: non-member → 403
  - [ ] Test: valid connection receives SSE frame for event published to mock subscriber
  - [ ] Test: Last-Event-ID header triggers replay of mock events before live stream
  - [ ] Use hand-written mocks for `EventSubscriber`, `EventRepository`, `ProjectUserRepository`

## Dev Notes

### Dependencies

- Story 3.6 (events table + pgxlisten EventBus) — DONE. `postgres.EventBus` implements `port.EventSubscriber`.
- `postgres.EventRepo` already exists in `backend/internal/adapter/postgres/event_repo.go` — extend it rather than creating new.
- `RequireProjectAccess` middleware in `backend/internal/api/middleware/rbac.go` uses `port.ProjectUserRepository`; use the same repo for project membership check inside SSEHandler.

### Architecture Requirements

- SSE is a long-lived HTTP connection — it must NOT go through the oapi-codegen generated mux (no OpenAPI schema for it). Register manually on the chi router in `main.go`.
- The `http.Server.WriteTimeout` must be zero or very large for SSE connections; document this constraint in the handler comment (MVP: rely on proxy-level timeout).
- `EventBus.Subscribe` already handles fan-out per project — no additional pub/sub layer needed.
- The handler goroutine owns the response writer for its lifetime; no concurrent writes.

### File Paths (exact)

```
backend/queries/events.sql                                   (add GetEventsSince query)
backend/internal/domain/port/event_repository.go             (new)
backend/internal/adapter/postgres/event_repo.go              (add GetEventsSince method)
backend/internal/api/handler/sse_handler.go                  (new)
backend/internal/api/handler/sse_handler_test.go             (new)
backend/cmd/api/main.go                                       (wire SSEHandler, register route)
```

### Technical Specifications

**New sqlc query in `backend/queries/events.sql`:**
```sql
-- name: GetEventsSince :many
SELECT e.*
FROM events e
WHERE e.project_id = $1
  AND e.created_at > (
      SELECT created_at FROM events WHERE id = $2
  )
ORDER BY e.created_at ASC;
```

**`EventRepository` port:**
```go
package port

import (
    "context"
    "github.com/google/uuid"
    "github.com/zakari/hopeitworks/backend/internal/domain/model"
)

// EventRepository defines read access to persisted events.
type EventRepository interface {
    // GetEventsSince returns all events for the project created after the event
    // identified by afterEventID. Returns empty slice if afterEventID is unknown.
    GetEventsSince(ctx context.Context, projectID uuid.UUID, afterEventID uuid.UUID) ([]*model.Event, error)
}
```

**`SSEHandler` struct:**
```go
type SSEHandler struct {
    eventSub       port.EventSubscriber
    eventRepo      port.EventRepository
    projectUserRepo port.ProjectUserRepository
    logger         *slog.Logger
}
```

**SSE frame format:**
```
event: run.started
data: {"id":"...","project_id":"...","entity_type":"run","entity_id":"...","action":"started","payload":{...},"created_at":"..."}
id: <event-uuid>

```
(blank line terminates each frame)

**Keepalive frame format:**
```
: keepalive

```
(comment line — browsers and EventSource clients ignore it but it flushes the connection)

**`writeSSEEvent` helper:**
```go
func writeSSEEvent(w io.Writer, f http.Flusher, event model.Event) error {
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }
    _, err = fmt.Fprintf(w, "event: %s\ndata: %s\nid: %s\n\n",
        event.EventName(), payload, event.ID)
    if err != nil {
        return err
    }
    f.Flush()
    return nil
}
```

**Route registration pattern in `main.go` (after existing route registrations):**
```go
sseHandler := handler.NewSSEHandler(eventBus, eventRepo, projectUserRepo, logger)
r.With(authmw.Auth(authService)).Get("/api/v1/events/stream", sseHandler.ServeHTTP)
```

**`ProjectUserRepository.IsMember` — check existing interface in `backend/internal/domain/port/project_user_repository.go`; if `IsMember` does not exist, use `GetProjectUser` and check for not-found error.**

### Testing Requirements

- Mock `EventSubscriber.Subscribe` returns a channel; test pushes an event to the channel and asserts the response body contains the SSE frame.
- Use `httptest.NewRecorder` for unit tests; note that `ResponseRecorder` implements `http.Flusher` via `Flush()` no-op.
- For replay test: mock `EventRepository.GetEventsSince` returns 2 events; assert both appear in the response body before any live events.
- All tests run with `-short` (no real Postgres).

### References

- `backend/internal/adapter/postgres/event_bus.go` — `EventBus.Subscribe` signature
- `backend/internal/domain/model/event.go` — `Event.EventName()`
- `backend/internal/api/middleware/rbac.go` — `RequireProjectAccess` pattern
- `backend/internal/api/middleware/auth.go` — `UserIDFromContext`, `SetUserContext`
- `backend/cmd/api/main.go` — manual route registration pattern (see `/api/v1/projects/{id}/users`)
- MDN SSE specification: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
