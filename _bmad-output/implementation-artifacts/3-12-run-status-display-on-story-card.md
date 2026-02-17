# Story 3.12: [FRONT] Run Status Display on Story Card

Status: ready-for-dev

## Story

As a user, I want to see run status on story cards in the board view, So that I can monitor execution progress at a glance.

## Acceptance Criteria (BDD)

**AC1: Display active run status with spinning icon**
- **Given** I am viewing a story card in the board view
- **When** the story has an active run (status = 'running')
- **Then** I see a spinning icon (PrimeVue ProgressSpinner, small) and "Running..." text in blue

**AC2: Display completed run status with relative time**
- **Given** I am viewing a story card in the board view
- **When** the story has a completed run
- **Then** I see a green check icon (pi pi-check-circle) and relative time text ("2h ago", "3d ago", etc.)

**AC3: Display failed run status with error details**
- **Given** I am viewing a story card in the board view
- **When** the story has a failed run
- **Then** I see a red X icon (pi pi-times-circle) and "Failed" text
- **And** clicking the status shows error details in a tooltip or dialog

**AC4: Display backlog/no-runs status**
- **Given** I am viewing a story card in the board view
- **When** the story has no runs or status is 'backlog'
- **Then** I see a gray dash icon (pi pi-minus-circle) and "Backlog" text

