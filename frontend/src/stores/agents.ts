import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { Pagination } from '@/types/pagination'

export type { Pagination }

/** Agent scope type */
export type AgentScope = 'global' | 'project'

/** Agent entity matching the OpenAPI Agent schema */
export interface Agent {
  id: string
  name: string
  model: string
  image: string
  template_content: string
  scope: AgentScope
  project_id?: string | null
  created_at: string
  updated_at: string
}

/** Parameters for fetching paginated agent lists */
export interface FetchAgentsParams {
  page?: number
  per_page?: number
}

/** Parameters for creating a new agent */
export interface CreateAgentParams {
  name: string
  model: string
  image: string
  template_content: string
  scope?: AgentScope
  provider?: 'claude' | 'opencode'
}

/** Parameters for updating an existing agent */
export interface UpdateAgentParams {
  name?: string
  model?: string
  image?: string
  template_content?: string
}

/**
 * Pinia store for agent state management.
 * Handles CRUD operations and storing the agent list with pagination.
 */
export const useAgentsStore = defineStore('agents', () => {
  const items = ref<Agent[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch agents from the API for a given project */
  async function fetchAgents(projectId: string, params: FetchAgentsParams = {}) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/agents',
        {
          params: {
            path: { projectId },
            query: {
              page: params.page,
              per_page: params.per_page,
            },
          },
        },
      )
      if (apiError) {
        error.value = 'Failed to load agents'
        return
      }
      items.value = (data?.data as Agent[]) ?? []
      pagination.value = (data?.pagination as Pagination) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load agents'
    } finally {
      isLoading.value = false
    }
  }

  /** Create a new agent */
  async function createAgent(projectId: string, params: CreateAgentParams): Promise<Agent | null> {
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.POST(
        '/projects/{projectId}/agents',
        {
          params: { path: { projectId } },
          body: {
            name: params.name,
            model: params.model,
            image: params.image,
            template_content: params.template_content,
            scope: params.scope ?? 'project',
            provider: params.provider ?? 'claude',
          },
        },
      )
      if (apiError) {
        error.value = 'Failed to create agent'
        return null
      }
      return (data as Agent) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create agent'
      return null
    }
  }

  /** Update an existing agent */
  async function updateAgent(
    projectId: string,
    agentId: string,
    params: UpdateAgentParams,
  ): Promise<Agent | null> {
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.PUT(
        '/projects/{projectId}/agents/{agentId}',
        {
          params: { path: { projectId, agentId } },
          body: params,
        },
      )
      if (apiError) {
        error.value = 'Failed to update agent'
        return null
      }
      return (data as Agent) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update agent'
      return null
    }
  }

  /** Delete an agent */
  async function deleteAgent(projectId: string, agentId: string): Promise<boolean> {
    error.value = null
    try {
      const { error: apiError } = await apiClient.DELETE(
        '/projects/{projectId}/agents/{agentId}',
        {
          params: { path: { projectId, agentId } },
        },
      )
      if (apiError) {
        error.value = 'Failed to delete agent'
        return false
      }
      items.value = items.value.filter((a) => a.id !== agentId)
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete agent'
      return false
    }
  }

  /** Clear the current error state */
  function clearError() {
    error.value = null
  }

  /** Reset store state to initial values */
  function reset() {
    items.value = []
    pagination.value = null
    error.value = null
  }

  return {
    items,
    pagination,
    isLoading,
    error,
    fetchAgents,
    createAgent,
    updateAgent,
    deleteAgent,
    clearError,
    reset,
  }
})
