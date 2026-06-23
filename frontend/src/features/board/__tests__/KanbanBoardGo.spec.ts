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

function mountBoard(stories: Story[]) {
  // Force détail view so manual-stage placement is exercised; default localStorage is macro.
  localStorage.setItem('board.viewMode', JSON.stringify('detail'))
  wrapper = mount(KanbanBoard, {
    props: { stories, selectedId: null, projectId: 'p1', stages: STAGES },
    global: {
      plugins: [PrimeVue, createPinia()],
    },
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  localStorage.clear()
})

describe('KanbanBoard Go affordance', () => {
  it('shows a Go button on a Backlog card and emits action=launch', async () => {
    const story = makeStory({ status: 'backlog' })
    const w = mountBoard([story])

    const go = w.find('[data-testid="board-go-button"]')
    expect(go.exists()).toBe(true)
    expect(go.text()).toContain('Go')

    await go.trigger('click')
    const emitted = w.emitted('go')
    expect(emitted).toBeTruthy()
    expect(emitted![0]![0]).toEqual({ story, action: 'launch' })
  })

  it('shows "Go · start stage" on a card idle in a manual stage and emits action=start-stage', async () => {
    const story = makeStory({
      status: 'running',
      current_stage: 'Review', // manual stage per STAGES
      latest_run: { id: 'r1', status: 'paused', current_step: null },
    })
    const w = mountBoard([story])

    const go = w.find('[data-testid="board-go-button"]')
    expect(go.exists()).toBe(true)
    expect(go.text()).toContain('start stage')

    await go.trigger('click')
    const emitted = w.emitted('go')
    expect(emitted![0]![0]).toEqual({ story, action: 'start-stage' })
  })

  it('does NOT show Go while a segment is running', () => {
    const story = makeStory({
      status: 'running',
      current_stage: 'Dev',
      latest_run: {
        id: 'r1',
        status: 'running',
        current_step: { id: 'st1', name: 'impl', action_type: 'agent_run', status: 'running', index: 0, total: 2 },
      },
    })
    const w = mountBoard([story])
    expect(w.find('[data-testid="board-go-button"]').exists()).toBe(false)
  })

  it('does NOT show Go at a gate (waiting_approval)', () => {
    const story = makeStory({
      status: 'running',
      current_stage: 'QA',
      latest_run: {
        id: 'r1',
        status: 'running',
        current_step: { id: 'st1', name: 'review', action_type: 'hitl_gate', status: 'waiting_approval', index: 1, total: 2 },
      },
    })
    const w = mountBoard([story])
    expect(w.find('[data-testid="board-go-button"]').exists()).toBe(false)
  })

  it('does NOT treat a paused run in an AUTO stage as manual-idle', () => {
    // Defensive: only manual stages get the start-stage Go; a paused run elsewhere
    // should not surface it.
    const story = makeStory({
      status: 'running',
      current_stage: 'Dev', // auto
      latest_run: { id: 'r1', status: 'paused', current_step: null },
    })
    const w = mountBoard([story])
    expect(w.find('[data-testid="board-go-button"]').exists()).toBe(false)
  })
})
