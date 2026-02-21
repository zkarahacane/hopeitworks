# Story 7.5: [FRONT] Epic Run Monitoring Dashboard

Status: ready-for-dev

## Story

As a user who launched an epic batch run, I want a real-time dashboard showing each story's execution status in the DAG, So that I can monitor progress and quickly navigate to logs of any running or failed story.

## Acceptance Criteria (BDD)

**AC1: Route /projects/:id/epic-runs/:epicRunId renders EpicRunView**
- **Given** a user has launched an epic run and been redirected
- **When** they navigate to `/projects/:id/epic-runs/:epicRunId`
- **Then** `EpicRunView.vue` is rendered with the epic run monitoring dashboard
- **And** the view fetches the epic run via `GET /api/v1/projects/{projectId}/epic-runs/{epicRunId}`

**AC2: DAG graph shows colored nodes based on live story status**
- **Given** the epic run is loaded and the DAG graph renders
- **When** a story has status `pending`
- **Then** its node is styled with grey background (Tailwind `bg-surface-200`)
- **When** a story has status `running`
- **Then** its node is styled with blue background and a CSS pulse animation class
- **When** a story has status `completed`
- **Then** its node is styled with green background
- **When** a story has status `failed`
- **Then** its node is styled with red background
- **And** the DAGVisualization component from Story 7-4 is reused, receiving nodes/edges derived from the epic run response

**AC3: Overall progress bar reflects completed stories**
- **Given** the epic run response includes a list of story statuses
- **When** the dashboard renders
- **Then** a PrimeVue `ProgressBar` displays `(completedCount / totalCount) * 100`
- **And** a label shows `"N / M stories completed"` (e.g., `"2 / 5 stories completed"`)
- **And** the progress bar updates in real time as SSE events arrive

**AC4: Execution group layers are shown**
- **Given** stories have a `group_index` field indicating which parallel execution layer they belong to
- **When** the dashboard renders
- **Then** a section titled "Execution Layers" lists groups in order
- **And** each group shows: layer number, count of stories, and a PrimeVue `Tag` indicating the group's aggregate status (`pending`, `running`, `completed`, `failed`)
- **And** the currently running group is visually highlighted with `severity="info"`

**AC5: Clicking a story node opens the log viewer for that story's run**
- **Given** a story in the DAG has a `run_id` available
- **When** the user clicks the story node
- **Then** the router navigates to `/runs/:runId` (route `run-detail`) for that story's run
- **And** if the story has no `run_id` (status is `pending`), the click is ignored (node is not interactive)

**AC6: Real-time updates via SSE events update the store and graph**
- **Given** the dashboard is mounted and SSE is connected via `useSSE`
- **When** an `epic_run.story.completed` event arrives
- **Then** `useEpicRunStore` updates the matching story's status to `completed`
- **And** the DAG node color updates reactively without a page reload
- **When** an `epic_run.completed` or `epic_run.failed` event arrives
- **Then** the overall epic run status in the store is updated
- **And** SSE known events list in `useSSE` includes: `epic_run.started`, `epic_run.group.started`, `epic_run.story.completed`, `epic_run.completed`, `epic_run.failed`

**AC7: Completion summary panel shown when epic run finishes**
- **Given** the epic run status becomes `completed` or `failed`
- **When** the status is detected (from initial fetch or SSE event)
- **Then** a summary panel appears below the progress bar
- **If** status is `completed`: PrimeVue `Message` with severity `"success"` and text `"All stories completed successfully"`
- **If** status is `failed`: PrimeVue `Message` with severity `"error"` listing failed story keys as clickable links navigating to their run-detail page
- **And** while the run is still in progress, the summary panel is not shown

