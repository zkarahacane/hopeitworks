import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePromptTemplatesStore } from '../promptTemplates'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('usePromptTemplatesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = usePromptTemplatesStore()
    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches templates successfully and populates items and pagination', async () => {
    const templates = [
      {
        id: 't1',
        project_id: 'p1',
        name: 'Implement Template',
        template_content: 'You are a developer...',
        type: 'implement',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
      {
        id: 't2',
        project_id: 'p1',
        name: 'Review Template',
        template_content: 'You are a code reviewer...',
        type: 'review',
        created_at: '2026-01-16T10:00:00Z',
        updated_at: '2026-01-16T10:00:00Z',
      },
    ]
    const pagination = { total: 2, page: 1, per_page: 20 }

    mockGet.mockResolvedValue({
      data: { data: templates, pagination },
      error: undefined,
    })

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1', { page: 1, per_page: 20 })

    expect(store.items).toEqual(templates)
    expect(store.pagination).toEqual(pagination)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: {
        path: { projectId: 'p1' },
        query: { page: 1, per_page: 20 },
      },
    })
  })

  it('sets error state when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1')

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBe('Failed to load templates')
    expect(store.isLoading).toBe(false)
  })

  it('sets error state when API call throws', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1')

    expect(store.items).toEqual([])
    expect(store.error).toBe('Network error')
    expect(store.isLoading).toBe(false)
  })

  it('sets fallback error message for non-Error thrown values', async () => {
    mockGet.mockRejectedValue('unknown error')

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1')

    expect(store.error).toBe('Failed to load templates')
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

    const store = usePromptTemplatesStore()

    await store.fetchTemplates('p1')
    expect(store.error).toBe('Failed to load templates')

    await store.fetchTemplates('p1')
    expect(store.error).toBeNull()
  })

  it('clearError resets error state', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'fail' } },
    })

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1')
    expect(store.error).toBe('Failed to load templates')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('resets all state', async () => {
    const templates = [
      {
        id: 't1',
        project_id: 'p1',
        name: 'Template A',
        template_content: 'content',
        type: 'implement',
        created_at: '2026-01-15T10:00:00Z',
        updated_at: '2026-01-15T10:00:00Z',
      },
    ]
    mockGet.mockResolvedValue({
      data: { data: templates, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = usePromptTemplatesStore()
    await store.fetchTemplates('p1')

    expect(store.items).toHaveLength(1)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.pagination).toBeNull()
    expect(store.error).toBeNull()
  })
})
