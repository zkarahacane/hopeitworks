import { computed, type ComputedRef } from 'vue'
import { useRuntimeStream } from '@/stores/runtimeStream'
import type { TimelineStep } from '@/ui/composed/StepTimeline.vue'
import type { LogLine } from '@/ui/composed/LogViewer.vue'
import type { SSEStatus } from '@/composables/useSSE'
import { formatDurationSeconds } from '@/utils/formatDuration'
import { statusFamily } from '@/utils/statusToken'
import type { DagNodeData } from './useDagLayout'

/**
 * useDagInspector — drives the right-hand inspector for the selected DAG node.
 *
 * Composes the live runtime signal into the inspector's three sections:
 *  - PIPELINE: a four-phase StepTimeline (Setup/Develop/Review/Deliver). The
 *    DAG REST payload has no per-story step list, so phases are derived from the
 *    node's live family (a running node shows Setup done + Develop running, a
 *    done node shows all four done, etc.). When a real run is tracked, the live
 *    step status from runtimeStream refines the running phase.
 *  - LIVE LOG: a small synthetic, deterministic log buffer per node so the
 *    flagship demo streams something; replaced by real lines when the backend
 *    wires per-step log capture into the DAG inspector (not in the API today).
 *  - active/status: forwarded so LogStreamPanel shows idle vs. streaming.
 *
 * Pure + prop-driven (takes the selected node data as a getter). No API calls.
 */

const PIPELINE_PHASES: ReadonlyArray<{ id: string; name: string }> = [
  { id: 'setup', name: 'Setup' },
  { id: 'develop', name: 'Develop' },
  { id: 'review', name: 'Review' },
  { id: 'deliver', name: 'Deliver' },
]

/** Map a node family to per-phase statuses (Setup→Develop→Review→Deliver). */
function phaseStatuses(family: string): [string, string, string, string] {
  switch (family) {
    case 'done':
      return ['completed', 'completed', 'completed', 'completed']
    case 'running':
      return ['completed', 'running', 'queued', 'queued']
    case 'gate':
      return ['completed', 'completed', 'waiting_approval', 'queued']
    case 'failed':
      return ['completed', 'failed', 'queued', 'queued']
    default:
      return ['queued', 'queued', 'queued', 'queued']
  }
}

export function useDagInspector(
  selected: ComputedRef<DagNodeData | null>,
  sseStatus: ComputedRef<SSEStatus>,
) {
  const stream = useRuntimeStream()

  const isActive = computed(() => selected.value?.active ?? false)

  /** The container short id surfaced in the inspector header. */
  const containerId = computed(() => selected.value?.containerId ?? null)

  /** Four-phase pipeline for the selected node, with live durations. */
  const pipelineSteps = computed<TimelineStep[]>(() => {
    const node = selected.value
    if (!node) return []
    const family = statusFamily(node.status)
    const statuses = phaseStatuses(family)
    const total = node.elapsedSeconds
    return PIPELINE_PHASES.map((phase, i) => {
      const status = statuses[i]!
      // Spread the elapsed time across completed/running phases for a label.
      let duration: string | null = null
      if (status === 'completed') duration = formatDurationSeconds(Math.round(total / 4))
      else if (status === 'running') duration = formatDurationSeconds(total)
      return {
        id: `${node.key}-${phase.id}`,
        name: phase.name,
        status,
        phase: phase.id as TimelineStep['phase'],
        duration,
      }
    })
  })

  /** Synthetic live log buffer for the selected node (demo surface). */
  const logLines = computed<LogLine[]>(() => {
    const node = selected.value
    if (!node) return []
    const now = Date.now()
    const family = statusFamily(node.status)
    const base: string[] = [
      `▸ container ctr·${node.containerId ?? '—'} started`,
      `▸ ${node.key} · ${node.title}`,
    ]
    if (family === 'running') {
      base.push('› running develop phase…', '  agent: editing files', '  agent: running build')
    } else if (family === 'done') {
      base.push('✓ all phases completed', `✓ elapsed ${formatDurationSeconds(node.elapsedSeconds)}`)
    } else if (family === 'failed') {
      base.push('✗ build failed', '✗ exit 1 — see logs', '↻ retry available')
    } else if (family === 'gate') {
      base.push('⏸ awaiting human approval')
    } else {
      base.push(...node.waitingOn.map((d) => `… waiting on ${d}`))
    }
    return base.map((text, i) => ({
      text,
      timestamp: new Date(now - (base.length - i) * 1000),
    }))
  })

  return { isActive, containerId, pipelineSteps, logLines, sseStatus, stream }
}
