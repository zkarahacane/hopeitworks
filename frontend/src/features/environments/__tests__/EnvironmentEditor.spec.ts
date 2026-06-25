import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed } from 'vue'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'
import ToastService from 'primevue/toastservice'

beforeAll(() => {
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
const mockConfirmRequire = vi.fn()
const mockToastAdd = vi.fn()

// useEnvironmentEditor is mocked so the test drives `remove`'s timing directly
// (in-flight double-click guard, #295).
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

// Mock useConfirm so we can capture confirm.require options (custom modal, #299)
// and drive the accept/reject paths deterministically.
vi.mock('primevue/useconfirm', () => ({
  useConfirm: () => ({ require: mockConfirmRequire }),
}))

vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: mockToastAdd }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>
let confirmSpy: ReturnType<typeof vi.spyOn>

function mountComponent() {
  wrapper = mount(EnvironmentEditor, {
    props: { projectId: 'proj-1' },
    global: {
      plugins: [PrimeVue, ConfirmationService, ToastService],
      stubs: { ProgressSpinner: true },
    },
  })
  return wrapper
}

function deleteButton() {
  return wrapper.findAll('button').find((b) => b.text().includes('Delete'))!
}

/** Make confirm.require immediately invoke accept (user confirms). */
function autoAccept() {
  mockConfirmRequire.mockImplementation((options: { accept?: () => void }) => {
    options.accept?.()
  })
}

describe('EnvironmentEditor — delete confirmation + double-click guard (#299, #295)', () => {
  beforeEach(() => {
    mockRemove.mockReset()
    mockSave.mockReset()
    mockConfirmRequire.mockReset()
    mockToastAdd.mockReset()
    // window.confirm must never be invoked anymore (#299). Spy and assert.
    confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    wrapper?.unmount()
    confirmSpy.mockRestore()
    vi.restoreAllMocks()
  })

  // #299 RG1: delete opens the custom design-system modal (confirm.require),
  // never the native window.confirm.
  it('opens the custom confirm modal and never calls window.confirm', async () => {
    mountComponent()
    await deleteButton().trigger('click')

    expect(mockConfirmRequire).toHaveBeenCalledTimes(1)
    expect(window.confirm).not.toHaveBeenCalled()
  })

  // #299: wording aligned on the "cannot be undone" pattern, header + danger
  // accept class consistent with the rest of the design system.
  it('requests confirmation with irreversible wording and danger accept class', async () => {
    mountComponent()
    await deleteButton().trigger('click')

    expect(mockConfirmRequire).toHaveBeenCalledWith(
      expect.objectContaining({
        header: 'Delete Environment',
        message:
          'Delete the environment configuration for this project? This cannot be undone.',
        icon: 'pi pi-exclamation-triangle',
        acceptClass: 'p-button-danger',
        accept: expect.any(Function),
      }),
    )
  })

  // #299 RG2 (confirm): accepting deletes and shows a success toast.
  it('deletes and toasts success when the user confirms', async () => {
    autoAccept()
    mockRemove.mockResolvedValue(true)

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(mockRemove).toHaveBeenCalledTimes(1)
    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'success' }),
    )
  })

  // #299 RG2 (cancel): rejecting (accept never invoked) performs no delete, no toast.
  it('does nothing when the user cancels', async () => {
    mockConfirmRequire.mockImplementation(() => {
      // Simulates Cancel: accept() is never invoked.
    })

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(mockRemove).not.toHaveBeenCalled()
    expect(mockToastAdd).not.toHaveBeenCalled()
  })

  // #295 RG1/RG2: a double-click while the DELETE is in flight fires exactly one delete.
  it('fires exactly one delete on a double-click while the call is in flight', async () => {
    const d = deferred<boolean>()
    mockRemove.mockReturnValue(d.promise)
    // Each confirm immediately accepts, simulating the user confirming twice.
    autoAccept()

    mountComponent()
    const btn = deleteButton()

    await btn.trigger('click')
    // Second click before the first DELETE settles must be ignored by the guard.
    await btn.trigger('click')

    expect(mockRemove).toHaveBeenCalledTimes(1)

    d.resolve(true)
    await flushPromises()
  })

  // #295 RG3: the Delete button is disabled while the call is in flight.
  it('disables the Delete button while the call is in flight', async () => {
    const d = deferred<boolean>()
    mockRemove.mockReturnValue(d.promise)
    autoAccept()

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(deleteButton().attributes('disabled')).toBeDefined()

    d.resolve(true)
    await flushPromises()
    expect(deleteButton().attributes('disabled')).toBeUndefined()
  })

  // #295 RG4 + #299 RG4: a failed delete releases the guard, toasts an error,
  // and the delete stays re-triggerable.
  it('toasts an error and stays re-triggerable when the delete fails', async () => {
    autoAccept()
    mockRemove.mockResolvedValueOnce(false)

    mountComponent()
    await deleteButton().trigger('click')
    await flushPromises()

    expect(mockRemove).toHaveBeenCalledTimes(1)
    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error' }),
    )
    expect(deleteButton().attributes('disabled')).toBeUndefined()

    // Re-triggerable after the failure.
    mockRemove.mockResolvedValueOnce(true)
    await deleteButton().trigger('click')
    await flushPromises()
    expect(mockRemove).toHaveBeenCalledTimes(2)
  })

  // #299 RG3 (coherence): the source file must not contain window.confirm anymore.
  it('contains no window.confirm in the component source', async () => {
    const src = await import('../EnvironmentEditor.vue?raw').then((m) => m.default)
    expect(src).not.toContain('window.confirm')
  })
})
