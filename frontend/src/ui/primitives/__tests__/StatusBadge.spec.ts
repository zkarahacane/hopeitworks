import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import StatusBadge from '../StatusBadge.vue'

type BadgeProps = InstanceType<typeof StatusBadge>['$props']

function mountBadge(props: BadgeProps) {
  return mount(StatusBadge, { props, global: { plugins: [PrimeVue] } })
}

describe('StatusBadge', () => {
  it('exposes the resolved family via data-family', () => {
    expect(mountBadge({ status: 'running' }).attributes('data-family')).toBe('running')
    expect(mountBadge({ status: 'completed' }).attributes('data-family')).toBe('done')
    expect(mountBadge({ status: 'paused' }).attributes('data-family')).toBe('gate')
    expect(mountBadge({ status: 'failed' }).attributes('data-family')).toBe('failed')
    expect(mountBadge({ status: 'backlog' }).attributes('data-family')).toBe('queued')
  })

  it('renders a pulse dot for live families (running)', () => {
    const w = mountBadge({ status: 'running' })
    const pulse = w.find('[data-testid="status-badge-pulse"]')
    expect(pulse.exists()).toBe(true)
    expect(pulse.classes()).toContain('live-pulse')
  })

  it('uses the amber-breathe class for a gate family pulse', () => {
    const w = mountBadge({ status: 'paused' })
    const pulse = w.find('[data-testid="status-badge-pulse"]')
    expect(pulse.exists()).toBe(true)
    expect(pulse.classes()).toContain('amber-breathe')
  })

  it('renders an icon (not a pulse) for static families (done)', () => {
    const w = mountBadge({ status: 'completed' })
    expect(w.find('[data-testid="status-badge-pulse"]').exists()).toBe(false)
    expect(w.find('[data-testid="status-badge-icon"]').exists()).toBe(true)
  })

  it('suppresses the pulse when animated=false', () => {
    const w = mountBadge({ status: 'running', animated: false })
    expect(w.find('[data-testid="status-badge-pulse"]').exists()).toBe(false)
    expect(w.find('[data-testid="status-badge-icon"]').exists()).toBe(true)
  })

  it('suppresses the pulse when resolved=true', () => {
    const w = mountBadge({ status: 'paused', resolved: true })
    expect(w.find('[data-testid="status-badge-pulse"]').exists()).toBe(false)
  })

  it('hides the icon when icon=false and not pulsing', () => {
    const w = mountBadge({ status: 'completed', icon: false })
    expect(w.find('[data-testid="status-badge-icon"]').exists()).toBe(false)
  })

  it('renders a custom label', () => {
    const w = mountBadge({ status: 'waiting_approval', label: 'HITL' })
    expect(w.text()).toContain('HITL')
  })

  it('defaults the label to the raw status', () => {
    expect(mountBadge({ status: 'failed' }).text()).toContain('failed')
  })

  it('applies the family color token via CSS variable, never a hardcoded hex', () => {
    const w = mountBadge({ status: 'running' })
    const style = w.attributes('style') ?? ''
    expect(style).toContain('var(--status-running-color)')
    expect(style).not.toMatch(/#[0-9a-f]{3,6}/i)
  })
})
