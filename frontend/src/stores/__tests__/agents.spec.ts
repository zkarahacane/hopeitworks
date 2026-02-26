import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAgentsStore } from '../agents'

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

const sampleAgent = {
  id: 'a1',
  name: 'Implement Agent',
  model: 'claude-opus-4-6',
  image: 'ghcr.io/org/agent:latest',
  template_content: 'You are a developer...',
  scope: 'project' as const,
  project_id: 'p1',
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useAgentsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
    mockDelete.mockReset()
  })

  it('starts with default state', () => {
    const store = useAgentsStore()
    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches agents successfully and populates items and pagination', async () => {
    const agents = [sampleAgent]
    const pagination = { total: 1, page: 1, per_page: 20 }

    mockGet.mockResolvedValue({
      data: { data: agents, pagination },
      error: undefined,
    })

    const store = useAgentsStore()
    await store.fetchAgents('p1', { page: 1, per_page: 20 })

    expect(store.items).toEqual(agents)
    expect(store.pagination).toEqual(pagination)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/agents', {
      params: {
        path: { projectId: 'p1' },
        query: { page: 1, per_page: 20 },
      },
    })
  })

  it('sets error state when fetchAgents API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const store = useAgentsStore()
    await store.fetchAgents('p1')

    expect(store.items).toEqual([])
    expect(store.error).toBe('Failed to load agents')
  })

  it('sets error state when fetchAgents throws', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const store = useAgentsStore()
    await store.fetchAgents('p1')

    expect(store.error).toBe('Network error')
  })

  it('sets fallback error message for non-Error thrown values', async () => {
    mockGet.mockRejectedValue('unknown error')

    const store = useAgentsStore()
    await store.fetchAgents('p1')

    expect(store.error).toBe('Failed to load agents')
  })

  it('creates an agent successfully', async () => {
    mockPost.mockResolvedValue({
      data: sampleAgent,
      error: undefined,
    })

    const store = useAgentsStore()
    const result = await store.createAgent('p1', {
      name: 'Implement Agent',
      model: 'claude-opus-4-6',
      image: 'ghcr.io/org/agent:latest',
      template_content: 'You are a developer...',
    })

    expect(result).toEqual(sampleAgent)
    expect(store.error).toBeNull()
    expect(mockPost).toHaveBeenCalledWith('/projects/{projectId}/agents', {
      params: { path: { projectId: 'p1' } },
      body: {
        name: 'Implement Agent',
        model: 'claude-opus-4-6',
        image: 'ghcr.io/org/agent:latest',
        template_content: 'You are a developer...',
        scope: 'project',
        provider: 'claude',
      },
    })
  })

  it('returns null when createAgent fails', async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'VALIDATION', message: 'Invalid' } },
    })

    const store = useAgentsStore()
    const result = await store.createAgent('p1', {
      name: 'Test',
      model: 'claude-opus-4-6',
      image: 'img',
      template_content: 'content',
    })

    expect(result).toBeNull()
    expect(store.error).toBe('Failed to create agent')
  })

  it('updates an agent successfully', async () => {
    const updated = { ...sampleAgent, name: 'Updated Agent' }
    mockPut.mockResolvedValue({
      data: updated,
      error: undefined,
    })

    const store = useAgentsStore()
    const result = await store.updateAgent('p1', 'a1', { name: 'Updated Agent' })

    expect(result).toEqual(updated)
    expect(store.error).toBeNull()
  })

  it('returns null when updateAgent fails', async () => {
    mockPut.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
    })

    const store = useAgentsStore()
    const result = await store.updateAgent('p1', 'a1', { name: 'Updated' })

    expect(result).toBeNull()
    expect(store.error).toBe('Failed to update agent')
  })

  it('deletes an agent successfully and removes from items', async () => {
    mockGet.mockResolvedValue({
      data: { data: [sampleAgent], pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })
    mockDelete.mockResolvedValue({ error: undefined })

    const store = useAgentsStore()
    await store.fetchAgents('p1')
    expect(store.items).toHaveLength(1)

    const result = await store.deleteAgent('p1', 'a1')

    expect(result).toBe(true)
    expect(store.items).toHaveLength(0)
    expect(store.error).toBeNull()
  })

  it('returns false when deleteAgent fails', async () => {
    mockDelete.mockResolvedValue({
      error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
    })

    const store = useAgentsStore()
    const result = await store.deleteAgent('p1', 'a1')

    expect(result).toBe(false)
    expect(store.error).toBe('Failed to delete agent')
  })

  it('clearError resets error state', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'fail' } },
    })

    const store = useAgentsStore()
    await store.fetchAgents('p1')
    expect(store.error).toBe('Failed to load agents')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('resets all state', async () => {
    mockGet.mockResolvedValue({
      data: { data: [sampleAgent], pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useAgentsStore()
    await store.fetchAgents('p1')
    expect(store.items).toHaveLength(1)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBeNull()
  })
})
