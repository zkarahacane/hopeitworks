import { describe, it, expect } from 'vitest'
import { stepTypeMeta, costRoleForStep, costRoleLabel, COST_ROLES } from '../stepType'

describe('stepTypeMeta', () => {
  it('resolves the agent flag only for agent_run', () => {
    expect(stepTypeMeta('agent_run').isAgent).toBe(true)
    expect(stepTypeMeta('git_branch').isAgent).toBe(false)
    expect(stepTypeMeta('human').isAgent).toBe(false)
  })

  it('resolves the gate flag only for human', () => {
    expect(stepTypeMeta('human').isGate).toBe(true)
    expect(stepTypeMeta('agent_run').isGate).toBe(false)
  })

  it('maps each known type to a distinct icon', () => {
    expect(stepTypeMeta('git_branch').icon).toContain('pi-code')
    expect(stepTypeMeta('agent_run').icon).toContain('pi-microchip-ai')
    expect(stepTypeMeta('human').icon).toContain('pi-user')
    expect(stepTypeMeta('git_pr').icon).toContain('pi-github')
    expect(stepTypeMeta('ci_wait').icon).toContain('pi-clock')
    expect(stepTypeMeta('notify').icon).toContain('pi-bell')
  })

  it('falls back gracefully for unknown / empty actions', () => {
    expect(stepTypeMeta('mystery').typeLabel).toBe('mystery')
    expect(stepTypeMeta('mystery').icon).toContain('pi-circle')
    expect(stepTypeMeta(null).typeLabel).toBe('step')
    expect(stepTypeMeta(undefined).isAgent).toBe(false)
  })

  it('normalizes case/whitespace on the action key', () => {
    expect(stepTypeMeta('  AGENT_RUN ').isAgent).toBe(true)
    expect(stepTypeMeta('  AGENT_RUN ').action).toBe('agent_run')
  })
})

describe('costRoleForStep', () => {
  it('classifies review steps', () => {
    expect(costRoleForStep({ stepName: 'Code review', action: 'agent_run' })).toBe('review')
    expect(costRoleForStep({ stepName: 'Run tests', action: 'agent_run' })).toBe('review')
  })

  it('classifies merge steps from the name', () => {
    expect(costRoleForStep({ stepName: 'Merge to main', action: 'agent_run' })).toBe('merge')
  })

  it('defaults agent steps with dev-ish names to dev', () => {
    expect(costRoleForStep({ stepName: 'Implement story', action: 'agent_run' })).toBe('dev')
    expect(costRoleForStep({ stepName: 'something opaque', action: 'agent_run' })).toBe('dev')
  })

  it('buckets non-agent steps as other (no model cost)', () => {
    expect(costRoleForStep({ stepName: 'Create branch', action: 'git_branch' })).toBe('other')
    expect(costRoleForStep({ stepName: 'Notify completion', action: 'notify' })).toBe('other')
    expect(costRoleForStep({ stepName: 'Wait for CI', action: 'ci_wait' })).toBe('other')
  })

  it('routes a git_pr / merge action into the merge bucket', () => {
    expect(costRoleForStep({ stepName: 'Create PR', action: 'git_pr' })).toBe('merge')
  })
})

describe('costRoleLabel / COST_ROLES', () => {
  it('exposes the canonical role labels', () => {
    expect(costRoleLabel('dev')).toBe('Dev Agent')
    expect(costRoleLabel('review')).toBe('Review Agent')
    expect(costRoleLabel('merge')).toBe('Merge Agent')
  })

  it('orders dev, review, merge, other', () => {
    expect(COST_ROLES.map((r) => r.key)).toEqual(['dev', 'review', 'merge', 'other'])
  })
})
