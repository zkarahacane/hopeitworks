<script setup lang="ts">
import { onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'
import EpicList from '@/features/stories/EpicList.vue'
import EpicEmptyState from '@/features/stories/EpicEmptyState.vue'
import { useEpics } from '@/composables/useEpics'
import type { Epic } from '@/stores/epics'

const route = useRoute()
const router = useRouter()
const { epics, isLoading, error, fetchEpics, retry } = useEpics()

const projectId = route.params.id as string

onMounted(() => {
  fetchEpics(projectId)
})

function handleEpicClick(epic: Epic) {
  router.push({
    name: 'epic-detail',
    params: { id: projectId, epicId: epic.id },
  })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Story Board</h1>
      <Button
        label="Import Stories"
        icon="pi pi-upload"
        severity="secondary"
        disabled
      />
    </div>

    <ProgressSpinner
      v-if="isLoading && epics.length === 0"
      class="flex justify-center"
    />

    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <EpicEmptyState
      v-else-if="!isLoading && !error && epics.length === 0"
      @import="() => {}"
    />

    <EpicList
      v-else
      :epics="epics"
      @epic-click="handleEpicClick"
    />
  </div>
</template>
