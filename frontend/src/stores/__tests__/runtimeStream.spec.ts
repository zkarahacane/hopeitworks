import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { reduceRuntimeEvent, useRuntimeStream } from '../runtimeStream'

/** Build a fresh mutable state object for pure-reducer tests. */
function emptyState() {
  return { runs: {} as Record<string, never>, steps: {} as Record<string, never> }
}

// Use `any` for terseness in the reducer tests — the reducer's job is to read
// loose payloads safely.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type S = any

describe('reduceRuntimeEvent (pure event → state reducer)', () => {
  let state: S

  beforeEach(() => {
    state = emptyState()
  })

  it('run.started creates a running run with startedAt', () => {
    reduceRuntimeEvent(state, 'run.started', {
      run_id: 'r1',
      started_at: '2026-06-17T10:00:00Z',
    })
    expect(state.runs.r1.status).toBe('running')
    expect(state.runs.r1.startedAt).toBe('2026-06-17T10:00:00Z')
    expect(state.runs.r1.finishedAt).toBeNull()
    expect(state.runs.r1.awaitingGate).toBe(false)
  })

  it('reads fields nested under payload (full model.Event shape)', () => {
    reduceRuntimeEvent(state, 'run.started', {
      entity_type: 'run',
      action: 'started',
      payload: { run_id: 'r9', started_at: '2026-06-17T10:00:00Z' },
    })
    expect(state.runs.r9.status).toBe('running')
    expect(state.runs.r9.startedAt).toBe('2026-06-17T10:00:00Z')
  })

  it('run.completed marks terminal + clears active step', () => {
    reduceRuntimeEvent(state, 'run.started', { run_id: 'r1' })
    reduceRuntimeEvent(state, 'step.started', { run_id: 'r1', step_id: 's1' })
    expect(state.runs.r1.activeStepId).toBe('s1')

    reduceRuntimeEvent(state, 'run.completed', {
      run_id: 'r1',
      completed_at: '2026-06-17T10:05:00Z',
    })
    expect(state.runs.r1.status).toBe('completed')
    expect(state.runs.r1.finishedAt).toBe('2026-06-17T10:05:00Z')
    expect(state.runs.r1.activeStepId).toBeNull()
  })

  it('run.failed and run.cancelled set the right status', () => {
    reduceRuntimeEvent(state, 'run.failed', { run_id: 'r1' })
    expect(state.runs.r1.status).toBe('failed')
    reduceRuntimeEvent(state, 'run.cancelled', { run_id: 'r2' })
    expect(state.runs.r2.status).toBe('cancelled')
  })

  it('run.paused sets awaitingGate, run.resumed clears it', () => {
    reduceRuntimeEvent(state, 'run.started', { run_id: 'r1' })
    reduceRuntimeEvent(state, 'run.paused', { run_id: 'r1' })
    expect(state.runs.r1.status).toBe('paused')
    expect(state.runs.r1.awaitingGate).toBe(true)

    reduceRuntimeEvent(state, 'run.resumed', { run_id: 'r1' })
    expect(state.runs.r1.status).toBe('running')
    expect(state.runs.r1.awaitingGate).toBe(false)
  })

  it('step.started marks step running + run.activeStepId', () => {
    reduceRuntimeEvent(state, 'step.started', {
      run_id: 'r1',
      step_id: 's1',
      started_at: '2026-06-17T10:01:00Z',
    })
    expect(state.steps.s1.status).toBe('running')
    expect(state.steps.s1.startedAt).toBe('2026-06-17T10:01:00Z')
    expect(state.runs.r1.activeStepId).toBe('s1')
  })

  it('step.completed/failed/cancelled set status + clear active step', () => {
    reduceRuntimeEvent(state, 'step.started', { run_id: 'r1', step_id: 's1' })
    reduceRuntimeEvent(state, 'step.completed', { run_id: 'r1', step_id: 's1' })
    expect(state.steps.s1.status).toBe('completed')
    expect(state.runs.r1.activeStepId).toBeNull()

    reduceRuntimeEvent(state, 'step.started', { run_id: 'r1', step_id: 's2' })
    reduceRuntimeEvent(state, 'step.failed', { run_id: 'r1', step_id: 's2' })
    expect(state.steps.s2.status).toBe('failed')

    reduceRuntimeEvent(state, 'step.started', { run_id: 'r1', step_id: 's3' })
    reduceRuntimeEvent(state, 'step.cancelled', { run_id: 'r1', step_id: 's3' })
    expect(state.steps.s3.status).toBe('cancelled')
  })

  it('log.emitted with type=cost accumulates tokens and usd', () => {
    reduceRuntimeEvent(state, 'run.started', { run_id: 'r1' })
    reduceRuntimeEvent(state, 'log.emitted', {
      run_id: 'r1',
      type: 'cost',
      input_tokens: 100,
      output_tokens: 20,
    })
    reduceRuntimeEvent(state, 'log.emitted', {
      run_id: 'r1',
      type: 'cost',
      input_tokens: 50,
      output_tokens: 10,
      cost_usd: 0.0042,
    })
    expect(state.runs.r1.inputTokens).toBe(150)
    expect(state.runs.r1.outputTokens).toBe(30)
    expect(state.runs.r1.costUsd).toBeCloseTo(0.0042, 6)
  })

  it('log.emitted without type=cost is ignored for accounting', () => {
    reduceRuntimeEvent(state, 'log.emitted', {
      run_id: 'r1',
      message: 'hello',
      input_tokens: 999,
    })
    expect(state.runs.r1).toBeUndefined()
  })

  it('hitl_gate.pending sets awaitingGate on run + step; resolution clears it', () => {
    reduceRuntimeEvent(state, 'hitl_gate.pending', { run_id: 'r1', step_id: 's1' })
    expect(state.runs.r1.awaitingGate).toBe(true)
    expect(state.steps.s1.awaitingGate).toBe(true)

    reduceRuntimeEvent(state, 'hitl_gate.approved', { run_id: 'r1', step_id: 's1' })
    expect(state.runs.r1.awaitingGate).toBe(false)
    expect(state.steps.s1.awaitingGate).toBe(false)
  })

  it('legacy hitl.* aliases also drive the gate flag', () => {
    reduceRuntimeEvent(state, 'hitl.pending', { run_id: 'r1' })
    expect(state.runs.r1.awaitingGate).toBe(true)
    reduceRuntimeEvent(state, 'hitl.rejected', { run_id: 'r1' })
    expect(state.runs.r1.awaitingGate).toBe(false)
  })

  it('story.status_updated with terminal status reflects onto the run', () => {
    reduceRuntimeEvent(state, 'run.started', { run_id: 'r1' })
    reduceRuntimeEvent(state, 'story.status_updated', { run_id: 'r1', status: 'failed' })
    expect(state.runs.r1.status).toBe('failed')
    expect(state.runs.r1.finishedAt).not.toBeNull()
  })

  it('ignores malformed / empty payloads', () => {
    reduceRuntimeEvent(state, 'run.started', null)
    reduceRuntimeEvent(state, 'run.started', 'nope')
    reduceRuntimeEvent(state, 'run.started', {}) // no run_id
    reduceRuntimeEvent(state, 'totally.unknown', { run_id: 'r1' })
    expect(Object.keys(state.runs)).toHaveLength(0)
  })
})

