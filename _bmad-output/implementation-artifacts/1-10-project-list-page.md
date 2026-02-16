# Story 1.10: [FRONT] Project list page

Status: ready-for-dev

## Story

As a user,
I want to see my projects,
So that I can select one to work with.

## Acceptance Criteria (BDD)

**AC1: Authenticated user sees project list**
- **Given** user is authenticated
- **When** they navigate to `/projects`
- **Then** a PrimeVue DataTable displays projects with columns: name, description, created date
- **And** pagination controls appear when total exceeds `per_page`

**AC2: Empty state when no projects exist**
- **Given** user is authenticated and has no projects
- **When** the page loads at `/projects`
- **Then** an empty state component shows a "Create your first project" CTA button
- **And** the DataTable is not rendered

**AC3: Loading state while fetching**
- **Given** user navigates to `/projects`
- **When** the API call is in progress
- **Then** a loading skeleton or spinner is displayed in place of the table

**AC4: Error state on API failure**
- **Given** user navigates to `/projects`
- **When** GET /api/v1/projects returns a non-200 response
- **Then** an error message is displayed with a retry action

**AC5: Row click navigates to project detail**
- **Given** the project list is displayed
- **When** user clicks a project row
- **Then** they are navigated to `/projects/:id`

**AC6: Sidebar "Projects" nav item is active**
- **Given** user is on the `/projects` page
- **When** the sidebar renders
- **Then** the "Projects" navigation item has an active visual state

