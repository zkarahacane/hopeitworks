import { describe, it, expect } from 'vitest'
import { DEFAULT_MODEL_SUGGESTIONS } from '@/utils/models'

// Test the searchModels filtering logic extracted as a pure function
// (mirrors the component's searchModels implementation)
function searchModels(query: string): string[] {
  const q = query.toLowerCase()
  return q
    ? DEFAULT_MODEL_SUGGESTIONS.filter((m) => m.toLowerCase().includes(q))
    : [...DEFAULT_MODEL_SUGGESTIONS]
}

describe('AgentEditorLayout — searchModels', () => {
  it('returns all suggestions when query is empty', () => {
    const result = searchModels('')
    expect(result).toEqual(DEFAULT_MODEL_SUGGESTIONS)
    expect(result).not.toBe(DEFAULT_MODEL_SUGGESTIONS) // must be a copy
  })

  it('filters suggestions by partial query (case-insensitive)', () => {
    const result = searchModels('opus')
    expect(result).toEqual(['claude-opus-4-6'])
  })

  it('filters suggestions matching multiple models', () => {
    const result = searchModels('claude')
    expect(result).toEqual(DEFAULT_MODEL_SUGGESTIONS)
  })

  it('returns empty array when query matches nothing', () => {
    const result = searchModels('gpt-4')
    expect(result).toEqual([])
  })

  it('is case-insensitive', () => {
    const lower = searchModels('sonnet')
    const upper = searchModels('SONNET')
    expect(lower).toEqual(upper)
    expect(lower).toEqual(['claude-sonnet-4-6'])
  })
})
