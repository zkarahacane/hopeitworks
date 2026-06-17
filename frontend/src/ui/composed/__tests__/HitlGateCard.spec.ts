import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import HitlGateCard from '../HitlGateCard.vue'

type CardProps = InstanceType<typeof HitlGateCard>['$props']

function mountCard(props: Partial<CardProps> = {}) {
  return mount(HitlGateCard, { props, global: { plugins: [PrimeVue] } })
}

describe('HitlGateCard', () => {
  it('renders the awaiting-approval header', () => {
    const w = mountCard()
    expect(w.text()).toContain('Awaiting your approval')
  })

  it('shows story key and step name when provided', () => {
    const w = mountCard({ storyKey: 'PROJ-12', stepName: 'agent: dev' })
    expect(w.find('[data-testid="hitl-gate-story"]').text()).toBe('PROJ-12')
    expect(w.find('[data-testid="hitl-gate-step"]').text()).toContain('agent: dev')
  })

  it('renders a PR link only when prUrl is set', () => {
    expect(mountCard().find('[data-testid="hitl-gate-pr-link"]').exists()).toBe(false)
    const w = mountCard({ prUrl: 'https://example.com/pr/1' })
    const link = w.find('[data-testid="hitl-gate-pr-link"]')
    expect(link.exists()).toBe(true)
    expect(link.attributes('href')).toBe('https://example.com/pr/1')
  })

  it('emits approve / requestChanges / reject (no API)', async () => {
    const w = mountCard()
    await w.find('[data-testid="hitl-gate-approve"]').trigger('click')
    await w.find('[data-testid="hitl-gate-request-changes"]').trigger('click')
    await w.find('[data-testid="hitl-gate-reject"]').trigger('click')
    expect(w.emitted('approve')).toHaveLength(1)
    expect(w.emitted('requestChanges')).toHaveLength(1)
    expect(w.emitted('reject')).toHaveLength(1)
  })

  it('disables actions when busy', () => {
    const w = mountCard({ busy: true })
    expect(w.find('[data-testid="hitl-gate-approve"]').attributes('disabled')).toBeDefined()
    expect(w.find('[data-testid="hitl-gate-reject"]').attributes('disabled')).toBeDefined()
  })

  it('breathes amber when animated (default)', () => {
    expect(mountCard().find('[data-testid="hitl-gate-card"]').classes()).toContain('amber-breathe')
  })

  it('does not breathe when animated=false', () => {
    expect(
      mountCard({ animated: false }).find('[data-testid="hitl-gate-card"]').classes(),
    ).not.toContain('amber-breathe')
  })
})
