import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import StoryDetailPanel from '../StoryDetailPanel.vue'
import type { Story } from '@/stores/stories'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
    POST: vi.fn(),
  },
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
    objective: 'Build the feature',
    acceptance_criteria: '- Item 1\n- Item 2',
    target_files: ['src/foo.ts', 'src/bar.ts'],
    depends_on: ['S-02', 'S-03'],
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
    ...overrides,
  }
}

function mountComponent(props: {
  story: Story | null
  allStories?: Story[]
  projectId?: string
  showLaunchButton?: boolean
}) {
  wrapper = mount(StoryDetailPanel, {
    props,
    global: {
      plugins: [PrimeVue, createPinia()],
      stubs: {
        RunLaunchButton: {
          template: '<button data-testid="launch-btn">Launch</button>',
          emits: ['launchClick'],
        },
        StoryEditorForm: {
          template: '<div data-testid="editor-form">Editor Form</div>',
        },
      },
    },
  })
  return wrapper
}

describe('StoryDetailPanel', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('empty state', () => {
    it('renders empty state when story is null', () => {
      mountComponent({ story: null })
      expect(wrapper.text()).toContain('Select a story to view details')
      expect(wrapper.find('.pi-book').exists()).toBe(true)
    })
  })

  describe('story data rendering', () => {
    it('renders title, key, and status badge when story is provided', () => {
      mountComponent({ story: makeStory() })
      expect(wrapper.text()).toContain('S-01')
      expect(wrapper.text()).toContain('Test Story')
      expect(wrapper.text()).toContain('backlog')
    })

    it('renders scope badge with correct value when scope is set', () => {
      mountComponent({ story: makeStory({ scope: 'backend' }) })
      expect(wrapper.text()).toContain('backend')
    })

    it('does not render scope badge when scope is absent', () => {
      mountComponent({ story: makeStory({ scope: undefined }) })
      // Tag component for scope should not be present
      const tags = wrapper.findAll('.p-tag')
      const scopeTag = tags.find((t) => ['backend', 'frontend', 'shared'].includes(t.text()))
      expect(scopeTag).toBeUndefined()
    })
  })

  describe('markdown rendering', () => {
    it('renders objective as HTML (not plain text)', () => {
      mountComponent({ story: makeStory({ objective: '**bold objective**' }) })
      const proseContent = wrapper.findAll('.prose-content')
      expect(proseContent.length).toBeGreaterThan(0)
      expect(proseContent[0]!.html()).toContain('<strong>bold objective</strong>')
    })

    it('renders acceptance_criteria as HTML with list items', () => {
      mountComponent({ story: makeStory({ acceptance_criteria: '- Item A\n- Item B' }) })
      const proseContent = wrapper.findAll('.prose-content')
      expect(proseContent.length).toBeGreaterThan(1)
      expect(proseContent[1]!.html()).toContain('<ul>')
      expect(proseContent[1]!.html()).toContain('<li>')
    })
  })

  describe('target files', () => {
    it('renders target files as monospace list', () => {
      mountComponent({ story: makeStory() })
      expect(wrapper.text()).toContain('src/foo.ts')
      expect(wrapper.text()).toContain('src/bar.ts')
    })

    it('does not render target files section when empty', () => {
      mountComponent({ story: makeStory({ target_files: [] }) })
      expect(wrapper.text()).not.toContain('Target Files')
    })
  })

  describe('clickable dependencies', () => {
    it('emits select-dependency with correct storyId when dependency key is clicked', async () => {
      const allStories = [
        makeStory({ id: 's2', key: 'S-02' }),
        makeStory({ id: 's3', key: 'S-03' }),
      ]
      mountComponent({ story: makeStory(), allStories })

      const buttons = wrapper.findAll('button[type="button"]')
      const s02Button = buttons.find((b) => b.text() === 'S-02')
      expect(s02Button).toBeDefined()

      await s02Button!.trigger('click')
      expect(wrapper.emitted('select-dependency')).toHaveLength(1)
      expect(wrapper.emitted('select-dependency')![0]).toEqual(['s2'])
    })

    it('does not emit when clicked key is not found in allStories', async () => {
      mountComponent({ story: makeStory(), allStories: [] })

      const buttons = wrapper.findAll('button[type="button"]')
      const s02Button = buttons.find((b) => b.text() === 'S-02')
      await s02Button!.trigger('click')

      expect(wrapper.emitted('select-dependency')).toBeUndefined()
    })

    it('does not emit when allStories is not provided', async () => {
      mountComponent({ story: makeStory() })

      const buttons = wrapper.findAll('button[type="button"]')
      const s02Button = buttons.find((b) => b.text() === 'S-02')
      await s02Button!.trigger('click')

      expect(wrapper.emitted('select-dependency')).toBeUndefined()
    })
  })

  describe('RunLaunchButton integration', () => {
    it('renders RunLaunchButton when showLaunchButton is true', () => {
      mountComponent({ story: makeStory(), showLaunchButton: true })
      expect(wrapper.find('[data-testid="launch-btn"]').exists()).toBe(true)
    })

    it('does not render RunLaunchButton when showLaunchButton is false', () => {
      mountComponent({ story: makeStory(), showLaunchButton: false })
      expect(wrapper.find('[data-testid="launch-btn"]').exists()).toBe(false)
    })

    it('does not render RunLaunchButton when showLaunchButton is not provided', () => {
      mountComponent({ story: makeStory() })
      expect(wrapper.find('[data-testid="launch-btn"]').exists()).toBe(false)
    })
  })
})
