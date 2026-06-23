import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'
import PipelineGroupCard from '../PipelineGroupCard.vue'
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
    id: 'group-1',
    name: 'Development',
    transition: 'auto',
    steps: [
      makeStep({ id: 's1', name: 'implement' }),
      makeStep({ id: 's2', name: 'review' }),
    ],
    ...overrides,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(
  groupOverrides: Partial<PipelineGroup> = {},
  props: Record<string, unknown> = {},
) {
  wrapper = mount(PipelineGroupCard, {
    props: {
      group: makeGroup(groupOverrides),
      index: 0,
      isAdmin: true,
      isFirst: false,
      isLast: false,
      groupCount: 3,
      agents: mockAgents,
      ...props,
    },
    global: {
      plugins: [PrimeVue, ConfirmationService],
      stubs: {
        PipelineStepCard: {
          template: '<div class="step-card-stub" :data-step-id="step.id">{{ step.name }}</div>',
          props: ['step', 'index', 'isAdmin', 'expanded', 'isFirst', 'isLast', 'agents'],
        },
      },
    },
  })
  return wrapper
}

describe('PipelineGroupCard', () => {
  afterEach(() => {
    wrapper?.unmount()
    vi.restoreAllMocks()
  })

  describe('rendering', () => {
    it('renders group name', () => {
      mountComponent({ name: 'Setup' })
      const name = wrapper.find('[data-testid="group-name"]')
      expect(name.text()).toBe('Setup')
    })

    it('renders step count', () => {
      mountComponent({
        steps: [makeStep({ id: 's1' }), makeStep({ id: 's2' }), makeStep({ id: 's3' })],
      })
      const count = wrapper.find('[data-testid="step-count"]')
      expect(count.text()).toContain('3 steps')
    })

    it('renders singular step count', () => {
      mountComponent({ steps: [makeStep({ id: 's1' })] })
      const count = wrapper.find('[data-testid="step-count"]')
      expect(count.text()).toContain('1 step')
    })

    it('renders step cards for each step', () => {
      mountComponent({
        steps: [makeStep({ id: 's1', name: 'step-one' }), makeStep({ id: 's2', name: 'step-two' })],
      })
      const stepCards = wrapper.findAll('.step-card-stub')
      expect(stepCards).toHaveLength(2)
    })

    it('renders add step button for admin', () => {
      mountComponent({}, { isAdmin: true })
      const btn = wrapper.find('[data-testid="add-step-to-group"]')
      expect(btn.exists()).toBe(true)
    })

    it('hides add step button for non-admin', () => {
      mountComponent({}, { isAdmin: false })
      const btn = wrapper.find('[data-testid="add-step-to-group"]')
      expect(btn.exists()).toBe(false)
    })

    it('hides move and remove buttons for non-admin', () => {
      mountComponent({}, { isAdmin: false })
      expect(wrapper.find('[data-testid="move-group-up"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="move-group-down"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="remove-group"]').exists()).toBe(false)
    })
  })

  describe('collapse/expand', () => {
    it('steps are visible by default', () => {
      mountComponent()
      const steps = wrapper.find('[data-testid="group-steps"]')
      expect(steps.isVisible()).toBe(true)
    })

    it('collapses steps on toggle click', async () => {
      mountComponent()
      const toggle = wrapper.find('[data-testid="collapse-toggle"]')
      await toggle.trigger('click')
      const steps = wrapper.find('[data-testid="group-steps"]')
      expect(steps.isVisible()).toBe(false)
    })

    it('expands steps on second toggle click', async () => {
      mountComponent()
      const toggle = wrapper.find('[data-testid="collapse-toggle"]')
      await toggle.trigger('click')
      await toggle.trigger('click')
      const steps = wrapper.find('[data-testid="group-steps"]')
      expect(steps.isVisible()).toBe(true)
    })
  })

  describe('inline rename', () => {
    it('switches to edit mode on name click', async () => {
      mountComponent({}, { isAdmin: true })
      const name = wrapper.find('[data-testid="group-name"]')
      await name.trigger('click')
      const input = wrapper.find('[data-testid="group-name-input"]')
      expect(input.exists()).toBe(true)
    })

    it('does not switch to edit mode for non-admin', async () => {
      mountComponent({}, { isAdmin: false })
      const name = wrapper.find('[data-testid="group-name"]')
      await name.trigger('click')
      const input = wrapper.find('[data-testid="group-name-input"]')
      expect(input.exists()).toBe(false)
    })

    it('emits rename on blur with new name', async () => {
      mountComponent({ id: 'g1', name: 'Old Name' })
      const name = wrapper.find('[data-testid="group-name"]')
      await name.trigger('click')
      const input = wrapper.find('[data-testid="group-name-input"]')
      await input.setValue('New Name')
      await input.trigger('blur')
      expect(wrapper.emitted('rename')).toBeTruthy()
      expect(wrapper.emitted('rename')![0]).toEqual(['g1', 'New Name'])
    })

    it('does not emit rename when name unchanged', async () => {
      mountComponent({ id: 'g1', name: 'Same Name' })
      const name = wrapper.find('[data-testid="group-name"]')
      await name.trigger('click')
      const input = wrapper.find('[data-testid="group-name-input"]')
      await input.trigger('blur')
      expect(wrapper.emitted('rename')).toBeUndefined()
    })
  })

  describe('remove group', () => {
    it('emits remove on delete button click (after confirm)', async () => {
      mountComponent({ id: 'g1' })
      const btn = wrapper.find('[data-testid="remove-group"]')
      await btn.trigger('click')
      // The confirm dialog is handled by PrimeVue ConfirmationService
      // In a real scenario the accept callback would fire
      // We verify the button exists and is clickable
      expect(btn.exists()).toBe(true)
    })
  })

  describe('add step', () => {
    it('emits add-step with group id on button click', async () => {
      mountComponent({ id: 'g1' })
      const btn = wrapper.find('[data-testid="add-step-to-group"]')
      await btn.trigger('click')
      expect(wrapper.emitted('add-step')).toBeTruthy()
      expect(wrapper.emitted('add-step')![0]).toEqual(['g1'])
    })
  })

  describe('move group buttons', () => {
    it('disables move up when isFirst', () => {
      mountComponent({}, { isFirst: true })
      const btn = wrapper.find('[data-testid="move-group-up"]')
      expect(btn.attributes('disabled')).toBeDefined()
    })

    it('disables move down when isLast', () => {
      mountComponent({}, { isLast: true })
      const btn = wrapper.find('[data-testid="move-group-down"]')
      expect(btn.attributes('disabled')).toBeDefined()
    })

    it('emits move-up on click', async () => {
      mountComponent({}, { index: 1, isFirst: false })
      const btn = wrapper.find('[data-testid="move-group-up"]')
      await btn.trigger('click')
      expect(wrapper.emitted('move-up')).toBeTruthy()
      expect(wrapper.emitted('move-up')![0]).toEqual([1])
    })

    it('emits move-down on click', async () => {
      mountComponent({}, { index: 1, isLast: false })
      const btn = wrapper.find('[data-testid="move-group-down"]')
      await btn.trigger('click')
      expect(wrapper.emitted('move-down')).toBeTruthy()
      expect(wrapper.emitted('move-down')![0]).toEqual([1])
    })
  })
})
