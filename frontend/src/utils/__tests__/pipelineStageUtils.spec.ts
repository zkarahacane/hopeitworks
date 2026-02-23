import { describe, it, expect } from 'vitest'
import { groupStepsByStage, type StageGroup } from '../pipelineStageUtils'

function makeGroup(id: string, name: string, stepCount: number): StageGroup {
  return {
    id,
    name,
    steps: Array.from({ length: stepCount }, (_, i) => ({ id: `${id}-step-${i}` })),
  }
}

function makeStep(order: number): { step_order: number; id: string; step_name: string } {
  return { step_order: order, id: `step-${order}`, step_name: `Step ${order}` }
}

describe('groupStepsByStage', () => {
  it('groups steps into multiple stages by cumulative step count', () => {
    const groups = [
      makeGroup('setup', 'Setup', 2),
      makeGroup('dev', 'Development', 2),
      makeGroup('review', 'Review', 2),
    ]
    const steps = [
      makeStep(0),
      makeStep(1),
      makeStep(2),
      makeStep(3),
      makeStep(4),
      makeStep(5),
    ]

    const result = groupStepsByStage(groups, steps)

    expect(result.size).toBe(3)
    expect(result.get('setup')).toHaveLength(2)
    expect(result.get('dev')).toHaveLength(2)
    expect(result.get('review')).toHaveLength(2)
    expect(result.get('setup')![0]!.step_order).toBe(0)
    expect(result.get('setup')![1]!.step_order).toBe(1)
    expect(result.get('dev')![0]!.step_order).toBe(2)
    expect(result.get('dev')![1]!.step_order).toBe(3)
    expect(result.get('review')![0]!.step_order).toBe(4)
    expect(result.get('review')![1]!.step_order).toBe(5)
  })

  it('returns all steps under "default" when groups is empty', () => {
    const steps = [makeStep(0), makeStep(1)]
    const result = groupStepsByStage([], steps)

    expect(result.size).toBe(1)
    expect(result.get('default')).toHaveLength(2)
  })

  it('returns all steps under "default" when groups is undefined', () => {
    const steps = [makeStep(0)]
    const result = groupStepsByStage(undefined, steps)

    expect(result.size).toBe(1)
    expect(result.get('default')).toHaveLength(1)
  })

  it('handles empty steps array', () => {
    const groups = [makeGroup('setup', 'Setup', 2)]
    const result = groupStepsByStage(groups, [])

    expect(result.size).toBe(1)
    expect(result.get('setup')).toHaveLength(0)
  })

  it('handles groups with zero steps', () => {
    const groups = [
      makeGroup('empty', 'Empty', 0),
      makeGroup('dev', 'Development', 2),
    ]
    const steps = [makeStep(0), makeStep(1)]
    const result = groupStepsByStage(groups, steps)

    expect(result.get('empty')).toHaveLength(0)
    expect(result.get('dev')).toHaveLength(2)
  })

  it('handles fewer live steps than config expects', () => {
    const groups = [
      makeGroup('setup', 'Setup', 2),
      makeGroup('dev', 'Development', 3),
    ]
    const steps = [makeStep(0), makeStep(1), makeStep(2)]
    const result = groupStepsByStage(groups, steps)

    expect(result.get('setup')).toHaveLength(2)
    expect(result.get('dev')).toHaveLength(1)
  })
})
