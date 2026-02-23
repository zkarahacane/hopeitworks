import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunStageColumn from '../RunStageColumn.vue'
import type { RunStep } from '../composables/useRunDetail'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeStep(overrides?: Partial<RunStep>): RunStep {
  return {
    id: 'step-1',
    run_id: 'run-1',
    step_name: 'dev-story',
    step_order: 0,
    action: 'agent_run',
    status: 'completed',
    created_at: '2026-02-17T10:00:00Z',
    ...overrides,
  }
}

function mountColumn(props: {
  stageName: string
  steps: RunStep[]
  selectedStepId?: string | null
  isLast?: boolean
}) {
  wrapper = mount(RunStageColumn, {
    props: {
      stageName: props.stageName,
      steps: props.steps,
      selectedStepId: props.selectedStepId ?? null,
      isLast: props.isLast ?? false,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunStageColumn', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders stage name in header', () => {
    mountColumn({ stageName: 'Development', steps: [] })
    expect(wrapper.find('[data-testid="stage-header"]').text()).toBe('Development')
  })

  it('renders correct number of job rows', () => {
    const steps = [
      makeStep({ id: 'step-1', step_name: 'dev-story' }),
      makeStep({ id: 'step-2', step_name: 'code-review' }),
      makeStep({ id: 'step-3', step_name: 'merge' }),
    ]
    mountColumn({ stageName: 'Dev', steps })
    const rows = wrapper.findAll('[data-testid="job-row"]')
    expect(rows).toHaveLength(3)
  })

  it('shows empty stage message when no steps', () => {
    mountColumn({ stageName: 'Empty', steps: [] })
    expect(wrapper.find('[data-testid="empty-stage"]').exists()).toBe(true)
  })

  it('emits step-selected when a job row is clicked', async () => {
    const step = makeStep({ id: 'step-1' })
    mountColumn({ stageName: 'Dev', steps: [step] })
    await wrapper.find('[data-testid="job-row"]').trigger('click')
    expect(wrapper.emitted('step-selected')).toBeTruthy()
    expect(wrapper.emitted('step-selected')![0]).toEqual([step])
  })

  it('shows connector arrow when not the last column', () => {
    mountColumn({ stageName: 'Dev', steps: [], isLast: false })
    expect(wrapper.find('[data-testid="stage-connector"]').exists()).toBe(true)
  })

  it('hides connector arrow for the last column', () => {
    mountColumn({ stageName: 'Dev', steps: [], isLast: true })
    expect(wrapper.find('[data-testid="stage-connector"]').exists()).toBe(false)
  })

  it('applies border-r class when not last column', () => {
    mountColumn({ stageName: 'Dev', steps: [], isLast: false })
    const col = wrapper.find('[data-testid="stage-column"]')
    expect(col.classes()).toContain('border-r')
  })

  it('does not apply border-r class on last column', () => {
    mountColumn({ stageName: 'Dev', steps: [], isLast: true })
    const col = wrapper.find('[data-testid="stage-column"]')
    expect(col.classes()).not.toContain('border-r')
  })
})
