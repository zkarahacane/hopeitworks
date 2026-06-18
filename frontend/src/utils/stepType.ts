/**
 * Step-type identity for the Run Detail hero.
 *
 * A run step carries an `action` (the typed step kind: `git_branch`,
 * `agent_run`, `human`, `git_pr`, `ci_wait`, `notify`, …). The redesign reads as
 * a typed pipeline, so each step shows its type chip and — for agent steps — a
 * human-readable role ("Dev Agent", "Review Agent", "Merge Agent").
 *
 * This is the single source of truth for: the type chip label/icon, whether a
 * step is an agent step (gets an AgentChip), and which COST-BY-ROLE bucket a
 * step rolls up into. Pure + deterministic so it is unit-testable and shared by
 * the timeline, steps list, and the cost-by-role panel.
 */

/** Known typed step kinds (the `action` field). Unknown values still render. */
export type StepActionType =
  | 'git_branch'
  | 'agent_run'
  | 'human'
  | 'git_pr'
  | 'ci_wait'
  | 'notify'
  | (string & {})

export interface StepTypeMeta {
  /** Canonical action key (snake_case), as stored on the step. */
  action: string
  /** Mono type label, e.g. `git_branch`. */
  typeLabel: string
  /** PrimeIcons class for the step type. */
  icon: string
  /** True for steps an agent executes (`agent_run`) — gets an AgentChip. */
  isAgent: boolean
  /** True for the human gate step (`human`). */
  isGate: boolean
}

const TYPE_ICONS: Record<string, string> = {
  git_branch: 'pi pi-code',
  agent_run: 'pi pi-microchip-ai',
  human: 'pi pi-user',
  git_pr: 'pi pi-github',
  ci_wait: 'pi pi-clock',
  notify: 'pi pi-bell',
}

/** Resolve the typed-step metadata for a raw `action` string. */
export function stepTypeMeta(action: string | null | undefined): StepTypeMeta {
  const key = (action ?? '').trim().toLowerCase()
  return {
    action: key,
    typeLabel: key || 'step',
    icon: TYPE_ICONS[key] ?? 'pi pi-circle',
    isAgent: key === 'agent_run',
    isGate: key === 'human',
  }
}

/**
 * The three product roles the COST-BY-ROLE panel rolls steps into. These map to
 * the design's "Dev Agent / Review Agent / Merge Agent" bars. Non-agent steps
 * (branch/notify/ci_wait) carry no model cost and fall into `other`.
 */
export type CostRole = 'dev' | 'review' | 'merge' | 'other'

export interface CostRoleDef {
  key: CostRole
  /** Human label for the bar, e.g. "Dev Agent". */
  label: string
}

/** Ordered role definitions for the cost-by-role bars (canonical order). */
export const COST_ROLES: readonly CostRoleDef[] = [
  { key: 'dev', label: 'Dev Agent' },
  { key: 'review', label: 'Review Agent' },
  { key: 'merge', label: 'Merge Agent' },
  { key: 'other', label: 'Other' },
] as const

const ROLE_KEYWORDS: Array<{ role: CostRole; match: RegExp }> = [
  { role: 'review', match: /\b(reviews?|review|qa|verify|lint|tests?|check)\b/i },
  { role: 'merge', match: /\b(merge|pr|pull_request|deliver|deploy|publish|release|ship)\b/i },
  { role: 'dev', match: /\b(dev|implement|code|build|generate|write|fix|feature|story)\b/i },
]

function normalize(s: string): string {
  return s.replace(/[_-]+/g, ' ').toLowerCase()
}

/**
 * Classify a step into a cost role from its name (and optional action).
 *
 * Heuristic — the backend per-role aggregation endpoint (lot #6) does not yet
 * exist, so the Run Detail derives a best-effort breakdown from the per-step
 * cost records by mapping each step to a role here. When that endpoint lands,
 * this becomes the fallback only.
 *
 * Defaults to `dev` for agent-ish steps with no clearer signal, `other` for
 * non-cost steps (branch, notify, ci_wait).
 */
export function costRoleForStep(input: {
  stepName?: string | null
  action?: string | null
}): CostRole {
  const action = (input.action ?? '').trim().toLowerCase()
  // Non-agent steps don't accrue model cost; bucket as "other".
  if (action && action !== 'agent_run' && action !== 'human') {
    // git_branch / git_pr / ci_wait / notify
    if (/\b(pr|pull_request|merge)\b/i.test(normalize(action))) return 'merge'
    return 'other'
  }
  const haystack = normalize(input.stepName ?? '')
  for (const { role, match } of ROLE_KEYWORDS) {
    if (match.test(haystack)) return role
  }
  return 'dev'
}

/** Label for a cost role key. */
export function costRoleLabel(role: CostRole): string {
  return COST_ROLES.find((r) => r.key === role)?.label ?? role
}
