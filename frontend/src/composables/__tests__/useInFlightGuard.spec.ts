import { describe, it, expect, vi } from 'vitest'
import { useInFlightGuard } from '../useInFlightGuard'

/** A promise whose resolution/rejection is controlled by the test. */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useInFlightGuard', () => {
  it('runs the action once and returns its value', async () => {
    const { run, isBusy } = useInFlightGuard()
    const fn = vi.fn().mockResolvedValue('ok')

    expect(isBusy()).toBe(false)
    const result = await run(fn)

    expect(result).toBe('ok')
    expect(fn).toHaveBeenCalledTimes(1)
    expect(isBusy()).toBe(false)
  })

  it('marks the default key busy while the action is in flight (RG3)', async () => {
    const { run, isBusy } = useInFlightGuard()
    const d = deferred<string>()

    const first = run(() => d.promise)
    expect(isBusy()).toBe(true)

    d.resolve('done')
    await first
    expect(isBusy()).toBe(false)
  })

  it('ignores a second call while the first is in flight (RG2)', async () => {
    const { run } = useInFlightGuard()
    const d = deferred<string>()
    const fn = vi.fn(() => d.promise)

    const first = run(fn)
    const second = run(fn)

    // The re-entrant call is ignored without invoking fn again.
    expect(await second).toBeUndefined()
    expect(fn).toHaveBeenCalledTimes(1)

    d.resolve('done')
    expect(await first).toBe('done')
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('releases the key after success so the action is re-triggerable', async () => {
    const { run, isBusy } = useInFlightGuard()
    const fn = vi.fn().mockResolvedValue('ok')

    await run(fn)
    expect(isBusy()).toBe(false)

    await run(fn)
    expect(fn).toHaveBeenCalledTimes(2)
  })

  it('releases the key after a rejection and re-throws (RG4)', async () => {
    const { run, isBusy } = useInFlightGuard()
    const d = deferred<string>()
    const fn = vi.fn(() => d.promise)

    const first = run(fn)
    expect(isBusy()).toBe(true)

    d.reject(new Error('boom'))
    await expect(first).rejects.toThrow('boom')
    expect(isBusy()).toBe(false)

    // Re-triggerable after the error.
    const ok = vi.fn().mockResolvedValue('retry')
    expect(await run(ok)).toBe('retry')
  })

  it('tracks busy state per key so distinct items do not block each other', async () => {
    const { run, isBusy } = useInFlightGuard()
    const dA = deferred<string>()
    const dB = deferred<string>()
    const fnA = vi.fn(() => dA.promise)
    const fnB = vi.fn(() => dB.promise)

    const a = run(fnA, 'a')
    const b = run(fnB, 'b')

    expect(isBusy('a')).toBe(true)
    expect(isBusy('b')).toBe(true)

    dA.resolve('a-done')
    await a
    expect(isBusy('a')).toBe(false)
    // 'b' is still in flight and unaffected by 'a' settling.
    expect(isBusy('b')).toBe(true)

    dB.resolve('b-done')
    await b
    expect(isBusy('b')).toBe(false)
    expect(fnA).toHaveBeenCalledTimes(1)
    expect(fnB).toHaveBeenCalledTimes(1)
  })

  it('ignores a second click on the same key but allows other keys (RG2 per item)', async () => {
    const { run } = useInFlightGuard()
    const dA = deferred<string>()
    const fnA = vi.fn(() => dA.promise)
    const fnB = vi.fn().mockResolvedValue('b-done')

    const a1 = run(fnA, 'a')
    const a2 = run(fnA, 'a')
    expect(await a2).toBeUndefined()
    expect(fnA).toHaveBeenCalledTimes(1)

    // A different item deletes fine while 'a' is in flight.
    expect(await run(fnB, 'b')).toBe('b-done')
    expect(fnB).toHaveBeenCalledTimes(1)

    dA.resolve('a-done')
    expect(await a1).toBe('a-done')
  })
})
