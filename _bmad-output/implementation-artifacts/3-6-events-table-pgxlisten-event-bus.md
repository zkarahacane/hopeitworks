# Story 3.6: [BACK] Events Table + pgxlisten Event Bus

Status: ready-for-dev

## Story

As a backend developer, I want an event log table with Postgres LISTEN/NOTIFY integration, so that all system events are persisted and broadcast to subscribers in real-time.

## Acceptance Criteria (BDD)

**AC1: Events table schema supports append-only event log**
- **Given** the database is initialized
- **When** I query the schema for the events table
- **Then** it contains: id (UUID PK), project_id (FK projects CASCADE), entity_type (VARCHAR NOT NULL), entity_id (UUID NOT NULL), action (VARCHAR NOT NULL), payload (JSONB), created_at (TIMESTAMPTZ NOT NULL DEFAULT now())
- **And** indexes exist on (project_id, created_at) and (entity_type, entity_id)
- **And** table is append-only (no UPDATE or DELETE triggers)

**AC2: Postgres trigger broadcasts events via NOTIFY on insert**
- **Given** the events table exists with notify trigger
- **When** I insert a new event with entity_type="run", action="started"
- **Then** Postgres fires NOTIFY on channel "events"
- **And** notification payload contains JSON with id, project_id, entity_type, entity_id, action

**AC3: EventPublisher persists events to database**
- **Given** an EventPublisher instance
- **When** I call Publish(ctx, event) with entity_type="run.started"
- **Then** a new row is inserted into events table
- **And** created_at is set to current timestamp
- **And** Postgres trigger fires NOTIFY automatically

**AC4: EventSubscriber subscribes to project events via pgxlisten**
- **Given** a Postgres connection with pgxlisten
- **When** I call Subscribe(ctx, projectID)
- **Then** a channel is returned that receives events for that project
- **And** the subscriber listens on Postgres channel "events"
- **And** events from other projects are filtered out

**AC5: EventSubscriber auto-reconnects on connection loss**
- **Given** an active subscription with pgxlisten
- **When** the Postgres connection is dropped
- **Then** pgxlisten attempts reconnection with exponential backoff
- **And** subscription resumes after successful reconnection
- **And** no events are lost (events are persisted in DB)

**AC6: Event format follows dot-notation convention**
- **Given** I publish events for different entities
- **When** I set entity_type and action
- **Then** event format is "{entity_type}.{action}" (e.g., "run.started", "step.completed", "hitl.pending")
- **And** payload uses snake_case JSON fields

**AC7: Unsubscribe cleanup releases resources**
- **Given** an active subscription
- **When** I call the cleanup function returned by Subscribe
- **Then** the subscription channel is closed
- **And** no further events are received
- **And** Postgres connection resources are released

## Tasks / Subtasks

