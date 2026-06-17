import { describe, it, expect, vi, afterEach } from 'vitest'
import { formatDuration, formatDurationSeconds } from '../formatDuration'

describe('formatDuration', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns "--" when startedAt is not provided', () => {
    expect(formatDuration()).toBe('--')
    expect(formatDuration(null)).toBe('--')
    expect(formatDuration(undefined)).toBe('--')
  })

  it('formats completed duration as mm:ss', () => {
    expect(
      formatDuration('2026-01-01T10:00:00Z', '2026-01-01T10:01:30Z'),
    ).toBe('01:30')
  })

  it('formats zero duration', () => {
    expect(
      formatDuration('2026-01-01T10:00:00Z', '2026-01-01T10:00:00Z'),
    ).toBe('00:00')
  })

  it('formats longer durations correctly', () => {
    expect(
      formatDuration('2026-01-01T10:00:00Z', '2026-01-01T10:12:45Z'),
    ).toBe('12:45')
  })

  it('uses Date.now() for running steps (no completedAt)', () => {
    const now = new Date('2026-01-01T10:02:00Z').getTime()
    vi.spyOn(Date, 'now').mockReturnValue(now)
    expect(formatDuration('2026-01-01T10:00:00Z')).toBe('02:00')
  })

  it('returns "--" when startedAt is null and completedAt is provided', () => {
    expect(formatDuration(null, '2026-01-01T10:01:30Z')).toBe('--')
  })
})

describe('formatDurationSeconds', () => {
  it('formats a seconds count as mm:ss', () => {
    expect(formatDurationSeconds(0)).toBe('00:00')
    expect(formatDurationSeconds(90)).toBe('01:30')
    expect(formatDurationSeconds(222)).toBe('03:42')
    expect(formatDurationSeconds(765)).toBe('12:45')
  })

  it('clamps negatives to 00:00 and floors fractional seconds', () => {
    expect(formatDurationSeconds(-5)).toBe('00:00')
    expect(formatDurationSeconds(90.9)).toBe('01:30')
  })
})
