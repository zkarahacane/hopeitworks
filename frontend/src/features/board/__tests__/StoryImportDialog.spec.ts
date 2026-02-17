import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { h, defineComponent, ref } from 'vue'
import PrimeVue from 'primevue/config'
import StoryImportDialog from '../StoryImportDialog.vue'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
    POST: vi.fn(),
  },
}))

// Mutable state used by composable mock
const mockState = {
  fileContent: ref<string | null>(null),
  fileName: ref<string | null>(null),
  parsedPreview: ref<{ key: string; title: string; scope?: string; valid: boolean; error?: string }[]>([]),
  importResult: ref<{ imported: number; updated: number; failed: number; errors: { key: string; message: string; code: string }[] } | null>(null),
  fileError: ref<string | null>(null),
  apiError: ref<string | null>(null),
  isImporting: ref(false),
}

const mockSelectFile = vi.fn()
const mockImportStories = vi.fn()
const mockReset = vi.fn().mockImplementation(() => {
  mockState.fileContent.value = null
  mockState.fileName.value = null
  mockState.parsedPreview.value = []
  mockState.importResult.value = null
  mockState.fileError.value = null
  mockState.apiError.value = null
  mockState.isImporting.value = false
})
const mockParseMarkdownPreview = vi.fn()

vi.mock('@/composables/useStoryImport', () => ({
  useStoryImport: () => ({
    ...mockState,
    selectFile: mockSelectFile,
    importStories: mockImportStories,
    reset: mockReset,
    parseMarkdownPreview: mockParseMarkdownPreview,
  }),
}))

/** Stub Dialog to render inline instead of teleporting */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header', 'class'],
  emits: ['update:visible'],
  setup(props, { slots, emit }) {
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

beforeEach(() => {
  // Reset all mock state
  mockState.fileContent.value = null
  mockState.fileName.value = null
  mockState.parsedPreview.value = []
  mockState.importResult.value = null
  mockState.fileError.value = null
  mockState.apiError.value = null
  mockState.isImporting.value = false
  vi.clearAllMocks()
  mockReset.mockImplementation(() => {
    mockState.fileContent.value = null
    mockState.fileName.value = null
    mockState.parsedPreview.value = []
    mockState.importResult.value = null
    mockState.fileError.value = null
    mockState.apiError.value = null
    mockState.isImporting.value = false
  })

  // Mock FileReader for any direct usage
  vi.stubGlobal(
    'FileReader',
    function MockFileReader(this: { readAsText: ReturnType<typeof vi.fn>; onload: null }) {
      this.readAsText = vi.fn()
      this.onload = null
    } as unknown as typeof FileReader,
  )
})

function mountComponent(props: Partial<{ visible: boolean; projectId: string }> = {}) {
  wrapper = mount(StoryImportDialog, {
    props: {
      visible: true,
      projectId: 'p1',
      ...props,
    },
    global: {
      plugins: [PrimeVue],
      stubs: {
        Dialog: DialogStub,
      },
    },
  })
  return wrapper
}

describe('StoryImportDialog', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('Step 1: Upload zone', () => {
    it('renders upload zone when no file is selected', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="drop-zone"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('Drag & drop a .md file here')
    })

    it('renders file input accepting .md files', () => {
      mountComponent()
      const fileInput = wrapper.find('[data-testid="file-input"]')
      expect(fileInput.exists()).toBe(true)
      expect(fileInput.attributes('accept')).toBe('.md')
    })

    it('does not show file error initially', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="file-error"]').exists()).toBe(false)
    })

    it('shows file error when fileError is set', async () => {
      mountComponent()
      mockState.fileError.value = 'Only .md files are supported'
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="file-error"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="file-error"]').text()).toBe('Only .md files are supported')
    })

    it('does not show preview table', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="preview-table"]').exists()).toBe(false)
    })

    it('does not show import result', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="import-result"]').exists()).toBe(false)
    })
  })

  describe('Step 2: Preview', () => {
    beforeEach(() => {
      mockState.fileContent.value = '---\nkey: S-01\n---\n# Story One\n'
      mockState.parsedPreview.value = [{ key: 'S-01', title: 'Story One', valid: true }]
    })

    it('renders preview table when fileContent is set', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="drop-zone"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="preview-table"]').exists()).toBe(true)
    })

    it('disables Import button when all parsed stories are invalid', async () => {
      mockState.parsedPreview.value = [
        { key: '(unknown)', title: '(no title)', valid: false, error: 'Missing key in frontmatter' },
      ]
      mountComponent()
      await wrapper.vm.$nextTick()
      const importButton = wrapper.find('[data-testid="import-button"]')
      expect(importButton.exists()).toBe(true)
      expect(importButton.attributes('disabled')).toBeDefined()
    })

    it('enables Import button when at least one valid story exists', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()
      const importButton = wrapper.find('[data-testid="import-button"]')
      expect(importButton.exists()).toBe(true)
      expect(importButton.attributes('disabled')).toBeUndefined()
    })

    it('shows invalid stories section when invalid stories exist', async () => {
      mockState.parsedPreview.value = [
        { key: 'S-01', title: 'Valid', valid: true },
        { key: '(unknown)', title: '(no title)', valid: false, error: 'Missing key in frontmatter' },
      ]
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="invalid-stories"]').exists()).toBe(true)
    })

    it('shows api error message when apiError is set', async () => {
      mockState.apiError.value = 'Import failed. Please try again.'
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="api-error"]').exists()).toBe(true)
    })
  })

  describe('Step 3: Result', () => {
    beforeEach(() => {
      mockState.importResult.value = { imported: 3, updated: 1, failed: 0, errors: [] }
    })

    it('renders summary tags when importResult is set', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="import-result"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('3 created')
      expect(wrapper.text()).toContain('1 updated')
    })

    it('does not render upload zone in Step 3', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[data-testid="drop-zone"]').exists()).toBe(false)
    })

    it('"Import Another File" triggers reset and returns to Step 1', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()

      const importAnotherButton = wrapper.find('[data-testid="import-another-button"]')
      expect(importAnotherButton.exists()).toBe(true)
      await importAnotherButton.trigger('click')
      await wrapper.vm.$nextTick()

      expect(mockReset).toHaveBeenCalled()
      // After reset, importResult is null so Step 1 should be visible
      expect(wrapper.find('[data-testid="drop-zone"]').exists()).toBe(true)
    })

    it('"Close" emits imported and update:visible false', async () => {
      mountComponent()
      await wrapper.vm.$nextTick()

      const closeButton = wrapper.find('[data-testid="close-button"]')
      expect(closeButton.exists()).toBe(true)
      await closeButton.trigger('click')

      expect(wrapper.emitted('imported')).toBeTruthy()
      expect(wrapper.emitted('update:visible')).toBeTruthy()
      const updateEvents = wrapper.emitted('update:visible') as unknown[][]
      expect(updateEvents[0]).toEqual([false])
    })
  })

  describe('dialog header', () => {
    it('has correct header text', () => {
      mountComponent()
      expect(wrapper.text()).toContain('Import Stories')
    })
  })

  describe('not visible', () => {
    it('renders nothing when visible is false', () => {
      mountComponent({ visible: false })
      expect(wrapper.find('.p-dialog').exists()).toBe(false)
    })
  })

  describe('emits', () => {
    it('emits update:visible false when dialog close is triggered', async () => {
      mountComponent()
      const dialog = wrapper.findComponent({ name: 'DialogStub' })
      await dialog.vm.$emit('update:visible', false)
      expect(wrapper.emitted('update:visible')).toBeTruthy()
    })
  })
})
