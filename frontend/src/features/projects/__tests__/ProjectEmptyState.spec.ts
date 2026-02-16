import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ProjectEmptyState from '../ProjectEmptyState.vue'

function mountComponent() {
  return mount(ProjectEmptyState, {
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('ProjectEmptyState', () => {
  it('renders the heading text', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('No projects yet')
  })

  it('renders the description text', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('Get started by creating your first project')
  })

  it('renders the CTA button with correct label', () => {
    const wrapper = mountComponent()
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toContain('Create your first project')
  })

  it('emits "create" event when CTA button is clicked', async () => {
    const wrapper = mountComponent()
    const button = wrapper.find('button')
    await button.trigger('click')
    expect(wrapper.emitted('create')).toHaveLength(1)
  })

  it('renders a folder icon', () => {
    const wrapper = mountComponent()
    const icon = wrapper.find('.pi-folder-open')
    expect(icon.exists()).toBe(true)
  })
})
