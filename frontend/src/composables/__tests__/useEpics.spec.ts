import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEpics } from '../useEpics'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('useEpics', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('exposes reactive computed properties from the store', () => {
    const { epics, pagination, isLoading, error } = useEpics()
    expect(epics.value).toEqual([])
    expect(pagination.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('fetches epics and updates reactive state', async () => {
    const epicData = [
      {
        id: 'e1',
        project_id: 'p1',
        name: 'Epic 1',
        status: 'open',
        story_counts: { total: 5, backlog: 5, running: 0, done: 0, failed: 0 },
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: epicData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { epics, pagination, isLoading, fetchEpics } = useEpics()
    await fetchEpics('p1', { page: 1, per_page: 20 })

    expect(epics.value).toEqual(epicData)
    expect(pagination.value).toEqual({ total: 1, page: 1, per_page: 20 })
    expect(isLoading.value).toBe(false)
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { error, fetchEpics } = useEpics()
    await fetchEpics('p1')

    expect(error.value).toBe('Failed to load epics')
  })

  it('retry re-fetches with the same project ID and params', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 2, per_page: 10 } },
      error: undefined,
    })

    const { fetchEpics, retry } = useEpics()
    await fetchEpics('p1', { page: 2, per_page: 10 })

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}/epics', {
      params: {
        path: { id: 'p1' },
        query: { page: 2, per_page: 10, sort_by: undefined },
      },
    })

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}/epics', {
      params: {
        path: { id: 'p1' },
        query: { page: 2, per_page: 10, sort_by: undefined },
      },
    })
  })

  it('retry does nothing when no previous fetch was made', async () => {
    const { retry } = useEpics()
    await retry()

    expect(mockGet).not.toHaveBeenCalled()
  })
})
