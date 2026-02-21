import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProfile } from '../useProfile'
import { useAuthStore } from '@/stores/auth'

const mockApiClient = {
  GET: vi.fn(),
  PUT: vi.fn(),
  POST: vi.fn(),
  DELETE: vi.fn(),
}

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockApiClient.GET(...args),
    PUT: (...args: unknown[]) => mockApiClient.PUT(...args),
    POST: (...args: unknown[]) => mockApiClient.POST(...args),
    DELETE: (...args: unknown[]) => mockApiClient.DELETE(...args),
  },
}))

const userFixture = {
  id: '1',
  email: 'test@example.com',
  name: 'Test User',
  role: 'user' as const,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

describe('useProfile', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('exposes user as a computed ref', () => {
    const { user } = useProfile()
    expect(user.value).toBeNull()
  })

  describe('fetchMe', () => {
    it('populates user on success', async () => {
      mockApiClient.GET.mockResolvedValueOnce({ data: userFixture, error: undefined })

      const { user, fetchMe } = useProfile()
      await fetchMe.execute()

      expect(mockApiClient.GET).toHaveBeenCalledWith('/users/me')
      expect(user.value).toBeTruthy()
      expect(user.value?.email).toBe('test@example.com')
      expect(fetchMe.error.value).toBeNull()
    })

    it('sets error on API error', async () => {
      mockApiClient.GET.mockResolvedValueOnce({ data: undefined, error: { message: 'fail' } })

      const { user, fetchMe } = useProfile()
      await fetchMe.execute()

      expect(user.value).toBeNull()
    })

    it('sets error on network failure', async () => {
      mockApiClient.GET.mockRejectedValueOnce(new Error('Network error'))

      const { user, fetchMe } = useProfile()
      await fetchMe.execute()

      expect(user.value).toBeNull()
    })
  })

  describe('updateMe', () => {
    it('updates store user on success', async () => {
      const updatedUser = { ...userFixture, name: 'Updated Name' }
      mockApiClient.GET.mockResolvedValueOnce({ data: userFixture, error: undefined })
      mockApiClient.PUT.mockResolvedValueOnce({ data: updatedUser, error: undefined })

      const { user, fetchMe, updateMe } = useProfile()
      await fetchMe.execute()
      const result = await updateMe.execute({ name: 'Updated Name' })

      expect(mockApiClient.PUT).toHaveBeenCalledWith('/users/me', {
        body: { name: 'Updated Name' },
      })
      expect(result).toBeTruthy()
      expect(user.value?.name).toBe('Updated Name')
      expect(updateMe.error.value).toBeNull()
    })

    it('sets error on API error', async () => {
      mockApiClient.PUT.mockResolvedValueOnce({ data: undefined, error: { message: 'fail' } })

      const { updateMe } = useProfile()
      const result = await updateMe.execute({ name: 'Test' })

      expect(result).toBeNull()
      expect(updateMe.error.value).toBeTruthy()
    })
  })

  describe('changePassword', () => {
    it('resolves without error on success', async () => {
      mockApiClient.PUT.mockResolvedValueOnce({ data: undefined, error: undefined })

      const { changePassword } = useProfile()
      await changePassword.execute({
        current_password: 'old',
        new_password: 'newpassword',
      })

      expect(mockApiClient.PUT).toHaveBeenCalledWith('/users/me/password', {
        body: { current_password: 'old', new_password: 'newpassword' },
      })
      expect(changePassword.error.value).toBeNull()
    })

    it('sets error on API error', async () => {
      mockApiClient.PUT.mockResolvedValueOnce({
        data: undefined,
        error: { message: 'Wrong password' },
      })

      const { changePassword } = useProfile()
      await changePassword.execute({
        current_password: 'wrong',
        new_password: 'newpassword',
      })

      expect(changePassword.error.value).toBeTruthy()
      expect(changePassword.error.value?.message).toContain('Failed to change password')
    })
  })
})
