import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { Pagination } from '@/types/pagination'

export type { Pagination }

/** Template type matching the OpenAPI PromptTemplate.type enum */
export type PromptTemplateType = 'implement' | 'retry' | 'review' | 'merge' | 'custom'

/** Prompt template entity matching the OpenAPI PromptTemplate schema */
export interface PromptTemplate {
  id: string
  project_id: string
  name: string
  template_content: string
  type: PromptTemplateType
  created_at: string
  updated_at: string
}

/** Parameters for fetching paginated template lists */
export interface FetchTemplatesParams {
  page?: number
  per_page?: number
}

/**
 * Pinia store for prompt template state management.
 * Handles fetching and storing the template list with pagination.
 */
export const usePromptTemplatesStore = defineStore('promptTemplates', () => {
  const items = ref<PromptTemplate[]>([])
  const pagination = ref<Pagination | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch prompt templates from the API for a given project */
  async function fetchTemplates(projectId: string, params: FetchTemplatesParams = {}) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/templates',
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
        error.value = 'Failed to load templates'
        return
      }
      items.value = (data?.data as PromptTemplate[]) ?? []
      pagination.value = (data?.pagination as Pagination) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load templates'
    } finally {
      isLoading.value = false
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

  return { items, pagination, isLoading, error, fetchTemplates, clearError, reset }
})
