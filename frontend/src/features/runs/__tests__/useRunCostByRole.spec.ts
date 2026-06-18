import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useRunCostByRole } from '../composables/useRunCostByRole'
import type { RunCostDetail } from '../composables/useRunCosts'

describe('useRunCostByRole', () => {
  it('derives a reactive breakdown from the cost detail ref', () => {
    const detail = ref<RunCostDetail | null>(null)
    const { breakdown } = useRunCostByRole(detail)

    expect(breakdown.value.total).toBe(0)
    expect(breakdown.value.roles).toEqual([])

    detail.value = {
      run_id: 'run-1',
      total_cost: 3,
      steps: [
        { step_id: '1', step_name: 'Implement', model: 'opus', tokens_input: 0, tokens_output: 0, cost_usd: 2 },
        { step_id: '2', step_name: 'Review', model: 'sonnet', tokens_input: 0, tokens_output: 0, cost_usd: 1 },
      ],
    }

    expect(breakdown.value.total).toBe(3)
    expect(breakdown.value.roles.map((r) => r.role)).toEqual(['dev', 'review'])
  })
})
