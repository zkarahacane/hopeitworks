# Story 2.5: [FRONT] Epic Detail — Split Focus + Story List + Filters

Status: ready-for-dev

## Story

As a project user, I want a split layout to browse and select stories within an epic, So that I can efficiently review story details.

## Acceptance Criteria (BDD)

**AC1: Split layout with story list and detail panel**
- **Given** I navigate to `/projects/:id/epics/:epicId`
- **When** the page loads and the epic has stories
- **Then** I see a left panel (300px fixed width) showing the story list and a right panel (flex-1) showing the selected story detail

**AC2: Story list displays compact cards with status**
- **Given** I am viewing the story list
- **When** the list renders
- **Then** each StoryStatusCard shows key (monospace font), title, and status badge with colors (backlog=gray-500, running=blue-500, done=green-500, failed=red-500)

**AC3: Filter bar with status and text search**
- **Given** I am viewing the story list
- **When** I interact with the filter bar
- **Then** I can filter by status (dropdown) and search by text (input with 200ms debounce)
- **And** filters are preserved in URL query params
- **And** the story list updates reactively

**AC4: Keyboard navigation**
- **Given** I am viewing the story list
- **When** I press J/K keys
- **Then** selection moves down/up respectively
- **When** I press Enter
- **Then** the selected story detail loads in the right panel

**AC5: Click story to show detail**
- **Given** I am viewing the story list
- **When** I click on a story card
- **Then** the story detail loads in the right panel without changing the route
- **And** the card shows selected state