**AC8: Redirect from EpicDagView after launch**
- **Given** the user clicks "Launch Epic" on EpicDagView and confirms
- **When** the `POST /api/v1/projects/{projectId}/epics/{epicId}/runs` call returns 202 with `{ epic_run_id }`
- **Then** the router pushes to `{ name: 'epic-run-monitor', params: { id: projectId, epicRunId: result.epic_run_id } }` instead of showing only a Toast
- **And** the existing success Toast is replaced by the navigation (no Toast needed — user lands on the dashboard)

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Regenerate API types to pick up epic run endpoints (AC: #1, #6)
  - [ ] Run `cd frontend && npm run generate-api` — requires Story 7-2 backend to be merged so `GET /projects/{projectId}/epic-runs/{epicRunId}` is in `api/openapi.yaml`
  - [ ] Verify generated schema includes `EpicRun`, `EpicRunStory` types with fields: `id`, `epic_id`, `status`, `stories[]` (`story_id`, `story_key`, `run_id`, `group_index`, `status`), `created_at`, `completed_at`
  - [ ] Update `useSSE.ts` known events list to include: `epic_run.started`, `epic_run.group.started`, `epic_run.story.completed`, `epic_run.completed`, `epic_run.failed`

- [ ] [FRONT] Task 2: Create `useEpicRunStore` Pinia store (AC: #3, #4, #6, #7)
  - [ ] File: `frontend/src/stores/epicRun.ts`
  - [ ] State: `epicRun: Ref<EpicRun | null>`, `isLoading: Ref<boolean>`, `error: Ref<string | null>`
  - [ ] Action `fetchEpicRun(projectId: string, epicRunId: string)`: calls `apiClient.GET('/projects/{projectId}/epic-runs/{epicRunId}', ...)`, sets `epicRun`, handles error
  - [ ] Action `handleSSEEvent(eventName: string, data: unknown)`: handles `epic_run.story.completed` by updating the matching story's status in `epicRun.value.stories`; handles `epic_run.completed` and `epic_run.failed` by updating `epicRun.value.status`
  - [ ] Getters: `completedCount` (computed: stories with status `completed`), `totalCount` (computed: `stories.length`), `progressPercent` (computed: `(completedCount / totalCount) * 100`), `failedStories` (computed: stories where `status === 'failed'`)
  - [ ] Action `reset()`: sets all state to initial values
  - [ ] Write unit tests: `frontend/src/stores/__tests__/epicRun.spec.ts`
    - [ ] Test `handleSSEEvent('epic_run.story.completed', { story_id })` updates matching story status
    - [ ] Test `progressPercent` computed value after partial completions
    - [ ] Test `failedStories` computed returns only failed ones

- [ ] [FRONT] Task 3: Create `EpicRunStatusNode.vue` custom DAG node component (AC: #2, #5)
  - [ ] File: `frontend/src/features/epics/EpicRunStatusNode.vue`
  - [ ] Props: `data: { key: string; title: string; status: 'pending' | 'running' | 'completed' | 'failed'; runId: string | null }`
  - [ ] Status → CSS class map: `pending` → `bg-surface-200`, `running` → `bg-blue-500 animate-pulse`, `completed` → `bg-green-500`, `failed` → `bg-red-500`
  - [ ] Renders: story key (bold monospace), title (truncated 40 chars), PrimeVue `Tag` for status
  - [ ] Emits `node-click: [runId: string]` when clicked — parent handles navigation; does NOT emit when `runId` is null
  - [ ] Uses `Handle` from `@vue-flow/core` for source/target connection points

- [ ] [FRONT] Task 4: Create `useEpicRunMonitor` composable (AC: #1, #2, #6)
  - [ ] File: `frontend/src/features/epics/composables/useEpicRunMonitor.ts`
  - [ ] Accepts `projectId: string`, `epicRunId: string`
  - [ ] Uses `useEpicRunStore()` — calls `fetchEpicRun` on `onMounted`, calls `reset` on `onBeforeUnmount`
  - [ ] Uses `useSSE(projectId, onEvent)` — dispatches events to `epicRunStore.handleSSEEvent(eventName, data)`
  - [ ] Derives `nodes: ComputedRef<Node[]>` from `epicRunStore.epicRun.stories` using `@vue-flow/core` Node format: `{ id: story.story_key, type: 'epicRunStatus', position: { x: story.group_index * 250, y: posWithinGroup * 120 }, data: { key: story.story_key, title: story.story_key, status: story.status, runId: story.run_id ?? null } }`
  - [ ] Derives `edges: ComputedRef<Edge[]>` — edges are carried from the initial epic run response if the API includes them, otherwise empty
  - [ ] Exposes: `epicRunStore`, `nodes`, `edges`, `sseStatus`
  - [ ] Write unit tests: `frontend/src/features/epics/__tests__/useEpicRunMonitor.spec.ts`
    - [ ] Test: `nodes` computed produces correct VueFlow Node shape from store stories
    - [ ] Test: SSE event dispatched to store correctly (mock `useSSE`)

- [ ] [FRONT] Task 5: Create `EpicRunGroupList.vue` component (AC: #4)
  - [ ] File: `frontend/src/features/epics/EpicRunGroupList.vue`
  - [ ] Props: `stories: EpicRunStory[]`
  - [ ] Groups stories by `group_index` using a computed property
  - [ ] Renders a vertical list of groups; each group shows: layer label (`"Layer N"`), story count, and a PrimeVue `Tag` for aggregate group status
  - [ ] Group status logic: if any story is `failed` → `danger`; else if any is `running` → `info` (highlighted); else if all `completed` → `success`; else `secondary`
  - [ ] No business logic beyond grouping — uses props only

- [ ] [FRONT] Task 6: Create `EpicRunView.vue` page view (AC: #1, #2, #3, #4, #5, #6, #7)
  - [ ] File: `frontend/src/views/EpicRunView.vue`
  - [ ] Reads `projectId` from `route.params.id as string`, `epicRunId` from `route.params.epicRunId as string`
  - [ ] Uses `useEpicRunMonitor(projectId, epicRunId)`
  - [ ] Layout:
    - Header row: back button (navigates to `epic-dag` if epicId is available, else `project-board`), `h1` "Epic Run Monitor", epic run ID (monospace, truncated)
    - `Skeleton` while `epicRunStore.isLoading`; `Message` + retry `Button` on `epicRunStore.error`
    - PrimeVue `ProgressBar` with `:value="epicRunStore.progressPercent"` and label `"N / M stories completed"`
    - Completion summary panel: conditional `Message` (success or error) when `epicRunStore.epicRun?.status` is `completed` or `failed`; failed stories listed as `RouterLink` to `run-detail`
    - `EpicRunGroupList` component passed `:stories="epicRunStore.epicRun?.stories ?? []"`
    - DAG graph area: `<VueFlow>` with custom node type `epicRunStatus` mapped to `EpicRunStatusNode`; handles `node-click` event → `router.push({ name: 'run-detail', params: { id: runId } })`
    - SSE status `Tag` in header (severity mapped from `sseStatus`)
  - [ ] Imports: `VueFlow`, `Controls`, `MiniMap` from `@vue-flow/core`; `ProgressBar`, `Message`, `Button`, `Tag` from `primevue`

- [ ] [FRONT] Task 7: Register route and update EpicDagView redirect (AC: #1, #8)
  - [ ] In `frontend/src/router/index.ts`: add child route under `/projects/:id` children array:
    ```typescript
    {
      path: 'epic-runs/:epicRunId',
      name: 'epic-run-monitor',
      component: () => import('@/views/EpicRunView.vue'),
    }
    ```
  - [ ] In `frontend/src/features/dag/composables/useEpicLauncher.ts`: after successful `launch()`, expose `epicRunId: computed(() => result.value?.epic_run_id ?? null)` so the view can redirect
  - [ ] In `frontend/src/views/EpicDagView.vue`: in the `accept` callback of `confirm.require`, replace the success Toast with `router.push({ name: 'epic-run-monitor', params: { id: projectId, epicRunId: result.value.epic_run_id } })`

- [ ] [FRONT] Task 8: Write unit tests for EpicRunGroupList.vue (AC: #4)
  - [ ] File: `frontend/src/features/epics/__tests__/EpicRunGroupList.spec.ts`
  - [ ] Test: stories grouped by `group_index` — 3 stories in 2 groups renders 2 group rows
  - [ ] Test: group with a `running` story gets `severity="info"` Tag
  - [ ] Test: group with all `completed` stories gets `severity="success"` Tag
  - [ ] Test: group with a `failed` story gets `severity="danger"` Tag

- [ ] [FRONT] Task 9: Run lint and type-check (AC: all)
  - [ ] `cd frontend && npm run lint` — must pass with zero errors
  - [ ] `cd frontend && npm run type-check` — tsc must pass (no type errors on `@vue-flow/core` Node type, `EpicRun`, `EpicRunStory`)
  - [ ] Fix any `vue/no-unused-vars` or strict TypeScript violations

## Dev Notes

### Dependencies

- **Story 7-2** (backend): `GET /api/v1/projects/{projectId}/epic-runs/{epicRunId}` and SSE events (`epic_run.*`) — must be merged before `npm run generate-api` picks up the types. This is the hard blocker.
- **Story 7-4** (frontend, done): `DagGraph.vue`, `DagStoryNode.vue`, `useDagLayout.ts`, `useEpicLauncher.ts`, `EpicDagView.vue` — all in place. Story 7-5 does NOT modify these files except `useEpicLauncher.ts` (redirect) and `EpicDagView.vue` (replace Toast with router push).
- **Story 4-3** (frontend, done): `useSSE` composable at `frontend/src/composables/useSSE.ts` — reused directly. The known events array in `useSSE.ts` must be extended with `epic_run.*` events.
- **Story 4-3** (frontend, done): `LogViewer.vue` at `frontend/src/ui/composed/LogViewer.vue` — NOT reused directly in this story; the story node click navigates to the existing `RunDetailView` which already has the LogViewer.

### Architecture Requirements

Feature directory structure for epics (additions only):

```
frontend/src/features/epics/
├── EpicRunStatusNode.vue                (new — custom VueFlow node for run monitoring)
├── EpicRunGroupList.vue                 (new — execution layer list)
├── composables/
│   └── useEpicRunMonitor.ts             (new — fetch + SSE wiring for epic run)
└── __tests__/
    ├── EpicRunStatusNode.spec.ts        (new — optional, if complex logic)
    ├── EpicRunGroupList.spec.ts         (new)
    └── useEpicRunMonitor.spec.ts        (new)
```

Note: There is currently no `frontend/src/features/epics/` directory — the existing epic features live in `frontend/src/features/board/` and views in `frontend/src/views/`. Create the `features/epics/` directory as the home for this feature's components.

Component hierarchy for EpicRunView:

```
EpicRunView.vue (new view, route: epic-run-monitor)
├── Header row
│   ├── Button (back)
│   ├── h1 "Epic Run Monitor"
│   ├── span (epicRunId, monospace)
│   └── Tag (sseStatus badge)
├── Skeleton (isLoading)
├── Message + Button "Retry" (error)
├── ProgressBar :value="progressPercent"
│   └── label "N / M stories completed"
├── Message (completion summary — conditional on status completed/failed)
│   └── RouterLink[] (failed story run links)
├── EpicRunGroupList :stories="epicRun.stories"
└── VueFlow (@vue-flow/core)
    ├── Controls
    ├── MiniMap
    └── EpicRunStatusNode (custom node type "epicRunStatus")
        ├── Handle (target — top)
        ├── story key (monospace bold)
        ├── title (truncated 40 chars)
        ├── PrimeVue Tag (status)
        └── Handle (source — bottom)
```

Data flow:

```
EpicRunView
  → useEpicRunMonitor(projectId, epicRunId)
      → useEpicRunStore.fetchEpicRun() on mount
          → GET /projects/{projectId}/epic-runs/{epicRunId}
      → useSSE(projectId, onEvent)
          → onEvent dispatches to useEpicRunStore.handleSSEEvent()
      → nodes (computed from store.epicRun.stories)
      → edges (from API response or empty)
  → VueFlow renders EpicRunStatusNode per story
      → node-click → router.push({ name: 'run-detail', params: { id: runId } })
  → EpicRunGroupList renders layer summary
  → ProgressBar :value="store.progressPercent"
  → completion Message when store.epicRun.status in ['completed', 'failed']
```

### File Paths (exact)

```
frontend/src/stores/epicRun.ts                                         (new)
frontend/src/stores/__tests__/epicRun.spec.ts                          (new)
frontend/src/features/epics/EpicRunStatusNode.vue                      (new)
frontend/src/features/epics/EpicRunGroupList.vue                       (new)
frontend/src/features/epics/composables/useEpicRunMonitor.ts           (new)
frontend/src/features/epics/__tests__/EpicRunGroupList.spec.ts         (new)
frontend/src/features/epics/__tests__/useEpicRunMonitor.spec.ts        (new)
frontend/src/views/EpicRunView.vue                                     (new)
frontend/src/router/index.ts                                           (add epic-run-monitor route)
frontend/src/composables/useSSE.ts                                     (add epic_run.* known events)
frontend/src/features/dag/composables/useEpicLauncher.ts               (add epicRunId computed)
frontend/src/views/EpicDagView.vue                                     (replace Toast with router.push on launch success)
```

### Technical Specifications

**`useEpicRunStore` (stores/epicRun.ts):**
```typescript
import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

export type EpicRun = components['schemas']['EpicRun']
export type EpicRunStory = components['schemas']['EpicRunStory']

export const useEpicRunStore = defineStore('epicRun', () => {
  const epicRun = ref<EpicRun | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const completedCount = computed(
    () => epicRun.value?.stories.filter((s) => s.status === 'completed').length ?? 0,
  )
  const totalCount = computed(() => epicRun.value?.stories.length ?? 0)
  const progressPercent = computed(() =>
    totalCount.value > 0 ? Math.round((completedCount.value / totalCount.value) * 100) : 0,
  )
  const failedStories = computed(
    () => epicRun.value?.stories.filter((s) => s.status === 'failed') ?? [],
  )

  async function fetchEpicRun(projectId: string, epicRunId: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiErr } = await apiClient.GET(
        '/projects/{projectId}/epic-runs/{epicRunId}',
        { params: { path: { projectId, epicRunId } } },
      )
      if (apiErr) { error.value = 'Failed to load epic run'; return }
      epicRun.value = data ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load epic run'
    } finally {
      isLoading.value = false
    }
  }

  function handleSSEEvent(eventName: string, data: unknown) {
    if (!epicRun.value) return
    const payload = data as { story_id?: string; status?: string }
    if (eventName === 'epic_run.story.completed' && payload.story_id) {
      const story = epicRun.value.stories.find((s) => s.story_id === payload.story_id)
      if (story) story.status = 'completed'
    }
    if (eventName === 'epic_run.failed') epicRun.value.status = 'failed'
    if (eventName === 'epic_run.completed') epicRun.value.status = 'completed'
  }

  function reset() {
    epicRun.value = null
    isLoading.value = false
    error.value = null
  }

  return {
    epicRun, isLoading, error,
    completedCount, totalCount, progressPercent, failedStories,
    fetchEpicRun, handleSSEEvent, reset,
  }
})
```

**`useEpicRunMonitor.ts`:**
```typescript
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { Node, Edge } from '@vue-flow/core'
import { useSSE } from '@/composables/useSSE'
import { useEpicRunStore } from '@/stores/epicRun'

export function useEpicRunMonitor(projectId: string, epicRunId: string) {
  const epicRunStore = useEpicRunStore()

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName.startsWith('epic_run.')) {
      epicRunStore.handleSSEEvent(eventName, data)
    }
  })

  const nodes = computed<Node[]>(() => {
    const stories = epicRunStore.epicRun?.stories ?? []
    const groupCounters = new Map<number, number>()
    return stories.map((s) => {
      const pos = groupCounters.get(s.group_index) ?? 0
      groupCounters.set(s.group_index, pos + 1)
      return {
        id: s.story_key,
        type: 'epicRunStatus',
        position: { x: s.group_index * 250, y: pos * 120 },
        data: {
          key: s.story_key,
          title: s.story_key,
          status: s.status,
          runId: s.run_id ?? null,
        },
      }
    })
  })

  const edges = computed<Edge[]>(() => [])  // populated if API response includes edges

  onMounted(() => epicRunStore.fetchEpicRun(projectId, epicRunId))
  onBeforeUnmount(() => epicRunStore.reset())

  return { epicRunStore, nodes, edges, sseStatus }
}
```

**`EpicRunStatusNode.vue`:**
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Tag from 'primevue/tag'

const props = defineProps<{
  data: { key: string; title: string; status: string; runId: string | null }
}>()

const emit = defineEmits<{ 'node-click': [runId: string] }>()

const truncatedTitle = computed(() =>
  props.data.title.length > 40 ? props.data.title.slice(0, 40) + '…' : props.data.title
)

const nodeClass = computed(() => ({
  'bg-surface-200': props.data.status === 'pending',
  'bg-blue-500 animate-pulse': props.data.status === 'running',
  'bg-green-500': props.data.status === 'completed',
  'bg-red-500': props.data.status === 'failed',
}))

const statusSeverity = computed(() => {
  const map: Record<string, string> = {
    pending: 'secondary',
    running: 'info',
    completed: 'success',
    failed: 'danger',
  }
  return map[props.data.status] ?? 'secondary'
})

function handleClick() {
  if (props.data.runId) emit('node-click', props.data.runId)
}
</script>

<template>
  <div
    :class="['epic-run-node rounded p-2 cursor-pointer', nodeClass]"
    @click="handleClick"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="flex flex-col gap-1">
      <span class="font-mono font-bold text-sm">{{ data.key }}</span>
      <span :title="data.title" class="text-xs">{{ truncatedTitle }}</span>
      <Tag :value="data.status" :severity="statusSeverity" class="text-xs" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>
```

**`EpicDagView.vue` — modified accept callback (Task 7):**
```typescript
// Replace the existing accept callback in handleLaunchClick:
accept: async () => {
  await launch()
  if (result.value?.epic_run_id) {
    router.push({
      name: 'epic-run-monitor',
      params: { id: projectId, epicRunId: result.value.epic_run_id },
    })
  } else {
    toast.add({ severity: 'error', summary: 'Launch failed',
      detail: launchError.value?.message ?? 'Unexpected error', life: 5000 })
  }
},
```

**`useSSE.ts` — add epic_run events to knownEvents array:**
```typescript
// In frontend/src/composables/useSSE.ts, extend the knownEvents array:
const knownEvents = [
  'run.started', 'run.completed', 'step.completed', 'step.failed',
  'log.emitted', 'hitl.pending',
  'epic_run.started', 'epic_run.group.started',
  'epic_run.story.completed', 'epic_run.completed', 'epic_run.failed',
]
```

**Router addition (router/index.ts) — insert inside `/projects/:id` children, after `epic-dag`:**
```typescript
{
  path: 'epic-runs/:epicRunId',
  name: 'epic-run-monitor',
  component: () => import('@/views/EpicRunView.vue'),
},
```

**`EpicRunGroupList.vue` group status logic:**
```typescript
function groupStatus(stories: EpicRunStory[]): string {
  if (stories.some((s) => s.status === 'failed')) return 'danger'
  if (stories.some((s) => s.status === 'running')) return 'info'
  if (stories.every((s) => s.status === 'completed')) return 'success'
  return 'secondary'
}
```

### Testing Requirements

**`epicRun.spec.ts`:**
- Mock `apiClient.GET` returning a valid `EpicRun` with 3 stories (1 completed, 1 running, 1 pending)
- Assert `completedCount.value === 1`, `totalCount.value === 3`, `progressPercent.value === 33`
- Call `handleSSEEvent('epic_run.story.completed', { story_id: runningStory.story_id })` → assert that story's status is now `'completed'`
- Call `handleSSEEvent('epic_run.failed', {})` → assert `epicRun.value.status === 'failed'`
- Assert `failedStories` computed returns only stories with `status === 'failed'`
- Assert `reset()` sets `epicRun.value` to null

**`useEpicRunMonitor.spec.ts`:**
- Mock `useEpicRunStore` and `useSSE`
- Assert `fetchEpicRun` is called on `onMounted`
- Assert `reset` is called on `onBeforeUnmount`
- Trigger the `onEvent` callback with `'epic_run.story.completed'` → assert `handleSSEEvent` was called
- Assert `nodes` computed produces correct VueFlow Node shape from store stories (2 stories in group 0 → y = 0 and y = 120)

**`EpicRunGroupList.spec.ts`:**
- Pass stories: 2 in group 0 (one running, one pending), 1 in group 1 (completed)
- Assert 2 group rows rendered
- Assert group 0 Tag has `severity="info"` (running present)
- Assert group 1 Tag has `severity="success"` (all completed)

### References

- Story 7-4 (done): `frontend/src/features/dag/DagGraph.vue`, `frontend/src/features/dag/DagStoryNode.vue` — patterns to follow for `EpicRunStatusNode.vue`
- Story 4-3 (done): `frontend/src/composables/useSSE.ts` — extend known events list only
- Existing store pattern: `frontend/src/stores/epics.ts`
- Existing router: `frontend/src/router/index.ts` — `epic-dag` child route is the insertion point reference
- `@vue-flow/core` Handle + custom node: https://vueflow.dev/guide/custom-nodes.html
- PrimeVue ProgressBar: https://primevue.org/progressbar/
- PrimeVue Message: https://primevue.org/message/
- PrimeVue Tag severity: https://primevue.org/tag/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
