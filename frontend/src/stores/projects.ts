import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

/** Project entity matching the OpenAPI Project schema */
export interface Project {
  id: string
  name: string
  description?: string
  owner_id: string
  created_at: string
  updated_at: string
}

/** Pagination metadata from API list responses */
export interface Pagination {
  total: number
  page: number
  per_page: number
}

/** Payload for updating a project */
export interface UpdateProjectPayload {
  name?: string
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
 * Handles fetching, storing, and resetting the project list with pagination,
 * as well as single-project CRUD operations.
 */
export const useProjectsStore = defineStore('projects', () => {
  const items = ref<Project[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const currentProject = ref<Project | null>(null)

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

  /** Fetch a single project by ID */
  async function getProject(id: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/projects/{id}', {
        params: { path: { id } },
      })
      if (apiError) {
        error.value = 'Failed to load project'
        return
      }
      currentProject.value = data as Project
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load project'
    } finally {
      isLoading.value = false
    }
  }

  /** Update an existing project */
  async function updateProject(id: string, payload: UpdateProjectPayload): Promise<Project> {
    const { data, error: apiError } = await apiClient.PUT('/projects/{id}', {
      params: { path: { id } },
      body: payload,
    })
    if (apiError) {
      throw new Error('Failed to update project')
    }
    const updated = data as Project
    currentProject.value = updated
    const idx = items.value.findIndex((p) => p.id === id)
    if (idx >= 0) {
      items.value[idx] = updated
    }
    return updated
  }

  /** Clear the currently loaded project */
  function clearCurrentProject() {
    currentProject.value = null
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    pagination.value = null
    error.value = null
    currentProject.value = null
  }

  return {
    items,
    pagination,
    isLoading,
    error,
    currentProject,
    fetchProjects,
    getProject,
    updateProject,
    clearCurrentProject,
    reset,
  }
})
