import { onBeforeUnmount, onMounted } from 'vue'
import { useSSE } from '@/composables/useSSE'
import { useRuntimeStream } from '@/stores/runtimeStream'

/**
 * useDagLiveStream — the host that makes the Execution Graph live.
 *
 * Wires the raw SSE feed for a project into `useRuntimeStream` and runs a 1s
 * interval calling `stream.tick()` so elapsed timers (per node) advance while
 * the page is open. This is the single place that connects the connection to
 * the store, per the Phase 0 contract:
 *   useSSE(projectId, (name, data) => stream.ingest(name, data)) + tick loop.
 *
 * Returns the live SSE `status` (for LogStreamPanel) and the store. Cleans up
 * the timer + (via useSSE's own onBeforeUnmount) the EventSource on unmount.
 *
 * Cost note: runtimeStream cost is token-based today (USD stays 0 until the
 * backend streams USD). The view uses runtimeStream values when a run is
 * tracked and demo seeds otherwise (see useDagLayout).
 */
export function useDagLiveStream(projectId: string) {
  const stream = useRuntimeStream()

  const { status, close } = useSSE(projectId, (name, data) => {
    stream.ingest(name, data)
  })

  let timer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    stream.tick()
    timer = setInterval(() => stream.tick(), 1000)
  })

  onBeforeUnmount(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  })

  return { sseStatus: status, stream, close }
}
