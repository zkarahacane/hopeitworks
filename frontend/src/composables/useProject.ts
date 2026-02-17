import { ref, onMounted } from 'vue'
import { apiClient } from '@/api/client'
import type { Project } from '@/stores/projects'

/**
 * Composable for fetching a single project by ID.
 * Auto-fetches on mount and provides retry logic.
 */
export function useProject(projectId: string) {
  const project = ref<Project | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /** Fetch project data from the API */
  async function fetchProject() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/projects/{id}', {
        params: { path: { id: projectId } },
      })
      if (apiError) {
        error.value = 'Failed to load project'
        return
      }
      project.value = (data as Project) ?? null
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load project'
    } finally {
      isLoading.value = false
    }
  }

  /** Re-execute the fetch with the same project ID */
  async function retry() {
    await fetchProject()
  }

  onMounted(() => {
    fetchProject()
  })

  return {
    project,
    isLoading,
    error,
    fetchProject,
    retry,
  }
}
