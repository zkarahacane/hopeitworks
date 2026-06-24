import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useAPIKeys } from '../useAPIKeys'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockDelete = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
    DELETE: (...args: unknown[]) => mockDelete(...args),
  },
}))

const sampleKey = {
  id: 'key-1',
  provider: 'claude',
  key_name: 'default',
  key_hint: 'abcd',
  created_at: '2026-01-15T10:00:00Z',
}

describe('useAPIKeys', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockDelete.mockReset()
  })

  it('starts with empty keys and no error', () => {
    const { keys, isLoading, error } = useAPIKeys()
    expect(keys.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  describe('fetchKeys', () => {
    it('populates keys on success', async () => {
      mockGet.mockResolvedValue({ data: [sampleKey], error: undefined })

      const { keys, fetchKeys } = useAPIKeys()
      await fetchKeys()

      expect(mockGet).toHaveBeenCalledWith('/users/me/api-keys')
      expect(keys.value).toEqual([sampleKey])
    })

    it('sets error when API returns error', async () => {
      mockGet.mockResolvedValue({ data: undefined, error: { message: 'fail' } })

      const { keys, error, fetchKeys } = useAPIKeys()
      await fetchKeys()

      expect(keys.value).toEqual([])
      expect(error.value).toBe('Failed to load API keys')
    })

    it('sets error on network failure', async () => {
      mockGet.mockRejectedValue(new Error('Network error'))

      const { error, fetchKeys } = useAPIKeys()
      await fetchKeys()

      expect(error.value).toBe('Network error')
    })

    it('sets fallback error for non-Error thrown value', async () => {
      mockGet.mockRejectedValue('unexpected')

      const { error, fetchKeys } = useAPIKeys()
      await fetchKeys()

      expect(error.value).toBe('Failed to load API keys')
    })
  })

  describe('createKey', () => {
    it('creates a key and refreshes the list on success', async () => {
      mockPost.mockResolvedValue({ data: sampleKey, error: undefined })
      mockGet.mockResolvedValue({ data: [sampleKey], error: undefined })

      const { keys, createKey } = useAPIKeys()
      const result = await createKey('claude', 'default', 'sk-ant-abc123')

      expect(result).toBe(true)
      expect(mockPost).toHaveBeenCalledWith('/users/me/api-keys', {
        body: { provider: 'claude', key_name: 'default', api_key: 'sk-ant-abc123' },
      })
      expect(keys.value).toEqual([sampleKey])
    })

    it('returns false when API returns error', async () => {
      mockPost.mockResolvedValue({ data: undefined, error: { message: 'conflict' } })

      const { error, createKey } = useAPIKeys()
      const result = await createKey('claude', 'default', 'sk-ant-abc')

      expect(result).toBe(false)
      expect(error.value).toBe('Failed to create API key')
    })

    it('returns false on network failure', async () => {
      mockPost.mockRejectedValue(new Error('Network error'))

      const { error, createKey } = useAPIKeys()
      const result = await createKey('claude', 'default', 'sk-ant-abc')

      expect(result).toBe(false)
      expect(error.value).toBe('Network error')
    })
  })

  describe('deleteKey', () => {
    it('removes the key from the list on success', async () => {
      mockGet.mockResolvedValue({ data: [sampleKey], error: undefined })
      mockDelete.mockResolvedValue({ error: undefined })

      const { keys, fetchKeys, deleteKey } = useAPIKeys()
      await fetchKeys()
      expect(keys.value).toHaveLength(1)

      const result = await deleteKey('key-1')

      expect(result).toBe('deleted')
      expect(mockDelete).toHaveBeenCalledWith('/users/me/api-keys/{keyId}', {
        params: { path: { keyId: 'key-1' } },
      })
      expect(keys.value).toHaveLength(0)
    })

    it("returns 'error' when API returns error", async () => {
      mockDelete.mockResolvedValue({ error: { message: 'not found' } })

      const { error, deleteKey } = useAPIKeys()
      const result = await deleteKey('key-missing')

      expect(result).toBe('error')
      expect(error.value).toBe('Failed to delete API key')
    })

    it("returns 'error' on network failure", async () => {
      mockDelete.mockRejectedValue(new Error('Network error'))

      const { error, deleteKey } = useAPIKeys()
      const result = await deleteKey('key-1')

      expect(result).toBe('error')
      expect(error.value).toBe('Network error')
    })

    it('keeps the key on error — no optimistic removal (RG4)', async () => {
      mockGet.mockResolvedValue({ data: [sampleKey], error: undefined })
      mockDelete.mockResolvedValue({ error: { message: 'boom' } })

      const { keys, fetchKeys, deleteKey } = useAPIKeys()
      await fetchKeys()
      expect(keys.value).toHaveLength(1)

      const result = await deleteKey('key-1')

      expect(result).toBe('error')
      expect(keys.value).toHaveLength(1)
    })

    it('coalesces concurrent deletes of the same key into one DELETE (RG3, anti double-fire)', async () => {
      let resolveDelete!: (value: { error: undefined }) => void
      mockDelete.mockReturnValue(
        new Promise((resolve) => {
          resolveDelete = resolve
        }),
      )

      const { deleteKey } = useAPIKeys()
      const first = deleteKey('key-1')
      // Second call while the first DELETE is still in flight must be coalesced.
      const second = deleteKey('key-1')

      expect(await second).toBe('busy')

      resolveDelete({ error: undefined })
      expect(await first).toBe('deleted')
      expect(mockDelete).toHaveBeenCalledTimes(1)
    })
  })
})
