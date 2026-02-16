# Frontend Agent Instructions — Vue 3 Patterns & Conventions

You are working exclusively in the `frontend/` directory. **NEVER** modify files outside `frontend/` except `api/openapi.yaml` (read-only reference).

## Technology Stack

- **Vue 3** — Composition API exclusively (no Options API)
- **TypeScript** — strict mode
- **PrimeVue 4** — Aura preset, unstyled mode with CSS layers
- **Tailwind CSS v4** — layout utilities only
- **Pinia** — state management
- **Vue Router** — routing with auth guards
- **openapi-fetch** — generated typed API client from OpenAPI spec
- **Vitest** — unit tests
- **Playwright** — E2E tests
- **Vite** — build tool and dev server
- **vee-validate + zod** — form validation
- **@vueuse/core** — utility composables

## Vue 3 Composition API Conventions

### Component Structure

All components use `<script setup>`:

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAsyncAction } from '@/composables/useAsyncAction'

const props = defineProps<{
  storyId: string
}>()

const emit = defineEmits<{
  updated: [story: Story]
}>()

// Reactive state
const isEditing = ref(false)

// Computed
const canEdit = computed(() => !isEditing.value)

// Logic
function handleSave() {
  emit('updated', story.value)
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- Template here -->
  </div>
</template>
```

### Rules

- **Pure Composition API only** — no Options API, no mixins
- **`<script setup>` always** — no `defineComponent()` wrapper
- **Props down, events up** — strictly enforced
- **Components are visual assemblers** — zero business logic in `.vue` files
- **Composables are the logic layer** — all reactive state management, API calls, event handling

## PrimeVue Component Usage Rules

### Core Principle

Use PrimeVue components for everything they provide. Never reinvent existing PrimeVue functionality.

### Available Components

Use these PrimeVue components instead of building custom ones:

| Need | PrimeVue Component |
|------|-------------------|
| Buttons | `Button` |
| Data display | `DataTable`, `DataView` |
| Dialogs/modals | `Dialog`, `ConfirmDialog` |
| Notifications | `Toast` service |
| Navigation | `Menubar`, `PanelMenu` |
| Status display | `Tag` with severity mapping |
| Badges | `Badge` |
| Text inputs | `InputText`, `Textarea`, `FloatLabel` |
| Select dropdowns | `Select` |
| Forms | `InputText`, `Select`, `Textarea`, `FloatLabel` |
| Story kanban | `DataView` + custom template |
| Run timeline | `Timeline` |
| Confirmation | `ConfirmDialog` service |

### Severity Props

Use PrimeVue severity props for status indication:

```vue
<Button label="Run" severity="success" @click="handleRun" />
<Tag :value="status" :severity="statusSeverity(status)" />
<Button label="Delete" severity="danger" @click="handleDelete" />
```

Never use inline styles for colors: `severity="danger"` NOT `:style="{ color: 'red' }"`.

### Override via Design Tokens

Override PrimeVue appearance via design tokens, NOT via custom CSS:

```typescript
// theme/tokens.ts
import { definePreset } from '@primevue/themes'
import Aura from '@primevue/themes/aura'

export const HopeTheme = definePreset(Aura, {
  // Override tokens here
})
```

### PrimeVue Setup

- **Mode:** Unstyled with Aura preset
- **CSS layers order:** `tailwind-base, primevue, tailwind-utilities` (Tailwind utilities override PrimeVue)
- **Theming:** Design tokens at 3 levels (primitive, semantic, component)
- **Dark mode:** `.dark` class on `<html>`, persisted in localStorage via `useTheme()` composable

## Tailwind CSS Layout-Only Usage

### Use Tailwind ONLY for Layout

Tailwind is for structural layout only:
- `flex`, `grid`, `gap`, `p-*`, `m-*`, `w-*`, `h-*`
- `items-center`, `justify-between`, `space-x-*`
- Responsive breakpoints: `md:`, `lg:`

### DO NOT Use Tailwind For

- Colors (use PrimeVue design tokens)
- Typography (use PrimeVue text styles)
- Borders and shadows (use PrimeVue component styles)
- Component-level styling (use PrimeVue severity/variants)

### Example

```vue
<template>
  <div class="flex flex-col gap-4 p-6">
    <div class="flex items-center justify-between">
      <h1>Projects</h1>
      <Button label="Create" severity="success" />
    </div>
    <DataTable :value="projects" />
  </div>
</template>
```

### CSS Rules

- No `<style scoped>` blocks except for complex animations or SVG
- No inline styles
- CSS layer order in `assets/main.css`:
  ```css
  @layer tailwind-base, primevue, tailwind-utilities;
  ```

## useAsyncAction Pattern

Every async operation wraps in `useAsyncAction`:

```typescript
import { useAsyncAction } from '@/composables/useAsyncAction'

const { execute, isLoading, error, data } = useAsyncAction(
  async (storyId: string) => {
    const response = await apiClient.GET('/api/v1/stories/{id}', {
      params: { path: { id: storyId } }
    })
    return response.data
  }
)

// In template
// <ProgressSpinner v-if="isLoading" />
// <Message v-if="error" severity="error" :text="error.message" />
// <StoryDetail v-if="data" :story="data" />
```

Components render based on `isLoading`, `error`, and `data` states. Every page and feature uses this pattern consistently.

## Pinia Store Patterns

### Store Structure

Use the setup store syntax (Composition API style):

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useStoriesStore = defineStore('stories', () => {
  // State
  const stories = ref<Story[]>([])
  const isLoading = ref(false)

  // Getters
  const storyCount = computed(() => stories.value.length)
  const byStatus = computed(() => (status: string) =>
    stories.value.filter(s => s.status === status)
  )

  // Actions
  async function fetchStories(projectId: string) {
    isLoading.value = true
    try {
      const response = await apiClient.GET('/api/v1/projects/{id}/stories', {
        params: { path: { id: projectId } }
      })
      stories.value = response.data || []
    } finally {
      isLoading.value = false
    }
  }

  function handleSSEEvent(event: SSEEvent) {
    // Update store reactively from SSE events
    if (event.type === 'story.updated') {
      const idx = stories.value.findIndex(s => s.id === event.payload.id)
      if (idx >= 0) stories.value[idx] = event.payload
    }
  }

  return { stories, isLoading, storyCount, byStatus, fetchStories, handleSSEEvent }
})
```

### Store Organization

One store per domain:
- `stores/auth.ts`
- `stores/projects.ts`
- `stores/stories.ts`
- `stores/runs.ts`
- `stores/approvals.ts`
- `stores/templates.ts`

SSE events update stores reactively via the `useSSE` composable dispatching to store handlers.

## openapi-fetch API Client Usage

### Client Setup

```typescript
// api/client.ts
import createClient from 'openapi-fetch'
import type { paths } from './generated/schema'

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include', // JWT in httpOnly cookie
})
```

### Usage

```typescript
// GET request
const { data, error } = await apiClient.GET('/api/v1/stories/{id}', {
  params: { path: { id: storyId } }
})

// POST request
const { data, error } = await apiClient.POST('/api/v1/projects', {
  body: { name: projectName, description }
})

// List with query params
const { data, error } = await apiClient.GET('/api/v1/projects/{id}/stories', {
  params: {
    path: { id: projectId },
    query: { status: 'backlog', per_page: 20 }
  }
})
```

### Type Generation

All types are generated from `api/openapi.yaml`:

```bash
cd frontend && npm run generate-api
```

This generates TypeScript types and the typed `paths` interface. Never manually write types that should come from the OpenAPI spec.

## Component Organization

### Directory Structure

```
frontend/src/
├── ui/                          # Atomic layer (shared only)
│   ├── primitives/              # PrimeVue wrappers, base components
│   │   ├── StatusBadge.vue      # Badge with status → color mapping
│   │   ├── CodeBlock.vue        # Code/log display
│   │   └── EmptyState.vue
│   ├── composed/                # Reusable combinations
│   │   ├── DataTable.vue        # Table + pagination + loading + empty
│   │   ├── ConfirmDialog.vue    # Dialog + standardized actions
│   │   ├── LogViewer.vue        # Stream logs with ANSI support
│   │   └── TimelineStep.vue     # Visual step with state
│   └── layout/                  # Page structure
│       ├── AppShell.vue         # Sidebar + header + content
│       ├── PageHeader.vue       # Title + breadcrumb + actions
│       └── SplitPanel.vue       # Resizable panel
│
├── features/                    # By business domain
│   ├── projects/
│   │   ├── ProjectList.vue
│   │   ├── ProjectSettings.vue
│   │   └── composables/useProjectForm.ts
│   ├── stories/
│   │   ├── StoryBoard.vue       # Kanban view
│   │   ├── StoryDetail.vue
│   │   ├── StoryEditor.vue
│   │   └── composables/useStoryFilters.ts
│   ├── runs/
│   │   ├── RunTimeline.vue      # Step timeline
│   │   ├── RunDetail.vue
│   │   ├── StepOutput.vue       # NDJSON logs for a step
│   │   └── composables/useRunPolling.ts
│   ├── dag/
│   │   ├── DagGraph.vue         # DAG visualization
│   │   ├── DagControls.vue
│   │   └── composables/useDagLayout.ts
│   ├── approvals/
│   │   ├── ApprovalQueue.vue
│   │   ├── DiffViewer.vue
│   │   └── composables/useApprovalActions.ts
│   └── pipeline-editor/
│       ├── PipelineCanvas.vue   # Visual YAML editor
│       ├── ActionPalette.vue    # Available actions list
│       ├── StepConfigForm.vue
│       └── composables/usePipelineValidation.ts
│
├── composables/                 # Shared functional (pure)
│   ├── useSSE.ts                # EventSource + dispatch to stores
│   ├── useAuth.ts               # JWT lifecycle
│   ├── usePagination.ts         # Generic pagination logic
│   ├── useAsyncAction.ts        # loading + error + execute pattern
│   └── useKeyboard.ts           # Keyboard shortcuts
│
├── stores/                      # Pinia stores
├── api/                         # openapi-fetch client
├── theme/                       # PrimeVue tokens + config
├── assets/                      # main.css
├── router/                      # Routes with auth guards
├── views/                       # 1 view = 1 route, composes features
└── utils/                       # Pure functions (formatters, parsers)
```

### Component Placement Rule

**If used by 2+ features** → `ui/`
**Otherwise** → stays in its feature directory

### View Components

Views are 1:1 with routes and compose feature components:

```
views/
├── LoginView.vue
├── DashboardView.vue
├── ProjectsView.vue
├── ProjectDetailView.vue
├── StoriesView.vue
├── RunDetailView.vue
├── ApprovalsView.vue
├── DagView.vue
└── PipelineEditorView.vue
```

## SSE (Server-Sent Events) Client

### useSSE Composable

```typescript
// composables/useSSE.ts
import { useEventSource } from '@vueuse/core'

export function useSSE(projectId: string) {
  const { data, error, close } = useEventSource(
    `/api/v1/events/stream?project_id=${projectId}`
  )

  // Auto-reconnect on disconnect
  // Dispatch events to relevant Pinia stores
  // Cleanup on component unmount
}
```

### Event Types

- `run.started` — new run began
- `step.completed` — pipeline step finished
- `step.failed` — pipeline step failed
- `hitl.pending` — human approval needed
- `run.completed` — entire run finished
- `log.line` — agent log output line

## Key Frontend Libraries

| Library | Usage |
|---------|-------|
| `@vue-flow/core` | DAG visualization |
| `@guolao/vue-monaco-editor` | YAML pipeline editor |
| `ansi-to-html` | Agent log rendering |
| `diff2html` | PR diff rendering |
| `vee-validate` + `zod` | Form validation |
| `@vueuse/core` | Utility composables (useLocalStorage, useEventSource, useDebounceFn) |
| `date-fns` | Date formatting (tree-shakeable) |

## Functional Patterns

### Loading States

Every async operation uses `useAsyncAction` pattern:
- `isLoading` — show PrimeVue `Skeleton` or `ProgressSpinner`
- `error` — show inline `Message` for validation (400), `Toast` for transient errors (500)
- `data` — render the actual content

### Error Recovery

- API errors → fetch error → frontend error ref
- `Toast` for transient errors (network, 500)
- Inline error display for validation (400)
- Redirect to login on 401

### Props Down, Events Up

```vue
<!-- Parent -->
<StoryCard :story="story" @updated="handleUpdate" />

<!-- Child -->
<script setup lang="ts">
const props = defineProps<{ story: Story }>()
const emit = defineEmits<{ updated: [story: Story] }>()
</script>
```

## Testing Patterns

### Vitest Unit Tests

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StoryCard from '../StoryCard.vue'

describe('StoryCard', () => {
  it('displays story title', () => {
    const wrapper = mount(StoryCard, {
      props: { story: { title: 'Test Story', status: 'backlog' } }
    })
    expect(wrapper.text()).toContain('Test Story')
  })
})
```

### What to Test

| Target | What to Test | Coverage Target |
|--------|-------------|----------------|
| Composables | Reactive logic, edge cases, error paths | 95%+ |
| Pinia stores | Actions, mutations, SSE event handlers | 90%+ |
| Utils | Formatters, parsers, validators (pure functions) | 100% |
| Zod schemas | Validation rules | 100% |
| Components | Only those with complex conditional logic | As needed |

Do NOT test that PrimeVue renders a button correctly — that is PrimeVue's responsibility.

### Test Organization

Tests live co-located in `__tests__/` directories next to source files:

```
features/stories/
├── StoryBoard.vue
├── StoryDetail.vue
├── composables/
│   └── useStoryFilters.ts
└── __tests__/
    ├── StoryBoard.spec.ts
    └── useStoryFilters.spec.ts
```

### Playwright E2E Tests

```typescript
import { test, expect } from '@playwright/test'

test('launch story run', async ({ page }) => {
  await page.goto('/projects/1/stories')
  await page.click('text=Run Story')
  await expect(page.locator('.run-timeline')).toBeVisible()
})
```

E2E tests in `frontend/e2e/tests/` cover critical user journey flows.

### Test Commands

```bash
# Unit tests
npm run test:unit

# Unit tests in watch mode
npm run test:unit -- --watch

# E2E tests
npm run test:e2e

# Lint
npm run lint

# Type check
npm run type-check   # tsc --noEmit
```

## Vue/TypeScript Naming Conventions

- Components: `PascalCase.vue` (`RunTimeline.vue`, `StoryBoard.vue`)
- Composables: `use` prefix, `camelCase` (`useSSE.ts`, `useAsyncAction.ts`)
- Stores: domain noun (`auth.ts`, `runs.ts`, `stories.ts`)
- Utils: `camelCase.ts` (`formatDate.ts`, `parseNdjson.ts`)
- Types/interfaces: `PascalCase` (`Run`, `Story`, `SSEEvent`)
- Props: `camelCase` (Vue convention)
- Events: `camelCase` (Vue convention)

## Build & Development

```bash
# Install dependencies
cd frontend && npm install

# Development server
npm run dev

# Production build
npm run build

# Preview production build
npm run preview

# Generate API types from OpenAPI spec
npm run generate-api

# Lint
npm run lint

# Type check
npm run type-check

# Format
npm run format
```
