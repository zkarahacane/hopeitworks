# Story 4.4: [FRONT] Run Progress Timeline Display

Status: ready-for-dev

## Story

As a user monitoring a run, I want a visual timeline of pipeline steps, so that I can see at a glance which steps completed, which is running, and how long each took.

## Acceptance Criteria (BDD)

**AC1: Timeline renders all run steps with correct status icons**
- **Given** a run with steps in various statuses (`pending`, `running`, `completed`, `failed`, `pending_hitl`)
- **When** the `RunProgressTimeline` component renders
- **Then** each step appears as a PrimeVue `Timeline` item with the step name and a `Tag` reflecting the step status
- **And** `completed` → severity `success`, `running` → severity `info`, `failed` → severity `danger`, `pending` → severity `secondary`, `pending_hitl` → severity `warn`
- **And** steps are ordered by `step_order` ascending

**AC2: Completed step shows formatted duration**
- **Given** a step with both `started_at` and `completed_at` set
- **When** the timeline renders that step
- **Then** the duration is displayed as `Xm Ys` (e.g., `2m 34s`) using `date-fns` `differenceInSeconds`
- **And** durations under 60 seconds display as `Xs` only (e.g., `42s`)

**AC3: Running step shows live elapsed timer updating every second**
- **Given** a step with `status === 'running'` and a `started_at` timestamp
- **When** the timeline is visible
- **Then** the elapsed time since `started_at` updates every second (e.g., `1m 05s elapsed`)
- **And** the timer increments by one second in real time
- **And** the timer stops and the interval is cleared when the component is unmounted

**AC4: Steps update in real time via SSE `run.step.updated` events**
- **Given** the timeline is mounted with the SSE connection open from `useRunProgress`
- **When** a `run.step.updated` SSE event arrives with a matching `run_id`
- **Then** the corresponding step in the timeline updates its status and timestamps without a full page refresh
- **And** events with a non-matching `run_id` are ignored

**AC5: Initial data is loaded from the run API**
- **Given** I navigate to a run log page
- **When** `useRunProgress(projectId, runId)` mounts
- **Then** it calls `GET /api/v1/projects/{projectId}/runs/{runId}` via the generated `apiClient`
- **And** exposes `steps` as a reactive array of `RunStep` sorted by `step_order`
- **And** `isLoading` is `true` during the fetch and `false` after
- **And** `error` is populated if the API call fails

**AC6: Timeline integrates into `RunLogView.vue` above the LogViewer**
- **Given** the existing `RunLogView.vue` at `/projects/:projectId/runs/:runId/logs`
- **When** the view renders
- **Then** `RunProgressTimeline` appears above the `LogViewer` component
- **And** both components share the same `projectId` and `runId` from the route params
- **And** `RunProgressTimeline` receives `isLoading` and `error` as props and renders a `Skeleton` / `Message` accordingly

**AC7: `pending_hitl` step shows HITL badge and link to approval page**
- **Given** a step with `status === 'pending_hitl'`
- **When** the timeline renders that step
- **Then** an orange `Tag` with label `Awaiting Approval` is shown next to the step name
- **And** a `Button` link (severity `warn`, size `small`) routes to `/projects/:projectId/runs/:runId/approve/:stepId`

