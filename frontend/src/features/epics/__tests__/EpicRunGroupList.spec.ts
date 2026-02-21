import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import EpicRunGroupList from '../EpicRunGroupList.vue'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(stories: Array<{ story_id: string; story_key: string; run_id: string | null; group_index: number; status: string }>) {
  wrapper = mount(EpicRunGroupList as never, {
    props: { stories } as never,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('EpicRunGroupList', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('groups stories by group_index into separate rows', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'running' },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'pending' },
      { story_id: 's3', story_key: 'S-03', run_id: 'r3', group_index: 1, status: 'completed' },
    ])

    const text = wrapper.text()
    expect(text).toContain('Layer 0')
    expect(text).toContain('Layer 1')
    expect(text).toContain('2 stories')
    expect(text).toContain('1 stories')
  })

  it('shows info severity for group with running story', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'running' },
      { story_id: 's2', story_key: 'S-02', run_id: null, group_index: 0, status: 'pending' },
    ])

    const text = wrapper.text()
    expect(text).toContain('running')
  })

  it('shows success severity for group with all completed stories', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'completed' },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'completed' },
    ])

    const text = wrapper.text()
    expect(text).toContain('completed')
  })

  it('shows danger severity for group with a failed story', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 0, status: 'completed' },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'failed' },
    ])

    const text = wrapper.text()
    expect(text).toContain('failed')
  })

  it('shows secondary severity for group with only pending stories', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: null, group_index: 0, status: 'pending' },
    ])

    const text = wrapper.text()
    expect(text).toContain('pending')
  })

  it('renders empty message when no stories', () => {
    mountComponent([])

    expect(wrapper.text()).toContain('No execution layers')
  })

  it('sorts groups by index', () => {
    mountComponent([
      { story_id: 's1', story_key: 'S-01', run_id: 'r1', group_index: 2, status: 'pending' },
      { story_id: 's2', story_key: 'S-02', run_id: 'r2', group_index: 0, status: 'completed' },
      { story_id: 's3', story_key: 'S-03', run_id: 'r3', group_index: 1, status: 'running' },
    ])

    const text = wrapper.text()
    const layer0Idx = text.indexOf('Layer 0')
    const layer1Idx = text.indexOf('Layer 1')
    const layer2Idx = text.indexOf('Layer 2')
    expect(layer0Idx).toBeLessThan(layer1Idx)
    expect(layer1Idx).toBeLessThan(layer2Idx)
  })
})
