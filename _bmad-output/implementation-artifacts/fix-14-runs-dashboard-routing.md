# Story fix-14: [FRONT] Runs dashboard, sidebar routing, and project runs tab

Status: ready-for-dev

## Story

As a user,
I want working navigation links, a useful dashboard, and access to run history inside projects,
so that I can monitor pipeline activity without hitting 404 pages or landing on an empty screen.

## Acceptance Criteria (BDD)

**AC1: Sidebar "Runs" link navigates to a cross-project runs list page**
- **Given** the user is authenticated
- **When** the user clicks "Runs" in the sidebar
- **Then** the browser navigates to `/runs` and a runs list page is rendered showing recent runs across all projects
- **And** each run row shows project name, story key, status badge, progress, and started-at timestamp
- **And** clicking a run row navigates to `/runs/:id?projectId=...` (existing `RunDetailView`)

**AC2: Sidebar "Settings" link navigates to user profile instead of 404**
- **Given** the user is authenticated
- **When** the user clicks "Settings" in the sidebar
- **Then** the browser navigates to `/profile` (existing `ProfileView`)
- **And** no 404 page is shown

**AC3: Dashboard shows recent runs across all projects**
- **Given** the user is authenticated and navigates to `/`
- **When** the dashboard loads
- **Then** the page renders a "Recent Runs" section listing the 10 most recent runs across all accessible projects
- **And** each run shows: project name, story key, status badge, progress bar, started-at relative timestamp
- **And** clicking a run navigates to its detail page

**AC4: Dashboard shows pending approvals count with link**
- **Given** there are one or more pending HITL approval requests
- **When** the dashboard loads
- **Then** a "Pending Approvals" stat card shows the count of pending items
- **And** the card links to `/approvals`

**AC5: Dashboard shows projects quick-access list**
- **Given** the user has one or more projects
- **When** the dashboard loads
- **Then** a "Projects" section lists up to 5 projects with their names and a link to each project's overview tab

**AC6: Project detail has a "Runs" tab listing project runs**
- **Given** the user is on a project detail page
- **When** the user clicks the "Runs" tab
- **Then** the browser navigates to `/projects/:id/runs`
- **And** a paginated table of runs for that project is shown
- **And** each row shows: run ID (short), story key, status badge, progress, started-at, duration
- **And** clicking a row navigates to the run detail page

**AC7: Empty states are handled gracefully**
- **Given** no runs exist yet for a project (or globally)
- **When** the runs list or dashboard recent runs section loads
- **Then** an empty state message is shown ("No runs yet") — no error, no blank space