**AC5: Relative time updates automatically**
- **Given** I am viewing a completed run status
- **When** time passes
- **Then** the relative time text updates every minute ("2h ago" → "3h ago")

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create useRelativeTime composable (AC: #2, #5)
  - [ ] Accept MaybeRef<string | Date | null> date parameter
  - [ ] Return reactive computed string: "just now", "2m ago", "1h ago", "3d ago", "2w ago"
  - [ ] Use useIntervalFn from @vueuse/core to update every minute
  - [ ] Handle null/invalid dates gracefully

- [ ] [FRONT] Task 2: Build RunStatusIndicator.vue component (AC: #1, #2, #3, #4)
  - [ ] Props: status ('running' | 'completed' | 'failed' | 'backlog' | null), completedAt (string), errorMessage (string)
  - [ ] Emits: errorClick
  - [ ] Status config object with icon, spinner, text, color for each state
  - [ ] Zero custom CSS, Tailwind only

- [ ] [FRONT] Task 3: Add running state rendering (AC: #1)
  - [ ] PrimeVue ProgressSpinner (style="width: 1rem; height: 1rem", strokeWidth="4")
  - [ ] "Running..." text in blue (text-blue-500)
  - [ ] Flex layout with gap

- [ ] [FRONT] Task 4: Add completed state rendering (AC: #2)
  - [ ] pi pi-check-circle icon in green (text-green-500)
  - [ ] Use useRelativeTime composable to format completedAt
  - [ ] Display relative time text beside icon

- [ ] [FRONT] Task 5: Add failed state rendering (AC: #3)
  - [ ] pi pi-times-circle icon in red (text-red-500)
  - [ ] "Failed" text in red
  - [ ] Clickable: emit errorClick on click
  - [ ] Cursor pointer on hover

- [ ] [FRONT] Task 6: Add backlog/no-runs state rendering (AC: #4)
  - [ ] pi pi-minus-circle icon in gray (text-gray-400)
  - [ ] "Backlog" text in gray

- [ ] [FRONT] Task 7: Integrate RunStatusIndicator into StoryStatusCard.vue (AC: #1, #2, #3, #4)
  - [ ] Modify StoryStatusCard.vue from Story 2-5
  - [ ] Add RunStatusIndicator below or beside status badge
  - [ ] Derive run status from story.latest_run or story.status field
  - [ ] Handle errorClick: show error details in PrimeVue Tooltip or Toast

- [ ] [FRONT] Task 8: Write unit tests (AC: #1, #2, #3, #4, #5)
  - [ ] RunStatusIndicator.spec.ts: test all 4 states, errorClick emit
  - [ ] useRelativeTime.spec.ts: test time formatting, auto-update, null handling

- [ ] [FRONT] Task 9: Write E2E test for run status display (AC: #1, #2, #3, #4)
  - [ ] run-status.spec.ts: navigate to board → verify run status indicators on story cards for all states

## Dev Notes

### Dependencies

- Story 2-5: Epic detail (wave 6) — StoryStatusCard.vue component exists in features/board/
- Story 2-4: Board page (wave 5) — EpicCard with story counts
- Story 3-11: Run launch button (wave 5) — RunLaunchButton component exists, provides run creation flow
- Story 3-1: Runs API (wave 5) — GET /api/v1/projects/{projectId}/stories/{storyId}/runs provides latest run data
- Story 1-16: apiClient setup
- @vueuse/core: useIntervalFn for auto-updating relative time

### Architecture Requirements

Component hierarchy:
```
StoryStatusCard.vue (from Story 2-5, MODIFIED)
├── story key (monospace)
├── story title
├── status badge
└── RunStatusIndicator.vue (new component)
    ├── PrimeVue ProgressSpinner (running state)
    ├── pi pi-check-circle icon (completed state)
    ├── pi pi-times-circle icon (failed state, clickable)
    ├── pi pi-minus-circle icon (backlog state)
    └── relative time text (completed state)
```

Composable usage:
```
useRelativeTime.ts → RunStatusIndicator.vue
```

### File Paths (exact)

```
frontend/src/composables/useRelativeTime.ts
frontend/src/features/board/RunStatusIndicator.vue
frontend/src/features/board/StoryStatusCard.vue (modify)
frontend/src/__tests__/composables/useRelativeTime.spec.ts
frontend/src/__tests__/features/board/RunStatusIndicator.spec.ts
frontend/e2e/tests/run-status.spec.ts
```

### Technical Specifications

**useRelativeTime composable signature:**
```typescript
import { computed, type MaybeRef } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { unref } from 'vue'

export function useRelativeTime(date: MaybeRef<string | Date | null>) {
  const now = ref(Date.now())

  // Update every minute
  useIntervalFn(() => {
    now.value = Date.now()
  }, 60000)

  const relativeTime = computed(() => {
    const dateValue = unref(date)
    if (!dateValue) return null

    const d = typeof dateValue === 'string' ? new Date(dateValue) : dateValue
    if (isNaN(d.getTime())) return null

    const diff = now.value - d.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    const weeks = Math.floor(days / 7)

    if (seconds < 60) return 'just now'
    if (minutes < 60) return `${minutes}m ago`
    if (hours < 24) return `${hours}h ago`
    if (days < 7) return `${days}d ago`
    return `${weeks}w ago`
  })

  return relativeTime
}
```

**RunStatusIndicator.vue props/emits:**
```typescript
interface Props {
  status: 'running' | 'completed' | 'failed' | 'backlog' | null
  completedAt?: string  // ISO 8601 date string
  errorMessage?: string
}
const emit = defineEmits<{
  errorClick: []
}>()
```

**Status display mapping:**
```typescript
const statusConfig = {
  running: {
    icon: null,
    spinner: true,
    text: 'Running...',
    color: 'text-blue-500',
    clickable: false
  },
  completed: {
    icon: 'pi pi-check-circle',
    spinner: false,
    text: null,  // text = relative time from useRelativeTime
    color: 'text-green-500',
    clickable: false
  },
  failed: {
    icon: 'pi pi-times-circle',
    spinner: false,
    text: 'Failed',
    color: 'text-red-500',
    clickable: true
  },
  backlog: {
    icon: 'pi pi-minus-circle',
    spinner: false,
    text: 'Backlog',
    color: 'text-gray-400',
    clickable: false
  },
}
```

**RunStatusIndicator.vue template structure:**
```html
<template>
  <div
    class="flex items-center gap-2"
    :class="{ 'cursor-pointer': config.clickable }"
    @click="config.clickable ? emit('errorClick') : undefined"
  >
    <ProgressSpinner
      v-if="config.spinner"
      style="width: 1rem; height: 1rem"
      strokeWidth="4"
    />
    <i v-else-if="config.icon" :class="[config.icon, config.color]" />

    <span v-if="status === 'completed'" :class="config.color">
      {{ relativeTime }}
    </span>
    <span v-else-if="config.text" :class="config.color">
      {{ config.text }}
    </span>
  </div>
</template>
```

**Integration with StoryStatusCard.vue:**
```typescript
// In StoryStatusCard.vue (modified from Story 2-5)
// Add RunStatusIndicator below or beside the status badge

// Props:
interface Props {
  story: Story  // from generated API types
  isSelected: boolean
}

// Template structure (example):
<template>
  <div class="p-4 border rounded cursor-pointer" :class="selectedClass">
    <div class="font-mono text-sm text-gray-600">{{ story.key }}</div>
    <div class="font-medium">{{ story.title }}</div>

    <div class="flex items-center gap-2 mt-2">
      <Badge :value="story.status" :severity="statusSeverity" />
      <RunStatusIndicator
        :status="runStatus"
        :completed-at="story.latest_run?.completed_at"
        :error-message="story.latest_run?.error"
        @error-click="handleErrorClick"
      />
    </div>
  </div>
</template>

// Compute run status from story data:
const runStatus = computed(() => {
  if (!story.latest_run) return 'backlog'
  return story.latest_run.status // 'running' | 'completed' | 'failed'
})
```

**PrimeVue ProgressSpinner for running state:**
```html
<ProgressSpinner style="width: 1rem; height: 1rem" strokeWidth="4" />
```

**Error handling for failed runs:**
```typescript
// In StoryStatusCard.vue
import { useToast } from 'primevue/usetoast'

const toast = useToast()

const handleErrorClick = () => {
  if (props.story.latest_run?.error) {
    toast.add({
      severity: 'error',
      summary: 'Run Failed',
      detail: props.story.latest_run.error,
      life: 5000
    })
  }
}
```

### Testing Requirements

**Unit tests:**
- useRelativeTime: test time formatting (just now, 2m ago, 1h ago, 3d ago, 2w ago), null handling, auto-update interval
- RunStatusIndicator: test all 4 states render correctly (running with spinner, completed with time, failed with clickable, backlog), errorClick emit

**E2E tests:**
- Navigate to board page → see story cards with run status indicators
- Verify running state: spinner + "Running..." text
- Verify completed state: check icon + relative time
- Verify failed state: X icon + "Failed" text
- Click failed status → see error details
- Verify backlog state: dash icon + "Backlog" text

### References

- Epic 3: Pipeline Execution Engine
- Backend Story 3-1: Runs API (GET /api/v1/projects/{projectId}/stories/{storyId}/runs)
- Story 2-5: Epic detail (StoryStatusCard component)
- Story 3-11: Run launch button (wave 5)
- PrimeVue ProgressSpinner: https://primevue.org/progressspinner/
- PrimeVue Toast: https://primevue.org/toast/
- PrimeVue Badge: https://primevue.org/badge/
- @vueuse/core useIntervalFn: https://vueuse.org/shared/useIntervalFn/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
