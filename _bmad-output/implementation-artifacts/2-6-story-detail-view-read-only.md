# Story 2.6: [FRONT] Story Detail View (Read-Only)

Status: ready-for-dev

## Story

As a project user, I want to see story details including objectives and acceptance criteria, So that I can understand what needs to be built.

## Acceptance Criteria (BDD)

**AC1: Detail panel shows all story fields with correct rendering**
- **Given** a story is selected in the epic detail split view
- **When** the detail panel renders
- **Then** I see: title (h2), key (monospace), status badge (PrimeVue severity), scope badge (backend/frontend/shared), objective rendered as markdown, acceptance criteria rendered as markdown, target files as monospace list, and dependencies as clickable keys

**AC2: Markdown rendering for objective and acceptance criteria**
- **Given** a story has objective or acceptance_criteria fields with markdown content
- **When** the detail panel renders those fields
- **Then** the content is rendered as HTML via a markdown parser (bold, code, lists, headings are formatted)
- **And** the rendered HTML is sanitized (no XSS)

**AC3: Dependency keys are clickable and select the story in the left panel**
- **Given** the detail panel shows a story with depends_on entries
- **When** I click a dependency key (e.g., "S-01")
- **Then** the stories store selects the story with that key
- **And** the left panel scroll-highlights the selected story card
- **And** the right panel updates to show the clicked dependency's detail

**AC4: StoryDetailView standalone route fetches real story data**
- **Given** I navigate to `/projects/:projectId/stories/:storyId`
- **When** the page loads
- **Then** the view calls GET /api/v1/projects/{projectId}/stories/{storyId}
- **And** renders the full story detail with RunLaunchButton and all fields
- **And** shows a Skeleton during loading and a Message with retry on error

**AC5: RunLaunchButton is integrated into StoryDetailPanel**
- **Given** a story is selected in the detail panel
- **When** the story status is "backlog"
- **Then** a "Launch Run" button is visible in the detail panel header
- **When** the story status is "running"
- **Then** a disabled "Running..." button is shown

