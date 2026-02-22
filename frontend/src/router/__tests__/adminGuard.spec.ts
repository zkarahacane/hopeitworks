import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createRouter, createWebHistory } from 'vue-router'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { setupAdminGuard } from '../guards'

function createTestRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div>Dashboard</div>' } },
      { path: '/login', name: 'login', component: { template: '<div>Login</div>' }, meta: { requiresAuth: false } },
      {
        path: '/admin/users',
        name: 'admin-users',
        component: { template: '<div>Admin Users</div>' },
        meta: { requiresAuth: true, requiresAdmin: true },
      },
      {
        path: '/projects',
        name: 'projects',
        component: { template: '<div>Projects</div>' },
        meta: { requiresAuth: true },
      },
    ],
  })
}

describe('setupAdminGuard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('allows admin user to access admin routes', async () => {
    const router = createTestRouter()
    setupAdminGuard(router)

    const auth = useAuthStore()
    auth.user = { id: '1', email: 'admin@test.com', name: 'Admin', role: 'admin' }

    await router.push('/admin/users')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/admin/users')
  })

  it('redirects non-admin user to dashboard', async () => {
    const router = createTestRouter()
    setupAdminGuard(router)

    const auth = useAuthStore()
    auth.user = { id: '2', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/admin/users')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })

  it('redirects when user has no role (null user)', async () => {
    const router = createTestRouter()
    setupAdminGuard(router)

    const auth = useAuthStore()
    auth.user = null

    await router.push('/admin/users')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/')
  })

  it('does not interfere with non-admin routes', async () => {
    const router = createTestRouter()
    setupAdminGuard(router)

    const auth = useAuthStore()
    auth.user = { id: '2', email: 'user@test.com', name: 'User', role: 'user' }

    await router.push('/projects')
    await router.isReady()

    expect(router.currentRoute.value.path).toBe('/projects')
  })
})
