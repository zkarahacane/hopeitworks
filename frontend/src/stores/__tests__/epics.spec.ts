import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEpicsStore } from '../epics'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('useEpicsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useEpicsStore()
    expect(store.items).toEqual([])
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches epics successfully and populates items', async () => {
    const epics = [
      {
        id: 'e1',
        project_id: 'p1',
        title: 'Epic A',
        description: 'Desc A',
        status: 'in_progress',
        story_counts: { backlog: 3, running: 1, done: 5, failed: 0 },
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
      {
        id: 'e2',
        project_id: 'p1',
        title: 'Epic B',
        status: 'backlog',
        story_counts: { backlog: 2, running: 0, done: 0, failed: 0 },
        created_at: '2026-01-16T10:00:00Z',
        updated_at: '2026-01-16T10:00:00Z',
      },
    ]

    mockGet.mockResolvedValue({
      data: { data: epics, pagination: { total: 2, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.items).toEqual(epics)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/epics', {
      params: { path: { id: 'p1' } },
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

  it('clearError resets error state', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'fail' } },
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1')
    expect(store.error).toBe('Failed to load epics')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('resets all state', async () => {
    const epics = [
      {
        id: 'e1',
        project_id: 'p1',
        title: 'Epic A',
        status: 'backlog',
        story_counts: { backlog: 1, running: 0, done: 0, failed: 0 },
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: epics, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useEpicsStore()
    await store.fetchEpics('p1')

    expect(store.items).toHaveLength(1)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.error).toBeNull()
    expect(store.isLoading).toBe(false)
  })
})
