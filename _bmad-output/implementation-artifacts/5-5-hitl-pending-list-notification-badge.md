# Story 5.5: [FRONT] HITL Pending List + Notification Badge

Status: ready-for-dev

## Story

As a reviewer, I want to see a notification badge with the pending HITL count and a global list of all pending approvals, so that I never miss a gate waiting for human review across any project.

## Acceptance Criteria (BDD)

**AC1: Notification badge in sidebar**
- **Given** the app is loaded and the sidebar is visible
- **When** there are one or more pending HITL approvals across any project
- **Then** an orange badge showing the count is displayed on the "Approvals" nav item in `AppSidebar.vue`
- **And** when the count is zero, no badge is rendered (not even a "0" badge)

**AC2: Badge updates on SSE `hitl.pending` event**
- **Given** the global SSE connection is active
- **When** a `hitl.pending` event arrives with payload `{ run_id, step_id, project_id, story_key, hitl_request_id }`
- **Then** `useHITLStore` adds the entry to `pendingItems` and `pendingCount` increments by 1
- **And** the sidebar badge re-renders immediately without a page refresh

**AC3: Badge decrements on `hitl.approved` and `hitl.rejected` SSE events**
- **Given** the SSE connection is active and the badge shows count N
- **When** a `hitl.approved` or `hitl.rejected` event arrives with payload `{ hitl_request_id }`
- **Then** the matching entry is removed from `pendingItems` and `pendingCount` becomes N-1
- **And** if N-1 equals zero, the badge disappears

**AC4: Pending HITL list page loads at `/approvals`**
- **Given** a reviewer navigates to `/approvals`
- **When** `ApprovalsView.vue` mounts
- **Then** it calls `GET /api/v1/hitl-requests?status=pending` and displays all pending approvals in a `DataTable`
- **And** a loading skeleton is shown during fetch
- **And** an inline `Message` severity="error" is shown if the fetch fails with a retry button

**AC5: DataTable columns and row content**
- **Given** the pending list has loaded successfully
- **When** the table is rendered
- **Then** each row displays: story key (`Tag` severity="info"), story title, project name, PR URL as an external link, and a relative "waiting since" time (using `useRelativeTime`)
- **And** a "Review" button per row navigates to `/projects/:projectId/runs/:runId/approve/:stepId`

**AC6: Empty state when no approvals are pending**
- **Given** the API returns an empty list
- **When** the `DataTable` renders
- **Then** an empty state message "No pending approvals" is displayed using PrimeVue's `DataTable` empty template
- **And** no error or spinner is shown

**AC7: Store hydration on page load**
- **Given** `useHITLStore` is initialized
- **When** `fetchPending()` is called on mount of `ApprovalsView`
- **Then** `pendingItems` is populated from the API response and `pendingCount` is computed from `pendingItems.length`
- **And** SSE events received after fetch merge with the hydrated list without duplication (dedup by `hitl_request_id`)

