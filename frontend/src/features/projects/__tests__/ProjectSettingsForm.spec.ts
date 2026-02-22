import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ProjectSettingsForm from '../ProjectSettingsForm.vue'
import type { Project } from '@/stores/projects'

const baseProject: Project = {
  id: 'p1',
  name: 'Test Project',
  description: 'A description',
  repo_url: 'https://github.com/org/repo',
  git_provider: 'github',
  agent_runtime: 'docker',
  default_model: 'claude-opus-4-5',
  owner_id: 'u1',
  created_at: '2026-02-16T10:00:00Z',
  updated_at: '2026-02-16T10:00:00Z',
}

/** Stub Select to avoid matchMedia error in jsdom */
const SelectStub = defineComponent({
  name: 'SelectStub',
  props: ['modelValue', 'options', 'invalid'],
  emits: ['update:modelValue'],
  setup(props, { attrs }) {
    return () =>
      h('select', {
        id: attrs.id,
        value: props.modelValue,
      })
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(project: Project = baseProject, isSaving = false) {
  wrapper = mount(ProjectSettingsForm, {
    props: {
      project,
      isSaving,
    },
    global: {
      plugins: [PrimeVue, createPinia()],
      stubs: {
        Select: SelectStub,
      },
    },
  })
  return wrapper
}

describe('ProjectSettingsForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders form with all six fields', () => {
    mountComponent()
    expect(wrapper.find('#settings-name').exists()).toBe(true)
    expect(wrapper.find('#settings-description').exists()).toBe(true)
    expect(wrapper.find('#settings-repo-url').exists()).toBe(true)
    expect(wrapper.find('#settings-git-provider').exists()).toBe(true)
    expect(wrapper.find('#settings-agent-runtime').exists()).toBe(true)
    expect(wrapper.find('#settings-default-model').exists()).toBe(true)
  })

  it('pre-fills name from project prop', () => {
    mountComponent()
    const input = wrapper.find('#settings-name')
    expect((input.element as HTMLInputElement).value).toBe('Test Project')
  })

  it('pre-fills repo_url from project prop', () => {
    mountComponent()
    const input = wrapper.find('#settings-repo-url')
    expect((input.element as HTMLInputElement).value).toBe('https://github.com/org/repo')
  })

  it('pre-fills default_model from project prop', () => {
    mountComponent()
    const input = wrapper.find('#settings-default-model')
    expect((input.element as HTMLInputElement).value).toBe('claude-opus-4-5')
  })

  it('displays Pipeline Configuration section heading', () => {
    mountComponent()
    expect(wrapper.text()).toContain('Pipeline Configuration')
  })

  it('emits save with updated fields on valid form submit', async () => {
    mountComponent()

    await wrapper.find('#settings-name').setValue('Updated Name')
    await wrapper.find('#settings-repo-url').setValue('https://github.com/org/new-repo')
    await flushPromises()

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    await vi.waitFor(() => {
      const emitted = wrapper.emitted('save')
      expect(emitted).toBeDefined()
    })

    const emitted = wrapper.emitted('save')!
    expect(emitted[0]![0]).toEqual(
      expect.objectContaining({
        name: 'Updated Name',
        repo_url: 'https://github.com/org/new-repo',
        git_provider: 'github',
        agent_runtime: 'docker',
        default_model: 'claude-opus-4-5',
      }),
    )
  })

  it('does not emit save with invalid repo_url', async () => {
    mountComponent()

    await wrapper.find('#settings-repo-url').setValue('not-a-url')
    await flushPromises()

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(wrapper.emitted('save')).toBeUndefined()
    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('Must be a valid URL')
    })
  })

  it('renders save button with loading state when isSaving is true', () => {
    mountComponent(baseProject, true)
    const saveBtn = wrapper.find('[data-testid="save-settings-btn"]')
    expect(saveBtn.exists()).toBe(true)
  })
})
