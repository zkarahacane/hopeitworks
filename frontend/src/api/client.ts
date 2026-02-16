import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema'
import router from '@/router'

const authMiddleware: Middleware = {
  async onResponse({ response }) {
    if (response.status === 401) {
      await router.push({ name: 'login' })
    }
    return response
  },
}

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include',
})

apiClient.use(authMiddleware)
