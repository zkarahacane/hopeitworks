import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import KanbanBoard, { type BoardStage } from '../KanbanBoard.vue'
import type { Story } from '@/stores/stories'

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/api/client', () => ({
  apiClient: { GET: vi.fn(), PUT: vi.fn(), POST: vi.fn() },
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeStory(overrides: Partial<Story> = {}): Story {
  return {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Test Story',
    status: 'backlog',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
    ...overrides,
  }
}

const STAGES: BoardStage[] = [
  { id: 'dev', name: 'Dev', transition: 'auto' },
  { id: 'review', name: 'Review', transition: 'manual' },
  { id: 'qa', name: 'QA', transition: 'gate' },
]

function mountBoard(stories: Story[], stages: BoardStage[] = STAGES) {
  // Détail view exercises the stage-column placement; default localStorage is macro.
  // useLocalStorage stores plain strings verbatim (no JSON quoting).
  localStorage.setItem('board.viewMode', 'detail')
  wrapper = mount(KanbanBoard, {
    props: { stories, selectedId: null, projectId: 'p1', stages },
    global: { plugins: [PrimeVue, createPinia()] },
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  localStorage.clear()
})

/**
 * Returns the story keys placed in the détail column whose header label matches.
 * Columns are header (label) + a card list of role="button" cards (aria-label
 * "Story: <key> - <title>").
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function keysInColumn(w: VueWrapper<any>, label: string): string[] {
  const column = w
    .findAll('.flex.flex-col.gap-2.min-w-\\[240px\\]')
    .find((col) => col.find('span').text() === label)
  if (!column) return []
  return column
    .findAll('[role="button"]')
    .map((card) => card.attributes('aria-label') ?? '')
    .map((aria) => aria.replace(/^Story:\s*/, '').split(' - ')[0] ?? '')
}

describe('KanbanBoard détail placement (#300)', () => {
  it('RG2/RG3: a running story with no current_stage lands in the entry stage, never Backlog', () => {
    const story = makeStory({
      key: 'S-05',
      status: 'running',
      current_stage: null,
      latest_run: { id: 'r1', status: 'running', current_step: null },
    })
    const w = mountBoard([story])

    expect(keysInColumn(w, 'Backlog')).not.toContain('S-05')
    expect(keysInColumn(w, 'Dev')).toContain('S-05') // Dev = first/entry stage
  })

  it('RG1: a running story with a current_stage stays in that stage lane', () => {
    const story = makeStory({
      key: 'S-02',
      status: 'running',
      current_stage: 'Review',
      latest_run: { id: 'r1', status: 'running', current_step: null },
    })
    const w = mountBoard([story])

    expect(keysInColumn(w, 'Review')).toContain('S-02')
    expect(keysInColumn(w, 'Backlog')).not.toContain('S-02')
  })

  it('RG4: a backlog story with no stage stays in Backlog', () => {
    const story = makeStory({ key: 'S-03', status: 'backlog', current_stage: null })
    const w = mountBoard([story])

    expect(keysInColumn(w, 'Backlog')).toContain('S-03')
    expect(keysInColumn(w, 'Dev')).not.toContain('S-03')
  })

  it('RG3: a running story with a stale current_stage (out of pipeline) lands in the entry stage, never Backlog', () => {
    const story = makeStory({
      key: 'S-06',
      status: 'running',
      current_stage: 'Ghost', // a stage name no longer in the pipeline (renamed/removed)
      latest_run: { id: 'r1', status: 'running', current_step: null },
    })
    const w = mountBoard([story])

    expect(keysInColumn(w, 'Backlog')).not.toContain('S-06')
    expect(keysInColumn(w, 'Dev')).toContain('S-06') // Dev = first/entry stage
  })

  it('RG4: a non-running story with a stale/unknown current_stage falls back to Backlog', () => {
    const story = makeStory({
      key: 'S-07',
      status: 'backlog',
      current_stage: 'Ghost', // unknown column key, but not running
    })
    const w = mountBoard([story])

    expect(keysInColumn(w, 'Backlog')).toContain('S-07')
    expect(keysInColumn(w, 'Dev')).not.toContain('S-07')
  })

  it('degenerate: an empty pipeline routes a running+NULL story to Done, never Backlog', () => {
    const story = makeStory({
      key: 'S-05',
      status: 'running',
      current_stage: null,
      latest_run: { id: 'r1', status: 'running', current_step: null },
    })
    const w = mountBoard([story], [])

    expect(keysInColumn(w, 'Backlog')).not.toContain('S-05')
    expect(keysInColumn(w, 'Done')).toContain('S-05')
  })
})
