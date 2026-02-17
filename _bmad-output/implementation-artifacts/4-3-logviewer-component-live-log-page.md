# Story 4.3: [FRONT] LogViewer Component + Live Log Page

Status: ready-for-dev

## Story

As a user watching a run execute, I want a live log view with ANSI color rendering and auto-scroll, So that I can monitor agent output in real time without refreshing.

## Acceptance Criteria (BDD)

**AC1: LogViewer renders ANSI-colored log lines with timestamps**
- **Given** log lines containing ANSI escape codes (e.g., `\u001b[32mOK\u001b[0m`)
- **When** LogViewer renders them
- **Then** ANSI codes are converted to styled `<span>` elements via `ansi-to-html`
- **And** each line is prefixed with a `HH:MM:SS` timestamp in a monospace dim style
- **And** raw ANSI codes are never visible as literal text

**AC2: Auto-scroll follows new log lines and pauses on user scroll-up**
- **Given** auto-scroll is enabled (default on mount)
- **When** new log lines arrive
- **Then** the log container scrolls to the bottom automatically
- **When** the user scrolls up more than 50px from the bottom
- **Then** auto-scroll pauses and the toolbar shows a "Resume" indicator
- **When** the user scrolls back to within 50px of the bottom
- **Then** auto-scroll resumes automatically

**AC3: Connection state indicator reflects SSE lifecycle**
- **Given** LogViewer mounts with a valid `projectId`
- **When** the SSE connection is being established
- **Then** the toolbar shows "Connecting..." with a spinner
- **When** connected and receiving events
- **Then** the toolbar shows a green "Live" badge
- **When** the SSE connection drops
- **Then** the toolbar shows "Disconnected" with a warning color and reconnect is attempted by `useSSE`

**AC4: useSSE composable manages the EventSource connection**
- **Given** `useSSE(projectId)` is called
- **When** the composable mounts
- **Then** it opens `EventSource` at `/api/v1/events/stream?project_id={projectId}`
- **And** on `message` or named events, it dispatches to a callback provided by the caller
- **And** on component unmount, it calls `eventSource.close()`
- **And** `status` ref reflects `'connecting' | 'open' | 'closed' | 'error'`

