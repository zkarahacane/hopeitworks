<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import EpicCardGrid from '@/features/board/EpicCardGrid.vue'
import BoardEmptyState from '@/features/board/BoardEmptyState.vue'
import { useEpics } from '@/composables/useEpics'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const projectId = route.params.id as string
const { epics, isLoading, error, retry } = useEpics(projectId)

const isAdmin = computed(() => authStore.user?.role === 'admin')

function handleEpicClick(epicId: string) {
  router.push({ name: 'epic-detail', params: { id: projectId, epicId } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Story Board</h1>
    </div>

    <div v-if="isLoading && epics.length === 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="n in 6" :key="n" class="flex flex-col gap-3 p-4">
        <Skeleton width="60%" height="1.5rem" />
        <Skeleton width="100%" height="1rem" />
        <Skeleton width="80%" height="1rem" />
        <div class="flex gap-2">
          <Skeleton width="3rem" height="1.5rem" />
          <Skeleton width="3rem" height="1.5rem" />
          <Skeleton width="3rem" height="1.5rem" />
          <Skeleton width="3rem" height="1.5rem" />
        </div>
      </div>
    </div>

    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <BoardEmptyState
      v-else-if="!isLoading && !error && epics.length === 0"
      :is-admin="isAdmin"
      @create-epic="() => {}"
    />

    <EpicCardGrid
      v-else
      :epics="epics"
      @epic-click="handleEpicClick"
    />
  </div>
</template>
