import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

/**
 * Composable for launching an epic run via POST /projects/{projectId}/epics/{epicId}/runs.
 * Returns the accepted response with epic_run_id on success.
 */
export function useEpicLauncher(projectId: string, epicId: string) {
  const {
    data: result,
    isLoading: isLaunching,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiErr } = await apiClient.POST(
      '/projects/{projectId}/epics/{epicId}/runs',
      { params: { path: { projectId, epicId } } },
    )
    if (apiErr) throw new Error('Failed to launch epic run')
    return data
  })

  async function launch() {
    await execute()
  }

  return { launch, isLaunching, error, result }
}
