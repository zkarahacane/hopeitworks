import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import HITLPendingTable from '../HITLPendingTable.vue'
import type { HITLPendingItem } from '@/stores/hitl'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

const sampleItems: HITLPendingItem[] = [
  {
    hitlRequestId: 'hr-1',
    runId: 'r-1',
    stepId: 's-1',
    projectId: 'p-1',
    projectName: 'Project Alpha',
    storyKey: 'S-01',
    storyTitle: 'Implement login page',
    prUrl: 'https://github.com/org/repo/pull/1',
    pendingSince: '2026-02-17T10:00:00Z',
  },
  {
    hitlRequestId: 'hr-2',
    runId: 'r-2',
    stepId: 's-2',
    projectId: 'p-2',
    projectName: 'Project Beta',
    storyKey: 'S-02',
    storyTitle: 'Add dashboard',
    prUrl: null,
    pendingSince: '2026-02-17T11:00:00Z',
  },
]

function mountComponent(props: { items: HITLPendingItem[]; loading: boolean }) {
  wrapper = mount(HITLPendingTable, {
    props,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('HITLPendingTable', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders empty state when items array is empty', () => {
    mountComponent({ items: [], loading: false })
    expect(wrapper.text()).toContain('No pending approvals')
  })

  it('renders rows for each item', () => {
    mountComponent({ items: sampleItems, loading: false })
    expect(wrapper.text()).toContain('S-01')
    expect(wrapper.text()).toContain('S-02')
    expect(wrapper.text()).toContain('Implement login page')
    expect(wrapper.text()).toContain('Add dashboard')
  })

  it('renders project name', () => {
    mountComponent({ items: sampleItems, loading: false })
    expect(wrapper.text()).toContain('Project Alpha')
    expect(wrapper.text()).toContain('Project Beta')
  })

  it('renders PR link when prUrl is provided', () => {
    mountComponent({ items: sampleItems, loading: false })
    const link = wrapper.find('a[href="https://github.com/org/repo/pull/1"]')
    expect(link.exists()).toBe(true)
    expect(link.attributes('target')).toBe('_blank')
    expect(link.text()).toBe('View PR')
  })

  it('renders dash when prUrl is null', () => {
    mountComponent({ items: [sampleItems[1]!], loading: false })
    expect(wrapper.find('a').exists()).toBe(false)
  })

  it('emits review event when Review button is clicked', async () => {
    mountComponent({ items: sampleItems, loading: false })
    const reviewButtons = wrapper.findAll('button').filter((b) => b.text().includes('Review'))
    expect(reviewButtons.length).toBeGreaterThan(0)
    await reviewButtons[0]!.trigger('click')
    expect(wrapper.emitted('review')).toBeTruthy()
    expect(wrapper.emitted('review')![0]).toEqual([sampleItems[0]])
  })
})
