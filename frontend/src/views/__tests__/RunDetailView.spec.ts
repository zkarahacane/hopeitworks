import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, h, defineComponent } from 'vue'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import RunDetailView from '../RunDetailView.vue'
import type { RunWithSteps } from '@/features/runs/composables/useRunDetail'

const mockRunId = ref('run-1')
const mockProjectId = ref('proj-1')

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...(actual as object),
    useRoute: () => ({
      params: { id: mockRunId.value },
      query: { projectId: mockProjectId.value },
    }),
  }
})

// ── Composable mocks ────────────────────────────────────────────────────────────
const mockRun = ref<RunWithSteps | null>(null)
const mockIsLoading = ref(false)
const mockError = ref<Error | null>(null)
const mockFetchRun = vi.fn()

vi.mock('@/features/runs/composables/useRunDetail', () => ({
  useRunDetail: () => ({
    run: mockRun,
    isLoading: mockIsLoading,
    error: mockError,
    fetchRun: mockFetchRun,
    retry: mockFetchRun,
  }),
}))

const mockCostDetail = ref<unknown>(null)
const mockCostLoading = ref(false)
vi.mock('@/features/runs/composables/useRunCosts', () => ({
  useRunCosts: () => ({
    costDetail: mockCostDetail,
    isLoading: mockCostLoading,
    retry: vi.fn(),
  }),
}))

// SSE + step logs touch EventSource — stub them out entirely.
vi.mock('@/composables/useSSE', () => ({
  useSSE: () => ({ status: ref('open'), close: vi.fn() }),
}))
vi.mock('@/features/runs/composables/useStepLogs', () => ({
  useStepLogs: () => ({
    lines: ref([]),
    sseStatus: ref('open'),
    clearLogs: vi.fn(),
  }),
}))

// HITL gate wiring — control gate visibility + capture handler calls.
const mockGateStep = ref<unknown>(null)
const mockHitlRequest = ref<unknown>(null)
const mockIsAtGate = ref(false)
const mockApprove = vi.fn().mockResolvedValue(undefined)
const mockReject = vi.fn().mockResolvedValue(undefined)
const mockRequestChanges = vi.fn().mockResolvedValue(undefined)
vi.mock('@/features/runs/composables/useRunHitl', () => ({
  useRunHitl: () => ({
    hitlRequest: mockHitlRequest,
    gateStep: mockGateStep,
    isAtGate: mockIsAtGate,
    busy: ref(false),
    pendingAction: ref(null),
    actionError: ref(null),
    approve: mockApprove,
    reject: mockReject,
    requestChanges: mockRequestChanges,
    refreshGate: vi.fn(),
  }),
}))

