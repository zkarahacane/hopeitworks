import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

let authChecked = false

export function setupAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()

    // One-time session restore on first navigation
    if (!authChecked) {
      authChecked = true
      await auth.checkAuth()
    }

    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !auth.isAuthenticated) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (to.path === '/login' && auth.isAuthenticated) {
      return { path: '/' }
    }
  })
}
