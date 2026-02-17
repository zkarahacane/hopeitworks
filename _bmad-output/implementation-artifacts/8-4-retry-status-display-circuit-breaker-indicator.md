# Story 8.4: [FRONT] Retry Status Display + Circuit Breaker Indicator

Status: ready-for-dev

## Story

As a developer, I want the run timeline to show retry attempts grouped under their original failed step, and I want a circuit breaker banner on project pages, so that I can understand what happened when a story failed and whether the project is currently locked.

## Acceptance Criteria (BDD)

**AC1: Retry steps are grouped under their parent in the run timeline**
- **Given** a run with a failed step and one or more retry steps (`parent_step_id` set)
- **When** the RunTimeline renders the step list
- **Then** retry steps appear indented under their parent step
- **And** each retry entry is labeled "Retry #1 (incremental)" or "Retry #2 (full)" based on `retry_count` and `retry_type`
- **And** the original failed step header remains visible above its retries

**AC2: Retry entries show expandable error context**
- **Given** a retry step has `error_context` or `log_tail` in its associated metadata
- **When** the user clicks the expand toggle on a retry entry
- **Then** a collapsible panel appears with the error context and log tail from the failed parent step
- **And** long log tails are truncated to the last 20 lines with a "Show more" link

**AC3: CircuitBreakerBanner renders when active**
- **Given** a project has `circuit_breaker_active = true` in its data
- **When** the `ProjectDetailView` renders
- **Then** a `CircuitBreakerBanner.vue` component is shown at the top of the page
- **And** the banner is red/danger severity with the message "Circuit breaker active — all pipeline runs are paused"
- **And** the banner is NOT shown when `circuit_breaker_active = false`

**AC4: Reset button (admin only) triggers confirmation and API call**
- **Given** the circuit breaker banner is visible and the current user has role `admin`
- **When** the user clicks the "Reset" button
- **Then** a PrimeVue `ConfirmDialog` appears asking for confirmation
- **And** on confirm, a `POST /api/v1/projects/{id}/circuit-breaker/reset` request is sent
- **And** on success, the banner disappears and a success `Toast` is shown
- **And** non-admin users do not see the Reset button

**AC5: Circuit breaker reset API contract defined in OpenAPI**
- **Given** the `api/openapi.yaml` spec
- **When** the circuit breaker reset endpoint is reviewed
- **Then** `POST /api/v1/projects/{id}/circuit-breaker/reset` is defined with a 204 No Content response
- **And** `circuit_breaker_active` field is present on the `Project` schema

**AC6: SSE event `circuit_breaker.triggered` shows the banner reactively**
- **Given** the user is viewing a project page
- **When** a `circuit_breaker.triggered` SSE event arrives for that project
- **Then** the circuit breaker banner appears without a page reload
- **And** the `runs` Pinia store updates `circuitBreakerActive` reactively

