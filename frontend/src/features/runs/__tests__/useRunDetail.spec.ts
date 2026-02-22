import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useRunDetail } from '../composables/useRunDetail'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

vi.mock('@/composables/useSSE', () => ({
  useSSE: vi.fn(),
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

const mockRun = {
  id: 'run-1',
  project_id: 'proj-1',
  story_id: 'story-1',
  status: 'running',
  created_at: '2026-02-17T10:00:00Z',
  updated_at: '2026-02-17T10:00:00Z',
  steps: [
    {
      id: 'step-1',
      run_id: 'run-1',
      step_name: 'dev-story',
      step_order: 1,
      action: 'agent_run',
      status: 'completed',
      created_at: '2026-02-17T10:00:00Z',
    },
    {
      id: 'step-2',
      run_id: 'run-1',
      step_name: 'code-review',
      step_order: 2,
      action: 'agent_run',
      status: 'running',
      created_at: '2026-02-17T10:01:00Z',
    },
  ],
}

describe('useRunDetail', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  it('fetches run on mount and populates run.value', async () => {
    mockGet.mockResolvedValue({ data: mockRun, error: undefined })

    const { run, isLoading } = withSetup(() => useRunDetail('run-1', ''))
    await flushPromises()

    expect(run.value).toEqual(mockRun)
    expect(isLoading.value).toBe(false)
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('sets error when API returns error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Run not found' } },
    })

    const { run, error } = withSetup(() => useRunDetail('run-1', ''))
    await flushPromises()

    expect(run.value).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load run')
  })

  it('retry() re-calls the API', async () => {
    mockGet.mockResolvedValue({ data: mockRun, error: undefined })

    const { retry } = withSetup(() => useRunDetail('run-1', ''))
    await flushPromises()

    expect(mockGet).toHaveBeenCalledTimes(1)

    mockGet.mockClear()
    mockGet.mockResolvedValue({ data: mockRun, error: undefined })

    await retry()
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('exposes isLoading as true during fetch', async () => {
    let resolvePromise: (value: unknown) => void
    mockGet.mockReturnValue(
      new Promise((resolve) => {
        resolvePromise = resolve
      }),
    )

    const { isLoading } = withSetup(() => useRunDetail('run-1', ''))

    expect(isLoading.value).toBe(true)

    resolvePromise!({ data: mockRun, error: undefined })
    await flushPromises()

    expect(isLoading.value).toBe(false)
  })
})
