import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

/** Epic entity matching the OpenAPI Epic schema */
export type Epic = components['schemas']['Epic']

/** Story count breakdown by status */
export type StoryCounts = components['schemas']['StoryCounts']

/** Pagination metadata from API list responses */
export type Pagination = components['schemas']['Pagination']

/** Parameters for fetching paginated epic lists */
export interface FetchEpicsParams {
  page?: number
  per_page?: number
  sort_by?: string
}

/**
 * Pinia store for epic state management within a project.
 * Handles fetching, storing, and resetting the epic list with pagination.
 */
export const useEpicsStore = defineStore('epics', () => {
  const items = ref<Epic[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const projectId = ref<string | null>(null)

  /** Fetch epics from the API for a given project */
  async function fetchEpics(pid: string, params: FetchEpicsParams = {}) {
    isLoading.value = true
    error.value = null
    projectId.value = pid
    try {
      const { data, error: apiError } = await apiClient.GET('/projects/{id}/epics', {
        params: {
          path: { id: pid },
          query: {
            page: params.page,
            per_page: params.per_page,
            sort_by: params.sort_by,
          },
        },
      })
      if (apiError) {
        error.value = 'Failed to load epics'
        return
      }
      items.value = (data?.data as Epic[]) ?? []
      pagination.value = (data?.pagination as Pagination) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load epics'
    } finally {
      isLoading.value = false
    }
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    pagination.value = null
    error.value = null
    projectId.value = null
  }

  return { items, pagination, isLoading, error, projectId, fetchEpics, reset }
})
