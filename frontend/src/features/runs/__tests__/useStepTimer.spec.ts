import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

/** Wraps composable that uses lifecycle hooks in a simulated Vue component */
function withSetup<T>(composable: () => T): { result: T; unmount: () => void } {
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
  const el = document.createElement('div')
  app.mount(el)
  return { result, unmount: () => app.unmount() }
}

describe('useStepTimer', () => {
  let useStepTimer: typeof import('../composables/useStepTimer').useStepTimer

  beforeEach(async () => {
    vi.useFakeTimers()
    const mod = await import('../composables/useStepTimer')
    useStepTimer = mod.useStepTimer
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns empty string when startedAt is undefined', () => {
    const { result } = withSetup(() => useStepTimer(undefined))
    expect(result.elapsed.value).toBe('')
  })

  it('formats elapsed time under 60 seconds as Xs elapsed', () => {
    const startTime = new Date('2026-01-01T10:00:00Z').getTime()
    vi.setSystemTime(startTime + 42000)

    const { result } = withSetup(() => useStepTimer('2026-01-01T10:00:00Z'))
    expect(result.elapsed.value).toBe('42s elapsed')
  })

  it('formats elapsed time over 60 seconds as Xm Ys elapsed', () => {
    const startTime = new Date('2026-01-01T10:00:00Z').getTime()
    vi.setSystemTime(startTime + 90000)

    const { result } = withSetup(() => useStepTimer('2026-01-01T10:00:00Z'))
    expect(result.elapsed.value).toBe('1m 30s elapsed')
  })

  it('formats elapsed time over 3600 seconds correctly', () => {
    const startTime = new Date('2026-01-01T10:00:00Z').getTime()
    vi.setSystemTime(startTime + 3600000)

    const { result } = withSetup(() => useStepTimer('2026-01-01T10:00:00Z'))
    expect(result.elapsed.value).toBe('60m 0s elapsed')
  })

  it('clears interval on unmount', () => {
    const clearIntervalSpy = vi.spyOn(globalThis, 'clearInterval')
    const startTime = new Date('2026-01-01T10:00:00Z').getTime()
    vi.setSystemTime(startTime)

    const { unmount } = withSetup(() => useStepTimer('2026-01-01T10:00:00Z'))
    unmount()

    expect(clearIntervalSpy).toHaveBeenCalled()
    clearIntervalSpy.mockRestore()
  })
})