// ── Child component stubs ─────────────────────────────────────────────────────
const RunPipelineViewStub = defineComponent({
  name: 'RunPipelineView',
  props: ['run', 'steps'],
  emits: ['step-selected'],
  setup(props, { emit }) {
    // Root click selects the first step (running) — keeps existing tests stable.
    // A per-step button lets tests select any step (e.g. the seed/pending one).
    return () =>
      h(
        'div',
        {
          'data-testid': 'run-pipeline-view',
          onClick: () => emit('step-selected', props.steps?.[0]),
        },
        (props.steps ?? []).map((s: { id: string }) =>
          h('button', {
            'data-testid': `select-${s.id}`,
            onClick: (e: Event) => {
              e.stopPropagation()
              emit('step-selected', s)
            },
          }),
        ),
      )
  },
})
const LogStreamPanelStub = defineComponent({
  name: 'LogStreamPanel',
  props: ['lines', 'status', 'active', 'stepStatus'],
  setup(props) {
    return () =>
      h('div', {
        'data-testid': 'log-stream-panel',
        'data-active': String(props.active),
        'data-step-status': props.stepStatus ?? '',
      })
  },
})
const RunStepLogPanelStub = defineComponent({
  name: 'RunStepLogPanel',
  props: ['step', 'runId', 'projectId', 'visible', 'retryLoading'],
  emits: ['close', 'update:visible', 'retry'],
  setup(props) {
    return () =>
      h('div', {
        'data-testid': 'step-log-panel',
        'data-visible': String(props.visible),
        'data-step-id': props.step?.id ?? '',
      })
  },
})
const RunCancelConfirmDialogStub = defineComponent({
  name: 'RunCancelConfirmDialog',
  props: ['visible', 'loading'],
  emits: ['confirm', 'cancel', 'update:visible'],
  setup(props) {
    return () => h('div', { 'data-testid': 'cancel-dialog', 'data-visible': String(props.visible) })
  },
})
const HitlGateCardStub = defineComponent({
  name: 'HitlGateCard',
  props: ['storyKey', 'stepName', 'prUrl', 'pendingSince', 'busy', 'pendingAction', 'animated'],
  emits: ['approve', 'requestChanges', 'reject'],
  setup(_, { emit }) {
    return () =>
      h('div', { 'data-testid': 'run-hitl-gate' }, [
        h('button', { 'data-testid': 'gate-approve', onClick: () => emit('approve') }),
        h('button', { 'data-testid': 'gate-request-changes', onClick: () => emit('requestChanges') }),
        h('button', { 'data-testid': 'gate-reject', onClick: () => emit('reject') }),
      ])
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeRun(overrides?: Partial<RunWithSteps>): RunWithSteps {
  return {
    id: '0a807b61-aaaa-bbbb-cccc-000000000000',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: 'running',
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T10:00:00Z',
    steps: [
      {
        id: 'step-1',
        run_id: '0a807b61-aaaa-bbbb-cccc-000000000000',
        step_name: 'dev-story',
        step_order: 0,
        action: 'agent_run',
        status: 'running',
        created_at: '2026-02-17T10:00:00Z',
      },
      {
        id: 'step-2',
        run_id: '0a807b61-aaaa-bbbb-cccc-000000000000',
        step_name: 'code-review',
        step_order: 1,
        action: 'agent_run',
        status: 'pending',
        created_at: '2026-02-17T10:00:00Z',
      },
    ],
    ...overrides,
  }
}

function mountView() {
  wrapper = mount(RunDetailView, {
    global: {
      plugins: [PrimeVue, ToastService, createPinia()],
      stubs: {
        RunPipelineView: RunPipelineViewStub,
        RunStepLogPanel: RunStepLogPanelStub,
        RunCancelConfirmDialog: RunCancelConfirmDialogStub,
        HitlGateCard: HitlGateCardStub,
        RunCostByRole: true,
        StepTimeline: true,
        LogStreamPanel: LogStreamPanelStub,
        LiveProgress: true,
        Toast: true,
        Skeleton: true,
      },
    },
  })
  return wrapper
}

describe('RunDetailView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockRun.value = null
    mockIsLoading.value = false
    mockError.value = null
    mockCostDetail.value = null
    mockCostLoading.value = false
    mockGateStep.value = null
    mockHitlRequest.value = null
    mockIsAtGate.value = false
    mockFetchRun.mockReset()
    mockApprove.mockClear()
    mockReject.mockClear()
    mockRequestChanges.mockClear()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders the run breadcrumb with the short run id (mono)', () => {
    mockRun.value = makeRun()
    mountView()
    const breadcrumb = wrapper.find('[data-testid="run-breadcrumb"]')
    expect(breadcrumb.exists()).toBe(true)
    expect(breadcrumb.text()).toContain('run·0a807b61')
  })

  it('renders the mono run id in the header', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="run-id-mono"]').text()).toContain('run·0a807b61')
  })

  it('renders the run status via StatusBadge (single derived status)', () => {
    mockRun.value = makeRun({ status: 'running' })
    mountView()
    const badge = wrapper.find('[data-testid="run-status-badge"]')
    expect(badge.exists()).toBe(true)
    expect(badge.attributes('data-family')).toBe('running')
  })

  it('renders the live ELAPSED ticker', () => {
    mockRun.value = makeRun()
    mountView()
    const elapsed = wrapper.find('[data-testid="run-elapsed-value"]')
    expect(elapsed.exists()).toBe(true)
    expect(elapsed.text()).toMatch(/^\d{2}:\d{2}$/)
  })

  it('renders RunPipelineView when run is loaded', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="run-pipeline-view"]').exists()).toBe(true)
  })

  it('does NOT render the old RunTimeline component', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.findComponent({ name: 'RunTimeline' }).exists()).toBe(false)
  })

  it('renders RunStepLogPanel', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="step-log-panel"]').exists()).toBe(true)
  })

  it('panel is initially not visible', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="step-log-panel"]').attributes('data-visible')).toBe('false')
  })

  it('opens step log panel when a step is selected from the pipeline', async () => {
    mockRun.value = makeRun()
    mountView()
    await wrapper.find('[data-testid="run-pipeline-view"]').trigger('click')
    await flushPromises()
    const panel = wrapper.find('[data-testid="step-log-panel"]')
    expect(panel.attributes('data-visible')).toBe('true')
    expect(panel.attributes('data-step-id')).toBe('step-1')
  })

  it('closes the step log panel on close event', async () => {
    mockRun.value = makeRun()
    mountView()
    await wrapper.find('[data-testid="run-pipeline-view"]').trigger('click')
    await flushPromises()
    wrapper.findComponent(RunStepLogPanelStub).vm.$emit('close')
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="step-log-panel"]').attributes('data-visible')).toBe('false')
  })

  it('renders cancel/pause buttons when running', () => {
    mockRun.value = makeRun({ status: 'running' })
    mountView()
    expect(wrapper.find('[data-testid="cancel-run-btn"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="pause-run-btn"]').exists()).toBe(true)
  })

  it('renders resume button when paused', () => {
    mockRun.value = makeRun({ status: 'paused' })
    mountView()
    expect(wrapper.find('[data-testid="resume-run-btn"]').exists()).toBe(true)
  })

  it('does not render lifecycle buttons when completed', () => {
    mockRun.value = makeRun({ status: 'completed' })
    mountView()
    expect(wrapper.find('[data-testid="cancel-run-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="pause-run-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="resume-run-btn"]').exists()).toBe(false)
  })

  it('renders the cancel dialog component', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="cancel-dialog"]').exists()).toBe(true)
  })

  it('shows loading skeleton when isLoading is true', () => {
    mockIsLoading.value = true
    mountView()
    expect(wrapper.findComponent({ name: 'Skeleton' }).exists()).toBe(true)
    expect(wrapper.find('[data-testid="run-pipeline-view"]').exists()).toBe(false)
  })

  it('shows error message when an error occurs', () => {
    mockError.value = new Error('Failed to load run')
    mountView()
    expect(wrapper.text()).toContain('Failed to load run')
  })

  // ── HITL gate wiring ──────────────────────────────────────────────────────────
  it('does not render the gate card when not at a gate', () => {
    mockRun.value = makeRun()
    mockIsAtGate.value = false
    mountView()
    expect(wrapper.find('[data-testid="run-hitl-gate"]').exists()).toBe(false)
  })

  it('renders the gate card when at a human gate', () => {
    mockRun.value = makeRun({ status: 'paused' })
    mockIsAtGate.value = true
    mockGateStep.value = { id: 'step-3', step_name: 'Approval gate', action: 'human', status: 'running' }
    mockHitlRequest.value = { id: 'hitl-1', status: 'pending', story_key: 'S-02', story_title: 'Setup CI' }
    mountView()
    expect(wrapper.find('[data-testid="run-hitl-gate"]').exists()).toBe(true)
  })

  it('wires the gate Approve emit to the real approve action', async () => {
    mockRun.value = makeRun({ status: 'paused' })
    mockIsAtGate.value = true
    mockGateStep.value = { id: 'step-3', step_name: 'Approval gate', action: 'human', status: 'running' }
    mockHitlRequest.value = { id: 'hitl-1', status: 'pending' }
    mountView()
    await wrapper.find('[data-testid="gate-approve"]').trigger('click')
    await flushPromises()
    expect(mockApprove).toHaveBeenCalledTimes(1)
  })

  it('wires the gate Request changes emit to requestChanges', async () => {
    mockRun.value = makeRun({ status: 'paused' })
    mockIsAtGate.value = true
    mockGateStep.value = { id: 'step-3', step_name: 'Approval gate', action: 'human', status: 'running' }
    mockHitlRequest.value = { id: 'hitl-1', status: 'pending' }
    mountView()
    await wrapper.find('[data-testid="gate-request-changes"]').trigger('click')
    await flushPromises()
    expect(mockRequestChanges).toHaveBeenCalledTimes(1)
  })

  it('wires the gate Reject emit to the real reject action', async () => {
    mockRun.value = makeRun({ status: 'paused' })
    mockIsAtGate.value = true
    mockGateStep.value = { id: 'step-3', step_name: 'Approval gate', action: 'human', status: 'running' }
    mockHitlRequest.value = { id: 'hitl-1', status: 'pending' }
    mountView()
    await wrapper.find('[data-testid="gate-reject"]').trigger('click')
    await flushPromises()
    expect(mockReject).toHaveBeenCalledTimes(1)
  })

  it('passes run and steps to RunPipelineView', () => {
    const run = makeRun()
    mockRun.value = run
    mountView()
    const pipelineView = wrapper.findComponent(RunPipelineViewStub)
    expect(pipelineView.props('run')).toEqual(run)
    expect(pipelineView.props('steps')).toEqual(run.steps)
  })

  // ── STREAM panel selection wiring (#297) ────────────────────────────────────────
  it('marks the STREAM panel inactive when no step runs and none is selected (RG3)', () => {
    // All steps pending → no running step + nothing selected yet → no target.
    mockRun.value = makeRun({
      status: 'pending',
      steps: [
        {
          id: 'step-1',
          run_id: 'run-1',
          step_name: 'dev-story',
          step_order: 0,
          action: 'agent_run',
          status: 'pending',
          created_at: '2026-02-17T10:00:00Z',
        },
      ],
    })
    mountView()
    const panel = wrapper.find('[data-testid="log-stream-panel"]')
    expect(panel.attributes('data-active')).toBe('false')
    expect(panel.attributes('data-step-status')).toBe('')
  })

  it('marks the STREAM panel active + forwards step status when a seed step is selected (RG1)', async () => {
    // No running step; user selects a pending (seed) step → panel must be active
    // with its status so the panel can render "No logs available", not idle.
    mockRun.value = makeRun({
      status: 'pending',
      steps: [
        {
          id: 'step-seed',
          run_id: 'run-1',
          step_name: 'setup',
          step_order: 0,
          action: 'git_branch',
          status: 'pending',
          created_at: '2026-02-17T10:00:00Z',
        },
      ],
    })
    mountView()
    await wrapper.find('[data-testid="select-step-seed"]').trigger('click')
    await flushPromises()
    const panel = wrapper.find('[data-testid="log-stream-panel"]')
    expect(panel.attributes('data-active')).toBe('true')
    expect(panel.attributes('data-step-status')).toBe('pending')
  })

  it('targets the running step (active) for the STREAM panel by default (RG2)', () => {
    mockRun.value = makeRun() // step-1 is running
    mountView()
    const panel = wrapper.find('[data-testid="log-stream-panel"]')
    expect(panel.attributes('data-active')).toBe('true')
    expect(panel.attributes('data-step-status')).toBe('running')
  })
})
