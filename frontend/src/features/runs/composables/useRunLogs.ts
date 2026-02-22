import { ref } from 'vue'
import { useSSE } from '@/composables/useSSE'
import type { LogLine } from '@/ui/composed/LogViewer.vue'

/** Shape of the SSE data for all events (full model.Event from backend). */
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
 * Composable that connects to SSE and collects log lines for a specific run.
 * Filters only `log.emitted` events matching the given runId.
 *
 * The SSE data field contains the full backend Event object. The actual log
 * data is nested inside the `payload` field (a serialized LogEvent).
 */
export function useRunLogs(projectId: string, runId: string) {
  const lines = ref<LogLine[]>([])

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName !== 'log.emitted') return
    const event = data as SSEEventData
    const logPayload = event.payload
    if (!logPayload || logPayload.run_id !== runId) return
    lines.value.push({
      text: logPayload.raw_line || logPayload.message,
      timestamp: new Date(logPayload.timestamp),
    })
  })

  function clearLogs() {
    lines.value = []
  }

  return { lines, sseStatus, clearLogs }
}