**AC8: Empty steps array shows a placeholder message**
- **Given** a run has been fetched but its `steps` array is empty
- **When** the timeline renders
- **Then** a `Message` component displays "No pipeline steps found for this run"
- **And** no `Timeline` component is rendered

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `useRunProgress` composable (AC: #4, #5)
  - [ ] Create `frontend/src/features/runs/composables/useRunProgress.ts`
  - [ ] Accepts `projectId: string`, `runId: string`
  - [ ] Fetches initial data via `apiClient.GET('/projects/{projectId}/runs/{runId}', ...)` on `onMounted`
  - [ ] Exposes `steps: Ref<RunStep[]>` (sorted by `step_order`), `isLoading: Ref<boolean>`, `error: Ref<Error | null>`
  - [ ] Calls `useSSE(projectId, onEvent)` internally; on `run.step.updated` event with matching `run_id`, patches the matching step in `steps` by its `id`
  - [ ] Add `'run.step.updated'` to the known events list in `useSSE.ts` if not already present
  - [ ] Write unit tests in `frontend/src/features/runs/__tests__/useRunProgress.spec.ts`

- [ ] [FRONT] Task 2: Create `useStepTimer` composable (AC: #3)
  - [ ] Create `frontend/src/features/runs/composables/useStepTimer.ts`
  - [ ] Accepts `startedAt: string | undefined`; returns `elapsed: ComputedRef<string>` (formatted as `Xm Ys` or `Xs`)
  - [ ] Uses `setInterval` at 1-second resolution, initialized on call, cleared via `onBeforeUnmount`
  - [ ] Returns empty string if `startedAt` is undefined
  - [ ] Write unit tests in `frontend/src/features/runs/__tests__/useStepTimer.spec.ts`: mock `Date.now`, assert formatting at 0s, 42s, 90s, 3600s

- [ ] [FRONT] Task 3: Create `formatStepDuration` utility (AC: #2)
  - [ ] Create `frontend/src/utils/formatStepDuration.ts`
  - [ ] Exports `formatStepDuration(startedAt: string, completedAt: string): string`
  - [ ] Uses `date-fns` `differenceInSeconds`; formats as `Xs` if < 60s, else `Xm Ys` (no leading zeros on seconds)
  - [ ] Write unit tests in `frontend/src/utils/__tests__/formatStepDuration.spec.ts`

- [ ] [FRONT] Task 4: Build `RunProgressTimeline.vue` component (AC: #1, #2, #3, #7, #8)
  - [ ] Create `frontend/src/features/runs/RunProgressTimeline.vue`
  - [ ] Props: `steps: RunStep[]`, `projectId: string`, `runId: string`, `isLoading: boolean`, `error: Error | null`
  - [ ] Shows PrimeVue `Skeleton` (height 120px) when `isLoading` is true
  - [ ] Shows PrimeVue `Message` severity `error` when `error` is non-null
  - [ ] Shows `Message` "No pipeline steps found for this run" when `steps` is empty and not loading
  - [ ] Renders PrimeVue `Timeline` with `#content` slot per step: step name, status `Tag`, duration (`formatStepDuration`) or live timer (`useStepTimer`) for running steps
  - [ ] For `pending_hitl` steps, renders `Tag` label "Awaiting Approval" severity `warn` and a `Button` `router-link` to the approval page
  - [ ] Status → severity mapping: `completed → success`, `running → info`, `failed → danger`, `pending → secondary`, `pending_hitl → warn`

- [ ] [FRONT] Task 5: Update `useSSE.ts` to handle `run.step.updated` named event (AC: #4)
  - [ ] Add `'run.step.updated'` to the `knownEvents` array in `frontend/src/composables/useSSE.ts`
  - [ ] Verify existing tests still pass after this addition

- [ ] [FRONT] Task 6: Integrate `RunProgressTimeline` into `RunLogView.vue` (AC: #6)
  - [ ] Open `frontend/src/views/RunLogView.vue`
  - [ ] Import and use `useRunProgress(projectId, runId)` (route params: `projectId` from `useRoute().params.projectId`, `runId` from `useRoute().params.runId`)
  - [ ] Mount `RunProgressTimeline` above the existing `LogViewer`, passing `steps`, `isLoading`, `error`, `projectId`, `runId`
  - [ ] Preserve the existing `LogViewer` wiring unchanged

- [ ] [FRONT] Task 7: Write unit tests for `RunProgressTimeline.vue` (AC: #1, #2, #7, #8)
  - [ ] Create `frontend/src/features/runs/__tests__/RunProgressTimeline.spec.ts`
  - [ ] Test: renders one Timeline item per step; `Skeleton` shown on `isLoading=true`; `Message` shown on error; empty steps shows placeholder message
  - [ ] Test: `completed` step shows formatted duration, not a live timer
  - [ ] Test: `pending_hitl` step renders "Awaiting Approval" tag and approval link button
  - [ ] Mock `useStepTimer` to return static string `'5s'` to avoid real timers in tests

- [ ] [FRONT] Task 8: Write unit tests for `useRunProgress` composable (AC: #4, #5)
  - [ ] File: `frontend/src/features/runs/__tests__/useRunProgress.spec.ts`
  - [ ] Mock `apiClient.GET` to return a run with 2 steps; assert `steps` is sorted by `step_order`
  - [ ] Mock `useSSE`; trigger `onEvent('run.step.updated', { run_id: runId, step: { id: step1.id, status: 'completed', completed_at: '...' } })`; assert step is patched in `steps`
  - [ ] Trigger event with wrong `run_id`; assert `steps` is unchanged
  - [ ] Trigger event with unrecognized `step.id`; assert `steps` is unchanged (no crash)

## Dev Notes

### Dependencies

- Story 4-3 (done): `useSSE` composable at `frontend/src/composables/useSSE.ts`, `LogViewer` at `frontend/src/ui/composed/LogViewer.vue`, `RunLogView.vue` at `frontend/src/views/RunLogView.vue`, route `/projects/:projectId/runs/:runId/logs`
- Story 4-2 (done): `GET /api/v1/projects/{projectId}/runs/{runId}` returns `RunWithSteps` with `steps[]` containing `started_at`, `completed_at`, `status`, `step_order`
- `RunStep` and `RunWithSteps` types already defined in `frontend/src/features/runs/composables/useRunDetail.ts` — import from there, do not redefine
- `useAsyncAction` at `frontend/src/composables/useAsyncAction.ts`
- `useSSE` at `frontend/src/composables/useSSE.ts` — already handles named events via `addEventListener`
- `date-fns` already installed (used in Story 4-3 for duration formatting in `RunDetailView`)

### Architecture Requirements

Component hierarchy after integration:

```
RunLogView.vue (route: /projects/:projectId/runs/:runId/logs)
├── RunProgressTimeline.vue
│   └── PrimeVue Timeline
│       └── per step: Tag (status), formatStepDuration / useStepTimer (elapsed), Button (HITL link)
└── RunLogViewer.vue (existing, unchanged)
    └── LogViewer.vue (existing, unchanged)
```

`useRunProgress` orchestrates data + SSE:
- Initial fetch on mount via `apiClient`
- SSE patch on `run.step.updated`
- Exposes `steps`, `isLoading`, `error` to `RunLogView.vue`

`RunProgressTimeline` is a pure display component:
- No API calls, no SSE — all reactive data comes from props
- Timer logic is local to the component via `useStepTimer` (called per running step)

SSE event payload expected for `run.step.updated`:
```json
{
  "run_id": "uuid",
  "step": {
    "id": "uuid",
    "step_name": "code-review",
    "step_order": 2,
    "action": "agent_run",
    "status": "completed",
    "started_at": "2026-02-17T10:30:00Z",
    "completed_at": "2026-02-17T10:32:34Z",
    "error_message": null
  }
}
```

The `useRunProgress` composable patches the step in-place: find by `step.id`, spread the new values. This is a shallow merge — do not replace the full array.

### File Paths (exact)

```
frontend/src/features/runs/composables/useRunProgress.ts             (new)
frontend/src/features/runs/composables/useStepTimer.ts               (new)
frontend/src/features/runs/RunProgressTimeline.vue                   (new)
frontend/src/features/runs/__tests__/useRunProgress.spec.ts          (new)
frontend/src/features/runs/__tests__/useStepTimer.spec.ts            (new)
frontend/src/features/runs/__tests__/RunProgressTimeline.spec.ts     (new)
frontend/src/utils/formatStepDuration.ts                             (new)
frontend/src/utils/__tests__/formatStepDuration.spec.ts              (new)
frontend/src/composables/useSSE.ts                                   (modified — add 'run.step.updated')
frontend/src/views/RunLogView.vue                                     (modified — integrate timeline)
```

### Technical Specifications

**`useRunProgress.ts`:**
```typescript
import { ref, onMounted } from 'vue'
import { apiClient } from '@/api/client'
import { useSSE } from '@/composables/useSSE'
import type { RunStep } from '@/features/runs/composables/useRunDetail'

interface RunStepUpdatedPayload {
  run_id: string
  step: RunStep
}

export function useRunProgress(projectId: string, runId: string) {
  const steps = ref<RunStep[]>([])
  const isLoading = ref(false)
  const error = ref<Error | null>(null)

  async function fetchRun() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/runs/{runId}' as never,
        { params: { path: { projectId, runId } } } as never,
      )
      if (apiError) throw new Error('Failed to load run steps')
      const run = data as { steps: RunStep[] }
      steps.value = [...(run.steps ?? [])].sort((a, b) => a.step_order - b.step_order)
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
    } finally {
      isLoading.value = false
    }
  }

  useSSE(projectId, (eventName, data) => {
    if (eventName !== 'run.step.updated') return
    const payload = data as RunStepUpdatedPayload
    if (payload.run_id !== runId) return
    const idx = steps.value.findIndex((s) => s.id === payload.step.id)
    if (idx === -1) return
    steps.value[idx] = { ...steps.value[idx], ...payload.step }
  })

  onMounted(fetchRun)

  return { steps, isLoading, error }
}
```

**`useStepTimer.ts`:**
```typescript
import { ref, computed, onBeforeUnmount } from 'vue'
import { formatStepDuration } from '@/utils/formatStepDuration'

export function useStepTimer(startedAt: string | undefined) {
  const now = ref(Date.now())

  if (!startedAt) {
    return { elapsed: computed(() => '') }
  }

  const intervalId = setInterval(() => {
    now.value = Date.now()
  }, 1000)

  onBeforeUnmount(() => clearInterval(intervalId))

  const elapsed = computed(() => {
    const startMs = new Date(startedAt).getTime()
    const totalSeconds = Math.floor((now.value - startMs) / 1000)
    if (totalSeconds < 60) return `${totalSeconds}s elapsed`
    const m = Math.floor(totalSeconds / 60)
    const s = totalSeconds % 60
    return `${m}m ${s}s elapsed`
  })

  return { elapsed }
}
```

**`formatStepDuration.ts`:**
```typescript
import { differenceInSeconds } from 'date-fns'

/** Formats the duration between two ISO timestamps as 'Xs' or 'Xm Ys'. */
export function formatStepDuration(startedAt: string, completedAt: string): string {
  const total = differenceInSeconds(new Date(completedAt), new Date(startedAt))
  if (total < 60) return `${total}s`
  const m = Math.floor(total / 60)
  const s = total % 60
  return `${m}m ${s}s`
}
```

**`RunProgressTimeline.vue` props/template sketch:**
```typescript
defineProps<{
  steps: RunStep[]
  projectId: string
  runId: string
  isLoading: boolean
  error: Error | null
}>()
```

Status → Tag severity mapping:
```typescript
const stepSeverity: Record<string, 'success' | 'info' | 'danger' | 'secondary' | 'warn'> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  pending_hitl: 'warn',
  cancelled: 'secondary',
}
```

PrimeVue `Timeline` usage:
```vue
<Timeline :value="steps" layout="vertical">
  <template #content="{ item: step }">
    <!-- step name, Tag, duration or timer, HITL link -->
  </template>
  <template #marker="{ item: step }">
    <ProgressSpinner v-if="step.status === 'running'" style="width: 1.5rem; height: 1.5rem" />
    <span v-else class="pi" :class="markerIcon(step.status)" />
  </template>
</Timeline>
```

Marker icon mapping:
```typescript
const markerIcon: Record<string, string> = {
  completed: 'pi-check-circle',
  running: '', // replaced by ProgressSpinner
  failed: 'pi-times-circle',
  pending: 'pi-circle',
  pending_hitl: 'pi-hourglass',
  cancelled: 'pi-ban',
}
```

**`RunLogView.vue` integration (additions only):**
```vue
<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useRunProgress } from '@/features/runs/composables/useRunProgress'
import RunProgressTimeline from '@/features/runs/RunProgressTimeline.vue'

const route = useRoute()
const projectId = route.params.projectId as string
const runId = route.params.runId as string

const { steps, isLoading, error } = useRunProgress(projectId, runId)
</script>

<template>
  <div class="flex flex-col gap-6 p-4">
    <RunProgressTimeline
      :steps="steps"
      :project-id="projectId"
      :run-id="runId"
      :is-loading="isLoading"
      :error="error"
    />
    <!-- existing RunLogViewer below, unchanged -->
  </div>
</template>
```

### Testing Requirements

**`formatStepDuration.spec.ts`:**
- `('2026-01-01T10:00:00Z', '2026-01-01T10:00:42Z')` → `'42s'`
- `('2026-01-01T10:00:00Z', '2026-01-01T10:02:34Z')` → `'2m 34s'`
- `('2026-01-01T10:00:00Z', '2026-01-01T10:01:00Z')` → `'1m 0s'`
- `('2026-01-01T10:00:00Z', '2026-01-01T10:00:00Z')` → `'0s'`

**`useStepTimer.spec.ts`:**
- Mock `Date.now` to return a fixed start; advance by 42000ms → `elapsed` is `'42s elapsed'`
- Advance by 90000ms → `elapsed` is `'1m 30s elapsed'`
- `startedAt = undefined` → `elapsed` is `''`
- Unmount → `clearInterval` called

**`useRunProgress.spec.ts`:**
- Mock `apiClient.GET` with steps in wrong `step_order` order; assert `steps` is sorted ascending after fetch
- Mock `useSSE`; fire `run.step.updated` with matching `run_id`; assert step is patched by id
- Fire `run.step.updated` with non-matching `run_id`; assert `steps` unchanged
- Fire `run.step.updated` with unknown step `id`; assert no error, `steps` unchanged
- API call throws; assert `error` is set and `isLoading` is `false`

**`RunProgressTimeline.spec.ts`:**
- Mount with `isLoading=true` → `Skeleton` rendered, `Timeline` not rendered
- Mount with `error` set → `Message` rendered with error text
- Mount with empty `steps` → placeholder `Message` rendered
- Mount with 3 steps → 3 Timeline items rendered
- Step with `status='pending_hitl'` → "Awaiting Approval" tag present and approval button href contains `/approve/`
- Step with `status='completed'` and both timestamps → duration string rendered (mock `formatStepDuration`)
- Step with `status='running'` → `ProgressSpinner` in marker slot, `elapsed` string rendered (mock `useStepTimer`)

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
