import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

const mockGET = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGET(...args),
  },
}))

// Must import after mock setup
const { useDagLayout, seedContainerId } = await import('../composables/useDagLayout')
const { useRuntimeStream } = await import('@/stores/runtimeStream')

/** Mount composable inside a component to trigger onMounted lifecycle. */
function withSetup<T>(composable: () => T): T {
  let result!: T
  const Comp = defineComponent({
    setup() {
      result = composable()
      return () => h('div')
    },
  })
  mount(Comp)
  return result
}

describe('useDagLayout', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGET.mockReset()
  })

  it('transforms API nodes to vue-flow Node[] with rich data + layered positions', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [{ key: 'S-01', layer: 0, title: 'First story', status: 'done' }],
        edges: [],
      },
    })

    const { nodes, edges, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodes.value).toHaveLength(1)
    const node = nodes.value[0]!
    expect(node.id).toBe('S-01')
    expect(node.type).toBe('story')
    expect(node.position).toEqual({ x: 0, y: 0 })
    expect(node.data).toMatchObject({
      key: 'S-01',
      title: 'First story',
      status: 'done',
      restStatus: 'done',
      layer: 0,
      active: false,
    })
    expect(edges.value).toHaveLength(0)
  })

  it('layers nodes: x by layer, y by position within layer', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-01', layer: 0, title: 'First', status: 'queued' },
          { key: 'S-02', layer: 0, title: 'Second', status: 'running' },
          { key: 'S-03', layer: 1, title: 'Third', status: 'queued' },
        ],
        edges: [],
      },
    })

    const { nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodes.value[0]!.position).toEqual({ x: 0, y: 0 })
    expect(nodes.value[1]!.position).toEqual({ x: 0, y: 170 })
    expect(nodes.value[2]!.position).toEqual({ x: 320, y: 0 })
  })

  it('marks running nodes active and seeds a container id', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [{ key: 'S-02', layer: 0, title: 'Running one', status: 'running' }],
        edges: [],
      },
    })

    const { nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    const data = nodes.value[0]!.data!
    expect(data.active).toBe(true)
    expect(data.containerId).toBeTruthy()
  })

  it('exposes exit message + retry affordance for failed nodes', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [{ key: 'S-05', layer: 0, title: 'Broken', status: 'failed' }],
        edges: [],
      },
    })

    const { nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodes.value[0]!.data!.exitMessage).toBe('exit 1')
  })

  it('derives waiting-on dependencies for queued nodes from edges', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-02', layer: 0, title: 'Dep', status: 'running' },
          { key: 'S-04', layer: 1, title: 'Waiter', status: 'queued' },
        ],
        edges: [{ source: 'S-02', target: 'S-04' }],
      },
    })

    const { nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    const waiter = nodes.value.find((n) => n.id === 'S-04')!
    expect(waiter.data!.waitingOn).toEqual(['S-02'])
  })

  it('marks an edge active when its source node is running (marching)', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-02', layer: 0, title: 'Active src', status: 'running' },
          { key: 'S-04', layer: 1, title: 'Target', status: 'queued' },
          { key: 'S-01', layer: 0, title: 'Done src', status: 'done' },
          { key: 'S-03', layer: 1, title: 'Target2', status: 'queued' },
        ],
        edges: [
          { source: 'S-02', target: 'S-04' },
          { source: 'S-01', target: 'S-03' },
        ],
      },
    })

    const { edges, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    const active = edges.value.find((e) => e.id === 'S-02-S-04')!
    const idle = edges.value.find((e) => e.id === 'S-01-S-03')!
    expect(active.data).toEqual({ active: true })
    expect(active.class).toContain('dag-edge--active')
    expect(idle.data).toEqual({ active: false })
    expect(idle.class).not.toContain('dag-edge--active')
  })

  it('uses live runtimeStream status over the REST status when a story is mapped to a run', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [{ key: 'S-02', layer: 0, title: 'Story', status: 'queued' }],
        edges: [],
      },
    })

    const stream = useRuntimeStream()
    stream.ingest('run.started', { run_id: 'run-xyz', started_at: '2026-06-17T10:00:00Z' })

    const { nodes, isLoading } = withSetup(() =>
      useDagLayout('proj-1', 'epic-1', () => ({ 'S-02': 'run-xyz' })),
    )
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    const data = nodes.value[0]!.data!
    expect(data.status).toBe('running')
    expect(data.restStatus).toBe('queued')
    expect(data.runId).toBe('run-xyz')
    expect(data.active).toBe(true)
  })

  it('computes a status summary for the subtitle + legend', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-01', layer: 0, title: 'a', status: 'done' },
          { key: 'S-02', layer: 0, title: 'b', status: 'running' },
          { key: 'S-03', layer: 0, title: 'c', status: 'running' },
          { key: 'S-05', layer: 1, title: 'd', status: 'failed' },
          { key: 'S-06', layer: 1, title: 'e', status: 'queued' },
        ],
        edges: [],
      },
    })

    const { summary, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(summary.value.total).toBe(5)
    expect(summary.value.running).toBe(2)
    expect(summary.value.done).toBe(1)
    expect(summary.value.failed).toBe(1)
    expect(summary.value.queued).toBe(1)
  })

  it('exposes nodeByKey lookup for the inspector', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [{ key: 'S-01', layer: 0, title: 'First', status: 'done' }],
        edges: [],
      },
    })

    const { nodeByKey, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodeByKey.value.get('S-01')?.title).toBe('First')
    expect(nodeByKey.value.get('missing')).toBeUndefined()
  })

  it('sets error ref when API returns an error', async () => {
    mockGET.mockResolvedValue({ error: { message: 'Not found' } })

    const { error, nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load DAG')
    expect(nodes.value).toHaveLength(0)
  })

  it('calls GET with correct path params', async () => {
    mockGET.mockResolvedValue({ data: { nodes: [], edges: [] } })

    const { isLoading } = withSetup(() => useDagLayout('my-project', 'my-epic'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(mockGET).toHaveBeenCalledWith('/projects/{projectId}/epics/{epicId}/dag', {
      params: { path: { projectId: 'my-project', epicId: 'my-epic' } },
    })
  })

  it('retry re-fetches data', async () => {
    mockGET.mockResolvedValueOnce({ error: { message: 'Server error' } })

    const { error, retry, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))
    await vi.waitFor(() => expect(isLoading.value).toBe(false))
    expect(error.value).not.toBeNull()

    mockGET.mockResolvedValueOnce({ data: { nodes: [], edges: [] } })
    await retry()

    expect(error.value).toBeNull()
    expect(mockGET).toHaveBeenCalledTimes(2)
  })
})

describe('seedContainerId', () => {
  it('is deterministic and 4 hex chars', () => {
    const a = seedContainerId('S-01')
    const b = seedContainerId('S-01')
    expect(a).toBe(b)
    expect(a).toMatch(/^[0-9a-f]{4}$/)
  })

  it('differs across keys (generally)', () => {
    expect(seedContainerId('S-01')).not.toBe(seedContainerId('S-99'))
  })
})
