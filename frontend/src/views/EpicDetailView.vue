<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import EpicDetailLayout from '@/features/board/EpicDetailLayout.vue'
import { useStories } from '@/composables/useStories'

const route = useRoute()
const router = useRouter()

const projectId = route.params.id as string
const epicId = route.params.epicId as string

const {
  stories,
  selectedStory,
  selectedStoryId,
  filters,
  isLoading,
  error,
  retry,
  setFilters,
  selectStory,
} = useStories(projectId, epicId)

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

    <EpicDetailLayout
      v-else
      class="flex-1 min-h-0"
      :stories="stories"
      :selected-story="selectedStory"
      :selected-story-id="selectedStoryId"
      :filters="filters"
      @select="selectStory"
      @update:filters="setFilters"
    />
  </div>
</template>
