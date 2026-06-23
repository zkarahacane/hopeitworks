import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

type ResolveAction = 'resume' | 'override' | 'send_back' | 'skip' | 'abort'

/**
 * Returns the suggested remedy for a given probe type.
 * Drives the "Suggested remedy" Message in HaltGateCard.
 */
export function suggestedRemedy(probe: string): {
  action: ResolveAction
  label: string
  hint: string
} {
  switch (probe) {
    case 'cost_batch':
      return {
        action: 'resume',
        label: 'Resume with more budget',
        hint: 'The step exceeded its per-batch cost cap. Resuming will retry with a fresh budget window.',
      }
    case 'log_silence':
      return {
        action: 'resume',
        label: 'Retry fresh',
        hint: 'The agent produced no output within the silence window. Resuming will restart the step from scratch.',
      }
    case 'wallclock':
      return {
        action: 'resume',
        label: 'Retry fresh',
        hint: 'The step exceeded its wall-clock time limit. Resuming will restart the step from scratch.',
      }
    default:
      return {
        action: 'resume',
        label: 'Resume',
        hint: 'Resume the halted step and let the runtime retry.',
      }
  }
}

async function resolveHalt(hitlRequestId: string, action: ResolveAction, reason?: string) {
  const { data, error } = await apiClient.POST('/hitl-requests/{hitlRequestId}/resolve', {
    params: { path: { hitlRequestId } },
    body: { action, ...(reason ? { reason } : {}) },
  })
  if (error) throw new Error(getApiErrorMessage(error, `${action} failed`))
  return data
}

export function useProbeHaltActions() {
  const resumeAction = useAsyncAction(async (hitlRequestId: string, reason?: string) =>
    resolveHalt(hitlRequestId, 'resume', reason),
  )

  const overrideAction = useAsyncAction(async (hitlRequestId: string, reason?: string) =>
    resolveHalt(hitlRequestId, 'override', reason),
  )

  const sendBackAction = useAsyncAction(async (hitlRequestId: string, reason?: string) =>
    resolveHalt(hitlRequestId, 'send_back', reason),
  )

  const skipAction = useAsyncAction(async (hitlRequestId: string, reason?: string) =>
    resolveHalt(hitlRequestId, 'skip', reason),
  )

  const abortAction = useAsyncAction(async (hitlRequestId: string, reason?: string) =>
    resolveHalt(hitlRequestId, 'abort', reason),
  )

  return { resumeAction, overrideAction, sendBackAction, skipAction, abortAction }
}
