# Story 3-15: Fix live log streaming from agent container to browser via SSE

Status: ready-for-dev

## Story

As a **developer monitoring a running pipeline**,
I want **live log output from the agent container to stream in real time to the RunDetailView log viewer**,
so that **I can observe what the agent is doing without waiting for the run to complete or manually inspecting Docker logs**.

## Problem Statement

The live logs in `RunDetailView` permanently display "No log output yet" even while an agent container is actively running and producing NDJSON output. The infrastructure is partially built on both sides but two critical issues break the end-to-end flow:

1. **Payload shape mismatch**: The backend publishes the full `model.LogEvent` struct as the SSE event payload, but the frontend expects a flattened payload with fields `{ run_id, step_id, line, timestamp }`. The frontend also reads properties directly off the SSE `data` object instead of the nested `payload` field inside the `model.Event` wrapper.
2. **No throttling/batching**: Claude Code agents emit NDJSON at very high frequency (dozens of lines per second). Each line triggers a Postgres INSERT (via `EventPublisher.Publish`) + NOTIFY. This will overwhelm the event bus and the SSE connection under real load.

## Acceptance Criteria (BDD)

**AC1: Log lines appear in real time in the browser**
- **Given** a run is in `running` status with an active agent container producing NDJSON output
- **When** the user opens `RunDetailView` for that run
- **Then** log lines appear incrementally in the `LogViewer` component within 2 seconds of being emitted by the container
- **And** the "No log output yet" placeholder disappears as soon as the first log line arrives

**AC2: SSE event payload matches frontend expectations**
- **Given** the backend publishes a `log.emitted` SSE event
- **When** the frontend receives it via `useSSE`
- **Then** the `useRunLogs` composable can extract `run_id`, `step_id`, `line`, and `timestamp` from the event data
- **And** events with a different `run_id` are correctly ignored

**AC3: High-frequency log output does not overwhelm SSE**
- **Given** the agent emits more than 50 log lines per second
- **When** those lines are processed by the log streamer
- **Then** they are batched into groups and published as bulk SSE events (max one publish per 200ms)
- **And** no log lines are dropped (all are delivered, just with slight batching delay)

**AC4: Cost events are accumulated and recorded correctly**
- **Given** the agent emits NDJSON lines with `"type": "cost"` containing `input_tokens`, `output_tokens`, and `model`
- **When** the container exits
- **Then** all cost events are aggregated and a single `cost_records` row is inserted via `CostService.RecordStepCost`
- **And** cost lines are NOT forwarded to the SSE event bus (existing behavior preserved)

**AC5: Non-JSON log lines are handled gracefully**
- **Given** the agent container emits a plain text line (not valid JSON)
- **When** the line is parsed by `parseNDJSONLine`
- **Then** it is published as a `log.emitted` event with the raw text in the `line` field
- **And** `is_json` is set to `false` in the payload

**AC6: Log streaming stops cleanly on container exit**
- **Given** the agent container exits (code 0 or non-zero)
- **When** the log stream EOF is reached
- **Then** the log consumer goroutine finishes, any pending batch is flushed, and no further events are published
- **And** on non-zero exit, the last N lines are persisted as `log_tail` on the run step (existing behavior preserved)

## Technical Notes

### Root Cause Analysis

The end-to-end log flow is:

```
Container stdout (NDJSON)
  -> LogStreamer.StreamLogs (docker/log_streamer.go)
    -> parseNDJSONLine (docker/ndjson_parser.go)
      -> logCh (chan model.LogEvent)
        -> agent_run.go streamAndWait goroutine
          -> publishLogEvent
            -> EventPublisher.Publish (INSERT + NOTIFY)
              -> SSEHandler.writeSSEEvent
                -> EventSource in browser
                  -> useSSE -> useRunLogs -> LogViewer
```

**Issue 1 -- Payload shape mismatch (backend -> frontend)**

`publishLogEvent` in `agent_run.go` (line 331-350) marshals the entire `model.LogEvent` struct as the event payload:

