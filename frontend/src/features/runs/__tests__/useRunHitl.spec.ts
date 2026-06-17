import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { useRunHitl } from '../composables/useRunHitl'
import type { RunWithSteps } from '../composables/useRunDetail'

const mockGet = vi.fn()
const mockPost = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
  },
}))

/** Mounts the composable inside a throwaway component so watchers run. */
function withSetup<T>(composable: () => T): T {
  let result!: T
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { createApp, defineComponent } = require('vue')
  const app = createApp(
    defineComponent({
      setup() {
        result = composable()
        return () => null
      },
    }),
  )
  app.mount(document.createElement('div'))
  return result
}

function runWith(steps: Partial<RunWithSteps['steps'][number]>[], status = 'paused'): RunWithSteps {
  return {
    id: 'run-1',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: status as RunWithSteps['status'],
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T10:00:00Z',
    steps: steps.map((s, i) => ({
      id: s.id ?? `step-${i}`,
      run_id: 'run-1',
      step_name: s.step_name ?? 'step',
      step_order: i,
      action: s.action ?? 'agent_run',
      status: s.status ?? 'pending',
      created_at: '2026-02-17T10:00:00Z',
      ...s,
    })),
  }
}

describe('useRunHitl', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockPost.mockReset()
  })

  it('finds the human gate step (awaiting) among the run steps', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-1', status: 'pending' }, error: undefined })
    const run = ref(
      runWith([
        { id: 's1', action: 'agent_run', status: 'completed' },
        { id: 's2', step_name: 'Approval gate', action: 'human', status: 'running' },
      ]),
    )
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()
    expect(hitl.gateStep.value?.id).toBe('s2')
  })

  it('fetches the HITL request for the gate step via /hitl-requests/by-step', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-99', status: 'pending' }, error: undefined })
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }]))
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()
    expect(mockGet).toHaveBeenCalledWith('/hitl-requests/by-step/{stepId}', {
      params: { path: { stepId: 's2' } },
    })
    expect(hitl.hitlRequest.value?.id).toBe('hitl-99')
  })

  it('reports isAtGate when the run is paused', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-1', status: 'pending' }, error: undefined })
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }], 'paused'))
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()
    expect(hitl.isAtGate.value).toBe(true)
  })

  it('approve() POSTs to the approve endpoint with the request id', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-1', status: 'pending' }, error: undefined })
    mockPost.mockResolvedValue({ data: { status: 'approved' }, error: undefined })
    const onResolved = vi.fn()
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }]))
    const hitl = withSetup(() => useRunHitl(run, onResolved))
    await flushPromises()

    await hitl.approve()
    expect(mockPost).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/approve', {
      params: { path: { hitlRequestId: 'hitl-1' } },
    })
    expect(hitl.hitlRequest.value?.status).toBe('approved')
    expect(onResolved).toHaveBeenCalled()
  })

  it('reject() POSTs to the reject endpoint with a reason', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-1', status: 'pending' }, error: undefined })
    mockPost.mockResolvedValue({ data: { status: 'rejected' }, error: undefined })
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }]))
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()

    await hitl.reject('nope')
    expect(mockPost).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/reject', {
      params: { path: { hitlRequestId: 'hitl-1' } },
      body: { reason: 'nope' },
    })
    expect(hitl.hitlRequest.value?.status).toBe('rejected')
  })

  it('requestChanges() rejects carrying a reason (no dedicated endpoint)', async () => {
    mockGet.mockResolvedValue({ data: { id: 'hitl-1', status: 'pending' }, error: undefined })
    mockPost.mockResolvedValue({ data: { status: 'rejected' }, error: undefined })
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }]))
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()

    await hitl.requestChanges('please fix lint')
    expect(mockPost).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/reject', {
      params: { path: { hitlRequestId: 'hitl-1' } },
      body: { reason: 'please fix lint' },
    })
  })

  it('does not act when there is no resolved HITL request id', async () => {
    mockGet.mockResolvedValue({ data: undefined, error: { error: { message: 'not found' } } })
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'running' }]))
    const hitl = withSetup(() => useRunHitl(run))
    await flushPromises()
    await hitl.approve()
    expect(mockPost).not.toHaveBeenCalled()
  })

  it('ignores completed human steps (gate already resolved)', async () => {
    const run = ref(runWith([{ id: 's2', action: 'human', status: 'completed' }], 'completed'))
    const hitl = withSetup(() => useRunHitl(run))
    await nextTick()
    await flushPromises()
    expect(hitl.gateStep.value).toBeNull()
  })
})
