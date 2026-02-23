import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'

let capturedOnEvent: (eventName: string, data: unknown) => void
const mockStatus = ref('open')
const mockClose = vi.fn()

vi.mock('@/composables/useSSE', () => ({
  useSSE: vi.fn((_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: mockStatus, close: mockClose }
  }),
}))

/** Helper to build an SSE event envelope matching the backend model.Event shape. */
function makeLogEvent(overrides: {
  run_id: string
  step_id?: string
  message: string
  timestamp: string
  raw_line?: string
}) {
  return {
    id: 'evt-1',
    project_id: 'proj-1',
    entity_type: 'log',
    entity_id: 'step-1',
    action: 'emitted',
    payload: {
      run_id: overrides.run_id,
      step_id: overrides.step_id,
      message: overrides.message,
      raw_line: overrides.raw_line ?? overrides.message,
      timestamp: overrides.timestamp,
    },
    created_at: overrides.timestamp,
  }
}

describe('useStepLogs', () => {
  let useStepLogs: typeof import('../composables/useStepLogs').useStepLogs

  beforeEach(async () => {
    vi.clearAllMocks()
    mockStatus.value = 'open'
    const mod = await import('../composables/useStepLogs')
    useStepLogs = mod.useStepLogs
  })

  it('appends log line when log.emitted event matches runId and stepId', () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        step_id: 'step-1',
        message: 'hello world',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(1)
    expect(lines.value[0]!.text).toBe('hello world')
    expect(lines.value[0]!.timestamp).toBeInstanceOf(Date)
  })

  it('ignores log events with different stepId', () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        step_id: 'step-OTHER',
        message: 'should be ignored',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(0)
  })

  it('ignores log events with different runId', () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-OTHER',
        step_id: 'step-1',
        message: 'should be ignored',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(0)
  })

  it('ignores non log.emitted events', () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent('run.started', { payload: { run_id: 'run-1' } })
    capturedOnEvent('step.completed', { payload: { run_id: 'run-1' } })

    expect(lines.value).toHaveLength(0)
  })

  it('clears logs when stepId changes', async () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        step_id: 'step-1',
        message: 'line for step 1',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(1)

    stepId.value = 'step-2'
    await nextTick()

    expect(lines.value).toHaveLength(0)
  })

  it('clearLogs resets lines to empty', () => {
    const stepId = ref<string | null>('step-1')
    const { lines, clearLogs } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        step_id: 'step-1',
        message: 'line 1',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(1)

    clearLogs()
    expect(lines.value).toHaveLength(0)
  })

  it('does not connect when stepId is null', () => {
    const stepId = ref<string | null>(null)
    const { sseStatus } = useStepLogs('proj-1', 'run-1', stepId)

    expect(sseStatus.value).toBe('closed')
  })

  it('prefers raw_line over message', () => {
    const stepId = ref<string | null>('step-1')
    const { lines } = useStepLogs('proj-1', 'run-1', stepId)

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        step_id: 'step-1',
        message: 'parsed',
        raw_line: '{"raw":"data"}',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value[0]!.text).toBe('{"raw":"data"}')
  })
})
