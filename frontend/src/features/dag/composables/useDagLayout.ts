import { computed, onMounted } from 'vue'
import type { Node, Edge } from '@vue-flow/core'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

/**
 * Composable for fetching and transforming the epic DAG data
 * into @vue-flow/core compatible nodes and edges.
 */
export function useDagLayout(projectId: string, epicId: string) {
  const {
    data: dagData,
    isLoading,
    error,
    execute,
  } = useAsyncAction(async () => {
    const { data, error: apiErr } = await apiClient.GET(
      '/projects/{projectId}/epics/{epicId}/dag',
      { params: { path: { projectId, epicId } } },
    )
    if (apiErr) throw new Error('Failed to load DAG')
    return data
  })

  const nodes = computed<Node[]>(() => {
    if (!dagData.value) return []
    const layerCounters = new Map<number, number>()
    return dagData.value.nodes.map((n) => {
      const pos = layerCounters.get(n.layer) ?? 0
      layerCounters.set(n.layer, pos + 1)
      return {
        id: n.key,
        type: 'story',
        position: { x: n.layer * 250, y: pos * 120 },
        data: { key: n.key, title: n.title, status: n.status },
      }
    })
  })

  const edges = computed<Edge[]>(() => {
    if (!dagData.value) return []
    return dagData.value.edges.map((e) => ({
      id: `${e.source}-${e.target}`,
      source: e.source,
      target: e.target,
    }))
  })

  async function retry() {
    await execute()
  }

  onMounted(execute)

  return { nodes, edges, isLoading, error, retry }
}
