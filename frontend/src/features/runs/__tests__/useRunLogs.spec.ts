import { describe, it, expect, vi, beforeEach } from 'vitest'

let capturedOnEvent: (eventName: string, data: unknown) => void
const mockStatus = { value: 'open' }

vi.mock('@/composables/useSSE', () => ({
  useSSE: vi.fn((_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: mockStatus }
  }),
}))

/** Helper to build an SSE event envelope matching the backend model.Event shape. */
function makeLogEvent(overrides: {
  run_id: string
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
      message: overrides.message,
      raw_line: overrides.raw_line ?? overrides.message,
      timestamp: overrides.timestamp,
    },
    created_at: overrides.timestamp,
  }
}

describe('useRunLogs', () => {
  let useRunLogs: typeof import('../composables/useRunLogs').useRunLogs

  beforeEach(async () => {
    vi.clearAllMocks()
    const mod = await import('../composables/useRunLogs')
    useRunLogs = mod.useRunLogs
  })

  it('appends log line when log.emitted event matches runId', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        message: 'hello world',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(1)
    expect(lines.value[0]!.text).toBe('hello world')
    expect(lines.value[0]!.timestamp).toBeInstanceOf(Date)
  })

  it('prefers raw_line over message when available', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        message: 'parsed message',
        raw_line: '{"type":"result","message":"parsed message"}',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(1)
    expect(lines.value[0]!.text).toBe('{"type":"result","message":"parsed message"}')
  })

  it('ignores log.emitted events with different runId', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-OTHER',
        message: 'should be ignored',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )

    expect(lines.value).toHaveLength(0)
  })

  it('ignores events that are not log.emitted', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent('run.started', { payload: { run_id: 'run-1' } })
    capturedOnEvent('step.completed', { payload: { run_id: 'run-1' } })

    expect(lines.value).toHaveLength(0)
  })

  it('clearLogs resets lines to empty', () => {
    const { lines, clearLogs } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        message: 'line 1',
        timestamp: '2026-02-17T10:30:00Z',
      }),
    )
    capturedOnEvent(
      'log.emitted',
      makeLogEvent({
        run_id: 'run-1',
        message: 'line 2',
        timestamp: '2026-02-17T10:30:01Z',
      }),
    )

    expect(lines.value).toHaveLength(2)

    clearLogs()

    expect(lines.value).toHaveLength(0)
  })

  it('exposes sseStatus from useSSE', () => {
    const { sseStatus } = useRunLogs('proj-1', 'run-1')

    expect(sseStatus.value).toBe('open')
  })
})