**AC8: Global SSE subscription managed at app shell level**
- **Given** the user is authenticated and the app shell is mounted
- **When** `AppShell.vue` (or `App.vue`) mounts
- **Then** a single `useSSE` connection subscribes to `hitl.pending`, `hitl.approved`, and `hitl.rejected` events and dispatches them to `useHITLStore`
- **And** the connection is torn down on unmount via `close()`

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `useHITLStore` Pinia store (AC: #2, #3, #7)
  - [ ] Define `HITLPendingItem` interface: `{ hitlRequestId, runId, stepId, projectId, projectName, storyKey, storyTitle, prUrl, pendingSince }`
  - [ ] State: `pendingItems: ref<HITLPendingItem[]>([])`, computed `pendingCount`
  - [ ] Actions: `fetchPending()` calling `GET /api/v1/hitl-requests?status=pending` via `apiClient`, `handlePendingEvent(payload)` (add with dedup), `handleResolvedEvent(hitlRequestId)` (remove by id)
  - [ ] Create `frontend/src/stores/hitl.ts`
  - [ ] Create unit tests in `frontend/src/stores/__tests__/hitl.spec.ts` covering add/remove/dedup logic

- [ ] [FRONT] Task 2: Wire global SSE subscription for HITL events (AC: #2, #3, #8)
  - [ ] In `frontend/src/ui/layout/AppShell.vue`, add `useSSE` for a global (no project filter) SSE stream or reuse the existing pattern with `project_id` omitted/wildcard
  - [ ] Register `hitl.pending`, `hitl.approved`, `hitl.rejected` event listeners dispatching to `useHITLStore`
  - [ ] Ensure `knownEvents` in `useSSE.ts` includes `hitl.approved` and `hitl.rejected`
  - [ ] Add cleanup via `close()` in `onBeforeUnmount`

- [ ] [FRONT] Task 3: Add notification badge to sidebar nav item (AC: #1)
  - [ ] In `frontend/src/ui/layout/AppSidebar.vue`, import `useHITLStore` and bind `pendingCount`
  - [ ] Add an "Approvals" entry to `navItems` with route `/approvals` and icon `pi pi-bell`
  - [ ] Render a PrimeVue `Badge` component overlaid on the button when `pendingCount > 0`, showing the count
  - [ ] When sidebar is collapsed, badge remains visible (overlaid on icon)

- [ ] [FRONT] Task 4: Build `HITLPendingTable` feature component (AC: #5, #6)
  - [ ] Create `frontend/src/features/approvals/HITLPendingTable.vue`
  - [ ] Props: `items: HITLPendingItem[]`, `loading: boolean`
  - [ ] Use PrimeVue `DataTable` with columns: story key (`Tag`), story title, project name, PR URL (`<a target="_blank">`), waiting since (`useRelativeTime`), actions ("Review" `Button`)
  - [ ] Empty slot: `<template #empty>No pending approvals</template>`
  - [ ] "Review" button emits `review` event with the item; parent handles navigation
  - [ ] Create unit test in `frontend/src/features/approvals/__tests__/HITLPendingTable.spec.ts`

- [ ] [FRONT] Task 5: Implement `ApprovalsView.vue` (AC: #4, #5, #6, #7)
  - [ ] Replace the placeholder in `frontend/src/views/ApprovalsView.vue` with full implementation
  - [ ] On mount: call `hitlStore.fetchPending()` via `useAsyncAction`
  - [ ] Render loading skeleton (`Skeleton` rows) while loading, `Message` on error with retry button
  - [ ] Render `HITLPendingTable` with `items` from store and `loading` bound to async state
  - [ ] On `review` event: `router.push({ name: 'hitl-approve', params: { id: projectId, runId, stepId } })`

- [ ] [FRONT] Task 6: Update `useSSE` composable known events (AC: #2, #3, #8)
  - [ ] In `frontend/src/composables/useSSE.ts`, add `hitl.approved` and `hitl.rejected` to the `knownEvents` array
  - [ ] Update unit test `frontend/src/composables/__tests__/useSSE.spec.ts` to cover the two new event types

- [ ] [FRONT] Task 7: Migrate `useApprovalsStore` into `useHITLStore` (AC: #7)
  - [ ] Update `frontend/src/stores/runs.ts` to dispatch HITL events to `useHITLStore` instead of `useApprovalsStore`
  - [ ] Keep `frontend/src/stores/approvals.ts` only if used elsewhere; otherwise deprecate it in favour of `useHITLStore`
  - [ ] Update any imports in `HITLApprovalView.vue` if `useApprovalsStore` is referenced there

- [ ] [FRONT] Task 8: E2E test for badge and list page (AC: #1, #4, #5)
  - [ ] Create `frontend/e2e/tests/hitl-pending-list.spec.ts`
  - [ ] Scenario 1: Mock API returns 2 pending items → sidebar badge shows "2", list page shows 2 rows
  - [ ] Scenario 2: Click "Review" on a row → navigates to the approval page URL
  - [ ] Scenario 3: Mock API returns empty list → empty state message visible, no badge in sidebar

## Dev Notes

### Dependencies

- Story 5-4 (done): `HITLApprovalView`, `DiffViewer`, `useApprovalActions`, route `hitl-approve` at `/projects/:id/runs/:runId/approve/:stepId`
- Story 5-2 (wave 11 backend): `GET /api/v1/hitl-requests?status=pending` endpoint — the store's `fetchPending()` depends on this endpoint being available

### Architecture Requirements

- `useHITLStore` is the single source of truth for pending HITL state. Do not duplicate pending count in `useApprovalsStore` or `useRunsStore`.
- The global SSE subscription for HITL events must live at the app shell level (not inside the `ApprovalsView`) so the badge updates regardless of which page the user is on.
- `pendingCount` must be a `computed` derived from `pendingItems.length`, never a separate `ref`, to guarantee consistency.
- Deduplication in `handlePendingEvent` is mandatory: check by `hitlRequestId` before pushing.

### File Paths (exact)

| File | Action |
|------|--------|
| `frontend/src/stores/hitl.ts` | Create — main HITL Pinia store |
| `frontend/src/stores/__tests__/hitl.spec.ts` | Create — unit tests for store |
| `frontend/src/stores/runs.ts` | Edit — dispatch to `useHITLStore` instead of `useApprovalsStore` |
| `frontend/src/composables/useSSE.ts` | Edit — add `hitl.approved`, `hitl.rejected` to `knownEvents` |
| `frontend/src/composables/__tests__/useSSE.spec.ts` | Edit — cover new event types |
| `frontend/src/ui/layout/AppSidebar.vue` | Edit — add Approvals nav item + Badge |
| `frontend/src/ui/layout/AppShell.vue` | Edit — add global HITL SSE subscription |
| `frontend/src/features/approvals/HITLPendingTable.vue` | Create — DataTable component |
| `frontend/src/features/approvals/__tests__/HITLPendingTable.spec.ts` | Create — unit tests |
| `frontend/src/views/ApprovalsView.vue` | Edit — replace placeholder with full implementation |
| `frontend/e2e/tests/hitl-pending-list.spec.ts` | Create — E2E tests |

### Technical Specifications

**`useHITLStore` store shape:**

```typescript
// frontend/src/stores/hitl.ts
import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

export interface HITLPendingItem {
  hitlRequestId: string
  runId: string
  stepId: string
  projectId: string
  projectName: string
  storyKey: string
  storyTitle: string
  prUrl: string | null
  pendingSince: string // ISO 8601
}

export const useHITLStore = defineStore('hitl', () => {
  const pendingItems = ref<HITLPendingItem[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const pendingCount = computed(() => pendingItems.value.length)

  async function fetchPending() { /* GET /api/v1/hitl-requests?status=pending */ }

  function handlePendingEvent(payload: { hitl_request_id: string; run_id: string; step_id: string; project_id: string; story_key: string; pr_url?: string; pending_since: string }) {
    /* dedup by hitlRequestId, then push */
  }

  function handleResolvedEvent(hitlRequestId: string) {
    pendingItems.value = pendingItems.value.filter(i => i.hitlRequestId !== hitlRequestId)
  }

  return { pendingItems, pendingCount, isLoading, error, fetchPending, handlePendingEvent, handleResolvedEvent }
})
```

**Badge in `AppSidebar.vue`:**

```vue
<script setup lang="ts">
import Badge from 'primevue/badge'
import { useHITLStore } from '@/stores/hitl'
const hitlStore = useHITLStore()
</script>

<!-- Inside navItems loop, for the Approvals item: -->
<div class="relative">
  <Button icon="pi pi-bell" :label="collapsed ? undefined : 'Approvals'" ... />
  <Badge
    v-if="hitlStore.pendingCount > 0"
    :value="hitlStore.pendingCount"
    severity="danger"
    class="absolute -top-1 -right-1"
  />
</div>
```

**Global SSE wiring in `AppShell.vue`:**

```typescript
import { useSSE } from '@/composables/useSSE'
import { useHITLStore } from '@/stores/hitl'

const hitlStore = useHITLStore()
// Global stream — project_id omitted or set to '*'
const { close } = useSSE('', (eventName, data) => {
  if (eventName === 'hitl.pending') hitlStore.handlePendingEvent(data as ...)
  if (eventName === 'hitl.approved' || eventName === 'hitl.rejected')
    hitlStore.handleResolvedEvent((data as { hitl_request_id: string }).hitl_request_id)
})
onBeforeUnmount(close)
```

**API endpoint used by `fetchPending()`:**

```
GET /api/v1/hitl-requests?status=pending
```

Response shape (list envelope):
```json
{
  "data": [
    {
      "id": "...",
      "run_id": "...",
      "step_id": "...",
      "project_id": "...",
      "project_name": "...",
      "story_key": "S-03",
      "story_title": "...",
      "pr_url": "https://github.com/...",
      "created_at": "2026-02-17T10:00:00Z"
    }
  ],
  "pagination": { "total": 2, "page": 1, "per_page": 20 }
}
```

**`HITLPendingTable` component interface:**

```typescript
// Props
defineProps<{ items: HITLPendingItem[]; loading: boolean }>()
// Emits
defineEmits<{ review: [item: HITLPendingItem] }>()
```

**Route for navigation from list row:**

```typescript
router.push({
  name: 'hitl-approve',
  params: { id: item.projectId, runId: item.runId, stepId: item.stepId },
})
```

This matches the existing route defined in `frontend/src/router/index.ts`:
`path: 'runs/:runId/approve/:stepId', name: 'hitl-approve'`

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.6 | Initial story creation |
