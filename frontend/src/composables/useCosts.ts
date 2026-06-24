import { computed, onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'
import { projectCostByRole, type CostByRoleResult } from '@/utils/costByRole'

type CostSummary = components['schemas']['CostSummary']
type CostDataPoint = components['schemas']['CostDataPoint']
type RunCostRow = components['schemas']['RunCostRow']
type AgentCostBreakdown = components['schemas']['AgentCostBreakdown']
type ProjectCostByRole = components['schemas']['ProjectCostByRole']

/**
 * Composable for fetching and managing cost data for a project.
 * Owns all state: summary, chart data, runs, period, isLoading, error.
 */
export function useCosts(projectId: string) {
  const period = ref<'7d' | '30d'>('7d')
  const summary = ref<CostSummary | null>(null)
  const chartData = ref<CostDataPoint[]>([])
  const runs = ref<RunCostRow[]>([])
  const byRole = ref<ProjectCostByRole | null>(null)
  const agentCosts = ref<AgentCostBreakdown[]>([])
  const agentCostsLoading = ref(false)
  const agentCostsError = ref<string | null>(null)
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  /**
   * COST BY ROLE breakdown for the Overview widget, mapped from the
   * authoritative project-level endpoint into the shared CostByRoleResult shape
   * consumed by RunCostByRole. Empty (zero roles, zero total) when there is no
   * cost in the period (RG3).
   */
  const byRoleBreakdown = computed<CostByRoleResult>(() => projectCostByRole(byRole.value))

  async function fetchAll() {
    isLoading.value = true
    error.value = null
    try {
      const [sumRes, chartRes, runsRes, byRoleRes] = await Promise.all([
        apiClient.GET('/projects/{projectId}/costs/summary', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
        apiClient.GET('/projects/{projectId}/costs/chart', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
        apiClient.GET('/projects/{projectId}/costs/runs', {
          params: { path: { projectId }, query: { period: period.value } },
        }),
        apiClient.GET('/projects/{projectId}/costs/by-role', {
          params: { path: { projectId } },
        }),
      ])
      if (sumRes.error) throw new Error('Failed to load cost summary')
      if (chartRes.error) throw new Error('Failed to load cost chart')
      if (runsRes.error) throw new Error('Failed to load cost runs')
      if (byRoleRes.error) throw new Error('Failed to load cost by role')
      summary.value = sumRes.data ?? null
      chartData.value = chartRes.data ?? []
      runs.value = runsRes.data?.data ?? []
      byRole.value = byRoleRes.data ?? null
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
    byRole,
    byRoleBreakdown,
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
