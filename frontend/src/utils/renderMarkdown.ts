import { marked } from 'marked'
import DOMPurify from 'dompurify'

/** Renders markdown string to sanitized HTML. Returns empty string for falsy input. */
export function renderMarkdown(input: string | undefined | null): string {
  if (!input) return ''
  const raw = marked.parse(input, { async: false }) as string
  return DOMPurify.sanitize(raw)
}
