import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useCosts } from '../useCosts'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

// Suppress onMounted in unit test context — tests call fetchAll directly
vi.mock('vue', async (importOriginal) => {
  const vue = await importOriginal<typeof import('vue')>()
  return { ...vue, onMounted: vi.fn() }
})

const summaryData = {
  total_cost_usd: 1.5,
  total_cost_week_usd: 0.5,
  total_cost_month_usd: 1.5,
  avg_cost_per_story_usd: 0.25,
  budget_limit_usd: 10.0,
  period_start: '2026-02-10T00:00:00Z',
  period_end: '2026-02-17T00:00:00Z',
}

const chartData = [
  { date: '2026-02-10', total_cost_usd: 0.2 },
  { date: '2026-02-11', total_cost_usd: 0.3 },
]

const runsData = [
  {
    run_id: 'run-1',
    story_key: 'S-01',
    status: 'completed',
    started_at: '2026-02-10T10:00:00Z',
    total_cost_usd: 0.01234,
  },
]

function mockSuccessResponses() {
  mockGet
    .mockResolvedValueOnce({ data: summaryData, error: undefined })
    .mockResolvedValueOnce({ data: chartData, error: undefined })
    .mockResolvedValueOnce({
      data: { data: runsData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })
}

describe('useCosts', () => {
  beforeEach(() => {
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const { period, summary, chartData: cd, runs, isLoading, error } = useCosts('p1')
    expect(period.value).toBe('7d')
    expect(summary.value).toBeNull()
    expect(cd.value).toEqual([])
    expect(runs.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(error.value).toBeNull()
  })

  it('fetchAll calls all three endpoints with the current period', async () => {
    mockSuccessResponses()
    const { fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(mockGet).toHaveBeenCalledTimes(3)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/summary', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/chart', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/runs', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
  })

  it('fetchAll populates summary, chartData and runs on success', async () => {
    mockSuccessResponses()
    const { summary, chartData: cd, runs, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(summary.value).toEqual(summaryData)
    expect(cd.value).toEqual(chartData)
    expect(runs.value).toEqual(runsData)
  })

  it('isLoading is true during fetch and false after', async () => {
    let resolve!: (v: unknown) => void
    mockGet.mockReturnValueOnce(new Promise((r) => (resolve = r)))
    mockGet.mockResolvedValue({ data: [], error: undefined })

    const { isLoading, fetchAll } = useCosts('proj-1')
    expect(isLoading.value).toBe(false)

    const fetchPromise = fetchAll()
    expect(isLoading.value).toBe(true)

    resolve({ data: summaryData, error: undefined })
    await fetchPromise

    expect(isLoading.value).toBe(false)
  })

  it('sets error and clears isLoading when API returns an error', async () => {
    mockGet
      .mockResolvedValueOnce({ data: undefined, error: { code: 'NOT_FOUND', message: 'not found' } })
      .mockResolvedValue({ data: [], error: undefined })

    const { error, isLoading, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(error.value).toBe('Failed to load cost summary')
    expect(isLoading.value).toBe(false)
  })

  it('setPeriod updates period and re-fetches with new period', async () => {
    mockSuccessResponses()
    // Second call set for 30d
    mockGet
      .mockResolvedValueOnce({ data: summaryData, error: undefined })
      .mockResolvedValueOnce({ data: [], error: undefined })
      .mockResolvedValueOnce({ data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } }, error: undefined })

    const { period, setPeriod } = useCosts('proj-1')

    // First fetch (7d)
    await setPeriod('7d')
    expect(period.value).toBe('7d')
    mockGet.mockClear()

    // Second fetch with 30d
    mockGet
      .mockResolvedValueOnce({ data: summaryData, error: undefined })
      .mockResolvedValueOnce({ data: [], error: undefined })
      .mockResolvedValueOnce({ data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } }, error: undefined })

    await setPeriod('30d')
    expect(period.value).toBe('30d')

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/summary', {
      params: { path: { projectId: 'proj-1' }, query: { period: '30d' } },
    })
  })

  it('clears error on successful retry', async () => {
    // First fetch fails
    mockGet.mockResolvedValue({ data: undefined, error: { code: 'INTERNAL', message: 'fail' } })
    const { error, fetchAll } = useCosts('proj-1')
    await fetchAll()
    expect(error.value).not.toBeNull()

    // Retry succeeds
    mockGet.mockReset()
    mockSuccessResponses()
    await fetchAll()
    expect(error.value).toBeNull()
  })
})
