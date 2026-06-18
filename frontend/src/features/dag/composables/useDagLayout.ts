import { computed, onMounted } from 'vue'
import type { Edge, Node } from '@vue-flow/core'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import { useRuntimeStream } from '@/stores/runtimeStream'
import { statusFamily } from '@/utils/statusToken'

/**
 * Layout + LIVE wiring for the Execution Graph hero.
 *
 * Transforms the epic DAG (`/projects/{id}/epics/{eid}/dag`) into @vue-flow/core
 * nodes/edges AND layers the live runtime signal on top:
 *  - per-node live status (runtimeStream run signal overrides the static REST
 *    status when a run for the story is being tracked)
 *  - marching edges: an edge animates when its SOURCE story is active (running)
 *  - the active-node set drives node glow + edge marching
 *
 * Data gaps handled here (the DAG REST payload is intentionally thin):
 *  - The payload carries no run_id per story, so live signals key off an
 *    OPTIONAL `storyRunMap` (story key → run id). When a story isn't mapped
 *    (the common case: opening /dag with no live epic run), we fall back to the
 *    node's own REST status and the demo container/cost/timer seeds below.
 *  - The payload carries no container id / cost / elapsed per node. The hero is
 *    a flagship demo surface, so we seed deterministic per-node values (stable
 *    hash of the key) when no live signal exists, matching the spec example
 *    data. Live runs replace the seed with real runtimeStream values.
 */

export type DagNodeStatus = ReturnType<typeof statusFamily>

/** Rich data carried on each VueFlow node, consumed by DagStoryNode.vue. */
export interface DagNodeData {
  key: string
  title: string
  /** Live-resolved status string (run signal wins over REST status). */
  status: string
  /** Static REST status (kept for reference / fallback). */
  restStatus: string
  layer: number
  /** Run id tracked for this story, if any (enables live timer/cost). */
  runId: string | null
  /** Whether this node is currently active (running) — drives glow. */
  active: boolean
  /** Container short identity, e.g. "a3f9". Demo seed when no live data. */
  containerId: string | null
  /** Live elapsed seconds (0 when not started). */
  elapsedSeconds: number
  /** Live USD cost for the node's run (0 until backend streams USD). */
  costUsd: number
  /** Failed-node exit message, e.g. "exit 1". */
  exitMessage: string | null
  /** Story keys this node is waiting on (queued nodes). */
  waitingOn: string[]
}

/** Demo seed values keyed by story key, matching the spec's example data. */
const DEMO_SEED: Record<string, { container: string | null; elapsed: number; cost: number }> = {
  'S-01': { container: null, elapsed: 222, cost: 0.18 },
  'S-02': { container: 'a3f9', elapsed: 198, cost: 0.11 },
  'S-03': { container: '7c1d', elapsed: 87, cost: 0.06 },
  'S-04': { container: null, elapsed: 0, cost: 0 },
  'S-05': { container: 'e2b8', elapsed: 0, cost: 0.04 },
  'S-06': { container: null, elapsed: 0, cost: 0 },
}

/** Deterministic 4-char hex container suffix from a story key (fallback seed). */
export function seedContainerId(key: string): string {
  let h = 0
  for (let i = 0; i < key.length; i++) {
    h = (h * 31 + key.charCodeAt(i)) >>> 0
  }
  return h.toString(16).padStart(4, '0').slice(-4)
}

export function useDagLayout(
  projectId: string,
  epicId: string,
  /** Optional story key → run id map enabling per-node live runtime signals. */
  storyRunMap?: () => Record<string, string>,
) {
  const stream = useRuntimeStream()

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

  /** Story key → its incoming dependency sources (for "waiting on" labels). */
  const dependencySources = computed<Record<string, string[]>>(() => {
    const map: Record<string, string[]> = {}
    if (!dagData.value) return map
    for (const e of dagData.value.edges) {
      ;(map[e.target] ??= []).push(e.source)
    }
    return map
  })

  /** Resolve the rich, live-aware data for one DAG node. */
  function resolveNodeData(
    n: { key: string; title: string; status: string; layer: number },
  ): DagNodeData {
    const runId = storyRunMap?.()[n.key] ?? null
    const runSig = runId ? stream.runSignal(runId) : null

    // Live status wins; otherwise the REST status from the DAG payload.
    const status = runSig?.status ?? n.status
    const family = statusFamily(status)
    const active = family === 'running'

    // Container: live active step implies a live container; else demo seed.
    const seed = DEMO_SEED[n.key]
    const liveContainer = runId && active ? seedContainerId(runId) : null
    const containerId =
      liveContainer ?? (active || seed?.container ? (seed?.container ?? seedContainerId(n.key)) : null)

    // Timing/cost: live when a run is tracked, else demo seed (frozen).
    const elapsedSeconds = runId
      ? stream.runElapsedSeconds(runId)
      : (seed?.elapsed ?? 0)
    const costUsd = runId ? stream.runCostUsd(runId) : (seed?.cost ?? 0)

    const exitMessage = family === 'failed' ? 'exit 1' : null
    const waitingOn = family === 'queued' ? (dependencySources.value[n.key] ?? []) : []

    return {
      key: n.key,
      title: n.title,
      status,
      restStatus: n.status,
      layer: n.layer,
      runId,
      active,
      containerId,
      elapsedSeconds,
      costUsd,
      exitMessage,
      waitingOn,
    }
  }

  const nodes = computed<Node<DagNodeData>[]>(() => {
    if (!dagData.value) return []
    const layerCounters = new Map<number, number>()
    return dagData.value.nodes.map((n) => {
      const pos = layerCounters.get(n.layer) ?? 0
      layerCounters.set(n.layer, pos + 1)
      return {
        id: n.key,
        type: 'story',
        position: { x: n.layer * 320, y: pos * 170 },
        data: resolveNodeData(n),
      }
    })
  })

  const edges = computed<Edge[]>(() => {
    if (!dagData.value) return []
    const byKey = new Map(nodes.value.map((node) => [node.id, node.data]))
    return dagData.value.edges.map((e) => {
      const sourceActive = byKey.get(e.source)?.active ?? false
      return {
        id: `${e.source}-${e.target}`,
        source: e.source,
        target: e.target,
        // Marching edges: animate while the source story is active (running).
        class: sourceActive ? 'dag-edge dag-edge--active' : 'dag-edge',
        data: { active: sourceActive },
      }
    })
  })

  /** Story counts for the subtitle / legend. */
  const summary = computed(() => {
    const families = { running: 0, done: 0, failed: 0, gate: 0, queued: 0 }
    for (const node of nodes.value) {
      families[statusFamily(node.data!.status)]++
    }
    return {
      total: nodes.value.length,
      running: families.running,
      done: families.done,
      failed: families.failed,
      gate: families.gate,
      queued: families.queued,
    }
  })

  /** Look up rich node data by story key (for the inspector). */
  const nodeByKey = computed(() => {
    const map = new Map<string, DagNodeData>()
    for (const node of nodes.value) map.set(node.id, node.data!)
    return map
  })

  async function retry() {
    await execute()
  }

  onMounted(execute)

  return { nodes, edges, summary, nodeByKey, isLoading, error, retry, execute }
}