**AC8: Loading and error states are handled**
- **Given** the API call is in flight
- **When** the runs list renders
- **Then** skeleton placeholders are shown
- **And** if the API returns an error, an inline error message with a Retry button is shown

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Fix sidebar routing — remap "Runs" and "Settings" links (AC: #1, #2)
  - [ ] In `frontend/src/ui/layout/AppSidebar.vue`, change the "Settings" nav item route from `/settings` to `/profile`
  - [ ] Change the "Settings" label/icon to remain "Settings" pointing to `/profile` (profile page already exists as `ProfileView`)
  - [ ] Ensure the "Runs" nav item route stays `/runs` (will be registered in Task 2)

- [ ] [FRONT] Task 2: Create `RunsView.vue` — cross-project runs list page (AC: #1, #7, #8)
  - [ ] Create `frontend/src/views/RunsView.vue`
  - [ ] Fetch runs using the composable created in Task 4 (`useRecentRuns`) with no projectId filter
  - [ ] Render a `DataTable` with columns: Project, Story Key, Status, Progress, Started At
  - [ ] Status column: `Tag` component with severity mapping (`running` → `info`, `completed` → `success`, `failed` → `danger`, `pending` → `secondary`, `paused` → `warn`, `cancelled` → `warn`)
  - [ ] Progress column: `ProgressBar` with `:value="run.progress"`
  - [ ] Clicking a row routes to `{ name: 'run-detail', params: { id: run.id }, query: { projectId: run.project_id } }`
  - [ ] Show `EmptyState` when no runs exist
  - [ ] Show skeleton rows while loading
  - [ ] Show inline `Message` severity="error" with Retry button on API error

- [ ] [FRONT] Task 3: Register `/runs` route in router (AC: #1)
  - [ ] In `frontend/src/router/index.ts`, import `RunsView` and add route: `{ path: '/runs', name: 'runs', component: RunsView, meta: { requiresAuth: true } }`
  - [ ] Place it after the `/approvals` route for logical grouping

- [ ] [FRONT] Task 4: Create `useRecentRuns` composable (AC: #1, #3)
  - [ ] Create `frontend/src/features/runs/composables/useRecentRuns.ts`
  - [ ] Accept optional `projectId?: string` and `limit?: number` (default 10)
  - [ ] When `projectId` is provided, call `GET /projects/{projectId}/runs` with `{ params: { path: { projectId }, query: { per_page: limit, page: 1 } } }`
  - [ ] When `projectId` is not provided, fetch the first page from a global runs endpoint — use `GET /runs` if it exists in the spec, otherwise fall back to fetching runs without a project filter using the generated client (check OpenAPI spec; if no global endpoint exists, document this limitation and fetch from the first available project)
  - [ ] Return `{ runs, isLoading, error, refresh }` using `useAsyncAction`
  - [ ] Call `execute()` on `onMounted`

- [ ] [FRONT] Task 5: Implement `DashboardView.vue` with real content (AC: #3, #4, #5, #7, #8)
  - [ ] Replace the placeholder `DashboardView.vue` with a full implementation
  - [ ] Section 1 — "Recent Runs": use `useRecentRuns()` (no projectId, limit=10); render a compact DataTable (same columns as `RunsView`); show empty state if none
  - [ ] Section 2 — "Pending Approvals" stat card: use `useHITLStore().pendingCount`; call `hitlStore.fetchPending()` on mounted; show count badge; link to `/approvals` via router-link or Button
  - [ ] Section 3 — "Projects": use `useProjects()` composable (already exists at `frontend/src/composables/useProjects.ts`); fetch page 1 with per_page=5; render a simple list of project names as clickable links to `{ name: 'project-overview', params: { id: project.id } }`
  - [ ] Page header: `<h1>Dashboard</h1>` with greeting (static: "Welcome back")
  - [ ] Layout: 3-column stat cards row at top (Pending Approvals, total run count if computable, projects count), then 2-column grid: Recent Runs (wide) + Projects list (narrow)

- [ ] [FRONT] Task 6: Add "Runs" tab to `ProjectDetailView` and create `ProjectRunsView.vue` (AC: #6, #7, #8)
  - [ ] In `frontend/src/views/ProjectDetailView.vue`, add a new tab entry to the `tabs` array: `{ label: 'Runs', icon: 'pi pi-play', route: 'project-runs' }`
  - [ ] Insert it between "Board" (index 1) and "Pipeline" (index 2) for visual grouping
  - [ ] Create `frontend/src/views/ProjectRunsView.vue`
  - [ ] It reads `projectId` from the parent `route.params.id` (already provided by `ProjectDetailView`)
  - [ ] Uses `useRecentRuns({ projectId, limit: 20 })` — pass props or inject project id via route
  - [ ] Renders a paginated `DataTable` with columns: Run ID (first 8 chars), Story Key, Status, Progress, Started At, Duration
  - [ ] Duration: compute from `started_at` to `completed_at` (or "running..." if no completed_at and status is running) using `date-fns` `differenceInSeconds` (same helper as `RunDetailView`)
  - [ ] Clicking a row routes to `{ name: 'run-detail', params: { id: run.id }, query: { projectId: run.project_id } }`
  - [ ] Register the child route in `frontend/src/router/index.ts` under `/projects/:id` children: `{ path: 'runs', name: 'project-runs', component: () => import('@/views/ProjectRunsView.vue') }`

- [ ] [FRONT] Task 7: Add unit tests for `useRecentRuns` composable (AC: #1, #3, #6)
  - [ ] Create `frontend/src/features/runs/__tests__/useRecentRuns.spec.ts`
  - [ ] Test: fetches with projectId when provided
  - [ ] Test: fetches without projectId (global) when not provided
  - [ ] Test: sets isLoading true during fetch, false after
  - [ ] Test: populates runs on success
  - [ ] Test: sets error on API failure

## Dev Notes

### Dependencies

| Dependency | Location | Notes |
|-----------|----------|-------|
| `useAsyncAction` | `frontend/src/composables/useAsyncAction.ts` | loading + error wrapper |
| `useHITLStore` | `frontend/src/stores/hitl.ts` | `pendingCount`, `fetchPending()` |
| `useProjects` | `frontend/src/composables/useProjects.ts` | projects list for dashboard |
| `useRunsStore` | `frontend/src/stores/runs.ts` | circuit breaker state (for `ProjectDetailView`) |
| `apiClient` | `frontend/src/api/client.ts` | typed openapi-fetch client |
| `date-fns` | already in package.json | `differenceInSeconds`, `formatDistanceToNow` |
| `RunDetailView` | `frontend/src/views/RunDetailView.vue` | run detail page (navigation target) |
| `ProfileView` | `frontend/src/views/ProfileView.vue` | already exists at route `/profile` |

### File Paths

| File | Change |
|------|--------|
| `frontend/src/ui/layout/AppSidebar.vue` | Remap "Settings" route to `/profile` |
| `frontend/src/router/index.ts` | Add `/runs` route; add `project-runs` child route |
| `frontend/src/views/RunsView.vue` | New — cross-project runs list |
| `frontend/src/views/DashboardView.vue` | Replace placeholder with full implementation |
| `frontend/src/views/ProjectRunsView.vue` | New — project-scoped runs tab content |
| `frontend/src/views/ProjectDetailView.vue` | Add "Runs" tab entry to `tabs` array |
| `frontend/src/features/runs/composables/useRecentRuns.ts` | New — shared runs fetch composable |
| `frontend/src/features/runs/__tests__/useRecentRuns.spec.ts` | New — unit tests |

### API Calls

| Endpoint | Trigger | Response schema |
|----------|---------|-----------------|
| `GET /projects/{projectId}/runs?page=1&per_page=N` | `ProjectRunsView` mount, `useRecentRuns` with projectId | `RunList` → `{ data: Run[], pagination: Pagination }` |
| `GET /hitl-requests?status=pending` | `DashboardView` mount (via `hitlStore.fetchPending()`) | Already implemented in `useHITLStore` |
| `GET /projects?page=1&per_page=5` | `DashboardView` mount (via `useProjects`) | Already implemented |

Note: The OpenAPI spec has no global `GET /runs` endpoint (only `GET /projects/{projectId}/runs` and `GET /stories/{storyId}/runs`). For the cross-project `RunsView`, fetch runs from `GET /projects/{projectId}/runs` by iterating over the user's accessible projects — or, if the project list is unavailable, show a notice and link to individual project pages. A simpler fallback: `RunsView` can render the HITL pending items list as a proxy for "active runs requiring attention" and add a note that per-project run history is available in each project's Runs tab. **Recommended approach**: fetch the project list first (up to 5 projects), then fan out a `GET /projects/{projectId}/runs?per_page=10` for each, merge and sort by `created_at` desc, take the top 10. This avoids requiring a new API endpoint and is feasible with the existing spec.

### Architecture / Component Structure

**`useRecentRuns.ts` signature:**

```typescript
export function useRecentRuns(options?: { projectId?: string; limit?: number }) {
  // Returns { runs, isLoading, error, refresh }
  // When projectId provided: GET /projects/{projectId}/runs?per_page=limit
  // When no projectId: fetch project list, fan out per-project, merge top N by created_at
}
```

**`DashboardView.vue` template wireframe:**

```
┌─────────────────────────────────────────────────────────┐
│ Dashboard                         [Welcome back]         │
├──────────────┬──────────────┬──────────────────────────┤
│  Pending     │  Active Runs  │  Projects                │
│  Approvals   │  (computed)   │  (count)                 │
│  [N] →       │              │                          │
├──────────────┴──────────────┴──────────────────────────┤
│                                                         │
│  Recent Runs (last 10 across all projects)              │
│  ┌─────────┬──────────┬────────┬──────────┬──────────┐ │
│  │ Project │ Story    │ Status │ Progress │ Started  │ │
│  ├─────────┼──────────┼────────┼──────────┼──────────┤ │
│  │ ...     │ S-12     │ ●run   │ ████░░   │ 2m ago   │ │
│  └─────────┴──────────┴────────┴──────────┴──────────┘ │
│                                                         │
│  Projects (quick access)                                │
│  • Project Alpha  →                                     │
│  • Project Beta   →                                     │
└─────────────────────────────────────────────────────────┘
```

**`ProjectRunsView.vue` template wireframe:**

```
┌─────────────────────────────────────────────────────────┐
│ [← Projects]  My Project                                │
│ [Overview] [Board] [Runs] [Pipeline] [Templates] [...]  │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Run History                                            │
│  ┌────────┬──────────┬────────┬──────────┬────────────┐ │
│  │ Run ID │ Story    │ Status │ Progress │ Duration   │ │
│  ├────────┼──────────┼────────┼──────────┼────────────┤ │
│  │ a1b2c3 │ S-12     │ ●done  │ 100%     │ 4m 12s     │ │
│  │ d4e5f6 │ S-11     │ ●fail  │  60%     │ 2m 05s     │ │
│  └────────┴──────────┴────────┴──────────┴────────────┘ │
│                          [paginator]                    │
└─────────────────────────────────────────────────────────┘
```

**Status severity mapping (reuse across all run views):**

```typescript
const runStatusSeverity: Record<string, 'info' | 'success' | 'warn' | 'danger' | 'secondary'> = {
  pending: 'secondary',
  running: 'info',
  paused: 'warn',
  completed: 'success',
  failed: 'danger',
  cancelled: 'warn',
}
```

This mapping already exists in `RunDetailView.vue` — extract it to a shared util at `frontend/src/utils/runStatus.ts` to avoid duplication across `RunsView`, `ProjectRunsView`, and `DashboardView`.

**Sidebar "Settings" fix — minimal change:**

```typescript
// frontend/src/ui/layout/AppSidebar.vue — navItems array
// Before:
{ label: 'Settings', icon: 'pi pi-cog', route: '/settings' },
// After:
{ label: 'Settings', icon: 'pi pi-cog', route: '/profile' },
```

The `isActive` function already uses `route.path.startsWith(itemRoute)` — since `/profile` is a flat route, this works correctly.

**`ProjectDetailView.vue` tabs array change:**

```typescript
// Before:
const tabs = [
  { label: 'Overview', icon: 'pi pi-home', route: 'project-overview' },
  { label: 'Board', icon: 'pi pi-th-large', route: 'project-board' },
  { label: 'Pipeline', icon: 'pi pi-cog', route: 'project-pipeline' },
  { label: 'Templates', icon: 'pi pi-file', route: 'project-templates' },
  { label: 'Costs', icon: 'pi pi-dollar', route: 'project-costs' },
  { label: 'Notifications', icon: 'pi pi-bell', route: 'project-notifications' },
]

// After (insert 'Runs' at index 2, after 'Board'):
const tabs = [
  { label: 'Overview', icon: 'pi pi-home', route: 'project-overview' },
  { label: 'Board', icon: 'pi pi-th-large', route: 'project-board' },
  { label: 'Runs', icon: 'pi pi-play', route: 'project-runs' },       // <-- new
  { label: 'Pipeline', icon: 'pi pi-cog', route: 'project-pipeline' },
  { label: 'Templates', icon: 'pi pi-file', route: 'project-templates' },
  { label: 'Costs', icon: 'pi pi-dollar', route: 'project-costs' },
  { label: 'Notifications', icon: 'pi pi-bell', route: 'project-notifications' },
]
```

The `activeIndex` computed already handles unknown routes by defaulting to 0 — no change needed there.

**Router additions:**

```typescript
// frontend/src/router/index.ts

// 1. Top-level route (after /approvals):
import RunsView from '@/views/RunsView.vue'
// ...
{
  path: '/runs',
  name: 'runs',
  component: RunsView,
  meta: { requiresAuth: true },
},

// 2. Child route under /projects/:id (after 'project-board'):
{
  path: 'runs',
  name: 'project-runs',
  component: () => import('@/views/ProjectRunsView.vue'),
},
```

### Testing Requirements

**Manual verification:**
1. Click "Runs" in sidebar → `/runs` renders a list (or empty state)
2. Click "Settings" in sidebar → `/profile` renders (no 404)
3. Visit `/` → dashboard has Recent Runs, Pending Approvals card, Projects list
4. Navigate to any project → "Runs" tab visible → click → `/projects/:id/runs` shows run history
5. Click any run row (in any list) → navigates to run detail page

**Unit tests (Vitest):**
- `useRecentRuns.spec.ts` — covers all branches: with/without projectId, loading states, error path, empty results

**E2E (Playwright) — optional for this fix:**
- If E2E coverage is desired, add to `frontend/e2e/tests/runs.spec.ts`:
  - "sidebar Runs link navigates to runs list page"
  - "sidebar Settings link navigates to profile page"
  - "dashboard renders recent runs section"
  - "project runs tab shows run history"

## Dev Agent Record

**Agent:** Claude Opus 4.6
**Branch:** feat/fix-14-runs-dashboard-routing
**Date:** 2026-02-22

### Files Changed

| File | Change |
|------|--------|
| `frontend/src/ui/layout/AppSidebar.vue` | Remapped "Settings" route from `/settings` to `/profile` |
| `frontend/src/router/index.ts` | Added `/runs` top-level route; added `project-runs` child route under `/projects/:id` |
| `frontend/src/utils/runStatus.ts` | New — shared run status severity mapping |
| `frontend/src/features/runs/composables/useRecentRuns.ts` | New — composable for fetching recent runs (project-scoped or cross-project fan-out) |
| `frontend/src/views/RunsView.vue` | New — cross-project runs list page |
| `frontend/src/views/DashboardView.vue` | Replaced placeholder with full dashboard (recent runs, pending approvals, projects) |
| `frontend/src/views/ProjectRunsView.vue` | New — project-scoped runs tab content with duration column |
| `frontend/src/views/ProjectDetailView.vue` | Added "Runs" tab to tabs array |
| `frontend/src/views/RunDetailView.vue` | Refactored to import shared `runStatusSeverity` from utils |
| `frontend/src/features/runs/__tests__/useRecentRuns.spec.ts` | New — 8 unit tests covering all composable branches |
| `frontend/src/api/schema.d.ts` | Regenerated from OpenAPI spec |

### Notes

- No global `GET /runs` endpoint exists in the OpenAPI spec. The cross-project runs view uses a fan-out strategy: fetch up to 5 projects, then concurrently fetch runs per project, merge by `created_at` desc, and take top N.
- All 523 tests pass (62 test files). ESLint and TypeScript type-check clean.

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | story-writer | Initial story created |
| 2026-02-22 | dev-agent | Implementation complete — all ACs addressed |
