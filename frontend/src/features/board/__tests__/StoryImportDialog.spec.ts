import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { h, defineComponent, ref } from 'vue'
import PrimeVue from 'primevue/config'
import StoryImportDialog from '../StoryImportDialog.vue'
import type { PlanningImportResult } from '@/composables/usePlanningImport'

vi.mock('@/api/client', () => ({
  apiClient: { GET: vi.fn(), PUT: vi.fn(), POST: vi.fn() },
}))

// ── Mutable state shared with the usePlanningImport mock ──────────────────────
const mockState = {
  source: ref<'markdown' | 'github_projects'>('markdown'),
  canSubmit: ref(false),
  fileContent: ref<string | null>(null),
  fileName: ref<string | null>(null),
  parsedPreview: ref<{ key: string; title: string; scope?: string; valid: boolean; error?: string }[]>([]),
  fileError: ref<string | null>(null),
  projectUrl: ref(''),
  statusField: ref('Status'),
  doneOptions: ref<string[]>([]),
  epicIssueType: ref('Epic'),
  result: ref<PlanningImportResult | null>(null),
  committed: ref(false),
  apiError: ref<string | null>(null),
  isLoading: ref(false),
}

const mockSelectFile = vi.fn()
const mockPreview = vi.fn()
const mockCommit = vi.fn()
const mockReset = vi.fn()

vi.mock('@/composables/usePlanningImport', () => ({
  usePlanningImport: () => ({
    ...mockState,
    selectFile: mockSelectFile,
    preview: mockPreview,
    commit: mockCommit,
    reset: mockReset,
    parseMarkdownPreview: vi.fn(),
    buildBody: vi.fn(),
  }),
}))

/** Stub Dialog to render inline instead of teleporting. */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header', 'class'],
  emits: ['update:visible'],
  setup(props, { slots }) {
    return () => {
      if (!props.visible) return null
      return h('div', { class: 'p-dialog' }, [
        h('div', { class: 'p-dialog-header' }, props.header),
        h('div', { class: 'p-dialog-content' }, slots.default?.()),
        h('div', { class: 'p-dialog-footer' }, slots.footer?.()),
      ])
    }
  },
})

function emptyResult(overrides: Partial<PlanningImportResult> = {}): PlanningImportResult {
  return {
    source: 'markdown',
    dry_run: true,
    epics_created: 0,
    epics_updated: 0,
    stories_created: 0,
    stories_updated: 0,
    skipped: 0,
    locked: 0,
    failed: 0,
    errors: [],
    warnings: [],
    items: [],
    ...overrides,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

beforeEach(() => {
  mockState.source.value = 'markdown'
  mockState.canSubmit.value = false
  mockState.fileContent.value = null
  mockState.fileName.value = null
  mockState.parsedPreview.value = []
  mockState.fileError.value = null
  mockState.projectUrl.value = ''
  mockState.statusField.value = 'Status'
  mockState.doneOptions.value = []
  mockState.epicIssueType.value = 'Epic'
  mockState.result.value = null
  mockState.committed.value = false
  mockState.apiError.value = null
  mockState.isLoading.value = false
  vi.clearAllMocks()
})

function mountComponent(props: Partial<{ visible: boolean; projectId: string }> = {}) {
  wrapper = mount(StoryImportDialog, {
    props: { visible: true, projectId: 'p1', ...props },
    global: {
      plugins: [PrimeVue],
      stubs: { Dialog: DialogStub },
    },
  })
  return wrapper
}

describe('StoryImportDialog', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders the source picker', () => {
    mountComponent()
    expect(wrapper.find('[data-testid="source-picker"]').exists()).toBe(true)
  })

  describe('source switching', () => {
    it('shows the markdown drop-zone by default', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="markdown-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="drop-zone"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="github-panel"]').exists()).toBe(false)
    })

    it('shows the GitHub form when source is github_projects', async () => {
      mockState.source.value = 'github_projects'
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="github-panel"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="github-project-url"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="github-status-field"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="github-done-options"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="markdown-panel"]').exists()).toBe(false)
    })
  })

  describe('preview table binding', () => {
    it('renders the decision table + tally bound to result.items', async () => {
      mockState.result.value = emptyResult({
        stories_created: 1,
        locked: 1,
        items: [
          { key: 'S-1', kind: 'story', action: 'create', mapped_status: 'backlog', reason: 'new' },
          { key: 'S-2', kind: 'story', action: 'lock', mapped_status: 'backlog', reason: 'running' },
        ],
      })
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="preview-result-table"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="preview-tally"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('S-1')
      expect(wrapper.text()).toContain('S-2')
      expect(wrapper.text()).toContain('1 locked')
    })

    it('does not render the decision table before a preview', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="preview-result-table"]').exists()).toBe(false)
    })
  })

  describe('actions', () => {
    it('disables Preview / Import until canSubmit', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="preview-button"]').attributes('disabled')).toBeDefined()
      expect(wrapper.find('[data-testid="import-button"]').attributes('disabled')).toBeDefined()
    })

    it('calls preview(projectId) on the Preview button', async () => {
      mockState.canSubmit.value = true
      mockState.fileContent.value = '# x'
      mountComponent()
      await wrapper.find('[data-testid="preview-button"]').trigger('click')
      expect(mockPreview).toHaveBeenCalledWith('p1')
    })

    it('commits and emits imported when Import succeeds', async () => {
      mockState.canSubmit.value = true
      mockState.fileContent.value = '# x'
      mockCommit.mockResolvedValue(emptyResult({ dry_run: false, stories_created: 1 }))
      mountComponent()
      await wrapper.find('[data-testid="import-button"]').trigger('click')
      expect(mockCommit).toHaveBeenCalledWith('p1')
      await wrapper.vm.$nextTick()
      expect(wrapper.emitted('imported')).toBeTruthy()
    })

    it('does NOT emit imported when commit fails (returns null)', async () => {
      mockState.canSubmit.value = true
      mockState.fileContent.value = '# x'
      mockCommit.mockResolvedValue(null)
      mountComponent()
      await wrapper.find('[data-testid="import-button"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.emitted('imported')).toBeFalsy()
    })
  })

  describe('committed footer', () => {
    beforeEach(() => {
      mockState.committed.value = true
      mockState.result.value = emptyResult({ dry_run: false })
    })

    it('shows Close + Import another and hides Preview/Import', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="close-button"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="import-another-button"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="preview-button"]').exists()).toBe(false)
    })

    it('Close resets and emits update:visible false', async () => {
      mountComponent()
      await wrapper.find('[data-testid="close-button"]').trigger('click')
      expect(mockReset).toHaveBeenCalled()
      const events = wrapper.emitted('update:visible') as unknown[][]
      expect(events[0]).toEqual([false])
    })
  })

  describe('errors', () => {
    it('shows the api error message when apiError is set', async () => {
      mockState.apiError.value = 'Source reachable but unusable.'
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="api-error"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="api-error"]').text()).toContain('unusable')
    })

    it('shows the file error in the markdown panel', async () => {
      mockState.fileError.value = 'Only .md files are supported'
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="file-error"]').text()).toBe('Only .md files are supported')
    })
  })

  describe('not visible', () => {
    it('renders nothing when visible is false', () => {
      mountComponent({ visible: false })
      expect(wrapper.find('.p-dialog').exists()).toBe(false)
    })
  })
})
