import { ref, computed, watch, type Ref } from 'vue'
import { apiClient } from '@/api/client'
import { useApprovalActions } from '@/features/approvals/composables/useApprovalActions'
import type { components } from '@/api/schema'
import type { RunWithSteps, RunStep } from './useRunDetail'

export type HITLRequest = components['schemas']['HITLRequest']

/** The action currently in-flight, for per-button spinners on HitlGateCard. */
export type RunHitlPendingAction = 'approve' | 'request_changes' | 'reject' | null

/**
 * useRunHitl — resolves the human gate for a run and wires the three HITL
 * actions to the real backend approve/reject flow.
 *
 * Run Detail's `HitlGateCard` is emit-only; this composable supplies the data
 * (which step is at the gate, the HITL request id, story key/title, PR/diff
 * context) and the handlers. The gate step is the run step whose action is
 * `human` and whose status is awaiting (waiting_approval / paused / running),
 * or — failing that — any `human` step that still has a pending HITL request.
 *
 * We fetch the concrete HITL request id via `/hitl-requests/by-step/{stepId}`
 * (the run payload does not carry it) so Approve / Request changes / Reject can
 * POST to `/hitl-requests/{id}/approve|reject`.
 *
 * Mapping note: the backend exposes only approve + reject. "Request changes" is
 * a reject carrying a reason (so the agent can iterate) — there is no dedicated
 * endpoint. This is surfaced to the user via the reason and called out in the
 * delivery report.
 */
export function useRunHitl(
  run: Ref<RunWithSteps | null>,
  onResolved?: () => void,
) {
  const hitlRequest = ref<HITLRequest | null>(null)
  const pendingAction = ref<RunHitlPendingAction>(null)
  const actionError = ref<string | null>(null)

  const { approveAction, rejectAction } = useApprovalActions()

  /** The run step currently sitting at the human gate, if any. */
  const gateStep = computed<RunStep | null>(() => {
    const steps = run.value?.steps ?? []
    const humanSteps = steps.filter((s) => (s.action ?? '').toLowerCase() === 'human')
    if (humanSteps.length === 0) return null
    // Prefer a step that is actively awaiting (not yet completed/cancelled).
    const awaiting = humanSteps.find(
      (s) => s.status !== 'completed' && s.status !== 'cancelled' && s.status !== 'failed',
    )
    return awaiting ?? null
  })

  /** Whether this run is currently parked on a human gate. */
  const isAtGate = computed(() => {
    if (run.value?.status === 'paused') return true
    return !!gateStep.value && hitlRequest.value?.status === 'pending'
  })

  const busy = computed(() => pendingAction.value !== null)

  /** Fetch the HITL request for the gate step (carries id + story + diff). */
  async function fetchGateRequest(stepId: string | null) {
    if (!stepId) {
      hitlRequest.value = null
      return
    }
    const { data, error } = await apiClient.GET('/hitl-requests/by-step/{stepId}', {
      params: { path: { stepId } },
    })
    if (error) {
      // 404 = no gate for this step (resolved or never gated) — not an error.
      hitlRequest.value = null
      return
    }
    hitlRequest.value = (data as HITLRequest) ?? null
  }

  // Re-resolve the gate request whenever the gate step changes.
  watch(
    () => gateStep.value?.id ?? null,
    (stepId) => {
      void fetchGateRequest(stepId)
    },
    { immediate: true },
  )

  async function approve() {
    const id = hitlRequest.value?.id
    if (!id || busy.value) return
    actionError.value = null
    pendingAction.value = 'approve'
    try {
      await approveAction.execute(id)
      hitlRequest.value = { ...hitlRequest.value!, status: 'approved' }
      onResolved?.()
    } catch (err) {
      actionError.value = err instanceof Error ? err.message : 'Approve failed'
      throw err
    } finally {
      pendingAction.value = null
    }
  }

  async function reject(reason: string) {
    const id = hitlRequest.value?.id
    if (!id || busy.value) return
    actionError.value = null
    pendingAction.value = 'reject'
    try {
      await rejectAction.execute(id, reason)
      hitlRequest.value = { ...hitlRequest.value!, status: 'rejected' }
      onResolved?.()
    } catch (err) {
      actionError.value = err instanceof Error ? err.message : 'Reject failed'
      throw err
    } finally {
      pendingAction.value = null
    }
  }

  /**
   * "Request changes" → reject carrying a reason so the agent iterates. The
   * backend has no distinct endpoint; we reuse reject with a request-changes
   * reason and surface a request_changes pending state for the button spinner.
   */
  async function requestChanges(reason: string) {
    const id = hitlRequest.value?.id
    if (!id || busy.value) return
    actionError.value = null
    pendingAction.value = 'request_changes'
    try {
      await rejectAction.execute(id, reason)
      hitlRequest.value = { ...hitlRequest.value!, status: 'rejected' }
      onResolved?.()
    } catch (err) {
      actionError.value = err instanceof Error ? err.message : 'Request changes failed'
      throw err
    } finally {
      pendingAction.value = null
    }
  }

  return {
    hitlRequest,
    gateStep,
    isAtGate,
    busy,
    pendingAction,
    actionError,
    approve,
    reject,
    requestChanges,
    refreshGate: () => fetchGateRequest(gateStep.value?.id ?? null),
  }
}
