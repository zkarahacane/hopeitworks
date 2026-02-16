import { computed, ref } from 'vue'
import { useProjectsStore, type FetchProjectsParams, type UpdateProjectPayload } from '@/stores/projects'
import { useAsyncAction } from '@/composables/useAsyncAction'

/**
 * Composable for project operations.
 * Wraps the projects store with reactive computed properties, retry logic,
 * and async action wrappers for single-project operations.
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

  const getProject = useAsyncAction((id: string) => store.getProject(id))

  const updateProject = useAsyncAction(
    (id: string, payload: UpdateProjectPayload) => store.updateProject(id, payload),
  )

  return {
    projects: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    currentProject: computed(() => store.currentProject),
    fetchProjects,
    retry,
    getProject,
    updateProject,
    clearCurrentProject: store.clearCurrentProject,
  }
}
