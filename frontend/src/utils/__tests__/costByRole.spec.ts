import { describe, it, expect } from 'vitest'
import { costByRole, projectCostByRole } from '../costByRole'
import type { RunCostDetail } from '@/features/runs/composables/useRunCosts'
import type { components } from '@/api/schema'

type ProjectCostByRole = components['schemas']['ProjectCostByRole']

function detail(steps: RunCostDetail['steps'], total: number): RunCostDetail {
  return { run_id: 'run-1', total_cost: total, steps }
}

describe('costByRole', () => {
  it('returns an empty breakdown for null detail but keeps total at 0', () => {
    const r = costByRole(null)
    expect(r.roles).toEqual([])
    expect(r.total).toBe(0)
    expect(r.derivedFromStepsOnly).toBe(true)
  })

  it('aggregates per-step cost into role buckets', () => {
    const r = costByRole(
      detail(
        [
          { step_id: '1', step_name: 'Implement story', model: 'opus', tokens_input: 100, tokens_output: 20, cost_usd: 4.0 },
          { step_id: '2', step_name: 'Code review', model: 'sonnet', tokens_input: 50, tokens_output: 10, cost_usd: 1.0 },
          { step_id: '3', step_name: 'Merge to main', model: 'opus', tokens_input: 10, tokens_output: 2, cost_usd: 0.5 },
        ],
        5.5,
      ),
    )
    const byRole = Object.fromEntries(r.roles.map((x) => [x.role, x.costUsd]))
    expect(byRole.dev).toBe(4.0)
    expect(byRole.review).toBe(1.0)
    expect(byRole.merge).toBe(0.5)
  })

  it('uses the REST total (rollup, fix #3) for the total, not the step sum', () => {
    // Total (8) is higher than the attributable step sum (5) — e.g. a failed run
    // with unattributed cost. The total must still report the real rollup.
    const r = costByRole(
      detail(
        [
          { step_id: '1', step_name: 'Implement', model: 'opus', tokens_input: 0, tokens_output: 0, cost_usd: 5 },
        ],
        8,
      ),
    )
    expect(r.total).toBe(8)
  })

  it('keeps a non-zero total even when there is no step breakdown (fix #3)', () => {
    const r = costByRole(detail([], 2.5))
    expect(r.total).toBe(2.5)
    expect(r.roles).toEqual([])
    expect(r.derivedFromStepsOnly).toBe(true)
  })

  it('scales bar fractions against the largest role (dominant role fills track)', () => {
    const r = costByRole(
      detail(
        [
          { step_id: '1', step_name: 'Implement', model: 'opus', tokens_input: 0, tokens_output: 0, cost_usd: 4 },
          { step_id: '2', step_name: 'Review', model: 'sonnet', tokens_input: 0, tokens_output: 0, cost_usd: 1 },
        ],
        5,
      ),
    )
    const dev = r.roles.find((x) => x.role === 'dev')!
    const review = r.roles.find((x) => x.role === 'review')!
    expect(dev.fraction).toBe(1)
    expect(review.fraction).toBeCloseTo(0.25, 5)
  })

  it('omits roles with no steps and preserves canonical order', () => {
    const r = costByRole(
      detail(
        [
          { step_id: '1', step_name: 'Review', model: 'sonnet', tokens_input: 0, tokens_output: 0, cost_usd: 1 },
          { step_id: '2', step_name: 'Implement', model: 'opus', tokens_input: 0, tokens_output: 0, cost_usd: 2 },
        ],
        3,
      ),
    )
    expect(r.roles.map((x) => x.role)).toEqual(['dev', 'review'])
  })
})

describe('projectCostByRole', () => {
  function payload(
    roles: ProjectCostByRole['roles'],
    total: number,
    input = 0,
    output = 0,
  ): ProjectCostByRole {
    return { total_cost: total, total_tokens_input: input, total_tokens_output: output, roles }
  }

  it('returns an empty authoritative breakdown for null data (RG3)', () => {
    const r = projectCostByRole(null)
    expect(r.roles).toEqual([])
    expect(r.total).toBe(0)
    // Authoritative (server) data, so NOT the "unavailable" run-level gap.
    expect(r.derivedFromStepsOnly).toBe(false)
  })

  it('maps server roles to bars and keeps the server roll-up total (RG1)', () => {
    const r = projectCostByRole(
      payload(
        [
          { role: 'implement', tokens_input: 500000, tokens_output: 100000, cost_usd: 12.5, runs_count: 3 },
          { role: 'review', tokens_input: 200000, tokens_output: 50000, cost_usd: 5.25, runs_count: 2 },
        ],
        17.75,
        700000,
        150000,
      ),
    )
    expect(r.total).toBe(17.75)
    expect(r.roles.map((x) => x.role)).toEqual(['implement', 'review'])
    const impl = r.roles.find((x) => x.role === 'implement')!
    const review = r.roles.find((x) => x.role === 'review')!
    expect(impl.label).toBe('Implement')
    expect(impl.costUsd).toBe(12.5)
    // Bars scale against the largest role.
    expect(impl.fraction).toBe(1)
    expect(review.fraction).toBeCloseTo(0.42, 2)
  })

  it('keeps the unknown bucket as its own bar (RG4)', () => {
    const r = projectCostByRole(
      payload(
        [
          { role: 'implement', tokens_input: 100000, tokens_output: 20000, cost_usd: 4.0, runs_count: 1 },
          { role: 'unknown', tokens_input: 30000, tokens_output: 5000, cost_usd: 1.5, runs_count: 1 },
        ],
        5.5,
        130000,
        25000,
      ),
    )
    const unknown = r.roles.find((x) => x.role === 'unknown')
    expect(unknown).toBeDefined()
    expect(unknown!.label).toBe('Unknown')
    expect(unknown!.costUsd).toBe(1.5)
    // RG4: the unknown bucket is part of the (server) total.
    expect(r.total).toBe(5.5)
  })

  it('treats an empty roles array as an authoritative empty period (RG3)', () => {
    const r = projectCostByRole(payload([], 0))
    expect(r.roles).toEqual([])
    expect(r.total).toBe(0)
    expect(r.derivedFromStepsOnly).toBe(false)
  })
})
