import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

/** Project entity matching the OpenAPI Project schema */
export interface Project {
  id: string
  name: string
  description?: string
  owner_id: string
  circuit_breaker_active?: boolean
  created_at: string
  updated_at: string
}

/** Pagination metadata from API list responses */
export interface Pagination {
  total: number
  page: number
  per_page: number
}

/** Payload for creating a new project */
export interface CreateProjectPayload {
  name: string
  description?: string
}

/** Parameters for fetching paginated project lists */
export interface FetchProjectsParams {
  page?: number
  per_page?: number
  sort_by?: string
}

/**
 * Pinia store for project state management.
 * Handles fetching, storing, and resetting the project list with pagination.
 */
export const useProjectsStore = defineStore('projects', () => {
  const items = ref<Project[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch projects from the API with optional pagination and sorting params */
  async function fetchProjects(params: FetchProjectsParams = {}) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/projects', {
        params: {
          query: {
            page: params.page,
            per_page: params.per_page,
            sort_by: params.sort_by,
          },
        },
      })
      if (apiError) {
        error.value = 'Failed to load projects'
        return
      }
      items.value = (data?.data as Project[]) ?? []
      pagination.value = (data?.pagination as Pagination) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load projects'
    } finally {
      isLoading.value = false
    }
  }

  /** Create a new project via the API */
  async function createProject(payload: CreateProjectPayload): Promise<Project> {
    const { data, error: apiError } = await apiClient.POST('/projects', {
      body: payload,
    })
    if (apiError) {
      const message =
        (apiError as { error?: { message?: string } })?.error?.message ??
        'Failed to create project'
      throw new Error(message)
    }
    return data as Project
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    pagination.value = null
    error.value = null
  }

  return { items, pagination, isLoading, error, fetchProjects, createProject, reset }
})
