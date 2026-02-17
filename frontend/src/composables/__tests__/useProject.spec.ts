import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useProject } from '../useProject'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('useProject', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockGet.mockResolvedValue({
      data: {
        id: 'p1',
        name: 'Test Project',
        description: 'A test project',
        owner_id: 'u1',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
      error: undefined,
    })
  })

  it('exposes reactive properties with initial values', () => {
    const { project, isLoading, error } = useProject('p1')
    expect(project.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('fetches project and updates reactive state', async () => {
    const { project, isLoading, fetchProject } = useProject('p1')
    await fetchProject()

    expect(project.value).toEqual({
      id: 'p1',
      name: 'Test Project',
      description: 'A test project',
      owner_id: 'u1',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    })
    expect(isLoading.value).toBe(false)
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}', {
      params: { path: { id: 'p1' } },
    })
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Project not found' } },
    })

    const { error, fetchProject } = useProject('p1')
    await fetchProject()

    expect(error.value).toBe('Failed to load project')
  })

  it('handles network errors', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const { error, fetchProject } = useProject('p1')
    await fetchProject()

    expect(error.value).toBe('Network error')
  })

  it('retry re-fetches with the same project ID', async () => {
    const { fetchProject, retry } = useProject('p1')
    await fetchProject()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}', {
      params: { path: { id: 'p1' } },
    })

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{id}', {
      params: { path: { id: 'p1' } },
    })
  })

  it('clears error on successful retry after failure', async () => {
    mockGet.mockResolvedValueOnce({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { project, error, fetchProject, retry } = useProject('p1')
    await fetchProject()

    expect(error.value).toBe('Failed to load project')

    mockGet.mockResolvedValueOnce({
      data: {
        id: 'p1',
        name: 'Test Project',
        owner_id: 'u1',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      },
      error: undefined,
    })

    await retry()

    expect(error.value).toBeNull()
    expect(project.value?.name).toBe('Test Project')
  })
})
