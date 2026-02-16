import { computed, onMounted, ref } from 'vue'
import { useEpicsStore } from '@/stores/epics'

/**
 * Composable for epic list operations.
 * Wraps the epics store with reactive computed properties, auto-fetch, and retry logic.
 */
export function useEpics(projectId: string) {
  const store = useEpicsStore()
  const lastProjectId = ref(projectId)

  /** Fetch epics for the given project */
  async function fetchEpics() {
    await store.fetchEpics(lastProjectId.value)
  }

  /** Re-execute the last fetchEpics call */
  async function retry() {
    await store.fetchEpics(lastProjectId.value)
  }

  onMounted(() => {
    fetchEpics()
  })

  return {
    epics: computed(() => store.items),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchEpics,
    retry,
  }
}
