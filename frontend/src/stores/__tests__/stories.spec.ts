import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useStoriesStore } from '../stories'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

const mockStories = [
  {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Setup authentication',
    status: 'done',
    objective: 'Implement auth flow',
    acceptance_criteria: 'Users can log in',
    target_files: ['src/auth.ts'],
    depends_on: [],
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 's2',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-02',
    title: 'Add user profile page',
    status: 'backlog',
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
  {
    id: 's3',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-03',
    title: 'Fix login bug',
    status: 'failed',
    created_at: '2026-01-17T10:00:00Z',
    updated_at: '2026-01-17T10:00:00Z',
  },
]

describe('useStoriesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useStoriesStore()
    expect(store.items).toEqual([])
    expect(store.selectedStoryId).toBeNull()
    expect(store.filters).toEqual({ status: null, search: '' })
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetches stories successfully and populates items', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    expect(store.items).toEqual(mockStories)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('sets error state when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    expect(store.items).toEqual([])
    expect(store.error).toBe('Failed to load stories')
    expect(store.isLoading).toBe(false)
  })

  it('sets error state when API call throws', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    expect(store.items).toEqual([])
    expect(store.error).toBe('Network error')
    expect(store.isLoading).toBe(false)
  })

  it('sets fallback error message for non-Error thrown values', async () => {
    mockGet.mockRejectedValue('unknown error')

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    expect(store.error).toBe('Failed to load stories')
  })

  it('clears previous error on new fetch', async () => {
    mockGet
      .mockResolvedValueOnce({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'fail' } },
      })
      .mockResolvedValueOnce({
        data: { data: [] },
        error: undefined,
      })

    const store = useStoriesStore()

    await store.fetchStoriesByEpic('p1', 'e1')
    expect(store.error).toBe('Failed to load stories')

    await store.fetchStoriesByEpic('p1', 'e1')
    expect(store.error).toBeNull()
  })

  it('clearError resets error state', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'fail' } },
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')
    expect(store.error).toBe('Failed to load stories')

    store.clearError()
    expect(store.error).toBeNull()
  })

  it('sets selected story', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setSelectedStory('s2')
    expect(store.selectedStoryId).toBe('s2')
    expect(store.selectedStory).toEqual(mockStories[1])
  })

  it('returns null for selectedStory when no story is selected', () => {
    const store = useStoriesStore()
    expect(store.selectedStory).toBeNull()
  })

  it('returns null for selectedStory when selected ID does not match', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setSelectedStory('nonexistent')
    expect(store.selectedStory).toBeNull()
  })

  it('filters stories by status', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setFilters({ status: 'done' })
    expect(store.filteredStories).toHaveLength(1)
    expect(store.filteredStories[0]!.id).toBe('s1')
  })

  it('filters stories by text search on key and title', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setFilters({ search: 'login' })
    expect(store.filteredStories).toHaveLength(1)
    expect(store.filteredStories[0]!.id).toBe('s3')
  })

  it('filters stories by search on key', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setFilters({ search: 'S-01' })
    expect(store.filteredStories).toHaveLength(1)
    expect(store.filteredStories[0]!.id).toBe('s1')
  })

  it('returns all stories when status filter is "all"', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setFilters({ status: 'all' })
    expect(store.filteredStories).toHaveLength(3)
  })

  it('combines status and text filters', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')

    store.setFilters({ status: 'backlog', search: 'profile' })
    expect(store.filteredStories).toHaveLength(1)
    expect(store.filteredStories[0]!.id).toBe('s2')
  })

  it('resets all state', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const store = useStoriesStore()
    await store.fetchStoriesByEpic('p1', 'e1')
    store.setSelectedStory('s1')
    store.setFilters({ status: 'done', search: 'test' })

    expect(store.items).toHaveLength(3)

    store.reset()

    expect(store.items).toEqual([])
    expect(store.selectedStoryId).toBeNull()
    expect(store.filters).toEqual({ status: null, search: '' })
    expect(store.error).toBeNull()
    expect(store.isLoading).toBe(false)
  })
})
