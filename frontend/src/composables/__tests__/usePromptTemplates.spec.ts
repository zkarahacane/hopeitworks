import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePromptTemplates } from '../usePromptTemplates'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('usePromptTemplates', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('exposes reactive computed properties from the store', () => {
    const { templates, pagination, isLoading, error } = usePromptTemplates('p1')
    expect(templates.value).toEqual([])
    expect(pagination.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('fetches templates and updates reactive state', async () => {
    const templateData = [
      {
        id: 't1',
        project_id: 'p1',
        name: 'Implement Template',
        template_content: 'content',
        type: 'implement',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: templateData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { templates, pagination, isLoading, fetchTemplates } = usePromptTemplates('p1')
    await fetchTemplates({ page: 1, per_page: 20 })

    expect(templates.value).toEqual(templateData)
    expect(pagination.value).toEqual({ total: 1, page: 1, per_page: 20 })
    expect(isLoading.value).toBe(false)
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { error, fetchTemplates } = usePromptTemplates('p1')
    await fetchTemplates()

    expect(error.value).toBe('Failed to load templates')
  })

  it('retry re-fetches with the same params', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 2, per_page: 10 } },
      error: undefined,
    })

    const { fetchTemplates, retry } = usePromptTemplates('p1')
    await fetchTemplates({ page: 2, per_page: 10 })

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: {
        path: { projectId: 'p1' },
        query: { page: 2, per_page: 10 },
      },
    })

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: {
        path: { projectId: 'p1' },
        query: { page: 2, per_page: 10 },
      },
    })
  })

  it('retry uses default params when fetchTemplates was called without params', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { fetchTemplates, retry } = usePromptTemplates('p1')
    await fetchTemplates()
    mockGet.mockClear()

    await retry()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: {
        path: { projectId: 'p1' },
        query: { page: undefined, per_page: undefined },
      },
    })
  })

  it('passes projectId correctly to the store', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
      error: undefined,
    })

    const { fetchTemplates } = usePromptTemplates('project-123')
    await fetchTemplates()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: {
        path: { projectId: 'project-123' },
        query: { page: undefined, per_page: undefined },
      },
    })
  })
})
