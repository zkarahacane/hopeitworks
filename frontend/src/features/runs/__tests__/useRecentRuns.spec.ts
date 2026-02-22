import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useRecentRuns } from '../composables/useRecentRuns'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

/** Wraps composable that calls onMounted in a simulated lifecycle */
function withSetup<T>(composable: () => T): T {
  let result!: T
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { createApp, defineComponent } = require('vue')
  const app = createApp(
    defineComponent({
      setup() {
        result = composable()
        return () => null
      },
    }),
  )
  app.mount(document.createElement('div'))
  return result
}

const mockRuns = [
  {
    id: 'run-1',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: 'running',
    progress: 50,
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T10:00:00Z',
  },
  {
    id: 'run-2',
    project_id: 'proj-1',
    story_id: 'story-2',
    status: 'completed',
    progress: 100,
    created_at: '2026-02-16T09:00:00Z',
    updated_at: '2026-02-16T09:30:00Z',
  },
]

const mockProjects = [
  { id: 'proj-1', name: 'Project Alpha' },
  { id: 'proj-2', name: 'Project Beta' },
]

describe('useRecentRuns', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  it('fetches runs for a specific project when projectId is provided', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockRuns, pagination: { total: 2, page: 1, per_page: 10 } },
      error: undefined,
    })

    const { runs, isLoading } = withSetup(() => useRecentRuns({ projectId: 'proj-1' }))
    await flushPromises()

    expect(runs.value).toHaveLength(2)
    expect(runs.value[0]?.id).toBe('run-1')
    expect(isLoading.value).toBe(false)
    expect(mockGet).toHaveBeenCalledTimes(1)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/runs', {
      params: { path: { projectId: 'proj-1' }, query: { per_page: 10, page: 1 } },
    })
  })

  it('fetches projects then fans out per-project when no projectId is provided', async () => {
    // First call: GET /projects
    mockGet.mockImplementation((path: string) => {
      if (path === '/projects') {
        return Promise.resolve({
          data: { data: mockProjects, pagination: { total: 2, page: 1, per_page: 5 } },
          error: undefined,
        })
      }
      // Per-project calls
      return Promise.resolve({
        data: { data: mockRuns.slice(0, 1), pagination: { total: 1, page: 1, per_page: 10 } },
        error: undefined,
      })
    })

    const { runs } = withSetup(() => useRecentRuns())
    await flushPromises()

    // Should have called GET /projects + GET /projects/{projectId}/runs for each project
    expect(mockGet).toHaveBeenCalledWith('/projects', {
      params: { query: { per_page: 5, page: 1 } },
    })
    // 1 projects call + 2 per-project runs calls = 3 total
    expect(mockGet).toHaveBeenCalledTimes(3)
    expect(runs.value.length).toBeGreaterThan(0)
  })

  it('sets isLoading true during fetch and false after', async () => {
    let resolvePromise: (value: unknown) => void
    mockGet.mockReturnValue(
      new Promise((resolve) => {
        resolvePromise = resolve
      }),
    )

    const { isLoading } = withSetup(() => useRecentRuns({ projectId: 'proj-1' }))
    expect(isLoading.value).toBe(true)

    resolvePromise!({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 10 } },
      error: undefined,
    })
    await flushPromises()

    expect(isLoading.value).toBe(false)
  })

  it('populates runs on success', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockRuns, pagination: { total: 2, page: 1, per_page: 10 } },
      error: undefined,
    })

    const { runs } = withSetup(() => useRecentRuns({ projectId: 'proj-1' }))
    await flushPromises()

    expect(runs.value).toHaveLength(2)
    expect(runs.value[0]).toMatchObject({ id: 'run-1', status: 'running' })
    expect(runs.value[1]).toMatchObject({ id: 'run-2', status: 'completed' })
  })

  it('sets error on API failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'SERVER_ERROR', message: 'Internal error' } },
    })

    const { runs, error } = withSetup(() => useRecentRuns({ projectId: 'proj-1' }))
    await flushPromises()

    expect(runs.value).toHaveLength(0)
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load runs')
  })

  it('returns empty runs when no projects exist (global mode)', async () => {
    mockGet.mockResolvedValue({
      data: { data: [], pagination: { total: 0, page: 1, per_page: 5 } },
      error: undefined,
    })

    const { runs } = withSetup(() => useRecentRuns())
    await flushPromises()

    expect(runs.value).toHaveLength(0)
  })

  it('refresh() re-fetches runs', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockRuns, pagination: { total: 2, page: 1, per_page: 10 } },
      error: undefined,
    })

    const { refresh } = withSetup(() => useRecentRuns({ projectId: 'proj-1' }))
    await flushPromises()

    expect(mockGet).toHaveBeenCalledTimes(1)

    mockGet.mockClear()
    mockGet.mockResolvedValue({
      data: { data: mockRuns, pagination: { total: 2, page: 1, per_page: 10 } },
      error: undefined,
    })

    await refresh()
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('respects custom limit parameter', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockRuns, pagination: { total: 2, page: 1, per_page: 5 } },
      error: undefined,
    })

    withSetup(() => useRecentRuns({ projectId: 'proj-1', limit: 5 }))
    await flushPromises()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/runs', {
      params: { path: { projectId: 'proj-1' }, query: { per_page: 5, page: 1 } },
    })
  })
})
