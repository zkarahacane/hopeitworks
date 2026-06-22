import { ref, onBeforeUnmount } from 'vue'
import { useAuthStore } from '@/stores/auth'

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
    const auth = useAuthStore()
    if (!auth.isAuthenticated) {
      es.close()
      status.value = 'closed'
    } else {
      status.value = 'error'
    }
  }
  es.onmessage = (e) => {
    try {
      onEvent('message', JSON.parse(e.data))
    } catch {
      /* ignore malformed JSON */
    }
  }

  // Authoritative event names = `{entity_type}.{action}` from the backend.
  // The `hitl.*` and `epic_run.group.*` names are kept for backwards-compat with
  // existing consumers/tests; the `hitl_gate.*`, `run.paused/resumed`,
  // `circuit_breaker.*` and `epic_run_group.started` names are what the backend
  // actually emits. Registering both is purely additive and harmless.
  const knownEvents = [
    'run.started',
    'run.completed',
    'run.failed',
    'run.cancelled',
    'run.paused',
    'run.resumed',
    'step.started',
    'step.completed',
    'step.failed',
    'step.cancelled',
    // stage boundary events — move the card between stage columns on the board
    'stage.entered',
    'stage.exited',
    // emitted when a card parks idle at the entry of a not-yet-started manual stage
    'stage.awaiting_start',
    'log.emitted',
    // legacy hitl.* aliases (kept for existing consumers)
    'hitl.pending',
    'hitl.approved',
    'hitl.rejected',
    // authoritative hitl_gate.* names
    'hitl_gate.pending',
    'hitl_gate.approved',
    'hitl_gate.rejected',
    'story.status_updated',
    'epic_run.started',
    'epic_run.group.started',
    'epic_run_group.started',
    'epic_run.story.completed',
    'epic_run.completed',
    'epic_run.failed',
    'circuit_breaker.triggered',
    'circuit_breaker.reset',
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