**AC6: Empty state when no story is selected**
- **Given** no story is selected in the epic detail view
- **When** the detail panel renders
- **Then** I see an empty state with icon and "Select a story to view details" message

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Install markdown dependency and add useMarkdown utility (AC: #2)
  - [ ] Install `marked` package: `npm install marked @types/marked` (lightweight, tree-shakeable)
  - [ ] Install `dompurify` package: `npm install dompurify @types/dompurify` for XSS sanitization
  - [ ] Create `frontend/src/utils/renderMarkdown.ts` — pure function wrapping `marked.parse` + `DOMPurify.sanitize`
  - [ ] Return type is `string` (sanitized HTML)
  - [ ] Write unit test in `frontend/src/utils/__tests__/renderMarkdown.spec.ts`

- [ ] [FRONT] Task 2: Add `scope` field to the Story type in the store (AC: #1)
  - [ ] In `frontend/src/stores/stories.ts`, add `scope?: 'backend' | 'frontend' | 'shared'` to the `Story` interface
  - [ ] This aligns with the OpenAPI Story schema which already defines scope as optional enum

- [ ] [FRONT] Task 3: Enhance StoryDetailPanel.vue — markdown rendering and scope badge (AC: #1, #2, #6)
  - [ ] Replace `white-space: pre-wrap` text display for objective and acceptance_criteria with `v-html` bound to `renderMarkdown(field)`
  - [ ] Add scope badge row below the key+status row using PrimeVue Tag with `severity` mapped from scope value (backend=info, frontend=warn, shared=secondary)
  - [ ] Add a `prose`-style wrapper div around v-html content (use inline PrimeVue surface tokens, no Tailwind typography plugin)
  - [ ] No business logic in the component — call renderMarkdown directly as a pure utility in the template

- [ ] [FRONT] Task 4: Make dependency keys clickable in StoryDetailPanel.vue (AC: #3)
  - [ ] Add `allStories` prop: `Story[]` — needed to resolve keys to IDs
  - [ ] Add `select-dependency` emit: `[storyId: string]`
  - [ ] In the depends_on list, render each key as a `<button>` styled like a monospace link (PrimeVue Button text severity or inline anchor style)
  - [ ] On click: find story in allStories by key, emit `select-dependency` with its id; if not found, do nothing silently
  - [ ] Update EpicDetailLayout.vue to pass `allStories` and wire `@select-dependency` → `emit('select', storyId)`

- [ ] [FRONT] Task 5: Integrate RunLaunchButton into StoryDetailPanel.vue (AC: #5)
  - [ ] Add props: `projectId: string`, `showLaunchButton?: boolean` (default false — only shown when context provides it)
  - [ ] Add emits: `launch-click: []` (forwards the RunLaunchButton event up)
  - [ ] Import and place RunLaunchButton in the panel header row (alongside key + status badge)
  - [ ] Pass story.id, story.key, story.title, story.status to RunLaunchButton
  - [ ] EpicDetailLayout.vue: pass `project-id` and `show-launch-button` props down; wire `@launch-click` up to parent

- [ ] [FRONT] Task 6: Create useStoryDetail composable (AC: #4)
  - [ ] File: `frontend/src/composables/useStoryDetail.ts`
  - [ ] Accepts `projectId: string`, `storyId: string`
  - [ ] Uses `useAsyncAction` to wrap apiClient.GET(`/projects/{projectId}/stories/{storyId}`)
  - [ ] Exposes: `story` (data ref), `isLoading`, `error`, `fetchStory`, `retry`
  - [ ] Calls `fetchStory` on mount via `onMounted`
  - [ ] Write unit test in `frontend/src/composables/__tests__/useStoryDetail.spec.ts` covering: success, 404 error, retry

- [ ] [FRONT] Task 7: Replace placeholder content in StoryDetailView.vue with real data (AC: #4, #5)
  - [ ] Remove hardcoded `storyKey`, `storyTitle`, `storyStatus` refs
  - [ ] Use `useStoryDetail(projectId.value, storyId.value)` composable
  - [ ] Show PrimeVue Skeleton (title bar + sections) while `isLoading` is true
  - [ ] Show PrimeVue Message with retry button when `error` is not null
  - [ ] When `story` is available, render StoryDetailPanel with all props wired
  - [ ] Wire RunLaunchConfirmDialog and Toast as they already exist, but use `story.key` and `story.title` from fetched data
  - [ ] Add back-navigation button to epic detail (if `epic_id` is available on the story)

- [ ] [FRONT] Task 8: Write unit tests for StoryDetailPanel.vue and updated EpicDetailLayout.vue (AC: #1, #2, #3)
  - [ ] File: `frontend/src/features/board/__tests__/StoryDetailPanel.spec.ts`
  - [ ] Test: empty state renders when story is null
  - [ ] Test: all fields render when story has full data
  - [ ] Test: markdown is rendered as HTML (not plain text) for objective and acceptance_criteria
  - [ ] Test: scope badge is shown when scope is set, hidden when absent
  - [ ] Test: clicking a dependency key emits `select-dependency` with the correct story id
  - [ ] Test: dependency key not found in allStories does not emit

## Dev Notes

### Dependencies

- Story 2-5: Epic detail split layout (DONE) — this story enhances StoryDetailPanel.vue and EpicDetailLayout.vue which are already in place
- Story 2-2: Stories CRUD API (DONE) — provides GET /api/v1/projects/{projectId}/stories/{storyId} which StoryDetailView.vue will now consume
- Story 1-16: Vue routing and apiClient already set up

### Architecture Requirements

Component hierarchy (changes from 2-5 baseline):

```
EpicDetailView.vue (unchanged — orchestration stays in composable)
└── EpicDetailLayout.vue (enhanced: passes allStories, projectId, wires select-dependency + launch-click)
    ├── StoryListPanel.vue (unchanged)
    └── StoryDetailPanel.vue (enhanced: markdown, scope, clickable deps, RunLaunchButton)
        ├── PrimeVue Tag (scope)
        ├── PrimeVue Badge (status)
        ├── RunLaunchButton (conditional on showLaunchButton prop)
        ├── v-html (renderMarkdown(objective))
        ├── v-html (renderMarkdown(acceptance_criteria))
        ├── <button> × N (depends_on clickable keys)
        └── <li> × N (target_files monospace)

StoryDetailView.vue (standalone route, fully rewritten)
├── PrimeVue Skeleton (loading)
├── PrimeVue Message + Button (error + retry)
└── StoryDetailPanel (story, projectId, showLaunchButton=true, allStories=[])
    └── RunLaunchConfirmDialog
```

State flow for dependency click:
```
StoryDetailPanel emits select-dependency(storyId)
  → EpicDetailLayout emits select(storyId)
    → EpicDetailView calls selectStory(storyId)
      → store.setSelectedStory(storyId)
        → selectedStory computed updates
          → StoryDetailPanel re-renders with new story
```

### File Paths (exact)

```
frontend/src/utils/renderMarkdown.ts                           (new)
frontend/src/utils/__tests__/renderMarkdown.spec.ts            (new)
frontend/src/composables/useStoryDetail.ts                     (new)
frontend/src/composables/__tests__/useStoryDetail.spec.ts      (new)
frontend/src/stores/stories.ts                                 (add scope field to Story interface)
frontend/src/features/board/StoryDetailPanel.vue               (enhance)
frontend/src/features/board/EpicDetailLayout.vue               (enhance — wire allStories + events)
frontend/src/features/board/__tests__/StoryDetailPanel.spec.ts (new)
frontend/src/views/StoryDetailView.vue                         (replace placeholder with real data)
```

No new routes needed — `story-detail` route already exists at `/projects/:projectId/stories/:storyId`.

### Technical Specifications

**renderMarkdown.ts:**
```typescript
import { marked } from 'marked'
import DOMPurify from 'dompurify'

/** Renders markdown string to sanitized HTML. Returns empty string for falsy input. */
export function renderMarkdown(input: string | undefined | null): string {
  if (!input) return ''
  const raw = marked.parse(input, { async: false }) as string
  return DOMPurify.sanitize(raw)
}
```

**useStoryDetail.ts:**
```typescript
import { onMounted } from 'vue'
import { useAsyncAction } from './useAsyncAction'
import { apiClient } from '@/api/client'

export function useStoryDetail(projectId: string, storyId: string) {
  const { data: story, isLoading, error, execute } = useAsyncAction(async () => {
    const { data, error: apiError } = await apiClient.GET(
      '/projects/{projectId}/stories/{storyId}',
      { params: { path: { projectId, storyId } } }
    )
    if (apiError) throw new Error('Failed to load story')
    return data
  })

  async function fetchStory() {
    await execute()
  }

  onMounted(fetchStory)

  return { story, isLoading, error, fetchStory, retry: fetchStory }
}
```

**Story interface update (stores/stories.ts):**
```typescript
export interface Story {
  // ... existing fields ...
  scope?: 'backend' | 'frontend' | 'shared'   // add this line
}
```

**StoryDetailPanel.vue updated props/emits:**
```typescript
defineProps<{
  story: Story | null
  allStories?: Story[]           // for resolving dependency key → id
  projectId?: string             // passed through to RunLaunchButton
  showLaunchButton?: boolean     // default false
}>()

const emit = defineEmits<{
  'select-dependency': [storyId: string]
  'launch-click': []
}>()
```

**Scope badge severity mapping:**
```typescript
const scopeSeverityMap: Record<string, 'info' | 'warn' | 'secondary'> = {
  backend: 'info',
  frontend: 'warn',
  shared: 'secondary',
}
```

**Clickable dependency key handler (in StoryDetailPanel.vue):**
```typescript
function handleDependencyClick(key: string) {
  const match = props.allStories?.find((s) => s.key === key)
  if (match) emit('select-dependency', match.id)
}
```

**EpicDetailLayout.vue updated props/emits:**
```typescript
defineProps<{
  stories: Story[]         // filteredStories (existing)
  allStories: Story[]      // store.items — needed for dep resolution
  selectedStory: Story | null
  selectedStoryId: string | null
  filters: StoryFilters
  projectId: string        // needed for RunLaunchButton inside panel
}>()

const emit = defineEmits<{
  select: [storyId: string]
  'update:filters': [filters: StoryFilters]
  'launch-click': []
}>()
```

**EpicDetailView.vue change to pass allStories:**
```typescript
// useStories already exposes allStories (store.items)
// Pass to EpicDetailLayout:
// :all-stories="allStories"
// :project-id="projectId"
// @launch-click="handleLaunchClick" (wire to dialog)
```

**Markdown HTML rendering in template:**
```html
<div v-if="story.objective" class="flex flex-col gap-1">
  <h3 ...>Objective</h3>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="prose-content" v-html="renderMarkdown(story.objective)" />
</div>
```

Use a minimal `prose-content` class override via CSS custom properties if needed, or rely on PrimeVue surface token defaults for `p`, `ul`, `code` inside the rendered HTML. Do NOT use `@tailwindcss/typography`.

**StoryDetailView.vue skeleton structure:**
```html
<!-- Loading -->
<div v-if="isLoading" class="flex flex-col gap-4 p-6">
  <div class="flex items-center justify-between">
    <Skeleton width="8rem" height="1.25rem" />
    <Skeleton width="6rem" height="2rem" />
  </div>
  <Skeleton width="60%" height="1.75rem" />
  <Skeleton width="100%" height="6rem" />
  <Skeleton width="100%" height="8rem" />
</div>
<!-- Error -->
<Message v-else-if="error" severity="error" :closable="false">
  <div class="flex items-center gap-3">
    <span>{{ error.message }}</span>
    <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
  </div>
</Message>
<!-- Story -->
<StoryDetailPanel
  v-else-if="story"
  :story="story"
  :project-id="projectId"
  :all-stories="[]"
  :show-launch-button="true"
  @launch-click="handleLaunchClick"
/>
```

### Testing Requirements

**renderMarkdown.spec.ts:**
- Returns empty string for null, undefined, empty string
- Converts `**bold**` to `<strong>bold</strong>`
- Converts `` `code` `` to `<code>code</code>`
- Strips `<script>` tags (XSS prevention via DOMPurify)

**useStoryDetail.spec.ts:**
- Fetches story on mount and populates `story.value`
- Sets `error` when API returns error
- `retry()` re-calls the API

**StoryDetailPanel.spec.ts:**
- Renders empty state when story is null
- Renders title, key, status badge when story is provided
- Renders scope badge with correct severity when scope is set
- Does not render scope badge when scope is absent
- Renders `v-html` with parsed markdown (check for `<strong>` or `<ul>` tags)
- Emits `select-dependency` with correct storyId when dependency key button is clicked
- Does not emit when clicked key is not found in allStories
- Renders RunLaunchButton when showLaunchButton is true and status is backlog

### References

- OpenAPI spec: `/api/v1/projects/{projectId}/stories/{storyId}` — operationId: `getStory`
- Story type in OpenAPI: includes `scope?: backend | frontend | shared` (add to store interface)
- `marked` library: https://marked.js.org/ (already in the JS ecosystem, zero config needed)
- `dompurify`: https://github.com/cure53/DOMPurify
- PrimeVue Tag: https://primevue.org/tag/ (for scope display)
- PrimeVue Badge: https://primevue.org/badge/ (for status, already in use)
- PrimeVue Skeleton: https://primevue.org/skeleton/
- PrimeVue Message: https://primevue.org/message/
- Existing RunLaunchButton: `frontend/src/features/runs/RunLaunchButton.vue`
- Existing RunLaunchConfirmDialog: `frontend/src/features/runs/RunLaunchConfirmDialog.vue`
- Existing useAsyncAction: `frontend/src/composables/useAsyncAction.ts`

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
