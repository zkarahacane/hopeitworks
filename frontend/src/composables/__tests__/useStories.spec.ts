import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useStories } from '../useStories'

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
]

describe('useStories', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockGet.mockResolvedValue({
      data: { data: [] },
      error: undefined,
    })
  })

  it('exposes reactive computed properties from the store', () => {
    const { stories, isLoading, error, selectedStory } = useStories('p1', 'e1')
    expect(stories.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
    expect(selectedStory.value).toBeNull()
  })

  it('fetches stories and updates reactive state', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const { stories, isLoading, fetchStories } = useStories('p1', 'e1')
    await fetchStories()

    expect(stories.value).toEqual(mockStories)
    expect(isLoading.value).toBe(false)
  })

  it('exposes error state on fetch failure', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'INTERNAL', message: 'Server error' } },
    })

    const { error, fetchStories } = useStories('p1', 'e1')
    await fetchStories()

    expect(error.value).toBe('Failed to load stories')
  })

  it('retry re-fetches with the same project and epic IDs', async () => {
    mockGet.mockResolvedValue({
      data: { data: [] },
      error: undefined,
    })

    const { fetchStories, retry } = useStories('p1', 'e1')
    await fetchStories()

    expect(mockGet).toHaveBeenCalledTimes(1)

    mockGet.mockClear()
    await retry()

    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('selectStory updates the selected story', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const { selectedStory, selectedStoryId, fetchStories, selectStory } = useStories('p1', 'e1')
    await fetchStories()

    selectStory('s2')
    expect(selectedStoryId.value).toBe('s2')
    expect(selectedStory.value).toEqual(mockStories[1])
  })

  it('setFilters updates the filter state', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockStories },
      error: undefined,
    })

    const { stories, filters, fetchStories, setFilters } = useStories('p1', 'e1')
    await fetchStories()

    setFilters({ search: 'profile' })
    expect(filters.value.search).toBe('profile')
    expect(stories.value).toHaveLength(1)
    expect(stories.value[0]!.id).toBe('s2')
  })
})
