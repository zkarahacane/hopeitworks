import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

/** Epic run entity from the generated API schema */
export type EpicRun = components['schemas']['EpicRun']

/** Story within an epic run */
export type EpicRunStory = components['schemas']['EpicRunStory']

/**
 * Pinia store for epic run monitoring state.
 * Handles fetching, SSE event updates, and computed progress metrics.
 */
export const useEpicRunStore = defineStore('epicRun', () => {
  const epicRun = ref<EpicRun | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const completedCount = computed(
    () => epicRun.value?.stories.filter((s) => s.status === 'completed').length ?? 0,
  )
  const totalCount = computed(() => epicRun.value?.stories.length ?? 0)
  const progressPercent = computed(() =>
    totalCount.value > 0 ? Math.round((completedCount.value / totalCount.value) * 100) : 0,
  )
  const failedStories = computed(
    () => epicRun.value?.stories.filter((s) => s.status === 'failed') ?? [],
  )

  /** Fetch an epic run by project and epic run ID */
  async function fetchEpicRun(projectId: string, epicRunId: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiErr } = await apiClient.GET(
        '/projects/{projectId}/epic-runs/{epicRunId}',
        { params: { path: { projectId, epicRunId } } },
      )
      if (apiErr) {
        error.value = 'Failed to load epic run'
        return
      }
      epicRun.value = data ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load epic run'
    } finally {
      isLoading.value = false
    }
  }

  /** Handle incoming SSE events to update epic run state reactively */
  function handleSSEEvent(eventName: string, data: unknown) {
    if (!epicRun.value) return
    const payload = data as { story_id?: string; status?: string }
    if (eventName === 'epic_run.story.completed' && payload.story_id) {
      const story = epicRun.value.stories.find((s) => s.story_id === payload.story_id)
      if (story) story.status = 'completed'
    }
    if (eventName === 'epic_run.failed') epicRun.value.status = 'failed'
    if (eventName === 'epic_run.completed') epicRun.value.status = 'completed'
  }

  /** Reset store state to initial values */
  function reset() {
    epicRun.value = null
    isLoading.value = false
    error.value = null
  }

  return {
    epicRun,
    isLoading,
    error,
    completedCount,
    totalCount,
    progressPercent,
    failedStories,
    fetchEpicRun,
    handleSSEEvent,
    reset,
  }
})