**AC7: RunStep schema updated with retry fields**
- **Given** `api/openapi.yaml` is updated and `npm run generate-api` is run
- **When** `RunStep` TypeScript types are regenerated
- **Then** `retry_count`, `retry_type`, and `parent_step_id` fields are available on the `RunStep` type

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Update OpenAPI spec with retry fields and circuit breaker endpoint (AC: #5, #7)
  - [ ] Add `retry_count: integer`, `retry_type: string (enum: incremental, full)`, `parent_step_id: string (format: uuid)` to `RunStep` schema in `api/openapi.yaml`
  - [ ] Add `circuit_breaker_active: boolean` field to `Project` schema
  - [ ] Add `POST /api/v1/projects/{id}/circuit-breaker/reset` endpoint with 204 response and 403 for non-admin
  - [ ] Run `cd frontend && npm run generate-api` to regenerate TypeScript types
  - [ ] Verify `RunStep` type in `frontend/src/api/generated/` has the new fields

- [ ] [FRONT] Task 2: Implement retry step grouping logic in `useRunTimeline` composable (AC: #1)
  - [ ] Create `frontend/src/features/runs/composables/useRunTimeline.ts`
  - [ ] Accept `steps: RunStep[]` as input
  - [ ] Return `groupedSteps: ComputedRef<StepGroup[]>` where `StepGroup = { root: RunStep; retries: RunStep[] }`
  - [ ] Group by: `retries` are steps where `parent_step_id` matches `root.id`; root steps have `parent_step_id == null`
  - [ ] Sort groups by `root.step_order`, retries within group by `retry_count`

- [ ] [FRONT] Task 3: Implement `RunTimeline.vue` with retry grouping (AC: #1, #2)
  - [ ] Create `frontend/src/features/runs/RunTimeline.vue`
  - [ ] Use PrimeVue `Timeline` component for the step list
  - [ ] For each root step: render step name, status badge (`StatusBadge.vue`), timestamps
  - [ ] For each retry in a group: render indented `RetryStepEntry.vue` sub-component
  - [ ] Retry label format: `"Retry #{{retry_count}} ({{retry_type}})"` e.g. `"Retry #1 (incremental)"`
  - [ ] Create `frontend/src/features/runs/RetryStepEntry.vue` — shows label, status, expand toggle

- [ ] [FRONT] Task 4: Implement expandable error context on retry entries (AC: #2)
  - [ ] In `RetryStepEntry.vue`: add an expand/collapse toggle button (PrimeVue `Button` icon-only)
  - [ ] On expand: show a `<pre>` block with `error_context` and log tail lines (last 20; "Show more" if truncated)
  - [ ] `error_context` and `log_tail` come from the retry step's metadata or from the parent step's `error_message` and `log_tail` fields on `RunStep`
  - [ ] Use `ref(false)` for `isExpanded` local state

- [ ] [FRONT] Task 5: Implement `CircuitBreakerBanner.vue` component (AC: #3, #4)
  - [ ] Create `frontend/src/features/projects/CircuitBreakerBanner.vue`
  - [ ] Props: `projectId: string`, `isAdmin: boolean`
  - [ ] Emit: `reset` — fired after successful reset API call
  - [ ] Show a PrimeVue `Message` with `severity="error"` when `circuitBreakerActive` is true
  - [ ] Show "Reset" `Button` only when `isAdmin = true`
  - [ ] On Reset click: use `useConfirm()` to show `ConfirmDialog` with warning message
  - [ ] On confirm: call `useAsyncAction` wrapping `apiClient.POST('/api/v1/projects/{id}/circuit-breaker/reset')`
  - [ ] On success: emit `reset`, show success `Toast`
  - [ ] On error: show error `Toast`

- [ ] [FRONT] Task 6: Integrate `CircuitBreakerBanner` into `ProjectDetailView` and update runs store (AC: #3, #4, #6)
  - [ ] In `frontend/src/views/ProjectDetailView.vue`: import and render `<CircuitBreakerBanner>` at the top, passing `project.circuit_breaker_active` and `isAdmin` from auth store
  - [ ] In `frontend/src/stores/runs.ts`: add `circuitBreakerActive = ref(false)` state
  - [ ] Add `handleSSEEvent` handler for `circuit_breaker.triggered`: set `circuitBreakerActive.value = true`
  - [ ] Update `useSSE` dispatch in the store to handle this new event type
  - [ ] In `CircuitBreakerBanner.vue`: read `circuitBreakerActive` from the runs store (or receive as prop from `ProjectDetailView`)

- [ ] [FRONT] Task 7: Write unit tests (AC: #1, #2, #3, #4)
  - [ ] Create `frontend/src/features/runs/__tests__/useRunTimeline.spec.ts` — test grouping logic with steps having parent_step_id
  - [ ] Create `frontend/src/features/runs/__tests__/RunTimeline.spec.ts` — test retry labels rendered correctly
  - [ ] Create `frontend/src/features/projects/__tests__/CircuitBreakerBanner.spec.ts` — test banner visibility, Reset button admin-only, confirm dialog flow
  - [ ] Run `npm run type-check` and `npm run lint` — must pass

## Dev Notes

### Dependencies

**Story 8-2 (IncrementalRetryAction — this wave):** Provides `retry_count`, `retry_type`, and `parent_step_id` on `RunStep`. The frontend reads these fields to group steps in the timeline. API types are generated from the OpenAPI spec updated in Task 1 of this story.

**Story 8-3 (Circuit Breaker API — wave 11):** The `POST /projects/{id}/circuit-breaker/reset` backend handler does not exist yet. This story defines the API contract in `openapi.yaml` and implements the frontend UI. The backend implementation comes in wave 11. The frontend should handle 501 Not Implemented gracefully (show an error toast) until the backend is deployed.

### Architecture Requirements

**Frontend conventions:**
- `RunTimeline.vue` and `RetryStepEntry.vue` live in `frontend/src/features/runs/` — feature-specific components
- `CircuitBreakerBanner.vue` lives in `frontend/src/features/projects/` — project feature component
- Grouping logic in `useRunTimeline.ts` composable — no business logic in `.vue` files
- All API calls via `useAsyncAction` wrapping `apiClient.*`
- SSE events update Pinia stores; components read stores reactively
- Admin check: read current user role from `useAuthStore().currentUser.role === 'admin'`

**OpenAPI contract (define now, backend implements wave 11):**
```yaml
# In api/openapi.yaml — add to paths:
/api/v1/projects/{id}/circuit-breaker/reset:
  post:
    operationId: resetCircuitBreaker
    summary: Reset circuit breaker for a project
    tags: [projects]
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    responses:
      '204':
        description: Circuit breaker reset successfully
      '403':
        $ref: '#/components/responses/Forbidden'
      '404':
        $ref: '#/components/responses/NotFound'
```

### File Paths (exact)

```
api/openapi.yaml                                                     # Add retry fields + CB endpoint
frontend/src/api/generated/                                          # Regenerated — never edit manually
frontend/src/features/runs/RunTimeline.vue                           # New component
frontend/src/features/runs/RetryStepEntry.vue                        # New sub-component
frontend/src/features/runs/composables/useRunTimeline.ts             # Grouping logic
frontend/src/features/runs/__tests__/useRunTimeline.spec.ts          # Composable tests
frontend/src/features/runs/__tests__/RunTimeline.spec.ts             # Component tests
frontend/src/features/projects/CircuitBreakerBanner.vue              # New component
frontend/src/features/projects/__tests__/CircuitBreakerBanner.spec.ts
frontend/src/stores/runs.ts                                          # Add circuitBreakerActive + SSE handler
frontend/src/views/ProjectDetailView.vue                             # Integrate CircuitBreakerBanner
```

### Technical Specifications

**`StepGroup` type (in composable):**
```typescript
interface StepGroup {
  root: RunStep
  retries: RunStep[]
}
```

**`useRunTimeline` composable:**
```typescript
// frontend/src/features/runs/composables/useRunTimeline.ts
import { computed } from 'vue'
import type { RunStep } from '@/api/generated/schema'

export interface StepGroup {
  root: RunStep
  retries: RunStep[]
}

export function useRunTimeline(steps: Ref<RunStep[]>) {
  const groupedSteps = computed<StepGroup[]>(() => {
    const rootSteps = steps.value
      .filter(s => !s.parent_step_id)
      .sort((a, b) => a.step_order - b.step_order)

    return rootSteps.map(root => ({
      root,
      retries: steps.value
        .filter(s => s.parent_step_id === root.id)
        .sort((a, b) => (a.retry_count ?? 0) - (b.retry_count ?? 0)),
    }))
  })

  return { groupedSteps }
}
```

**Retry label helper:**
```typescript
// In RetryStepEntry.vue or utils
function retryLabel(step: RunStep): string {
  const num = step.retry_count ?? 1
  const type = step.retry_type ?? 'incremental'
  return `Retry #${num} (${type})`
}
```

**`CircuitBreakerBanner.vue` structure:**
```vue
<script setup lang="ts">
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

const props = defineProps<{
  projectId: string
  isAdmin: boolean
}>()
const emit = defineEmits<{ reset: [] }>()

const confirm = useConfirm()
const toast = useToast()

const { execute: doReset, isLoading } = useAsyncAction(async () => {
  await apiClient.POST('/api/v1/projects/{id}/circuit-breaker/reset', {
    params: { path: { id: props.projectId } },
  })
  emit('reset')
  toast.add({ severity: 'success', summary: 'Circuit breaker reset', life: 3000 })
})

function handleReset() {
  confirm.require({
    message: 'This will allow new pipeline runs to start. Continue?',
    header: 'Reset Circuit Breaker',
    icon: 'pi pi-exclamation-triangle',
    rejectLabel: 'Cancel',
    acceptLabel: 'Reset',
    acceptClass: 'p-button-danger',
    accept: () => doReset(),
  })
}
</script>

<template>
  <Message severity="error" :closable="false" class="mb-4">
    <div class="flex items-center justify-between w-full">
      <span>Circuit breaker active — all pipeline runs are paused.</span>
      <Button
        v-if="isAdmin"
        label="Reset"
        severity="danger"
        size="small"
        :loading="isLoading"
        @click="handleReset"
      />
    </div>
  </Message>
</template>
```

**`runs.ts` store additions:**
```typescript
// Add to existing defineStore('runs', () => { ... })
const circuitBreakerActive = ref(false)

function handleSSEEvent(event: SSEEvent) {
  // ... existing handlers ...
  if (event.type === 'circuit_breaker.triggered') {
    circuitBreakerActive.value = true
  }
}

return {
  // ... existing exports ...
  circuitBreakerActive,
  handleSSEEvent,
}
```

**SSE event type** (document in code, handled when backend sends it):
- `circuit_breaker.triggered` — payload: `{ project_id: string, triggered_at: string }`
- `circuit_breaker.reset` — payload: `{ project_id: string, reset_at: string }` — set `circuitBreakerActive = false`

**`RetryStepEntry.vue` log truncation:**
```typescript
const MAX_LOG_LINES = 20

const truncatedLog = computed(() => {
  const lines = props.step.log_tail?.split('\n') ?? []
  const isTruncated = lines.length > MAX_LOG_LINES
  return {
    lines: isTruncated ? lines.slice(-MAX_LOG_LINES) : lines,
    isTruncated,
    totalLines: lines.length,
  }
})
```

### Testing Requirements

**`useRunTimeline.spec.ts` — key test cases:**
1. Steps without `parent_step_id` → each becomes a root group with empty `retries`
2. Steps with `parent_step_id` → grouped under matching root, sorted by `retry_count`
3. Mixed: 2 root steps, one with 2 retries — verify correct grouping and sort order
4. Empty steps array → returns empty `groupedSteps`

**`CircuitBreakerBanner.spec.ts` — key test cases:**
1. `isAdmin = false` → Reset button not rendered
2. `isAdmin = true` → Reset button rendered
3. Confirm accepted → `apiClient.POST` called with correct path, `reset` event emitted
4. Confirm rejected → no API call made
5. API error → error toast shown, `reset` not emitted

Use `@vue/test-utils` mount + stub `useConfirm`, `useToast`, and `apiClient`. Mock `apiClient.POST` to return `{ response: { status: 204 } }`.

### References

- `frontend/src/features/runs/RunLaunchButton.vue` — existing runs feature component pattern
- `frontend/src/stores/runs.ts` — existing runs Pinia store (extend, not replace)
- `frontend/src/views/ProjectDetailView.vue` — integration point for CircuitBreakerBanner
- `frontend/src/composables/useAsyncAction.ts` — async action wrapper pattern
- `api/openapi.yaml` — single source of truth; update before generating types
- PrimeVue `Timeline` docs — component for step list rendering
- PrimeVue `ConfirmDialog` / `useConfirm()` docs — confirmation dialog pattern
- Story 8-2 — provides `retry_count`, `retry_type`, `parent_step_id` on the backend RunStep

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
