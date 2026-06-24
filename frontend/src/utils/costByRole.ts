/**
 * COST-BY-ROLE derivation for the Run Detail hero.
 *
 * The design's right panel shows cost bars per role (Dev Agent / Review Agent /
 * Merge Agent) plus a "Total this run". The dedicated per-role aggregation
 * endpoint is a separate backend lot (#6) and is NOT available yet, so we derive
 * a best-effort breakdown here from the per-step cost records that the existing
 * `/projects/{projectId}/runs/{runId}/costs` endpoint already returns.
 *
 * Each step is mapped to a role via `costRoleForStep` and its `cost_usd` summed.
 * The run-level total is taken from the REST `total_cost` (fix #3: the real
 * rolled-up cost, never $0.00 on failed runs) rather than re-summing steps, so
 * costs the breakdown can't attribute still count toward the total.
 */

import type { RunCostDetail, StepCostBreakdown } from '@/features/runs/composables/useRunCosts'
import type { components } from '@/api/schema'
import { COST_ROLES, costRoleForStep, costRoleLabel, type CostRole } from './stepType'

type ProjectCostByRoleResponse = components['schemas']['ProjectCostByRole']

export interface RoleCost {
  /**
   * Role key. For the run-level heuristic this is a `CostRole`
   * ('dev'|'review'|'merge'|'other'); for the project-level endpoint it is the
   * real agent type ('implement'|'review'|'merge'|…|'unknown'). Both are strings.
   */
  role: string
  label: string
  costUsd: number
  tokensInput: number
  tokensOutput: number
  /** Fraction of `total` this role represents (0–1), for the bar width. */
  fraction: number
}

export interface CostByRoleResult {
  /** Per-role rows, in canonical order, omitting empty roles. */
  roles: RoleCost[]
  /** The authoritative run-level total (REST rollup, fix #3). */
  total: number
  /**
   * True when no per-step breakdown was available to attribute the total —
   * the panel should show a graceful "breakdown unavailable" note (the #6 gap).
   */
  derivedFromStepsOnly: boolean
}

/**
 * Aggregate a run's per-step cost records into per-role buckets.
 *
 * @param detail  The REST RunCostDetail (total + per-step breakdown), or null.
 */
export function costByRole(detail: RunCostDetail | null | undefined): CostByRoleResult {
  const total = detail?.total_cost ?? 0
  const steps = (detail?.steps ?? []) as StepCostBreakdown[]

  // Accumulate per role.
  const acc = new Map<CostRole, { costUsd: number; tokensInput: number; tokensOutput: number }>()
  for (const step of steps) {
    const role = costRoleForStep({ stepName: step.step_name, action: undefined })
    const bucket = acc.get(role) ?? { costUsd: 0, tokensInput: 0, tokensOutput: 0 }
    bucket.costUsd += step.cost_usd ?? 0
    bucket.tokensInput += step.tokens_input ?? 0
    bucket.tokensOutput += step.tokens_output ?? 0
    acc.set(role, bucket)
  }

  // Bars scale against the largest single role cost so the dominant role fills
  // the track — clearer than scaling against the (often larger) run total.
  const maxRoleCost = Math.max(0, ...Array.from(acc.values()).map((b) => b.costUsd))

  const roles: RoleCost[] = COST_ROLES.filter((def) => acc.has(def.key)).map((def) => {
    const bucket = acc.get(def.key)!
    return {
      role: def.key,
      label: costRoleLabel(def.key),
      costUsd: bucket.costUsd,
      tokensInput: bucket.tokensInput,
      tokensOutput: bucket.tokensOutput,
      fraction: maxRoleCost > 0 ? bucket.costUsd / maxRoleCost : 0,
    }
  })

  return {
    roles,
    total,
    derivedFromStepsOnly: steps.length === 0,
  }
}

/** Human label for a project-level role key (agent type), e.g. "implement" → "Implement". */
function projectRoleLabel(role: string): string {
  if (!role) return 'Unknown'
  return role.charAt(0).toUpperCase() + role.slice(1)
}

/**
 * Map the project-level COST-BY-ROLE endpoint response into the shared
 * `CostByRoleResult` so the existing `RunCostByRole` panel renders it unchanged.
 *
 * Unlike the run-level `costByRole` heuristic, this consumes the authoritative
 * server aggregation (role = agent type, "unknown" bucket included per RG4). The
 * total comes straight from the server roll-up. Bars scale against the largest
 * single role so the dominant role fills the track.
 */
export function projectCostByRole(
  data: ProjectCostByRoleResponse | null | undefined,
): CostByRoleResult {
  const total = data?.total_cost ?? 0
  const rows = data?.roles ?? []

  const maxRoleCost = Math.max(0, ...rows.map((r) => r.cost_usd ?? 0))

  const roles: RoleCost[] = rows.map((r) => {
    const costUsd = r.cost_usd ?? 0
    return {
      role: r.role,
      label: projectRoleLabel(r.role),
      costUsd,
      tokensInput: r.tokens_input ?? 0,
      tokensOutput: r.tokens_output ?? 0,
      fraction: maxRoleCost > 0 ? costUsd / maxRoleCost : 0,
    }
  })

  return {
    roles,
    total,
    // The server is the source of truth, so the breakdown is authoritative even
    // when empty (RG3: a period with no cost shows zero roles + zero total).
    derivedFromStepsOnly: false,
  }
}
