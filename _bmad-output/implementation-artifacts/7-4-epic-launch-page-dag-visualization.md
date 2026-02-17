# Story 7.4: [FRONT] Epic Launch Page + DAG Visualization

Status: ready-for-dev

## Story

As a project user, I want to see the dependency graph of an epic's stories and launch the entire epic as a batch run, So that I can understand execution order at a glance and kick off all stories with a single action.

## Acceptance Criteria (BDD)

**AC1: DAG graph renders stories as nodes organized in layers**
- **Given** an epic with stories that have declared dependencies
- **When** the EpicDetailView loads and the DAG tab/section is displayed
- **Then** each story is shown as a node with: key (bold monospace), title (truncated to 40 chars), status badge
- **And** nodes are positioned at x = layer * 250, y = position_within_layer * 120
- **And** edges connect dependent stories with directional arrows

**AC2: Done stories are visually dimmed in the graph**
- **Given** some stories in the epic have status "done"
- **When** the graph renders
- **Then** nodes for done stories are shown at reduced opacity (0.4)

**AC3: Graph supports zoom, pan, and minimap**
- **Given** the DAG graph is rendered
- **When** the user interacts with the graph
- **Then** zoom in/out works via scroll wheel and the Controls panel
- **And** pan works by dragging the background
- **And** a minimap is visible in the bottom-right corner

**AC4: Loading and error states are handled gracefully**
- **Given** the DAG endpoint is slow or returns an error
- **When** the component mounts
- **Then** a PrimeVue Skeleton is shown during loading
- **And** a PrimeVue Message with retry button is shown on error

**AC5: "Launch Epic" button triggers POST and shows confirmation**
- **Given** an epic with stories
- **When** the user clicks "Launch Epic" in the EpicDetailView header
- **Then** a PrimeVue ConfirmDialog is shown with the count of stories and a warning
- **When** the user confirms
- **Then** POST /api/v1/projects/{projectId}/epics/{epicId}/runs is called
- **And** a success Toast with the epic_run_id is shown on 202 response
- **And** the button is disabled and shows "Launching..." during the request

**AC6: New route /projects/:id/epics/:epicId/dag is registered**
- **Given** a user navigates to `/projects/:id/epics/:epicId/dag`
- **When** the page loads
- **Then** the EpicDagView is rendered with the DAG graph and Launch Epic button
- **And** a back-navigation button returns to the epic detail view

