import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ProjectCard from '../ProjectCard.vue'
import type { Project } from '@/stores/projects'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

const baseProject: Project = {
  id: 'p1',
  name: 'Alpha Project',
  git_provider: 'github',
  agent_runtime: 'docker',
  owner_id: 'u1',
  created_at: '2026-02-10T10:00:00Z',
  updated_at: '2026-02-10T10:00:00Z',
}

function mountCard(props: { storyCount?: number } = {}) {
  wrapper = mount(ProjectCard, {
    props: {
      project: baseProject,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('ProjectCard story count (#289)', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('RG1: renders "5 stories" when storyCount is 5', () => {
    mountCard({ storyCount: 5 })
    expect(wrapper.text()).toContain('5 stories')
    expect(wrapper.text()).not.toContain('no stories')
  })

  it('RG3: renders the singular "1 story" when storyCount is exactly 1', () => {
    mountCard({ storyCount: 1 })
    expect(wrapper.text()).toContain('1 story')
    // Must not pluralise to "1 stories".
    expect(wrapper.text()).not.toContain('1 stories')
  })

  it('RG2: renders the distinct "no stories" empty state when storyCount is 0', () => {
    mountCard({ storyCount: 0 })
    expect(wrapper.text()).toContain('no stories')
  })

  it('RG2: renders "no stories" when storyCount is absent (data missing)', () => {
    mountCard({})
    expect(wrapper.text()).toContain('no stories')
  })

  it('renders the plural "2 stories" for a small plural count', () => {
    mountCard({ storyCount: 2 })
    expect(wrapper.text()).toContain('2 stories')
  })
})