```go
payload, err := json.Marshal(logEvent)  // full LogEvent with run_id, step_id, message, raw_line, is_json, data, type, ...
event := model.Event{
    Payload: payload,
}
```

The SSE handler then marshals the full `model.Event` wrapper:

```go
payload, _ := json.Marshal(event)  // { id, project_id, entity_type, entity_id, action, payload: {LogEvent...}, created_at }
fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.EventName(), payload)
```

The frontend `useRunLogs.ts` (line 14) expects:
```typescript
const payload = data as { run_id: string; line: string; timestamp: string }
```

But receives the full `Event` object, and `line` does not exist in `LogEvent` (it has `message` and `raw_line`).

**Issue 2 -- Frontend reads wrong nesting level**

`useRunLogs.ts` reads `data.run_id` directly, but the actual SSE data has `data.payload.run_id`. The `useSSE` composable passes the parsed JSON to `onEvent` callback, and `useRunLogs` treats it as the flat payload.

**Fix approach**: Two things need to happen:

1. **Backend**: Create a slim SSE-specific payload struct in `publishLogEvent` that matches frontend expectations: `{ run_id, step_id, line, timestamp }`. Use `raw_line` as `line` (or `message` for non-JSON lines).

2. **Frontend**: Fix `useRunLogs.ts` to correctly extract the payload from the SSE event data. The SSE event data is the full `model.Event` object, so the frontend needs to read `data.payload.run_id`, `data.payload.line`, etc.

### Key Files

| File | Role | Change |
|------|------|--------|
| `backend/internal/adapter/action/agent_run.go` | Consumes logCh, publishes events | Restructure `publishLogEvent` payload; add batching to `streamAndWait` goroutine |
| `backend/internal/adapter/docker/ndjson_parser.go` | Parses NDJSON lines | No changes needed |
| `backend/internal/adapter/docker/log_streamer.go` | Streams Docker container logs | No changes needed |
| `backend/internal/domain/model/log_event.go` | LogEvent struct | No changes needed |
| `backend/internal/domain/model/event.go` | Event struct (SSE wrapper) | No changes needed |
| `backend/internal/domain/port/event_publisher.go` | EventPublisher interface | No changes needed |
| `backend/internal/api/handler/sse_handler.go` | Writes SSE frames | No changes needed |
| `frontend/src/features/runs/composables/useRunLogs.ts` | Filters log.emitted SSE events | Fix payload extraction path |
| `frontend/src/features/runs/RunLogViewer.vue` | Renders LogViewer | No changes needed |
| `frontend/src/ui/composed/LogViewer.vue` | Pure display component | No changes needed |
| `frontend/src/composables/useSSE.ts` | SSE connection manager | No changes needed |
| `backend/internal/adapter/action/agent_run_test.go` | Tests for agent_run | Update test assertions for new payload shape and batching |

### Payload Contract (SSE data for `log.emitted`)

The SSE frame sent by `writeSSEEvent` looks like:
```
event: log.emitted
data: {"id":"<event-uuid>","project_id":"<uuid>","entity_type":"log","entity_id":"<step-uuid>","action":"emitted","payload":<inner-json>,"created_at":"..."}
id: <event-uuid>
```

The `payload` field (inner JSON) must match this shape for frontend consumption:

```json
{
  "run_id": "uuid-string",
  "step_id": "uuid-string",
  "line": "raw log text or message",
  "timestamp": "2026-02-22T10:30:00Z",
  "level": "info",
  "is_json": true
}
```

The frontend `useRunLogs.ts` must extract from the outer Event object:
```typescript
const event = data as { payload: { run_id: string; step_id: string; line: string; timestamp: string } }
const payload = event.payload
```

### Batching Strategy

To avoid overwhelming Postgres and SSE with per-line INSERTs and NOTIFYs at high frequency:

