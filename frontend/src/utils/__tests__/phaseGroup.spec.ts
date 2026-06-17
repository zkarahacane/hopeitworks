import { describe, it, expect } from 'vitest'
import { phaseForStep, phaseLabel, PHASES } from '../phaseGroup'

describe('phaseForStep', () => {
  it('classifies setup work', () => {
    expect(phaseForStep({ actionType: 'clone_repo' })).toBe('setup')
    expect(phaseForStep({ name: 'Provision container' })).toBe('setup')
  })

  it('classifies dev work (and is the default bucket)', () => {
    expect(phaseForStep({ actionType: 'agent_run' })).toBe('dev')
    expect(phaseForStep({ name: 'Implement feature' })).toBe('dev')
    expect(phaseForStep({ actionType: 'something_unknown' })).toBe('dev')
    expect(phaseForStep({})).toBe('dev')
  })

  it('classifies review work', () => {
    expect(phaseForStep({ actionType: 'hitl_gate' })).toBe('review')
    expect(phaseForStep({ name: 'Run tests' })).toBe('review')
    expect(phaseForStep({ actionType: 'lint' })).toBe('review')
  })

  it('classifies delivery work', () => {
    expect(phaseForStep({ actionType: 'open_pr' })).toBe('delivery')
    expect(phaseForStep({ name: 'Merge to main' })).toBe('delivery')
    expect(phaseForStep({ actionType: 'deploy' })).toBe('delivery')
  })

  it('prefers actionType over name when both match different phases', () => {
    // actionType "deploy" → delivery wins even though name says "review"
    expect(phaseForStep({ actionType: 'deploy', name: 'review step' })).toBe('delivery')
  })
})

describe('phaseLabel / PHASES', () => {
  it('returns labels in canonical order', () => {
    expect(PHASES.map((p) => p.key)).toEqual(['setup', 'dev', 'review', 'delivery'])
    expect(phaseLabel('setup')).toBe('Setup')
    expect(phaseLabel('delivery')).toBe('Delivery')
  })
})
