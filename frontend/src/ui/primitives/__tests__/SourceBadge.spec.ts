import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import SourceBadge from '../SourceBadge.vue'

type BadgeProps = InstanceType<typeof SourceBadge>['$props']

function mountBadge(props: BadgeProps) {
  return mount(SourceBadge, { props, global: { plugins: [PrimeVue] } })
}

describe('SourceBadge', () => {
  it('renders "In-app" for manual, without a deep-link', () => {
    const w = mountBadge({ source: 'manual' })
    expect(w.text()).toContain('In-app')
    expect(w.find('[data-testid="source-badge-link"]').exists()).toBe(false)
  })

  it('renders "Markdown" for the markdown source', () => {
    const w = mountBadge({ source: 'markdown' })
    expect(w.text()).toContain('Markdown')
  })

  it('renders "GitHub Projects" for the github_projects source', () => {
    const w = mountBadge({ source: 'github_projects' })
    expect(w.text()).toContain('GitHub Projects')
  })

  it('falls back to "In-app" for an unknown / nullish source', () => {
    expect(mountBadge({ source: undefined }).text()).toContain('In-app')
    expect(mountBadge({ source: null }).text()).toContain('In-app')
  })

  it('renders a deep-link ONLY when a sourceUrl is present', () => {
    const without = mountBadge({ source: 'github_projects' })
    expect(without.find('[data-testid="source-badge-link"]').exists()).toBe(false)

    const withUrl = mountBadge({ source: 'github_projects', sourceUrl: 'https://github.com/acme/repo/issues/1' })
    const link = withUrl.find('[data-testid="source-badge-link"]')
    expect(link.exists()).toBe(true)
    expect(link.attributes('href')).toBe('https://github.com/acme/repo/issues/1')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.attributes('rel')).toBe('noopener')
  })

  it('exposes the source via data-source on the badge', () => {
    expect(
      mountBadge({ source: 'github_projects' }).find('[data-testid="source-badge"]').attributes('data-source'),
    ).toBe('github_projects')
    expect(
      mountBadge({ source: undefined }).find('[data-testid="source-badge"]').attributes('data-source'),
    ).toBe('manual')
  })
})
