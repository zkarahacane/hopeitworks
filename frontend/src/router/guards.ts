import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
  }
}

let authCheckPromise: Promise<void> | null = null

/** Resets the cached auth check promise — must be called on logout or session expiry
 *  so the next navigation triggers a fresh checkAuth() instead of reusing a stale result. */
export function resetAuthCheck() {
  authCheckPromise = null
}

export function setupAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()

    // One-time session restore on first navigation — concurrent guards share the same promise
    if (!authCheckPromise) {
      authCheckPromise = auth.checkAuth()
    }
    await authCheckPromise

    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !auth.isAuthenticated) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (to.path === '/login' && auth.isAuthenticated) {
      const redirect = to.query.redirect as string
      return { path: redirect || '/' }
    }
  })
}

/** Guard that redirects non-admin users to dashboard. Must run after setupAuthGuard. */
export function setupAdminGuard(router: Router) {
  router.beforeEach((to) => {
    if (to.meta.requiresAdmin !== true) return

    const auth = useAuthStore()
    if (auth.user?.role !== 'admin') {
      return { path: '/' }
    }
  })
}
