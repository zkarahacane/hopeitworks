import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import { useHITLStore } from './hitl'

export const useRunsStore = defineStore('runs', () => {
  const items = ref<Array<{ id: string; status: string }>>([])
  const current = ref<{ id: string; status: string; steps: Array<unknown> } | null>(null)
  const isLoading = ref(false)
  const isPausing = ref(false)
  const isResuming = ref(false)
  const isRetrying = ref(false)

  /**
   * Whether the circuit breaker is currently active for the viewed project.
   * Updated reactively via SSE events.
   */
  const circuitBreakerActive = ref(false)

  /** Pause a running run. */
  async function pauseRun(projectId: string, runId: string) {
    isPausing.value = true
    try {
      const { data, error } = await apiClient.POST(
        '/projects/{projectId}/runs/{runId}/pause',
        { params: { path: { projectId, runId } } },
      )
      if (error) throw error
      if (data) {
        updateRunStatus(runId, data.status)
      }
      return data
    } finally {
      isPausing.value = false
    }
  }

  /** Resume a paused run. */
  async function resumeRun(projectId: string, runId: string) {
    isResuming.value = true
    try {
      const { data, error } = await apiClient.POST(
        '/projects/{projectId}/runs/{runId}/resume',
        { params: { path: { projectId, runId } } },
      )
      if (error) throw error
      if (data) {
        updateRunStatus(runId, data.status)
      }
      return data
    } finally {
      isResuming.value = false
    }
  }

  /** Retry a failed step. */
  async function retryStep(runId: string, stepId: string) {
    isRetrying.value = true
    try {
      const { data, error } = await apiClient.POST(
        '/runs/{runId}/steps/{stepId}/retry',
        { params: { path: { runId, stepId } } },
      )
      if (error) throw error
      if (data) {
        updateRunStatus(runId, data.status)
      }
      return data
    } finally {
      isRetrying.value = false
    }
  }

  /** Update run status in local state. */
  function updateRunStatus(runId: string, status: string) {
    const item = items.value.find((r) => r.id === runId)
    if (item) {
      item.status = status
    }
    if (current.value?.id === runId) {
      current.value.status = status
    }
  }

  /** Handle SSE events dispatched from the useSSE composable */
  function handleSSEEvent(event: { type: string; payload: Record<string, unknown> }) {
    if (event.type === 'hitl_gate.pending') {
      const hitlStore = useHITLStore()
      hitlStore.handlePendingEvent(
        event.payload as {
          hitl_request_id: string
          run_id: string
          step_id: string
          project_id: string
          story_key: string
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

  return {
    items,
    current,
    isLoading,
    isPausing,
    isResuming,
    isRetrying,
    circuitBreakerActive,
    pauseRun,
    resumeRun,
    retryStep,
    updateRunStatus,
    handleSSEEvent,
  }
})
