import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import PipelineStepCard from '../PipelineStepCard.vue'
import type { PipelineStep } from '@/stores/pipelineConfig'

function makeStep(overrides: Partial<PipelineStep> = {}): PipelineStep {
  return {
    id: crypto.randomUUID(),
    name: 'implement',
    action_type: 'agent_run',
    model: 'claude-opus-4-6',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
    ...overrides,
  }
}

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

  describe('model label visibility', () => {
    it('shows model label when model is set', () => {
      mountComponent({ model: 'claude-opus-4-6' })
      expect(wrapper.text()).toContain('Claude Opus 4.6')
    })

    it('hides model label when model is not set', () => {
      mountComponent({ model: undefined as unknown as PipelineStep['model'] })
      expect(wrapper.text()).not.toContain('Claude Opus')
      expect(wrapper.text()).not.toContain('Claude Sonnet')
      expect(wrapper.text()).not.toContain('Claude Haiku')
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
