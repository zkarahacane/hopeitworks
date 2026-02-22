import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

/** Error code thrown when a story already has a run in progress (409 Conflict). */
export const ALREADY_RUNNING_ERROR = 'ALREADY_RUNNING'

/**
 * Composable for launching a story run via the pipeline API.
 * Wraps the POST /projects/{id}/stories/{storyId}/runs endpoint
 * using useAsyncAction for consistent loading/error state management.
 */
export function useRunLauncher() {
  const { data, error, isLoading, execute } = useAsyncAction(
    async (projectId: string, storyId: string) => {
      const { data, error: apiError, response } = await apiClient.POST(
        '/projects/{projectId}/stories/{storyId}/runs',
        {
          params: { path: { projectId, storyId } },
        },
      )

      if (apiError) {
        if (response?.status === 409) {
          throw new Error(ALREADY_RUNNING_ERROR)
        }
        throw new Error(getApiErrorMessage(apiError, 'Failed to launch run'))
      }

      return data
    },
  )

  return { data, error, isLoading, launchRun: execute }
}
