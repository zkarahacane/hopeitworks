import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProjectsStore } from '../projects'

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

describe('useProjectsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
    mockDelete.mockReset()
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

  it('createProject passes new fields (repo_url, git_provider, agent_runtime, default_model) to API', async () => {
    const createdProject = {
      id: 'p2',
      name: 'Full Project',
      description: 'Desc',
      repo_url: 'https://github.com/org/repo',
      git_provider: 'github',
      agent_runtime: 'docker',
      default_model: 'claude-opus-4-5',
      owner_id: 'u1',
      created_at: '2026-02-22T10:00:00Z',
      updated_at: '2026-02-22T10:00:00Z',
    }

    mockPost.mockResolvedValue({ data: createdProject, error: undefined })

    const store = useProjectsStore()
    const result = await store.createProject({
      name: 'Full Project',
      description: 'Desc',
      repo_url: 'https://github.com/org/repo',
      git_provider: 'github',
      agent_runtime: 'docker',
      default_model: 'claude-opus-4-5',
    })

    expect(result).toEqual(createdProject)
    expect(mockPost).toHaveBeenCalledWith('/projects', {
      body: {
        name: 'Full Project',
        description: 'Desc',
        repo_url: 'https://github.com/org/repo',
        git_provider: 'github',
        agent_runtime: 'docker',
        default_model: 'claude-opus-4-5',
      },
    })
  })

  it('updateProject returns updated project on success', async () => {
    const updatedProject = {
      id: 'p1',
      name: 'Updated Project',
      description: 'Updated desc',
      repo_url: 'https://github.com/org/updated-repo',
      git_provider: 'github',
      agent_runtime: 'docker',
      default_model: 'claude-opus-4-5',
      owner_id: 'u1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-22T12:00:00Z',
    }

    mockPut.mockResolvedValue({ data: updatedProject, error: undefined })

    const store = useProjectsStore()
    const result = await store.updateProject('p1', {
      name: 'Updated Project',
      description: 'Updated desc',
      repo_url: 'https://github.com/org/updated-repo',
      git_provider: 'github',
      agent_runtime: 'docker',
      default_model: 'claude-opus-4-5',
    })

    expect(result).toEqual(updatedProject)
    expect(mockPut).toHaveBeenCalledWith('/projects/{id}', {
      params: { path: { id: 'p1' } },
      body: {
        name: 'Updated Project',
        description: 'Updated desc',
        repo_url: 'https://github.com/org/updated-repo',
        git_provider: 'github',
        agent_runtime: 'docker',
        default_model: 'claude-opus-4-5',
      },
    })
  })

  it('updateProject throws with API error message', async () => {
    mockPut.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Project not found' } },
    })

    const store = useProjectsStore()
    await expect(store.updateProject('p999', { name: 'X' })).rejects.toThrow('Project not found')
  })

  it('updateProject throws fallback message when API error has no message', async () => {
    mockPut.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL' } },
    })

    const store = useProjectsStore()
    await expect(store.updateProject('p1', { name: 'X' })).rejects.toThrow(
      'Failed to update project',
    )
  })

  // RG1: deleteProject calls DELETE /projects/{id} and resolves on 204
  it('deleteProject calls DELETE with the project id and resolves on success', async () => {
    mockDelete.mockResolvedValue({ data: undefined, error: undefined })

    const store = useProjectsStore()
    await expect(store.deleteProject('p1')).resolves.toBeUndefined()
    expect(mockDelete).toHaveBeenCalledWith('/projects/{id}', {
      params: { path: { id: 'p1' } },
    })
  })

  it('deleteProject removes the project from local state when present', async () => {
    mockDelete.mockResolvedValue({ data: undefined, error: undefined })

    const store = useProjectsStore()
    store.items = [
      { id: 'p1', name: 'A', owner_id: 'u1', created_at: 'x', updated_at: 'x' },
      { id: 'p2', name: 'B', owner_id: 'u1', created_at: 'x', updated_at: 'x' },
    ]
    await store.deleteProject('p1')
    expect(store.items.map((p) => p.id)).toEqual(['p2'])
  })

  // RG5: error path — throws and leaves state untouched (project preserved)
  it('deleteProject throws with API error message and keeps state on failure', async () => {
    mockDelete.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Project not found' } },
    })

    const store = useProjectsStore()
    store.items = [{ id: 'p1', name: 'A', owner_id: 'u1', created_at: 'x', updated_at: 'x' }]
    await expect(store.deleteProject('p1')).rejects.toThrow('Project not found')
    expect(store.items.map((p) => p.id)).toEqual(['p1'])
  })

  it('deleteProject throws fallback message when API error has no message', async () => {
    mockDelete.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL' } },
    })

    const store = useProjectsStore()
    await expect(store.deleteProject('p1')).rejects.toThrow('Failed to delete project')
  })
})
