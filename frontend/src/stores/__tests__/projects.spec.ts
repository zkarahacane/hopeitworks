import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjectsStore } from '../projects'

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

describe('useProjectsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPut.mockReset()
  })

  it('starts with default state', () => {
    const store = useProjectsStore()
    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.currentProject).toBeNull()
  })

  it('fetches projects successfully and populates items and pagination', async () => {
    const projects = [
      {
        id: '1',
        name: 'Project A',
        description: 'Desc A',
        owner_id: 'u1',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
      {
        id: '2',
        name: 'Project B',
        owner_id: 'u1',
        created_at: '2026-01-16T10:00:00Z',
        updated_at: '2026-01-16T10:00:00Z',
      },
    ]
    const pagination = { total: 2, page: 1, per_page: 20 }

    mockGet.mockResolvedValue({
      data: { data: projects, pagination },
      error: undefined,
    })

    const store = useProjectsStore()
    await store.fetchProjects({ page: 1, per_page: 20 })

    expect(store.items).toEqual(projects)
    expect(store.pagination).toEqual(pagination)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(mockGet).toHaveBeenCalledWith('/projects', {
      params: { query: { page: 1, per_page: 20, sort_by: undefined } },
    })
  })

  it('sets error state when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const store = useProjectsStore()
    await store.fetchProjects()

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBe('Failed to load projects')
    expect(store.isLoading).toBe(false)
  })

  it('sets error state when API call throws', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const store = useProjectsStore()
    await store.fetchProjects()

    expect(store.items).toEqual([])
    expect(store.error).toBe('Network error')
    expect(store.isLoading).toBe(false)
  })

  it('sets fallback error message for non-Error thrown values', async () => {
    mockGet.mockRejectedValue('unknown error')

    const store = useProjectsStore()
    await store.fetchProjects()

    expect(store.error).toBe('Failed to load projects')
  })

  it('resets all state', async () => {
    const projects = [
      {
        id: '1',
        name: 'Project A',
        owner_id: 'u1',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: projects, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useProjectsStore()
    await store.fetchProjects()

    expect(store.items).toHaveLength(1)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBeNull()
    expect(store.currentProject).toBeNull()
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

    const store = useProjectsStore()

    await store.fetchProjects()
    expect(store.error).toBe('Failed to load projects')

    await store.fetchProjects()
    expect(store.error).toBeNull()
  })

  describe('getProject', () => {
    it('fetches a single project and sets currentProject', async () => {
      mockGet.mockResolvedValue({
        data: mockProject,
        error: undefined,
      })

      const store = useProjectsStore()
      await store.getProject('p1')

      expect(store.currentProject).toEqual(mockProject)
      expect(store.isLoading).toBe(false)
      expect(store.error).toBeNull()
      expect(mockGet).toHaveBeenCalledWith('/projects/{id}', {
        params: { path: { id: 'p1' } },
      })
    })

    it('sets error when API returns an error', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
      })

      const store = useProjectsStore()
      await store.getProject('p1')

      expect(store.currentProject).toBeNull()
      expect(store.error).toBe('Failed to load project')
      expect(store.isLoading).toBe(false)
    })

    it('sets error when API call throws', async () => {
      mockGet.mockRejectedValue(new Error('Network failure'))

      const store = useProjectsStore()
      await store.getProject('p1')

      expect(store.currentProject).toBeNull()
      expect(store.error).toBe('Network failure')
      expect(store.isLoading).toBe(false)
    })

    it('sets fallback error for non-Error thrown values', async () => {
      mockGet.mockRejectedValue('something went wrong')

      const store = useProjectsStore()
      await store.getProject('p1')

      expect(store.error).toBe('Failed to load project')
    })
  })

  describe('updateProject', () => {
    it('updates a project and sets currentProject', async () => {
      const updatedProject = { ...mockProject, name: 'Updated Name' }
      mockPut.mockResolvedValue({
        data: updatedProject,
        error: undefined,
      })

      const store = useProjectsStore()
      const result = await store.updateProject('p1', { name: 'Updated Name' })

      expect(result).toEqual(updatedProject)
      expect(store.currentProject).toEqual(updatedProject)
      expect(mockPut).toHaveBeenCalledWith('/projects/{id}', {
        params: { path: { id: 'p1' } },
        body: { name: 'Updated Name' },
      })
    })

    it('updates project in items list when present', async () => {
      const updatedProject = { ...mockProject, name: 'Updated Name' }
      mockGet.mockResolvedValue({
        data: { data: [mockProject], pagination: { total: 1, page: 1, per_page: 20 } },
        error: undefined,
      })
      mockPut.mockResolvedValue({
        data: updatedProject,
        error: undefined,
      })

      const store = useProjectsStore()
      await store.fetchProjects()
      await store.updateProject('p1', { name: 'Updated Name' })

      expect(store.items[0]).toEqual(updatedProject)
    })

    it('throws when API returns an error', async () => {
      mockPut.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'BAD_REQUEST', message: 'Invalid' } },
      })

      const store = useProjectsStore()
      await expect(store.updateProject('p1', { name: '' })).rejects.toThrow(
        'Failed to update project',
      )
    })
  })

  describe('clearCurrentProject', () => {
    it('resets currentProject to null', async () => {
      mockGet.mockResolvedValue({
        data: mockProject,
        error: undefined,
      })

      const store = useProjectsStore()
      await store.getProject('p1')
      expect(store.currentProject).toEqual(mockProject)

      store.clearCurrentProject()
      expect(store.currentProject).toBeNull()
    })
  })
})
