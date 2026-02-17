import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

/**
 * Story entity type.
 * TODO(2-2): Replace with generated type from OpenAPI schema once Stories CRUD API lands.
 */
export interface Story {
  id: string
  epic_id: string
  project_id: string
  key: string
  title: string
  status: 'backlog' | 'running' | 'done' | 'failed'
  objective?: string
  acceptance_criteria?: string
  target_files?: string[]
  depends_on?: string[]
  created_at: string
  updated_at: string
}

/** Filter state for the story list */
export interface StoryFilters {
  status: string | null
  search: string
}

/**
 * Pinia store for story state management.
 * Handles fetching, filtering, and selecting stories within an epic.
 */
export const useStoriesStore = defineStore('stories', () => {
  const items = ref<Story[]>([])
  const selectedStoryId = ref<string | null>(null)
  const filters = ref<StoryFilters>({ status: null, search: '' })
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Stories filtered by current filter state */
  const filteredStories = computed(() => {
    let result = items.value

    if (filters.value.status && filters.value.status !== 'all') {
      result = result.filter((s) => s.status === filters.value.status)
    }

    if (filters.value.search) {
      const term = filters.value.search.toLowerCase()
      result = result.filter(
        (s) => s.key.toLowerCase().includes(term) || s.title.toLowerCase().includes(term),
      )
    }

    return result
  })

  /** Currently selected story */
  const selectedStory = computed(() =>
    items.value.find((s) => s.id === selectedStoryId.value) ?? null,
  )

  /** Fetch stories for an epic from the API */
  async function fetchStoriesByEpic(projectId: string, epicId: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/stories' as '/projects/{projectId}/epics',
        {
          params: {
            path: { projectId },
            query: { epic_id: epicId } as Record<string, string>,
          },
        } as Parameters<typeof apiClient.GET>[1],
      )
      if (apiError) {
        error.value = 'Failed to load stories'
        return
      }
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const responseData = data as any
      items.value = responseData?.data ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load stories'
    } finally {
      isLoading.value = false
    }
  }

  /** Set the currently selected story */
  function setSelectedStory(storyId: string | null) {
    selectedStoryId.value = storyId
  }

  /** Update filter state */
  function setFilters(newFilters: Partial<StoryFilters>) {
    filters.value = { ...filters.value, ...newFilters }
  }

  /** Clear current error state */
  function clearError() {
    error.value = null
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    selectedStoryId.value = null
    filters.value = { status: null, search: '' }
    error.value = null
    isLoading.value = false
  }

  return {
    items,
    selectedStoryId,
    filters,
    isLoading,
    error,
    filteredStories,
    selectedStory,
    fetchStoriesByEpic,
    setSelectedStory,
    setFilters,
    clearError,
    reset,
  }
})
