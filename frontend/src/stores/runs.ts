import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

export const useRunsStore = defineStore('runs', () => {
  const items = ref<Array<{ id: string; status: string }>>([])
  const current = ref<{ id: string; status: string; steps: Array<unknown> } | null>(null)
  const isLoading = ref(false)
  const isPausing = ref(false)
  const isResuming = ref(false)

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

  return { items, current, isLoading, isPausing, isResuming, pauseRun, resumeRun, updateRunStatus }
})
