import { describe, it, expect, afterEach, vi, beforeEach } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'
import ToastService from 'primevue/toastservice'
import CircuitBreakerBanner from '../CircuitBreakerBanner.vue'

const mockConfirmRequire = vi.fn()
const mockToastAdd = vi.fn()
const mockPost = vi.fn()

vi.mock('primevue/useconfirm', () => ({
  useConfirm: () => ({ require: mockConfirmRequire }),
}))

vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: mockToastAdd }),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => mockPost(...args),
  },
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountBanner(props: { projectId?: string; isAdmin?: boolean } = {}) {
  wrapper = mount(CircuitBreakerBanner, {
    props: {
      projectId: 'proj-1',
      isAdmin: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue, ConfirmationService, ToastService],
    },
  })
  return wrapper
}

describe('CircuitBreakerBanner', () => {
  beforeEach(() => {
    mockConfirmRequire.mockReset()
    mockToastAdd.mockReset()
    mockPost.mockReset()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders the banner message', () => {
    mountBanner()
    expect(wrapper.find('[data-testid="circuit-breaker-banner"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Circuit breaker active')
    expect(wrapper.text()).toContain('all pipeline runs are paused')
  })

  it('does not render Reset button when isAdmin is false', () => {
    mountBanner({ isAdmin: false })
    expect(wrapper.find('[data-testid="reset-button"]').exists()).toBe(false)
  })

  it('renders Reset button when isAdmin is true', () => {
    mountBanner({ isAdmin: true })
    expect(wrapper.find('[data-testid="reset-button"]').exists()).toBe(true)
  })

  it('calls confirm.require when Reset button is clicked', async () => {
    mountBanner({ isAdmin: true })
    await wrapper.find('[data-testid="reset-button"]').trigger('click')
    expect(mockConfirmRequire).toHaveBeenCalledExactlyOnceWith(expect.objectContaining({
        message: 'This will allow new pipeline runs to start. Continue?',
        header: 'Reset Circuit Breaker',
      }))
  })

  it('calls apiClient.POST on confirm accept and emits reset event on success', async () => {
    mockPost.mockResolvedValue({ response: { ok: true, status: 204 } })

    // Capture the accept callback from confirm.require
    mockConfirmRequire.mockImplementation((options: { accept?: () => void }) => {
      options.accept?.()
    })

    mountBanner({ isAdmin: true, projectId: 'proj-abc' })
    await wrapper.find('[data-testid="reset-button"]').trigger('click')
    await flushPromises()

    expect(mockPost).toHaveBeenCalledWith('/projects/{id}/circuit-breaker/reset', {
      params: { path: { id: 'proj-abc' } },
    })
    expect(wrapper.emitted('reset')).toBeDefined()
    expect(wrapper.emitted('reset')).toHaveLength(1)
    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'success' }),
    )
  })

  it('shows error toast and does not emit reset when API call fails', async () => {
    mockPost.mockResolvedValue({ response: { ok: false, status: 500 } })

    mockConfirmRequire.mockImplementation((options: { accept?: () => void }) => {
      options.accept?.()
    })

    mountBanner({ isAdmin: true })
    await wrapper.find('[data-testid="reset-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('reset')).toBeUndefined()
    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error' }),
    )
  })

  it('does not call apiClient.POST when confirm is rejected', async () => {
    // confirm.require but never calls accept
    mockConfirmRequire.mockImplementation((_options: unknown) => {
      // Simulates user clicking Cancel — no accept() invocation
    })

    mountBanner({ isAdmin: true })
    await wrapper.find('[data-testid="reset-button"]').trigger('click')
    await flushPromises()

    expect(mockPost).not.toHaveBeenCalled()
    expect(wrapper.emitted('reset')).toBeUndefined()
  })
})
