# Story 5.4: [FRONT] HITL Approval Page + DiffViewer

Status: ready-for-dev

## Story

As a reviewer, I want a dedicated approval page that shows the agent's diff and story context, so that I can make an informed approve or reject decision without leaving the platform.

## Acceptance Criteria (BDD)

**AC1: Route and page load**
- **Given** a pipeline step is in `waiting_approval` state
- **When** a reviewer navigates to `/projects/:projectId/runs/:runId/approve/:stepId`
- **Then** the page fetches the HITL request for that step and displays story context (key, title, objective)
- **And** a loading skeleton is shown while data is being fetched
- **And** a `Message` severity="error" is shown if the fetch fails

**AC2: DiffViewer component renders git diff**
- **Given** the HITL request contains a `diff_content` string (unified diff format)
- **When** the DiffViewer component mounts
- **Then** it renders the diff using `diff2html` with syntax highlighting
- **And** a toggle button switches between side-by-side and unified view
- **And** if `diff_content` is null/empty, a placeholder "No diff available" message is shown

**AC3: Approve action**
- **Given** the reviewer clicks the Approve button
- **When** `POST /api/v1/hitl-requests/{id}/approve` succeeds
- **Then** a success Toast is shown ("Approval submitted")
- **And** the router navigates to the run detail page (`/runs/:runId`)

**AC4: Reject action**
- **Given** the reviewer clicks the Reject button
- **When** a dialog appears requiring a rejection reason (Textarea, min 10 characters, validated with zod)
- **And** the reviewer enters a valid reason and confirms
- **Then** `POST /api/v1/hitl-requests/{id}/reject` is called with `{ reason: "..." }`
- **And** a success Toast is shown ("Rejection submitted")
- **And** the router navigates to the run detail page

**AC5: Reject action validation**
- **Given** the reject dialog is open
- **When** the reviewer submits with a reason shorter than 10 characters
- **Then** an inline validation error is shown ("Reason must be at least 10 characters")
- **And** the API call is NOT made

**AC6: SSE toast notification for new HITL requests**
- **Given** the SSE event bus receives a `hitl_gate.pending` event for the current project
- **When** the event payload contains `{ run_id, step_id, story_key, hitl_request_id }`
- **Then** a persistent Toast notification appears: "Review required for [story_key]" with a "Review Now" action button
- **And** clicking "Review Now" navigates to the approval page for that step

