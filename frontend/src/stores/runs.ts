import { ref } from 'vue'
import { defineStore } from 'pinia'
import { useApprovalsStore } from './approvals'

export const useRunsStore = defineStore('runs', () => {
  const items = ref<Array<{ id: string; status: string }>>([])
  const current = ref<{ id: string; status: string; steps: Array<unknown> } | null>(null)
  const isLoading = ref(false)

  /**
   * Whether the circuit breaker is currently active for the viewed project.
   * Updated reactively via SSE events.
   */
  const circuitBreakerActive = ref(false)

  /** Handle SSE events dispatched from the useSSE composable */
  function handleSSEEvent(event: { type: string; payload: Record<string, unknown> }) {
    if (event.type === 'hitl_gate.pending') {
      const approvalsStore = useApprovalsStore()
      approvalsStore.handleHITLPendingEvent(
        event.payload as {
          run_id: string
          step_id: string
          story_key: string
          hitl_request_id: string
          project_id: string
        },
      )
    }

    if (event.type === 'circuit_breaker.triggered') {
      circuitBreakerActive.value = true
    }

    if (event.type === 'circuit_breaker.reset') {
      circuitBreakerActive.value = false
    }
  }

  return { items, current, isLoading, circuitBreakerActive, handleSSEEvent }
})
