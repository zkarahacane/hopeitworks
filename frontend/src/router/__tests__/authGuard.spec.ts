import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createRouter, createWebHistory } from 'vue-router'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { setupAuthGuard, resetAuthCheck } from '../guards'

const mockCheckAuth = vi.fn()

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(),
}))

function createTestRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div>Dashboard</div>' }, meta: { requiresAuth: true } },
      { path: '/login', name: 'login', component: { template: '<div>Login</div>' }, meta: { requiresAuth: false } },
      { path: '/projects', name: 'projects', component: { template: '<div>Projects</div>' }, meta: { requiresAuth: true } },
    ],
  })
}

describe('setupAuthGuard', () => {
  let authStore: { user: { id: string; email: string; name: string; role: string } | null; isAuthenticated: boolean; checkAuth: ReturnType<typeof vi.fn> }

  beforeEach(() => {
    setActivePinia(createPinia())
    mockCheckAuth.mockReset()
    mockCheckAuth.mockResolvedValue(undefined)

    authStore = {
      user: null,
      get isAuthenticated() { return this.user !== null },
      checkAuth: mockCheckAuth,
    }

    vi.mocked(useAuthStore).mockReturnValue(authStore as unknown as ReturnType<typeof useAuthStore>)

    // Reset the module-level authCheckPromise between tests
    resetAuthCheck()
  })

  it('redirects unauthenticated user from protected route to /login', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = null

    await router.push('/projects')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query.redirect).toBe('/projects')
  })

  it('allows authenticated user to access protected route', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = { id: '1', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/projects')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/projects')
  })

  it('redirects authenticated user away from /login to /', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = { id: '1', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/login')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })

  it('redirects authenticated user from /login to the redirect query param', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = { id: '1', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/login?redirect=/projects')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/projects')
  })

  it('calls checkAuth only once across multiple navigations (promise caching)', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = { id: '1', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/projects')
    await router.push('/')
    await router.push('/projects')
    await router.isReady()

    expect(mockCheckAuth).toHaveBeenCalledTimes(1)
  })

  it('calls checkAuth again after resetAuthCheck()', async () => {
    const router = createTestRouter()
    setupAuthGuard(router)

    authStore.user = { id: '1', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/projects')
    await router.isReady()
    expect(mockCheckAuth).toHaveBeenCalledTimes(1)

    // Simulate logout — resets the promise so next navigation re-checks
    resetAuthCheck()

    await router.push('/')
    await router.isReady()
    expect(mockCheckAuth).toHaveBeenCalledTimes(2)
  })
})

describe('resetAuthCheck', () => {
  it('is a callable function', () => {
    expect(typeof resetAuthCheck).toBe('function')
  })

  it('forces a fresh checkAuth on the next navigation', async () => {
    vi.mocked(useAuthStore).mockReturnValue({
      user: { id: '1', email: 'u@t.com', name: 'U', role: 'user' },
      get isAuthenticated() { return true },
      checkAuth: mockCheckAuth,
    } as unknown as ReturnType<typeof useAuthStore>)

    mockCheckAuth.mockReset()
    mockCheckAuth.mockResolvedValue(undefined)
    resetAuthCheck()

    const router = createTestRouter()
    setupAuthGuard(router)

    await router.push('/projects')
    await router.isReady()

    expect(mockCheckAuth).toHaveBeenCalledTimes(1)

    resetAuthCheck()

    await router.push('/')
    await router.isReady()

    expect(mockCheckAuth).toHaveBeenCalledTimes(2)
  })
})
