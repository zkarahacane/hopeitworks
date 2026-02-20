import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'

const mockGet = vi.fn()
let capturedOnEvent: (eventName: string, data: unknown) => void

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

vi.mock('@/composables/useSSE', () => ({
  useSSE: vi.fn((_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: { value: 'open' }, close: vi.fn() }
  }),
}))

/** Wraps composable that calls onMounted in a simulated lifecycle */
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

const step1 = {
  id: 'step-1',
  run_id: 'run-1',
  step_name: 'dev-story',
  step_order: 2,
  action: 'agent_run',
  status: 'running',
  created_at: '2026-02-17T10:00:00Z',
}

const step2 = {
  id: 'step-2',
  run_id: 'run-1',
  step_name: 'code-review',
  step_order: 1,
  action: 'agent_run',
  status: 'pending',
  created_at: '2026-02-17T10:00:00Z',
}

describe('useRunProgress', () => {
  let useRunProgress: typeof import('../composables/useRunProgress').useRunProgress

  beforeEach(async () => {
    vi.clearAllMocks()
    mockGet.mockReset()
    const mod = await import('../composables/useRunProgress')
    useRunProgress = mod.useRunProgress
  })

  it('fetches steps sorted by step_order ascending', async () => {
    mockGet.mockResolvedValue({
      data: { steps: [step1, step2] },
      error: undefined,
    })

    const { steps, isLoading } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    expect(steps.value).toHaveLength(2)
    expect(steps.value[0]!.id).toBe('step-2')
    expect(steps.value[1]!.id).toBe('step-1')
    expect(isLoading.value).toBe(false)
  })

  it('sets error when API call fails', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
    })

    const { error, isLoading } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to load run steps')
    expect(isLoading.value).toBe(false)
  })

  it('patches step on run.step.updated event with matching run_id', async () => {
    mockGet.mockResolvedValue({
      data: { steps: [step2, step1] },
      error: undefined,
    })

    const { steps } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    capturedOnEvent('run.step.updated', {
      run_id: 'run-1',
      step: {
        id: 'step-1',
        status: 'completed',
        completed_at: '2026-02-17T10:05:00Z',
      },
    })

    expect(steps.value[1]!.status).toBe('completed')
    expect(steps.value[1]!.completed_at).toBe('2026-02-17T10:05:00Z')
  })

  it('ignores run.step.updated event with non-matching run_id', async () => {
    mockGet.mockResolvedValue({
      data: { steps: [step1] },
      error: undefined,
    })

    const { steps } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    capturedOnEvent('run.step.updated', {
      run_id: 'run-OTHER',
      step: { id: 'step-1', status: 'completed' },
    })

    expect(steps.value[0]!.status).toBe('running')
  })

  it('ignores run.step.updated event with unknown step id', async () => {
    mockGet.mockResolvedValue({
      data: { steps: [step1] },
      error: undefined,
    })

    const { steps } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    capturedOnEvent('run.step.updated', {
      run_id: 'run-1',
      step: { id: 'step-unknown', status: 'completed' },
    })

    expect(steps.value).toHaveLength(1)
    expect(steps.value[0]!.status).toBe('running')
  })

  it('ignores non-run.step.updated events', async () => {
    mockGet.mockResolvedValue({
      data: { steps: [step1] },
      error: undefined,
    })

    const { steps } = withSetup(() => useRunProgress('proj-1', 'run-1'))
    await flushPromises()

    capturedOnEvent('run.started', { run_id: 'run-1' })

    expect(steps.value[0]!.status).toBe('running')
  })
})
