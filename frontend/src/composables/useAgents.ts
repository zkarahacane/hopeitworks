import { computed, ref } from 'vue'
import { useAgentsStore, type FetchAgentsParams } from '@/stores/agents'

/**
 * Composable for agent list operations.
 * Wraps the agents store with reactive computed properties and retry logic.
 */
export function useAgents(projectId: string) {
  const store = useAgentsStore()
  const lastParams = ref<FetchAgentsParams>({})

  /** Fetch agents with given params, storing params for retry */
  async function fetchAgents(params: FetchAgentsParams = {}) {
    lastParams.value = params
    await store.fetchAgents(projectId, params)
  }

  /** Re-execute the last fetchAgents call with same params */
  async function retry() {
    await store.fetchAgents(projectId, lastParams.value)
  }

  return {
    agents: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchAgents,
    retry,
  }
}
