import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjectsStore } from '../projects'

const mockGet = vi.fn()
const mockPost = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
  },
}))

describe('useProjectsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
  })

  it('starts with default state', () => {
    const store = useProjectsStore()
    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
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

  it('createProject returns project on success', async () => {
    const createdProject = {
      id: 'p1',
      name: 'New Project',
      description: 'A description',
      owner_id: 'u1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    }

    mockPost.mockResolvedValue({
      data: createdProject,
      error: undefined,
    })

    const store = useProjectsStore()
    const result = await store.createProject({ name: 'New Project', description: 'A description' })

    expect(result).toEqual(createdProject)
    expect(mockPost).toHaveBeenCalledWith('/projects', {
      body: { name: 'New Project', description: 'A description' },
    })
  })

  it('createProject throws with API error message', async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'BAD_REQUEST', message: 'Name already exists' } },
    })

    const store = useProjectsStore()
    await expect(store.createProject({ name: 'Dup' })).rejects.toThrow('Name already exists')
  })

  it('createProject throws fallback message when API error has no message', async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL' } },
    })

    const store = useProjectsStore()
    await expect(store.createProject({ name: 'Test' })).rejects.toThrow(
      'Failed to create project',
    )
  })

  it('createProject propagates network errors', async () => {
    mockPost.mockRejectedValue(new Error('Network failure'))

    const store = useProjectsStore()
    await expect(store.createProject({ name: 'Test' })).rejects.toThrow('Network failure')
  })
})
