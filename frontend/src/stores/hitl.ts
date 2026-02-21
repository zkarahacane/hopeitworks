import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

export interface HITLPendingItem {
  hitlRequestId: string
  runId: string
  stepId: string
  projectId: string
  projectName: string
  storyKey: string
  storyTitle: string
  prUrl: string | null
  pendingSince: string
}

export const useHITLStore = defineStore('hitl', () => {
  const pendingItems = ref<HITLPendingItem[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const pendingCount = computed(() => pendingItems.value.length)

  /** Fetch all pending HITL requests from the API */
  async function fetchPending() {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/hitl-requests', {
        params: { query: { status: 'pending' } },
      })

      if (apiError) {
        error.value = 'Failed to load pending approvals'
        return
      }

      if (data) {
        pendingItems.value = data.data.map((item) => ({
          hitlRequestId: item.id,
          runId: item.run_id ?? '',
          stepId: item.step_id,
          projectId: item.project_id ?? '',
          projectName: '',
          storyKey: item.story_key,
          storyTitle: item.story_title,
          prUrl: null,
          pendingSince: item.created_at,
        }))
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load pending approvals'
    } finally {
      isLoading.value = false
    }
  }

  /** Handle SSE hitl.pending event — adds with dedup by hitlRequestId */
  function handlePendingEvent(payload: {
    hitl_request_id: string
    run_id: string
    step_id: string
    project_id: string
    story_key: string
    pr_url?: string
    pending_since?: string
  }) {
    const exists = pendingItems.value.some(
      (i) => i.hitlRequestId === payload.hitl_request_id,
    )
    if (!exists) {
      pendingItems.value.push({
        hitlRequestId: payload.hitl_request_id,
        runId: payload.run_id,
        stepId: payload.step_id,
        projectId: payload.project_id,
        projectName: '',
        storyKey: payload.story_key,
        storyTitle: '',
        prUrl: payload.pr_url ?? null,
        pendingSince: payload.pending_since ?? new Date().toISOString(),
      })
    }
  }

  /** Handle SSE hitl.approved or hitl.rejected event — removes by hitlRequestId */
  function handleResolvedEvent(hitlRequestId: string) {
    pendingItems.value = pendingItems.value.filter(
      (i) => i.hitlRequestId !== hitlRequestId,
    )
  }

  return {
    pendingItems,
    pendingCount,
    isLoading,
    error,
    fetchPending,
    handlePendingEvent,
    handleResolvedEvent,
  }
})
