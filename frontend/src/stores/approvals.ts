import { ref } from 'vue'
import { defineStore } from 'pinia'

export interface PendingApproval {
  runId: string
  stepId: string
  storyKey: string
  hitlRequestId: string
  projectId: string
}

export const useApprovalsStore = defineStore('approvals', () => {
  const pendingApprovals = ref<PendingApproval[]>([])

  function addPendingApproval(approval: PendingApproval) {
    const exists = pendingApprovals.value.some(
      (a) => a.hitlRequestId === approval.hitlRequestId,
    )
    if (!exists) {
      pendingApprovals.value.push(approval)
    }
  }

  function removePendingApproval(hitlRequestId: string) {
    pendingApprovals.value = pendingApprovals.value.filter(
      (a) => a.hitlRequestId !== hitlRequestId,
    )
  }

  /** Handle SSE hitl_gate.pending event */
  function handleHITLPendingEvent(payload: {
    run_id: string
    step_id: string
    story_key: string
    hitl_request_id: string
    project_id: string
  }) {
    addPendingApproval({
      runId: payload.run_id,
      stepId: payload.step_id,
      storyKey: payload.story_key,
      hitlRequestId: payload.hitl_request_id,
      projectId: payload.project_id,
    })
  }

  return {
    pendingApprovals,
    addPendingApproval,
    removePendingApproval,
    handleHITLPendingEvent,
  }
})
