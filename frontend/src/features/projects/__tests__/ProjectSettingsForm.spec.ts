import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ProjectSettingsForm from '../ProjectSettingsForm.vue'
import type { Project } from '@/stores/projects'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
  },
}))

const mockProject: Project = {
  id: 'p1',
  name: 'Test Project',
  description: 'A test description',
  owner_id: 'u1',
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

function mountForm(props: Partial<{ project: Project; isSaving: boolean }> = {}) {
  return mount(ProjectSettingsForm, {
    props: {
      project: mockProject,
      isSaving: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('ProjectSettingsForm', () => {
  it('renders with project name and description pre-filled', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const nameInput = wrapper.find('input#name')
    const descriptionTextarea = wrapper.find('textarea#description')

    expect(nameInput.exists()).toBe(true)
    expect((nameInput.element as HTMLInputElement).value).toBe('Test Project')
    expect(descriptionTextarea.exists()).toBe(true)
    expect((descriptionTextarea.element as HTMLTextAreaElement).value).toBe('A test description')
  })

  it('Save button is disabled when form is pristine', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const saveButton = wrapper.find('button[type="submit"]')
    expect(saveButton.exists()).toBe(true)
    expect(saveButton.attributes('disabled')).toBeDefined()
  })

  it('Save button is enabled when form is dirty and valid', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const nameInput = wrapper.find('input#name')
    await nameInput.setValue('Updated Name')
    await flushPromises()

    const saveButton = wrapper.find('button[type="submit"]')
    expect(saveButton.attributes('disabled')).toBeUndefined()
  })

  it('emits save event with form values on submit', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const nameInput = wrapper.find('input#name')
    await nameInput.setValue('New Project Name')
    await flushPromises()

    // Wait for vee-validate async validation to complete
    await new Promise((resolve) => setTimeout(resolve, 10))
    await flushPromises()

    // Trigger native submit event on the form
    const form = wrapper.find('form')
    await form.trigger('submit')
    await flushPromises()

    // vee-validate handleSubmit is async — wait for it
    await new Promise((resolve) => setTimeout(resolve, 10))
    await flushPromises()

    const emitted = wrapper.emitted('save')
    expect(emitted).toBeDefined()
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toEqual({
      name: 'New Project Name',
      description: 'A test description',
    })
  })

  it('shows loading state on Save button when isSaving is true', async () => {
    const wrapper = mountForm({ isSaving: true })
    await flushPromises()

    const saveButton = wrapper.find('button[type="submit"]')
    // PrimeVue Button renders a loading icon when loading prop is true
    const loadingIcon = saveButton.find('[data-pc-section="loadingicon"]')
    expect(loadingIcon.exists()).toBe(true)
  })

  it('displays future tabs info message', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain(
      'Git, Agent, and Budget settings will be available in a future release.',
    )
  })

  it('shows validation error when name is cleared', async () => {
    const wrapper = mountForm()
    await flushPromises()

    const nameInput = wrapper.find('input#name')
    await nameInput.setValue('')
    // Trigger input and change events to ensure vee-validate picks up changes
    await nameInput.trigger('input')
    await nameInput.trigger('change')
    await nameInput.trigger('blur')
    await flushPromises()

    // vee-validate validates asynchronously, allow microtasks to complete
    await new Promise((resolve) => setTimeout(resolve, 10))
    await flushPromises()

    const errorTexts = wrapper.findAll('small.text-red-500')
    const nameErrorEl = errorTexts.find((el) => el.text().includes('Project name is required'))
    expect(nameErrorEl).toBeDefined()
  })
})