1. In the `streamAndWait` log consumption goroutine, replace the per-event `publishLogEvent` call with a batch accumulator
2. Use a ticker (200ms interval) to flush accumulated log events as a single bulk publish
3. On channel close (container exit), flush remaining batch immediately
4. Each batch publish creates ONE `model.Event` with a `payload` containing an array of log entries
5. Alternatively (simpler): keep individual events but use a rate-limited publisher that coalesces events within a 200ms window into a single Postgres INSERT + NOTIFY

**Recommended approach (simpler, less change)**: Batch log events in the goroutine, publish a single event containing an array of log lines every 200ms. The frontend then unpacks the array.

Updated payload contract for batched events:
```json
{
  "run_id": "uuid-string",
  "step_id": "uuid-string",
  "lines": [
    { "line": "...", "timestamp": "...", "level": "info", "is_json": true },
    { "line": "...", "timestamp": "...", "level": "info", "is_json": false }
  ]
}
```

Frontend `useRunLogs.ts` then loops over `payload.lines` and pushes each entry to the `lines` ref.

### Implementation Details

#### Backend: `publishLogEvent` replacement in `agent_run.go`

Define a slim payload struct for SSE:

```go
// logSSELine is the per-line shape sent inside a log.emitted SSE event.
type logSSELine struct {
    Line      string    `json:"line"`
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    IsJSON    bool      `json:"is_json"`
}

// logSSEPayload is the payload for a batched log.emitted SSE event.
type logSSEPayload struct {
    RunID  string       `json:"run_id"`
    StepID string       `json:"step_id"`
    Lines  []logSSELine `json:"lines"`
}
```

In `streamAndWait`, replace the direct `publishLogEvent` call with:

```go
const logFlushInterval = 200 * time.Millisecond

var logBatch []logSSELine
flushTicker := time.NewTicker(logFlushInterval)
defer flushTicker.Stop()

flushLogs := func() {
    if len(logBatch) == 0 {
        return
    }
    a.publishLogBatch(ctx, runCtx, logBatch)
    logBatch = logBatch[:0]
}

// In the goroutine consuming logCh:
for {
    select {
    case logEvent, ok := <-logCh:
        if !ok {
            flushLogs() // flush remaining on channel close
            return
        }
        if logEvent.Type == "cost" {
            costEvents = append(costEvents, model.CostEvent{...})
            continue
        }
        line := logEvent.Message
        if line == "" {
            line = logEvent.RawLine
        }
        logBatch = append(logBatch, logSSELine{
            Line:      line,
            Timestamp: logEvent.Timestamp,
            Level:     logEvent.Level,
            IsJSON:    logEvent.IsJSON,
        })
        // Also maintain ring buffer for log tail
        if len(logTail) >= tailSize {
            logTail = logTail[1:]
        }
        logTail = append(logTail, logEvent.Message)
    case <-flushTicker.C:
        flushLogs()
    }
}
```

New `publishLogBatch` method:

```go
func (a *AgentRunAction) publishLogBatch(ctx context.Context, runCtx *model.RunContext, lines []logSSELine) {
    payload := logSSEPayload{
        RunID:  runCtx.Run.ID.String(),
        StepID: runCtx.RunStep.ID.String(),
        Lines:  lines,
    }
    payloadJSON, err := json.Marshal(payload)
    if err != nil {
        a.logger.Error("failed to marshal log batch", "error", err)
        return
    }
    event := model.Event{
        ID:         uuid.New(),
        ProjectID:  runCtx.ProjectID,
        EntityType: "log",
        EntityID:   runCtx.RunStep.ID,
        Action:     "emitted",
        Payload:    payloadJSON,
    }
    if err := a.eventPub.Publish(ctx, event); err != nil {
        a.logger.Warn("failed to publish log batch", "error", err, "line_count", len(lines))
    }
}
```

Remove the old `publishLogEvent` method entirely.

#### Frontend: `useRunLogs.ts` fix

