import { onMounted } from 'vue'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

/** Run with steps shape matching the API RunWithSteps schema. */
export interface RunWithSteps {
  id: string
  project_id: string
  story_id: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  pipeline_config_snapshot?: Record<string, unknown>
  started_at?: string
  completed_at?: string
  error_message?: string
  created_at: string
  updated_at: string
  progress?: number
  steps: RunStep[]
}

export interface RunStep {
  id: string
  run_id: string
  step_name: string
  step_order: number
  action: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled' | 'waiting_approval'
  started_at?: string
  completed_at?: string
  error_message?: string
  container_id?: string
  log_tail?: string
  created_at: string
}

/**
 * Composable for fetching a single run by ID with its steps.
 * Uses useAsyncAction to wrap the API call with loading/error state.
 */
export function useRunDetail(runId: string) {
  const {
    data: run,
    isLoading,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiError } = await apiClient.GET(
      '/runs/{runId}' as never,
      {
        params: { path: { runId } },
      } as never,
    )
    if (apiError) throw new Error('Failed to load run')
    return data as unknown as RunWithSteps
  })

  async function fetchRun() {
    await execute()
  }

  onMounted(fetchRun)

  return { run, isLoading, error, fetchRun, retry: fetchRun }
}
