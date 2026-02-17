import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunTimeline from '../RunTimeline.vue'
import type { components } from '@/api/schema'

type RunStep = components['schemas']['RunStep']

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

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

function mountTimeline(steps: RunStep[]) {
  wrapper = mount(RunTimeline, {
    props: { steps },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunTimeline', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders empty state when no steps provided', () => {
    mountTimeline([])
    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(true)
  })

  it('renders root steps', () => {
    mountTimeline([
      makeStep({ id: 'step-1', step_order: 1, step_name: 'dev-story' }),
      makeStep({ id: 'step-2', step_order: 2, step_name: 'code-review' }),
    ])
    const groups = wrapper.findAll('[data-testid="step-group"]')
    expect(groups).toHaveLength(2)
  })

  it('renders retry entries with correct label', () => {
    mountTimeline([
      makeStep({ id: 'root', step_order: 1, step_name: 'dev-story', status: 'failed' }),
      makeStep({
        id: 'retry-1',
        step_order: 1,
        step_name: 'dev-story',
        parent_step_id: 'root',
        retry_count: 1,
        retry_type: 'incremental',
      }),
    ])

    const retryEntries = wrapper.findAll('[data-testid="retry-entry"]')
    expect(retryEntries).toHaveLength(1)
    expect(retryEntries[0]!.text()).toContain('Retry #1 (incremental)')
  })

  it('renders "Retry #2 (full)" for a full retry', () => {
    mountTimeline([
      makeStep({ id: 'root', step_order: 1, step_name: 'dev-story', status: 'failed' }),
      makeStep({
        id: 'retry-2',
        step_order: 1,
        step_name: 'dev-story',
        parent_step_id: 'root',
        retry_count: 2,
        retry_type: 'full',
      }),
    ])

    expect(wrapper.text()).toContain('Retry #2 (full)')
  })

  it('displays root step name and status', () => {
    mountTimeline([
      makeStep({ id: 'step-1', step_order: 1, step_name: 'dev-story', status: 'running' }),
    ])
    expect(wrapper.text()).toContain('dev-story')
    expect(wrapper.text()).toContain('running')
  })
})
