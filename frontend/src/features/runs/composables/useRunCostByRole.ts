import { computed, type Ref, type ComputedRef } from 'vue'
import { costByRole, type CostByRoleResult } from '@/utils/costByRole'
import type { RunCostDetail } from './useRunCosts'

/**
 * useRunCostByRole — reactive COST-BY-ROLE breakdown for the Run Detail panel.
 *
 * Wraps the pure `costByRole` aggregation over the reactive RunCostDetail from
 * `useRunCosts`. Pure derivation, no fetching: the host already owns the cost
 * fetch. See `costByRole` for the #6-gap workaround rationale (no per-role
 * endpoint yet → derive from per-step cost records).
 */
export function useRunCostByRole(
  costDetail: Ref<RunCostDetail | null>,
): { breakdown: ComputedRef<CostByRoleResult> } {
  const breakdown = computed(() => costByRole(costDetail.value))
  return { breakdown }
}
