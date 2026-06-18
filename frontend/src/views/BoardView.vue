<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import SelectButton from 'primevue/selectbutton'
import Select from 'primevue/select'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import Button from 'primevue/button'
import KanbanBoard from '@/features/board/KanbanBoard.vue'
import StoryDetailPanel from '@/features/board/StoryDetailPanel.vue'
import { useBoard } from '@/composables/useBoard'
import { useProject } from '@/composables/useProject'
import { useStoriesStore } from '@/stores/stories'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import RunLaunchConfirmDialog from '@/features/runs/RunLaunchConfirmDialog.vue'
import { useRunLauncher, ALREADY_RUNNING_ERROR } from '@/composables/useRunLauncher'

const route = useRoute()
const projectId = route.params.id as string

const storiesStore = useStoriesStore()

const {
  epics,
  isLoadingEpics,
  epicsError,
  selectedEpicId,
  setEpicId,
  stories,
  isLoadingStories,
  storiesError,
} = useBoard(projectId)

const selectedEpicName = computed(() => {
  if (!selectedEpicId.value) return 'All epics'
  const epic = epics.value.find((e) => e.id === selectedEpicId.value)
  return epic?.name ?? 'Unknown epic'
})

const { project } = useProject(projectId)

// ── PLANNED IN segmented control ─────────────────────────────────────────────
// Frontend-only derivation: maps git_provider to a planning source label.
// Backend gap: no planning_source field on Epic or Project — this is a
// UI-layer heuristic until the backend exposes an explicit planning_source.

interface PlanningOption {
  label: string
  value: string
}

const planningOptions: PlanningOption[] = [
  { label: '✓ GitHub Issues', value: 'github' },
  { label: 'GitLab Issues', value: 'gitlab' },
  { label: 'BMAD', value: 'bmad' },
  { label: 'Jira', value: 'jira' },
  { label: 'Markdown', value: 'markdown' },
]

/**
 * The active planning source — null means "use derived default".
 * Initialized to null so we can show the derived value as highlighted
 * without hardcoding it before the project loads.
 *
 * Backend gap: no planning_source field on Epic or Project.
 * This computed is the source-of-truth for the SelectButton value.
 */
const selectedPlanningSource = ref<string | null>(null)

const derivedPlanningSource = computed((): string => {
  const provider = project.value?.git_provider
  if (provider === 'github') return 'github'
  if (provider === 'gitlab') return 'gitlab'
  return 'github' // sensible default
})

/**
 * The value actually bound to SelectButton.
 * Falls back to the git_provider-derived source when user hasn't picked.
 */
const activePlanningSource = computed({
  get: () => selectedPlanningSource.value ?? derivedPlanningSource.value,
  set: (val: string | null) => { selectedPlanningSource.value = val },
})

// ── Epic selector ─────────────────────────────────────────────────────────────

interface EpicOption {
  label: string
  value: string
}

const epicOptions = computed((): EpicOption[] => [
  { label: 'All epics', value: '' },
  ...epics.value.map((e) => ({ label: e.name, value: e.id })),
])

const epicSelectValue = computed({
  get: () => selectedEpicId.value ?? '',
  set: (val: string) => setEpicId(val === '' ? null : val),
})

// ── Story selection ───────────────────────────────────────────────────────────

const selectedStoryId = computed(() => storiesStore.selectedStoryId)

function handleSelectStory(storyId: string) {
  storiesStore.setSelectedStory(storyId === selectedStoryId.value ? null : storyId)
}

function handleStoryUpdated() {
  // story updated in panel — store already reflects change via updateStory
}

// ── Run launch ────────────────────────────────────────────────────────────────

const toast = useToast()
const dialogVisible = ref(false)
const { isLoading: launchLoading, error: launchError, launchRun } = useRunLauncher()

function handleLaunchClick() {
  dialogVisible.value = true
}

async function handleConfirm() {
  const story = storiesStore.selectedStory
  if (!story) return

  const result = await launchRun(projectId, story.id)

  if (result !== null) {
    toast.add({
      severity: 'success',
      summary: 'Run launched',
      detail: `Run started for ${story.key}`,
      life: 3000,
    })
    dialogVisible.value = false
    return
  }

  if (launchError.value?.message === ALREADY_RUNNING_ERROR) {
    toast.add({
      severity: 'warn',
      summary: 'Already running',
      detail: 'This story already has a run in progress',
      life: 5000,
    })
    return
  }

  toast.add({
    severity: 'error',
    summary: 'Launch failed',
    detail: launchError.value?.message ?? 'An unexpected error occurred',
    life: 5000,
  })
  dialogVisible.value = false
}
</script>

