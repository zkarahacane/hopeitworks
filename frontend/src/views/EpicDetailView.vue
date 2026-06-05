<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import SelectButton from 'primevue/selectbutton'
import EpicDetailLayout from '@/features/board/EpicDetailLayout.vue'
import KanbanBoard from '@/features/board/KanbanBoard.vue'
import RunLaunchConfirmDialog from '@/features/runs/RunLaunchConfirmDialog.vue'
import CreateStoryDialog from '@/features/board/CreateStoryDialog.vue'
import { useStories } from '@/composables/useStories'
import { useRunLauncher, ALREADY_RUNNING_ERROR } from '@/composables/useRunLauncher'
import { useSSE } from '@/composables/useSSE'
import { useStoriesStore } from '@/stores/stories'

const route = useRoute()
const router = useRouter()

const projectId = route.params.id as string
const epicId = route.params.epicId as string

const toast = useToast()
const storiesStore = useStoriesStore()

const {
  stories,
  allStories,
  selectedStory,
  selectedStoryId,
  filters,
  isLoading,
  error,
  retry,
  setFilters,
  selectStory,
} = useStories(projectId, epicId)

// ── SSE: wire events to the store so the board updates live ─────────────────
useSSE(projectId, (name, data) => storiesStore.handleSSEEvent(name, data))

// ── View toggle: list vs kanban ─────────────────────────────────────────────
const VIEW_OPTIONS = [
  { label: 'List', value: 'list' },
  { label: 'Kanban', value: 'kanban' },
]
const activeView = ref<'list' | 'kanban'>('list')

const dialogVisible = ref(false)
const createDialogVisible = ref(false)
const { isLoading: launchLoading, error: launchError, launchRun } = useRunLauncher()

function handleLaunchClick() {
  dialogVisible.value = true
}

function handleCreateStory() {
  createDialogVisible.value = true
}

function handleStoryCreated() {
  toast.add({
    severity: 'success',
    summary: 'Story created',
    detail: 'New story has been created',
    life: 3000,
  })
}

function handleStoryUpdated() {
  toast.add({
    severity: 'success',
    summary: 'Story updated',
    detail: 'Story has been updated',
    life: 3000,
  })
}

async function handleConfirm() {
  if (!selectedStory.value) return

  const result = await launchRun(projectId, selectedStory.value.id)

  if (result !== null) {
    toast.add({
      severity: 'success',
      summary: 'Run launched',
      detail: `Run started for ${selectedStory.value.key}`,
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

/** Initialize filters from URL query params */
const initialStatus = (route.query.status as string) || null
const initialSearch = (route.query.search as string) || ''
if (initialStatus || initialSearch) {
  setFilters({ status: initialStatus, search: initialSearch })
}

/** Sync filters to URL query params — skip the initial render to avoid redundant replace */
const filtersInitialized = ref(false)
watch(
  filters,
  (newFilters) => {
    if (!filtersInitialized.value) {
      filtersInitialized.value = true
      return
    }
    router.replace({
      query: {
        ...route.query,
        status: newFilters.status && newFilters.status !== 'all' ? newFilters.status : undefined,
        search: newFilters.search || undefined,
      },
    })
  },
  { deep: true },
)
</script>

<template>
  <div class="flex flex-col h-full p-6">
    <Toast />
    <div class="flex items-center gap-3 mb-4">
      <Button
        icon="pi pi-arrow-left"
        severity="secondary"
        text
        rounded
        aria-label="Back to board"
        @click="router.push({ name: 'project-board', params: { id: projectId } })"
      />
      <h1 class="m-0 text-2xl font-bold">Epic Stories</h1>
      <Button
        icon="pi pi-sitemap"
        label="View DAG"
        severity="secondary"
        @click="router.push({ name: 'epic-dag', params: { id: projectId, epicId } })"
      />

      <!-- List / Kanban toggle -->
      <div class="ml-auto">
        <SelectButton
          v-model="activeView"
          :options="VIEW_OPTIONS"
          option-label="label"
          option-value="value"
          aria-label="Switch view"
        />
      </div>
    </div>

    <div v-if="isLoading && stories.length === 0" class="flex gap-4 flex-1">
      <div class="w-[300px] shrink-0 flex flex-col gap-3">
        <Skeleton width="100%" height="2.5rem" />
        <Skeleton width="100%" height="2.5rem" />
        <div v-for="n in 5" :key="n" class="flex flex-col gap-2 p-3">
          <div class="flex justify-between">
            <Skeleton width="4rem" height="1rem" />
            <Skeleton width="3rem" height="1.25rem" />
          </div>
          <Skeleton width="80%" height="1rem" />
        </div>
      </div>
      <div class="flex-1 flex flex-col gap-3 p-4">
        <Skeleton width="6rem" height="1rem" />
        <Skeleton width="60%" height="1.5rem" />
        <Skeleton width="100%" height="3rem" />
        <Skeleton width="100%" height="3rem" />
      </div>
    </div>

    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <template v-else>
      <!-- List view (default) -->
      <EpicDetailLayout
        v-if="activeView === 'list'"
        class="flex-1 min-h-0"
        :stories="stories"
        :all-stories="allStories"
        :selected-story="selectedStory"
        :selected-story-id="selectedStoryId"
        :filters="filters"
        :project-id="projectId"
        @select="selectStory"
        @update:filters="setFilters"
        @launch-click="handleLaunchClick"
        @create-story="handleCreateStory"
        @story-updated="handleStoryUpdated"
      />

      <!-- Kanban view: uses storiesStore.items directly for live reactivity -->
      <KanbanBoard
        v-else
        class="flex-1 min-h-0"
        :stories="storiesStore.items"
        :selected-id="selectedStoryId"
        @select="selectStory"
      />
    </template>

    <RunLaunchConfirmDialog
      v-if="selectedStory"
      v-model:visible="dialogVisible"
      :story-key="selectedStory.key"
      :story-title="selectedStory.title"
      :loading="launchLoading"
      @confirm="handleConfirm"
      @cancel="dialogVisible = false"
    />

    <CreateStoryDialog
      v-model:visible="createDialogVisible"
      :project-id="projectId"
      :epic-id="epicId"
      @created="handleStoryCreated"
    />
  </div>
</template>
