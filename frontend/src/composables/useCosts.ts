import { onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

type CostSummary = components['schemas']['CostSummary']
type CostDataPoint = components['schemas']['CostDataPoint']
type RunCostRow = components['schemas']['RunCostRow']
type AgentCostBreakdown = components['schemas']['AgentCostBreakdown']

/**
 * Composable for fetching and managing cost data for a project.
 * Owns all state: summary, chart data, runs, period, isLoading, error.
 */
export function useCosts(projectId: string) {
  const period = ref<'7d' | '30d'>('7d')
  const summary = ref<CostSummary | null>(null)
  const chartData = ref<CostDataPoint[]>([])
  const runs = ref<RunCostRow[]>([])
  const agentCosts = ref<AgentCostBreakdown[]>([])
  const agentCostsLoading = ref(false)
  const agentCostsError = ref<string | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    isLoading.value = true
    error.value = null
    try {
      const [sumRes, chartRes, runsRes] = await Promise.all([
        apiClient.GET('/projects/{projectId}/costs/summary', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
        apiClient.GET('/projects/{projectId}/costs/chart', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
        apiClient.GET('/projects/{projectId}/costs/runs', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
      ])
      if (sumRes.error) throw new Error('Failed to load cost summary')
      if (chartRes.error) throw new Error('Failed to load cost chart')
      if (runsRes.error) throw new Error('Failed to load cost runs')
      summary.value = sumRes.data ?? null
      chartData.value = chartRes.data ?? []
      runs.value = runsRes.data?.data ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load cost data'
    } finally {
      isLoading.value = false
    }
  }

  /** Fetch cost data aggregated by agent for the project. */
  async function fetchAgentCosts() {
    agentCostsLoading.value = true
    agentCostsError.value = null
    try {
      const { data, error: err } = await apiClient.GET(
        '/projects/{projectId}/costs/agents',
        { params: { path: { projectId } } },
      )
      if (err) throw new Error('Failed to load agent costs')
      agentCosts.value = data ?? []
    } catch (e) {
      agentCostsError.value = e instanceof Error ? e.message : 'Failed to load agent costs'
    } finally {
      agentCostsLoading.value = false
    }
  }

  /** Update the active period and re-fetch all cost data. */
  function setPeriod(p: '7d' | '30d') {
    period.value = p
    return fetchAll()
  }

  onMounted(fetchAll)

  return {
    period,
    summary,
    chartData,
    runs,
    isLoading,
    error,
    fetchAll,
    setPeriod,
    agentCosts,
    agentCostsLoading,
    agentCostsError,
    fetchAgentCosts,
  }
}
