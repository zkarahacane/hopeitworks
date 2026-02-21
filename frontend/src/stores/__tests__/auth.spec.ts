import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('forgotPassword', () => {
    it('returns true on 200', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 200 }))

      const store = useAuthStore()
      const result = await store.forgotPassword('user@example.com')

      expect(result).toBe(true)
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: 'user@example.com' }),
      })
    })

    it('returns true on 404 (no disclosure)', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 404 }))

      const store = useAuthStore()
      const result = await store.forgotPassword('unknown@example.com')

      expect(result).toBe(true)
    })

    it('returns true on network error (no disclosure)', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      const result = await store.forgotPassword('user@example.com')

      expect(result).toBe(true)
    })

    it('never sets error state', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      await store.forgotPassword('user@example.com')

      expect(store.error).toBeNull()
    })

    it('sets loading during request', async () => {
      let resolvePromise: (value: Response) => void
      vi.spyOn(globalThis, 'fetch').mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        }),
      )

      const store = useAuthStore()
      const promise = store.forgotPassword('user@example.com')

      expect(store.loading).toBe(true)

      resolvePromise!(new Response(null, { status: 200 }))
      await promise

      expect(store.loading).toBe(false)
    })
  })

  describe('resetPassword', () => {
    it('returns true on 200', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 200 }))

      const store = useAuthStore()
      const result = await store.resetPassword('valid-token', 'newpassword123')

      expect(result).toBe(true)
      expect(store.error).toBeNull()
      expect(globalThis.fetch).toHaveBeenCalledWith('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: 'valid-token', password: 'newpassword123' }),
      })
    })

    it('returns false and sets error on 400', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(
        new Response(
          JSON.stringify({ error: { message: 'Token expired' } }),
          { status: 400, headers: { 'Content-Type': 'application/json' } },
        ),
      )

      const store = useAuthStore()
      const result = await store.resetPassword('expired-token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Token expired')
    })

    it('returns false and sets error on 422', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(
        new Response(
          JSON.stringify({ error: { message: 'Invalid token format' } }),
          { status: 422, headers: { 'Content-Type': 'application/json' } },
        ),
      )

      const store = useAuthStore()
      const result = await store.resetPassword('bad-token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Invalid token format')
    })

    it('returns false with fallback error when API returns non-JSON body', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(
        new Response('Bad Request', { status: 400 }),
      )

      const store = useAuthStore()
      const result = await store.resetPassword('bad-token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Token expired or invalid. Please request a new link.')
    })

    it('returns false and sets generic error on network failure', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      const result = await store.resetPassword('token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Network error. Please try again.')
    })

    it('sets loading during request', async () => {
      let resolvePromise: (value: Response) => void
      vi.spyOn(globalThis, 'fetch').mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        }),
      )

      const store = useAuthStore()
      const promise = store.resetPassword('token', 'newpassword123')

      expect(store.loading).toBe(true)

      resolvePromise!(new Response(null, { status: 200 }))
      await promise

      expect(store.loading).toBe(false)
    })
  })

  describe('logout()', () => {
    it('clears user and error state on success', async () => {
      vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 200 }))
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'member' }
      store.error = 'some previous error'
      await store.logout()
      expect(store.user).toBeNull()
      expect(store.error).toBeNull()
    })

    it('clears user state even when fetch throws', async () => {
      vi.spyOn(globalThis, 'fetch').mockRejectedValue(new Error('Network error'))
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'member' }
      await store.logout()
      expect(store.user).toBeNull()
      expect(store.error).toBeNull()
    })

    it('calls POST /api/v1/auth/logout with credentials include', async () => {
      const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(null, { status: 200 }))
      const store = useAuthStore()
      await store.logout()
      expect(fetchSpy).toHaveBeenCalledWith('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      })
    })
  })
})
