import { onMounted } from 'vue'
import { useAsyncAction } from './useAsyncAction'
import { apiClient } from '@/api/client'
import type { Story } from '@/stores/stories'

/**
 * Composable for fetching a single story by ID.
 * Uses useAsyncAction to wrap the API call with loading/error state.
 */
export function useStoryDetail(projectId: string, storyId: string) {
  const {
    data: story,
    isLoading,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiError } = await apiClient.GET(
      '/projects/{projectId}/stories/{storyId}' as never,
      {
        params: { path: { projectId, storyId } },
      } as never,
    )
    if (apiError) throw new Error('Failed to load story')
    return data as unknown as Story
  })

  async function fetchStory() {
    await execute()
  }

  onMounted(fetchStory)

  return { story, isLoading, error, fetchStory, retry: fetchStory }
}
