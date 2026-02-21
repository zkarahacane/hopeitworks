import { ref, onMounted } from 'vue'
import { apiClient } from '@/api/client'
import { useSSE } from '@/composables/useSSE'
import type { RunStep } from '@/features/runs/composables/useRunDetail'

interface RunStepUpdatedPayload {
  run_id: string
  step: RunStep
}

/**
 * Composable that fetches run steps and patches them in real time via SSE.
 * Exposes sorted steps, loading state, and error for the RunProgressTimeline.
 */
export function useRunProgress(projectId: string, runId: string) {
  const steps = ref<RunStep[]>([])
  const isLoading = ref(false)
  const error = ref<Error | null>(null)

  async function fetchRun() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/runs/{runId}' as never,
        { params: { path: { runId } } } as never,
      )
      if (apiError) throw new Error('Failed to load run steps')
      const run = data as unknown as { steps: RunStep[] }
      steps.value = [...(run.steps ?? [])].sort((a, b) => a.step_order - b.step_order)
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
    } finally {
      isLoading.value = false
    }
  }

  useSSE(projectId, (eventName, data) => {
    if (eventName !== 'run.step.updated') return
    const payload = data as RunStepUpdatedPayload
    if (payload.run_id !== runId) return
    const idx = steps.value.findIndex((s) => s.id === payload.step.id)
    if (idx === -1) return
    steps.value[idx] = { ...steps.value[idx], ...payload.step }
  })

  onMounted(fetchRun)

  return { steps, isLoading, error }
}
