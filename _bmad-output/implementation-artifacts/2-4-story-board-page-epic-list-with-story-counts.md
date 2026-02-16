# Story 2.4: [FRONT] Story Board Page — Epic List with Story Counts

Status: ready-for-dev

## Story

As a project user, I want to see epics with story counts by status, So that I can understand project progress at a glance.

## Acceptance Criteria (BDD)

**AC1: Display epic list with story counts**
- **Given** I am on `/projects/:id/board`
- **When** the page loads and the project has epics
- **Then** I see a grid of epic cards, each showing title, description, and story counts by status (backlog, running, done, failed) with colored badges (backlog=gray, running=blue, done=green, failed=red)

**AC2: Empty state with CTA**
- **Given** I am on `/projects/:id/board`
- **When** the project has no epics
- **Then** I see an empty state message with a "Create Epic" CTA button (admin only) or informational text (non-admin)

**AC3: Navigate to epic detail**
- **Given** I am viewing the epic list
- **When** I click on an epic card
- **Then** I navigate to `/projects/:id/epics/:epicId`

**AC4: Loading and error states**
- **Given** I am on `/projects/:id/board`
- **When** the API request is pending
- **Then** I see a loading skeleton
- **When** the API request fails
- **Then** I see an error message with a retry button

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create epics Pinia store (AC: #1, #4)
  - [ ] Define state: epics, loading, error
  - [ ] Actions: fetchEpics, clearError
  - [ ] Use apiClient GET /api/v1/projects/{projectId}/epics
  - [ ] Export typed getters

- [ ] [FRONT] Task 2: Create useEpics composable (AC: #1, #4)
  - [ ] Wrap useAsyncAction pattern around store.fetchEpics
  - [ ] Export loading, error, retry, epics
  - [ ] Auto-fetch on mount with projectId param

- [ ] [FRONT] Task 3: Build EpicCard.vue component (AC: #1, #3)
  - [ ] Props: epic (Epic type from generated API)
  - [ ] Emits: click
  - [ ] Display title, description, story counts with PrimeVue Badge
  - [ ] Color scheme: backlog=gray-500, running=blue-500, done=green-500, failed=red-500
  - [ ] Card hover state with cursor pointer
  - [ ] Zero custom CSS, Tailwind only

- [ ] [FRONT] Task 4: Build EpicCardGrid.vue component (AC: #1)
  - [ ] Props: epics (Epic[])
  - [ ] Emits: epicClick
  - [ ] Grid layout: responsive (1 col mobile, 2 col tablet, 3 col desktop)
  - [ ] Map over epics, render EpicCard for each

- [ ] [FRONT] Task 5: Build BoardEmptyState.vue component (AC: #2)
  - [ ] Props: isAdmin (boolean)
  - [ ] Display PrimeVue Message with severity="info"
  - [ ] Admin: show "Create Epic" primary button
  - [ ] Non-admin: show informational text only
  - [ ] Emits: createEpic (admin only)

- [ ] [FRONT] Task 6: Build BoardView.vue route view (AC: #1, #2, #3, #4)
  - [ ] Use useEpics composable
  - [ ] Route param: projectId from useRoute()
  - [ ] Loading state: PrimeVue Skeleton grid
  - [ ] Error state: Message with retry button
  - [ ] Success + no epics: render BoardEmptyState
  - [ ] Success + epics: render EpicCardGrid
  - [ ] Handle epicClick: router.push to epic detail

- [ ] [FRONT] Task 7: Register route in router/index.ts (AC: #1)
  - [ ] Path: /projects/:id/board
  - [ ] Name: project-board
  - [ ] Component: BoardView
  - [ ] Meta: requiresAuth: true

- [ ] [FRONT] Task 8: Write unit tests for store and composable (AC: #1, #4)
  - [ ] epics.spec.ts: test fetchEpics, loading states, error handling
  - [ ] useEpics.spec.ts: test auto-fetch, retry logic

- [ ] [FRONT] Task 9: Write E2E test for board page (AC: #1, #2, #3)
  - [ ] board.spec.ts: test epic list display, empty state, navigation to epic detail, loading/error states

## Dev Notes

### Dependencies

- Story 1-8: App shell (AppLayout)
- Story 1-9: Login/auth guard (requiresAuth meta)
- Story 1-16: Routing, stores, apiClient setup
- Backend peer: Story 2-1 (epics API) — consumes GET /api/v1/projects/{projectId}/epics

### Architecture Requirements

Component hierarchy:
```
BoardView.vue (route view)
├── PrimeVue Skeleton (loading state)
├── PrimeVue Message (error state with retry button)
├── BoardEmptyState.vue (no epics)
└── EpicCardGrid.vue (has epics)
    └── EpicCard.vue (repeated)
        ├── title (h3)
        ├── description (p, truncated)
        └── story counts (PrimeVue Badge components)
```

State management:
```
stores/epics.ts → composables/useEpics.ts → BoardView.vue
```

### File Paths (exact)

```
frontend/src/stores/epics.ts
frontend/src/composables/useEpics.ts
frontend/src/features/board/EpicCard.vue
frontend/src/features/board/EpicCardGrid.vue
frontend/src/features/board/BoardEmptyState.vue
frontend/src/views/BoardView.vue
frontend/src/router/index.ts (append route)
frontend/src/__tests__/stores/epics.spec.ts
frontend/src/__tests__/composables/useEpics.spec.ts
frontend/e2e/tests/board.spec.ts
```

### Technical Specifications

**EpicCard.vue props/emits:**
```typescript
interface Props {
  epic: Epic // from generated API types
}
const emit = defineEmits<{
  click: [epicId: string]
}>()
```

**EpicCardGrid.vue props/emits:**
```typescript
interface Props {
  epics: Epic[]
}
const emit = defineEmits<{
  epicClick: [epicId: string]
}>()
```

**BoardEmptyState.vue props/emits:**
```typescript
interface Props {
  isAdmin: boolean
}
const emit = defineEmits<{
  createEpic: []
}>()
```

**Badge color mapping:**
```typescript
const statusColors = {
  backlog: 'bg-gray-500',
  running: 'bg-blue-500',
  done: 'bg-green-500',
  failed: 'bg-red-500'
}
```

**Grid responsive classes (Tailwind CSS v4):**
```html
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
```

### Testing Requirements

**Unit tests:**
- epics store: fetchEpics success/error, loading states, clearError
- useEpics composable: auto-fetch on mount, retry logic, reactivity

**E2E tests:**
- Load board page → see epic cards with counts
- No epics → see empty state
- Click epic card → navigate to epic detail
- API error → see error message + retry button → retry works

### References

- Epic 2: Story Board Management
- Backend Story 2-1: Epics API (GET /api/v1/projects/{projectId}/epics)
- PrimeVue Badge: https://primevue.org/badge/
- PrimeVue Message: https://primevue.org/message/
- PrimeVue Skeleton: https://primevue.org/skeleton/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
