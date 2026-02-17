import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../renderMarkdown'

describe('renderMarkdown', () => {
  it('returns empty string for null', () => {
    expect(renderMarkdown(null)).toBe('')
  })

  it('returns empty string for undefined', () => {
    expect(renderMarkdown(undefined)).toBe('')
  })

  it('returns empty string for empty string', () => {
    expect(renderMarkdown('')).toBe('')
  })

  it('converts **bold** to <strong>', () => {
    const result = renderMarkdown('**bold**')
    expect(result).toContain('<strong>bold</strong>')
  })

  it('converts `code` to <code>', () => {
    const result = renderMarkdown('`code`')
    expect(result).toContain('<code>code</code>')
  })

  it('converts markdown lists to <ul>/<li>', () => {
    const result = renderMarkdown('- item 1\n- item 2')
    expect(result).toContain('<ul>')
    expect(result).toContain('<li>item 1</li>')
    expect(result).toContain('<li>item 2</li>')
  })

  it('converts headings', () => {
    const result = renderMarkdown('## Heading')
    expect(result).toContain('<h2')
    expect(result).toContain('Heading')
  })

  it('strips <script> tags (XSS prevention)', () => {
    const result = renderMarkdown('<script>alert("xss")</script>')
    expect(result).not.toContain('<script>')
    expect(result).not.toContain('alert')
  })

  it('strips onerror attributes (XSS prevention)', () => {
    const result = renderMarkdown('<img src=x onerror="alert(1)">')
    expect(result).not.toContain('onerror')
  })
})