**AC5: RunDetailView shows run info, step timeline, and live LogViewer**
- **Given** I navigate to `/runs/{runId}`
- **When** the view loads
- **Then** it fetches run data via `GET /api/v1/runs/{runId}` and displays title, status badge, progress bar, and step list
- **And** LogViewer is mounted below the step timeline filtering `log.emitted` events by `runId`
- **And** a PrimeVue `ProgressBar` shows `run.progress` (0–100 integer from Story 4-2)

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Install `ansi-to-html` and create `formatLogLine` utility (AC: #1)
  - [ ] Run `npm install ansi-to-html` in `frontend/`
  - [ ] Create `frontend/src/utils/formatLogLine.ts` — exports `formatLogLine(raw: string, timestamp: Date): string`
  - [ ] Uses `new AnsiUp().ansi_to_html(raw)` (or `ansi-to-html` Convert class) for ANSI rendering
  - [ ] Prepends `HH:MM:SS` timestamp formatted from `timestamp` parameter
  - [ ] Write unit tests in `frontend/src/utils/__tests__/formatLogLine.spec.ts`

- [ ] [FRONT] Task 2: Create `useSSE` composable (AC: #4)
  - [ ] Create `frontend/src/composables/useSSE.ts`
  - [ ] Uses native `EventSource` API (not `@vueuse/core` `useEventSource` — need explicit named-event support)
  - [ ] Accepts `projectId: string` and `onEvent: (eventName: string, data: unknown) => void` callback
  - [ ] Exposes `status: Ref<'connecting' | 'open' | 'closed' | 'error'>`
  - [ ] Opens connection on call, calls `onEvent` for all message events, closes on `onBeforeUnmount`
  - [ ] Write unit tests in `frontend/src/composables/__tests__/useSSE.spec.ts` using mock `EventSource`

- [ ] [FRONT] Task 3: Build `LogViewer.vue` shared component (AC: #1, #2, #3)
  - [ ] Create `frontend/src/ui/composed/LogViewer.vue`
  - [ ] Props: `lines: LogLine[]` (type `{ text: string; timestamp: Date }`), `status: 'connecting' | 'open' | 'closed' | 'error'`
  - [ ] Renders lines as `v-html` from `formatLogLine(line.text, line.timestamp)` inside a `<pre>` or fixed-height `<div>` with `overflow-y: auto`
  - [ ] Toolbar row: connection state badge (PrimeVue Tag severity mapped from status), Clear button, auto-scroll indicator
  - [ ] Auto-scroll logic: use `useTemplateRef` on the scroll container; on new `lines`, if auto-scroll enabled, scroll to bottom via `scrollTop = scrollHeight`
  - [ ] Scroll listener: detect user scroll-up → pause; detect near-bottom → resume

- [ ] [FRONT] Task 4: Create `useRunLogs` composable (AC: #4, #5)
  - [ ] Create `frontend/src/features/runs/composables/useRunLogs.ts`
  - [ ] Accepts `projectId: string`, `runId: string`
  - [ ] Uses `useSSE(projectId, onEvent)` internally
  - [ ] Filters events: only processes events where `eventName === 'log.emitted'` and `payload.run_id === runId`
  - [ ] Appends to `lines: Ref<LogLine[]>` on matching events
  - [ ] Exposes `lines`, `sseStatus`, `clearLogs`

- [ ] [FRONT] Task 5: Create `useRunDetail` composable (AC: #5)
  - [ ] Create `frontend/src/features/runs/composables/useRunDetail.ts`
  - [ ] Accepts `runId: string`
  - [ ] Uses `useAsyncAction` to fetch `GET /api/v1/runs/{runId}` (generated type `RunWithSteps`)
  - [ ] Exposes `run`, `isLoading`, `error`, `fetchRun`, `retry`
  - [ ] Calls `fetchRun` on mount via `onMounted`
  - [ ] Write unit tests in `frontend/src/features/runs/__tests__/useRunDetail.spec.ts`

- [ ] [FRONT] Task 6: Implement `RunDetailView.vue` (AC: #5)
  - [ ] Replace placeholder content in `frontend/src/views/RunDetailView.vue`
  - [ ] Extract `runId` from `useRoute().params.id`
  - [ ] Use `useRunDetail(runId)` for run data; show `Skeleton` on load, `Message` on error
  - [ ] Render: run ID (monospace), status `Tag`, `ProgressBar :value="run.progress"`, step list via PrimeVue `Timeline`
  - [ ] Mount `LogViewer` below, wired to `useRunLogs(run.project_id, runId)` — only when `run` is loaded
  - [ ] Extract `projectId` from `run.project_id` (available after fetch, not from route)

- [ ] [FRONT] Task 7: Wire step status to PrimeVue Timeline in RunDetailView (AC: #5)
  - [ ] Map step `status` to Timeline marker severity: `completed → success`, `running → info`, `failed → danger`, `pending → secondary`, `cancelled → warn`
  - [ ] Each Timeline item shows: step name, action, status badge, duration (if `started_at` and `completed_at` available)
  - [ ] Use `date-fns` `differenceInSeconds` for duration formatting

- [ ] [FRONT] Task 8: Write unit tests for LogViewer.vue and useRunLogs (AC: #1, #2, #3, #4)
  - [ ] `frontend/src/ui/composed/__tests__/LogViewer.spec.ts`: renders lines with ANSI HTML; shows "Connecting..." when status is connecting; Clear button empties lines (via emit)
  - [ ] `frontend/src/features/runs/__tests__/useRunLogs.spec.ts`: mock `useSSE`; assert only `log.emitted` events with matching `run_id` are appended; assert other events are ignored; `clearLogs` resets lines

## Dev Notes

### Dependencies

- Story 4-1 (SSE endpoint) — required for live connection; `useSSE` targets `/api/v1/events/stream`
- Story 4-2 (Run progress field) — required for `ProgressBar`; `run.progress` is the integer 0–100
- `RunDetailView.vue` already exists as a placeholder at `frontend/src/views/RunDetailView.vue`
- Route `/runs/:id` → name `run-detail` already registered in `frontend/src/router/index.ts`
- `useAsyncAction` composable already at `frontend/src/composables/useAsyncAction.ts`

### Architecture Requirements

Component hierarchy:
```
RunDetailView.vue (route view, composes everything)
├── PrimeVue Skeleton / Message (loading / error)
├── PageHeader (run ID + status Tag)
├── ProgressBar :value="run.progress"
├── Timeline (steps, one item per RunStep)
│   └── PrimeVue Tag (step status severity)
└── LogViewer.vue (ui/composed)
    └── <div> (scroll container, v-html per line)
```

SSE event payload for `log.emitted` (expected shape from backend `log_streamer` adapter):
```json
{
  "run_id": "uuid",
  "step_id": "uuid",
  "line": "raw log text with possible ANSI codes",
  "timestamp": "2026-02-17T10:30:00Z"
}
```
`useRunLogs` must parse `data` as JSON and check `payload.run_id === props.runId`.

`LogViewer` is a pure display component — it receives `lines` as props and emits `clear`. Filtering logic lives in `useRunLogs`.

### File Paths (exact)

```
frontend/src/utils/formatLogLine.ts                               (new)
frontend/src/utils/__tests__/formatLogLine.spec.ts                (new)
frontend/src/composables/useSSE.ts                                (new)
frontend/src/composables/__tests__/useSSE.spec.ts                 (new)
frontend/src/ui/composed/LogViewer.vue                            (new)
frontend/src/ui/composed/__tests__/LogViewer.spec.ts              (new)
frontend/src/features/runs/composables/useRunLogs.ts              (new)
frontend/src/features/runs/composables/useRunDetail.ts            (new)
frontend/src/features/runs/__tests__/useRunLogs.spec.ts           (new)
frontend/src/features/runs/__tests__/useRunDetail.spec.ts         (new)
frontend/src/views/RunDetailView.vue                              (replace placeholder)
```

### Technical Specifications

**`LogLine` type:**
```typescript
export interface LogLine {
  text: string      // raw text, may contain ANSI codes
  timestamp: Date
}
```

**`formatLogLine.ts`:**
```typescript
import Convert from 'ansi-to-html'

const converter = new Convert({ escapeXML: true })

/** Converts a raw log line (possibly with ANSI codes) to HTML with HH:MM:SS prefix. */
export function formatLogLine(raw: string, timestamp: Date): string {
  const hh = String(timestamp.getHours()).padStart(2, '0')
  const mm = String(timestamp.getMinutes()).padStart(2, '0')
  const ss = String(timestamp.getSeconds()).padStart(2, '0')
  const timePrefix = `<span class="log-ts">${hh}:${mm}:${ss}</span> `
  return timePrefix + converter.toHtml(raw)
}
```

**`useSSE.ts`:**
```typescript
import { ref, onBeforeUnmount } from 'vue'

export type SSEStatus = 'connecting' | 'open' | 'closed' | 'error'

export function useSSE(
  projectId: string,
  onEvent: (eventName: string, data: unknown) => void
) {
  const status = ref<SSEStatus>('connecting')
  const es = new EventSource(`/api/v1/events/stream?project_id=${projectId}`)

  es.onopen = () => { status.value = 'open' }
  es.onerror = () => { status.value = 'error' }
  es.onmessage = (e) => {
    try { onEvent('message', JSON.parse(e.data)) } catch { /* ignore */ }
  }

  // Listen for named events (SSE `event:` field maps to EventSource event type)
  const knownEvents = ['run.started', 'run.completed', 'step.completed', 'step.failed', 'log.emitted', 'hitl.pending']
  for (const name of knownEvents) {
    es.addEventListener(name, (e) => {
      try { onEvent(name, JSON.parse((e as MessageEvent).data)) } catch { /* ignore */ }
    })
  }

  function close() {
    es.close()
    status.value = 'closed'
  }

  onBeforeUnmount(close)

  return { status, close }
}
```

**`LogViewer.vue` props/emits:**
```typescript
defineProps<{
  lines: LogLine[]
  status: SSEStatus
}>()

const emit = defineEmits<{
  clear: []
}>()
```

**Auto-scroll implementation in `LogViewer.vue`:**
```typescript
const scrollEl = useTemplateRef<HTMLElement>('scrollContainer')
const autoScroll = ref(true)

watch(() => props.lines.length, async () => {
  if (!autoScroll.value) return
  await nextTick()
  if (scrollEl.value) {
    scrollEl.value.scrollTop = scrollEl.value.scrollHeight
  }
})

function onScroll() {
  if (!scrollEl.value) return
  const { scrollTop, scrollHeight, clientHeight } = scrollEl.value
  const distFromBottom = scrollHeight - scrollTop - clientHeight
  autoScroll.value = distFromBottom < 50
}
```

**`useRunLogs.ts`:**
```typescript
import { ref } from 'vue'
import { useSSE, type SSEStatus } from '@/composables/useSSE'

export function useRunLogs(projectId: string, runId: string) {
  const lines = ref<LogLine[]>([])

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName !== 'log.emitted') return
    const payload = data as { run_id: string; line: string; timestamp: string }
    if (payload.run_id !== runId) return
    lines.value.push({ text: payload.line, timestamp: new Date(payload.timestamp) })
  })

  function clearLogs() {
    lines.value = []
  }

  return { lines, sseStatus, clearLogs }
}
```

**Connection state badge severity mapping:**
```typescript
const statusSeverity: Record<SSEStatus, 'info' | 'success' | 'warn' | 'danger'> = {
  connecting: 'info',
  open: 'success',
  closed: 'warn',
  error: 'danger',
}

const statusLabel: Record<SSEStatus, string> = {
  connecting: 'Connecting...',
  open: 'Live',
  closed: 'Disconnected',
  error: 'Error',
}
```

**Step status → Timeline severity mapping:**
```typescript
const stepSeverity: Record<string, string> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  cancelled: 'warn',
}
```

### Testing Requirements

**`formatLogLine.spec.ts`:**
- Input with ANSI green code → output contains `<span` with color style, no literal `\u001b`
- Timestamp `new Date('2026-01-01T10:05:03Z')` → prefix `10:05:03`
- Empty string input → returns timestamp prefix + empty converted string

**`useSSE.spec.ts`:**
- Mock `EventSource` class: capture `onopen`, `onerror`, `onmessage`, `addEventListener` calls
- Call `onopen` → `status` becomes `'open'`
- Call `onerror` → `status` becomes `'error'`
- Dispatch `MessageEvent` with JSON data → `onEvent` called with parsed object
- `onBeforeUnmount` triggers → `EventSource.close()` is called

**`LogViewer.spec.ts`:**
- Renders `lines.length` number of `<div>` elements (or similar container)
- Status `'connecting'` → Tag label "Connecting..."
- Status `'open'` → Tag label "Live"
- Clear button click → emits `clear`

**`useRunLogs.spec.ts`:**
- Mock `useSSE`; trigger `onEvent('log.emitted', { run_id: props.runId, line: 'hello', timestamp: '...' })` → `lines` has 1 item
- Trigger with wrong `run_id` → `lines` stays empty
- Trigger with `eventName !== 'log.emitted'` → `lines` stays empty
- `clearLogs()` → `lines` becomes `[]`

### References

- `frontend/src/composables/useAsyncAction.ts` — existing async action pattern
- `frontend/src/router/index.ts` — `run-detail` route at `/runs/:id`
- `frontend/src/views/RunDetailView.vue` — placeholder to replace
- `api/openapi.yaml` — `RunWithSteps` schema (includes `progress` after Story 4-2 lands)
- `ansi-to-html` npm: https://www.npmjs.com/package/ansi-to-html
- PrimeVue Timeline: https://primevue.org/timeline/
- PrimeVue ProgressBar: https://primevue.org/progressbar/
- PrimeVue Tag: https://primevue.org/tag/
- SSE named events via `addEventListener`: https://developer.mozilla.org/en-US/docs/Web/API/EventSource

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
