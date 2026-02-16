import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjects } from '../useProjects'

const mockGet = vi.fn()
const mockPost = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
  },
}))

describe('useProjects', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
  })

  it('exposes reactive computed properties from the store', () => {
    const { projects, pagination, isLoading, error } = useProjects()
    expect(projects.value).toEqual([])
    expect(pagination.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
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

  it('exposes createProject with execute, isLoading, and error', () => {
    const { createProject } = useProjects()
    expect(createProject.execute).toBeTypeOf('function')
    expect(createProject.isLoading.value).toBe(false)
    expect(createProject.error.value).toBeNull()
  })

  it('createProject.execute returns project on success', async () => {
    const createdProject = {
      id: 'p1',
      name: 'New Project',
      owner_id: 'u1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    }

    mockPost.mockResolvedValue({
      data: createdProject,
      error: undefined,
    })

    const { createProject } = useProjects()
    const result = await createProject.execute({ name: 'New Project' })

    expect(result).toEqual(createdProject)
    expect(createProject.isLoading.value).toBe(false)
    expect(createProject.error.value).toBeNull()
  })

  it('createProject.execute sets error on failure and returns null', async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'BAD_REQUEST', message: 'Name already exists' } },
    })

    const { createProject } = useProjects()
    const result = await createProject.execute({ name: 'Dup' })

    expect(result).toBeNull()
    expect(createProject.isLoading.value).toBe(false)
    expect(createProject.error.value).toBeInstanceOf(Error)
    expect(createProject.error.value?.message).toBe('Name already exists')
  })
})
