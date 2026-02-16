import { computed, ref } from 'vue'
import { useProjectsStore, type FetchProjectsParams } from '@/stores/projects'

/**
 * Composable for project list operations.
 * Wraps the projects store with reactive computed properties and retry logic.
 */
export function useProjects() {
  const store = useProjectsStore()
  const lastParams = ref<FetchProjectsParams>({})

  /** Fetch projects with given params, storing params for retry */
  async function fetchProjects(params: FetchProjectsParams = {}) {
    lastParams.value = params
    await store.fetchProjects(params)
  }

  /** Re-execute the last fetchProjects call with same params */
  async function retry() {
    await store.fetchProjects(lastParams.value)
  }

  return {
    projects: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchProjects,
    retry,
  }
}
