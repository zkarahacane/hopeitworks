/**
 * Single source of truth for product status → visual identity.
 *
 * Kills "blue means everything": every run / step / story / epic status string
 * normalizes into exactly FIVE product families, each with a stable design
 * token, icon, label, and pulse flag. Blue is reserved for non-status
 * informational accents and never appears here.
 *
 * Hero screens and shared components MUST route status through `statusToken`
 * rather than re-deriving severities. The token names below are CSS custom
 * properties exposed by the theme preset (see `theme/index.ts`).
 */

import { statusSeverity as legacySeverity } from './runStatus'

/** The five product status families. */
export type StatusFamily = 'running' | 'done' | 'gate' | 'failed' | 'queued'

/** PrimeVue Tag/Badge severity, kept for components that still take a severity prop. */
export type TagSeverity = 'info' | 'success' | 'warn' | 'danger' | 'secondary' | 'contrast'

/** Resolved visual identity for a status. */
export interface StatusToken {
  /** Normalized product family. */
  family: StatusFamily
  /**
   * CSS custom property name (without `var(...)`) for the family's primary
   * color. Components read it via `var(<colorToken>)`. Stable across themes.
   */
  colorToken: string
  /** Background/surface tint token for soft badges. */
  surfaceToken: string
  /** PrimeIcons class for the family. */
  icon: string
  /** Human label for the family (Title Case). */
  label: string
  /** Whether the family should animate (running = pulse, gate = breathe). */
  pulse: boolean
  /** Closest PrimeVue Tag severity, for components driven by severity. */
  severity: TagSeverity
}

/** Per-family visual definition. Single place to retune the whole system. */
const FAMILY: Record<StatusFamily, Omit<StatusToken, 'family'>> = {
  running: {
    colorToken: '--status-running-color',
    surfaceToken: '--status-running-surface',
    icon: 'pi pi-spin pi-spinner',
    label: 'Running',
    pulse: true,
    severity: 'success',
  },
  done: {
    colorToken: '--status-done-color',
    surfaceToken: '--status-done-surface',
    icon: 'pi pi-check-circle',
    label: 'Done',
    pulse: false,
    severity: 'success',
  },
  gate: {
    colorToken: '--status-gate-color',
    surfaceToken: '--status-gate-surface',
    icon: 'pi pi-pause-circle',
    label: 'Awaiting',
    pulse: true,
    severity: 'warn',
  },
  failed: {
    colorToken: '--status-failed-color',
    surfaceToken: '--status-failed-surface',
    icon: 'pi pi-times-circle',
    label: 'Failed',
    pulse: false,
    severity: 'danger',
  },
  queued: {
    colorToken: '--status-queued-color',
    surfaceToken: '--status-queued-surface',
    icon: 'pi pi-clock',
    label: 'Queued',
    pulse: false,
    severity: 'secondary',
  },
}

/**
 * Maps every known raw status string (across run / step / story / epic / hitl
 * enums) to a product family. Includes both real backend enum values and the
 * kanban-derived / spec-named aliases so callers can pass any of them safely.
 *
 * Keys are matched case-insensitively after trimming (see `statusFamily`).
 */
const FAMILY_BY_STATUS: Record<string, StatusFamily> = {
  // ── running ──────────────────────────────────────────────────────────────
  running: 'running',
  started: 'running', // step.started (event-derived status)
  in_progress: 'running', // kanban-derived story column
  active: 'running',

  // ── done ─────────────────────────────────────────────────────────────────
  completed: 'done',
  done: 'done',
  succeeded: 'done',
  success: 'done',
  approved: 'done', // hitl resolution

  // ── gate (awaiting a human) ────────────────────────────────────────────────
  paused: 'gate',
  waiting_approval: 'gate', // step status
  waiting: 'gate',
  blocked: 'gate', // kanban-derived story column
  in_review: 'gate', // spec-named story status
  pending_approval: 'gate',
  pending_review: 'gate',

  // ── failed ─────────────────────────────────────────────────────────────────
  failed: 'failed',
  cancelled: 'failed',
  canceled: 'failed',
  error: 'failed',
  rejected: 'failed',
  timeout: 'failed',

  // ── queued ─────────────────────────────────────────────────────────────────
  pending: 'queued',
  backlog: 'queued',
  queued: 'queued',
  scheduled: 'queued',
  created: 'queued',
}

/**
 * Normalizes a raw status string to its product family.
 * Unknown / null / empty statuses fall back to `queued` (neutral gray).
 */
export function statusFamily(raw: string | null | undefined): StatusFamily {
  if (!raw) return 'queued'
  const key = raw.trim().toLowerCase()
  return FAMILY_BY_STATUS[key] ?? 'queued'
}

/**
 * Resolves the full visual identity for a raw status string.
 *
 * @param raw  Any run / step / story / epic / hitl status string.
 * @param opts.resolved  When true, a `gate` family (paused / waiting) is treated
 *                       as NOT awaiting a human, so it does not breathe. Use for
 *                       historical / resolved rows where the pulse would mislead.
 */
export function statusToken(
  raw: string | null | undefined,
  opts: { resolved?: boolean } = {},
): StatusToken {
  const family = statusFamily(raw)
  const def = FAMILY[family]
  const pulse = opts.resolved ? false : def.pulse
  return { family, ...def, pulse }
}

/**
 * Bridge for PrimeVue components still driven by severity props.
 * Routes through the unified family so severities stay consistent.
 */
export function statusTokenSeverity(raw: string | null | undefined): TagSeverity {
  return statusToken(raw).severity
}

/** Re-exported legacy helper so existing imports keep working during migration. */
export { legacySeverity }
