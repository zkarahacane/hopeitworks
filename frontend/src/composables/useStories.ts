import { computed, onMounted, ref, watch } from 'vue'
import { useStoriesStore, type StoryFilters } from '@/stores/stories'

/**
 * Composable for story list operations within an epic.
 * Wraps the stories store with reactive computed properties, auto-fetch, and retry logic.
 */
export function useStories(projectId: string, epicId: string) {
  const store = useStoriesStore()
  const lastProjectId = ref(projectId)
  const lastEpicId = ref(epicId)

  /** Fetch stories for the given epic */
  async function fetchStories() {
    await store.fetchStoriesByEpic(lastProjectId.value, lastEpicId.value)
  }

  /** Re-execute the last fetch call */
  async function retry() {
    await store.fetchStoriesByEpic(lastProjectId.value, lastEpicId.value)
  }

  /** Update filters and trigger re-render via computed */
  function setFilters(newFilters: Partial<StoryFilters>) {
    store.setFilters(newFilters)
  }

  /** Select a story by ID */
  function selectStory(storyId: string | null) {
    store.setSelectedStory(storyId)
  }

  /** Watch for filter changes that require re-fetch (status) */
  watch(
    () => store.filters.status,
    () => {
      fetchStories()
    },
  )

  onMounted(() => {
    fetchStories()
  })

  return {
    stories: computed(() => store.filteredStories),
    allStories: computed(() => store.items),
    selectedStory: computed(() => store.selectedStory),
    selectedStoryId: computed(() => store.selectedStoryId),
    filters: computed(() => store.filters),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchStories,
    retry,
    setFilters,
    selectStory,
  }
}
