import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

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
      const response = await apiClient.POST(
        '/projects/{id}/stories/{story_id}/runs' as never,
        {
          params: { path: { id: projectId, story_id: storyId } },
        } as never,
      )

      const res = response as { error?: { message?: string }; response?: { status: number }; data?: unknown }

      if (res.error) {
        if (res.response?.status === 409) {
          throw new Error(ALREADY_RUNNING_ERROR)
        }
        throw new Error(res.error.message ?? 'Failed to launch run')
      }

      return res.data
    },
  )

  return { data, error, isLoading, launchRun: execute }
}
