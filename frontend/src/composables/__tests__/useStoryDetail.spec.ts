import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useStoryDetail } from '../useStoryDetail'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

/** Must wrap composable usage that calls onMounted in a simulated lifecycle */
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

const mockStory = {
  id: 's1',
  epic_id: 'e1',
  project_id: 'p1',
  key: 'S-01',
  title: 'Setup authentication',
  status: 'backlog',
  objective: 'Set up auth',
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useStoryDetail', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  it('fetches story on mount and populates story.value', async () => {
    mockGet.mockResolvedValue({ data: mockStory, error: undefined })

    const { story, isLoading } = withSetup(() => useStoryDetail('p1', 's1'))
    await flushPromises()

    expect(story.value).toEqual(mockStory)
    expect(isLoading.value).toBe(false)
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('sets error when API returns error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Story not found' } },
    })

    const { story, error } = withSetup(() => useStoryDetail('p1', 's1'))
    await flushPromises()

    expect(story.value).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load story')
  })

  it('retry() re-calls the API', async () => {
    mockGet.mockResolvedValue({ data: mockStory, error: undefined })

    const { retry } = withSetup(() => useStoryDetail('p1', 's1'))
    await flushPromises()

    expect(mockGet).toHaveBeenCalledTimes(1)

    mockGet.mockClear()
    mockGet.mockResolvedValue({ data: mockStory, error: undefined })

    await retry()
    expect(mockGet).toHaveBeenCalledTimes(1)
  })
})
