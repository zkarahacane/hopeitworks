import { defineStore } from 'pinia'
import { reactive, computed, ref } from 'vue'

/**
 * useRuntimeStream — derived LIVE signals layered over the raw SSE stream.
 *
 * `useSSE` gives raw `(eventName, payload)` callbacks for the 19 backend event
 * types. This store reduces those into the shape hero screens actually want:
 *  - per-run live status, elapsed seconds, accumulated cost/tokens, active step
 *  - per-step live status + timing
 *  - the set of currently-active step ids (the DAG hero's "active nodes")
 *  - gate-awaiting flags (runs/steps blocked on a human)
 *
 * It does NOT open its own EventSource — a host (view/composable) wires it to
 * `useSSE` by forwarding every event to `ingest(name, payload)`. This keeps it
 * decoupled and trivially unit-testable, and does not interfere with the
 * existing `useSSE` consumers (stores) that subscribe independently.
 *
 * Cost note: the backend currently emits token counts on `log.emitted`
 * (type === "cost") with no USD field; USD is REST-only. We accumulate tokens
 * live and ALSO pick up a USD amount if a payload ever carries one
 * (cost_usd / cost / usd) — forward-compatible without breaking today.
 */

// ── Public signal shapes ──────────────────────────────────────────────────────

export interface RunSignal {
  runId: string
  status: string
  /** ISO timestamp the run started (for elapsed derivation). */
  startedAt: string | null
  /** ISO timestamp the run finished, or null while live. */
  finishedAt: string | null
  /** Accumulated USD cost seen on the stream (0 until backend emits USD). */
  costUsd: number
  /** Accumulated input tokens seen on the stream. */
  inputTokens: number
  /** Accumulated output tokens seen on the stream. */
  outputTokens: number
  /** Step id of the currently running step, if any. */
  activeStepId: string | null
  /** True when the run is blocked on a human (HITL pending / paused). */
  awaitingGate: boolean
}

export interface StepSignal {
  stepId: string
  runId: string
  status: string
  startedAt: string | null
  finishedAt: string | null
  awaitingGate: boolean
}

