import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed, h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import type { Agent } from '@/stores/agents'
import AgentListView from '../AgentListView.vue'

/** A promise whose resolution is controlled by the test (in-flight window). */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

const mockPush = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'proj-1' } }),
  useRouter: () => ({ push: mockPush }),
}))

const mockFetchAgents = vi.fn()
const mockRetry = vi.fn()
const mockAgents = ref<Agent[]>([])
const mockIsLoading = ref(false)
const mockError = ref<string | null>(null)

vi.mock('@/composables/useAgents', () => ({
  useAgents: () => ({
    agents: computed(() => mockAgents.value),
    pagination: computed(() => null),
    isLoading: computed(() => mockIsLoading.value),
    error: computed(() => mockError.value),
    fetchAgents: mockFetchAgents,
    retry: mockRetry,
  }),
}))

const mockUser = ref<{ role: string } | null>(null)

vi.mock('@/composables/useAuth', () => ({
  useAuth: () => ({
    user: computed(() => mockUser.value),
  }),
}))

// Capture the accept callback handed to PrimeVue's confirm.require so the test
// can invoke it twice (simulating a double-click on the dialog's accept).
let capturedAccept: (() => void) | null = null
vi.mock('primevue/useconfirm', () => ({
  useConfirm: () => ({
    require: (options: { accept: () => void }) => {
      capturedAccept = options.accept
    },
  }),
}))

const mockDeleteAgent = vi.fn()
vi.mock('@/stores/agents', () => ({
  useAgentsStore: () => ({ deleteAgent: mockDeleteAgent }),
}))

// Stub AgentTable: exposes a delete button + reflects the isDeleting predicate.
const AgentTableStub = defineComponent({
  name: 'AgentTable',
  props: ['agents', 'isAdmin', 'isDeleting'],
  emits: ['rowClick', 'delete'],
  setup(props, { emit }) {
    return () =>
      h('div', { 'data-testid': 'agent-table' }, [
        h('button', {
          'data-testid': 'delete-a1',
          disabled: props.isDeleting?.('a1') ?? false,
          onClick: () => emit('delete', 'a1'),
        }),
      ])
  },
})

/** Stub the EmptyState so we can assert the single CTA testid through it. */
const AgentEmptyStateStub = defineComponent({
  name: 'AgentEmptyState',
  props: ['isAdmin'],
  emits: ['createClick'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        { 'data-testid': 'empty-state' },
        props.isAdmin
          ? [
              h(
                'button',
                {
                  'data-testid': 'empty-create-agent-button',
                  onClick: () => emit('createClick'),
                },
                'New Agent',
              ),
            ]
          : ['No agents'],
      )
  },
})

