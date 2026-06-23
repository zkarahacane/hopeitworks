import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

/** Error code thrown when the run is not paused awaiting a manual start (409 Conflict). */
export const STAGE_NOT_STARTABLE_ERROR = 'STAGE_NOT_STARTABLE'

/**
 * Composable for triggering the "Go" on a card idle in a manual stage.
 * Wraps POST /projects/{id}/stories/{storyId}/stage/start, which resumes the run
 * parked awaiting a manual start so its segment runs and then auto-advances through
 * subsequent auto stages until the next manual/gate. Uses useAsyncAction for
 * consistent loading/error state management.
 */
export function useStageStarter() {
  const { data, error, isLoading, execute } = useAsyncAction(
    async (projectId: string, storyId: string) => {
      const { data, error: apiError, response } = await apiClient.POST(
        '/projects/{projectId}/stories/{storyId}/stage/start',
        {
          params: { path: { projectId, storyId } },
        },
      )

      if (apiError) {
        if (response?.status === 409) {
          throw new Error(STAGE_NOT_STARTABLE_ERROR)
        }
        throw new Error(getApiErrorMessage(apiError, 'Failed to start stage'))
      }

      return data
    },
  )

  return { data, error, isLoading, startStage: execute }
}
