import { computed, ref } from 'vue'
import {
  usePromptTemplatesStore,
  type FetchTemplatesParams,
} from '@/stores/promptTemplates'

/**
 * Composable for prompt template list operations.
 * Wraps the promptTemplates store with reactive computed properties and retry logic.
 */
export function usePromptTemplates(projectId: string) {
  const store = usePromptTemplatesStore()
  const lastParams = ref<FetchTemplatesParams>({})

  /** Fetch templates with given params, storing params for retry */
  async function fetchTemplates(params: FetchTemplatesParams = {}) {
    lastParams.value = params
    await store.fetchTemplates(projectId, params)
  }

  /** Re-execute the last fetchTemplates call with same params */
  async function retry() {
    await store.fetchTemplates(projectId, lastParams.value)
  }

  return {
    templates: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchTemplates,
    retry,
  }
}
