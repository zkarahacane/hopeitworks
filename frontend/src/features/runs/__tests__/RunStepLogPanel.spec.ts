import { describe, it, expect, vi, afterEach } from 'vitest'
import { ref } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunStepLogPanel from '../RunStepLogPanel.vue'
import type { components } from '@/api/schema'
import { useStepLogs } from '../composables/useStepLogs'

type RunStep = components['schemas']['RunStep']

const mockLines = ref<unknown[]>([])
const mockSseStatus = ref('open')
const mockClearLogs = vi.fn()

vi.mock('../composables/useStepLogs', () => ({
  useStepLogs: vi.fn(() => ({
    lines: mockLines,
    sseStatus: mockSseStatus,
    clearLogs: mockClearLogs,
  })),
}))

const mockedUseStepLogs = vi.mocked(useStepLogs)

vi.mock('@/ui/composed/LogViewer.vue', () => ({
  default: {
    name: 'LogViewer',
    template: '<div data-testid="log-viewer" />',
    props: ['lines', 'status'],
  },
}))

vi.mock('primevue/drawer', () => ({
  default: {
    name: 'Drawer',
    template: `
      <div v-if="visible" data-testid="step-log-panel">
        <slot name="header" />
        <slot />
      </div>
    `,
    props: ['visible', 'position', 'modal', 'dismissable', 'closeOnEscape', 'pt'],
    emits: ['update:visible'],
  },
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeStep(overrides?: Partial<RunStep>): RunStep {
  return {
    id: 'step-1',
    run_id: 'run-1',
    step_name: 'dev-story',
    step_order: 1,
    action: 'agent_run',
    status: 'running',
    started_at: '2026-02-17T10:30:00Z',
    created_at: '2026-02-17T10:00:00Z',
    parent_step_id: null,
    retry_count: null,
    retry_type: null,
    ...overrides,
  }
}

function mountPanel(props: {
  step: RunStep | null
  visible: boolean
  runId?: string
  projectId?: string
}) {
  wrapper = mount(RunStepLogPanel, {
    props: {
      runId: props.runId ?? 'run-1',
      projectId: props.projectId ?? 'proj-1',
      step: props.step,
      visible: props.visible,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunStepLogPanel', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders the panel when visible is true with a step', () => {
    mountPanel({ step: makeStep(), visible: true })
    expect(wrapper.find('[data-testid="step-log-panel"]').exists()).toBe(true)
  })

  it('does not render panel content when visible is false', () => {
    mountPanel({ step: makeStep(), visible: false })
    expect(wrapper.find('[data-testid="step-log-panel"]').exists()).toBe(false)
  })

  it('displays step name in the header', () => {
    mountPanel({ step: makeStep({ step_name: 'code-review' }), visible: true })
    const stepName = wrapper.find('[data-testid="step-name"]')
    expect(stepName.exists()).toBe(true)
    expect(stepName.text()).toBe('code-review')
  })

  it('displays step status tag', () => {
    mountPanel({ step: makeStep({ status: 'failed' }), visible: true })
    const statusTag = wrapper.find('[data-testid="step-status"]')
    expect(statusTag.exists()).toBe(true)
    expect(statusTag.text()).toBe('failed')
  })

  it('displays started_at timestamp', () => {
    mountPanel({
      step: makeStep({ started_at: '2026-02-17T10:30:00Z' }),
      visible: true,
    })
    const startedAt = wrapper.find('[data-testid="step-started-at"]')
    expect(startedAt.exists()).toBe(true)
    expect(startedAt.text()).toContain('Started:')
  })

  it('displays completed_at when present', () => {
    mountPanel({
      step: makeStep({
        started_at: '2026-02-17T10:30:00Z',
        completed_at: '2026-02-17T10:35:00Z',
        status: 'completed',
      }),
      visible: true,
    })
    const completedAt = wrapper.find('[data-testid="step-completed-at"]')
    expect(completedAt.exists()).toBe(true)
    expect(completedAt.text()).toContain('Completed:')
  })

  it('shows "Running..." when step is running and has no completed_at', () => {
    mountPanel({
      step: makeStep({ status: 'running', completed_at: undefined }),
      visible: true,
    })
    const runningIndicator = wrapper.find('[data-testid="step-running-indicator"]')
    expect(runningIndicator.exists()).toBe(true)
    expect(runningIndicator.text()).toBe('Running...')
  })

  it('displays duration', () => {
    mountPanel({
      step: makeStep({
        started_at: '2026-02-17T10:30:00Z',
        completed_at: '2026-02-17T10:35:30Z',
        status: 'completed',
      }),
      visible: true,
    })
    const duration = wrapper.find('[data-testid="step-duration"]')
    expect(duration.exists()).toBe(true)
    expect(duration.text()).toContain('Duration:')
    expect(duration.text()).toContain('5m 30s')
  })

  it('displays error message when step has error_message', () => {
    mountPanel({
      step: makeStep({
        status: 'failed',
        error_message: 'Container exited with code 1',
      }),
      visible: true,
    })
    const errorMsg = wrapper.find('[data-testid="step-error-message"]')
    expect(errorMsg.exists()).toBe(true)
    expect(errorMsg.text()).toContain('Container exited with code 1')
  })

  it('does not display error message when step has no error_message', () => {
    mountPanel({
      step: makeStep({ error_message: undefined }),
      visible: true,
    })
    expect(wrapper.find('[data-testid="step-error-message"]').exists()).toBe(false)
  })

  it('renders LogViewer component', () => {
    mountPanel({ step: makeStep(), visible: true })
    expect(wrapper.find('[data-testid="log-viewer"]').exists()).toBe(true)
  })

  it('emits close and update:visible when drawer closes', async () => {
    mountPanel({ step: makeStep(), visible: true })
    const drawer = wrapper.findComponent({ name: 'Drawer' })
    expect(drawer.exists()).toBe(true)
    await drawer.vm.$emit('update:visible', false)
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('calls useStepLogs with correct arguments', () => {
    mountPanel({ step: makeStep({ id: 'step-42' }), visible: true, projectId: 'proj-X', runId: 'run-X' })
    expect(mockedUseStepLogs).toHaveBeenCalledWith('proj-X', 'run-X', expect.any(Object))
  })
})
