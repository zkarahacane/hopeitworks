import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

export type HaltReason = components['schemas']['HaltReason']

export interface ProbeHaltItem {
  id: string
  runStepId: string
  runId: string
  projectId: string
  storyKey: string
  storyTitle: string
  stepName: string
  stageName?: string
  haltReason?: HaltReason
  pendingSince: string
}

export const useProbeHaltsStore = defineStore('probeHalts', () => {
  const items = ref<ProbeHaltItem[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const count = computed(() => items.value.length)

  /** Group items by halt_reason.probe key */
  const byReason = computed<Record<string, ProbeHaltItem[]>>(() => {
    const groups: Record<string, ProbeHaltItem[]> = {}
    for (const item of items.value) {
      const key = item.haltReason?.probe ?? 'unknown'
      if (!groups[key]) groups[key] = []
      groups[key]!.push(item)
    }
    return groups
  })

  /** Fetch all pending probe-halt gates from the API */
  async function fetchPending(projectId?: string) {
    isLoading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET('/probe-halts', {
        params: {
          query: projectId ? { project_id: projectId } : {},
        },
      })

      if (apiError) {
        error.value = 'Failed to load probe halts'
        return
      }

      if (data) {
        items.value = data.data.map((item) => ({
          id: item.id,
          runStepId: item.run_step_id,
          runId: item.run_id,
          projectId: item.project_id,
          storyKey: item.story_key,
          storyTitle: item.story_title,
          stepName: item.step_name,
          stageName: item.stage_name,
          haltReason: item.halt_reason ?? undefined,
          pendingSince: item.created_at,
        }))
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load probe halts'
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Handle SSE hitl_gate.pending event.
   * Only processes events where gate_type === 'probe_halt'.
   * Deduplicates by hitl_request_id.
   */
  function handlePendingEvent(payload: {
    gate_type?: string
    hitl_request_id: string
    run_id?: string
    step_id?: string
    probe?: string
    observed?: number
    threshold?: number
    unit?: string
    story_key?: string
  }) {
    if (payload.gate_type !== 'probe_halt') return

    const exists = items.value.some((i) => i.id === payload.hitl_request_id)
    if (!exists) {
      items.value.push({
        id: payload.hitl_request_id,
        runStepId: payload.step_id ?? '',
        runId: payload.run_id ?? '',
        projectId: '',
        storyKey: payload.story_key ?? '',
        storyTitle: '',
        stepName: '',
        haltReason:
          payload.probe != null
            ? {
                probe: payload.probe as 'log_silence' | 'wallclock' | 'cost_batch',
                observed: payload.observed,
                threshold: payload.threshold,
                unit: payload.unit,
              }
            : undefined,
        pendingSince: new Date().toISOString(),
      })
    }
  }

  /**
   * Handle SSE hitl_gate.approved / hitl_gate.rejected / hitl_gate.resolved.
   * Removes the item from the pending list.
   */
  function handleResolvedEvent(id: string) {
    items.value = items.value.filter((i) => i.id !== id)
  }

  return {
    items,
    isLoading,
    error,
    count,
    byReason,
    fetchPending,
    handlePendingEvent,
    handleResolvedEvent,
  }
})
