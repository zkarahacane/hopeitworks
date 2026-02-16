import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import CreateProjectDialog from '../CreateProjectDialog.vue'

const mockExecute = vi.fn()
const mockIsLoading = ref(false)
const mockError = ref<Error | null>(null)

vi.mock('@/composables/useProjects', () => ({
  useProjects: () => ({
    createProject: {
      execute: mockExecute,
      isLoading: mockIsLoading,
      error: mockError,
    },
    projects: ref([]),
    pagination: ref(null),
    isLoading: ref(false),
    error: ref(null),
    fetchProjects: vi.fn(),
    retry: vi.fn(),
  }),
}))

/** Stub Dialog to render inline instead of teleporting */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header'],
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(visible = true) {
  wrapper = mount(CreateProjectDialog, {
    props: {
      visible,
    },
    global: {
      plugins: [PrimeVue, createPinia()],
      stubs: {
        Dialog: DialogStub,
      },
    },
  })
  return wrapper
}

describe('CreateProjectDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockExecute.mockReset()
    mockIsLoading.value = false
    mockError.value = null
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders dialog with Name and Description fields when visible', () => {
    mountComponent()
    expect(wrapper.text()).toContain('Create Project')
    expect(wrapper.find('#project-name').exists()).toBe(true)
    expect(wrapper.find('#project-description').exists()).toBe(true)
  })

  it('does not render content when not visible', () => {
    mountComponent(false)
    expect(wrapper.find('#project-name').exists()).toBe(false)
  })

  it('renders Cancel and Create buttons in footer', () => {
    mountComponent()
    const buttons = wrapper.findAll('button')
    const cancelBtn = buttons.find((b) => b.text().includes('Cancel'))
    const createBtn = buttons.find((b) => b.text().includes('Create'))
    expect(cancelBtn).toBeDefined()
    expect(createBtn).toBeDefined()
  })

  it('emits update:visible with false when cancel is clicked', async () => {
    mountComponent()

    const cancelBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('Cancel'))
    await cancelBtn!.trigger('click')

    await wrapper.vm.$nextTick()

    const emitted = wrapper.emitted('update:visible')
    expect(emitted).toBeDefined()
    expect(emitted![0]![0]).toBe(false)
  })

  it('displays API error message inside dialog', async () => {
    mockError.value = new Error('Server validation failed')

    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Server validation failed')
  })

  it('does not call execute when form is submitted with empty name', async () => {
    mountComponent()

    const form = wrapper.find('form')
    await form.trigger('submit')
    await flushPromises()

    expect(mockExecute).not.toHaveBeenCalled()
  })

  it('calls execute and emits created on valid submission', async () => {
    const createdProject = {
      id: 'p1',
      name: 'My Project',
      owner_id: 'u1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    }
    mockExecute.mockResolvedValue(createdProject)

    mountComponent()

    // Set value via setValue which properly handles v-model
    await wrapper.find('#project-name').setValue('My Project')
    await flushPromises()

    // Submit form
    await wrapper.find('form').trigger('submit')

    // vee-validate handleSubmit uses deep async chains; poll until execute is called
    await vi.waitFor(() => {
      expect(mockExecute).toHaveBeenCalled()
    })

    expect(mockExecute).toHaveBeenCalledWith({
      name: 'My Project',
      description: undefined,
    })

    // Wait for emit to propagate after execute resolves
    await flushPromises()

    const createdEmits = wrapper.emitted('created')
    expect(createdEmits).toBeDefined()
    expect(createdEmits![0]![0]).toEqual(createdProject)
  })

  it('does not emit created when execute returns null', async () => {
    mockExecute.mockResolvedValue(null)

    mountComponent()

    await wrapper.find('#project-name').setValue('Test Project')
    await flushPromises()

    await wrapper.find('form').trigger('submit')

    await vi.waitFor(() => {
      expect(mockExecute).toHaveBeenCalled()
    })

    await flushPromises()

    expect(wrapper.emitted('created')).toBeUndefined()
  })
})
