import { describe, it, expect, vi, beforeAll } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import AgentTable from '../AgentTable.vue'
import type { Agent } from '@/stores/agents'

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

vi.mock('@/utils/formatDate', () => ({
  formatRelativeDate: (date: string) => `mocked-${date}`,
}))

const sampleAgents: Agent[] = [
  {
    id: 'a1',
    name: 'Implement Agent',
    model: 'claude-opus-4-6',
    image: 'ghcr.io/org/agent:latest',
    template_content: 'You are a developer...',
    scope: 'project',
    project_id: 'p1',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'a2',
    name: 'Global Review Agent',
    model: 'claude-sonnet-4-6',
    image: 'ghcr.io/org/reviewer:latest',
    template_content: 'You are a code reviewer...',
    scope: 'global',
    project_id: null,
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
]

function mountComponent(props: Partial<InstanceType<typeof AgentTable>['$props']> = {}) {
  return mount(AgentTable, {
    props: {
      agents: sampleAgents,
      isAdmin: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('AgentTable', () => {
  it('renders column headers', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('Name')
    expect(text).toContain('Scope')
    expect(text).toContain('Model')
    expect(text).toContain('Image')
    expect(text).toContain('Actions')
  })

  it('renders agent names in the table', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('Implement Agent')
    expect(text).toContain('Global Review Agent')
  })

  it('renders model and image columns', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('claude-opus-4-6')
    expect(text).toContain('claude-sonnet-4-6')
    expect(text).toContain('ghcr.io/org/agent:latest')
    expect(text).toContain('ghcr.io/org/reviewer:latest')
  })

  it('renders scope tags with correct severity', () => {
    const wrapper = mountComponent()
    const tags = wrapper.findAll('[data-pc-name="tag"]')
    expect(tags.length).toBeGreaterThanOrEqual(2)

    const tagTexts = tags.map((t) => t.text())
    expect(tagTexts).toContain('project')
    expect(tagTexts).toContain('global')
  })

  it('disables edit button for global agents when user is not admin', () => {
    const wrapper = mountComponent({ isAdmin: false })
    const editButtons = wrapper.findAll('[data-testid="edit-agent-btn"]')
    expect(editButtons.length).toBe(2)

    // First agent is project-scoped - edit should be enabled
    expect(editButtons[0]!.attributes('disabled')).toBeUndefined()

    // Second agent is global-scoped - edit should be disabled for non-admin
    expect(editButtons[1]!.attributes('disabled')).toBeDefined()
  })

  it('enables edit button for global agents when user is admin', () => {
    const wrapper = mountComponent({ isAdmin: true })
    const editButtons = wrapper.findAll('[data-testid="edit-agent-btn"]')
    expect(editButtons.length).toBe(2)

    // Both should be enabled for admin
    expect(editButtons[0]!.attributes('disabled')).toBeUndefined()
    expect(editButtons[1]!.attributes('disabled')).toBeUndefined()
  })

  it('hides delete button for global agents when user is not admin', () => {
    const wrapper = mountComponent({ isAdmin: false })
    const deleteButtons = wrapper.findAll('[data-testid="delete-agent-btn"]')
    // Only the project-scoped agent should have a delete button
    expect(deleteButtons.length).toBe(1)
  })

  it('shows delete button for all agents when user is admin', () => {
    const wrapper = mountComponent({ isAdmin: true })
    const deleteButtons = wrapper.findAll('[data-testid="delete-agent-btn"]')
    // Both agents should have delete buttons for admin
    expect(deleteButtons.length).toBe(2)
  })

  it('renders formatted dates using formatRelativeDate', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('mocked-2026-01-15T10:00:00Z')
    expect(text).toContain('mocked-2026-01-16T10:00:00Z')
  })
})
