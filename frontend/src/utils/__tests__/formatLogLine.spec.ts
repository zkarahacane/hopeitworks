import { describe, it, expect } from 'vitest'
import { formatLogLine } from '../formatLogLine'

describe('formatLogLine', () => {
  it('converts ANSI green code to HTML span with color style', () => {
    const raw = '\u001b[32mOK\u001b[0m'
    const timestamp = new Date('2026-01-01T10:05:03Z')
    const result = formatLogLine(raw, timestamp)

    expect(result).toContain('<span')
    expect(result).not.toContain('\u001b')
    expect(result).toContain('OK')
  })

  it('prefixes with HH:MM:SS timestamp', () => {
    const raw = 'hello world'
    const timestamp = new Date('2026-01-01T10:05:03Z')
    const result = formatLogLine(raw, timestamp)

    // Timestamp is rendered in local time via getHours/getMinutes/getSeconds
    const hh = String(timestamp.getHours()).padStart(2, '0')
    const mm = String(timestamp.getMinutes()).padStart(2, '0')
    const ss = String(timestamp.getSeconds()).padStart(2, '0')
    expect(result).toContain(`<span class="log-ts">${hh}:${mm}:${ss}</span>`)
  })

  it('handles empty string input', () => {
    const timestamp = new Date('2026-01-01T10:05:03Z')
    const result = formatLogLine('', timestamp)

    expect(result).toContain('<span class="log-ts">')
  })

  it('escapes HTML entities in raw text', () => {
    const raw = '<script>alert("xss")</script>'
    const timestamp = new Date('2026-01-01T10:05:03Z')
    const result = formatLogLine(raw, timestamp)

    expect(result).not.toContain('<script>')
    expect(result).toContain('&lt;script&gt;')
  })
})
