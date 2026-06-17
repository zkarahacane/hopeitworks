<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { VueFlow } from '@vue-flow/core'
import { Controls } from '@vue-flow/controls'
import { MiniMap } from '@vue-flow/minimap'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/controls/dist/style.css'
import '@vue-flow/minimap/dist/style.css'
import ProgressBar from 'primevue/progressbar'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'
import EpicRunStatusNode from '@/features/epics/EpicRunStatusNode.vue'
import EpicRunGroupList from '@/features/epics/EpicRunGroupList.vue'
import { useEpicRunMonitor } from '@/features/epics/composables/useEpicRunMonitor'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string
const epicRunId = route.params.epicRunId as string

const { epicRunStore, nodes, edges, sseStatus } = useEpicRunMonitor(projectId, epicRunId)

const truncatedId = computed(() =>
  epicRunId.length > 8 ? epicRunId.slice(0, 8) + '\u2026' : epicRunId,
)

const sseSeverity = computed(() => {
  const map: Record<string, 'success' | 'info' | 'danger' | 'secondary'> = {
    open: 'success',
    connecting: 'info',
    error: 'danger',
    closed: 'secondary',
  }
  return map[sseStatus.value] ?? 'secondary'
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const nodeTypes: Record<string, any> = {
  epicRunStatus: markRaw(EpicRunStatusNode),
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function handleNodeClick(_event: any) {
  const runId = _event.node?.data?.runId
  if (runId) {
    router.push({ name: 'run-detail', params: { id: runId } })
  }
}

function handleBack() {
  const epicId = epicRunStore.epicRun?.epic_id
  if (epicId) {
    router.push({ name: 'epic-dag', params: { id: projectId, epicId } })
  } else {
    router.push({ name: 'project-board', params: { id: projectId } })
  }
}
</script>

<template>
  <div class="flex flex-col h-full p-6 gap-4">
    <!-- Header -->
    <div class="flex items-center gap-3">
      <Button
        icon="pi pi-arrow-left"
        severity="secondary"
        text
        rounded
        aria-label="Back"
        @click="handleBack"
      />
      <h1 class="m-0 text-2xl font-bold flex-1">Epic Run Monitor</h1>
      <code
        class="text-sm px-2 py-1 rounded font-mono"
        :style="{ background: 'var(--surface-overlay)' }"
      >
        {{ truncatedId }}
      </code>
      <Tag :value="sseStatus" :severity="sseSeverity" class="text-xs" />
    </div>

    <!-- Loading -->
    <div v-if="epicRunStore.isLoading" class="flex flex-col gap-4">
      <Skeleton width="100%" height="2rem" />
      <Skeleton width="100%" height="12rem" />
    </div>

    <!-- Error -->
    <Message v-else-if="epicRunStore.error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ epicRunStore.error }}</span>
        <Button
          label="Retry"
          severity="secondary"
          text
          size="small"
          @click="epicRunStore.fetchEpicRun(projectId, epicRunId)"
        />
      </div>
    </Message>

    <!-- Content -->
    <template v-else-if="epicRunStore.epicRun">
      <!-- Progress Bar -->
      <div class="flex flex-col gap-1">
        <ProgressBar :value="epicRunStore.progressPercent" :show-value="false" />
        <span class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          {{ epicRunStore.completedCount }} / {{ epicRunStore.totalCount }} stories completed
        </span>
      </div>

      <!-- Completion Summary -->
      <Message
        v-if="epicRunStore.epicRun.status === 'completed'"
        severity="success"
        :closable="false"
      >
        All stories completed successfully
      </Message>
      <Message
        v-else-if="epicRunStore.epicRun.status === 'failed'"
        severity="error"
        :closable="false"
      >
        <div class="flex flex-col gap-1">
          <span>Epic run failed. Failed stories:</span>
          <ul class="list-disc list-inside">
            <li v-for="story in epicRunStore.failedStories" :key="story.story_id">
              <router-link
                v-if="story.run_id"
                :to="{ name: 'run-detail', params: { id: story.run_id } }"
                class="underline"
              >
                {{ story.story_key }}
              </router-link>
              <span v-else>{{ story.story_key }}</span>
            </li>
          </ul>
        </div>
      </Message>

      <!-- Execution Layers -->
      <EpicRunGroupList :stories="epicRunStore.epicRun.stories" />

      <!-- DAG Graph -->
      <div class="dag-container flex-1">
        <VueFlow
          :nodes="nodes"
          :edges="edges"
          :node-types="nodeTypes"
          fit-view-on-init
          class="h-full"
          @node-click="handleNodeClick"
        >
          <Controls />
          <MiniMap />
        </VueFlow>
      </div>
    </template>
  </div>
</template>

<style scoped>
.dag-container {
  min-height: 400px;
}
</style>
