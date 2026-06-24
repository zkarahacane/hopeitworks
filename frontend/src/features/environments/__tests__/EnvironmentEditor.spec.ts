import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed } from 'vue'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import EnvironmentEditor from '../EnvironmentEditor.vue'

/** A promise whose resolution is controlled by the test (in-flight window). */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

const mockRemove = vi.fn()
const mockSave = vi.fn()

// useEnvironmentEditor is mocked so the test drives `remove`'s timing directly.
vi.mock('@/composables/useEnvironmentEditor', () => ({
  useEnvironmentEditor: () => ({
    stacks: ref<string[]>([]),
    source: ref('declared'),
    services: ref([]),
    commandsPairs: ref([]),
    isLoading: computed(() => false),
    isSaving: computed(() => false),
    error: computed(() => null),
    exists: computed(() => true),
    canSave: computed(() => true),
    addService: vi.fn(),
    removeService: vi.fn(),
    addEnvPair: vi.fn(),
    removeEnvPair: vi.fn(),
    addCommand: vi.fn(),
    removeCommand: vi.fn(),
    save: mockSave,
    remove: mockRemove,
  }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent() {
  wrapper = mount(EnvironmentEditor, {
    props: { projectId: 'proj-1' },
    global: {
      plugins: [PrimeVue, ToastService],
      stubs: { Toast: true, ProgressSpinner: true },
    },
  })
  return wrapper
}

function deleteButton() {
  return wrapper.findAll('button').find((b) => b.text().includes('Delete'))!
}

describe('EnvironmentEditor — delete double-click guard (#295)', () => {
  beforeEach(() => {
    mockRemove.mockReset()
    mockSave.mockReset()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    // PrimeVue Select/MultiSelect call matchMedia, absent in jsdom.
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  afterEach(() => {
    wrapper?.unmount()
    vi.restoreAllMocks()
  })

  it('fires exactly one delete on a double-click while the call is in flight (RG1, RG2)', async () => {
    const d = deferred<boolean>()
    mockRemove.mockReturnValue(d.promise)

    mountComponent()
    const btn = deleteButton()

    await btn.trigger('click')
    // Second click before the first DELETE settles must be ignored.
    await btn.trigger('click')

    expect(mockRemove).toHaveBeenCalledTimes(1)

    d.resolve(true)
    await flushPromises()
  })

  it('disables the Delete button while the call is in flight (RG3)', async () => {
    const d = deferred<boolean>()
    mockRemove.mockReturnValue(d.promise)

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(deleteButton().attributes('disabled')).toBeDefined()

    d.resolve(true)
    await flushPromises()
    expect(deleteButton().attributes('disabled')).toBeUndefined()
  })

  it('releases the guard after an error so delete is re-triggerable (RG4)', async () => {
    mockRemove.mockResolvedValueOnce(false)

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(mockRemove).toHaveBeenCalledTimes(1)
    expect(deleteButton().attributes('disabled')).toBeUndefined()

    // Re-triggerable after the failure.
    mockRemove.mockResolvedValueOnce(true)
    await deleteButton().trigger('click')
    await flushPromises()
    expect(mockRemove).toHaveBeenCalledTimes(2)
  })

  it('does not fire delete when the confirm is dismissed', async () => {
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    mockRemove.mockResolvedValue(true)

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(mockRemove).not.toHaveBeenCalled()
  })
})
