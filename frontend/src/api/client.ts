import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema'
import router from '@/router'

let redirecting = false

router.afterEach(() => {
  redirecting = false
})

const authMiddleware: Middleware = {
  async onResponse({ request, response }) {
    if (response.status === 401) {
      const url = new URL(request.url, window.location.origin)
      // Skip redirect for auth endpoints — those 401s are handled by callers
      if (!url.pathname.startsWith('/api/v1/auth/')) {
        const currentRoute = router.currentRoute.value
        const isPublic = currentRoute.meta.requiresAuth === false
        if (!isPublic && !redirecting) {
          redirecting = true
          // Clear auth state BEFORE redirect — prevents guard bounce loop:
          // without this, guard sees isAuthenticated=true on /login and redirects back
          const { useAuthStore } = await import('@/stores/auth')
          const auth = useAuthStore()
          auth.user = null
          await router.push({ name: 'login' })
        }
      }
    }
    return response
  },
}

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include',
})

apiClient.use(authMiddleware)
