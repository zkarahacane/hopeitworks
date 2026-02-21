import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'

const mockGET = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGET(...args),
  },
}))

// Must import after mock setup
const { useDagLayout } = await import('../composables/useDagLayout')

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
    mockGET.mockReset()
  })

  it('transforms API nodes to vue-flow Node[] with correct positions', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-01', layer: 0, title: 'First story', status: 'backlog' },
        ],
        edges: [],
      },
    })

    const { nodes, edges, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    // Wait for onMounted + async action
    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodes.value).toHaveLength(1)
    expect(nodes.value[0]).toEqual({
      id: 'S-01',
      type: 'story',
      position: { x: 0, y: 0 },
      data: { key: 'S-01', title: 'First story', status: 'backlog' },
    })
    expect(edges.value).toHaveLength(0)
  })

  it('assigns y positions for nodes in the same layer', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-01', layer: 0, title: 'First', status: 'backlog' },
          { key: 'S-02', layer: 0, title: 'Second', status: 'running' },
        ],
        edges: [],
      },
    })

    const { nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(nodes.value[0]!.position).toEqual({ x: 0, y: 0 })
    expect(nodes.value[1]!.position).toEqual({ x: 0, y: 120 })
  })

  it('transforms API edges to vue-flow Edge[]', async () => {
    mockGET.mockResolvedValue({
      data: {
        nodes: [
          { key: 'S-01', layer: 0, title: 'A', status: 'backlog' },
          { key: 'S-02', layer: 1, title: 'B', status: 'backlog' },
        ],
        edges: [{ source: 'S-01', target: 'S-02' }],
      },
    })

    const { edges, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(edges.value).toHaveLength(1)
    expect(edges.value[0]).toEqual({
      id: 'S-01-S-02',
      source: 'S-01',
      target: 'S-02',
    })
  })

  it('sets error ref when API returns an error', async () => {
    mockGET.mockResolvedValue({
      error: { message: 'Not found' },
    })

    const { error, nodes, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load DAG')
    expect(nodes.value).toHaveLength(0)
  })

  it('calls GET with correct path params', async () => {
    mockGET.mockResolvedValue({
      data: { nodes: [], edges: [] },
    })

    const { isLoading } = withSetup(() => useDagLayout('my-project', 'my-epic'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))

    expect(mockGET).toHaveBeenCalledWith('/projects/{projectId}/epics/{epicId}/dag', {
      params: { path: { projectId: 'my-project', epicId: 'my-epic' } },
    })
  })

  it('retry re-fetches data', async () => {
    mockGET.mockResolvedValueOnce({
      error: { message: 'Server error' },
    })

    const { error, retry, isLoading } = withSetup(() => useDagLayout('proj-1', 'epic-1'))

    await vi.waitFor(() => expect(isLoading.value).toBe(false))
    expect(error.value).not.toBeNull()

    mockGET.mockResolvedValueOnce({
      data: { nodes: [], edges: [] },
    })

    await retry()

    expect(error.value).toBeNull()
    expect(mockGET).toHaveBeenCalledTimes(2)
  })
})
