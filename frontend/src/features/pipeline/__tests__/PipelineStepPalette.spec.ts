import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import PipelineStepPalette from '../PipelineStepPalette.vue'
import type { Agent } from '@/stores/agents'

const mockAgents: Agent[] = [
  { id: 'agent-1', name: 'Dev Agent', model: 'claude-opus-4-6', image: '', template_content: '', scope: 'project', created_at: '', updated_at: '' },
  { id: 'agent-2', name: 'Review Agent', model: 'claude-sonnet-4-6', image: '', template_content: '', scope: 'project', created_at: '', updated_at: '' },
]

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(agents: Agent[] = mockAgents) {
  wrapper = mount(PipelineStepPalette, {
    props: { agents },
    global: { plugins: [PrimeVue] },
  })
  return wrapper
}

describe('PipelineStepPalette', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders all step type tiles', () => {
    mountComponent()
    const tiles = ['git_branch', 'agent_run', 'human', 'git_pr', 'ci_poll', 'notification', 'hitl_gate']
    for (const type of tiles) {
      expect(wrapper.find(`[data-testid="step-type-tile-${type}"]`).exists()).toBe(true)
    }
  })

  it('human tile has amber-breathe class', () => {
    mountComponent()
    const humanTile = wrapper.find('[data-testid="step-type-tile-human"]')
    expect(humanTile.classes()).toContain('amber-breathe')
  })

  it('click on tile emits add-step with action type', async () => {
    mountComponent()
    const tile = wrapper.find('[data-testid="step-type-tile-git_branch"]')
    await tile.trigger('click')
    expect(wrapper.emitted('add-step')).toBeTruthy()
    expect(wrapper.emitted('add-step')![0]).toEqual(['git_branch'])
  })

  it('dragstart sets correct dataTransfer data', () => {
    mountComponent()
    const tile = wrapper.find('[data-testid="step-type-tile-agent_run"]')
    const mockDataTransfer = { setData: vi.fn() }
    tile.trigger('dragstart', { dataTransfer: mockDataTransfer })
    expect(tile.attributes('draggable')).toBe('true')
  })

  it('agents section shows AgentChip for each agent', () => {
    mountComponent()
    const agentItems = wrapper.findAll('[data-testid="palette-agent-item"]')
    expect(agentItems).toHaveLength(2)
  })

  it('shows empty state when no agents', () => {
    mountComponent([])
    const agentItems = wrapper.findAll('[data-testid="palette-agent-item"]')
    expect(agentItems).toHaveLength(0)
  })
})