<template>
  <div class="flex flex-col h-full overflow-hidden">
    <Toast />
    <!-- ── Header ──────────────────────────────────────────────────────────── -->
    <div class="flex items-start justify-between gap-4 px-6 pt-6 pb-4 shrink-0">
      <div class="flex flex-col gap-1">
        <!-- Breadcrumb -->
        <div class="flex items-center gap-1" style="font-size: 0.8rem; color: var(--p-text-muted-color)">
          <span>{{ project?.name ?? 'Project' }}</span>
          <i class="pi pi-chevron-right" style="font-size: 0.65rem" aria-hidden="true" />
          <span>Board</span>
        </div>
        <!-- Title -->
        <h1
          class="m-0"
          style="font-family: var(--font-sans); font-size: 1.5rem; font-weight: 700"
        >
          Story Board
        </h1>
        <!-- Subtitle -->
        <p
          class="m-0"
          style="font-size: 0.82rem; color: var(--p-text-muted-color)"
        >
          Epic · {{ selectedEpicName }} — board generated from your stories, kept live by the runtime
        </p>
      </div>

      <!-- PLANNED IN segmented control -->
      <div class="flex flex-col items-end gap-1 shrink-0">
        <span style="font-size: 0.72rem; color: var(--p-text-muted-color); text-transform: uppercase; letter-spacing: 0.05em">
          Planned in
        </span>
        <SelectButton
          v-model="activePlanningSource"
          :options="planningOptions"
          option-label="label"
          option-value="value"
          :allow-empty="true"
          :pt="{ root: { style: 'font-size: 0.78rem' } }"
        />
      </div>
    </div>

    <!-- ── Epic selector ───────────────────────────────────────────────────── -->
    <div class="px-6 pb-3 shrink-0">
      <div class="flex items-center gap-3">
        <label
          for="epic-select"
          style="font-size: 0.82rem; color: var(--p-text-muted-color); white-space: nowrap"
        >
          Epic
        </label>
        <Select
          id="epic-select"
          v-model="epicSelectValue"
          :options="epicOptions"
          option-label="label"
          option-value="value"
          :loading="isLoadingEpics"
          placeholder="All epics"
          style="min-width: 220px; font-size: 0.85rem"
        />
        <Message
          v-if="epicsError"
          severity="warn"
          :closable="false"
          style="font-size: 0.8rem; padding: 0.25rem 0.5rem"
        >
          {{ epicsError }}
        </Message>
      </div>
    </div>

    <!-- ── Board + detail panel ────────────────────────────────────────────── -->
    <div class="flex flex-1 overflow-hidden gap-0">
      <!-- Kanban area -->
      <div class="flex-1 overflow-auto px-6 pb-6">
        <!-- Loading skeleton -->
        <div
          v-if="isLoadingStories && stories.length === 0"
          class="flex gap-3"
        >
          <div
            v-for="n in 5"
            :key="n"
            class="flex flex-col gap-2 min-w-[220px]"
          >
            <Skeleton width="100%" height="1.25rem" class="mb-1" />
            <div v-for="m in 3" :key="m" class="flex flex-col gap-2 p-3" style="border: 1px solid var(--surface-border); border-radius: var(--p-border-radius)">
              <Skeleton width="60%" height="0.9rem" />
              <Skeleton width="100%" height="0.9rem" />
              <Skeleton width="80%" height="0.9rem" />
            </div>
          </div>
        </div>

        <!-- Error state -->
        <Message v-else-if="storiesError" severity="error" :closable="false">
          <div class="flex items-center gap-3">
            <span>{{ storiesError }}</span>
            <Button
              label="Retry"
              severity="secondary"
              text
              size="small"
              @click="selectedEpicId && setEpicId(selectedEpicId)"
            />
          </div>
        </Message>

        <!-- Empty state when no epic selected -->
        <div
          v-else-if="!selectedEpicId && !isLoadingEpics && epics.length === 0"
          class="flex flex-col items-center justify-center h-full gap-3"
          style="color: var(--p-text-muted-color)"
        >
          <i class="pi pi-th-large" style="font-size: 2.5rem" aria-hidden="true" />
          <p style="font-size: 1rem">No epics found. Import stories to get started.</p>
        </div>

        <!-- Kanban board -->
        <KanbanBoard
          v-else
          :stories="stories"
          :selected-id="selectedStoryId"
          :project-id="projectId"
          @select="handleSelectStory"
        />
      </div>

      <!-- Story detail panel (slides in when a story is selected) -->
      <Transition name="panel-slide">
        <div
          v-if="selectedStoryId"
          class="shrink-0 overflow-hidden"
          style="width: 380px; border-left: 1px solid var(--surface-border)"
        >
          <StoryDetailPanel
            :story="storiesStore.selectedStory"
            :all-stories="stories"
            :project-id="projectId"
            :show-launch-button="true"
            @select-dependency="handleSelectStory"
            @launch-click="handleLaunchClick"
            @story-updated="handleStoryUpdated"
          />
        </div>
      </Transition>
    </div>

    <RunLaunchConfirmDialog
      v-if="storiesStore.selectedStory"
      v-model:visible="dialogVisible"
      :story-key="storiesStore.selectedStory.key"
      :story-title="storiesStore.selectedStory.title"
      :loading="launchLoading"
      @confirm="handleConfirm"
      @cancel="dialogVisible = false"
    />
  </div>
</template>

<style scoped>
.panel-slide-enter-active,
.panel-slide-leave-active {
  transition: width 0.25s ease, opacity 0.25s ease;
  overflow: hidden;
}
.panel-slide-enter-from,
.panel-slide-leave-to {
  width: 0 !important;
  opacity: 0;
}
</style>