const sampleAgents: Agent[] = [
  {
    id: 'a1',
    name: 'Implement Agent',
    model: 'claude-opus-4-6',
    image: 'ghcr.io/org/agent:latest',
    template_content: 'You are a developer...',
    scope: 'project',
    project_id: 'proj-1',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
]

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent() {
  wrapper = mount(AgentListView, {
    global: {
      plugins: [PrimeVue, ToastService, createPinia()],
      stubs: {
        AgentTable: AgentTableStub,
        AgentEmptyState: AgentEmptyStateStub,
        ConfirmDialog: true,
        Toast: true,
        Skeleton: true,
        Message: true,
      },
    },
  })
  return wrapper
}

function deleteTrigger() {
  return wrapper.find('[data-testid="delete-a1"]')
}

/** Count visible "New Agent" buttons across the whole view (header + CTA). */
function newAgentButtons() {
  return wrapper.findAll('button').filter((b) => b.text().includes('New Agent'))
}

function resetState() {
  setActivePinia(createPinia())
  capturedAccept = null
  mockPush.mockReset()
  mockFetchAgents.mockReset()
  mockRetry.mockReset()
  mockDeleteAgent.mockReset()
  mockAgents.value = []
  mockIsLoading.value = false
  mockError.value = null
  mockUser.value = null
}

describe('AgentListView — delete double-click guard (#295)', () => {
  beforeEach(() => {
    resetState()
    // The guard suite operates on a populated table as an admin.
    mockUser.value = { role: 'admin' }
    mockAgents.value = sampleAgents
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('fires exactly one DELETE when the dialog accept is double-clicked (RG1, RG2)', async () => {
    const d = deferred<boolean>()
    mockDeleteAgent.mockReturnValue(d.promise)

    mountComponent()
    await deleteTrigger().trigger('click')
    expect(capturedAccept).toBeTypeOf('function')

    // Two rapid accepts before the DELETE settles → only one store call.
    capturedAccept!()
    capturedAccept!()
    await flushPromises()

    expect(mockDeleteAgent).toHaveBeenCalledTimes(1)
    expect(mockDeleteAgent).toHaveBeenCalledWith('proj-1', 'a1')

    d.resolve(true)
    await flushPromises()
  })

  it('marks the row delete button disabled while its DELETE is in flight (RG3)', async () => {
    const d = deferred<boolean>()
    mockDeleteAgent.mockReturnValue(d.promise)

    mountComponent()
    await deleteTrigger().trigger('click')
    capturedAccept!()
    await flushPromises()

    expect(deleteTrigger().attributes('disabled')).toBeDefined()

    d.resolve(true)
    await flushPromises()
    expect(deleteTrigger().attributes('disabled')).toBeUndefined()
  })

  it('releases the guard after a failure so the agent is re-deletable (RG4)', async () => {
    mockDeleteAgent.mockResolvedValueOnce(false)

    mountComponent()
    await deleteTrigger().trigger('click')
    capturedAccept!()
    await flushPromises()

    expect(mockDeleteAgent).toHaveBeenCalledTimes(1)
    expect(deleteTrigger().attributes('disabled')).toBeUndefined()

    // Re-triggerable after the error.
    mockDeleteAgent.mockResolvedValueOnce(true)
    await deleteTrigger().trigger('click')
    capturedAccept!()
    await flushPromises()
    expect(mockDeleteAgent).toHaveBeenCalledTimes(2)
  })
})

describe('AgentListView — empty-state CTA disambiguation (#304)', () => {
  beforeEach(() => {
    resetState()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('fetches agents on mount', () => {
    mountComponent()
    expect(mockFetchAgents).toHaveBeenCalled()
  })

  it('RG1: admin + 0 agent shows the empty state with exactly one New Agent button (the CTA), header hidden', async () => {
    mockUser.value = { role: 'admin' }
    mockAgents.value = []
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(true)
    // Header button must be gone in empty state.
    expect(wrapper.find('[data-testid="create-agent-button"]').exists()).toBe(false)
    // Exactly one "New Agent" button in the DOM, and it is the CTA.
    expect(newAgentButtons()).toHaveLength(1)
    expect(wrapper.find('[data-testid="empty-create-agent-button"]').exists()).toBe(true)
  })

  it('RG1: clicking the empty-state CTA navigates to the create page', async () => {
    mockUser.value = { role: 'admin' }
    mockAgents.value = []
    mountComponent()
    await wrapper.vm.$nextTick()

    await wrapper.find('[data-testid="empty-create-agent-button"]').trigger('click')

    expect(mockPush).toHaveBeenCalledWith({
      name: 'agent-create',
      params: { id: 'proj-1' },
    })
  })

  it('RG2: non-admin + 0 agent shows no New Agent button at all', async () => {
    mockUser.value = { role: 'member' }
    mockAgents.value = []
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(true)
    expect(newAgentButtons()).toHaveLength(0)
    expect(wrapper.find('[data-testid="create-agent-button"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="empty-create-agent-button"]').exists()).toBe(false)
  })

  it('non-regression: admin with agents shows the header button (no empty state)', async () => {
    mockUser.value = { role: 'admin' }
    mockAgents.value = sampleAgents
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="agent-table"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="create-agent-button"]').exists()).toBe(true)
    expect(newAgentButtons()).toHaveLength(1)
  })

  it('non-regression: admin while loading shows the header button (no empty state)', async () => {
    mockUser.value = { role: 'admin' }
    mockAgents.value = []
    mockIsLoading.value = true
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="create-agent-button"]').exists()).toBe(true)
  })

  it('non-regression: admin on error shows the header button (no empty state)', async () => {
    mockUser.value = { role: 'admin' }
    mockAgents.value = []
    mockError.value = 'boom'
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="create-agent-button"]').exists()).toBe(true)
  })
})