**AC7: useDagLayout composable fetches DAG data**
- **Given** a projectId and epicId
- **When** useDagLayout is called and mounted
- **Then** it calls GET /api/v1/projects/{projectId}/epics/{epicId}/dag
- **And** transforms nodes/edges into @vue-flow/core compatible format
- **And** exposes: nodes, edges, isLoading, error, retry

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Install @vue-flow/core and regenerate API types (AC: #1, #7)
  - [ ] Run `cd frontend && npm install @vue-flow/core`
  - [ ] Run `cd frontend && npm run generate-api` to pick up new `getEpicDAG` and `launchEpicRun` endpoints from openapi.yaml (requires Story 7-1 OpenAPI changes to be merged first)
  - [ ] Verify generated types include `EpicDAGResponse`, `EpicDAGNode`, `EpicDAGEdge`, `EpicRunAccepted`

- [ ] [FRONT] Task 2: Create useDagLayout composable (AC: #7)
  - [ ] File: `frontend/src/features/dag/composables/useDagLayout.ts`
  - [ ] Accepts `projectId: string`, `epicId: string`
  - [ ] Uses `useAsyncAction` to call `apiClient.GET('/projects/{projectId}/epics/{epicId}/dag', ...)`
  - [ ] Transforms API nodes to `@vue-flow/core` `Node[]`: `{ id: node.key, position: { x: node.layer * 250, y: posWithinLayer * 120 }, data: { key, title, status } }`
  - [ ] Transforms API edges to `@vue-flow/core` `Edge[]`: `{ id: \`${edge.source}-${edge.target}\`, source: edge.source, target: edge.target }`
  - [ ] Tracks `posWithinLayer` per layer using a `Map<number, number>` counter while iterating nodes
  - [ ] Exposes: `nodes` (Ref<Node[]>), `edges` (Ref<Edge[]>), `isLoading`, `error`, `retry`
  - [ ] Calls fetch on `onMounted`
  - [ ] Write unit test: `frontend/src/features/dag/__tests__/useDagLayout.spec.ts`

- [ ] [FRONT] Task 3: Create DagStoryNode.vue custom node component (AC: #1, #2)
  - [ ] File: `frontend/src/features/dag/DagStoryNode.vue`
  - [ ] Receives `@vue-flow/core` node data props: `{ key: string, title: string, status: string }`
  - [ ] Renders: story key in bold monospace, title truncated to 40 chars with `title` attribute for full text, PrimeVue Tag for status with severity mapping (backlog=secondary, running=info, done=success, failed=danger)
  - [ ] Applies `opacity-40` class when `status === 'done'` (AC2)
  - [ ] Uses `Handle` from `@vue-flow/core` for source and target connection points
  - [ ] No business logic — purely presentational

- [ ] [FRONT] Task 4: Create DagGraph.vue component (AC: #1, #2, #3, #4)
  - [ ] File: `frontend/src/features/dag/DagGraph.vue`
  - [ ] Props: `nodes: Node[]`, `edges: Edge[]`, `isLoading: boolean`, `error: string | null`
  - [ ] Emits: `retry: []`
  - [ ] Uses `<VueFlow>` from `@vue-flow/core` with `<Controls />` and `<MiniMap />`
  - [ ] Registers `DagStoryNode` as a custom node type: `:node-types="{ story: DagStoryNode }"`
  - [ ] Shows PrimeVue Skeleton (full height) while isLoading; shows PrimeVue Message + retry Button on error
  - [ ] Import `@vue-flow/core/dist/style.css` and `@vue-flow/core/dist/theme-default.css` in component
  - [ ] Pass `fit-view-on-init` prop to VueFlow so graph is auto-centered on load

- [ ] [FRONT] Task 5: Create useEpicLauncher composable (AC: #5)
  - [ ] File: `frontend/src/features/dag/composables/useEpicLauncher.ts`
  - [ ] Accepts `projectId: string`, `epicId: string`
  - [ ] Uses `useAsyncAction` to wrap `apiClient.POST('/projects/{projectId}/epics/{epicId}/runs', ...)`
  - [ ] Exposes: `launch()`, `isLaunching` (alias for isLoading), `error`, `result` (EpicRunAccepted | null)
  - [ ] Write unit test: `frontend/src/features/dag/__tests__/useEpicLauncher.spec.ts`
  - [ ] Test: success path returns EpicRunAccepted with epic_run_id
  - [ ] Test: error path sets error ref

- [ ] [FRONT] Task 6: Create EpicDagView.vue page view (AC: #1–#6)
  - [ ] File: `frontend/src/views/EpicDagView.vue`
  - [ ] Reads `projectId` from `route.params.id` and `epicId` from `route.params.epicId`
  - [ ] Uses `useDagLayout(projectId, epicId)` and `useEpicLauncher(projectId, epicId)`
  - [ ] Header row: back button (navigate to `epic-detail` route), h1 "Epic DAG", "Launch Epic" Button (severity="success")
  - [ ] Launch button: disabled + label "Launching..." while `isLaunching`; on click opens PrimeVue ConfirmDialog
  - [ ] ConfirmDialog message: `"Launch all {nodes.length} stories in this epic?"` with note that already-running stories will be skipped
  - [ ] On confirm: calls `launch()`, on success shows Toast (severity=success, summary="Epic run scheduled", detail=`epic_run_id`), on error shows Toast (severity=error)
  - [ ] Renders `<DagGraph :nodes :edges :is-loading :error @retry="retry" />`
  - [ ] Import Toast, ConfirmDialog from primevue; use `useToast()` and `useConfirm()` hooks

- [ ] [FRONT] Task 7: Register route and add navigation entry point (AC: #6)
  - [ ] Update `frontend/src/router/index.ts`: add child route under `/projects/:id` — `{ path: 'epics/:epicId/dag', name: 'epic-dag', component: () => import('@/views/EpicDagView.vue') }`
  - [ ] Update `frontend/src/views/EpicDetailView.vue`: add "View DAG" button in header row (alongside back button) that navigates to `epic-dag` route with current projectId and epicId
  - [ ] Button: PrimeVue Button, severity="secondary", icon="pi pi-sitemap", label="View DAG"

- [ ] [FRONT] Task 8: Write unit tests for DagGraph.vue (AC: #1, #2, #4)
  - [ ] File: `frontend/src/features/dag/__tests__/DagGraph.spec.ts`
  - [ ] Test: renders Skeleton when isLoading is true
  - [ ] Test: renders Message when error is not null
  - [ ] Test: emits retry when retry button is clicked
  - [ ] Test: renders VueFlow when nodes and edges are provided (check VueFlow is mounted, not internal rendering — stub @vue-flow/core)

- [ ] [FRONT] Task 9: Run lint and type-check (AC: all)
  - [ ] `cd frontend && npm run lint` — must pass with zero errors
  - [ ] `cd frontend && npm run type-check` — tsc must pass (no type errors on new vue-flow Node/Edge types)
  - [ ] Fix any ESLint `vue/no-unused-vars` or TypeScript strict mode violations

## Dev Notes

### Dependencies

- Story 7-1: DAG builder backend + GET /epics/{epicId}/dag endpoint + POST /epics/{epicId}/runs OpenAPI stub — must be merged first so `npm run generate-api` picks up the new types
- Story 2-5, 2-6: EpicDetailView.vue and EpicDetailLayout.vue already in place — this story adds a nav button only, does not restructure them

### Architecture Requirements

Feature directory structure for DAG:

```
frontend/src/features/dag/
├── DagGraph.vue                        (new — graph container with loading/error)
├── DagStoryNode.vue                    (new — custom node component for @vue-flow/core)
├── composables/
│   ├── useDagLayout.ts                 (new — fetch + transform DAG data)
│   └── useEpicLauncher.ts              (new — POST epic run)
└── __tests__/
    ├── DagGraph.spec.ts                (new)
    ├── useDagLayout.spec.ts            (new)
    └── useEpicLauncher.spec.ts         (new)
```

Component hierarchy for EpicDagView:

```
EpicDagView.vue (new view, route: epic-dag)
├── Toast (primevue)
├── ConfirmDialog (primevue)
├── Header row
│   ├── Button (back — navigate to epic-detail)
│   ├── h1 "Epic DAG"
│   └── Button "Launch Epic" (opens ConfirmDialog → useEpicLauncher)
└── DagGraph.vue
    ├── Skeleton (isLoading)
    ├── Message + Button (error)
    └── VueFlow (@vue-flow/core)
        ├── Controls
        ├── MiniMap
        └── DagStoryNode (custom node type "story")
            ├── Handle (target — top)
            ├── story key (monospace bold)
            ├── title (truncated)
            ├── PrimeVue Tag (status)
            └── Handle (source — bottom)
```

Data flow:

```
EpicDagView
  → useDagLayout(projectId, epicId)
      → GET /projects/{projectId}/epics/{epicId}/dag
      → transforms to Node[] + Edge[]
  → DagGraph receives :nodes :edges :is-loading :error
      → VueFlow renders with DagStoryNode custom nodes
  → useEpicLauncher(projectId, epicId)
      → on confirm: POST /projects/{projectId}/epics/{epicId}/runs
      → on 202: Toast success
```

### File Paths (exact)

```
frontend/src/features/dag/DagGraph.vue                               (new)
frontend/src/features/dag/DagStoryNode.vue                           (new)
frontend/src/features/dag/composables/useDagLayout.ts                (new)
frontend/src/features/dag/composables/useEpicLauncher.ts             (new)
frontend/src/features/dag/__tests__/DagGraph.spec.ts                 (new)
frontend/src/features/dag/__tests__/useDagLayout.spec.ts             (new)
frontend/src/features/dag/__tests__/useEpicLauncher.spec.ts          (new)
frontend/src/views/EpicDagView.vue                                   (new)
frontend/src/router/index.ts                                         (add epic-dag route)
frontend/src/views/EpicDetailView.vue                                (add "View DAG" button in header)
```

### Technical Specifications

**useDagLayout.ts:**
```typescript
import { onMounted } from 'vue'
import type { Node, Edge } from '@vue-flow/core'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useDagLayout(projectId: string, epicId: string) {
  const layerCounters = new Map<number, number>()

  const { data: dagData, isLoading, error, execute } = useAsyncAction(async () => {
    const { data, error: apiErr } = await apiClient.GET(
      '/projects/{projectId}/epics/{epicId}/dag',
      { params: { path: { projectId, epicId } } },
    )
    if (apiErr) throw new Error('Failed to load DAG')
    return data
  })

  const nodes = computed<Node[]>(() => {
    if (!dagData.value) return []
    layerCounters.clear()
    return dagData.value.nodes.map((n) => {
      const pos = layerCounters.get(n.layer) ?? 0
      layerCounters.set(n.layer, pos + 1)
      return {
        id: n.key,
        type: 'story',
        position: { x: n.layer * 250, y: pos * 120 },
        data: { key: n.key, title: n.title, status: n.status },
      }
    })
  })

  const edges = computed<Edge[]>(() => {
    if (!dagData.value) return []
    return dagData.value.edges.map((e) => ({
      id: `${e.source}-${e.target}`,
      source: e.source,
      target: e.target,
    }))
  })

  async function retry() {
    await execute()
  }

  onMounted(execute)

  return { nodes, edges, isLoading, error, retry }
}
```

**DagStoryNode.vue:**
```vue
<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import Tag from 'primevue/tag'

const props = defineProps<{
  data: { key: string; title: string; status: string }
}>()

const truncatedTitle = computed(() =>
  props.data.title.length > 40 ? props.data.title.slice(0, 40) + '…' : props.data.title
)

const statusSeverity = computed(() => {
  const map: Record<string, string> = {
    backlog: 'secondary',
    running: 'info',
    done: 'success',
    failed: 'danger',
  }
  return map[props.data.status] ?? 'secondary'
})

const isDone = computed(() => props.data.status === 'done')
</script>

<template>
  <div :class="['dag-story-node', { 'opacity-40': isDone }]">
    <Handle type="target" :position="Position.Top" />
    <div class="flex flex-col gap-1 p-2">
      <span class="font-mono font-bold text-sm">{{ data.key }}</span>
      <span :title="data.title" class="text-xs">{{ truncatedTitle }}</span>
      <Tag :value="data.status" :severity="statusSeverity" class="text-xs" />
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
</template>
```

**useEpicLauncher.ts:**
```typescript
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useEpicLauncher(projectId: string, epicId: string) {
  const { data: result, isLoading: isLaunching, error, execute } = useAsyncAction(async () => {
    const { data, error: apiErr } = await apiClient.POST(
      '/projects/{projectId}/epics/{epicId}/runs',
      { params: { path: { projectId, epicId } } },
    )
    if (apiErr) throw new Error('Failed to launch epic run')
    return data
  })

  async function launch() {
    await execute()
  }

  return { launch, isLaunching, error, result }
}
```

**EpicDagView.vue structure:**
```vue
<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import DagGraph from '@/features/dag/DagGraph.vue'
import { useDagLayout } from '@/features/dag/composables/useDagLayout'
import { useEpicLauncher } from '@/features/dag/composables/useEpicLauncher'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string
const epicId = route.params.epicId as string
const toast = useToast()
const confirm = useConfirm()

const { nodes, edges, isLoading, error, retry } = useDagLayout(projectId, epicId)
const { launch, isLaunching, error: launchError, result } = useEpicLauncher(projectId, epicId)

function handleLaunchClick() {
  confirm.require({
    message: `Launch all ${nodes.value.length} stories in this epic? Already-running stories will be skipped.`,
    header: 'Launch Epic Run',
    icon: 'pi pi-play',
    acceptLabel: 'Launch',
    rejectLabel: 'Cancel',
    accept: async () => {
      await launch()
      if (result.value) {
        toast.add({ severity: 'success', summary: 'Epic run scheduled',
          detail: `Run ID: ${result.value.epic_run_id}`, life: 5000 })
      } else {
        toast.add({ severity: 'error', summary: 'Launch failed',
          detail: launchError.value?.message ?? 'Unexpected error', life: 5000 })
      }
    },
  })
}
</script>
```

**Router addition (router/index.ts) — insert inside /projects/:id children:**
```typescript
{
  path: 'epics/:epicId/dag',
  name: 'epic-dag',
  component: () => import('@/views/EpicDagView.vue'),
},
```

**EpicDetailView.vue header addition — add next to back button:**
```html
<Button
  icon="pi pi-sitemap"
  label="View DAG"
  severity="secondary"
  @click="router.push({ name: 'epic-dag', params: { id: projectId, epicId } })"
/>
```

**DagGraph.vue CSS import note:**
```typescript
// In DagGraph.vue <script setup>:
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
```
These must be imported at the component level (not in main.css) to avoid polluting global styles.

### Testing Requirements

**useDagLayout.spec.ts:**
- Mock `apiClient.GET` to return `{ nodes: [{ key: 'S-01', layer: 0, title: 'Test', status: 'backlog' }], edges: [] }`
- Assert `nodes.value` has 1 entry with `position: { x: 0, y: 0 }` and `type: 'story'`
- Assert `edges.value` is empty
- Assert `isLoading` transitions true → false
- Test with two nodes in same layer: second gets `y: 120`
- Test error path: apiClient returns error → error ref is set

**useEpicLauncher.spec.ts:**
- Mock `apiClient.POST` → 202 `{ epic_run_id: 'uuid', status: 'scheduling', stories_count: 3 }`
- Assert `result.value.epic_run_id` is set after `launch()` resolves
- Test error path: apiClient POST returns error → `error.value` is set, `result.value` is null

**DagGraph.spec.ts:**
- Stub `@vue-flow/core` entirely (it has a complex internal WASM-like setup — avoid in unit tests)
- Test: `isLoading=true` → Skeleton is rendered
- Test: `error="fetch failed"` → Message component renders with error text
- Test: `error + retry click` → `retry` event emitted

### References

- @vue-flow/core docs: https://vueflow.dev/guide/
- @vue-flow/core custom nodes: https://vueflow.dev/guide/custom-nodes.html
- @vue-flow/core Handle: `import { Handle, Position } from '@vue-flow/core'`
- @vue-flow/core Controls + MiniMap: `import { Controls, MiniMap } from '@vue-flow/core'`
- Story 7-1: adds GET /epics/{epicId}/dag and POST /epics/{epicId}/runs to openapi.yaml
- Existing composable pattern: `frontend/src/composables/useRunLauncher.ts`
- Existing view pattern: `frontend/src/views/EpicDetailView.vue`
- Existing router pattern: `frontend/src/router/index.ts` (children under /projects/:id)
- PrimeVue ConfirmDialog: https://primevue.org/confirmdialog/
- PrimeVue Toast: https://primevue.org/toast/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
