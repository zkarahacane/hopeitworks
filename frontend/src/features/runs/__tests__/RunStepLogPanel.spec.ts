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

vi.mock('@/ui/composed/LogStreamPanel.vue', () => ({
  default: {
    name: 'LogStreamPanel',
    template: '<div data-testid="log-stream-mock" :data-lines="JSON.stringify(lines)" />',
    props: ['lines', 'status', 'active'],
    emits: ['clear'],
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

  it('displays step status via StatusBadge', () => {
    mountPanel({ step: makeStep({ status: 'failed' }), visible: true })
    const statusBadge = wrapper.find('[data-testid="step-status"]')
    expect(statusBadge.exists()).toBe(true)
    // statusToken normalizes "failed" → failed family.
    expect(statusBadge.attributes('data-family')).toBe('failed')
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

  it('renders LogStreamPanel component (replaces the old LogViewer)', () => {
    mountPanel({ step: makeStep(), visible: true })
    expect(wrapper.find('[data-testid="log-stream-mock"]').exists()).toBe(true)
  })

  it('renders the typed step type chip', () => {
    mountPanel({ step: makeStep({ action: 'agent_run' }), visible: true })
    const typeChip = wrapper.find('[data-testid="step-type-chip"]')
    expect(typeChip.exists()).toBe(true)
    expect(typeChip.text()).toContain('agent_run')
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

  it('shows retry button when step status is failed', () => {
    mountPanel({ step: makeStep({ status: 'failed' }), visible: true })
    expect(wrapper.find('[data-testid="retry-step-btn"]').exists()).toBe(true)
  })

  it('does not show retry button when step status is running', () => {
    mountPanel({ step: makeStep({ status: 'running' }), visible: true })
    expect(wrapper.find('[data-testid="retry-step-btn"]').exists()).toBe(false)
  })

  it('does not show retry button when step status is completed', () => {
    mountPanel({ step: makeStep({ status: 'completed' }), visible: true })
    expect(wrapper.find('[data-testid="retry-step-btn"]').exists()).toBe(false)
  })

  it('emits retry with step id when retry button is clicked', async () => {
    mountPanel({ step: makeStep({ id: 'step-99', status: 'failed' }), visible: true })
    await wrapper.find('[data-testid="retry-step-btn"]').trigger('click')
    expect(wrapper.emitted('retry')?.[0]).toEqual(['step-99'])
  })

  it('passes persisted log_tail lines to LogStreamPanel when no live SSE lines exist', () => {
    mockLines.value = []
    mountPanel({
      step: makeStep({ status: 'completed', log_tail: 'line one\nline two\nline three' }),
      visible: true,
    })
    const mock = wrapper.find('[data-testid="log-stream-mock"]')
    const passed = JSON.parse(mock.attributes('data-lines') ?? '[]') as { text: string }[]
    expect(passed).toHaveLength(3)
    expect(passed[0]!.text).toBe('line one')
    expect(passed[2]!.text).toBe('line three')
  })

  it('prefers live SSE lines over log_tail when both exist', () => {
    mockLines.value = [{ text: 'live line', timestamp: new Date() }]
    mountPanel({
      step: makeStep({ status: 'running', log_tail: 'persisted line' }),
      visible: true,
    })
    const mock = wrapper.find('[data-testid="log-stream-mock"]')
    const passed = JSON.parse(mock.attributes('data-lines') ?? '[]') as { text: string }[]
    expect(passed).toHaveLength(1)
    expect(passed[0]!.text).toBe('live line')
  })
})
