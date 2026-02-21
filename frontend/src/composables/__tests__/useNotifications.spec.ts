import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useNotifications } from '../useNotifications'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()
const mockDelete = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
    DELETE: (...args: unknown[]) => mockDelete(...args),
  },
}))

const mockConfig = {
  id: 'n1',
  project_id: 'p1',
  channel_type: 'discord' as const,
  config: { url: 'https://discord.com/api/webhooks/123/abc123' },
  events_filter: ['run.completed', 'run.failed'],
  enabled: true,
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useNotifications', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
    mockDelete.mockReset()
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })
  })

  it('exposes reactive properties with initial values', () => {
    const { configs, isLoading, error } = useNotifications('p1')
    expect(configs.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  describe('fetchConfigs', () => {
    it('populates configs.value from API response', async () => {
      mockGet.mockResolvedValue({
        data: { data: [mockConfig], pagination: { total: 1, page: 1, per_page: 20 } },
        error: undefined,
      })

      const { configs, isLoading, fetchConfigs } = useNotifications('p1')
      await fetchConfigs()

      expect(configs.value).toEqual([mockConfig])
      expect(isLoading.value).toBe(false)
      expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/notifications', {
        params: { path: { projectId: 'p1' } },
      })
    })

    it('sets error state on fetch failure', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      const { error, fetchConfigs } = useNotifications('p1')
      await fetchConfigs()

      expect(error.value).toBe('Failed to load channels')
    })

    it('sets isLoading to false after fetch completes', async () => {
      const { isLoading, fetchConfigs } = useNotifications('p1')
      await fetchConfigs()
      expect(isLoading.value).toBe(false)
    })
  })

  describe('createConfig', () => {
    it('pushes to configs.value on success and returns the new config', async () => {
      mockPost.mockResolvedValue({
        data: mockConfig,
        error: undefined,
      })

      const { configs, createConfig } = useNotifications('p1')
      const result = await createConfig({
        channel_type: 'discord',
        config: { url: 'https://discord.com/api/webhooks/123/abc123' },
        events_filter: ['run.completed'],
        enabled: true,
      })

      expect(result).toEqual(mockConfig)
      expect(configs.value).toContainEqual(mockConfig)
    })

    it('returns null on API error', async () => {
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'BAD_REQUEST', message: 'Invalid URL' } },
      })

      const { configs, createConfig } = useNotifications('p1')
      const result = await createConfig({
        channel_type: 'discord',
        config: { url: 'https://discord.com/api/webhooks/123/abc123' },
        events_filter: [],
        enabled: true,
      })

      expect(result).toBeNull()
      expect(configs.value).toHaveLength(0)
    })
  })

  describe('toggleEnabled', () => {
    it('optimistically updates config.enabled', async () => {
      mockGet.mockResolvedValue({
        data: { data: [mockConfig], pagination: { total: 1, page: 1, per_page: 20 } },
        error: undefined,
      })
      mockPut.mockResolvedValue({ error: undefined })

      const { configs, fetchConfigs, toggleEnabled } = useNotifications('p1')
      await fetchConfigs()

      expect(configs.value).toHaveLength(1)
      const first = configs.value[0]!
      expect(first.enabled).toBe(true)

      await toggleEnabled(first)

      expect(configs.value[0]!.enabled).toBe(false)
      expect(mockPut).toHaveBeenCalledWith('/projects/{projectId}/notifications/{notificationId}', {
        params: { path: { projectId: 'p1', notificationId: 'n1' } },
        body: {
          channel_type: 'discord',
          config: { url: 'https://discord.com/api/webhooks/123/abc123' },
          events_filter: ['run.completed', 'run.failed'],
          enabled: false,
        },
      })
    })

    it('reverts optimistic update on API error', async () => {
      mockGet.mockResolvedValue({
        data: { data: [mockConfig], pagination: { total: 1, page: 1, per_page: 20 } },
        error: undefined,
      })
      mockPut.mockResolvedValue({
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      const { configs, fetchConfigs, toggleEnabled } = useNotifications('p1')
      await fetchConfigs()

      expect(configs.value).toHaveLength(1)
      const first = configs.value[0]!
      expect(first.enabled).toBe(true)

      await expect(toggleEnabled(first)).rejects.toBeDefined()

      expect(configs.value[0]!.enabled).toBe(true)
    })
  })

  describe('deleteConfig', () => {
    it('removes item from configs.value on success', async () => {
      mockGet.mockResolvedValue({
        data: { data: [mockConfig], pagination: { total: 1, page: 1, per_page: 20 } },
        error: undefined,
      })
      mockDelete.mockResolvedValue({ error: undefined })

      const { configs, fetchConfigs, deleteConfig } = useNotifications('p1')
      await fetchConfigs()

      expect(configs.value).toHaveLength(1)

      await deleteConfig('n1')

      expect(configs.value).toHaveLength(0)
      expect(mockDelete).toHaveBeenCalledWith(
        '/projects/{projectId}/notifications/{notificationId}',
        {
          params: { path: { projectId: 'p1', notificationId: 'n1' } },
        },
      )
    })
  })

  describe('testConfig', () => {
    it('resolves on success', async () => {
      mockPost.mockResolvedValue({ error: undefined })

      const { testConfig } = useNotifications('p1')

      await expect(testConfig('n1')).resolves.toBeUndefined()
      expect(mockPost).toHaveBeenCalledWith(
        '/projects/{projectId}/notifications/{notificationId}/test',
        {
          params: { path: { projectId: 'p1', notificationId: 'n1' } },
        },
      )
    })

    it('throws on API error', async () => {
      const apiErr = { error: { code: 'INTERNAL', message: 'Webhook failed' } }
      mockPost.mockResolvedValue({ error: apiErr })

      const { testConfig } = useNotifications('p1')

      await expect(testConfig('n1')).rejects.toBeDefined()
    })
  })
})