describe('useRuntimeStream store getters', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('ingest forwards to the reducer and exposes signals', () => {
    const s = useRuntimeStream()
    s.ingest('run.started', { run_id: 'r1', started_at: '2026-06-17T10:00:00Z' })
    expect(s.runSignal('r1')?.status).toBe('running')
    expect(s.runSignal('missing')).toBeNull()
  })

  it('runElapsedSeconds uses the ticking clock for live runs', () => {
    const s = useRuntimeStream()
    s.ingest('run.started', { run_id: 'r1', started_at: '2026-06-17T10:00:00Z' })
    s.tick(new Date('2026-06-17T10:00:42Z').getTime())
    expect(s.runElapsedSeconds('r1')).toBe(42)
  })

  it('runElapsedSeconds freezes at finishedAt once terminal', () => {
    const s = useRuntimeStream()
    s.ingest('run.started', { run_id: 'r1', started_at: '2026-06-17T10:00:00Z' })
    s.ingest('run.completed', { run_id: 'r1', completed_at: '2026-06-17T10:00:30Z' })
    s.tick(new Date('2026-06-17T11:00:00Z').getTime()) // long after
    expect(s.runElapsedSeconds('r1')).toBe(30)
  })

  it('stepElapsedSeconds tracks a live step', () => {
    const s = useRuntimeStream()
    s.ingest('step.started', { run_id: 'r1', step_id: 's1', started_at: '2026-06-17T10:00:00Z' })
    s.tick(new Date('2026-06-17T10:00:10Z').getTime())
    expect(s.stepElapsedSeconds('s1')).toBe(10)
  })

  it('activeStepIds reflects only running steps', () => {
    const s = useRuntimeStream()
    s.ingest('step.started', { run_id: 'r1', step_id: 's1' })
    s.ingest('step.started', { run_id: 'r1', step_id: 's2' })
    s.ingest('step.completed', { run_id: 'r1', step_id: 's1' })
    expect([...s.activeStepIds]).toEqual(['s2'])
    expect([...s.activeStepIdsForRun('r1')]).toEqual(['s2'])
  })

  it('gatedRunIds + isRunAwaitingGate track HITL gates', () => {
    const s = useRuntimeStream()
    s.ingest('hitl_gate.pending', { run_id: 'r1' })
    expect(s.isRunAwaitingGate('r1')).toBe(true)
    expect([...s.gatedRunIds]).toEqual(['r1'])
    s.ingest('hitl_gate.approved', { run_id: 'r1' })
    expect([...s.gatedRunIds]).toEqual([])
  })

  it('runCostUsd accumulates from cost log lines', () => {
    const s = useRuntimeStream()
    s.ingest('log.emitted', { run_id: 'r1', type: 'cost', input_tokens: 1, cost_usd: 0.01 })
    s.ingest('log.emitted', { run_id: 'r1', type: 'cost', input_tokens: 1, cost_usd: 0.02 })
    expect(s.runCostUsd('r1')).toBeCloseTo(0.03, 6)
  })

  it('reset clears all tracked signals', () => {
    const s = useRuntimeStream()
    s.ingest('run.started', { run_id: 'r1' })
    s.reset()
    expect(s.runSignal('r1')).toBeNull()
    expect([...s.activeStepIds]).toEqual([])
  })
})

