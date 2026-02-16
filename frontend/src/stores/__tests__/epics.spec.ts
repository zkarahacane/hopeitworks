import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEpicsStore } from '../epics'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

const mockEpic = {
  id: 'e1',
  project_id: 'p1',
  name: 'Epic 1',
  description: 'First epic',
  status: 'in_progress',
  story_counts: { total: 12, backlog: 5, running: 3, done: 3, failed: 1 },
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useEpicsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useEpicsStore()
    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.projectId).toBeNull()
  })

  it('fetches epics successfully and populates items and pagination', async () => {
    const epics = [mockEpic]
    const pagination = { total: 1, page: 1, per_page: 20 }

    mockGet.mockResolvedValue({
      data: { data: epics, pagination },
      error: undefined,
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1', { page: 1, per_page: 20 })

    expect(store.items).toEqual(epics)
    expect(store.pagination).toEqual(pagination)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.projectId).toBe('p1')
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}/epics', {
      params: {
        path: { id: 'p1' },
        query: { page: 1, per_page: 20, sort_by: undefined },
      },
    })
  })

  it('sets error state when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBe('Failed to load epics')
    expect(store.isLoading).toBe(false)
  })

  it('sets error state when API call throws', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.items).toEqual([])
    expect(store.error).toBe('Network error')
    expect(store.isLoading).toBe(false)
  })

  it('sets fallback error message for non-Error thrown values', async () => {
    mockGet.mockRejectedValue('unknown error')

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.error).toBe('Failed to load epics')
  })

  it('resets all state', async () => {
    mockGet.mockResolvedValue({
      data: { data: [mockEpic], pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.items).toHaveLength(1)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBeNull()
    expect(store.projectId).toBeNull()
  })

  it('clears previous error on new fetch', async () => {
    mockGet
      .mockResolvedValueOnce({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'fail' } },
      })
      .mockResolvedValueOnce({
        data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
        error: undefined,
      })

    const store = useEpicsStore()

    await store.fetchEpics('p1')
    expect(store.error).toBe('Failed to load epics')

    await store.fetchEpics('p1')
    expect(store.error).toBeNull()
  })
})
