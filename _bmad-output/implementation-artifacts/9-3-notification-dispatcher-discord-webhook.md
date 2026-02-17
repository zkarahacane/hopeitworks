# Story 9.3: [BACK] Notification Dispatcher + Discord Webhook

Status: ready-for-dev

## Story

As a project user, I want to receive Discord or webhook notifications when key pipeline events occur, So that I stay informed about run completions, failures, and approval requests without polling the UI.

## Acceptance Criteria (BDD)

**AC1: notification_configs table exists**
- **Given** migrations are applied
- **When** the backend starts
- **Then** a `notification_configs` table exists with columns: `id` (UUID PK), `project_id` (FK projects CASCADE), `channel_type` (VARCHAR: `discord` or `webhook`), `config` (JSONB: `{"url":"..."}`), `events_filter` (JSONB array of event names), `enabled` (BOOLEAN DEFAULT true), `created_at`, `updated_at`
- **And** an index exists on `(project_id, enabled)`

**AC2: Discord notifier sends correctly formatted payloads**
- **Given** a Discord notifier configured with a webhook URL
- **When** `Send(ctx, event, config)` is called
- **Then** a POST request is made to `config["url"]` with Content-Type `application/json`
- **And** the body contains `{"embeds":[{"title":"<event_name>","description":"<summary>","color":<severity_color>}]}`
- **And** errors from the HTTP call are returned as `DomainError` internal

**AC3: Generic webhook notifier sends full event payload**
- **Given** a webhook notifier configured with a URL
- **When** `Send(ctx, event, config)` is called
- **Then** a POST request is made to `config["url"]` with the full `model.Event` JSON as body
- **And** non-2xx responses are returned as errors

**AC4: Dispatcher routes events to matching enabled configs**
- **Given** a project has multiple notification configs, some disabled, some filtering by event name
- **When** an event is published to the EventBus
- **Then** only enabled configs whose `events_filter` includes the event's EventName are dispatched
- **And** disabled configs are skipped silently
- **And** dispatch errors are logged with `slog.Warn` and do not stop other dispatches

**AC5: CRUD API for notification configs**
- **Given** a project admin is authenticated
- **When** calling `GET /api/v1/projects/{projectId}/notifications`
- **Then** returns list of notification configs for the project (no `config.url` masking at DB level; masking is UI responsibility)
- **And** `POST`, `PUT /notifications/{id}`, and `DELETE /notifications/{id}` endpoints exist and work correctly

**AC6: Dispatcher runs as background goroutine**
- **Given** the app starts
- **When** `NotificationDispatcher.Start(ctx)` is called in main.go
- **Then** it subscribes to the EventBus for all projects and runs until context cancellation

## Tasks / Subtasks

