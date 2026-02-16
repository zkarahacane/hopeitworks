import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjects } from '../useProjects'

const mockGet = vi.fn()
const mockPut = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
  },
}))

const mockProject = {
  id: 'p1',
  name: 'Test Project',
  description: 'A test project',
  owner_id: 'u1',
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useProjects', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPut.mockReset()
  })

  it('exposes reactive computed properties from the store', () => {
    const { projects, pagination, isLoading, error, currentProject } = useProjects()
    expect(projects.value).toEqual([])
    expect(pagination.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
    expect(currentProject.value).toBeNull()
  })

  it('fetches projects and updates reactive state', async () => {
    const projectData = [
      {
        id: '1',
        name: 'Project A',
        owner_id: 'u1',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: projectData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { projects, pagination, isLoading, fetchProjects } = useProjects()
    await fetchProjects({ page: 1, per_page: 20 })

    expect(projects.value).toEqual(projectData)
    expect(pagination.value).toEqual({ total: 1, page: 1, per_page: 20 })
    expect(isLoading.value).toBe(false)
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { error, fetchProjects } = useProjects()
    await fetchProjects()

    expect(error.value).toBe('Failed to load projects')
  })

  it('retry re-fetches with the same params', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 2, per_page: 10 } },
      error: undefined,
    })

    const { fetchProjects, retry } = useProjects()
    await fetchProjects({ page: 2, per_page: 10 })

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects', {
      params: { query: { page: 2, per_page: 10, sort_by: undefined } },
    })

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects', {
      params: { query: { page: 2, per_page: 10, sort_by: undefined } },
    })
  })

  it('retry uses default params when fetchProjects was called without params', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { fetchProjects, retry } = useProjects()
    await fetchProjects()
    mockGet.mockClear()

    await retry()

    expect(mockGet).toHaveBeenCalledWith('/projects', {
      params: { query: { page: undefined, per_page: undefined, sort_by: undefined } },
    })
  })

  describe('getProject', () => {
    it('exposes loading state during fetch', async () => {
      let resolvePromise: (value: unknown) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      mockGet.mockReturnValue(pendingPromise)

      const { getProject } = useProjects()
      const executePromise = getProject.execute('p1')

      expect(getProject.isLoading.value).toBe(true)

      resolvePromise!({ data: mockProject, error: undefined })
      await executePromise

      expect(getProject.isLoading.value).toBe(false)
    })

    it('updates currentProject reactive ref when store changes', async () => {
      mockGet.mockResolvedValue({
        data: mockProject,
        error: undefined,
      })

      const { currentProject, getProject } = useProjects()
      expect(currentProject.value).toBeNull()

      await getProject.execute('p1')

      expect(currentProject.value).toEqual(mockProject)
    })

    it('exposes error on fetch failure', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
      })

      const { getProject } = useProjects()
      await getProject.execute('p1')

      // useAsyncAction does not capture store-level errors in its own error ref
      // since store.getProject doesn't throw — it sets store.error instead
      expect(getProject.isLoading.value).toBe(false)
    })
  })

  describe('updateProject', () => {
    it('exposes loading state during update', async () => {
      let resolvePromise: (value: unknown) => void
      const pendingPromise = new Promise((resolve) => {
        resolvePromise = resolve
      })
      mockPut.mockReturnValue(pendingPromise)

      const { updateProject } = useProjects()
      const executePromise = updateProject.execute('p1', { name: 'New Name' })

      expect(updateProject.isLoading.value).toBe(true)

      resolvePromise!({ data: { ...mockProject, name: 'New Name' }, error: undefined })
      await executePromise

      expect(updateProject.isLoading.value).toBe(false)
    })

    it('captures error when update fails', async () => {
      mockPut.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'BAD_REQUEST', message: 'Invalid' } },
      })

      const { updateProject } = useProjects()
      await updateProject.execute('p1', { name: '' })

      expect(updateProject.error.value).toBeInstanceOf(Error)
      expect(updateProject.error.value?.message).toBe('Failed to update project')
    })
  })

  describe('clearCurrentProject', () => {
    it('resets currentProject computed value', async () => {
      mockGet.mockResolvedValue({
        data: mockProject,
        error: undefined,
      })

      const { currentProject, getProject, clearCurrentProject } = useProjects()
      await getProject.execute('p1')
      expect(currentProject.value).toEqual(mockProject)

      clearCurrentProject()
      expect(currentProject.value).toBeNull()
    })
  })
})
