import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunProgressTimeline from '../RunProgressTimeline.vue'
import type { RunStep } from '../composables/useRunDetail'

vi.mock('../composables/useStepTimer', () => ({
  useStepTimer: vi.fn(() => ({
    elapsed: { value: '5s elapsed' },
  })),
}))

vi.mock('@/utils/formatStepDuration', () => ({
  formatStepDuration: vi.fn(() => '2m 34s'),
}))

function makeStep(overrides: Partial<RunStep> & Pick<RunStep, 'id'>): RunStep {
  return {
    run_id: 'run-1',
    step_name: 'dev-story',
    step_order: 1,
    action: 'agent_run',
    status: 'pending',
    created_at: '2026-02-17T10:00:00Z',
    ...overrides,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

const defaultProps = {
  steps: [] as RunStep[],
  projectId: 'proj-1',
  runId: 'run-1',
  isLoading: false,
  error: null as Error | null,
}

function mountComponent(propsOverride: Partial<typeof defaultProps> = {}) {
  wrapper = mount(RunProgressTimeline, {
    props: { ...defaultProps, ...propsOverride },
    global: {
      plugins: [PrimeVue],
      stubs: {
        'router-link': {
          template: '<a :href="to"><slot /></a>',
          props: ['to'],
        },
      },
    },
  })
  return wrapper
}

describe('RunProgressTimeline', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders skeleton when isLoading is true', () => {
    mountComponent({ isLoading: true })
    expect(wrapper.find('[data-testid="timeline-skeleton"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="timeline"]').exists()).toBe(false)
  })

  it('renders error message when error is set', () => {
    mountComponent({ error: new Error('Something went wrong') })
    const msg = wrapper.find('[data-testid="timeline-error"]')
    expect(msg.exists()).toBe(true)
    expect(msg.text()).toContain('Something went wrong')
  })

  it('renders empty state message when steps is empty', () => {
    mountComponent({ steps: [] })
    const msg = wrapper.find('[data-testid="timeline-empty"]')
    expect(msg.exists()).toBe(true)
    expect(msg.text()).toContain('No pipeline steps found for this run')
  })

  it('renders Timeline with correct number of steps', () => {
    mountComponent({
      steps: [
        makeStep({ id: 'step-1', step_order: 1, step_name: 'dev-story' }),
        makeStep({ id: 'step-2', step_order: 2, step_name: 'code-review' }),
        makeStep({ id: 'step-3', step_order: 3, step_name: 'merge-story' }),
      ],
    })
    expect(wrapper.find('[data-testid="timeline"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="step-status-tag"]')).toHaveLength(3)
  })

  it('renders formatted duration for completed step', () => {
    mountComponent({
      steps: [
        makeStep({
          id: 'step-1',
          status: 'completed',
          started_at: '2026-02-17T10:00:00Z',
          completed_at: '2026-02-17T10:02:34Z',
        }),
      ],
    })
    const duration = wrapper.find('[data-testid="step-duration"]')
    expect(duration.exists()).toBe(true)
    expect(duration.text()).toBe('2m 34s')
  })

  it('renders live elapsed timer for running step', () => {
    mountComponent({
      steps: [
        makeStep({
          id: 'step-1',
          status: 'running',
          started_at: '2026-02-17T10:00:00Z',
        }),
      ],
    })
    const elapsed = wrapper.find('[data-testid="step-elapsed"]')
    expect(elapsed.exists()).toBe(true)
    expect(elapsed.text()).toBe('5s elapsed')
  })

  it('renders ProgressSpinner for running step marker', () => {
    mountComponent({
      steps: [
        makeStep({
          id: 'step-1',
          status: 'running',
          started_at: '2026-02-17T10:00:00Z',
        }),
      ],
    })
    expect(wrapper.find('[data-testid="step-spinner"]').exists()).toBe(true)
  })

  it('renders Awaiting Approval tag and review button for waiting_approval step', () => {
    mountComponent({
      steps: [
        makeStep({
          id: 'step-1',
          status: 'waiting_approval' as RunStep['status'],
        }),
      ],
    })
    const hitlTag = wrapper.find('[data-testid="hitl-tag"]')
    expect(hitlTag.exists()).toBe(true)

    const hitlButton = wrapper.find('[data-testid="hitl-button"]')
    expect(hitlButton.exists()).toBe(true)

    const link = wrapper.find('a')
    expect(link.attributes('href')).toContain('/approve/step-1')
  })

  it('does not render timeline when loading', () => {
    mountComponent({ isLoading: true, steps: [makeStep({ id: 'step-1' })] })
    expect(wrapper.find('[data-testid="timeline"]').exists()).toBe(false)
  })

  it('displays step error message when present', () => {
    mountComponent({
      steps: [
        makeStep({
          id: 'step-1',
          status: 'failed',
          error_message: 'Container exited with code 1',
        }),
      ],
    })
    const errorEl = wrapper.find('[data-testid="step-error"]')
    expect(errorEl.exists()).toBe(true)
    expect(errorEl.text()).toContain('Container exited with code 1')
  })
})
