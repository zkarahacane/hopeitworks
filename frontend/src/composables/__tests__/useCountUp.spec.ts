import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { ref, nextTick } from 'vue'

// onBeforeUnmount is a no-op outside a component — stub it so the composable
// can run standalone in tests (matches the useSSE test pattern).
vi.mock('vue', async () => {
  const actual = await vi.importActual<typeof import('vue')>('vue')
  return { ...actual, onBeforeUnmount: vi.fn() }
})

import { useCountUp } from '../useCountUp'

/**
 * Deterministic rAF + clock control. requestAnimationFrame callbacks are queued
 * and flushed manually; performance.now is driven by a settable `clockNow`.
 */
let rafQueue: FrameRequestCallback[] = []
let clockNow = 0

function flushFrame() {
  const cbs = rafQueue
  rafQueue = []
  for (const cb of cbs) cb(clockNow)
}

beforeEach(() => {
  rafQueue = []
  clockNow = 0
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    rafQueue.push(cb)
    return rafQueue.length
  })
  vi.stubGlobal('cancelAnimationFrame', vi.fn())
  vi.stubGlobal('performance', { now: () => clockNow })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useCountUp', () => {
  it('starts at the initial target value', () => {
    const { current } = useCountUp(0)
    expect(current.value).toBe(0)
  })

  it('honours an explicit initial value', () => {
    const { current } = useCountUp(100, { initial: 25 })
    expect(current.value).toBe(25)
  })

  it('animates upward toward a new target and lands exactly on it', async () => {
    const target = ref(0)
    const { current } = useCountUp(target, { durationMs: 1000 })

    target.value = 10
    await nextTick()

    // first frame at t=0 → still ~0
    clockNow = 0
    flushFrame()
    expect(current.value).toBeCloseTo(0, 5)

    // halfway through duration → between start and target
    clockNow = 500
    flushFrame()
    expect(current.value).toBeGreaterThan(0)
    expect(current.value).toBeLessThan(10)

    // at/after full duration → exact target, animation stops
    clockNow = 1000
    flushFrame()
    expect(current.value).toBe(10)
    expect(rafQueue.length).toBe(0)
  })

  it('snaps instantly when durationMs is 0', async () => {
    const target = ref(0)
    const { current } = useCountUp(target, { durationMs: 0 })
    target.value = 42
    await nextTick()
    expect(current.value).toBe(42)
  })

  it('does not rewind on a lower target by default (live ticker semantics)', async () => {
    const target = ref(50)
    const { current } = useCountUp(target, { durationMs: 1000 })
    // climb to 50
    clockNow = 1000
    flushFrame()
    expect(current.value).toBe(50)

    // target drops → snaps, no backward animation
    target.value = 5
    await nextTick()
    expect(current.value).toBe(5)
    expect(rafQueue.length).toBe(0)
  })

  it('animates downward when allowDecrease is set', async () => {
    const target = ref(100)
    const { current } = useCountUp(target, { durationMs: 1000, allowDecrease: true })
    clockNow = 1000
    flushFrame()
    expect(current.value).toBe(100)

    target.value = 0
    await nextTick()
    clockNow = 1000 // start frame
    flushFrame()
    clockNow = 1500 // halfway
    flushFrame()
    expect(current.value).toBeLessThan(100)
    expect(current.value).toBeGreaterThan(0)
    clockNow = 2000 // done
    flushFrame()
    expect(current.value).toBe(0)
  })

  it('ignores non-finite targets', async () => {
    const target = ref(10)
    const { current } = useCountUp(target, { durationMs: 1000 })
    clockNow = 1000
    flushFrame()
    expect(current.value).toBe(10)

    target.value = NaN
    await nextTick()
    expect(current.value).toBe(10) // unchanged
  })

  it('accepts a getter as the target source', async () => {
    const base = ref(3)
    const { current } = useCountUp(() => base.value * 2, { durationMs: 0 })
    expect(current.value).toBe(6)
    base.value = 5
    await nextTick()
    expect(current.value).toBe(10)
  })
})
