import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useRunCosts } from '../composables/useRunCosts'

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

const mockCostDetail = {
  run_id: 'run-1',
  total_cost: 4.52,
  steps: [
    {
      step_id: 'step-1',
      step_name: 'dev-story',
      model: 'claude-opus-4-6',
      tokens_input: 150000,
      tokens_output: 30000,
      cost_usd: 4.5,
    },
    {
      step_id: 'step-2',
      step_name: 'code-review',
      model: 'claude-sonnet-4-6',
      tokens_input: 5000,
      tokens_output: 1000,
      cost_usd: 0.02,
    },
  ],
}

describe('useRunCosts', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  it('fetches cost detail on mount', async () => {
    mockGet.mockResolvedValue({ data: mockCostDetail, error: undefined })

    const { costDetail, isLoading } = withSetup(() => useRunCosts('proj-1', 'run-1'))
    await flushPromises()

    expect(costDetail.value).toEqual(mockCostDetail)
    expect(isLoading.value).toBe(false)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/runs/{runId}/costs', {
      params: { path: { projectId: 'proj-1', runId: 'run-1' } },
    })
  })

  it('sets error when API returns error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Run not found' } },
    })

    const { costDetail, error } = withSetup(() => useRunCosts('proj-1', 'run-1'))
    await flushPromises()

    expect(costDetail.value).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load run costs')
  })

  it('retry() re-calls the API', async () => {
    mockGet.mockResolvedValue({ data: mockCostDetail, error: undefined })

    const { retry } = withSetup(() => useRunCosts('proj-1', 'run-1'))
    await flushPromises()

    expect(mockGet).toHaveBeenCalledTimes(1)

    mockGet.mockClear()
    mockGet.mockResolvedValue({ data: mockCostDetail, error: undefined })

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

    const { isLoading } = withSetup(() => useRunCosts('proj-1', 'run-1'))

    expect(isLoading.value).toBe(true)

    resolvePromise!({ data: mockCostDetail, error: undefined })
    await flushPromises()

    expect(isLoading.value).toBe(false)
  })
})
