import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { h, defineComponent } from 'vue'
import PrimeVue from 'primevue/config'
import StoryImportDialog from '../StoryImportDialog.vue'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
    POST: vi.fn(),
  },
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
  // Mock FileReader for the composable
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

    it('does not show preview table', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="preview-table"]').exists()).toBe(false)
    })

    it('does not show import result', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="import-result"]').exists()).toBe(false)
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
