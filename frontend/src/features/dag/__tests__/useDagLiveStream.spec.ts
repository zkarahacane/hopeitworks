import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'

let capturedOnEvent: ((eventName: string, data: unknown) => void) | null = null
const mockClose = vi.fn()

vi.mock('@/composables/useSSE', () => ({
  useSSE: (_projectId: string, onEvent: (eventName: string, data: unknown) => void) => {
    capturedOnEvent = onEvent
    return { status: { value: 'open' }, close: mockClose }
  },
}))

const { useDagLiveStream } = await import('../composables/useDagLiveStream')
const { useRuntimeStream } = await import('@/stores/runtimeStream')

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

describe('useDagLiveStream', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    capturedOnEvent = null
    mockClose.mockReset()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('forwards SSE events into the runtime stream store', () => {
    const stream = useRuntimeStream()
    const { unmount } = withSetup(() => useDagLiveStream('proj-1'))

    expect(capturedOnEvent).toBeTypeOf('function')
    capturedOnEvent!('run.started', { run_id: 'r1', started_at: '2026-06-17T10:00:00Z' })

    expect(stream.runSignal('r1')?.status).toBe('running')
    unmount()
  })

  it('advances the clock via tick on an interval', () => {
    const stream = useRuntimeStream()
    const tickSpy = vi.spyOn(stream, 'tick')
    const { unmount } = withSetup(() => useDagLiveStream('proj-1'))

    // onMounted tick (1) + each interval fire.
    expect(tickSpy).toHaveBeenCalledTimes(1)
    vi.advanceTimersByTime(3000)
    expect(tickSpy).toHaveBeenCalledTimes(4)
    unmount()
  })

  it('stops ticking after unmount', () => {
    const stream = useRuntimeStream()
    const tickSpy = vi.spyOn(stream, 'tick')
    const { unmount } = withSetup(() => useDagLiveStream('proj-1'))
    vi.advanceTimersByTime(1000)
    const callsBefore = tickSpy.mock.calls.length
    unmount()
    vi.advanceTimersByTime(5000)
    expect(tickSpy.mock.calls.length).toBe(callsBefore)
  })

  it('exposes the live SSE status', () => {
    const { result, unmount } = withSetup(() => useDagLiveStream('proj-1'))
    expect(result.sseStatus.value).toBe('open')
    unmount()
  })
})
