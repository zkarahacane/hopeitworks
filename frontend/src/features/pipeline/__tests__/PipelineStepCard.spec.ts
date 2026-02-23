import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import PipelineStepCard from '../PipelineStepCard.vue'
import type { PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

function makeStep(overrides: Partial<PipelineStep> = {}): PipelineStep {
  return {
    id: crypto.randomUUID(),
    name: 'implement',
    action_type: 'agent_run',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
    ...overrides,
  }
}

const mockAgents: Agent[] = [
  { id: 'agent-1', name: 'Dev Agent', model: 'claude-opus-4-6', image: '', template_content: '', scope: 'project', created_at: '', updated_at: '' },
  { id: 'agent-2', name: 'Review Agent', model: 'claude-sonnet-4-6', image: '', template_content: '', scope: 'project', created_at: '', updated_at: '' },
]

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(stepOverrides: Partial<PipelineStep> = {}, props: Record<string, unknown> = {}) {
  wrapper = mount(PipelineStepCard, {
    props: {
      step: makeStep(stepOverrides),
      index: 0,
      isAdmin: true,
      expanded: false,
      isFirst: true,
      isLast: false,
      agents: mockAgents,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('PipelineStepCard', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('action type icon/badge', () => {
    it('renders action type tag', () => {
      mountComponent()
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.exists()).toBe(true)
    })

    it('displays correct icon for agent_run', () => {
      mountComponent({ action_type: 'agent_run' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.exists()).toBe(true)
      // The Tag component renders with the icon attribute
      expect(tag.attributes('icon') || tag.html()).toContain('android')
    })

    it('displays correct icon for git_branch', () => {
      mountComponent({ action_type: 'git_branch' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('code-branch')
    })

    it('displays correct icon for git_pr', () => {
      mountComponent({ action_type: 'git_pr' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('arrow-right-arrow-left')
    })

    it('displays correct icon for notification', () => {
      mountComponent({ action_type: 'notification' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('bell')
    })

    it('displays correct icon for human', () => {
      mountComponent({ action_type: 'human' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('user')
    })

    it('displays correct icon for ci_poll', () => {
      mountComponent({ action_type: 'ci_poll' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('sync')
    })

    it('displays correct icon for hitl_gate', () => {
      mountComponent({ action_type: 'hitl_gate' })
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.html()).toContain('shield')
    })
  })

  describe('agent/model display visibility', () => {
    it('shows agent name when agent_id matches an agent', () => {
      mountComponent({ agent_id: 'agent-1' })
      expect(wrapper.find('[data-testid="agent-display"]').text()).toBe('Dev Agent')
    })

    it('shows legacy model string when model is set and no agent_id', () => {
      mountComponent({ model: 'claude-opus-4-6' })
      expect(wrapper.find('[data-testid="agent-display"]').text()).toBe('claude-opus-4-6')
    })

    it('hides display when neither agent_id nor model is set', () => {
      mountComponent({ agent_id: undefined, model: undefined })
      expect(wrapper.find('[data-testid="agent-display"]').exists()).toBe(false)
    })
  })

  describe('step name and action type display', () => {
    it('displays step name', () => {
      mountComponent({ name: 'my-step' })
      expect(wrapper.text()).toContain('my-step')
    })

    it('displays action type in tag', () => {
      mountComponent({ action_type: 'git_pr' })
      expect(wrapper.text()).toContain('git_pr')
    })
  })
})
