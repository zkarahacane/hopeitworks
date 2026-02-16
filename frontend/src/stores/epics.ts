import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

/** Epic entity from the generated API schema */
export type Epic = components['schemas']['Epic']

/** Story counts by status */
export type StoryCounts = components['schemas']['StoryCounts']

/** Pagination metadata from API list responses */
export type Pagination = components['schemas']['Pagination']

/**
 * Pinia store for epic state management.
 * Handles fetching and storing the epic list for a project.
 */
export const useEpicsStore = defineStore('epics', () => {
  const items = ref<Epic[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch epics for a project from the API */
  async function fetchEpics(projectId: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/projects/{id}/epics', {
        params: {
          path: { id: projectId },
        },
      })
      if (apiError) {
        error.value = 'Failed to load epics'
        return
      }
      items.value = (data?.data as Epic[]) ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load epics'
    } finally {
      isLoading.value = false
    }
  }

  /** Clear current error state */
  function clearError() {
    error.value = null
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    error.value = null
    isLoading.value = false
  }

  return { items, isLoading, error, fetchEpics, clearError, reset }
})
