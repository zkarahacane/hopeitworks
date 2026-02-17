import { ref, onBeforeUnmount } from 'vue'

export type SSEStatus = 'connecting' | 'open' | 'closed' | 'error'

/**
 * Composable that manages an EventSource connection for server-sent events.
 * Opens a connection to the SSE endpoint filtered by projectId,
 * dispatches parsed events to the provided callback, and cleans up on unmount.
 */
export function useSSE(
  projectId: string,
  onEvent: (eventName: string, data: unknown) => void,
) {
  const status = ref<SSEStatus>('connecting')
  const es = new EventSource(`/api/v1/events/stream?project_id=${projectId}`)

  es.onopen = () => {
    status.value = 'open'
  }
  es.onerror = () => {
    status.value = 'error'
  }
  es.onmessage = (e) => {
    try {
      onEvent('message', JSON.parse(e.data))
    } catch {
      /* ignore malformed JSON */
    }
  }

  const knownEvents = [
    'run.started',
    'run.completed',
    'step.completed',
    'step.failed',
    'log.emitted',
    'hitl.pending',
  ]
  for (const name of knownEvents) {
    es.addEventListener(name, (e) => {
      try {
        onEvent(name, JSON.parse((e as MessageEvent).data))
      } catch {
        /* ignore malformed JSON */
      }
    })
  }

  function close() {
    es.close()
    status.value = 'closed'
  }

  onBeforeUnmount(close)

  return { status, close }
}
