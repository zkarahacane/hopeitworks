import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { ref, computed } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'

// --- Mock dependencies before importing AppShell ---

const mockIsAuthenticated = ref(false)
const mockAuthLoading = ref(false)

vi.mock('@/composables/useAuth', () => ({
  useAuth: () => ({
    isAuthenticated: computed(() => mockIsAuthenticated.value),
    loading: computed(() => mockAuthLoading.value),
  }),
}))

const mockRoute = ref({ matched: [{}] })

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRouter: () => ({ push: vi.fn() }),
    useRoute: () => computed(() => mockRoute.value).value,
  }
})

vi.mock('@/stores/layout', () => ({
  useLayoutStore: () => ({
    sidebarCollapsed: false,
    toggleSidebar: vi.fn(),
  }),
}))

vi.mock('@/stores/hitl', () => ({
  useHITLStore: () => ({
    pendingCount: 0,
    pendingItems: [],
    handleResolvedEvent: vi.fn(),
  }),
}))

vi.mock('@/composables/useKeyboard', () => ({
  useKeyboard: vi.fn(),
}))

vi.mock('@/composables/useBreakpoint', () => ({
  useBreakpoint: () => ({ isMobile: ref(false) }),
}))

vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: vi.fn() }),
}))

import AppShell from '../AppShell.vue'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountShell() {
  wrapper = mount(AppShell, {
    global: {
      plugins: [PrimeVue, ToastService, createPinia()],
      stubs: {
        AppHeader: true,
        AppSidebar: true,
        AppStatusBar: true,
        ProgressSpinner: { template: '<div data-testid="progress-spinner" />' },
        Toast: true,
        RouterView: { template: '<div data-testid="router-view" />' },
      },
    },
  })
  return wrapper
}

describe('AppShell loading guard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockIsAuthenticated.value = false
    mockAuthLoading.value = false
    mockRoute.value = { matched: [{}] }
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('shows ProgressSpinner when authLoading=true and routeResolved=false', async () => {
    mockAuthLoading.value = true
    mockRoute.value = { matched: [] } // route not yet resolved

    mountShell()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="progress-spinner"]').exists()).toBe(true)
  })

  it('does not show ProgressSpinner when authLoading=false (normal post-mount state)', async () => {
    mockAuthLoading.value = false
    mockIsAuthenticated.value = false
    mockRoute.value = { matched: [{}] }

    mountShell()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="progress-spinner"]').exists()).toBe(false)
  })

  it('does not show ProgressSpinner when authLoading=true but route is already resolved', async () => {
    // This is the normal case post router.isReady() fix — route is always resolved at mount
    mockAuthLoading.value = true
    mockRoute.value = { matched: [{}] } // route resolved

    mountShell()
    await wrapper.vm.$nextTick()

    // authLoading && !routeResolved → false → spinner not shown
    expect(wrapper.find('[data-testid="progress-spinner"]').exists()).toBe(false)
  })

  it('renders app chrome (authenticated layout) when isAuthenticated=true', async () => {
    mockAuthLoading.value = false
    mockIsAuthenticated.value = true
    mockRoute.value = { matched: [{}] }

    mountShell()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="progress-spinner"]').exists()).toBe(false)
    // AppHeader stub is rendered in authenticated layout
    expect(wrapper.findComponent({ name: 'AppHeader' }).exists()).toBe(true)
  })

  it('renders unauthenticated router-view when isAuthenticated=false and not loading', async () => {
    mockAuthLoading.value = false
    mockIsAuthenticated.value = false
    mockRoute.value = { matched: [{}] }

    mountShell()
    await wrapper.vm.$nextTick()

    // Unauthenticated branch shows router-view directly (login page)
    expect(wrapper.find('[data-testid="router-view"]').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'AppHeader' }).exists()).toBe(false)
  })
})

describe('AppShell router.isReady() error resilience', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockIsAuthenticated.value = false
    mockAuthLoading.value = false
    mockRoute.value = { matched: [{}] }
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('mounts and renders without throwing when route has no matched entries', () => {
    // Simulates the edge case where router.isReady() .catch() mounted the app
    // before navigation completed — AppShell must not crash
    mockRoute.value = { matched: [] }
    mockAuthLoading.value = false
    mockIsAuthenticated.value = false

    expect(() => mountShell()).not.toThrow()
  })

  it('shows spinner when both authLoading and unresolved route occur together', async () => {
    // This is exactly the scenario that .catch() in main.ts can produce:
    // app mounts (due to error recovery), auth is still loading, route not matched
    mockAuthLoading.value = true
    mockRoute.value = { matched: [] }

    mountShell()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="progress-spinner"]').exists()).toBe(true)
  })
})