interface RuntimeState {
  runs: Record<string, RunSignal>
  steps: Record<string, StepSignal>
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type Payload = Record<string, unknown>

function str(p: Payload, ...keys: string[]): string | null {
  for (const k of keys) {
    const v = p[k]
    if (typeof v === 'string' && v.length > 0) return v
  }
  return null
}

function num(p: Payload, ...keys: string[]): number {
  for (const k of keys) {
    const v = p[k]
    if (typeof v === 'number' && Number.isFinite(v)) return v
  }
  return 0
}

const TERMINAL_RUN = new Set(['completed', 'failed', 'cancelled'])

function ensureRun(state: RuntimeState, runId: string): RunSignal {
  let r = state.runs[runId]
  if (!r) {
    r = {
      runId,
      status: 'pending',
      startedAt: null,
      finishedAt: null,
      costUsd: 0,
      inputTokens: 0,
      outputTokens: 0,
      activeStepId: null,
      awaitingGate: false,
    }
    state.runs[runId] = r
  }
  return r
}

function ensureStep(state: RuntimeState, stepId: string, runId: string): StepSignal {
  let s = state.steps[stepId]
  if (!s) {
    s = {
      stepId,
      runId,
      status: 'pending',
      startedAt: null,
      finishedAt: null,
      awaitingGate: false,
    }
    state.steps[stepId] = s
  }
  return s
}

// ── Pure reducer ────────────────────────────────────────────────────────────────
// Exported for unit testing. Mutates `state` in place from one SSE event.

export function reduceRuntimeEvent(
  state: RuntimeState,
  name: string,
  data: unknown,
): void {
  if (!data || typeof data !== 'object') return
  const p = data as Payload

  // Some payloads nest the real fields under `payload` (full model.Event shape).
  const inner =
    p.payload && typeof p.payload === 'object' ? (p.payload as Payload) : p

  const runId = str(inner, 'run_id', 'runId')
  const stepId = str(inner, 'step_id', 'stepId')

  switch (name) {
    case 'run.started': {
      if (!runId) return
      const r = ensureRun(state, runId)
      r.status = 'running'
      r.startedAt = str(inner, 'started_at', 'startedAt') ?? r.startedAt ?? new Date().toISOString()
      r.finishedAt = null
      r.awaitingGate = false
      return
    }
    case 'run.resumed': {
      if (!runId) return
      const r = ensureRun(state, runId)
      r.status = 'running'
      r.awaitingGate = false
      return
    }
    case 'run.paused': {
      if (!runId) return
      const r = ensureRun(state, runId)
      r.status = 'paused'
      r.awaitingGate = true
      return
    }
    case 'run.completed':
    case 'run.failed':
    case 'run.cancelled': {
      if (!runId) return
      const r = ensureRun(state, runId)
      r.status = name.split('.')[1]!
      r.finishedAt = str(inner, 'completed_at', 'finished_at') ?? new Date().toISOString()
      r.activeStepId = null
      r.awaitingGate = false
      return
    }

    case 'step.started': {
      if (!stepId) return
      const s = ensureStep(state, stepId, runId ?? '')
      s.status = 'running'
      s.startedAt = str(inner, 'started_at', 'startedAt') ?? s.startedAt ?? new Date().toISOString()
      s.finishedAt = null
      s.awaitingGate = false
      if (runId) ensureRun(state, runId).activeStepId = stepId
      return
    }
    case 'step.completed':
    case 'step.failed':
    case 'step.cancelled': {
      if (!stepId) return
      const s = ensureStep(state, stepId, runId ?? '')
      s.status = name.split('.')[1]!
      s.finishedAt = str(inner, 'completed_at', 'finished_at') ?? new Date().toISOString()
      s.awaitingGate = false
      if (runId) {
        const r = ensureRun(state, runId)
        if (r.activeStepId === stepId) r.activeStepId = null
      }
      return
    }

    case 'log.emitted': {
      // Accumulate cost/token signal. Cost lines carry type === "cost".
      const type = str(inner, 'type')
      if (type !== 'cost') return
      if (!runId) return
      const r = ensureRun(state, runId)
      r.inputTokens += num(inner, 'input_tokens', 'inputTokens')
      r.outputTokens += num(inner, 'output_tokens', 'outputTokens')
      // Forward-compatible: pick up USD if the backend ever sends it.
      r.costUsd += num(inner, 'cost_usd', 'cost', 'usd', 'amount')
      return
    }

    case 'hitl.pending':
    case 'hitl_gate.pending': {
      if (runId) ensureRun(state, runId).awaitingGate = true
      if (stepId) ensureStep(state, stepId, runId ?? '').awaitingGate = true
      return
    }
    case 'hitl.approved':
    case 'hitl.rejected':
    case 'hitl_gate.approved':
    case 'hitl_gate.rejected': {
      if (runId) ensureRun(state, runId).awaitingGate = false
      if (stepId) ensureStep(state, stepId, runId ?? '').awaitingGate = false
      return
    }

    case 'story.status_updated': {
      // Story-level only; runs/steps are tracked elsewhere. If a run id rides
      // along and reports terminal, reflect it.
      if (runId) {
        const s = str(inner, 'status')
        if (s && TERMINAL_RUN.has(s)) {
          const r = ensureRun(state, runId)
          r.status = s
          r.finishedAt = r.finishedAt ?? new Date().toISOString()
        }
      }
      return
    }

    default:
      return
  }
}

// ── Store ─────────────────────────────────────────────────────────────────────

export const useRuntimeStream = defineStore('runtimeStream', () => {
  const state = reactive<RuntimeState>({ runs: {}, steps: {} })

  /** A ticking clock (ms) used to derive live elapsed values. Advanced by tick(). */
  const clock = ref<number>(Date.now())

  /**
   * Feed one raw SSE event into the stream. Wire this from a host that owns the
   * `useSSE` connection: `useSSE(projectId, (name, data) => stream.ingest(name, data))`.
   */
  function ingest(name: string, data: unknown): void {
    reduceRuntimeEvent(state, name, data)
  }

  /** Advance the internal clock so elapsed tickers recompute (call from rAF/interval). */
  function tick(nowMs: number = Date.now()): void {
    clock.value = nowMs
  }

  /**
   * Seed a run's timing from a REST source (e.g. list endpoints carry
   * `started_at`/`completed_at`) so elapsed derivation works for runs that were
   * already running before this client opened — i.e. before any `run.started`
   * SSE event could be captured.
   *
   * Idempotent and SSE-deferential: it only fills `startedAt`/`finishedAt`/`status`
   * that are not already known from the live stream. A non-null SSE `startedAt`
   * stays authoritative and is never overwritten. Pass a null/empty `startedAt`
   * (e.g. a pending run) and nothing is seeded — the run keeps signalling "no
   * duration" so callers can render a placeholder instead of 00:00.
   */
  function hydrateRunStartedAt(
    runId: string,
    startedAt: string | null | undefined,
    completedAt?: string | null,
    status?: string | null,
  ): void {
    if (!runId || !startedAt) return
    const r = ensureRun(state, runId)
    if (!r.startedAt) r.startedAt = startedAt
    if (!r.finishedAt && completedAt) r.finishedAt = completedAt
    // Only adopt a REST status while the run has no live signal yet (still the
    // default 'pending'); never clobber a status the stream has already moved on.
    if (status && r.status === 'pending') r.status = status
  }

  /** Reset all tracked signals (e.g. on project switch). */
  function reset(): void {
    state.runs = {}
    state.steps = {}
  }

  // ── Reactive getters hero screens read ──────────────────────────────────────

  const runSignal = computed(() => (runId: string): RunSignal | null => state.runs[runId] ?? null)
  const stepSignal = computed(() => (stepId: string): StepSignal | null => state.steps[stepId] ?? null)

  /** Live elapsed seconds for a run (uses the ticking clock; 0 if not started). */
  const runElapsedSeconds = computed(() => (runId: string): number => {
    const r = state.runs[runId]
    if (!r?.startedAt) return 0
    const start = new Date(r.startedAt).getTime()
    const end = r.finishedAt ? new Date(r.finishedAt).getTime() : clock.value
    return Math.max(0, Math.floor((end - start) / 1000))
  })

  /** Live elapsed seconds for a step. */
  const stepElapsedSeconds = computed(() => (stepId: string): number => {
    const s = state.steps[stepId]
    if (!s?.startedAt) return 0
    const start = new Date(s.startedAt).getTime()
    const end = s.finishedAt ? new Date(s.finishedAt).getTime() : clock.value
    return Math.max(0, Math.floor((end - start) / 1000))
  })

  /** Accumulated USD cost for a run (0 until backend streams USD). */
  const runCostUsd = computed(() => (runId: string): number => state.runs[runId]?.costUsd ?? 0)

  /** Whether a run is currently blocked awaiting a human. */
  const isRunAwaitingGate = computed(() => (runId: string): boolean => state.runs[runId]?.awaitingGate ?? false)

  /** Set of step ids currently running across all tracked runs — DAG active nodes. */
  const activeStepIds = computed((): Set<string> => {
    const s = new Set<string>()
    for (const step of Object.values(state.steps)) {
      if (step.status === 'running') s.add(step.stepId)
    }
    return s
  })

  /** Active step ids scoped to a single run. */
  const activeStepIdsForRun = computed(() => (runId: string): Set<string> => {
    const s = new Set<string>()
    for (const step of Object.values(state.steps)) {
      if (step.runId === runId && step.status === 'running') s.add(step.stepId)
    }
    return s
  })

  /** Run ids currently awaiting a human gate. */
  const gatedRunIds = computed((): Set<string> => {
    const s = new Set<string>()
    for (const run of Object.values(state.runs)) {
      if (run.awaitingGate) s.add(run.runId)
    }
    return s
  })

  return {
    // raw state (read-only intent)
    runs: computed(() => state.runs),
    steps: computed(() => state.steps),
    clock,
    // control
    ingest,
    tick,
    reset,
    hydrateRunStartedAt,
    // getters
    runSignal,
    stepSignal,
    runElapsedSeconds,
    stepElapsedSeconds,
    runCostUsd,
    isRunAwaitingGate,
    activeStepIds,
    activeStepIdsForRun,
    gatedRunIds,
  }
})
