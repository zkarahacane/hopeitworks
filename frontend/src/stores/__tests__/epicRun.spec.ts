import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEpicRunStore } from '../epicRun'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

function makeEpicRun() {
  return {
    id: 'run-1',
    epic_id: 'epic-1',
    project_id: 'proj-1',
    status: 'running' as const,
    stories: [
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'completed' as const },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'running' as const },
      { story_id: 's3', story_key: 'S-03', run_id: null, group_index: 1, status: 'pending' as const },
    ],
    created_at: '2026-02-15T10:00:00Z',
    completed_at: null,
  }
}

describe('useEpicRunStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useEpicRunStore()
    expect(store.epicRun).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches epic run successfully', async () => {
    const epicRun = makeEpicRun()
    mockGet.mockResolvedValue({ data: epicRun, error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.epicRun).toEqual(epicRun)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/epic-runs/{epicRunId}', {
      params: { path: { projectId: 'proj-1', epicRunId: 'run-1' } },
    })
  })

  it('sets error when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
    })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.epicRun).toBeNull()
    expect(store.error).toBe('Failed to load epic run')
  })

  it('sets error when API call throws', async () => {
    mockGet.mockRejectedValue(new Error('Network failure'))

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.error).toBe('Network failure')
  })

  it('computes completedCount correctly', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.completedCount).toBe(1)
  })

  it('computes totalCount correctly', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.totalCount).toBe(3)
  })

  it('computes progressPercent correctly', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.progressPercent).toBe(33)
  })

  it('computes failedStories correctly', async () => {
    const epicRun = makeEpicRun()
    ;(epicRun.stories[1] as { status: string }).status = 'failed'
    mockGet.mockResolvedValue({ data: epicRun, error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.failedStories).toHaveLength(1)
    expect(store.failedStories[0]!.story_key).toBe('S-02')
  })

  it('handleSSEEvent updates story status on epic_run.story.completed', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    store.handleSSEEvent('epic_run.story.completed', { story_id: 's2' })

    const story = store.epicRun!.stories.find((s) => s.story_id === 's2')
    expect(story!.status).toBe('completed')
  })

  it('handleSSEEvent updates epic run status on epic_run.failed', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    store.handleSSEEvent('epic_run.failed', {})

    expect(store.epicRun!.status).toBe('failed')
  })

  it('handleSSEEvent updates epic run status on epic_run.completed', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    store.handleSSEEvent('epic_run.completed', {})

    expect(store.epicRun!.status).toBe('completed')
  })

  it('handleSSEEvent does nothing when epicRun is null', () => {
    const store = useEpicRunStore()
    // Should not throw
    store.handleSSEEvent('epic_run.story.completed', { story_id: 's1' })
    expect(store.epicRun).toBeNull()
  })

  it('progressPercent returns 0 when no stories', () => {
    const store = useEpicRunStore()
    expect(store.progressPercent).toBe(0)
  })

  it('reset clears all state', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const store = useEpicRunStore()
    await store.fetchEpicRun('proj-1', 'run-1')

    expect(store.epicRun).not.toBeNull()

    store.reset()

    expect(store.epicRun).toBeNull()
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })
})
