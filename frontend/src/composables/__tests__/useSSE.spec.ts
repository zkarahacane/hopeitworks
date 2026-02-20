import { describe, it, expect, vi, beforeEach } from 'vitest'

type EventSourceHandler = ((event: Event | MessageEvent) => void) | null
type EventListenerEntry = { type: string; handler: EventListener }

interface MockESInstance {
  onopen: EventSourceHandler
  onerror: EventSourceHandler
  onmessage: EventSourceHandler
  close: ReturnType<typeof vi.fn>
  addEventListener: ReturnType<typeof vi.fn>
  listeners: EventListenerEntry[]
}

let mockInstance: MockESInstance

class MockEventSource {
  onopen: EventSourceHandler = null
  onerror: EventSourceHandler = null
  onmessage: EventSourceHandler = null
  close = vi.fn()
  addEventListener = vi.fn((type: string, handler: EventListener) => {
    mockInstance.listeners.push({ type, handler })
  })
  listeners: EventListenerEntry[] = []

  constructor(public url: string) {
    mockInstance = this
  }
}

const mockOnBeforeUnmount = vi.fn()

vi.mock('vue', async () => {
  const actual = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    onBeforeUnmount: (fn: () => void) => mockOnBeforeUnmount(fn),
  }
})

vi.stubGlobal('EventSource', MockEventSource)

describe('useSSE', () => {
  let useSSE: typeof import('../useSSE').useSSE

  beforeEach(async () => {
    vi.clearAllMocks()
    const mod = await import('../useSSE')
    useSSE = mod.useSSE
  })

  it('opens EventSource with correct URL', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    expect(mockInstance).toBeDefined()
    expect((mockInstance as unknown as MockEventSource).url).toBe(
      '/api/v1/events/stream?project_id=proj-123',
    )
  })

  it('starts with connecting status', () => {
    const onEvent = vi.fn()
    const { status } = useSSE('proj-123', onEvent)

    expect(status.value).toBe('connecting')
  })

  it('sets status to open when onopen fires', () => {
    const onEvent = vi.fn()
    const { status } = useSSE('proj-123', onEvent)

    mockInstance.onopen!(new Event('open'))

    expect(status.value).toBe('open')
  })

  it('sets status to error when onerror fires', () => {
    const onEvent = vi.fn()
    const { status } = useSSE('proj-123', onEvent)

    mockInstance.onerror!(new Event('error'))

    expect(status.value).toBe('error')
  })

  it('dispatches parsed JSON on message event', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    mockInstance.onmessage!(
      new MessageEvent('message', { data: '{"foo":"bar"}' }),
    )

    expect(onEvent).toHaveBeenCalledWith('message', { foo: 'bar' })
  })

  it('ignores malformed JSON on message event', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    mockInstance.onmessage!(
      new MessageEvent('message', { data: 'not-json' }),
    )

    expect(onEvent).not.toHaveBeenCalled()
  })

  it('registers listeners for all known named events', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    const registeredTypes = mockInstance.listeners.map((l) => l.type)
    expect(registeredTypes).toContain('run.started')
    expect(registeredTypes).toContain('run.completed')
    expect(registeredTypes).toContain('step.completed')
    expect(registeredTypes).toContain('step.failed')
    expect(registeredTypes).toContain('log.emitted')
    expect(registeredTypes).toContain('hitl.pending')
    expect(registeredTypes).toContain('hitl.approved')
    expect(registeredTypes).toContain('hitl.rejected')
  })

  it('dispatches hitl.approved event with parsed data', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    const listener = mockInstance.listeners.find(
      (l) => l.type === 'hitl.approved',
    )
    listener!.handler(
      new MessageEvent('hitl.approved', {
        data: '{"hitl_request_id":"hr-1"}',
      }),
    )

    expect(onEvent).toHaveBeenCalledWith('hitl.approved', {
      hitl_request_id: 'hr-1',
    })
  })

  it('dispatches hitl.rejected event with parsed data', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    const listener = mockInstance.listeners.find(
      (l) => l.type === 'hitl.rejected',
    )
    listener!.handler(
      new MessageEvent('hitl.rejected', {
        data: '{"hitl_request_id":"hr-2"}',
      }),
    )

    expect(onEvent).toHaveBeenCalledWith('hitl.rejected', {
      hitl_request_id: 'hr-2',
    })
  })

  it('dispatches named events with parsed data', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    const logListener = mockInstance.listeners.find(
      (l) => l.type === 'log.emitted',
    )
    logListener!.handler(
      new MessageEvent('log.emitted', {
        data: '{"run_id":"r1","line":"hello"}',
      }),
    )

    expect(onEvent).toHaveBeenCalledWith('log.emitted', {
      run_id: 'r1',
      line: 'hello',
    })
  })

  it('registers onBeforeUnmount callback that closes EventSource', () => {
    const onEvent = vi.fn()
    useSSE('proj-123', onEvent)

    expect(mockOnBeforeUnmount).toHaveBeenCalledTimes(1)

    const unmountCallback = mockOnBeforeUnmount.mock.calls[0]![0]
    unmountCallback()

    expect(mockInstance.close).toHaveBeenCalledTimes(1)
  })

  it('sets status to closed when close() is called', () => {
    const onEvent = vi.fn()
    const { status, close } = useSSE('proj-123', onEvent)

    close()

    expect(status.value).toBe('closed')
    expect(mockInstance.close).toHaveBeenCalled()
  })
})
