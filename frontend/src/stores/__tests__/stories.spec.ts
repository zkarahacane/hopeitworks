import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import {
  useStoriesStore,
  boardColumn,
  stageColumn,
  STAGE_BACKLOG_COLUMN,
  STAGE_DONE_COLUMN,
  STAGE_FAILED_COLUMN,
  STAGE_RUNNING_ENTRY,
  type Story,
} from '../stories'

const mockGet = vi.fn()
const mockPut = vi.fn()
const mockPost = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
    POST: (...args: unknown[]) => mockPost(...args),
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
    mockPut.mockReset()
    mockPost.mockReset()
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

  describe('updateStory', () => {
    it('updates item in-place on success', async () => {
      mockGet.mockResolvedValue({
        data: { data: mockStories },
        error: undefined,
      })

      const store = useStoriesStore()
      await store.fetchStoriesByEpic('p1', 'e1')

      const updatedStory = { ...mockStories[1], title: 'Updated title' }
      mockPut.mockResolvedValue({ data: updatedStory, error: undefined })

      const result = await store.updateStory('p1', 's2', { title: 'Updated title' })

      expect(result).toEqual(updatedStory)
      expect(store.items[1]!.title).toBe('Updated title')
      expect(store.error).toBeNull()
    })

    it('returns null and sets error on API error', async () => {
      mockGet.mockResolvedValue({
        data: { data: mockStories },
        error: undefined,
      })

      const store = useStoriesStore()
      await store.fetchStoriesByEpic('p1', 'e1')

      mockPut.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'NOT_FOUND', message: 'Story not found' } },
      })

      const result = await store.updateStory('p1', 's2', { title: 'Updated' })

      expect(result).toBeNull()
      expect(store.error).toBe('Story not found')
    })

    it('returns null and sets error on thrown exception', async () => {
      const store = useStoriesStore()
      mockPut.mockRejectedValue(new Error('Network error'))

      const result = await store.updateStory('p1', 's1', { title: 'Updated' })

      expect(result).toBeNull()
      expect(store.error).toBe('Network error')
    })
  })

  describe('createStory', () => {
    it('pushes new story to items on success', async () => {
      const store = useStoriesStore()
      const newStory = {
        id: 's4',
        epic_id: 'e1',
        project_id: 'p1',
        key: 'S-04',
        title: 'New story',
        status: 'backlog',
        created_at: '2026-01-18T10:00:00Z',
        updated_at: '2026-01-18T10:00:00Z',
      }
      mockPost.mockResolvedValue({ data: newStory, error: undefined })

      const result = await store.createStory('p1', { key: 'S-04', title: 'New story' })

      expect(result).toEqual(newStory)
      expect(store.items).toHaveLength(1)
      expect(store.items[0]!.key).toBe('S-04')
    })

    it('returns null and sets error on API error', async () => {
      const store = useStoriesStore()
      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'CONFLICT', message: 'Key already exists' } },
      })

      const result = await store.createStory('p1', { key: 'S-01', title: 'Duplicate' })

      expect(result).toBeNull()
      expect(store.error).toBe('Key already exists')
    })

    it('returns null and sets error on thrown exception', async () => {
      const store = useStoriesStore()
      mockPost.mockRejectedValue(new Error('Network error'))

      const result = await store.createStory('p1', { key: 'S-05', title: 'Story' })

      expect(result).toBeNull()
      expect(store.error).toBe('Network error')
    })
  })

  describe('handleSSEEvent: stage.entered', () => {
    function seedRunningStory(): Story {
      const story: Story = {
        id: 's-stage',
        epic_id: 'e1',
        project_id: 'p1',
        key: 'S-09',
        title: 'Stage card',
        status: 'running',
        current_stage: 'Setup',
        created_at: '2026-01-20T10:00:00Z',
        updated_at: '2026-01-20T10:00:00Z',
      }
      const store = useStoriesStore()
      store.items = [story]
      return story
    }

    it('advances current_stage to the entered stage name', () => {
      seedRunningStory()
      const store = useStoriesStore()
      store.handleSSEEvent('stage.entered', {
        story_id: 's-stage',
        stage_id: 'g2',
        stage_name: 'Development',
      })
      expect(store.items[0]!.current_stage).toBe('Development')
    })

    it('ignores events for unknown stories', () => {
      seedRunningStory()
      const store = useStoriesStore()
      store.handleSSEEvent('stage.entered', {
        story_id: 'does-not-exist',
        stage_name: 'Development',
      })
      expect(store.items[0]!.current_stage).toBe('Setup')
    })

    it('ignores events missing a stage_name', () => {
      seedRunningStory()
      const store = useStoriesStore()
      store.handleSSEEvent('stage.entered', { story_id: 's-stage' })
      expect(store.items[0]!.current_stage).toBe('Setup')
    })
  })
})

describe('stageColumn', () => {
  function makeStory(overrides: Partial<Story>): Story {
    return {
      id: 's',
      epic_id: 'e1',
      project_id: 'p1',
      key: 'S-01',
      title: 'Story',
      status: 'backlog',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
      ...overrides,
    }
  }

  it('places a story in its current_stage', () => {
    expect(stageColumn(makeStory({ status: 'running', current_stage: 'In Dev' }))).toBe('In Dev')
  })

  it('falls back to the backlog lane when no stage is set (backlog story)', () => {
    expect(stageColumn(makeStory({ status: 'backlog' }))).toBe(STAGE_BACKLOG_COLUMN)
  })

  it('RG2 (#300): a running story without a stage routes to the running-entry sentinel, not Backlog', () => {
    const story = makeStory({ status: 'running', current_stage: null })
    const col = stageColumn(story)
    expect(col).toBe(STAGE_RUNNING_ENTRY)
    expect(col).not.toBe(STAGE_BACKLOG_COLUMN)
  })

  it('RG3 (#300): a running+NULL story is in_progress (macro) AND off the backlog lane (détail)', () => {
    const story = makeStory({ status: 'running', current_stage: null })
    expect(boardColumn(story)).toBe('in_progress')
    expect(stageColumn(story)).not.toBe(STAGE_BACKLOG_COLUMN)
  })

  it('sends done stories to the done lane regardless of stage', () => {
    expect(stageColumn(makeStory({ status: 'done', current_stage: 'In Dev' }))).toBe(
      STAGE_DONE_COLUMN,
    )
  })

  it('sends failed stories to the failed lane regardless of stage', () => {
    expect(stageColumn(makeStory({ status: 'failed', current_stage: 'In Dev' }))).toBe(
      STAGE_FAILED_COLUMN,
    )
  })
})
