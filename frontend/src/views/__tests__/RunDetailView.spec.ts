import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, computed, h, defineComponent, type Ref } from 'vue'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import RunDetailView from '../RunDetailView.vue'
import type { RunWithSteps, RunStep } from '@/features/runs/composables/useRunDetail'

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

const mockCostDetail = ref(null)
const mockCostLoading = ref(false)

vi.mock('@/features/runs/composables/useRunCosts', () => ({
  useRunCosts: () => ({
    costDetail: mockCostDetail,
    isLoading: mockCostLoading,
    retry: vi.fn(),
  }),
}))

const RunPipelineViewStub = defineComponent({
  name: 'RunPipelineView',
  props: ['run', 'steps'],
  emits: ['step-selected'],
  setup(_, { emit }) {
    return () =>
      h('div', {
        'data-testid': 'run-pipeline-view',
        onClick: () =>
          emit('step-selected', {
            id: 'step-1',
            run_id: 'run-1',
            step_name: 'dev-story',
            step_order: 0,
            action: 'agent_run',
            status: 'running',
            created_at: '2026-02-17T10:00:00Z',
          }),
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
    return () =>
      h('div', {
        'data-testid': 'cancel-dialog',
        'data-visible': String(props.visible),
      })
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeRun(overrides?: Partial<RunWithSteps>): RunWithSteps {
  return {
    id: 'run-1',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: 'running',
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T10:00:00Z',
    steps: [
      {
        id: 'step-1',
        run_id: 'run-1',
        step_name: 'dev-story',
        step_order: 0,
        action: 'agent_run',
        status: 'running',
        created_at: '2026-02-17T10:00:00Z',
      },
      {
        id: 'step-2',
        run_id: 'run-1',
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
        Toast: true,
        Skeleton: true,
        ProgressBar: true,
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
    mockFetchRun.mockReset()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders page header with "Run Detail" title', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.text()).toContain('Run Detail')
  })

  it('displays run ID when run is loaded', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.text()).toContain('run-1')
  })

  it('displays run status Tag', () => {
    mockRun.value = makeRun({ status: 'running' })
    mountView()
    expect(wrapper.text()).toContain('running')
  })

  it('renders RunPipelineView when run is loaded', () => {
    mockRun.value = makeRun()
    mountView()
    expect(wrapper.find('[data-testid="run-pipeline-view"]').exists()).toBe(true)
  })

  it('does NOT render old RunTimeline component', () => {
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
    const panel = wrapper.find('[data-testid="step-log-panel"]')
    expect(panel.attributes('data-visible')).toBe('false')
  })

  it('opens step log panel when step is selected from pipeline view', async () => {
    mockRun.value = makeRun()
    mountView()

    await wrapper.find('[data-testid="run-pipeline-view"]').trigger('click')
    await flushPromises()

    const panel = wrapper.find('[data-testid="step-log-panel"]')
    expect(panel.attributes('data-visible')).toBe('true')
    expect(panel.attributes('data-step-id')).toBe('step-1')
  })

  it('closes step log panel on close event', async () => {
    mockRun.value = makeRun()
    mountView()

    // Open panel first
    await wrapper.find('[data-testid="run-pipeline-view"]').trigger('click')
    await flushPromises()

    // Close via event
    const panel = wrapper.findComponent(RunStepLogPanelStub)
    panel.vm.$emit('close')
    await wrapper.vm.$nextTick()

    const panelEl = wrapper.find('[data-testid="step-log-panel"]')
    expect(panelEl.attributes('data-visible')).toBe('false')
  })

  it('renders cancel button when run is in progress', () => {
    mockRun.value = makeRun({ status: 'running' })
    mountView()
    expect(wrapper.find('[data-testid="cancel-run-btn"]').exists()).toBe(true)
  })

  it('renders pause button when run is running', () => {
    mockRun.value = makeRun({ status: 'running' })
    mountView()
    expect(wrapper.find('[data-testid="pause-run-btn"]').exists()).toBe(true)
  })

  it('renders resume button when run is paused', () => {
    mockRun.value = makeRun({ status: 'paused' })
    mountView()
    expect(wrapper.find('[data-testid="resume-run-btn"]').exists()).toBe(true)
  })

  it('does not render cancel/pause/resume buttons when run is completed', () => {
    mockRun.value = makeRun({ status: 'completed' })
    mountView()
    expect(wrapper.find('[data-testid="cancel-run-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="pause-run-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="resume-run-btn"]').exists()).toBe(false)
  })

  it('renders cancel dialog component', () => {
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

  it('shows error message when error occurs', () => {
    mockError.value = new Error('Failed to load run')
    mountView()
    expect(wrapper.text()).toContain('Failed to load run')
  })

  it('passes run and steps to RunPipelineView', () => {
    const run = makeRun()
    mockRun.value = run
    mountView()
    const pipelineView = wrapper.findComponent(RunPipelineViewStub)
    expect(pipelineView.props('run')).toEqual(run)
    expect(pipelineView.props('steps')).toEqual(run.steps)
  })
})