- [ ] [BACK] Task 1: Create migration for events table with NOTIFY trigger (AC: #1, #2)
  - [ ] Write 000008_create_events_table.up.sql with schema, indexes, append-only constraints
  - [ ] Create notify_event() trigger function that fires pg_notify on INSERT
  - [ ] Attach trigger to events table AFTER INSERT FOR EACH ROW
  - [ ] Write 000008_create_events_table.down.sql with DROP TRIGGER, DROP FUNCTION, DROP TABLE
  - [ ] Test migration: up, down, up cycle validates schema and trigger

- [ ] [BACK] Task 2: Create sqlc queries for events (AC: #3)
  - [ ] Write backend/queries/events.sql: CreateEvent, ListEventsByProject, GetEventsByEntityID
  - [ ] Run `make generate` to generate sqlc types
  - [ ] Verify generated code matches Event domain model

- [ ] [BACK] Task 3: Implement domain model and port interfaces (AC: #3, #4, #6, #7)
  - [ ] Create backend/internal/domain/model/event.go with Event struct
  - [ ] Create backend/internal/domain/port/event_publisher.go with EventPublisher interface: Publish(ctx, event) error
  - [ ] Create backend/internal/domain/port/event_subscriber.go with EventSubscriber interface: Subscribe(ctx, projectID) (chan Event, func(), error), Close() error

- [ ] [BACK] Task 4: Implement EventPublisher Postgres adapter (AC: #3)
  - [ ] Create backend/internal/adapter/postgres/event_repo.go implementing EventPublisher
  - [ ] Publish(ctx, event): insert into events table via sqlc
  - [ ] Map DB errors to DomainErrors (CategoryNotFound for missing project FK)
  - [ ] Write unit test: publish event, verify DB row, verify trigger fires NOTIFY

- [ ] [BACK] Task 5: Implement EventSubscriber with pgxlisten (AC: #4, #5, #7)
  - [ ] Create backend/internal/adapter/postgres/event_bus.go implementing EventSubscriber
  - [ ] Create dedicated pgx connection (separate from pool) for LISTEN
  - [ ] Subscribe(ctx, projectID): LISTEN on "events" channel, filter by projectID, return buffered chan Event + cleanup func
  - [ ] Implement auto-reconnection with exponential backoff (max 5 retries, 1s → 32s)
  - [ ] Close() gracefully closes all subscriptions and Postgres connection
  - [ ] Parse NOTIFY payload JSON and unmarshal into Event struct

- [ ] [BACK] Task 6: Write unit tests for event publisher (AC: #3)
  - [ ] Test EventPublisher.Publish: success inserts row with correct fields
  - [ ] Test EventPublisher.Publish: returns error for missing project FK
  - [ ] Test event format follows dot-notation convention (entity_type.action)

- [ ] [BACK] Task 7: Write integration tests for event bus (AC: #2, #4, #5, #7)
  - [ ] Test Subscribe + Publish: subscriber receives event via pgxlisten
  - [ ] Test project filtering: subscriber only receives events for subscribed project
  - [ ] Test cleanup function: closes channel, stops receiving events
  - [ ] Test reconnection: kill Postgres connection, verify reconnect + resume
  - [ ] Use testcontainer for Postgres integration test

- [ ] [BACK] Task 8: Wire dependencies and verify end-to-end (AC: #1-#7)
  - [ ] Add EventPublisher and EventSubscriber to wire.go provider sets
  - [ ] Create EventBus provider: NewEventBus(cfg, logger) with dedicated pgx connection
  - [ ] Run `go generate ./cmd/api` to regenerate wire_gen.go
  - [ ] Write E2E test: create project → publish run.started event → verify subscriber receives event
  - [ ] Verify migrations apply cleanly in testcontainer

## Dev Notes

### Dependencies
- **Story 1-1:** Go scaffolding, pgx/v5, docker-compose dev stack
- **Story 1-5:** projects table must exist (FK project_id)
- **pgxlisten:** Use pgx/v5 native LISTEN/NOTIFY support (no external library needed)

### Architecture Requirements
- Hexagonal architecture: domain/model → domain/port → adapter/postgres
- EventPublisher and EventSubscriber are separate ports (write vs. read)
- EventBus (pgxlisten wrapper) maintains dedicated Postgres connection for LISTEN
- Event persistence via EventPublisher, real-time broadcast via EventSubscriber
- No business logic in adapters — domain service layer will orchestrate publish + subscribe

### File Paths (exact)

```
backend/migrations/000008_create_events_table.up.sql
backend/migrations/000008_create_events_table.down.sql
backend/queries/events.sql
backend/internal/domain/model/event.go
backend/internal/domain/port/event_publisher.go
backend/internal/domain/port/event_subscriber.go
backend/internal/adapter/postgres/event_repo.go
backend/internal/adapter/postgres/event_bus.go
backend/internal/adapter/postgres/event_bus_test.go
backend/cmd/api/wire.go                              # Add providers
backend/cmd/api/wire_gen.go                          # Auto-generated
```

### Technical Specifications

#### Domain Model

```go
// backend/internal/domain/model/event.go
package model

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

type Event struct {
    ID         uuid.UUID       `json:"id"`
    ProjectID  uuid.UUID       `json:"project_id"`
    EntityType string          `json:"entity_type"`  // e.g., "run", "step", "hitl"
    EntityID   uuid.UUID       `json:"entity_id"`
    Action     string          `json:"action"`       // e.g., "started", "completed", "pending"
    Payload    json.RawMessage `json:"payload"`
    CreatedAt  time.Time       `json:"created_at"`
}

// EventName returns dot-notation event name (entity_type.action)
func (e Event) EventName() string {
    return e.EntityType + "." + e.Action
}
```

#### Port Interfaces

```go
// backend/internal/domain/port/event_publisher.go
package port

import (
    "context"
    "hopeitworks/backend/internal/domain/model"
)

type EventPublisher interface {
    Publish(ctx context.Context, event model.Event) error
}
```

```go
// backend/internal/domain/port/event_subscriber.go
package port

import (
    "context"
    "github.com/google/uuid"
    "hopeitworks/backend/internal/domain/model"
)

type EventSubscriber interface {
    // Subscribe returns a channel of events for the given project and a cleanup function
    Subscribe(ctx context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error)

    // Close gracefully shuts down all subscriptions
    Close() error
}
```

#### Migration Schema with Trigger

```sql
-- 000008_create_events_table.up.sql
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes for common query patterns
CREATE INDEX idx_events_project_id_created_at ON events(project_id, created_at);
CREATE INDEX idx_events_entity_type_entity_id ON events(entity_type, entity_id);

-- Trigger function to broadcast events via NOTIFY
CREATE OR REPLACE FUNCTION notify_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('events', json_build_object(
        'id', NEW.id,
        'project_id', NEW.project_id,
        'entity_type', NEW.entity_type,
        'entity_id', NEW.entity_id,
        'action', NEW.action
    )::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to events table
CREATE TRIGGER events_notify_trigger
    AFTER INSERT ON events
    FOR EACH ROW EXECUTE FUNCTION notify_event();
```

```sql
-- 000008_create_events_table.down.sql
DROP TRIGGER IF EXISTS events_notify_trigger ON events;
DROP FUNCTION IF EXISTS notify_event();
DROP TABLE IF EXISTS events;
```

#### EventBus Implementation (pgxlisten)

```go
// backend/internal/adapter/postgres/event_bus.go
package postgres

import (
    "context"
    "encoding/json"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "hopeitworks/backend/internal/domain/model"
    "hopeitworks/backend/internal/domain/port"
)

type EventBus struct {
    conn   *pgx.Conn
    logger *slog.Logger
}

func NewEventBus(ctx context.Context, connString string, logger *slog.Logger) (*EventBus, error) {
    // Create dedicated connection for LISTEN (separate from connection pool)
    conn, err := pgx.Connect(ctx, connString)
    if err != nil {
        return nil, err
    }

    return &EventBus{conn: conn, logger: logger}, nil
}

func (b *EventBus) Subscribe(ctx context.Context, projectID uuid.UUID) (<-chan model.Event, func(), error) {
    // Start listening on "events" channel
    _, err := b.conn.Exec(ctx, "LISTEN events")
    if err != nil {
        return nil, nil, err
    }

    eventChan := make(chan model.Event, 100) // Buffered channel
    stopChan := make(chan struct{})

    go func() {
        defer close(eventChan)

        for {
            select {
            case <-stopChan:
                return
            case <-ctx.Done():
                return
            default:
                // Wait for notification with timeout
                notification, err := b.conn.WaitForNotification(context.Background())
                if err != nil {
                    // Implement reconnection logic with exponential backoff
                    b.logger.Error("notification error, reconnecting", "error", err)
                    time.Sleep(1 * time.Second)
                    continue
                }

                // Parse notification payload
                var notif struct {
                    ID         uuid.UUID `json:"id"`
                    ProjectID  uuid.UUID `json:"project_id"`
                    EntityType string    `json:"entity_type"`
                    EntityID   uuid.UUID `json:"entity_id"`
                    Action     string    `json:"action"`
                }
                if err := json.Unmarshal([]byte(notification.Payload), &notif); err != nil {
                    b.logger.Error("failed to parse notification", "error", err)
                    continue
                }

                // Filter by project_id
                if notif.ProjectID != projectID {
                    continue
                }

                // Fetch full event from DB (notification only has metadata)
                var event model.Event
                row := b.conn.QueryRow(ctx,
                    "SELECT id, project_id, entity_type, entity_id, action, payload, created_at FROM events WHERE id = $1",
                    notif.ID)
                if err := row.Scan(&event.ID, &event.ProjectID, &event.EntityType, &event.EntityID, &event.Action, &event.Payload, &event.CreatedAt); err != nil {
                    b.logger.Error("failed to fetch event", "error", err)
                    continue
                }

                eventChan <- event
            }
        }
    }()

    cleanup := func() {
        close(stopChan)
        _, _ = b.conn.Exec(context.Background(), "UNLISTEN events")
    }

    return eventChan, cleanup, nil
}

func (b *EventBus) Close() error {
    return b.conn.Close(context.Background())
}
```

#### sqlc Queries

```sql
-- backend/queries/events.sql

-- name: CreateEvent :one
INSERT INTO events (id, project_id, entity_type, entity_id, action, payload, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListEventsByProject :many
SELECT * FROM events
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetEventsByEntityID :many
SELECT * FROM events
WHERE entity_type = $1 AND entity_id = $2
ORDER BY created_at ASC;
```

### Testing Requirements

1. **Unit Tests (backend/internal/adapter/postgres/event_repo_test.go)**
   - Publish event: success inserts row, trigger fires NOTIFY
   - Publish event: returns error for missing project FK
   - Event format validation: entity_type.action dot-notation

2. **Integration Tests (backend/internal/adapter/postgres/event_bus_test.go)**
   - Subscribe + Publish: subscriber receives event via pgxlisten
   - Project filtering: only receives events for subscribed project
   - Cleanup function: closes channel, stops receiving events
   - Reconnection: kill Postgres connection, verify auto-reconnect
   - Use testcontainer for Postgres

3. **E2E Test**
   - Create project → publish run.started event → verify subscriber receives event with correct payload
   - Verify Postgres trigger fires automatically on INSERT

4. **Linting**
   - Run `golangci-lint run ./...` — must pass before commit

### References

- Story 1-1: Go scaffolding, pgx/v5
- Story 1-5: projects table schema
- Epic 3 Planning: Event bus design
- `backend/.golangci.yml`: Linting rules
- pgx/v5 docs: LISTEN/NOTIFY support

## Dev Agent Record

## Change Log