// ── #294 — Dashboard elapsed hydration (running runs without a `run.started`
// SSE event must still show a real elapsed, not 00:00). Fake timers anchor the
// clock so derived elapsed is deterministic. ──────────────────────────────────
describe('hydrateRunStartedAt — REST seeding for elapsed (#294)', () => {
  // A fixed "now" 3 minutes after the seeded start.
  const NOW = new Date('2026-06-17T10:03:00Z')
  const START = '2026-06-17T10:00:00Z'

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  // RG1: a run whose `run.started` SSE arrives while the dashboard is open keeps
  // ticking live; hydration of the SAME run must not disturb the live start.
  it('RG1: SSE-started run keeps a live ticking elapsed', () => {
    const s = useRuntimeStream()
    s.ingest('run.started', { run_id: 'r1', started_at: '2026-06-17T10:02:30Z' })
    // REST list later carries a (slightly different) started_at — must be ignored.
    s.hydrateRunStartedAt('r1', '2026-06-17T09:00:00Z', undefined, 'running')
    s.tick(NOW.getTime())
    expect(s.runSignal('r1')?.startedAt).toBe('2026-06-17T10:02:30Z')
    expect(s.runElapsedSeconds('r1')).toBe(30)
  })

  // RG2 (the bug): a run already running for 3min, known only via REST
  // started_at, with no SSE — elapsed must be ~180s, never 0.
  it('RG2: running run hydrated from REST shows real elapsed, not 0', () => {
    const s = useRuntimeStream()
    expect(s.runElapsedSeconds('r1')).toBe(0) // nothing known yet
    s.hydrateRunStartedAt('r1', START, undefined, 'running')
    s.tick(NOW.getTime())
    expect(s.runSignal('r1')?.status).toBe('running')
    expect(s.runElapsedSeconds('r1')).toBe(180)
  })

  // RG3: a pending run with no started_at must not be seeded — the getter keeps
  // signalling "no duration" (0 here; the view renders the placeholder).
  it('RG3: run without started_at is not seeded (no phantom 00:00)', () => {
    const s = useRuntimeStream()
    s.hydrateRunStartedAt('r1', null, undefined, 'pending')
    s.hydrateRunStartedAt('r1', undefined, undefined, 'pending')
    expect(s.runSignal('r1')).toBeNull()
    expect(s.runElapsedSeconds('r1')).toBe(0)
  })

  // RG4: dashboard elapsed (store, hydrated) == list duration (raw started_at
  // diff) within ±1s, because both derive from the same started_at.
  it('RG4: hydrated elapsed matches the list duration computed from started_at', () => {
    const s = useRuntimeStream()
    s.hydrateRunStartedAt('r1', START, undefined, 'running')
    s.tick(NOW.getTime())
    const listSecs = Math.floor((NOW.getTime() - new Date(START).getTime()) / 1000)
    expect(Math.abs(s.runElapsedSeconds('r1') - listSecs)).toBeLessThanOrEqual(1)
  })

  // RG5: a terminal run hydrated from REST freezes at completed_at - started_at
  // and does not keep running as the clock advances.
  it('RG5: terminal run hydrated from REST freezes at completed_at - started_at', () => {
    const s = useRuntimeStream()
    s.hydrateRunStartedAt('r1', START, '2026-06-17T10:00:45Z', 'completed')
    s.tick(new Date('2026-06-17T11:00:00Z').getTime()) // long after
    expect(s.runSignal('r1')?.status).toBe('completed')
    expect(s.runElapsedSeconds('r1')).toBe(45)
  })

  it('is idempotent: re-hydrating the same run does not change timing', () => {
    const s = useRuntimeStream()
    s.hydrateRunStartedAt('r1', START, undefined, 'running')
    s.hydrateRunStartedAt('r1', '2026-06-17T08:00:00Z', undefined, 'running')
    s.tick(NOW.getTime())
    expect(s.runSignal('r1')?.startedAt).toBe(START)
    expect(s.runElapsedSeconds('r1')).toBe(180)
  })

  it('ignores empty runId', () => {
    const s = useRuntimeStream()
    s.hydrateRunStartedAt('', START, undefined, 'running')
    expect(Object.keys(s.runs)).toHaveLength(0)
  })
})
