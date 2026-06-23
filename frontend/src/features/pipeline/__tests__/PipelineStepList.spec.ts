import { describe, it, expect, afterEach, beforeAll, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'

beforeAll(() => {
  // PrimeVue Select (transition policy) uses matchMedia which jsdom lacks
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

import PipelineStepList from '../PipelineStepList.vue'
import type { PipelineGroup, PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

const mockAgents: Agent[] = [
  { id: 'agent-1', name: 'Dev Agent', model: 'claude-opus-4-6', image: '', template_content: '', scope: 'project', created_at: '', updated_at: '' },
]

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

function makeGroup(overrides: Partial<PipelineGroup> = {}): PipelineGroup {
  return {
    id: crypto.randomUUID(),
    name: 'Group',
    transition: 'auto',
    steps: [makeStep()],
    ...overrides,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(groups: PipelineGroup[], isAdmin = true) {
  wrapper = mount(PipelineStepList, {
    props: {
      groups,
      isAdmin,
      agents: mockAgents,
    },
    global: {
      plugins: [PrimeVue, ConfirmationService],
      stubs: {
        PipelineStepCard: {
          template: '<div class="step-card-stub">{{ step.name }}</div>',
          props: ['step', 'index', 'isAdmin', 'expanded', 'isFirst', 'isLast', 'agents'],
        },
      },
    },
  })
  return wrapper
}

describe('PipelineStepList', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders a PipelineGroupCard for each group', () => {
    const groups = [
      makeGroup({ id: 'g1', name: 'Setup' }),
      makeGroup({ id: 'g2', name: 'Development' }),
      makeGroup({ id: 'g3', name: 'Review' }),
    ]
    mountComponent(groups)
    const cards = wrapper.findAll('[data-testid="pipeline-group-card"]')
    expect(cards).toHaveLength(3)
  })

  it('renders no cards when groups is empty', () => {
    mountComponent([])
    const cards = wrapper.findAll('[data-testid="pipeline-group-card"]')
    expect(cards).toHaveLength(0)
  })

  it('renders a single group with its name', () => {
    const groups = [makeGroup({ id: 'g1', name: 'My Stage' })]
    mountComponent(groups)
    expect(wrapper.text()).toContain('My Stage')
  })

  it('passes isAdmin to group cards', () => {
    const groups = [makeGroup({ id: 'g1' })]
    mountComponent(groups, false)
    // Non-admin should not see admin controls
    expect(wrapper.find('[data-testid="move-group-up"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="remove-group"]').exists()).toBe(false)
  })
})
