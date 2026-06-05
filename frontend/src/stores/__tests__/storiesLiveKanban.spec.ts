import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useStoriesStore, boardColumn } from '../stories'
import type { Story } from '../stories'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
    POST: vi.fn(),
  },
}))

// ── Fixture factories ───────────────────────────────────────────────────────

function makeStory(overrides: Partial<Story> = {}): Story {
  return {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Test story',
    status: 'backlog',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

// ── boardColumn() ────────────────────────────────────────────────────────────

describe('boardColumn()', () => {
  it('maps status=done to "done"', () => {
    expect(boardColumn(makeStory({ status: 'done' }))).toBe('done')
  })

  it('maps status=failed to "failed"', () => {
    expect(boardColumn(makeStory({ status: 'failed' }))).toBe('failed')
  })

  it('maps status=backlog to "backlog"', () => {
    expect(boardColumn(makeStory({ status: 'backlog' }))).toBe('backlog')
  })

  it('maps status=running with no latest_run to "in_progress"', () => {
    expect(boardColumn(makeStory({ status: 'running' }))).toBe('in_progress')
  })

  it('maps status=running with current_step.status=running to "in_progress"', () => {
    const story = makeStory({
      status: 'running',
      latest_run: {
        id: 'r1',
        status: 'running',
        current_step: { id: 'step1', name: 'impl', action_type: 'agent_run', status: 'running', index: 0, total: 3 },
      },
    })
    expect(boardColumn(story)).toBe('in_progress')
  })

  it('maps status=running with current_step.status=waiting_approval to "blocked"', () => {
    const story = makeStory({
      status: 'running',
      latest_run: {
        id: 'r1',
        status: 'running',
        current_step: { id: 'step1', name: 'review', action_type: 'hitl_gate', status: 'waiting_approval', index: 1, total: 3 },
      },
    })
    expect(boardColumn(story)).toBe('blocked')
  })

  it('maps status=running with null current_step to "in_progress"', () => {
    const story = makeStory({
      status: 'running',
      latest_run: { id: 'r1', status: 'running', current_step: null },
    })
    expect(boardColumn(story)).toBe('in_progress')
  })

  it('maps status=running with null latest_run to "in_progress"', () => {
    const story = makeStory({ status: 'running', latest_run: null })
    expect(boardColumn(story)).toBe('in_progress')
  })
})

// ── handleSSEEvent() ─────────────────────────────────────────────────────────

describe('useStoriesStore.handleSSEEvent()', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  function storeWithStories(stories: Story[]) {
    const store = useStoriesStore()
    store.items = stories
    return store
  }

  // ── story.status_updated ──────────────────────────────────────────────────

  describe('story.status_updated', () => {
    it('updates story status in items', () => {
      const store = storeWithStories([makeStory({ id: 's1', status: 'backlog' })])
      store.handleSSEEvent('story.status_updated', { story_id: 's1', status: 'running' })
      expect(store.items[0]!.status).toBe('running')
    })

    it('ignores event if story not in store', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      store.handleSSEEvent('story.status_updated', { story_id: 'unknown', status: 'running' })
      expect(store.items[0]!.status).toBe('backlog')
    })
  })

  // ── run.* events ──────────────────────────────────────────────────────────

  describe('run.started', () => {
    it('sets latest_run.status to "started" and clears current_step', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          status: 'running',
          latest_run: { id: 'r-old', status: 'completed', current_step: { id: 'step1', name: 'impl', action_type: 'agent_run', status: 'completed', index: 2, total: 3 } },
        }),
      ])
      store.handleSSEEvent('run.started', { story_id: 's1', run_id: 'r-new' })
      const run = store.items[0]!.latest_run!
      expect(run.id).toBe('r-new')
      expect(run.status).toBe('started')
      expect(run.current_step).toBeNull()
    })

    it('ignores event if story_id missing', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      const before = JSON.stringify(store.items)
      store.handleSSEEvent('run.started', { run_id: 'r1' })
      expect(JSON.stringify(store.items)).toBe(before)
    })

    it('ignores event if story not in store', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      store.handleSSEEvent('run.started', { story_id: 'unknown', run_id: 'r1' })
      expect(store.items[0]!.latest_run).toBeUndefined()
    })
  })

  describe('run.completed', () => {
    it('sets latest_run.status to "completed"', () => {
      const store = storeWithStories([
        makeStory({ id: 's1', latest_run: { id: 'r1', status: 'running' } }),
      ])
      store.handleSSEEvent('run.completed', { story_id: 's1', run_id: 'r1' })
      expect(store.items[0]!.latest_run!.status).toBe('completed')
    })
  })

  describe('run.failed', () => {
    it('sets latest_run.status to "failed"', () => {
      const store = storeWithStories([
        makeStory({ id: 's1', latest_run: { id: 'r1', status: 'running' } }),
      ])
      store.handleSSEEvent('run.failed', { story_id: 's1', run_id: 'r1' })
      expect(store.items[0]!.latest_run!.status).toBe('failed')
    })
  })

  describe('run.cancelled', () => {
    it('sets latest_run.status to "cancelled"', () => {
      const store = storeWithStories([
        makeStory({ id: 's1', latest_run: { id: 'r1', status: 'running' } }),
      ])
      store.handleSSEEvent('run.cancelled', { story_id: 's1', run_id: 'r1' })
      expect(store.items[0]!.latest_run!.status).toBe('cancelled')
    })
  })

  // ── step.* events ─────────────────────────────────────────────────────────

  describe('step.started', () => {
    it('sets current_step with payload data', () => {
      const store = storeWithStories([
        makeStory({ id: 's1', latest_run: { id: 'r1', status: 'running' } }),
      ])
      store.handleSSEEvent('step.started', {
        story_id: 's1',
        run_id: 'r1',
        step_id: 'step1',
        name: 'implement',
        action_type: 'agent_run',
        status: 'running',
        index: 0,
        total: 4,
      })
      const step = store.items[0]!.latest_run!.current_step!
      expect(step.id).toBe('step1')
      expect(step.name).toBe('implement')
      expect(step.action_type).toBe('agent_run')
      expect(step.status).toBe('running')
      expect(step.index).toBe(0)
      expect(step.total).toBe(4)
    })

    it('ignores if story not in store', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      store.handleSSEEvent('step.started', { story_id: 'unknown', step_id: 'step1', name: 'impl', action_type: 'agent_run', status: 'running', index: 0, total: 3 })
      expect(store.items[0]!.latest_run).toBeUndefined()
    })

    it('ignores if story_id missing', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      const before = JSON.stringify(store.items)
      store.handleSSEEvent('step.started', { run_id: 'r1', step_id: 'step1' })
      expect(JSON.stringify(store.items)).toBe(before)
    })
  })

  describe('step.completed', () => {
    it('updates current_step status to completed', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          latest_run: {
            id: 'r1',
            status: 'running',
            current_step: { id: 'step1', name: 'implement', action_type: 'agent_run', status: 'running', index: 0, total: 3 },
          },
        }),
      ])
      store.handleSSEEvent('step.completed', {
        story_id: 's1',
        step_id: 'step1',
        name: 'implement',
        action_type: 'agent_run',
        status: 'completed',
        index: 0,
        total: 3,
      })
      expect(store.items[0]!.latest_run!.current_step!.status).toBe('completed')
    })
  })

  describe('step.failed', () => {
    it('updates current_step status to failed', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          latest_run: {
            id: 'r1',
            status: 'running',
            current_step: { id: 'step1', name: 'implement', action_type: 'agent_run', status: 'running', index: 0, total: 3 },
          },
        }),
      ])
      store.handleSSEEvent('step.failed', {
        story_id: 's1',
        step_id: 'step1',
        name: 'implement',
        action_type: 'agent_run',
        status: 'failed',
        index: 0,
        total: 3,
      })
      expect(store.items[0]!.latest_run!.current_step!.status).toBe('failed')
    })
  })

  describe('step.cancelled', () => {
    it('clears current_step (sets to null)', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          latest_run: {
            id: 'r1',
            status: 'running',
            current_step: { id: 'step1', name: 'implement', action_type: 'agent_run', status: 'running', index: 0, total: 3 },
          },
        }),
      ])
      store.handleSSEEvent('step.cancelled', { story_id: 's1', step_id: 'step1' })
      expect(store.items[0]!.latest_run!.current_step).toBeNull()
    })
  })

  // ── hitl_gate.pending ─────────────────────────────────────────────────────

  describe('hitl_gate.pending', () => {
    it('sets current_step.status to waiting_approval', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          status: 'running',
          latest_run: {
            id: 'r1',
            status: 'running',
            current_step: { id: 'gate1', name: 'code-review', action_type: 'hitl_gate', status: 'running', index: 1, total: 3 },
          },
        }),
      ])
      store.handleSSEEvent('hitl_gate.pending', { story_id: 's1' })
      expect(store.items[0]!.latest_run!.current_step!.status).toBe('waiting_approval')
    })

    it('ignores if story has no latest_run', () => {
      const store = storeWithStories([makeStory({ id: 's1', status: 'running', latest_run: null })])
      // Should not throw
      store.handleSSEEvent('hitl_gate.pending', { story_id: 's1' })
      expect(store.items[0]!.latest_run).toBeNull()
    })

    it('ignores if story_id not in store', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      store.handleSSEEvent('hitl_gate.pending', { story_id: 'unknown' })
      expect(store.items[0]!.latest_run).toBeUndefined()
    })

    it('ignores if story_id missing', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      const before = JSON.stringify(store.items)
      store.handleSSEEvent('hitl_gate.pending', { run_id: 'r1' })
      expect(JSON.stringify(store.items)).toBe(before)
    })
  })

  // ── unknown events ────────────────────────────────────────────────────────

  describe('unknown event names', () => {
    it('ignores unrecognised event names without throwing', () => {
      const store = storeWithStories([makeStory({ id: 's1' })])
      const before = JSON.stringify(store.items)
      store.handleSSEEvent('log.emitted', { story_id: 's1', line: 'some log' })
      expect(JSON.stringify(store.items)).toBe(before)
    })
  })

  // ── end-to-end column derivation via SSE ──────────────────────────────────

  describe('SSE → boardColumn integration', () => {
    it('story moves from backlog to in_progress after story.status_updated', () => {
      const store = storeWithStories([makeStory({ id: 's1', status: 'backlog' })])
      expect(boardColumn(store.items[0]!)).toBe('backlog')

      store.handleSSEEvent('story.status_updated', { story_id: 's1', status: 'running' })
      expect(boardColumn(store.items[0]!)).toBe('in_progress')
    })

    it('story moves to blocked after hitl_gate.pending', () => {
      const store = storeWithStories([
        makeStory({
          id: 's1',
          status: 'running',
          latest_run: {
            id: 'r1',
            status: 'running',
            current_step: { id: 'gate1', name: 'review', action_type: 'hitl_gate', status: 'running', index: 1, total: 3 },
          },
        }),
      ])
      expect(boardColumn(store.items[0]!)).toBe('in_progress')

      store.handleSSEEvent('hitl_gate.pending', { story_id: 's1' })
      expect(boardColumn(store.items[0]!)).toBe('blocked')
    })

    it('story moves to done after story.status_updated with status=done', () => {
      const store = storeWithStories([
        makeStory({ id: 's1', status: 'running', latest_run: { id: 'r1', status: 'running' } }),
      ])
      store.handleSSEEvent('story.status_updated', { story_id: 's1', status: 'done' })
      expect(boardColumn(store.items[0]!)).toBe('done')
    })
  })
})
