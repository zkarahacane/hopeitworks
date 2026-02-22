import { onMounted } from 'vue'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { useSSE } from '@/composables/useSSE'
import { apiClient } from '@/api/client'

/** Run with steps shape matching the API RunWithSteps schema. */
export interface RunWithSteps {
  id: string
  project_id: string
  story_id: string
  status: 'pending' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled'
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
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'
  started_at?: string
  completed_at?: string
  error_message?: string
  container_id?: string
  log_tail?: string
  created_at: string
}

/** SSE event types that should trigger a run data refetch. */
const RUN_REFRESH_EVENTS = new Set([
  'run.started',
  'run.completed',
  'run.failed',
  'step.started',
  'step.completed',
  'step.failed',
  'step.retry_initiated',
])

/**
 * Composable for fetching a single run by ID with its steps.
 * Uses useAsyncAction to wrap the API call with loading/error state.
 * Subscribes to SSE events for the given project and refetches whenever
 * a run or step event matches the current run ID.
 */
export function useRunDetail(runId: string, projectId: string) {
  const {
    data: run,
    isLoading,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiError } = await apiClient.GET('/runs/{runId}', {
      params: { path: { runId } },
    })
    if (apiError) throw new Error('Failed to load run')
    return data as RunWithSteps
  })

  async function fetchRun() {
    await execute()
  }

  // Subscribe to SSE events if a projectId is available so run/step status
  // changes are reflected immediately without polling.
  if (projectId) {
    useSSE(projectId, (eventName, data) => {
      if (!RUN_REFRESH_EVENTS.has(eventName)) return
      const payload = data as { run_id?: string; step?: { run_id?: string } }
      // Events carry run_id at the top level or nested inside a step object.
      const eventRunId = payload.run_id ?? payload.step?.run_id
      if (eventRunId && eventRunId !== runId) return
      fetchRun()
    })
  }

  onMounted(fetchRun)

  return { run, isLoading, error, fetchRun, retry: fetchRun }
}
