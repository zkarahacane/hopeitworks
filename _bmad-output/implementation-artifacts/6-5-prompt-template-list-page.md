# Story 6.5: [FRONT] Prompt Template List Page

Status: ready-for-dev

## Story

As a user, I want to view all prompt templates for a project, So that I can see which templates are available and navigate to their details.

## Acceptance Criteria (BDD)

**AC1: Display template list in DataTable**
- **Given** I am on `/projects/:id/templates`
- **When** the page loads and the project has templates
- **Then** I see a PrimeVue DataTable with columns: name, type (implement/retry/review/merge/custom), last updated

**AC2: Filter templates by type**
- **Given** I am viewing the template list
- **When** I select a type from the filter dropdown (or "All")
- **Then** the table shows only templates matching the selected type

**AC3: Navigate to template detail**
- **Given** I am viewing the template list
- **When** I click on a table row
- **Then** I navigate to `/projects/:id/templates/:templateId` (template detail/editor page)

**AC4: Create template button (admin only)**
- **Given** I am on `/projects/:id/templates` as an admin
- **When** the page loads
- **Then** I see a "Create Template" primary button above the table
- **When** I click "Create Template"
- **Then** I navigate to `/projects/:id/templates/new`

**AC5: No create button for non-admin users**
- **Given** I am on `/projects/:id/templates` as a non-admin user
- **When** the page loads
- **Then** I do not see a "Create Template" button

**AC6: Empty state**
- **Given** I am on `/projects/:id/templates`
- **When** the project has no templates
- **Then** I see an empty state message with informational text
- **And** if I am admin, the empty state includes a "Create Template" CTA button

