/**
 * DashboardView — unit tests.
 *
 * Focus: the dedupedRuns logic (fix #5) — verifies that when multiple runs exist
 * for the same story, only the most recent one surfaces in the list.
 *
 * We test the pure deduplicate logic as a standalone function rather than mounting
 * the full component (which requires SSE, Pinia stores, etc.) to keep tests fast
 * and deterministic.
 */
import { describe, it, expect } from 'vitest'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

// ── Replicated dedup logic (mirrors DashboardView.vue `dedupedRuns` computed) ──

function dedupRuns(runs: RunSummary[]): RunSummary[] {
  const byStory = new Map<string, RunSummary>()
  for (const run of runs) {
    const existing = byStory.get(run.story_id)
    if (!existing || new Date(run.created_at) > new Date(existing.created_at)) {
      byStory.set(run.story_id, run)
    }
  }
  return [...byStory.values()].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
  )
}

function makeRun(overrides: Partial<RunSummary> = {}): RunSummary {
  return {
    id: overrides.id ?? 'run-1',
    project_id: overrides.project_id ?? 'proj-1',
    story_id: overrides.story_id ?? 'story-1',
    status: overrides.status ?? 'completed',
    progress: overrides.progress ?? 100,
    created_at: overrides.created_at ?? '2026-06-17T10:00:00Z',
    updated_at: overrides.updated_at ?? '2026-06-17T10:00:00Z',
    project_name: overrides.project_name,
    story_key: overrides.story_key ?? 'S-01',
    started_at: overrides.started_at,
    completed_at: overrides.completed_at,
  }
}

describe('DashboardView — dedupedRuns (fix #5)', () => {
  it('returns an empty array when there are no runs', () => {
    expect(dedupRuns([])).toEqual([])
  })

  it('keeps a single run unchanged', () => {
    const run = makeRun()
    expect(dedupRuns([run])).toEqual([run])
  })

  it('deduplicates runs with the same story_id, keeping the most recent', () => {
    const older = makeRun({
      id: 'run-old',
      story_id: 'S-01',
      story_key: 'S-01',
      created_at: '2026-06-17T08:00:00Z',
      status: 'failed',
    })
    const newer = makeRun({
      id: 'run-new',
      story_id: 'S-01',
      story_key: 'S-01',
      created_at: '2026-06-17T10:00:00Z',
      status: 'running',
    })

    const result = dedupRuns([older, newer])
    expect(result).toHaveLength(1)
    expect(result[0]!.id).toBe('run-new')
    expect(result[0]!.status).toBe('running')
  })

  it('keeps 6 identical story_id rows as a single entry (original bug #5 scenario)', () => {
    // The old dashboard showed 6 duplicate S-01 rows — one per retry attempt.
    const duplicates = Array.from({ length: 6 }, (_, i) =>
      makeRun({
        id: `run-${i}`,
        story_id: 'S-01',
        story_key: 'S-01',
        created_at: `2026-06-17T${String(i).padStart(2, '0')}:00:00Z`,
        status: i === 5 ? 'running' : 'failed',
      }),
    )

    const result = dedupRuns(duplicates)
    expect(result).toHaveLength(1)
    // The most recent (index 5, hour 05) is the retained one
    expect(result[0]!.id).toBe('run-5')
  })

  it('preserves distinct runs from different stories', () => {
    const runA = makeRun({ id: 'run-a', story_id: 'S-01' })
    const runB = makeRun({ id: 'run-b', story_id: 'S-02' })
    const runC = makeRun({ id: 'run-c', story_id: 'S-03' })

    const result = dedupRuns([runA, runB, runC])
    expect(result).toHaveLength(3)
    expect(result.map((r) => r.id)).toEqual(
      expect.arrayContaining(['run-a', 'run-b', 'run-c']),
    )
  })

  it('sorts results by created_at descending after deduplication', () => {
    const runs = [
      makeRun({ id: 'r1', story_id: 'S-01', created_at: '2026-06-17T08:00:00Z' }),
      makeRun({ id: 'r2', story_id: 'S-02', created_at: '2026-06-17T12:00:00Z' }),
      makeRun({ id: 'r3', story_id: 'S-03', created_at: '2026-06-17T10:00:00Z' }),
    ]

    const result = dedupRuns(runs)
    expect(result.map((r) => r.id)).toEqual(['r2', 'r3', 'r1'])
  })

  it('handles mixed scenarios: 3 stories with retries, returns only most recent per story', () => {
    const runsInput = [
      makeRun({ id: 'a1', story_id: 'S-A', created_at: '2026-06-17T08:00:00Z', status: 'failed' }),
      makeRun({ id: 'a2', story_id: 'S-A', created_at: '2026-06-17T09:00:00Z', status: 'running' }),
      makeRun({ id: 'b1', story_id: 'S-B', created_at: '2026-06-17T07:00:00Z', status: 'completed' }),
      makeRun({ id: 'c1', story_id: 'S-C', created_at: '2026-06-17T11:00:00Z', status: 'paused' }),
      makeRun({ id: 'c2', story_id: 'S-C', created_at: '2026-06-17T06:00:00Z', status: 'failed' }),
    ]

    const result = dedupRuns(runsInput)
    expect(result).toHaveLength(3)

    const ids = result.map((r) => r.id)
    expect(ids).toContain('a2')   // newer of S-A
    expect(ids).toContain('b1')   // only one of S-B
    expect(ids).toContain('c1')   // newer of S-C
    expect(ids).not.toContain('a1')
    expect(ids).not.toContain('c2')
  })
})
