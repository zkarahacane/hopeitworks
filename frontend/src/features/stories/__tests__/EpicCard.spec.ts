import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import EpicCard from '../EpicCard.vue'
import type { Epic } from '@/stores/epics'

const baseEpic: Epic = {
  id: 'e1',
  project_id: 'p1',
  name: 'Project Foundation',
  description: 'Admin can create an account and configure a project',
  status: 'in_progress',
  story_counts: { total: 12, backlog: 5, running: 3, done: 3, failed: 1 },
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

function mountComponent(epic: Epic = baseEpic) {
  return mount(EpicCard, {
    props: { epic },
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('EpicCard', () => {
  it('renders the epic name', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('Project Foundation')
  })

  it('renders the epic description', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('Admin can create an account')
  })

  it('renders story count tags for non-zero statuses', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('3 Done')
    expect(wrapper.text()).toContain('3 Running')
    expect(wrapper.text()).toContain('5 Backlog')
    expect(wrapper.text()).toContain('1 Failed')
  })

  it('does not render tags for zero-count statuses', () => {
    const epic: Epic = {
      ...baseEpic,
      story_counts: { total: 5, backlog: 5, running: 0, done: 0, failed: 0 },
    }
    const wrapper = mountComponent(epic)
    expect(wrapper.text()).toContain('5 Backlog')
    expect(wrapper.text()).not.toContain('Done')
    expect(wrapper.text()).not.toContain('Running')
    expect(wrapper.text()).not.toContain('Failed')
  })

  it('renders the total story count badge', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('12')
  })

  it('renders the epic status tag', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('In Progress')
  })

  it('renders Completed status for completed epics', () => {
    const epic: Epic = {
      ...baseEpic,
      status: 'completed',
      story_counts: { total: 12, backlog: 0, running: 0, done: 12, failed: 0 },
    }
    const wrapper = mountComponent(epic)
    expect(wrapper.text()).toContain('Completed')
  })

  it('renders progress percentage', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('25%')
  })

  it('renders 0% progress when no stories are done', () => {
    const epic: Epic = {
      ...baseEpic,
      story_counts: { total: 5, backlog: 5, running: 0, done: 0, failed: 0 },
    }
    const wrapper = mountComponent(epic)
    expect(wrapper.text()).toContain('0%')
  })

  it('renders 100% progress when all stories are done', () => {
    const epic: Epic = {
      ...baseEpic,
      story_counts: { total: 10, backlog: 0, running: 0, done: 10, failed: 0 },
    }
    const wrapper = mountComponent(epic)
    expect(wrapper.text()).toContain('100%')
  })

  it('emits click event with the epic when card is clicked', async () => {
    const wrapper = mountComponent()
    await wrapper.find('.p-card').trigger('click')
    expect(wrapper.emitted('click')).toBeDefined()
    expect(wrapper.emitted('click')![0]).toEqual([baseEpic])
  })
})
