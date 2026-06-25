import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed } from 'vue'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import APIKeyList from '../APIKeyList.vue'

/** A promise whose resolution is controlled by the test (in-flight window). */
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

const sampleKey = {
  id: 'key-1',
  provider: 'claude',
  key_name: 'default',
  key_hint: 'abcd',
  created_at: '2026-01-15T10:00:00Z',
}

const mockFetchKeys = vi.fn()
const mockDeleteKey = vi.fn()

vi.mock('@/composables/useAPIKeys', () => ({
  useAPIKeys: () => ({
    keys: computed(() => [sampleKey]),
    isLoading: computed(() => false),
    error: ref<string | null>(null),
    fetchKeys: mockFetchKeys,
    deleteKey: mockDeleteKey,
  }),
}))

// Capture the accept callback so the test can invoke it twice (double accept).
let capturedAccept: (() => void) | null = null
vi.mock('primevue/useconfirm', () => ({
  useConfirm: () => ({
    require: (options: { accept: () => void }) => {
      capturedAccept = options.accept
    },
  }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent() {
  wrapper = mount(APIKeyList, {
    global: {
      plugins: [PrimeVue, ToastService],
      stubs: {
        ConfirmDialog: true,
        Toast: true,
        APIKeyDialog: true,
      },
    },
  })
  return wrapper
}

function trashButton() {
  return wrapper.find('[aria-label="Delete API key"]')
}

describe('APIKeyList — delete double-click guard (#295)', () => {
  beforeEach(() => {
    capturedAccept = null
    mockFetchKeys.mockReset()
    mockDeleteKey.mockReset()
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
  })

  it('fires exactly one DELETE when the dialog accept is double-clicked (RG1, RG2)', async () => {
    const d = deferred<'deleted'>()
    mockDeleteKey.mockReturnValue(d.promise)

    mountComponent()
    await trashButton().trigger('click')
    expect(capturedAccept).toBeTypeOf('function')

    capturedAccept!()
    capturedAccept!()
    await flushPromises()

    expect(mockDeleteKey).toHaveBeenCalledTimes(1)
    expect(mockDeleteKey).toHaveBeenCalledWith('key-1')

    d.resolve('deleted')
    await flushPromises()
  })

  it('disables the key trash button while its DELETE is in flight (RG3)', async () => {
    const d = deferred<'deleted'>()
    mockDeleteKey.mockReturnValue(d.promise)

    mountComponent()
    await trashButton().trigger('click')
    capturedAccept!()
    await flushPromises()

    expect(trashButton().attributes('disabled')).toBeDefined()

    d.resolve('deleted')
    await flushPromises()
    expect(trashButton().attributes('disabled')).toBeUndefined()
  })

  it('releases the guard after an error so the key is re-deletable (RG4)', async () => {
    mockDeleteKey.mockResolvedValueOnce('error')

    mountComponent()
    await trashButton().trigger('click')
    capturedAccept!()
    await flushPromises()

    expect(mockDeleteKey).toHaveBeenCalledTimes(1)
    expect(trashButton().attributes('disabled')).toBeUndefined()

    mockDeleteKey.mockResolvedValueOnce('deleted')
    await trashButton().trigger('click')
    capturedAccept!()
    await flushPromises()
    expect(mockDeleteKey).toHaveBeenCalledTimes(2)
  })
})
