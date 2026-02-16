import { describe, it, expect } from 'vitest'
import { formatRelativeDate, formatDate } from '../formatDate'

describe('formatRelativeDate', () => {
  it('returns a relative time string for a valid ISO date', () => {
    const oneHourAgo = new Date(Date.now() - 60 * 60 * 1000).toISOString()
    const result = formatRelativeDate(oneHourAgo)
    expect(result).toContain('ago')
  })

  it('includes "ago" suffix', () => {
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
    const result = formatRelativeDate(yesterday)
    expect(result).toMatch(/ago$/)
  })

  it('returns "Invalid date" for unparseable input', () => {
    expect(formatRelativeDate('not-a-date')).toBe('Invalid date')
  })

  it('returns "Invalid date" for empty string', () => {
    expect(formatRelativeDate('')).toBe('Invalid date')
  })
})

describe('formatDate', () => {
  it('formats an ISO date string as "MMM d, yyyy"', () => {
    expect(formatDate('2026-02-15T10:30:00Z')).toBe('Feb 15, 2026')
  })

  it('formats another date correctly', () => {
    expect(formatDate('2025-12-01T00:00:00Z')).toBe('Dec 1, 2025')
  })

  it('returns "Invalid date" for unparseable input', () => {
    expect(formatDate('garbage')).toBe('Invalid date')
  })

  it('returns "Invalid date" for empty string', () => {
    expect(formatDate('')).toBe('Invalid date')
  })
})
