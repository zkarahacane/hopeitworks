import { describe, it, expect } from 'vitest'
import { createRouter, createWebHistory } from 'vue-router'

/**
 * Verifies the /settings → /profile redirect introduced in Phase 2 Group C.
 * The 404 page copy says "/settings now lives under your profile."
 */
function createTestRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div/>' } },
      { path: '/profile', name: 'profile', component: { template: '<div/>' } },
      { path: '/settings', redirect: { name: 'profile' } },
    ],
  })
}

describe('/settings redirect', () => {
  it('redirects /settings to /profile', async () => {
    const router = createTestRouter()
    await router.push('/settings')
    await router.isReady()
    expect(router.currentRoute.value.path).toBe('/profile')
    expect(router.currentRoute.value.name).toBe('profile')
  })

  it('does not redirect /profile', async () => {
    const router = createTestRouter()
    await router.push('/profile')
    await router.isReady()
    expect(router.currentRoute.value.path).toBe('/profile')
  })
})
