# Story F-2.4: [FRONT] Fix cost dashboard API call with empty project ID

Status: ready-for-dev

## Story

As a user viewing the cost dashboard,
I want to see accurate cost data for my project,
so that I can track spending on pipeline runs.

## Acceptance Criteria (BDD)

**AC1: Cost API called with correct project ID**
- **Given** user navigates to `/projects/{id}/costs`
- **When** the cost dashboard loads
- **Then** API calls to `/projects/{projectId}/costs/*` include the correct project ID from the route params (`:id`)

**AC2: Run cost API called with correct project ID**
- **Given** user navigates to `/runs/{id}` (from any navigation path)
- **When** the run detail page loads
- **Then** the API call to `/projects/{projectId}/runs/{runId}/costs` includes the correct project ID
- **And** the URL does not contain an empty segment (e.g. `/projects//runs/...`)

**AC3: Cost data displays when available**
- **Given** the project has runs with cost data
- **When** the cost dashboard loads
- **Then** total cost, average cost per story, and cost over time are displayed

**AC4: Empty state when no costs**
- **Given** the project has no runs with cost data
- **When** the cost dashboard loads
- **Then** a friendly "No cost data yet" message is shown (not an error)

**AC5: Period filters work**
- **Given** cost data exists
- **When** user toggles between 7d and 30d period
- **Then** displayed costs reflect the selected time range

**AC6: Run detail navigation from cost dashboard passes project ID**
- **Given** user is on the cost dashboard
- **When** user clicks a run row to navigate to run detail
- **Then** the run detail page receives the project ID and loads cost data correctly

## Tasks / Subtasks

### Task 1 — Fix `RunDetailView.vue`: make `projectId` reactive and source it from the run data

**File:** `frontend/src/views/RunDetailView.vue`

**Root cause:** `projectId` is sourced from `route.query.projectId` (line 22), which is empty when navigating without a query param. It is then passed as a static `.value` snapshot to `useRunDetail` and `useRunCosts` (lines 27, 31–35), so even if the value were fixed reactively, the composables would not update.

**Sub-tasks:**
- 1a. Change `projectId` to be derived from the loaded `run` object once it resolves: `computed(() => run.value?.project_id ?? '')`. The `Run` API response includes `project_id`.
- 1b. Update `useRunDetail` call: since `runId` is already a `computed`, also make `projectId` a `Ref<string>` or pass the computed directly. Check `useRunDetail` signature to confirm it accepts a reactive value or plain string. Adjust the call so `projectId` is not snapshotted at call time.
- 1c. Update `useRunCosts` call: `useRunCosts` in `frontend/src/features/runs/composables/useRunCosts.ts` currently accepts plain `string` args. Change signature to accept `Ref<string>` or use a `watch`-based approach so it re-fetches when `projectId` becomes available (after run loads). Alternatively, trigger `fetchCosts` imperatively once `run.value?.project_id` is known.
- 1d. Update `handlePause`, `handleResume`, `handleCancelConfirm` to use the reactive computed `projectId` (they already call `.value` so this should be transparent once the computed is fixed).

### Task 2 — Fix `CostDashboardView.vue`: guard against empty project ID before calling `useCosts`

**File:** `frontend/src/views/CostDashboardView.vue`

**Current code (line 22):**
```typescript
const projectId = route.params.id as string
```

**Issue:** `route.params.id` is read synchronously at setup time. For a lazy-loaded child route, it should already be available, but the cast to `string` hides an `undefined` or empty string edge case if the route param is missing.

**Sub-tasks:**
- 2a. Add a guard: if `!projectId`, log a warning and avoid calling `useCosts` — or pass a default that prevents the API call from firing (e.g. skip `onMounted` fetch when `projectId` is falsy inside the composable).
- 2b. In `useCosts` composable (`frontend/src/composables/useCosts.ts`), add a guard at the top of `fetchAll` and `fetchAgentCosts`: if `projectId` is empty/falsy, set an error message `'No project ID available'` and return early without calling the API.

### Task 3 — Fix `CostDashboardView.vue`: pass `projectId` when navigating to run detail

**File:** `frontend/src/views/CostDashboardView.vue`

**Current code (line 50):**
```typescript
function onRunNavigate(runId: string) {
  router.push({ name: 'run-detail', params: { id: runId } })
}
```

**Issue:** No `projectId` is forwarded. `RunDetailView` previously relied on `route.query.projectId` to get the project ID for the cost API call.

**Sub-tasks:**
- 3a. Pass `projectId` as a query param when navigating:
  ```typescript
  function onRunNavigate(runId: string) {
    router.push({ name: 'run-detail', params: { id: runId }, query: { projectId } })
  }
  ```
  **Note:** This is a stopgap fix. Task 1 (deriving `projectId` from `run.project_id`) is the proper long-term fix. Both fixes are complementary — Task 1 handles direct navigation; Task 3 handles navigation from the cost dashboard.

### Task 4 — Update unit tests for `useCosts` composable

**File:** `frontend/src/composables/__tests__/useCosts.spec.ts`

**Sub-tasks:**
- 4a. Add a test: `fetchAll does not call the API when projectId is empty string`.
- 4b. Add a test: `fetchAll sets error to "No project ID available" when projectId is empty`.
- 4c. Add a test: `fetchAgentCosts does not call the API when projectId is empty`.

### Task 5 — Update unit tests for `useRunCosts` composable

**File:** `frontend/src/features/runs/__tests__/useRunCosts.spec.ts`

**Sub-tasks:**
- 5a. If `useRunCosts` signature is changed to accept `Ref<string>`, update existing test calls to pass `ref('proj-1')` instead of `'proj-1'`.
- 5b. Add a test: `does not call the API when projectId ref is empty, and calls it once projectId becomes available`.

## Dev Notes

- Priority: P2
- **Root cause summary:**
  - `RunDetailView.vue` reads `projectId` from `route.query.projectId` (a query param), but nothing passes this query param when navigating to `/runs/:id`. Result: `projectId` is always `''`, producing `/projects//runs/.../costs` in the API URL.
  - `CostDashboardView.vue` reads `route.params.id` correctly (the route param is `:id` per `router/index.ts` line 52), so `useCosts` itself is not the primary broken path — but it lacks a guard against an empty `projectId`.
- **Route param name:** The project route is `/projects/:id` (not `:projectId`). The `CostDashboardView` correctly reads `route.params.id`. No rename needed.
- **`useRunCosts` signature change:** The composable currently accepts `projectId: string` and `runId: string` as plain strings. To support reactive late-binding (project ID known only after run loads), consider changing to `projectId: Ref<string> | string` using `toRef`/`toValue` (Vue 3.3+).
- **Also related to F-2-1** — even after fixing the API call, seed data needs correct UUIDs to show cost data. Fix this story independently of seed data concerns.
- **No OpenAPI spec changes needed** — this is a pure frontend wiring fix.
