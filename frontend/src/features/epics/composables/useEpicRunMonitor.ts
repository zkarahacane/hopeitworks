import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { Node, Edge } from '@vue-flow/core'
import { useSSE } from '@/composables/useSSE'
import { useEpicRunStore } from '@/stores/epicRun'

/**
 * Composable that wires together the epic run store and SSE events
 * for real-time monitoring of an epic run. Derives VueFlow nodes/edges
 * from the store's epic run stories.
 */
export function useEpicRunMonitor(projectId: string, epicRunId: string) {
  const epicRunStore = useEpicRunStore()

  const { status: sseStatus } = useSSE(projectId, (eventName, data) => {
    if (eventName.startsWith('epic_run.')) {
      epicRunStore.handleSSEEvent(eventName, data)
    }
  })

  const nodes = computed<Node[]>(() => {
    const stories = epicRunStore.epicRun?.stories ?? []
    const groupCounters = new Map<number, number>()
    return stories.map((s) => {
      const pos = groupCounters.get(s.group_index) ?? 0
      groupCounters.set(s.group_index, pos + 1)
      const storyKey = s.story_key ?? s.story_id
      return {
        id: storyKey,
        type: 'epicRunStatus',
        position: { x: s.group_index * 250, y: pos * 120 },
        data: {
          key: storyKey,
          title: storyKey,
          status: s.status,
          runId: s.run_id ?? null,
        },
      }
    })
  })

  const edges = computed<Edge[]>(() => [])

  onMounted(() => epicRunStore.fetchEpicRun(projectId, epicRunId))
  onBeforeUnmount(() => epicRunStore.reset())

  return { epicRunStore, nodes, edges, sseStatus }
}
