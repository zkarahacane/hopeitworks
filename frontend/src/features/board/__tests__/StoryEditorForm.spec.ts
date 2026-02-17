import { describe, it, expect, afterEach, vi, beforeAll } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import StoryEditorForm from '../StoryEditorForm.vue'
import type { UpdateStoryFields } from '@/stores/stories'

beforeAll(() => {
  // PrimeVue Select uses matchMedia which is not available in jsdom
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(props: {
  modelValue: UpdateStoryFields
  errors?: Record<string, string>
  apiError?: string | null
  isSaving?: boolean
}) {
  wrapper = mount(StoryEditorForm, {
    props: {
      errors: {},
      apiError: null,
      isSaving: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('StoryEditorForm', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders all fields with initial values', () => {
    mountComponent({
      modelValue: {
        title: 'My Story',
        objective: 'Build it',
        acceptance_criteria: '- done',
        scope: 'backend',
        target_files: ['src/foo.ts'],
      },
    })

    expect(wrapper.find('#story-title').exists()).toBe(true)
    expect(wrapper.find('#story-objective').exists()).toBe(true)
    expect(wrapper.find('#story-acceptance-criteria').exists()).toBe(true)
    expect(wrapper.find('#story-scope').exists()).toBe(true)
  })

  it('shows inline error when errors.title is set', () => {
    mountComponent({
      modelValue: { title: '' },
      errors: { title: 'Title is required' },
    })

    const errorEl = wrapper.find('.text-red-500')
    expect(errorEl.exists()).toBe(true)
    expect(errorEl.text()).toBe('Title is required')
  })

  it('does not show inline error when no errors', () => {
    mountComponent({
      modelValue: { title: 'Valid' },
      errors: {},
    })

    expect(wrapper.find('.text-red-500').exists()).toBe(false)
  })

  it('renders API error Message when apiError is set', () => {
    mountComponent({
      modelValue: { title: 'Story' },
      apiError: 'Server error occurred',
    })

    expect(wrapper.text()).toContain('Server error occurred')
  })

  it('does not render API error Message when apiError is null', () => {
    mountComponent({
      modelValue: { title: 'Story' },
      apiError: null,
    })

    // There should be no error message component rendered
    const messages = wrapper.findAll('.p-message')
    expect(messages.length).toBe(0)
  })

  it('emits save event on form submit', async () => {
    mountComponent({ modelValue: { title: 'Story' } })

    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.emitted('save')).toHaveLength(1)
  })

  it('emits cancel event when Cancel button is clicked', async () => {
    mountComponent({ modelValue: { title: 'Story' } })

    const cancelBtn = wrapper.findAll('button').find((b) => b.text().includes('Cancel'))
    expect(cancelBtn).toBeDefined()

    await cancelBtn!.trigger('click')

    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('renders target file inputs for each file', () => {
    mountComponent({
      modelValue: {
        title: 'Story',
        target_files: ['src/a.ts', 'src/b.ts'],
      },
    })

    const fileInputs = wrapper.findAll('input[placeholder="path/to/file"]')
    expect(fileInputs).toHaveLength(2)
  })

  it('emits update:modelValue with new file when Add file clicked', async () => {
    mountComponent({
      modelValue: {
        title: 'Story',
        target_files: ['src/a.ts'],
      },
    })

    const addBtn = wrapper.find('[aria-label="Add file"]')
    expect(addBtn.exists()).toBe(true)

    await addBtn.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    expect(emitted![0]![0]).toEqual({
      title: 'Story',
      target_files: ['src/a.ts', ''],
    })
  })

  it('emits update:modelValue with file removed when Remove file clicked', async () => {
    mountComponent({
      modelValue: {
        title: 'Story',
        target_files: ['src/a.ts', 'src/b.ts'],
      },
    })

    const removeButtons = wrapper.findAll('[aria-label="Remove file"]')
    expect(removeButtons).toHaveLength(2)

    await removeButtons[0]!.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeDefined()
    expect(emitted![0]![0]).toEqual({
      title: 'Story',
      target_files: ['src/b.ts'],
    })
  })
})
