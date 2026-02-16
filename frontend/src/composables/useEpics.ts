import { computed, ref } from 'vue'
import { useEpicsStore, type FetchEpicsParams } from '@/stores/epics'

/**
 * Composable for epic list operations within a project.
 * Wraps the epics store with reactive computed properties and retry logic.
 */
export function useEpics() {
  const store = useEpicsStore()
  const lastProjectId = ref<string>('')
  const lastParams = ref<FetchEpicsParams>({})

  /** Fetch epics for a project, storing params for retry */
  async function fetchEpics(projectId: string, params: FetchEpicsParams = {}) {
    lastProjectId.value = projectId
    lastParams.value = params
    await store.fetchEpics(projectId, params)
  }

  /** Re-execute the last fetchEpics call with same params */
  async function retry() {
    if (lastProjectId.value) {
      await store.fetchEpics(lastProjectId.value, lastParams.value)
    }
  }

  return {
    epics: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchEpics,
    retry,
  }
}
