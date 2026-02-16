import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import EpicEmptyState from '../EpicEmptyState.vue'

function mountComponent() {
  return mount(EpicEmptyState, {
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('EpicEmptyState', () => {
  it('renders the heading text', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('No epics yet')
  })

  it('renders the description text', () => {
    const wrapper = mountComponent()
    expect(wrapper.text()).toContain('Import stories from your repository to get started')
  })

  it('renders the CTA button with correct label', () => {
    const wrapper = mountComponent()
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toContain('Import Stories')
  })

  it('emits "import" event when CTA button is clicked', async () => {
    const wrapper = mountComponent()
    const button = wrapper.find('button')
    await button.trigger('click')
    expect(wrapper.emitted('import')).toHaveLength(1)
  })

  it('renders an icon', () => {
    const wrapper = mountComponent()
    const icon = wrapper.find('.pi-th-large')
    expect(icon.exists()).toBe(true)
  })
})