**AC7: useApprovalActions composable wraps API calls**
- **Given** the composable is used by the approval page
- **When** `approve(hitlRequestId)` or `reject(hitlRequestId, reason)` is called
- **Then** it uses `useAsyncAction` internally and returns `{ isLoading, error, approve, reject }`
- **And** error from the API propagates to the composable's `error` ref

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Update OpenAPI-generated client types (AC: #1, #3, #4)
  - [ ] Confirm `api/openapi.yaml` has `HITLRequest` schema and `GET /hitl-requests/{id}`, `POST /hitl-requests/{id}/approve`, `POST /hitl-requests/{id}/reject` endpoints (added by Story 5.1 Task 9)
  - [ ] Run `cd frontend && npm run generate-api` to regenerate types
  - [ ] Verify `HITLRequest` type and endpoint paths are present in the generated schema

- [ ] [FRONT] Task 2: Add approval route to Vue Router (AC: #1)
  - [ ] In `frontend/src/router/index.ts`, add route under `/projects/:projectId`:
    ```
    path: 'runs/:runId/approve/:stepId',
    name: 'hitl-approve',
    component: () => import('@/views/HITLApprovalView.vue'),
    meta: { requiresAuth: true }
    ```

- [ ] [FRONT] Task 3: Install diff2html and implement DiffViewer component (AC: #2)
  - [ ] Run `cd frontend && npm install diff2html`
  - [ ] Create `frontend/src/features/approvals/DiffViewer.vue`
  - [ ] Props: `diff: string | null | undefined`, `mode: 'side-by-side' | 'line-by-line'` (default `'side-by-side'`)
  - [ ] Emit `update:mode` for v-model binding
  - [ ] Use `diff2html` to convert unified diff to HTML; inject in a `<div>` via `v-html`
  - [ ] Import `diff2html/bundles/css` in the component's `<style>` block
  - [ ] Show `EmptyState` with "No diff available" when `diff` is null/empty

- [ ] [FRONT] Task 4: Implement useApprovalActions composable (AC: #7, #3, #4)
  - [ ] Create `frontend/src/features/approvals/composables/useApprovalActions.ts`
  - [ ] Use `useAsyncAction` for both `approve` and `reject` operations
  - [ ] `approve(hitlRequestId: string): Promise<void>` — `POST /api/v1/hitl-requests/{id}/approve`
  - [ ] `reject(hitlRequestId: string, reason: string): Promise<void>` — `POST /api/v1/hitl-requests/{id}/reject` with body `{ reason }`
  - [ ] Export `{ approveAction, rejectAction }` where each has its own `isLoading`, `error`, `execute`

- [ ] [FRONT] Task 5: Implement HITLApprovalView page (AC: #1, #2, #3, #4, #5)
  - [ ] Create `frontend/src/views/HITLApprovalView.vue`
  - [ ] Fetch HITL request on mount via `GET /api/v1/hitl-requests/{id}` using the `stepId` from route to look up the request
  - [ ] Show `Skeleton` while loading, `Message` severity="error" on fetch error
  - [ ] Display story context: key (`Tag`), title (`<h1>`), objective (collapsible `Panel`)
  - [ ] Render `DiffViewer` with toggle for view mode (side-by-side / unified)
  - [ ] Approve button: `Button` severity="success", calls `useApprovalActions.approve`, then navigates to `/runs/:runId`
  - [ ] Reject button: `Button` severity="danger", opens reject dialog

- [ ] [FRONT] Task 6: Implement reject dialog with zod validation (AC: #4, #5)
  - [ ] In `HITLApprovalView.vue`, include a PrimeVue `Dialog` for rejection
  - [ ] Use `vee-validate` + `zod` schema: `z.object({ reason: z.string().min(10, 'Reason must be at least 10 characters') })`
  - [ ] `Textarea` for reason input with inline error message on validation failure
  - [ ] Confirm button calls `useApprovalActions.reject`, closes dialog, shows Toast, navigates to run detail
  - [ ] Cancel button closes dialog without side effects

- [ ] [FRONT] Task 7: SSE toast notification for hitl_gate.pending (AC: #6)
  - [ ] In `frontend/src/stores/runs.ts` (or a new `approvals.ts` store), handle `hitl_gate.pending` SSE event
  - [ ] When event received, call PrimeVue `useToast().add()` with:
    - severity: `'warn'`
    - summary: `'Review Required'`
    - detail: `'Review required for ${storyKey}'`
    - life: `0` (persistent until dismissed)
    - a custom action slot or life=0 + closable=true
  - [ ] Provide a "Review Now" button in the toast that calls `router.push` to the approval page

- [ ] [FRONT] Task 8: Unit tests for composable and DiffViewer (AC: #7)
  - [ ] Create `frontend/src/features/approvals/__tests__/useApprovalActions.spec.ts`
  - [ ] Test approve: mocks API call, verifies success path and error path
  - [ ] Test reject: validates min-length enforcement, verifies API called with correct body
  - [ ] Create `frontend/src/features/approvals/__tests__/DiffViewer.spec.ts`
  - [ ] Test: renders diff2html output when diff prop is provided
  - [ ] Test: renders EmptyState when diff is null

- [ ] [FRONT] Task 9: Type-check and lint (AC: all)
  - [ ] Run `cd frontend && npm run type-check` — must pass with zero errors
  - [ ] Run `cd frontend && npm run lint` — must pass
  - [ ] Fix any ESLint or TypeScript strict-mode issues before committing

## Dev Notes

### Dependencies

**Story 5.1 (HITL Gate Action — Wave 10, same wave):** Story 5.1 Task 9 adds the `HITLRequest` schema and stub endpoints (`GET /hitl-requests/{id}`, `POST /hitl-requests/{id}/approve`, `POST /hitl-requests/{id}/reject`) to `api/openapi.yaml`. This story's Task 1 regenerates the frontend types from that spec. The backend handler implementation is in Story 5.2 (Wave 11) — the frontend can be built against the generated types now.

**Story 5.2 (Approve/Reject API — Wave 11):** Backend handlers. Frontend built against spec contract. In local dev, responses can be mocked via MSW or the real backend once Story 5.2 lands.

### Architecture Requirements

- `HITLApprovalView.vue` is a view (1:1 with route) — composes feature components
- `DiffViewer.vue` lives in `frontend/src/features/approvals/` (single-feature use for now; promote to `ui/composed/` if reused)
- `useApprovalActions.ts` is the composable — all API calls go here, zero business logic in the view
- SSE event handling belongs in the Pinia store or the `useSSE` composable dispatcher — not in a component
- The `approve` and `reject` actions each have independent `isLoading` / `error` refs (separate `useAsyncAction` calls)

### File Paths (exact)

```
frontend/src/views/HITLApprovalView.vue                                        # New: approval page view
frontend/src/features/approvals/DiffViewer.vue                                 # New: diff rendering component
frontend/src/features/approvals/composables/useApprovalActions.ts              # New: approve/reject API calls
frontend/src/features/approvals/__tests__/useApprovalActions.spec.ts           # New: composable unit tests
frontend/src/features/approvals/__tests__/DiffViewer.spec.ts                   # New: component unit tests
frontend/src/router/index.ts                                                    # Add hitl-approve route
frontend/src/stores/runs.ts                                                     # Add hitl_gate.pending SSE handler
api/openapi.yaml                                                                # Extended by Story 5.1 (read-only here)
```

### Technical Specifications

**Route definition:**
```typescript
// frontend/src/router/index.ts — add inside /projects/:id children:
{
  path: 'runs/:runId/approve/:stepId',
  name: 'hitl-approve',
  component: () => import('@/views/HITLApprovalView.vue'),
  meta: { requiresAuth: true },
}
```

**useApprovalActions composable:**
```typescript
// frontend/src/features/approvals/composables/useApprovalActions.ts
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useApprovalActions() {
  const approveAction = useAsyncAction(async (hitlRequestId: string) => {
    const { error } = await apiClient.POST('/api/v1/hitl-requests/{id}/approve', {
      params: { path: { id: hitlRequestId } },
    })
    if (error) throw new Error(error.error?.message ?? 'Approve failed')
  })

  const rejectAction = useAsyncAction(async (hitlRequestId: string, reason: string) => {
    const { error } = await apiClient.POST('/api/v1/hitl-requests/{id}/reject', {
      params: { path: { id: hitlRequestId } },
      body: { reason },
    })
    if (error) throw new Error(error.error?.message ?? 'Reject failed')
  })

  return { approveAction, rejectAction }
}
```

**DiffViewer component pattern:**
```vue
<script setup lang="ts">
import { computed } from 'vue'
import * as Diff2Html from 'diff2html'
import 'diff2html/bundles/css/diff2html.min.css'

const props = defineProps<{
  diff: string | null | undefined
  mode: 'side-by-side' | 'line-by-line'
}>()

const emit = defineEmits<{ 'update:mode': [mode: 'side-by-side' | 'line-by-line'] }>()

const html = computed(() => {
  if (!props.diff) return null
  return Diff2Html.html(props.diff, {
    drawFileList: true,
    matching: 'lines',
    outputFormat: props.mode,
  })
})
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex justify-end gap-2">
      <Button
        :label="mode === 'side-by-side' ? 'Unified' : 'Side by side'"
        size="small"
        severity="secondary"
        @click="emit('update:mode', mode === 'side-by-side' ? 'line-by-line' : 'side-by-side')"
      />
    </div>
    <div v-if="html" v-html="html" />
    <EmptyState v-else message="No diff available" />
  </div>
</template>
```

**HITLApprovalView skeleton structure:**
```vue
<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { useApprovalActions } from '@/features/approvals/composables/useApprovalActions'
import { apiClient } from '@/api/client'
import DiffViewer from '@/features/approvals/DiffViewer.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const projectId = route.params.projectId as string
const runId = route.params.runId as string
const stepId = route.params.stepId as string

const diffMode = ref<'side-by-side' | 'line-by-line'>('side-by-side')
const showRejectDialog = ref(false)
const rejectReason = ref('')

// Fetch HITL request by step ID
const fetchAction = useAsyncAction(async () => {
  const { data, error } = await apiClient.GET('/api/v1/hitl-requests/{id}', {
    params: { path: { id: stepId } }, // stepId used to look up HITL request
  })
  if (error) throw new Error(error.error?.message ?? 'Failed to load review')
  return data
})

fetchAction.execute()

const { approveAction, rejectAction } = useApprovalActions()

async function handleApprove() {
  if (!fetchAction.data.value) return
  await approveAction.execute(fetchAction.data.value.id)
  if (!approveAction.error.value) {
    toast.add({ severity: 'success', summary: 'Approval submitted', life: 3000 })
    router.push({ name: 'run-detail', params: { id: runId } })
  }
}

async function handleReject() {
  if (!fetchAction.data.value) return
  await rejectAction.execute(fetchAction.data.value.id, rejectReason.value)
  if (!rejectAction.error.value) {
    showRejectDialog.value = false
    toast.add({ severity: 'success', summary: 'Rejection submitted', life: 3000 })
    router.push({ name: 'run-detail', params: { id: runId } })
  }
}
</script>
```

**Zod schema for reject form:**
```typescript
import { z } from 'zod'

const rejectSchema = z.object({
  reason: z.string().min(10, 'Reason must be at least 10 characters'),
})
```

**SSE handler in runs.ts store:**
```typescript
// In frontend/src/stores/runs.ts — add to SSE event dispatcher
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'

// Called from useSSE composable when event type is 'hitl_gate' and action is 'pending'
function handleHITLPendingEvent(payload: {
  run_id: string
  step_id: string
  story_key: string
  hitl_request_id: string
}) {
  const toast = useToast()
  const router = useRouter()
  toast.add({
    severity: 'warn',
    summary: 'Review Required',
    detail: `Review required for ${payload.story_key}`,
    life: 0, // persistent
    group: 'hitl',
  })
  // "Review Now" navigation is handled via a custom toast template or
  // by storing the pending approval in the approvals store for the queue to show
}
```

**Note on SSE toast "Review Now" button:** PrimeVue Toast does not natively support action buttons in the default template. Two approaches are valid:
1. Use a `ToastMessage` with a custom slot (PrimeVue 4 supports `<template #message>`).
2. Store the pending HITL request in a Pinia `approvals` store; a persistent notification badge in the AppShell navbar links to `/approvals`. This is the simpler approach if the custom slot adds too much complexity.

Choose approach 2 if approach 1 requires excessive boilerplate — document the decision inline.

**diff2html import:** The CSS must be imported; either in the component's `<style>` block or in `assets/main.css`. Prefer the component-level import to keep the dependency scoped.

### Testing Requirements

**useApprovalActions.spec.ts:**
```typescript
import { describe, it, expect, vi } from 'vitest'
import { useApprovalActions } from '../composables/useApprovalActions'

describe('useApprovalActions', () => {
  it('approve calls correct endpoint', async () => {
    // mock apiClient.POST
    // call approveAction.execute('hitl-id')
    // assert endpoint called with correct params
  })

  it('reject requires reason in body', async () => {
    // call rejectAction.execute('hitl-id', 'this is my reason')
    // assert body: { reason: 'this is my reason' }
  })

  it('propagates API error to error ref', async () => {
    // mock API to return error
    // verify rejectAction.error.value is set
  })
})
```

**DiffViewer.spec.ts:**
```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DiffViewer from '../DiffViewer.vue'

describe('DiffViewer', () => {
  it('renders EmptyState when diff is null', () => {
    const wrapper = mount(DiffViewer, { props: { diff: null, mode: 'side-by-side' } })
    expect(wrapper.text()).toContain('No diff available')
  })

  it('renders diff html when diff is provided', () => {
    const wrapper = mount(DiffViewer, {
      props: { diff: '--- a/foo.go\n+++ b/foo.go\n@@ -1,1 +1,1 @@\n-old\n+new', mode: 'side-by-side' }
    })
    expect(wrapper.find('.d2h-wrapper').exists()).toBe(true)
  })
})
```

**Lint and type-check must pass before commit:**
```bash
cd frontend && npm run lint && npm run type-check
```

### References

- `frontend/src/composables/useAsyncAction.ts` — async operation pattern
- `frontend/src/router/index.ts` — existing route structure (add under `/projects/:id` children)
- `frontend/src/stores/runs.ts` — SSE event handling pattern
- `frontend/src/features/runs/` — existing run feature components for structural reference
- `api/openapi.yaml` — HITL endpoints added by Story 5.1 Task 9
- `diff2html` npm package — https://github.com/rtfpessoa/diff2html
- `frontend/CLAUDE.md` — Vue 3, PrimeVue, Tailwind conventions
- `frontend/src/views/ApprovalsView.vue` — existing approvals view (may contain related code to reuse)

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
