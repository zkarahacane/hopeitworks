import { describe, it, expect } from 'vitest'
import {
  statusToken,
  statusFamily,
  statusTokenSeverity,
  type StatusFamily,
} from '../statusToken'

/**
 * Exhaustive enum → family coverage. Every status string the platform can
 * produce (real backend enums + kanban-derived + spec-named aliases) must map
 * to exactly one of the 5 families. This is the contract the hero screens rely
 * on, so we assert each one explicitly.
 */
const CASES: Array<{ raw: string; family: StatusFamily; source: string }> = [
  // ── run statuses ──────────────────────────────────────────────────────────
  { raw: 'pending', family: 'queued', source: 'run' },
  { raw: 'running', family: 'running', source: 'run' },
  { raw: 'paused', family: 'gate', source: 'run' },
  { raw: 'waiting_approval', family: 'gate', source: 'run/step' },
  { raw: 'completed', family: 'done', source: 'run' },
  { raw: 'failed', family: 'failed', source: 'run' },
  { raw: 'cancelled', family: 'failed', source: 'run' },

  // ── step statuses (incl. event-derived "started") ──────────────────────────
  { raw: 'started', family: 'running', source: 'step' },

  // ── story statuses (backend + kanban-derived + spec aliases) ────────────────
  { raw: 'backlog', family: 'queued', source: 'story' },
  { raw: 'done', family: 'done', source: 'story' },
  { raw: 'in_progress', family: 'running', source: 'story/kanban' },
  { raw: 'blocked', family: 'gate', source: 'story/kanban' },
  { raw: 'in_review', family: 'gate', source: 'story/spec' },

  // ── epic_run statuses ───────────────────────────────────────────────────────
  // (pending/running/completed/failed/paused already covered above)

  // ── hitl statuses ────────────────────────────────────────────────────────────
  { raw: 'approved', family: 'done', source: 'hitl' },
  { raw: 'rejected', family: 'failed', source: 'hitl' },

  // ── extra aliases the platform may surface ──────────────────────────────────
  { raw: 'active', family: 'running', source: 'alias' },
  { raw: 'succeeded', family: 'done', source: 'alias' },
  { raw: 'success', family: 'done', source: 'alias' },
  { raw: 'waiting', family: 'gate', source: 'alias' },
  { raw: 'pending_approval', family: 'gate', source: 'alias' },
  { raw: 'pending_review', family: 'gate', source: 'alias' },
  { raw: 'canceled', family: 'failed', source: 'alias' },
  { raw: 'error', family: 'failed', source: 'alias' },
  { raw: 'timeout', family: 'failed', source: 'alias' },
  { raw: 'queued', family: 'queued', source: 'alias' },
  { raw: 'scheduled', family: 'queued', source: 'alias' },
  { raw: 'created', family: 'queued', source: 'alias' },
]

describe('statusFamily', () => {
  for (const { raw, family, source } of CASES) {
    it(`maps "${raw}" (${source}) → ${family}`, () => {
      expect(statusFamily(raw)).toBe(family)
    })
  }

  it('normalizes case and whitespace', () => {
    expect(statusFamily('  RUNNING ')).toBe('running')
    expect(statusFamily('Waiting_Approval')).toBe('gate')
  })

  it('falls back to queued for null/undefined/empty', () => {
    expect(statusFamily(null)).toBe('queued')
    expect(statusFamily(undefined)).toBe('queued')
    expect(statusFamily('')).toBe('queued')
  })

  it('falls back to queued for unknown strings', () => {
    expect(statusFamily('totally_unknown')).toBe('queued')
  })
})

describe('statusToken', () => {
  it('returns running family as animated phosphor green', () => {
    const t = statusToken('running')
    expect(t.family).toBe('running')
    expect(t.pulse).toBe(true)
    expect(t.colorToken).toBe('--status-running-color')
    expect(t.surfaceToken).toBe('--status-running-surface')
    expect(t.icon).toContain('pi')
    expect(t.severity).toBe('success')
  })

  it('done is the same hue family as running but NOT animated', () => {
    const done = statusToken('completed')
    const running = statusToken('running')
    expect(done.family).toBe('done')
    expect(done.pulse).toBe(false)
    // both green-family severities, visually distinct via pulse + token
    expect(done.severity).toBe('success')
    expect(running.severity).toBe('success')
    expect(done.colorToken).not.toBe(running.colorToken)
  })

  it('gate breathes by default (awaiting human)', () => {
    const t = statusToken('paused')
    expect(t.family).toBe('gate')
    expect(t.pulse).toBe(true)
    expect(t.severity).toBe('warn')
  })

  it('gate does not pulse when resolved option is set', () => {
    const t = statusToken('paused', { resolved: true })
    expect(t.family).toBe('gate')
    expect(t.pulse).toBe(false)
  })

  it('running does not pulse when resolved option is set', () => {
    const t = statusToken('running', { resolved: true })
    expect(t.pulse).toBe(false)
  })

  it('failed is red, not animated', () => {
    const t = statusToken('failed')
    expect(t.family).toBe('failed')
    expect(t.pulse).toBe(false)
    expect(t.severity).toBe('danger')
  })

  it('queued is gray, neutral fallback', () => {
    const t = statusToken('backlog')
    expect(t.family).toBe('queued')
    expect(t.pulse).toBe(false)
    expect(t.severity).toBe('secondary')
  })

  it('every family exposes a distinct color token', () => {
    const tokens = ['running', 'completed', 'paused', 'failed', 'backlog'].map(
      (s) => statusToken(s).colorToken,
    )
    expect(new Set(tokens).size).toBe(5)
  })

  it('never resolves a status to a blue/accent token', () => {
    for (const { raw } of CASES) {
      expect(statusToken(raw).colorToken).not.toContain('accent')
    }
  })
})

describe('statusTokenSeverity', () => {
  it('bridges to PrimeVue severities consistently with statusToken', () => {
    expect(statusTokenSeverity('running')).toBe('success')
    expect(statusTokenSeverity('paused')).toBe('warn')
    expect(statusTokenSeverity('failed')).toBe('danger')
    expect(statusTokenSeverity('backlog')).toBe('secondary')
    expect(statusTokenSeverity(null)).toBe('secondary')
  })
})
