import { ref, watch, type Ref, onBeforeUnmount } from 'vue'
import { useSSE, type SSEStatus } from '@/composables/useSSE'
import type { LogLine } from '@/ui/composed/LogViewer.vue'

/** Shape of the SSE data for log events (full model.Event from backend). */
interface SSEEventData {
  id: string
  project_id: string
  entity_type: string
  entity_id: string
  action: string
  payload: {
    run_id: string
    step_id?: string
    message: string
    raw_line?: string
    timestamp: string
    level?: string
    is_json?: boolean
  }
  created_at: string
}

/**
 * Composable that connects to SSE and collects log lines for a specific step.
 * Filters `log.emitted` events matching both the given runId and stepId.
 * Reactively watches stepId — when it changes, clears logs and filters for
 * the new step. Cleans up the SSE connection on unmount.
 */
export function useStepLogs(
  projectId: string,
  runId: string,
  stepId: Ref<string | null>,
) {
  const lines = ref<LogLine[]>([])
  const sseStatus = ref<SSEStatus>('connecting')

  let closeFn: (() => void) | null = null

  function connect() {
    if (closeFn) {
      closeFn()
      closeFn = null
    }

    lines.value = []

    if (!stepId.value) {
      sseStatus.value = 'closed'
      return
    }

    const currentStepId = stepId.value

    const { status, close } = useSSE(projectId, (eventName, data) => {
      if (eventName !== 'log.emitted') return
      const event = data as SSEEventData
      const logPayload = event.payload
      if (!logPayload || logPayload.run_id !== runId) return
      if (logPayload.step_id !== currentStepId) return
      lines.value.push({
        text: logPayload.raw_line || logPayload.message,
        timestamp: new Date(logPayload.timestamp),
      })
    })

    closeFn = close
    sseStatus.value = status.value

    watch(status, (val) => {
      sseStatus.value = val
    })
  }

  watch(stepId, () => {
    connect()
  }, { immediate: true })

  onBeforeUnmount(() => {
    if (closeFn) {
      closeFn()
      closeFn = null
    }
  })

  function clearLogs() {
    lines.value = []
  }

  return { lines, sseStatus, clearLogs }
}
