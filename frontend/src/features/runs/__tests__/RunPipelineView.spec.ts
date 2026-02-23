import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunPipelineView from '../RunPipelineView.vue'
import type { RunWithSteps, RunStep } from '../composables/useRunDetail'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeStep(overrides?: Partial<RunStep>): RunStep {
  return {
    id: 'step-0',
    run_id: 'run-1',
    step_name: 'dev-story',
    step_order: 0,
    action: 'agent_run',
    status: 'completed',
    created_at: '2026-02-17T10:00:00Z',
    ...overrides,
  }
}

function makeRun(overrides?: Partial<RunWithSteps>): RunWithSteps {
  return {
    id: 'run-1',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: 'running',
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T10:00:00Z',
    steps: [],
    ...overrides,
  }
}

function mountView(props: { run: RunWithSteps | null; steps: RunStep[] }) {
  wrapper = mount(RunPipelineView, {
    props,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunPipelineView', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('shows loading skeleton when run is null', () => {
    mountView({ run: null, steps: [] })
    expect(wrapper.find('[data-testid="pipeline-loading"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="pipeline-view"]').exists()).toBe(false)
  })

  it('renders pipeline view when run is provided', () => {
    mountView({ run: makeRun(), steps: [] })
    expect(wrapper.find('[data-testid="pipeline-view"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="pipeline-loading"]').exists()).toBe(false)
  })

  it('renders one column labeled "Pipeline" when no groups in snapshot', () => {
    mountView({ run: makeRun(), steps: [makeStep()] })
    const columns = wrapper.findAll('[data-testid="stage-column"]')
    expect(columns).toHaveLength(1)
    expect(wrapper.find('[data-testid="stage-header"]').text()).toBe('Pipeline')
  })

  it('renders three columns when snapshot has three groups', () => {
    const run = makeRun({
      pipeline_config_snapshot: {
        groups: [
          { id: 'setup', name: 'Setup', steps: [{ id: 's1' }] },
          { id: 'dev', name: 'Development', steps: [{ id: 's2' }, { id: 's3' }] },
          { id: 'review', name: 'Review', steps: [{ id: 's4' }] },
        ],
      },
    })
    const steps = [
      makeStep({ id: 'step-0', step_order: 0 }),
      makeStep({ id: 'step-1', step_order: 1 }),
      makeStep({ id: 'step-2', step_order: 2 }),
      makeStep({ id: 'step-3', step_order: 3 }),
    ]
    mountView({ run, steps })
    const columns = wrapper.findAll('[data-testid="stage-column"]')
    expect(columns).toHaveLength(3)
    const headers = wrapper.findAll('[data-testid="stage-header"]')
    expect(headers[0]!.text()).toBe('Setup')
    expect(headers[1]!.text()).toBe('Development')
    expect(headers[2]!.text()).toBe('Review')
  })

  it('distributes steps to correct columns', () => {
    const run = makeRun({
      pipeline_config_snapshot: {
        groups: [
          { id: 'a', name: 'A', steps: [{ id: 's1' }] },
          { id: 'b', name: 'B', steps: [{ id: 's2' }] },
        ],
      },
    })
    const steps = [
      makeStep({ id: 'step-0', step_order: 0, step_name: 'first' }),
      makeStep({ id: 'step-1', step_order: 1, step_name: 'second' }),
    ]
    mountView({ run, steps })
    const rows = wrapper.findAll('[data-testid="job-row"]')
    expect(rows).toHaveLength(2)
  })

  it('emits step-selected when a step is clicked', async () => {
    const step = makeStep({ id: 'step-0', step_order: 0 })
    mountView({ run: makeRun(), steps: [step] })
    await wrapper.find('[data-testid="job-row"]').trigger('click')
    expect(wrapper.emitted('step-selected')).toBeTruthy()
    expect(wrapper.emitted('step-selected')![0]).toEqual([step])
  })

  it('renders fallback single column when pipeline_config_snapshot is undefined', () => {
    const run = makeRun({ pipeline_config_snapshot: undefined })
    mountView({ run, steps: [makeStep()] })
    const columns = wrapper.findAll('[data-testid="stage-column"]')
    expect(columns).toHaveLength(1)
    expect(wrapper.find('[data-testid="stage-header"]').text()).toBe('Pipeline')
  })

  it('renders fallback single column when groups is empty array', () => {
    const run = makeRun({
      pipeline_config_snapshot: { groups: [] },
    })
    mountView({ run, steps: [makeStep()] })
    const columns = wrapper.findAll('[data-testid="stage-column"]')
    expect(columns).toHaveLength(1)
    expect(wrapper.find('[data-testid="stage-header"]').text()).toBe('Pipeline')
  })
})
