import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useRelativeTime } from '../useRelativeTime'

vi.mock('@vueuse/core', () => ({
  useIntervalFn: vi.fn((callback: () => void) => {
    // Store the callback so tests can invoke it manually
    ;(globalThis as Record<string, unknown>).__intervalCallback = callback
    return { pause: vi.fn(), resume: vi.fn(), isActive: ref(true) }
  }),
}))

describe('useRelativeTime', () => {
  const NOW = new Date('2026-02-17T12:00:00Z').getTime()

  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(NOW)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns null for null input', () => {
    const result = useRelativeTime(null)
    expect(result.value).toBeNull()
  })

  it('returns null for undefined ref value', () => {
    const dateRef = ref<string | null>(null)
    const result = useRelativeTime(dateRef)
    expect(result.value).toBeNull()
  })

  it('returns null for invalid date string', () => {
    const result = useRelativeTime('not-a-date')
    expect(result.value).toBeNull()
  })

  it('returns "just now" for dates less than 60 seconds ago', () => {
    const date = new Date(NOW - 30 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('just now')
  })

  it('returns minutes ago for dates less than 60 minutes ago', () => {
    const date = new Date(NOW - 5 * 60 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('5m ago')
  })

  it('returns hours ago for dates less than 24 hours ago', () => {
    const date = new Date(NOW - 3 * 60 * 60 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('3h ago')
  })

  it('returns days ago for dates less than 7 days ago', () => {
    const date = new Date(NOW - 2 * 24 * 60 * 60 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('2d ago')
  })

  it('returns weeks ago for dates 7+ days ago', () => {
    const date = new Date(NOW - 14 * 24 * 60 * 60 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('2w ago')
  })

  it('accepts a Date object', () => {
    const date = new Date(NOW - 10 * 60 * 1000)
    const result = useRelativeTime(date)
    expect(result.value).toBe('10m ago')
  })

  it('reacts to ref changes', async () => {
    const dateRef = ref<string | null>(null)
    const result = useRelativeTime(dateRef)
    expect(result.value).toBeNull()

    dateRef.value = new Date(NOW - 2 * 60 * 1000).toISOString()
    await nextTick()
    expect(result.value).toBe('2m ago')
  })

  it('updates when interval callback fires', async () => {
    const date = new Date(NOW - 59 * 1000).toISOString()
    const result = useRelativeTime(date)
    expect(result.value).toBe('just now')

    // Advance time by 2 minutes
    vi.setSystemTime(NOW + 2 * 60 * 1000)
    const callback = (globalThis as Record<string, unknown>).__intervalCallback as () => void
    callback()
    await nextTick()

    expect(result.value).toBe('2m ago')
  })
})