**AC6: Loading and error states**
- **Given** I navigate to `/projects/:id/epics/:epicId`
- **When** the API request is pending
- **Then** I see a loading skeleton
- **When** the API request fails
- **Then** I see an error message with a retry button

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create stories Pinia store (AC: #1, #3, #6)
  - [ ] Define state: stories, selectedStoryId, filters, loading, error
  - [ ] Actions: fetchStoriesByEpic, setSelectedStory, setFilters, clearError
  - [ ] Use apiClient GET /api/v1/projects/{projectId}/stories?epic_id={epicId}&status={status}
  - [ ] Export typed getters for filteredStories, selectedStory

- [ ] [FRONT] Task 2: Create useStories composable (AC: #1, #6)
  - [ ] Wrap useAsyncAction pattern around store.fetchStoriesByEpic
  - [ ] Export loading, error, retry, stories, selectedStory
  - [ ] Auto-fetch on mount with projectId and epicId params
  - [ ] Handle filter changes and re-fetch

- [ ] [FRONT] Task 3: Build StoryStatusCard.vue component (AC: #2, #5)
  - [ ] Props: story (Story type from generated API), isSelected (boolean)
  - [ ] Emits: click
  - [ ] Display key (monospace font), title, status badge with PrimeVue Badge
  - [ ] Color scheme: backlog=gray-500, running=blue-500, done=green-500, failed=red-500
  - [ ] Selected state with border/background highlight
  - [ ] Hover state with cursor pointer
  - [ ] Zero custom CSS, Tailwind only

- [ ] [FRONT] Task 4: Build StoryFilterBar.vue component (AC: #3)
  - [ ] Props: modelValue (filters object with status and search)
  - [ ] Emits: update:modelValue
  - [ ] PrimeVue Dropdown for status filter (all, backlog, running, done, failed)
  - [ ] PrimeVue InputText for text search with icon
  - [ ] watchDebounced from @vueuse/core (200ms) on search input
  - [ ] Sync filters to URL query params via useRouter

- [ ] [FRONT] Task 5: Build StoryListPanel.vue component (AC: #1, #2, #4, #5)
  - [ ] Props: stories (Story[]), selectedId (string | null)
  - [ ] Emits: select
  - [ ] Fixed width 300px, flex-shrink-0
  - [ ] Render StoryFilterBar at top
  - [ ] Render StoryStatusCard for each story (scrollable list)
  - [ ] Keyboard nav: useEventListener for keydown, J=next, K=prev, Enter=emit select
  - [ ] Track internal selection index for keyboard nav
  - [ ] Handle click on card → emit select

- [ ] [FRONT] Task 6: Build StoryDetailPanel.vue component (AC: #1, #5)
  - [ ] Props: story (Story | null)
  - [ ] Flex-1 width, scrollable
  - [ ] Display full story detail: key (monospace), title, status badge, objective, acceptance_criteria, target_files, depends_on
  - [ ] Empty state when story is null (e.g., "Select a story to view details")
  - [ ] Use PrimeVue Panel or Card for layout

- [ ] [FRONT] Task 7: Build EpicDetailView.vue + EpicDetailLayout.vue (AC: #1, #6)
  - [ ] Route view: EpicDetailView.vue
  - [ ] Use useStories composable
  - [ ] Route params: projectId and epicId from useRoute()
  - [ ] Loading state: PrimeVue Skeleton for split layout
  - [ ] Error state: PrimeVue Message with retry button
  - [ ] Success: render EpicDetailLayout with StoryListPanel + StoryDetailPanel
  - [ ] EpicDetailLayout: flex container, left panel + right panel
  - [ ] Handle story selection from list panel → update selectedStoryId in store

- [ ] [FRONT] Task 8: Register route + write unit tests (AC: #1)
  - [ ] Path: /projects/:id/epics/:epicId
  - [ ] Name: epic-detail
  - [ ] Component: EpicDetailView
  - [ ] Meta: requiresAuth: true
  - [ ] stories.spec.ts: test fetchStoriesByEpic, setFilters, setSelectedStory, loading/error states
  - [ ] useStories.spec.ts: test auto-fetch, retry logic, filter reactivity

- [ ] [FRONT] Task 9: Write E2E test for epic detail page (AC: #1, #2, #3, #4, #5)
  - [ ] epic-detail.spec.ts: test story list display, filter by status, text search, keyboard nav (J/K/Enter), click story → detail loads, loading/error states

## Dev Notes

### Dependencies

- Story 2-4: Board page (wave 5) — provides navigation to this page from epic card click
- Story 2-2: Stories CRUD API (wave 6) — provides GET /api/v1/projects/{projectId}/stories?epic_id={epicId}&status={status}
- Story 1-8: App shell (AppLayout)
- Story 1-9: Login/auth guard (requiresAuth meta)
- Story 1-16: Routing, stores, apiClient setup

### Architecture Requirements

Component hierarchy:
```
EpicDetailView.vue (route view)
├── PrimeVue Skeleton (loading state)
├── PrimeVue Message (error state with retry button)
└── EpicDetailLayout.vue (split layout wrapper)
    ├── StoryListPanel.vue (left 300px)
    │   ├── StoryFilterBar.vue
    │   │   ├── PrimeVue Dropdown (status filter)
    │   │   └── PrimeVue InputText (search, 200ms debounce)
    │   └── StoryStatusCard.vue (repeated)
    └── StoryDetailPanel.vue (right, flex-1)
        └── story detail content (key, title, objective, acceptance_criteria, target_files, depends_on)
```

State management:
```
stores/stories.ts → composables/useStories.ts → EpicDetailView.vue
```

### File Paths (exact)

```
frontend/src/stores/stories.ts
frontend/src/composables/useStories.ts
frontend/src/features/board/StoryStatusCard.vue
frontend/src/features/board/StoryFilterBar.vue
frontend/src/features/board/StoryListPanel.vue
frontend/src/features/board/StoryDetailPanel.vue
frontend/src/features/board/EpicDetailLayout.vue
frontend/src/views/EpicDetailView.vue
frontend/src/router/index.ts (append route)
frontend/src/__tests__/stores/stories.spec.ts
frontend/src/__tests__/composables/useStories.spec.ts
frontend/e2e/tests/epic-detail.spec.ts
```

### Technical Specifications

**StoryStatusCard.vue props/emits:**
```typescript
interface Props {
  story: Story // from generated API types
  isSelected: boolean
}
const emit = defineEmits<{
  click: [storyId: string]
}>()
```

**StoryFilterBar.vue props/emits:**
```typescript
interface Props {
  modelValue: {
    status: string | null // 'all' | 'backlog' | 'running' | 'done' | 'failed'
    search: string
  }
}
const emit = defineEmits<{
  'update:modelValue': [filters: Props['modelValue']]
}>()
```

**StoryListPanel.vue props/emits:**
```typescript
interface Props {
  stories: Story[]
  selectedId: string | null
}
const emit = defineEmits<{
  select: [storyId: string]
}>()
```

**StoryDetailPanel.vue props:**
```typescript
interface Props {
  story: Story | null
}
```

**Status badge color mapping:**
```typescript
const statusColors = {
  backlog: 'bg-gray-500',
  running: 'bg-blue-500',
  done: 'bg-green-500',
  failed: 'bg-red-500'
}
```

**Keyboard navigation:**
```typescript
// In StoryListPanel.vue
import { useEventListener } from '@vueuse/core'

const selectedIndex = ref(0)
const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'j') {
    selectedIndex.value = Math.min(selectedIndex.value + 1, props.stories.length - 1)
  } else if (e.key === 'k') {
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    emit('select', props.stories[selectedIndex.value].id)
  }
}
useEventListener('keydown', handleKeydown)
```

**Debounced search:**
```typescript
// In StoryFilterBar.vue
import { watchDebounced } from '@vueuse/core'

const localSearch = ref(props.modelValue.search)
watchDebounced(
  localSearch,
  (newValue) => {
    emit('update:modelValue', { ...props.modelValue, search: newValue })
  },
  { debounce: 200 }
)
```

**URL sync for filters:**
```typescript
// In EpicDetailView.vue or StoryFilterBar.vue
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

watch(filters, (newFilters) => {
  router.replace({
    query: {
      ...route.query,
      status: newFilters.status || undefined,
      search: newFilters.search || undefined
    }
  })
}, { deep: true })
```

**Split layout (Tailwind CSS v4):**
```html
<!-- EpicDetailLayout.vue -->
<div class="flex h-full gap-4">
  <div class="w-[300px] shrink-0 overflow-y-auto">
    <StoryListPanel ... />
  </div>
  <div class="flex-1 overflow-y-auto">
    <StoryDetailPanel ... />
  </div>
</div>
```

### Testing Requirements

**Unit tests:**
- stories store: fetchStoriesByEpic success/error, setFilters, setSelectedStory, loading states, clearError
- useStories composable: auto-fetch on mount with epicId, retry logic, filter reactivity, selectedStory computed

**E2E tests:**
- Load epic detail page → see story list on left, detail on right
- Filter by status → list updates
- Search by text → list updates with debounce
- Press J/K → selection moves
- Press Enter → detail loads in right panel
- Click story card → detail loads in right panel
- API error → see error message + retry button → retry works

### References

- Epic 2: Story Board Management
- Backend Story 2-2: Stories API (GET /api/v1/projects/{projectId}/stories with epic_id and status filters)
- PrimeVue Badge: https://primevue.org/badge/
- PrimeVue Dropdown: https://primevue.org/dropdown/
- PrimeVue InputText: https://primevue.org/inputtext/
- PrimeVue Message: https://primevue.org/message/
- PrimeVue Skeleton: https://primevue.org/skeleton/
- @vueuse/core watchDebounced: https://vueuse.org/shared/watchDebounced/
- @vueuse/core useEventListener: https://vueuse.org/core/useEventListener/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
