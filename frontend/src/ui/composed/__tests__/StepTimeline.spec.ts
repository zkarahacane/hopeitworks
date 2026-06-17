import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import StepTimeline from '../StepTimeline.vue'
import type { TimelineStep } from '../StepTimeline.vue'

type TimelineProps = InstanceType<typeof StepTimeline>['$props']

function mountTimeline(props: TimelineProps) {
  return mount(StepTimeline, { props, global: { plugins: [PrimeVue] } })
}

const STEPS: TimelineStep[] = [
  { id: 's1', name: 'Clone repo', status: 'completed', actionType: 'clone_repo' },
  { id: 's2', name: 'Run agent', status: 'running', actionType: 'agent_run' },
  { id: 's3', name: 'Review', status: 'waiting_approval', actionType: 'hitl_gate' },
  { id: 's4', name: 'Open PR', status: 'pending', actionType: 'open_pr' },
]

describe('StepTimeline', () => {
  it('shows an empty message with no steps', () => {
    const w = mountTimeline({ steps: [] })
    expect(w.find('[data-testid="step-timeline-empty"]').exists()).toBe(true)
  })

  it('groups steps into the four phases (one PhaseGroup per non-empty phase)', () => {
    const w = mountTimeline({ steps: STEPS })
    const groups = w.findAll('[data-testid="phase-group"]')
    // setup, dev, review, delivery — all present, in order
    expect(groups).toHaveLength(4)
    expect(groups.map((g) => g.attributes('data-phase'))).toEqual([
      'setup',
      'dev',
      'review',
      'delivery',
    ])
  })

  it('only renders phases that have steps', () => {
    const w = mountTimeline({
      steps: [{ id: 's1', name: 'Run agent', status: 'running', actionType: 'agent_run' }],
    })
    const groups = w.findAll('[data-testid="phase-group"]')
    expect(groups).toHaveLength(1)
    expect(groups[0]!.attributes('data-phase')).toBe('dev')
  })

  it('respects an explicit phase override on a step', () => {
    const w = mountTimeline({
      steps: [
        { id: 's1', name: 'Weird step', status: 'completed', actionType: 'agent_run', phase: 'delivery' },
      ],
    })
    expect(w.find('[data-testid="phase-group"]').attributes('data-phase')).toBe('delivery')
  })

  it('renders one timeline item per step', () => {
    const w = mountTimeline({ steps: STEPS })
    expect(w.findAll('[data-testid="step-timeline-item"]')).toHaveLength(4)
  })

  it('marks the selected step', () => {
    const w = mountTimeline({ steps: STEPS, selectedId: 's2' })
    const selected = w
      .findAll('[data-testid="step-timeline-item"]')
      .filter((b) => b.attributes('data-selected') === 'true')
    expect(selected).toHaveLength(1)
  })

  it('emits select with the step id on click', async () => {
    const w = mountTimeline({ steps: STEPS })
    await w.findAll('[data-testid="step-timeline-item"]')[0]!.trigger('click')
    expect(w.emitted('select')?.[0]).toEqual(['s1'])
  })

  it('shows a duration label in mono when provided', () => {
    const w = mountTimeline({
      steps: [{ id: 's1', name: 'Run agent', status: 'running', actionType: 'agent_run', duration: '01:23' }],
    })
    expect(w.text()).toContain('01:23')
  })
})
