import { describe, it, expect, vi, beforeEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

const mockGet = vi.fn()
const mockSSEClose = vi.fn()
let capturedOnEvent: ((eventName: string, data: unknown) => void) | null = null

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

vi.mock('@/composables/useSSE', () => ({
  useSSE: (_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: { value: 'open' }, close: mockSSEClose }
  },
}))

const { useEpicRunMonitor } = await import('../composables/useEpicRunMonitor')
const { useEpicRunStore } = await import('@/stores/epicRun')

function withSetup<T>(composable: () => T): { result: T; unmount: () => void } {
  let result!: T
  const Comp = defineComponent({
    setup() {
      result = composable()
      return () => h('div')
    },
  })
  const wrapper = mount(Comp)
  return { result, unmount: () => wrapper.unmount() }
}

function makeEpicRun() {
  return {
    id: 'run-1',
    epic_id: 'epic-1',
    project_id: 'proj-1',
    status: 'running' as const,
    stories: [
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'completed' as const },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'running' as const },
      { story_id: 's3', story_key: 'S-03', run_id: null, group_index: 1, status: 'pending' as const },
    ],
    created_at: '2026-02-15T10:00:00Z',
    completed_at: null,
  }
}

describe('useEpicRunMonitor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    capturedOnEvent = null
  })

  it('calls fetchEpicRun on mount', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    withSetup(() => useEpicRunMonitor('proj-1', 'run-1'))

    await vi.waitFor(() => expect(mockGet).toHaveBeenCalledTimes(1))
    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/epic-runs/{epicRunId}', {
      params: { path: { projectId: 'proj-1', epicRunId: 'run-1' } },
    })
  })

  it('resets store on unmount', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const { unmount } = withSetup(() => useEpicRunMonitor('proj-1', 'run-1'))

    await vi.waitFor(() => {
      const store = useEpicRunStore()
      expect(store.epicRun).not.toBeNull()
    })

    unmount()

    const store = useEpicRunStore()
    expect(store.epicRun).toBeNull()
  })

  it('produces correct VueFlow nodes from store stories', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    const { result } = withSetup(() => useEpicRunMonitor('proj-1', 'run-1'))

    await vi.waitFor(() => expect(result.nodes.value).toHaveLength(3))

    const nodes = result.nodes.value
    // Group 0: S-01 at y=0, S-02 at y=120
    expect(nodes[0]).toMatchObject({
      id: 'S-01',
      type: 'epicRunStatus',
      position: { x: 0, y: 0 },
      data: { key: 'S-01', status: 'completed', runId: 'r1' },
    })
    expect(nodes[1]).toMatchObject({
      id: 'S-02',
      type: 'epicRunStatus',
      position: { x: 0, y: 120 },
      data: { key: 'S-02', status: 'running', runId: 'r2' },
    })
    // Group 1: S-03 at x=250, y=0
    expect(nodes[2]).toMatchObject({
      id: 'S-03',
      type: 'epicRunStatus',
      position: { x: 250, y: 0 },
      data: { key: 'S-03', status: 'pending', runId: null },
    })
  })

  it('dispatches SSE events to store handleSSEEvent', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    withSetup(() => useEpicRunMonitor('proj-1', 'run-1'))

    await vi.waitFor(() => {
      const store = useEpicRunStore()
      expect(store.epicRun).not.toBeNull()
    })

    // Simulate an SSE event
    capturedOnEvent!('epic_run.story.completed', { story_id: 's2' })
    await nextTick()

    const store = useEpicRunStore()
    const story = store.epicRun!.stories.find((s) => s.story_id === 's2')
    expect(story!.status).toBe('completed')
  })

  it('ignores non-epic_run SSE events', async () => {
    mockGet.mockResolvedValue({ data: makeEpicRun(), error: undefined })

    withSetup(() => useEpicRunMonitor('proj-1', 'run-1'))

    await vi.waitFor(() => {
      const store = useEpicRunStore()
      expect(store.epicRun).not.toBeNull()
    })

    // This should not cause any changes
    capturedOnEvent!('run.completed', { run_id: 'r1' })
    await nextTick()

    const store = useEpicRunStore()
    // Stories should be unchanged
    expect(store.epicRun!.status).toBe('running')
  })
})
