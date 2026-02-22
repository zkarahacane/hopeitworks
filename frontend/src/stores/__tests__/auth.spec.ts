import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
  },
}))

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  describe('login', () => {
    it('returns true and sets user on success', async () => {
      mockPost.mockResolvedValue({
        data: { id: '1', email: 'user@example.com', name: 'User', role: 'user' },
        error: undefined,
      })

      const store = useAuthStore()
      const result = await store.login('user@example.com', 'password123')

      expect(result).toBe(true)
      expect(store.user).toEqual({
        id: '1', email: 'user@example.com', name: 'User', role: 'user',
      })
      expect(mockPost).toHaveBeenCalledWith('/auth/login', {
        body: { email: 'user@example.com', password: 'password123' },
      })
    })

    it('returns false and sets error on API error', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { message: 'Invalid credentials' } },
      })

      const store = useAuthStore()
      const result = await store.login('user@example.com', 'wrong')

      expect(result).toBe(false)
      expect(store.error).toBe('Invalid credentials')
      expect(store.user).toBeNull()
    })

    it('returns false with fallback error when no message', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: {},
      })

      const store = useAuthStore()
      const result = await store.login('user@example.com', 'wrong')

      expect(result).toBe(false)
      expect(store.error).toBe('Invalid email or password')
    })

    it('returns false with network error on exception', async () => {
      mockPost.mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      const result = await store.login('user@example.com', 'password')

      expect(result).toBe(false)
      expect(store.error).toBe('Network error. Please try again.')
    })

    it('sets loading during request', async () => {
      let resolvePromise: (value: unknown) => void
      mockPost.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        }),
      )

      const store = useAuthStore()
      const promise = store.login('user@example.com', 'password')

      expect(store.loading).toBe(true)

      resolvePromise!({ data: { id: '1', email: 'user@example.com', name: 'User', role: 'user' }, error: undefined })
      await promise

      expect(store.loading).toBe(false)
    })
  })

  describe('logout()', () => {
    it('clears user and error state on success', async () => {
      mockPost.mockResolvedValue({ data: undefined, error: undefined })
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'user' }
      store.error = 'some previous error'
      await store.logout()
      expect(store.user).toBeNull()
      expect(store.error).toBeNull()
    })

    it('clears user state even when apiClient throws', async () => {
      mockPost.mockRejectedValue(new Error('Network error'))
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'user' }
      await store.logout()
      expect(store.user).toBeNull()
      expect(store.error).toBeNull()
    })

    it('calls apiClient.POST /auth/logout', async () => {
      mockPost.mockResolvedValue({ data: undefined, error: undefined })
      const store = useAuthStore()
      await store.logout()
      expect(mockPost).toHaveBeenCalledWith('/auth/logout', {})
    })
  })

  describe('checkAuth', () => {
    it('sets user when authenticated', async () => {
      mockGet.mockResolvedValue({
        data: { id: '1', email: 'user@example.com', name: 'User', role: 'user' },
        error: undefined,
      })

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.user).toEqual({
        id: '1', email: 'user@example.com', name: 'User', role: 'user',
      })
      expect(mockGet).toHaveBeenCalledWith('/auth/me')
    })

    it('sets user to null when not authenticated', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { code: 'UNAUTHORIZED' },
      })

      const store = useAuthStore()
      await store.checkAuth()

      expect(store.user).toBeNull()
    })

    it('sets user to null on network error', async () => {
      mockGet.mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'user' }
      await store.checkAuth()

      expect(store.user).toBeNull()
    })
  })

  describe('forgotPassword', () => {
    it('returns true on success', async () => {
      mockPost.mockResolvedValue({ data: undefined, error: undefined })

      const store = useAuthStore()
      const result = await store.forgotPassword('user@example.com')

      expect(result).toBe(true)
      expect(mockPost).toHaveBeenCalledWith('/auth/forgot-password', {
        body: { email: 'user@example.com' },
      })
    })

    it('returns true on API error (no disclosure)', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: { code: 'NOT_FOUND' },
      })

      const store = useAuthStore()
      const result = await store.forgotPassword('unknown@example.com')

      expect(result).toBe(true)
    })

    it('returns true on network error (no disclosure)', async () => {
      mockPost.mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      const result = await store.forgotPassword('user@example.com')

      expect(result).toBe(true)
    })

    it('never sets error state', async () => {
      mockPost.mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      await store.forgotPassword('user@example.com')

      expect(store.error).toBeNull()
    })

    it('sets loading during request', async () => {
      let resolvePromise: (value: unknown) => void
      mockPost.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        }),
      )

      const store = useAuthStore()
      const promise = store.forgotPassword('user@example.com')

      expect(store.loading).toBe(true)

      resolvePromise!({ data: undefined, error: undefined })
      await promise

      expect(store.loading).toBe(false)
    })
  })

  describe('resetPassword', () => {
    it('returns true on success', async () => {
      mockPost.mockResolvedValue({ data: undefined, error: undefined })

      const store = useAuthStore()
      const result = await store.resetPassword('valid-token', 'newpassword123')

      expect(result).toBe(true)
      expect(store.error).toBeNull()
      expect(mockPost).toHaveBeenCalledWith('/auth/reset-password', {
        body: { token: 'valid-token', password: 'newpassword123' },
      })
    })

    it('returns false and sets error on API error', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { message: 'Token expired' } },
      })

      const store = useAuthStore()
      const result = await store.resetPassword('expired-token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Token expired')
    })

    it('returns false with fallback error when no message in error', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: {},
      })

      const store = useAuthStore()
      const result = await store.resetPassword('bad-token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Token expired or invalid. Please request a new link.')
    })

    it('returns false and sets generic error on network failure', async () => {
      mockPost.mockRejectedValue(new Error('Network failure'))

      const store = useAuthStore()
      const result = await store.resetPassword('token', 'newpassword123')

      expect(result).toBe(false)
      expect(store.error).toBe('Network error. Please try again.')
    })

    it('sets loading during request', async () => {
      let resolvePromise: (value: unknown) => void
      mockPost.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        }),
      )

      const store = useAuthStore()
      const promise = store.resetPassword('token', 'newpassword123')

      expect(store.loading).toBe(true)

      resolvePromise!({ data: undefined, error: undefined })
      await promise

      expect(store.loading).toBe(false)
    })
  })
})
