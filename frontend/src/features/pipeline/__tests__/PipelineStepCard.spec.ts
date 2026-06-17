import { describe, it, expect, afterEach, vi, beforeAll } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'

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

  describe('type chip', () => {
    it('renders step type chip', () => {
      mountComponent()
      const chip = wrapper.find('[data-testid="step-type-chip"]')
      expect(chip.exists()).toBe(true)
    })

    it('displays action type in chip', () => {
      mountComponent({ action_type: 'git_pr' })
      const chip = wrapper.find('[data-testid="step-type-chip"]')
      expect(chip.text()).toContain('git_pr')
    })

    it('backward compat: action-type-tag testid still works', () => {
      mountComponent()
      const tag = wrapper.find('[data-testid="action-type-tag"]')
      expect(tag.exists()).toBe(true)
    })

    it('normal step type chip has normal class', () => {
      mountComponent({ action_type: 'git_branch' })
      const chip = wrapper.find('[data-testid="step-type-chip"]')
      expect(chip.classes()).toContain('type-chip--normal')
    })
  })

  describe('human gate row', () => {
    it('human step row has amber-breathe class', () => {
      mountComponent({ action_type: 'human' })
      const row = wrapper.find('[data-testid="pipeline-step-card"]')
      expect(row.classes()).toContain('amber-breathe')
    })

    it('human step row has amber-breathe class for gate surface', () => {
      mountComponent({ action_type: 'human' })
      const row = wrapper.find('[data-testid="pipeline-step-card"]')
      // amber-breathe class confirms the gate surface styling path is taken
      expect(row.classes()).toContain('amber-breathe')
    })

    it('human step type chip has gate class', () => {
      mountComponent({ action_type: 'human' })
      const chip = wrapper.find('[data-testid="step-type-chip"]')
      expect(chip.classes()).toContain('type-chip--gate')
    })

    it('human step shows descriptive text', () => {
      mountComponent({ action_type: 'human' })
      expect(wrapper.text()).toContain('human stops the pipeline here')
    })

    it('human step does not show agent selector', () => {
      mountComponent({ action_type: 'human' })
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })
  })

  describe('auto/manual toggle', () => {
    it('toggle reflects auto_approve=false as Manual', () => {
      mountComponent({ auto_approve: false })
      const toggle = wrapper.find('[data-testid="auto-approve-toggle"]')
      expect(toggle.exists()).toBe(true)
      expect(wrapper.text()).toContain('Manual')
    })

    it('toggle reflects auto_approve=true as Auto', () => {
      mountComponent({ auto_approve: true })
      expect(wrapper.text()).toContain('Auto')
    })
  })

  describe('agent/model display visibility', () => {
    it('shows agent display when agent_id matches an agent', () => {
      mountComponent({ agent_id: 'agent-1' })
      expect(wrapper.find('[data-testid="agent-display"]').exists()).toBe(true)
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

  describe('step name display', () => {
    it('displays step name', () => {
      mountComponent({ name: 'my-step' })
      expect(wrapper.text()).toContain('my-step')
    })

    it('displays action type in chip', () => {
      mountComponent({ action_type: 'git_pr' })
      expect(wrapper.text()).toContain('git_pr')
    })
  })
})
