import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useRunTimeline } from '../composables/useRunTimeline'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

function makeStep(overrides: Partial<RunStep> & Pick<RunStep, 'id' | 'step_order'>): RunStep {
  return {
    run_id: 'run-1',
    step_name: 'dev-story',
    action: 'agent_run',
    status: 'completed',
    created_at: '2026-02-17T10:00:00Z',
    parent_step_id: null,
    retry_count: null,
    retry_type: null,
    ...overrides,
  }
}

describe('useRunTimeline', () => {
  it('returns an empty array for empty steps input', () => {
    const steps = ref<RunStep[]>([])
    const { groupedSteps } = useRunTimeline(steps)
    expect(groupedSteps.value).toEqual([])
  })

  it('makes each root step (no parent_step_id) its own group with empty retries', () => {
    const steps = ref<RunStep[]>([
      makeStep({ id: 'step-1', step_order: 1 }),
      makeStep({ id: 'step-2', step_order: 2 }),
    ])

    const { groupedSteps } = useRunTimeline(steps)
    const groups = groupedSteps.value
    expect(groups).toHaveLength(2)
    expect(groups[0]!.root.id).toBe('step-1')
    expect(groups[0]!.retries).toEqual([])
    expect(groups[1]!.root.id).toBe('step-2')
    expect(groups[1]!.retries).toEqual([])
  })

  it('groups retry steps under their parent step', () => {
    const retry1 = makeStep({
      id: 'step-1-retry-1',
      step_order: 1,
      parent_step_id: 'step-1',
      retry_count: 1,
      retry_type: 'incremental',
    })
    const retry2 = makeStep({
      id: 'step-1-retry-2',
      step_order: 1,
      parent_step_id: 'step-1',
      retry_count: 2,
      retry_type: 'full',
    })

    const steps = ref<RunStep[]>([
      makeStep({ id: 'step-1', step_order: 1, status: 'failed' }),
      retry1,
      retry2,
    ])

    const { groupedSteps } = useRunTimeline(steps)
    const groups = groupedSteps.value
    expect(groups).toHaveLength(1)
    expect(groups[0]!.root.id).toBe('step-1')
    expect(groups[0]!.retries).toHaveLength(2)
    expect(groups[0]!.retries[0]!.id).toBe('step-1-retry-1')
    expect(groups[0]!.retries[1]!.id).toBe('step-1-retry-2')
  })

  it('sorts root steps by step_order', () => {
    const steps = ref<RunStep[]>([
      makeStep({ id: 'step-3', step_order: 3 }),
      makeStep({ id: 'step-1', step_order: 1 }),
      makeStep({ id: 'step-2', step_order: 2 }),
    ])

    const { groupedSteps } = useRunTimeline(steps)
    expect(groupedSteps.value.map((g) => g.root.id)).toEqual(['step-1', 'step-2', 'step-3'])
  })

  it('sorts retry steps by retry_count within their group', () => {
    const steps = ref<RunStep[]>([
      makeStep({ id: 'step-1', step_order: 1, status: 'failed' }),
      makeStep({
        id: 'retry-2',
        step_order: 1,
        parent_step_id: 'step-1',
        retry_count: 2,
        retry_type: 'full',
      }),
      makeStep({
        id: 'retry-1',
        step_order: 1,
        parent_step_id: 'step-1',
        retry_count: 1,
        retry_type: 'incremental',
      }),
    ])

    const { groupedSteps } = useRunTimeline(steps)
    const group = groupedSteps.value[0]!
    expect(group.retries[0]!.id).toBe('retry-1')
    expect(group.retries[1]!.id).toBe('retry-2')
  })

  it('handles a mixed scenario: 2 root steps, one with 2 retries', () => {
    const steps = ref<RunStep[]>([
      makeStep({ id: 'root-a', step_order: 1, status: 'failed' }),
      makeStep({ id: 'root-b', step_order: 2, status: 'completed' }),
      makeStep({
        id: 'retry-a-1',
        step_order: 1,
        parent_step_id: 'root-a',
        retry_count: 1,
        retry_type: 'incremental',
      }),
      makeStep({
        id: 'retry-a-2',
        step_order: 1,
        parent_step_id: 'root-a',
        retry_count: 2,
        retry_type: 'full',
      }),
    ])

    const { groupedSteps } = useRunTimeline(steps)
    const groups = groupedSteps.value
    expect(groups).toHaveLength(2)
    expect(groups[0]!.root.id).toBe('root-a')
    expect(groups[0]!.retries).toHaveLength(2)
    expect(groups[1]!.root.id).toBe('root-b')
    expect(groups[1]!.retries).toHaveLength(0)
  })

  it('is reactive: updates groupedSteps when steps change', () => {
    const steps = ref<RunStep[]>([makeStep({ id: 'step-1', step_order: 1 })])
    const { groupedSteps } = useRunTimeline(steps)
    expect(groupedSteps.value).toHaveLength(1)

    steps.value = [
      makeStep({ id: 'step-1', step_order: 1 }),
      makeStep({ id: 'step-2', step_order: 2 }),
    ]
    expect(groupedSteps.value).toHaveLength(2)
  })
})
