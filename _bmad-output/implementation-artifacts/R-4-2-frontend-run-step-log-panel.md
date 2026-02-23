# Story R-4-2: [FRONT] Slide-in log panel for run steps

Status: ready-for-dev

## Story

As a developer monitoring a pipeline run,
I want to click on a step and see its logs in a slide-in panel without leaving the pipeline view,
so that I can inspect step output while keeping the overall pipeline status visible.

## Acceptance Criteria (BDD)

**AC1: Panel slides in from the right**
- **Given** I am on the run detail page viewing the pipeline
- **When** I click on a step's job row
- **Then** a panel slides in from the right side of the screen
- **And** the panel reveals the step's logs using `LogViewer`
- **And** the pipeline view remains partially visible on the left (not fully hidden)

**AC2: Panel content — step metadata**
- **Given** the log panel is open for a step
- **When** I view the panel header
- **Then** I see: step name, current status (PrimeVue `Tag`), started_at timestamp, completed_at timestamp (or "Running..." if active), and duration

**AC3: Panel content — logs**
- **Given** the log panel is open for a step
- **When** the step has logs
- **Then** the `LogViewer` component renders all log lines with ANSI color support
- **And** new log lines stream in real-time via SSE if the step is currently running

**AC4: Panel content — error message**
- **Given** the log panel is open for a step with status `failed`
- **When** the step has an error message
- **Then** the error message is displayed in a PrimeVue `Message` component with severity `error` above the logs

**AC5: Panel close button**
- **Given** the log panel is open
- **When** I click the X button in the panel header
- **Then** the panel slides out to the right and closes
- **And** the pipeline view returns to full width

**AC6: Keyboard close**
- **Given** the log panel is open
- **When** I press Escape
- **Then** the panel closes

**AC7: Mobile — full width with backdrop**
- **Given** I am on a mobile-sized screen (< md breakpoint)
- **When** the log panel is open
- **Then** the panel takes up the full viewport width
- **And** a semi-transparent backdrop overlay covers the pipeline view behind it
- **And** clicking the backdrop closes the panel

**AC8: Panel width on desktop**
- **Given** I am on a desktop-sized screen (>= md breakpoint)
- **When** the log panel is open
- **Then** the panel width is approximately 50% of the viewport (`w-1/2` or `50vw`)
- **And** the panel is fixed-position on the right side of the screen

**AC9: SSE log streaming**
- **Given** the log panel is open for a running step
- **When** the step emits new `log.line` SSE events
- **Then** new log lines appear at the bottom of `LogViewer` in real-time
- **And** the log viewer auto-scrolls to the bottom for new lines

