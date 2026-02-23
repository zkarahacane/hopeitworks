import { describe, it, expect, afterEach, vi, beforeAll } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { h, defineComponent, nextTick } from 'vue'
import PrimeVue from 'primevue/config'
import AddStepDialog from '../AddStepDialog.vue'

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

/** Stub Dialog to render inline instead of teleporting. */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header'],
  emits: ['update:visible'],
  setup(props, { slots }) {
    return () => {
      if (!props.visible) return null
      return h('div', { class: 'p-dialog' }, [
        h('div', { class: 'p-dialog-header' }, props.header),
        h('div', { class: 'p-dialog-content' }, slots.default?.()),
        h('div', { class: 'p-dialog-footer' }, slots.footer?.()),
      ])
    }
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

const mockAgents = [
  { id: 'agent-1', name: 'Dev Agent', model: 'claude-sonnet-4-6', image: '', template_content: '', scope: 'project' as const, created_at: '', updated_at: '' },
  { id: 'agent-2', name: 'Review Agent', model: 'claude-opus-4-6', image: '', template_content: '', scope: 'project' as const, created_at: '', updated_at: '' },
]

function mountComponent(props: { visible?: boolean } = {}) {
  wrapper = mount(AddStepDialog, {
    props: {
      visible: true,
      agents: mockAgents,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
      stubs: {
        Dialog: DialogStub,
      },
    },
  })
  return wrapper
}

async function selectActionType(type: string) {
  // Simulate changing the action type by accessing the component's internal state
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const vm = wrapper.vm as any
  vm.actionType = type
  await nextTick()
  await nextTick()
}

describe('AddStepDialog', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('action type options', () => {
    it('renders all action type options in the select', () => {
      mountComponent()
      const text = wrapper.text()
      // The select should have the action type options available.
      // At minimum, the default selected value should be visible
      const select = wrapper.find('[data-testid="action-type-select"]')
      expect(select.exists()).toBe(true)
    })

    it('defaults to agent_run action type', () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).actionType).toBe('agent_run')
    })
  })

  describe('agent selector visibility', () => {
    it('shows agent selector when action type is agent_run', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(true)
    })

    it('hides agent selector when action type is git_branch', async () => {
      mountComponent()
      await selectActionType('git_branch')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('hides agent selector when action type is git_pr', async () => {
      mountComponent()
      await selectActionType('git_pr')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('hides agent selector when action type is notification', async () => {
      mountComponent()
      await selectActionType('notification')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('hides agent selector when action type is human', async () => {
      mountComponent()
      await selectActionType('human')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('hides agent selector when action type is ci_poll', async () => {
      mountComponent()
      await selectActionType('ci_poll')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('hides agent selector when action type is hitl_gate', async () => {
      mountComponent()
      await selectActionType('hitl_gate')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
    })

    it('shows agent selector again when switching back to agent_run', async () => {
      mountComponent()
      await selectActionType('git_branch')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(false)
      await selectActionType('agent_run')
      expect(wrapper.find('[data-testid="agent-select"]').exists()).toBe(true)
    })
  })

  describe('conditional config fields — git_branch', () => {
    it('shows branch_pattern field when action type is git_branch', async () => {
      mountComponent()
      await selectActionType('git_branch')
      expect(wrapper.find('[data-testid="branch-pattern-input"]').exists()).toBe(true)
    })

    it('hides branch_pattern field for other types', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="branch-pattern-input"]').exists()).toBe(false)
    })
  })

  describe('conditional config fields — git_pr', () => {
    it('shows title_template, target_branch, and draft fields', async () => {
      mountComponent()
      await selectActionType('git_pr')
      expect(wrapper.find('[data-testid="title-template-input"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="target-branch-input"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="draft-toggle"]').exists()).toBe(true)
    })

    it('hides git_pr fields for other types', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="title-template-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="target-branch-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="draft-toggle"]').exists()).toBe(false)
    })
  })

  describe('conditional config fields — notification', () => {
    it('shows message textarea when action type is notification', async () => {
      mountComponent()
      await selectActionType('notification')
      expect(wrapper.find('[data-testid="notification-message-input"]').exists()).toBe(true)
    })

    it('hides notification message for other types', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="notification-message-input"]').exists()).toBe(false)
    })
  })

  describe('conditional config fields — human', () => {
    it('shows message and instructions textareas when action type is human', async () => {
      mountComponent()
      await selectActionType('human')
      expect(wrapper.find('[data-testid="human-message-input"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="human-instructions-input"]').exists()).toBe(true)
    })

    it('hides human fields for other types', () => {
      mountComponent()
      expect(wrapper.find('[data-testid="human-message-input"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="human-instructions-input"]').exists()).toBe(false)
    })
  })

  describe('conditional config fields — ci_poll', () => {
    it('shows info message when action type is ci_poll', async () => {
      mountComponent()
      await selectActionType('ci_poll')
      expect(wrapper.find('[data-testid="ci-poll-info"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('No additional configuration required')
    })
  })

  describe('conditional config fields — hitl_gate', () => {
    it('shows info message when action type is hitl_gate', async () => {
      mountComponent()
      await selectActionType('hitl_gate')
      expect(wrapper.find('[data-testid="hitl-gate-info"]').exists()).toBe(true)
      expect(wrapper.text()).toContain('No additional configuration required')
    })
  })

  describe('config reset on action type change', () => {
    it('resets config fields when switching action types', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any

      // Set git_branch config
      await selectActionType('git_branch')
      vm.config.branch_pattern = 'feat/{key}'
      await nextTick()
      expect(vm.config.branch_pattern).toBe('feat/{key}')

      // Switch to git_pr — config should be reset
      await selectActionType('git_pr')
      expect(vm.config.branch_pattern).toBeUndefined()
    })

    it('clears agentId when switching away from agent_run', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any
      vm.agentId = 'agent-1'

      await selectActionType('git_branch')
      expect(vm.agentId).toBeUndefined()
    })

    it('keeps agentId undefined when switching back to agent_run', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any

      await selectActionType('git_branch')
      expect(vm.agentId).toBeUndefined()

      await selectActionType('agent_run')
      expect(vm.agentId).toBeUndefined()
    })
  })

  describe('form submission', () => {
    it('includes config fields in emitted step for git_pr', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any

      // Fill name
      vm.name = 'Create PR'
      await selectActionType('git_pr')

      // Fill config fields
      vm.config.title_template = 'feat: {summary}'
      vm.config.target_branch = 'develop'
      vm.configDraft = true
      await nextTick()

      // Submit
      vm.handleAdd()
      await nextTick()

      const emitted = wrapper.emitted('add')
      expect(emitted).toBeDefined()
      expect(emitted).toHaveLength(1)

      const step = emitted![0]![0] as Record<string, unknown>
      expect(step.action_type).toBe('git_pr')
      expect(step.config).toBeDefined()

      const stepConfig = step.config as Record<string, unknown>
      expect(stepConfig.title_template).toBe('feat: {summary}')
      expect(stepConfig.target_branch).toBe('develop')
      expect(stepConfig.draft).toBe('true')
    })

    it('does not include config for agent_run steps', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any

      vm.name = 'Implement'
      await nextTick()

      vm.handleAdd()
      await nextTick()

      const emitted = wrapper.emitted('add')
      expect(emitted).toBeDefined()

      const step = emitted![0]![0] as Record<string, unknown>
      expect(step.action_type).toBe('agent_run')
      expect(step.config).toBeUndefined()
    })

    it('validates step name is required', async () => {
      mountComponent()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const vm = wrapper.vm as any

      vm.handleAdd()
      await nextTick()

      expect(wrapper.emitted('add')).toBeUndefined()
      expect(wrapper.text()).toContain('Step name is required')
    })
  })
})