**AC7: Loading and error states**
- **Given** I am on `/projects/:id/templates`
- **When** the API request is pending
- **Then** I see a loading skeleton
- **When** the API request fails
- **Then** I see an error message with a retry button

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create promptTemplates Pinia store (AC: #1, #7)
  - [ ] State: templates, loading, error
  - [ ] Actions: fetchTemplates, clearError
  - [ ] Use apiClient GET /api/v1/projects/{projectId}/templates
  - [ ] Export typed getters

- [ ] [FRONT] Task 2: Create usePromptTemplates composable (AC: #1, #7)
  - [ ] Wrap useAsyncAction pattern around store.fetchTemplates
  - [ ] Export loading, error, retry, templates
  - [ ] Auto-fetch on mount with projectId param

- [ ] [FRONT] Task 3: Build PromptTemplateTable.vue component (AC: #1, #2, #3)
  - [ ] Props: templates (PromptTemplate[]), isAdmin
  - [ ] Emits: rowClick, createClick
  - [ ] PrimeVue DataTable with columns: name, type, updated_at
  - [ ] Row click handler: emit rowClick with templateId
  - [ ] Type column: display badge with color (implement=blue, retry=orange, review=purple, merge=green, custom=gray)
  - [ ] Updated_at column: format as relative time (e.g., "2 hours ago")
  - [ ] Sortable columns (name, type, updated_at)
  - [ ] Paginator if > 10 rows

- [ ] [FRONT] Task 4: Add type filter to PromptTemplateTable (AC: #2)
  - [ ] Add PrimeVue Dropdown above table: options = ["All", "implement", "retry", "review", "merge", "custom"]
  - [ ] Filter templates locally based on selected type
  - [ ] Default: "All"

- [ ] [FRONT] Task 5: Build PromptTemplateEmptyState.vue component (AC: #6)
  - [ ] Props: isAdmin
  - [ ] Display PrimeVue Message with severity="info"
  - [ ] Message: "No prompt templates found for this project."
  - [ ] Admin: show "Create Template" primary button
  - [ ] Non-admin: no button
  - [ ] Emits: createClick (admin only)

- [ ] [FRONT] Task 6: Build PromptTemplatesView.vue route view (AC: #1, #4, #5, #6, #7)
  - [ ] Use usePromptTemplates composable
  - [ ] Use useAuth to check isAdmin
  - [ ] Route param: projectId from useRoute()
  - [ ] Loading state: PrimeVue Skeleton
  - [ ] Error state: PrimeVue Message with retry button
  - [ ] Success + no templates: render PromptTemplateEmptyState
  - [ ] Success + templates: render PromptTemplateTable
  - [ ] Admin: show "Create Template" button above table (if templates exist)
  - [ ] Handle rowClick: router.push to template detail
  - [ ] Handle createClick: router.push to /projects/:id/templates/new

- [ ] [FRONT] Task 7: Register route in router/index.ts (AC: #1)
  - [ ] Path: /projects/:id/templates
  - [ ] Name: project-templates
  - [ ] Component: PromptTemplatesView
  - [ ] Meta: requiresAuth: true

- [ ] [FRONT] Task 8: Write unit tests for store and composable (AC: #1, #7)
  - [ ] promptTemplates.spec.ts: test fetchTemplates, loading states, error handling
  - [ ] usePromptTemplates.spec.ts: test auto-fetch, retry logic

- [ ] [FRONT] Task 9: Write E2E test for template list page (AC: #1, #2, #3, #4, #5, #6)
  - [ ] templates.spec.ts: test table display, type filter, row click navigation, create button (admin vs non-admin), empty state

## Dev Notes

### Dependencies

- Story 1-8: App shell (AppLayout)
- Story 1-9: Login/auth guard (requiresAuth meta)
- Story 1-16: Routing, stores, apiClient setup
- Backend peer: Story 6-2 (prompt templates API) — consumes GET /api/v1/projects/{projectId}/templates

### Architecture Requirements

Component hierarchy:
```
PromptTemplatesView.vue (route view)
├── PrimeVue Skeleton (loading state)
├── PrimeVue Message (error state with retry button)
├── PromptTemplateEmptyState.vue (no templates)
└── PromptTemplateTable.vue (has templates)
    ├── PrimeVue Dropdown (type filter)
    └── PrimeVue DataTable
        └── Columns: name, type (badge), updated_at (relative time)
```

State management:
```
stores/promptTemplates.ts → composables/usePromptTemplates.ts → PromptTemplatesView.vue
```

### File Paths (exact)

```
frontend/src/stores/promptTemplates.ts
frontend/src/composables/usePromptTemplates.ts
frontend/src/features/templates/PromptTemplateTable.vue
frontend/src/features/templates/PromptTemplateEmptyState.vue
frontend/src/views/PromptTemplatesView.vue
frontend/src/router/index.ts (append route)
frontend/src/__tests__/stores/promptTemplates.spec.ts
frontend/src/__tests__/composables/usePromptTemplates.spec.ts
frontend/e2e/tests/templates.spec.ts
```

### Technical Specifications

**PromptTemplateTable.vue props/emits:**
```typescript
interface Props {
  templates: PromptTemplate[] // from generated API types
  isAdmin: boolean
}
const emit = defineEmits<{
  rowClick: [templateId: string]
  createClick: []
}>()
```

**PromptTemplateEmptyState.vue props/emits:**
```typescript
interface Props {
  isAdmin: boolean
}
const emit = defineEmits<{
  createClick: []
}>()
```

**Type badge color mapping:**
```typescript
const typeBadgeColors = {
  implement: 'bg-blue-500',
  retry: 'bg-orange-500',
  review: 'bg-purple-500',
  merge: 'bg-green-500',
  custom: 'bg-gray-500'
}
```

**Type filter options:**
```typescript
const typeFilterOptions = [
  { label: 'All', value: null },
  { label: 'Implement', value: 'implement' },
  { label: 'Retry', value: 'retry' },
  { label: 'Review', value: 'review' },
  { label: 'Merge', value: 'merge' },
  { label: 'Custom', value: 'custom' }
]
```

**DataTable columns:**
```typescript
const columns = [
  { field: 'name', header: 'Name', sortable: true },
  { field: 'type', header: 'Type', sortable: true },
  { field: 'updated_at', header: 'Last Updated', sortable: true }
]
```

**Relative time formatting:**
```typescript
// Use dayjs or date-fns for relative time
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
dayjs.extend(relativeTime)

const formatRelativeTime = (date: string) => dayjs(date).fromNow()
// Example: "2 hours ago", "3 days ago"
```

### Testing Requirements

**Unit tests:**
- promptTemplates store: fetchTemplates success/error, loading states, clearError
- usePromptTemplates composable: auto-fetch on mount, retry logic, reactivity

**E2E tests:**
- Load templates page → see DataTable with templates
- Filter by type → see filtered results
- Click row → navigate to template detail
- Admin user → see "Create Template" button
- Non-admin user → no "Create Template" button
- No templates → see empty state
- API error → see error message + retry button → retry works

### References

- Epic 6: Pipeline Configuration & Prompt Templates
- Backend Story 6-2: Prompt templates API (GET /api/v1/projects/{projectId}/templates)
- PrimeVue DataTable: https://primevue.org/datatable/
- PrimeVue Dropdown: https://primevue.org/dropdown/
- PrimeVue Badge: https://primevue.org/badge/
- PrimeVue Message: https://primevue.org/message/
- dayjs relative time: https://day.js.org/docs/en/plugin/relative-time

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