```typescript
export function useRunLogs(projectId: string, runId: string) {
  const lines = ref<LogLine[]>([])

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName !== 'log.emitted') return

    // SSE data is the full Event object; extract nested payload
    const event = data as {
      payload: {
        run_id: string
        step_id: string
        lines: Array<{ line: string; timestamp: string; level: string; is_json: boolean }>
      }
    }

    // payload is a JSON string inside the Event — parse if needed
    let payload = event.payload
    if (typeof payload === 'string') {
      try { payload = JSON.parse(payload) } catch { return }
    }

    if (payload.run_id !== runId) return

    for (const entry of payload.lines) {
      lines.value.push({
        text: entry.line,
        timestamp: new Date(entry.timestamp),
      })
    }
  })

  function clearLogs() {
    lines.value = []
  }

  return { lines, sseStatus, clearLogs }
}
```

**Note on `payload` field type**: In the SSE frame, `writeSSEEvent` does `json.Marshal(event)` which marshals `Payload json.RawMessage` as raw JSON embedded in the outer object. Since `json.RawMessage` is `[]byte`, when marshaled it appears as the raw JSON inline (not double-encoded). So the frontend receives `event.payload` as a parsed object, not a string. The `typeof === 'string'` guard is defensive.

## Tasks / Subtasks

- [ ] [BACK] Task 1: Refactor `publishLogEvent` into batched `publishLogBatch` in `agent_run.go`
  - [ ] Define `logSSELine` and `logSSEPayload` structs with correct JSON tags
  - [ ] Replace single-event publishing in `streamAndWait` goroutine with batch accumulator + 200ms flush ticker
  - [ ] Implement `publishLogBatch` method that marshals `logSSEPayload` and publishes one `model.Event`
  - [ ] Use `logEvent.Message` as `line` (fall back to `logEvent.RawLine` if message is empty)
  - [ ] Flush remaining batch on `logCh` close (before goroutine returns)
  - [ ] Remove old `publishLogEvent` method
  - [ ] Preserve existing ring buffer (`logTail`) and cost event accumulation behavior

- [ ] [BACK] Task 2: Update `agent_run_test.go` for new batched payload
  - [ ] Update `TestAgentRunAction_Execute` assertions to expect batched payload shape (`lines` array inside payload)
  - [ ] Verify payload contains `run_id`, `step_id`, and `lines` array
  - [ ] Verify cost events are still excluded from published events
  - [ ] Add test case for high-frequency log emission to verify batching behavior (multiple log lines result in fewer events)
  - [ ] Verify flush on channel close delivers remaining lines

- [ ] [FRONT] Task 3: Fix `useRunLogs.ts` payload extraction
  - [ ] Update type assertion to match actual SSE Event wrapper shape (extract `data.payload`)
  - [ ] Handle batched `lines` array: iterate and push each entry to `lines` ref
  - [ ] Handle `payload` being either an object or a JSON string (defensive parsing)
  - [ ] Keep `run_id` filtering on `payload.run_id`

- [ ] [FRONT] Task 4: Update `useRunLogs.spec.ts` for new payload shape
  - [ ] Update mock SSE event data to use the full `Event` wrapper with nested `payload`
  - [ ] Update payload to use `lines` array instead of single `line` field
  - [ ] Test that multiple lines in a single batch event all get appended
  - [ ] Test that `run_id` filtering still works on the nested payload
  - [ ] Test that non-`log.emitted` events are still ignored

- [ ] [BACK] Task 5: Verify end-to-end SSE flow with integration test
  - [ ] Add or update integration test that publishes a `log.emitted` event and verifies it arrives via the SSE handler with correct payload shape
  - [ ] Verify `entity_type="log"`, `action="emitted"`, `EventName()="log.emitted"`
  - [ ] Verify the SSE `event:` field matches `log.emitted` so the frontend `addEventListener('log.emitted', ...)` triggers

## Out of Scope

- Log persistence to database (logs are ephemeral, only `log_tail` on failure is persisted)
- Log search/filtering in the frontend (future story)
- WebSocket upgrade from SSE (MVP uses SSE)
- Log level filtering in the UI (future enhancement)
- Backpressure handling if frontend cannot keep up (future story)
