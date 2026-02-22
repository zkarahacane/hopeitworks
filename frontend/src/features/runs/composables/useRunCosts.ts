import { onMounted, watch, type Ref } from 'vue'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

export type RunCostDetail = components['schemas']['RunCostDetail']
export type StepCostBreakdown = components['schemas']['StepCostBreakdown']

/**
 * Composable for fetching cost detail for a specific run.
 * Returns total cost and per-step cost breakdown.
 * Refetches when the run status changes to completed or failed.
 */
export function useRunCosts(
  projectId: string,
  runId: string,
  runStatus?: Ref<string | undefined>,
) {
  const {
    data: costDetail,
    isLoading,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiError } = await apiClient.GET(
      '/projects/{projectId}/runs/{runId}/costs',
      {
        params: { path: { projectId, runId } },
      },
    )
    if (apiError) throw new Error('Failed to load run costs')
    return data as RunCostDetail
  })

  async function fetchCosts() {
    await execute()
  }

  // Refetch costs when run status changes (costs finalize on step completion)
  if (runStatus) {
    watch(runStatus, (newStatus, oldStatus) => {
      if (newStatus !== oldStatus) {
        fetchCosts()
      }
    })
  }

  onMounted(fetchCosts)

  return { costDetail, isLoading, error, retry: fetchCosts }
}
