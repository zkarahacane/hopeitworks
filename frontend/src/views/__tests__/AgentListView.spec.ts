import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed, h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import AgentListView from '../AgentListView.vue'

/** A promise whose resolution is controlled by the test (in-flight window). */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'proj-1' } }),
  useRouter: () => ({ push: vi.fn() }),
}))

const mockFetchAgents = vi.fn()
const mockAgents = ref<unknown[]>([
  {
    id: 'a1',
    name: 'Implement Agent',
    model: 'claude-opus-4-6',
    image: 'img:1',
    template_content: '',
    scope: 'project',
    project_id: 'proj-1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
])

vi.mock('@/composables/useAgents', () => ({
  useAgents: () => ({
    agents: computed(() => mockAgents.value),
    isLoading: computed(() => false),
    error: computed(() => null),
    fetchAgents: mockFetchAgents,
    retry: vi.fn(),
  }),
}))

vi.mock('@/composables/useAuth', () => ({
  useAuth: () => ({ user: ref({ role: 'admin' }) }),
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent() {
  wrapper = mount(AgentListView, {
    global: {
      plugins: [PrimeVue, ToastService, createPinia()],
      stubs: {
        AgentTable: AgentTableStub,
        AgentEmptyState: true,
        ConfirmDialog: true,
        Toast: true,
        Skeleton: true,
      },
    },
  })
  return wrapper
}

function deleteTrigger() {
  return wrapper.find('[data-testid="delete-a1"]')
}

describe('AgentListView — delete double-click guard (#295)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    capturedAccept = null
    mockFetchAgents.mockReset()
    mockDeleteAgent.mockReset()
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
