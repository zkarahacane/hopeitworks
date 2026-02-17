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
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })
  })

  it('exposes reactive computed properties from the store', () => {
    const { epics, isLoading, error } = useEpics('p1')
    expect(epics.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('fetches epics and updates reactive state', async () => {
    const epicData = [
      {
        id: 'e1',
        project_id: 'p1',
        title: 'Epic A',
        status: 'backlog',
        story_counts: { backlog: 2, running: 0, done: 0, failed: 0 },
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: epicData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { epics, isLoading, fetchEpics } = useEpics('p1')
    await fetchEpics()

    expect(epics.value).toEqual(epicData)
    expect(isLoading.value).toBe(false)
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { error, fetchEpics } = useEpics('p1')
    await fetchEpics()

    expect(error.value).toBe('Failed to load epics')
  })

  it('retry re-fetches with the same project ID', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { fetchEpics, retry } = useEpics('p1')
    await fetchEpics()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/epics', {
      params: { path: { projectId: 'p1' } },
    })

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/epics', {
      params: { path: { projectId: 'p1' } },
    })
  })
})