- [ ] [BACK] Task 1: DB migration for notification_configs table (AC: #1)
  - [ ] Create `backend/migrations/000014_create_notification_configs_table.up.sql`
  - [ ] Create `backend/migrations/000014_create_notification_configs_table.down.sql`

- [ ] [BACK] Task 2: sqlc queries for notification configs (AC: #1, #5)
  - [ ] Create `backend/queries/notification_configs.sql` with: `InsertNotificationConfig :one`, `GetNotificationConfig :one`, `ListNotificationConfigsByProject :many`, `UpdateNotificationConfig :one`, `DeleteNotificationConfig :exec`, `ListEnabledConfigsByProject :many`
  - [ ] Run `cd backend && sqlc generate`

- [ ] [BACK] Task 3: NotificationConfig domain model + Notifier port (AC: #2, #3)
  - [ ] Create `backend/internal/domain/model/notification_config.go`: `NotificationConfig` struct with all columns + `ChannelTypeDiscord`, `ChannelTypeWebhook` constants
  - [ ] Create `backend/internal/domain/port/notifier.go`: `Notifier` interface with `Send(ctx context.Context, event model.Event, config map[string]string) error`
  - [ ] Create `backend/internal/domain/port/notification_config_repository.go`: `NotificationConfigRepository` interface matching sqlc queries

- [ ] [BACK] Task 4: Discord notifier adapter (AC: #2)
  - [ ] Create `backend/internal/adapter/discord/notifier.go`
  - [ ] HTTP POST to `config["url"]` with Discord embed payload
  - [ ] Color map: `run.completed` → green (0x2ECC71), `run.failed` → red (0xE74C3C), `hitl_gate.pending` → yellow (0xF1C40F), default → grey (0x95A5A6)
  - [ ] Unit test with `httptest.Server`

- [ ] [BACK] Task 5: Generic webhook notifier adapter (AC: #3)
  - [ ] Create `backend/internal/adapter/webhook/notifier.go`
  - [ ] HTTP POST full `model.Event` JSON to `config["url"]`
  - [ ] Return error on non-2xx status
  - [ ] Unit test with `httptest.Server`

- [ ] [BACK] Task 6: NotificationConfigRepository postgres adapter (AC: #1, #5)
  - [ ] Create `backend/internal/adapter/postgres/notification_config_repository.go`
  - [ ] Implement all methods from port using sqlc-generated queries

- [ ] [BACK] Task 7: NotificationDispatcher service (AC: #4, #6)
  - [ ] Create `backend/internal/domain/service/notification_dispatcher.go`
  - [ ] `Start(ctx context.Context)` subscribes to `EventSubscriber` for all active projects, loops on event channel
  - [ ] Per event: fetch enabled configs via repo, filter by `events_filter`, call matching `Notifier.Send`
  - [ ] Errors logged via `slog.Warn`, never fatal
  - [ ] Unit test with mock EventSubscriber, mock Notifier, mock repo

- [ ] [BACK] Task 8: CRUD handlers + OpenAPI spec update (AC: #5)
  - [ ] Update `api/openapi.yaml`: add `NotificationConfig` schema, CRUD paths under `/projects/{projectId}/notifications`
  - [ ] Run `cd backend && make generate` to regenerate handler interfaces
  - [ ] Create `backend/internal/api/handler/notification_handler.go` implementing generated interface
  - [ ] Register routes in `backend/internal/api/handler/server.go`

- [ ] [BACK] Task 9: Wire dispatcher + adapters into DI (AC: #6)
  - [ ] Add providers in `backend/cmd/api/wire.go`: `postgres.NewNotificationConfigRepository`, `discord.NewNotifier`, `webhook.NewNotifier`, `service.NewNotificationDispatcher`
  - [ ] Add `dispatcher.Start(ctx)` call as goroutine in `backend/cmd/api/main.go`
  - [ ] Run `cd backend && wire ./cmd/api/`

- [ ] [BACK] Task 10: Lint + unit test validation (AC: #2, #3, #4)
  - [ ] Run `cd backend && golangci-lint run ./...` — must pass
  - [ ] Run `cd backend && go test ./... -short` — must pass

## Dev Notes

### Dependencies

- **Story 3.6 (DONE):** EventBus + EventSubscriber — `port.EventSubscriber` interface is in `backend/internal/domain/port/event_subscriber.go`
- **Migration sequence:** Last used number is `000013` (Story 9.1) — use `000014`

### Architecture Requirements

Notifier is a port with two adapters (discord, webhook). The dispatcher is a domain service, not an adapter.

```
NotificationDispatcher (domain/service)
    ├─ injects EventSubscriber (domain/port)
    ├─ injects NotificationConfigRepository (domain/port)
    └─ injects map[string]Notifier {"discord": ..., "webhook": ...}
              ├─ discord.Notifier (adapter/discord)
              └─ webhook.Notifier (adapter/webhook)
```

The dispatcher must handle projects being added after startup. For MVP, subscribe once at startup to a global channel or re-subscribe on project creation events.

### File Paths (exact)

```
backend/migrations/000014_create_notification_configs_table.up.sql      (new)
backend/migrations/000014_create_notification_configs_table.down.sql    (new)
backend/queries/notification_configs.sql                                 (new)
backend/internal/domain/model/notification_config.go                    (new)
backend/internal/domain/port/notifier.go                                (new)
backend/internal/domain/port/notification_config_repository.go          (new)
backend/internal/adapter/discord/notifier.go                            (new)
backend/internal/adapter/discord/notifier_test.go                       (new)
backend/internal/adapter/webhook/notifier.go                            (new)
backend/internal/adapter/webhook/notifier_test.go                       (new)
backend/internal/adapter/postgres/notification_config_repository.go     (new)
backend/internal/domain/service/notification_dispatcher.go              (new)
backend/internal/domain/service/notification_dispatcher_test.go         (new)
backend/internal/api/handler/notification_handler.go                    (new)
backend/internal/api/handler/server.go                                  (extend: register notification routes)
api/openapi.yaml                                                         (extend: NotificationConfig schema + CRUD paths)
backend/cmd/api/wire.go                                                  (extend)
backend/cmd/api/main.go                                                  (extend: start dispatcher goroutine)
```

### Technical Specifications

**Migration up (000014):**
```sql
CREATE TABLE notification_configs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    channel_type VARCHAR NOT NULL CHECK (channel_type IN ('discord', 'webhook')),
    config       JSONB NOT NULL DEFAULT '{}',
    events_filter JSONB NOT NULL DEFAULT '[]',
    enabled      BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notification_configs_project_enabled
    ON notification_configs(project_id, enabled);
```

**NotificationConfig model:**
```go
const (
    ChannelTypeDiscord = "discord"
    ChannelTypeWebhook = "webhook"
)

type NotificationConfig struct {
    ID           uuid.UUID
    ProjectID    uuid.UUID
    ChannelType  string
    Config       map[string]string // e.g., {"url": "https://discord.com/api/webhooks/..."}
    EventsFilter []string          // e.g., ["run.completed", "run.failed"]
    Enabled      bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

**Notifier port:**
```go
// Notifier dispatches a single notification for a given event.
type Notifier interface {
    Send(ctx context.Context, event model.Event, config map[string]string) error
}
```

**Discord embed payload:**
```go
type discordPayload struct {
    Embeds []discordEmbed `json:"embeds"`
}
type discordEmbed struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Color       int    `json:"color"`
}
```

**Dispatcher event loop:**
```go
func (d *NotificationDispatcher) Start(ctx context.Context) {
    // For MVP: subscribe at global level (all projects share one subscription key)
    // Implementation detail: use EventSubscriber.Subscribe with a sentinel projectID
    // or iterate known projects — defer to implementer
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case event := <-eventCh:
                d.dispatch(ctx, event)
            }
        }
    }()
}

func (d *NotificationDispatcher) dispatch(ctx context.Context, event model.Event) {
    configs, err := d.repo.ListEnabledConfigsByProject(ctx, event.ProjectID)
    if err != nil {
        slog.Warn("notification dispatch: list configs failed", "project_id", event.ProjectID, "err", err)
        return
    }
    eventName := event.EventName()
    for _, cfg := range configs {
        if !slices.Contains(cfg.EventsFilter, eventName) {
            continue
        }
        notifier, ok := d.notifiers[cfg.ChannelType]
        if !ok {
            continue
        }
        if err := notifier.Send(ctx, event, cfg.Config); err != nil {
            slog.Warn("notification send failed", "channel_type", cfg.ChannelType, "config_id", cfg.ID, "err", err)
        }
    }
}
```

**OpenAPI additions:**
```yaml
# Under /projects/{projectId}/notifications
GET:    list notification configs
POST:   create notification config
PUT /notifications/{id}:    update
DELETE /notifications/{id}: delete
```

**NotificationConfig schema:**
```yaml
NotificationConfig:
  type: object
  required: [id, project_id, channel_type, config, events_filter, enabled, created_at]
  properties:
    id: { type: string, format: uuid }
    project_id: { type: string, format: uuid }
    channel_type: { type: string, enum: [discord, webhook] }
    config: { type: object, additionalProperties: { type: string } }
    events_filter: { type: array, items: { type: string } }
    enabled: { type: boolean }
    created_at: { type: string, format: date-time }
    updated_at: { type: string, format: date-time }
```

### Testing Requirements

**Discord notifier unit test (`notifier_test.go`):**
- Starts `httptest.NewServer`, verifies POST body contains `embeds`
- Verifies correct color for `run.completed`, `run.failed`, `hitl_gate.pending`, unknown event

**Webhook notifier unit test (`notifier_test.go`):**
- Starts `httptest.NewServer`, verifies full event JSON is posted
- Non-2xx response → error returned

**Dispatcher unit test (`notification_dispatcher_test.go`):**
- Event matching `events_filter` → `Notifier.Send` called once
- Disabled config → `Notifier.Send` NOT called
- Event NOT in `events_filter` → `Notifier.Send` NOT called
- `Notifier.Send` error → logged, no panic, other configs still dispatched

### References

- EventSubscriber port: `backend/internal/domain/port/event_subscriber.go`
- Event model: `backend/internal/domain/model/event.go` (uses `EventName()` → `entity_type.action`)
- Existing adapter pattern: `backend/internal/adapter/action/agent_run.go`
- chi route registration: `backend/internal/api/handler/server.go`
- DomainError constructors: `backend/pkg/errors/`

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