**AC10: Switching steps**
- **Given** the log panel is already open for step A
- **When** I click on step B in the pipeline view
- **Then** the panel content switches to step B's logs (no close/reopen animation)
- **And** the previous step's SSE log subscription is cleaned up

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `RunStepLogPanel.vue` component (AC: #1, #5, #6, #7, #8)
  - [ ] Place in `frontend/src/features/runs/RunStepLogPanel.vue`
  - [ ] Props: `stepId: string | null`, `runId: string`, `visible: boolean`
  - [ ] Emits: `close`
  - [ ] Use PrimeVue `Drawer` component with `position="right"` if available in PrimeVue 4 (check), otherwise implement with CSS transition
  - [ ] Desktop: `class="w-1/2"` on the Drawer/panel element
  - [ ] Mobile (`< md`): full width + backdrop overlay
  - [ ] Close button: PrimeVue `Button` icon-only with `icon="pi pi-times"` in panel header, `@click="emit('close')"`
  - [ ] Keyboard Escape: use `@vueuse/core` `onKeyStroke('Escape', () => emit('close'))` while panel is visible
  - [ ] Transition: CSS `transform: translateX(100%)` → `translateX(0)` with `transition-transform duration-300`

- [ ] [FRONT] Task 2: Panel header — step metadata (AC: #2)
  - [ ] Display `step.step_name` as heading (`h2` or `h3`)
  - [ ] Display status `Tag` (reuse STATUS_CONFIG from R-4-1 or a shared utility)
  - [ ] Display `started_at` formatted with `date-fns` `format(date, 'HH:mm:ss')`
  - [ ] Display `completed_at` formatted or "Running..." if step is active
  - [ ] Display duration using `formatDuration(step.started_at, step.completed_at)` from R-4-1's utility

- [ ] [FRONT] Task 3: Panel body — log display (AC: #3, #9)
  - [ ] Import and render `LogViewer` from `frontend/src/ui/composed/LogViewer.vue`
  - [ ] Pass log lines from `useRunLogs(runId, stepId)` composable to `LogViewer`
  - [ ] Ensure `useRunLogs` is called reactively on `stepId` change (watch `stepId` prop)
  - [ ] `LogViewer` should auto-scroll to bottom (`autoScroll` prop if supported)

- [ ] [FRONT] Task 4: Panel body — error message (AC: #4)
  - [ ] Add `v-if="step?.error_message"` block above `LogViewer`
  - [ ] Render PrimeVue `Message` with `severity="error"` and text = `step.error_message`

- [ ] [FRONT] Task 5: Handle step switching (AC: #10)
  - [ ] Watch `stepId` prop: when it changes, reset log state and re-subscribe to `useRunLogs` for the new step
  - [ ] The `useRunLogs` composable should clean up its SSE subscription on `stepId` change (verify existing composable behavior, fix if needed)

- [ ] [FRONT] Task 6: Mobile backdrop (AC: #7)
  - [ ] Add a `<div>` overlay element behind the panel: `class="fixed inset-0 bg-black/50 z-40 md:hidden"` visible only when panel is open on mobile
  - [ ] `@click` on backdrop emits `close`

- [ ] [FRONT] Task 7: Tests
  - [ ] Unit test `RunStepLogPanel.vue`: renders when `visible=true`, hidden when `visible=false`, close button emits `close`, Escape key emits `close`, error message shown when `step.error_message` set
  - [ ] Unit test step switching: `useRunLogs` called with new stepId when prop changes

- [ ] [FRONT] Task 8: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- No backend dependencies — reuses existing `useRunLogs` composable and `LogViewer` component
- **R-4-1** (RunPipelineView): the panel is opened in response to `step-selected` events emitted by R-4-1 components
- **R-4-3** (RunDetailView integration): wires this panel into the view

### Architecture Requirements

- `RunStepLogPanel.vue` is a feature-level component → `frontend/src/features/runs/`
- It does NOT manage its own visibility (open/close state lives in the parent `RunDetailView.vue`)
- It receives `stepId` as a prop and reacts to changes — no imperative open/close methods
- `LogViewer` from `ui/composed/` is reused as-is — no modification
- `useRunLogs` composable from `features/runs/composables/useRunLogs.ts` is reused — clean up if it does not support step switching

### Technical Specifications

#### PrimeVue Drawer usage (if available in PrimeVue 4)

```vue
<Drawer v-model:visible="visibleModel" position="right" :pt="{ root: 'w-full md:w-1/2' }">
  <template #header>
    <!-- step metadata -->
  </template>
  <!-- log panel body -->
</Drawer>
```

PrimeVue 4 `Drawer` (formerly `Sidebar`) supports `position`, `pt` passthrough for class overrides. Verify component name in PrimeVue 4 — may be `Sidebar` or `Drawer`.

If `Drawer`/`Sidebar` is not available or does not meet layout needs, implement manually:

```vue
<Teleport to="body">
  <!-- Backdrop (mobile only) -->
  <div
    v-if="visible"
    class="fixed inset-0 bg-black/50 z-40 md:hidden"
    @click="emit('close')"
  />
  <!-- Panel -->
  <Transition name="slide-right">
    <div
      v-if="visible"
      class="fixed top-0 right-0 h-full w-full md:w-1/2 z-50 bg-surface-0 shadow-xl flex flex-col"
    >
      <!-- header -->
      <!-- body -->
    </div>
  </Transition>
</Teleport>
```

#### Slide transition CSS

```css
/* In main.css or scoped style */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.3s ease;
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
}
```

#### useRunLogs composable interface (existing)

```typescript
// frontend/src/features/runs/composables/useRunLogs.ts
// Expected API — verify against actual implementation
export function useRunLogs(runId: Ref<string>, stepId: Ref<string | null>) {
  const lines = ref<string[]>([])
  // subscribes to SSE log.line events filtered by stepId
  // cleans up on stepId change or unmount
  return { lines }
}
```

If `useRunLogs` does not support reactive `stepId`, wrap it:

```typescript
// In RunStepLogPanel.vue
const stepIdRef = toRef(props, 'stepId')
const runIdRef = toRef(props, 'runId')
const { lines } = useRunLogs(runIdRef, stepIdRef)
```

#### Step metadata from RunStep type

```typescript
interface RunStep {
  id: string
  step_name: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'skipped' | 'waiting_approval'
  started_at?: string
  completed_at?: string
  error_message?: string
  // ... other fields
}
```

The parent must pass the full `RunStep` object via a prop (resolved in R-4-3 integration). Or, the panel can accept `stepId` and look up the step from `useRunsStore` directly. Decision: accept `step: RunStep | null` as a separate prop alongside `stepId` (avoids store coupling in the component).

Revised props:

```typescript
const props = defineProps<{
  step: RunStep | null    // full step object for metadata display
  runId: string
  visible: boolean
}>()
```

Derive `stepId` from `props.step?.id` internally.

### Testing Requirements

- Vitest unit tests for `RunStepLogPanel.vue`:
  - `visible=false` → panel not rendered or off-screen
  - `visible=true` → panel rendered
  - Close button click → emits `close`
  - Escape key while visible → emits `close`
  - `step.error_message` set → `Message` component rendered with error text
  - `step.error_message` absent → no `Message` component
  - `step` prop change → `useRunLogs` called with new step id (mock the composable)

### References

- `frontend/src/ui/composed/LogViewer.vue` — reused as-is for log rendering with ANSI support
- `frontend/src/features/runs/composables/useRunLogs.ts` — existing SSE log composable
- `frontend/src/features/runs/RunLogViewer.vue` — existing component (reference for log display pattern, will be deprecated in R-4-3)
- `frontend/CLAUDE.md` — PrimeVue component usage, Tailwind layout, no inline styles

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
