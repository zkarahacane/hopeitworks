import { describe, it, expect, vi, beforeEach } from 'vitest'

let capturedOnEvent: (eventName: string, data: unknown) => void
const mockStatus = { value: 'open' }

vi.mock('@/composables/useSSE', () => ({
  useSSE: vi.fn((_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: mockStatus }
  }),
}))

describe('useRunLogs', () => {
  let useRunLogs: typeof import('../composables/useRunLogs').useRunLogs

  beforeEach(async () => {
    vi.clearAllMocks()
    const mod = await import('../composables/useRunLogs')
    useRunLogs = mod.useRunLogs
  })

  it('appends log line when log.emitted event matches runId', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent('log.emitted', {
      run_id: 'run-1',
      line: 'hello world',
      timestamp: '2026-02-17T10:30:00Z',
    })

    expect(lines.value).toHaveLength(1)
    expect(lines.value[0]!.text).toBe('hello world')
    expect(lines.value[0]!.timestamp).toBeInstanceOf(Date)
  })

  it('ignores log.emitted events with different runId', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent('log.emitted', {
      run_id: 'run-OTHER',
      line: 'should be ignored',
      timestamp: '2026-02-17T10:30:00Z',
    })

    expect(lines.value).toHaveLength(0)
  })

  it('ignores events that are not log.emitted', () => {
    const { lines } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent('run.started', { run_id: 'run-1' })
    capturedOnEvent('step.completed', { run_id: 'run-1' })

    expect(lines.value).toHaveLength(0)
  })

  it('clearLogs resets lines to empty', () => {
    const { lines, clearLogs } = useRunLogs('proj-1', 'run-1')

    capturedOnEvent('log.emitted', {
      run_id: 'run-1',
      line: 'line 1',
      timestamp: '2026-02-17T10:30:00Z',
    })
    capturedOnEvent('log.emitted', {
      run_id: 'run-1',
      line: 'line 2',
      timestamp: '2026-02-17T10:30:01Z',
    })

    expect(lines.value).toHaveLength(2)

    clearLogs()

    expect(lines.value).toHaveLength(0)
  })

  it('exposes sseStatus from useSSE', () => {
    const { sseStatus } = useRunLogs('proj-1', 'run-1')

    expect(sseStatus.value).toBe('open')
  })
})
