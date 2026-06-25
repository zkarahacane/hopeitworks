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
    tokens_input: 120000,
    tokens_output: 30000,
  },
]

const byRoleData = {
  total_cost: 17.75,
  total_tokens_input: 700000,
  total_tokens_output: 150000,
  roles: [
    { role: 'implement', tokens_input: 500000, tokens_output: 100000, cost_usd: 12.5, runs_count: 3 },
    { role: 'review', tokens_input: 200000, tokens_output: 50000, cost_usd: 5.25, runs_count: 2 },
  ],
}

function mockSuccessResponses() {
  mockGet
    .mockResolvedValueOnce({ data: summaryData, error: undefined })
    .mockResolvedValueOnce({ data: chartData, error: undefined })
    .mockResolvedValueOnce({
      data: { data: runsData, pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })
    .mockResolvedValueOnce({ data: byRoleData, error: undefined })
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

  it('fetchAll calls all four endpoints with the current period', async () => {
    mockSuccessResponses()
    const { fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(mockGet).toHaveBeenCalledTimes(4)
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/summary', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/chart', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/runs', {
      params: { path: { projectId: 'proj-1' }, query: { period: '7d' } },
    })
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/costs/by-role', {
      params: { path: { projectId: 'proj-1' } },
    })
  })

  it('fetchAll populates summary, chartData, runs and byRole on success', async () => {
    mockSuccessResponses()
    const { summary, chartData: cd, runs, byRole, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(summary.value).toEqual(summaryData)
    expect(cd.value).toEqual(chartData)
    expect(runs.value).toEqual(runsData)
    expect(byRole.value).toEqual(byRoleData)
  })

  it('byRoleBreakdown maps the by-role endpoint into the shared CostByRoleResult', async () => {
    mockSuccessResponses()
    const { byRoleBreakdown, fetchAll } = useCosts('proj-1')
    await fetchAll()

    // RG1: roles present + total equals the server roll-up.
    expect(byRoleBreakdown.value.total).toBe(17.75)
    expect(byRoleBreakdown.value.roles.map((r) => r.role)).toEqual(['implement', 'review'])
    const impl = byRoleBreakdown.value.roles.find((r) => r.role === 'implement')!
    expect(impl.costUsd).toBe(12.5)
    // Bars scale against the largest role.
    expect(impl.fraction).toBe(1)
    expect(byRoleBreakdown.value.derivedFromStepsOnly).toBe(false)
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

  it('sets error and clears isLoading when summary API returns an error', async () => {
    mockGet
      .mockResolvedValueOnce({ data: undefined, error: { code: 'NOT_FOUND', message: 'not found' } })
      .mockResolvedValue({ data: [], error: undefined })

    const { error, isLoading, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(error.value).toBe('Failed to load cost summary')
    expect(isLoading.value).toBe(false)
  })

  it('sets error when chart API returns an error', async () => {
    mockGet
      .mockResolvedValueOnce({ data: summaryData, error: undefined })
      .mockResolvedValueOnce({ data: undefined, error: { code: 'INTERNAL', message: 'chart error' } })
      .mockResolvedValueOnce({ data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } }, error: undefined })
      .mockResolvedValueOnce({ data: byRoleData, error: undefined })

    const { error, isLoading, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(error.value).toBe('Failed to load cost chart')
    expect(isLoading.value).toBe(false)
  })

  it('sets error when runs API returns an error', async () => {
    mockGet
      .mockResolvedValueOnce({ data: summaryData, error: undefined })
      .mockResolvedValueOnce({ data: [], error: undefined })
      .mockResolvedValueOnce({ data: undefined, error: { code: 'INTERNAL', message: 'runs error' } })
      .mockResolvedValueOnce({ data: byRoleData, error: undefined })

    const { error, isLoading, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(error.value).toBe('Failed to load cost runs')
    expect(isLoading.value).toBe(false)
  })

  it('sets error when by-role API returns an error (RG5)', async () => {
    mockGet
      .mockResolvedValueOnce({ data: summaryData, error: undefined })
      .mockResolvedValueOnce({ data: chartData, error: undefined })
      .mockResolvedValueOnce({ data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } }, error: undefined })
      .mockResolvedValueOnce({ data: undefined, error: { code: 'INTERNAL', message: 'role error' } })

    const { error, isLoading, fetchAll } = useCosts('proj-1')
    await fetchAll()

    expect(error.value).toBe('Failed to load cost by role')
    expect(isLoading.value).toBe(false)
  })

  it('setPeriod updates period and re-fetches with new period', async () => {
    mockSuccessResponses()

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
      .mockResolvedValueOnce({ data: byRoleData, error: undefined })

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
