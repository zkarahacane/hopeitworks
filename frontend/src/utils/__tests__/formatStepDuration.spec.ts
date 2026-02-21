import { describe, it, expect } from 'vitest'
import { formatStepDuration } from '../formatStepDuration'

describe('formatStepDuration', () => {
  it('formats duration under 60 seconds as Xs', () => {
    expect(formatStepDuration('2026-01-01T10:00:00Z', '2026-01-01T10:00:42Z')).toBe('42s')
  })

  it('formats duration over 60 seconds as Xm Ys', () => {
    expect(formatStepDuration('2026-01-01T10:00:00Z', '2026-01-01T10:02:34Z')).toBe('2m 34s')
  })

  it('formats exactly 60 seconds as 1m 0s', () => {
    expect(formatStepDuration('2026-01-01T10:00:00Z', '2026-01-01T10:01:00Z')).toBe('1m 0s')
  })

  it('formats zero duration as 0s', () => {
    expect(formatStepDuration('2026-01-01T10:00:00Z', '2026-01-01T10:00:00Z')).toBe('0s')
  })
})
