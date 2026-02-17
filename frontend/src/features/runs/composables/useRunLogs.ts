import { ref } from 'vue'
import { useSSE } from '@/composables/useSSE'
import type { LogLine } from '@/ui/composed/LogViewer.vue'

/**
 * Composable that connects to SSE and collects log lines for a specific run.
 * Filters only `log.emitted` events matching the given runId.
 */
export function useRunLogs(projectId: string, runId: string) {
  const lines = ref<LogLine[]>([])

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName !== 'log.emitted') return
    const payload = data as { run_id: string; line: string; timestamp: string }
    if (payload.run_id !== runId) return
    lines.value.push({
      text: payload.line,
      timestamp: new Date(payload.timestamp),
    })
  })

  function clearLogs() {
    lines.value = []
  }

  return { lines, sseStatus, clearLogs }
}
