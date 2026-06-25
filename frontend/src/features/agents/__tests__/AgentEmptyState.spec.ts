import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import AgentEmptyState from '../AgentEmptyState.vue'

function mountComponent(isAdmin: boolean) {
  return mount(AgentEmptyState, {
    props: { isAdmin },
    global: { plugins: [PrimeVue] },
  })
}

describe('AgentEmptyState', () => {
  it('renders the empty message regardless of role', () => {
    const wrapper = mountComponent(false)
    expect(wrapper.text()).toContain('No agents found for this project.')
  })

  it('shows the New Agent CTA with a unique testid for admins (RG1)', () => {
    const wrapper = mountComponent(true)
    const cta = wrapper.find('[data-testid="empty-create-agent-button"]')
    expect(cta.exists()).toBe(true)
    expect(cta.text()).toContain('New Agent')
  })

  it('hides the New Agent CTA for non-admins (RG2)', () => {
    const wrapper = mountComponent(false)
    expect(wrapper.find('[data-testid="empty-create-agent-button"]').exists()).toBe(false)
  })

  it('emits createClick when the CTA is clicked', async () => {
    const wrapper = mountComponent(true)
    await wrapper.find('[data-testid="empty-create-agent-button"]').trigger('click')
    expect(wrapper.emitted('createClick')).toHaveLength(1)
  })
})