> **Note on AC vs epic description:** The epic AC references "repo URL" and "provider" columns. The current `Project` schema in `api/openapi.yaml` does not include `repo_url` or `provider` fields. This story implements against the actual schema (name, description, created_at). When those fields are added to the spec (likely in a Git integration story), the DataTable columns should be updated accordingly.

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Upgrade `useProjectsStore` with fetch action and typed state (AC: #1, #3, #4)
  - [ ] Replace the shell store at `frontend/src/stores/projects.ts` with full implementation
  - [ ] Import `Project` and `Pagination` types from generated `api/schema` (or define local types matching the OpenAPI spec if schema.d.ts is not yet generated)
  - [ ] State: `items: Project[]`, `pagination: Pagination | null`, `isLoading: boolean`, `error: string | null`
  - [ ] Action: `fetchProjects(params: { page?: number; per_page?: number; sort_by?: string })` calls `apiClient.GET('/projects', { params: { query } })`, sets `items` and `pagination` on success, sets `error` on failure
  - [ ] Action: `reset()` clears items, pagination, and error

- [ ] [FRONT] Task 2: Create `useProjects` composable at `frontend/src/composables/useProjects.ts` (AC: #1, #3, #4)
  - [ ] Import `useProjectsStore` and `useAsyncAction`
  - [ ] Wrap `store.fetchProjects` in `useAsyncAction` for reactive `isLoading` / `error` / `data`
  - [ ] Expose: `projects` (computed from store.items), `pagination` (computed from store.pagination), `isLoading`, `error`, `fetchProjects(params)`, `retry()`
  - [ ] `retry()` re-executes the last `fetchProjects` call with same params

- [ ] [FRONT] Task 3: Create `ProjectEmptyState.vue` at `frontend/src/features/projects/ProjectEmptyState.vue` (AC: #2)
  - [ ] Create `frontend/src/features/projects/` directory
  - [ ] Props: none (or optional `onCreate: () => void` emit)
  - [ ] Emits: `create` — fired when CTA button is clicked
  - [ ] Render: centered layout with an icon (`pi pi-folder-open`), heading "No projects yet", description text, and a PrimeVue `Button` label "Create your first project" with `severity="success"`
  - [ ] Use Tailwind for centering layout (`flex flex-col items-center justify-center gap-4 py-16`)

- [ ] [FRONT] Task 4: Create `ProjectListTable.vue` at `frontend/src/features/projects/ProjectListTable.vue` (AC: #1, #5)
  - [ ] Props: `projects: Project[]`, `totalRecords: number`, `rows: number`, `loading: boolean`, `first: number`
  - [ ] Emits: `page` (PrimeVue DataTablePageEvent), `row-click` (project: Project)
  - [ ] PrimeVue `DataTable` with `:value="projects"`, `:lazy="true"`, `:paginator="true"`, `:rows="rows"`, `:totalRecords="totalRecords"`, `:loading="loading"`, `:first="first"`, `stripedRows`, `@page="emit('page', $event)"`, `@row-click="emit('row-click', $event.data)"`
  - [ ] Columns: `name` (header: "Name"), `description` (header: "Description"), `created_at` (header: "Created", formatted via `date-fns` `formatDistanceToNow` or `format`)
  - [ ] Name column renders as bold text; description truncated with `max-w-md truncate`
  - [ ] `created_at` column uses a formatting helper (e.g., `formatDate` util or inline `date-fns` call)
  - [ ] Add `rowClass` or cursor pointer style so rows look clickable

- [ ] [FRONT] Task 5: Implement `ProjectsView.vue` at `frontend/src/views/ProjectsView.vue` (AC: #1, #2, #3, #4, #5)
  - [ ] Replace the placeholder content in existing `ProjectsView.vue`
  - [ ] Use `useProjects` composable to fetch data on mount (`onMounted` or immediate watch)
  - [ ] Call `fetchProjects({ page: 1, per_page: 20 })` on mount
  - [ ] Render page header: `<h1>Projects</h1>` with optional "New Project" button (placeholder, no-op for now)
  - [ ] Conditional rendering: loading state (PrimeVue `Skeleton` or `ProgressSpinner`), error state (PrimeVue `Message` with retry button), empty state (`ProjectEmptyState`), data state (`ProjectListTable`)
  - [ ] On `@page` event from DataTable: call `fetchProjects` with new page number
  - [ ] On `@row-click`: `router.push({ name: 'project-detail', params: { id: project.id } })`
  - [ ] Layout: `<div class="flex flex-col gap-6 p-6">`

- [ ] [FRONT] Task 6: Add active state to sidebar navigation (AC: #6)
  - [ ] Update `frontend/src/ui/layout/AppSidebar.vue`
  - [ ] Import `useRoute` from `vue-router`
  - [ ] Compare `route.path` with each nav item's `route` to determine active state
  - [ ] Apply PrimeVue Button `severity` or a distinct Tailwind class (e.g., `!bg-primary-50 !text-primary-700`) to the active item
  - [ ] Active detection: use `route.path === item.route` for exact match or `route.path.startsWith(item.route)` for `/projects/:id` sub-routes (but not `/` matching everything — handle dashboard as exact match)

- [ ] [FRONT] Task 7: Create date formatting utility (AC: #1)
  - [ ] Create `frontend/src/utils/formatDate.ts`
  - [ ] Install `date-fns` if not already installed: `npm install date-fns`
  - [ ] Export `formatRelativeDate(dateStr: string): string` — returns "X ago" format using `formatDistanceToNow`
  - [ ] Export `formatDate(dateStr: string): string` — returns "Feb 15, 2026" format using `format`
  - [ ] Both functions parse ISO 8601 strings

- [ ] [FRONT] Task 8: Unit tests for store, composable, and utils (AC: #1, #2, #3, #4)
  - [ ] Create `frontend/src/stores/__tests__/projects.spec.ts`
    - [ ] Test `fetchProjects` success: items and pagination populated
    - [ ] Test `fetchProjects` error: error state set, items empty
    - [ ] Test `reset`: clears all state
  - [ ] Create `frontend/src/composables/__tests__/useProjects.spec.ts`
    - [ ] Test loading state transitions
    - [ ] Test retry re-fetches with same params
  - [ ] Create `frontend/src/utils/__tests__/formatDate.spec.ts`
    - [ ] Test `formatRelativeDate` returns relative string
    - [ ] Test `formatDate` returns formatted date string
    - [ ] Test invalid date handling
  - [ ] Mock `apiClient` using `vi.mock('@/api/client')`
  - [ ] Use `createPinia()` + `setActivePinia()` in store tests

- [ ] [FRONT] Task 9: Component unit tests for ProjectListTable and ProjectEmptyState (AC: #1, #2, #5)
  - [ ] Create `frontend/src/features/projects/__tests__/ProjectEmptyState.spec.ts`
    - [ ] Test CTA button renders with correct label
    - [ ] Test clicking CTA emits `create` event
  - [ ] Create `frontend/src/features/projects/__tests__/ProjectListTable.spec.ts`
    - [ ] Test columns render correctly with project data
    - [ ] Test row click emits `row-click` with project data
    - [ ] Test loading prop shows loading state
  - [ ] Use `@vue/test-utils` `mount` with PrimeVue plugin configured

- [ ] [FRONT] Task 10: E2E test with Playwright (AC: #1, #2)
  - [ ] Create `frontend/e2e/tests/project-list.spec.ts`
  - [ ] Test: navigate to `/projects` with mocked API returning projects -> DataTable visible with rows
  - [ ] Test: navigate to `/projects` with mocked API returning empty list -> empty state CTA visible
  - [ ] Test: click a project row -> navigated to `/projects/:id`
  - [ ] Use Playwright route interception (`page.route()`) to mock API responses

## Dev Notes

This story transforms the placeholder `ProjectsView` into a fully functional project list page. It introduces the first feature module under `features/projects/` and establishes the pattern for all subsequent list views.

### Dependencies

**Story dependencies (already implemented):**
- Story 1-7: Vue 3 scaffold, PrimeVue 4, Tailwind CSS v4
- Story 1-8: App shell with AppSidebar (nav items already include "Projects" at `/projects`)
- Story 1-9: Auth guard, Pinia auth store
- Story 1-16: Router with `/projects` route, Pinia projects store (shell), `useAsyncAction`, `usePagination`, `apiClient`

**npm packages to install:**
```bash
cd frontend && npm install date-fns
```

### Architecture Requirements

**Component Hierarchy:**
```
ProjectsView.vue (route: /projects)
├── <h1> page header + optional "New Project" button
├── ProgressSpinner (v-if="isLoading && !projects.length")
├── Message (v-if="error")
│   └── retry Button
├── ProjectEmptyState (v-if="!isLoading && !error && projects.length === 0")
│   └── "Create your first project" Button
└── ProjectListTable (v-if="projects.length > 0")
    └── PrimeVue DataTable
        ├── Column: name
        ├── Column: description
        └── Column: created_at (formatted)
```

### File Paths (exact)

| File | Action |
|------|--------|
| `frontend/src/stores/projects.ts` | Replace shell with full implementation |
| `frontend/src/composables/useProjects.ts` | Create |
| `frontend/src/features/projects/ProjectEmptyState.vue` | Create |
| `frontend/src/features/projects/ProjectListTable.vue` | Create |
| `frontend/src/views/ProjectsView.vue` | Replace placeholder |
| `frontend/src/ui/layout/AppSidebar.vue` | Update (active state) |
| `frontend/src/utils/formatDate.ts` | Create |
| `frontend/src/stores/__tests__/projects.spec.ts` | Create |
| `frontend/src/composables/__tests__/useProjects.spec.ts` | Create |
| `frontend/src/utils/__tests__/formatDate.spec.ts` | Create |
| `frontend/src/features/projects/__tests__/ProjectEmptyState.spec.ts` | Create |
| `frontend/src/features/projects/__tests__/ProjectListTable.spec.ts` | Create |
| `frontend/e2e/tests/project-list.spec.ts` | Create |

### Technical Specifications

**Projects Store — full implementation:**

```typescript
// frontend/src/stores/projects.ts
import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

interface Project {
  id: string
  name: string
  description?: string
  owner_id: string
  created_at: string
  updated_at: string
}

interface Pagination {
  total: number
  page: number
  per_page: number
}

interface FetchParams {
  page?: number
  per_page?: number
  sort_by?: string
}

export const useProjectsStore = defineStore('projects', () => {
  const items = ref<Project[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchProjects(params: FetchParams = {}) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/projects', {
        params: { query: { page: params.page, per_page: params.per_page, sort_by: params.sort_by } },
      })
      if (apiError) {
        error.value = 'Failed to load projects'
        return
      }
      items.value = data?.data ?? []
      pagination.value = data?.pagination ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load projects'
    } finally {
      isLoading.value = false
    }
  }

  function reset() {
    items.value = []
    pagination.value = null
    error.value = null
  }

  return { items, pagination, isLoading, error, fetchProjects, reset }
})
```

> Note: Use generated types from `@/api/schema` if `schema.d.ts` is available. Otherwise, define the interfaces locally as above, matching the OpenAPI spec exactly.

**useProjects Composable:**

```typescript
// frontend/src/composables/useProjects.ts
import { computed, ref } from 'vue'
import { useProjectsStore } from '@/stores/projects'

interface FetchParams {
  page?: number
  per_page?: number
  sort_by?: string
}

export function useProjects() {
  const store = useProjectsStore()
  const lastParams = ref<FetchParams>({})

  async function fetchProjects(params: FetchParams = {}) {
    lastParams.value = params
    await store.fetchProjects(params)
  }

  async function retry() {
    await store.fetchProjects(lastParams.value)
  }

  return {
    projects: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchProjects,
    retry,
  }
}
```

**ProjectListTable Props/Emits:**

```typescript
// Props
interface Props {
  projects: Project[]
  totalRecords: number
  rows: number
  loading: boolean
  first: number  // 0-based offset for DataTable pagination
}

// Emits
interface Emits {
  page: [event: DataTablePageEvent]
  'row-click': [project: Project]
}
```

**ProjectEmptyState Emits:**

```typescript
interface Emits {
  create: []
}
```

**Date Formatting Utilities:**

```typescript
// frontend/src/utils/formatDate.ts
import { formatDistanceToNow, format, parseISO } from 'date-fns'

export function formatRelativeDate(dateStr: string): string {
  return formatDistanceToNow(parseISO(dateStr), { addSuffix: true })
}

export function formatDate(dateStr: string): string {
  return format(parseISO(dateStr), 'MMM d, yyyy')
}
```

**Sidebar Active State Logic:**

```typescript
// In AppSidebar.vue
import { useRoute } from 'vue-router'

const route = useRoute()

function isActive(itemRoute: string): boolean {
  if (itemRoute === '/') return route.path === '/'
  return route.path.startsWith(itemRoute)
}
```

### PrimeVue Components Used

- `DataTable` — project list with lazy pagination
- `Column` — table columns
- `Button` — CTA, retry, "New Project" placeholder
- `Message` — error display (severity="error")
- `ProgressSpinner` or `Skeleton` — loading state

### API Endpoints

| Method | Path | Query Params | Response |
|--------|------|-------------|----------|
| GET | /api/v1/projects | `page`, `per_page`, `sort_by` | 200: `{ data: Project[], pagination: Pagination }` |

### Style Conventions

- PrimeVue components for all interactive/display elements
- Tailwind for layout only (flex, gap, padding)
- Zero `<style scoped>` blocks
- No custom CSS classes

### Testing Requirements

**Unit tests (Vitest):**
- Store: fetchProjects success/error/reset (mock apiClient)
- Composable: loading transitions, retry logic
- Utils: formatRelativeDate, formatDate, edge cases
- Components: ProjectEmptyState CTA render/emit, ProjectListTable columns/row-click/loading

**E2E tests (Playwright):**
- Project list renders with data (mocked API)
- Empty state shows when no projects (mocked API)
- Row click navigates to detail page

**Manual verification checklist:**
1. `npm run dev` — navigate to `/projects`, see DataTable with projects (or empty state if no data)
2. Click a project row — navigated to `/projects/:id`
3. Sidebar "Projects" item shows active state on `/projects`
4. Resize to mobile — layout adapts correctly
5. `npm run build` — no TypeScript errors
6. `npm run lint` — no lint errors
7. `npm run test:unit` — all new tests pass

### References

- [Source: api/openapi.yaml — GET /projects endpoint, Project schema, Pagination schema]
- [Source: _bmad-output/planning-artifacts/architecture.md — Frontend hybrid structure, features/ directory]
- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.10]
- [Source: _bmad-output/implementation-artifacts/1-16-vue-app-routing-state-tooling.md — useAsyncAction, usePagination, apiClient, projects store shell]
- [Source: _bmad-output/implementation-artifacts/1-8-app-shell-layout-header-sidebar-status-bar.md — AppSidebar with nav items]
- [Source: frontend/CLAUDE.md — PrimeVue DataTable patterns, useAsyncAction pattern, component conventions]

## Dev Agent Record

## Change Log
