import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunStatusIndicator from '../RunStatusIndicator.vue'

vi.mock('@vueuse/core', () => ({
  useIntervalFn: vi.fn(() => ({
    pause: vi.fn(),
    resume: vi.fn(),
    isActive: { value: true },
  })),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(props: {
  status: 'running' | 'completed' | 'failed' | 'paused' | 'backlog' | null
  completedAt?: string
  errorMessage?: string
}) {
  wrapper = mount(RunStatusIndicator, {
    props,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunStatusIndicator', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('running state', () => {
    it('renders a spinner and "Running..." text in blue', () => {
      mountComponent({ status: 'running' })

      const spinner = wrapper.find('[data-testid="run-status-spinner"]')
      expect(spinner.exists()).toBe(true)

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Running...')
      expect(text.classes()).toContain('text-blue-500')
    })

    it('does not render an icon', () => {
      mountComponent({ status: 'running' })
      expect(wrapper.find('[data-testid="run-status-icon"]').exists()).toBe(false)
    })
  })

  describe('completed state', () => {
    it('renders a check circle icon in green', () => {
      mountComponent({
        status: 'completed',
        completedAt: new Date().toISOString(),
      })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-check-circle')
      expect(icon.classes()).toContain('text-green-500')
    })

    it('renders relative time text', () => {
      mountComponent({
        status: 'completed',
        completedAt: new Date().toISOString(),
      })

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.classes()).toContain('text-green-500')
    })

    it('does not render a spinner', () => {
      mountComponent({
        status: 'completed',
        completedAt: new Date().toISOString(),
      })
      expect(wrapper.find('[data-testid="run-status-spinner"]').exists()).toBe(false)
    })
  })

  describe('failed state', () => {
    it('renders a times circle icon in red with "Failed" text', () => {
      mountComponent({ status: 'failed', errorMessage: 'Build error' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-times-circle')
      expect(icon.classes()).toContain('text-red-500')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Failed')
      expect(text.classes()).toContain('text-red-500')
    })

    it('has cursor-pointer class for clickable behavior', () => {
      mountComponent({ status: 'failed', errorMessage: 'Build error' })

      const root = wrapper.find('[data-testid="run-status-indicator"]')
      expect(root.classes()).toContain('cursor-pointer')
    })

    it('emits errorClick when clicked', async () => {
      mountComponent({ status: 'failed', errorMessage: 'Build error' })

      await wrapper.find('[data-testid="run-status-indicator"]').trigger('click')
      expect(wrapper.emitted('errorClick')).toHaveLength(1)
    })
  })

  describe('paused state', () => {
    it('renders a pause circle icon in yellow with "Paused" text', () => {
      mountComponent({ status: 'paused' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-pause-circle')
      expect(icon.classes()).toContain('text-yellow-500')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Paused')
      expect(text.classes()).toContain('text-yellow-500')
    })

    it('does not render a spinner', () => {
      mountComponent({ status: 'paused' })
      expect(wrapper.find('[data-testid="run-status-spinner"]').exists()).toBe(false)
    })

    it('does not have cursor-pointer class', () => {
      mountComponent({ status: 'paused' })

      const root = wrapper.find('[data-testid="run-status-indicator"]')
      expect(root.classes()).not.toContain('cursor-pointer')
    })
  })

  describe('backlog state', () => {
    it('renders a minus circle icon in gray with "Backlog" text', () => {
      mountComponent({ status: 'backlog' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-minus-circle')
      expect(icon.classes()).toContain('text-gray-400')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Backlog')
      expect(text.classes()).toContain('text-gray-400')
    })

    it('does not have cursor-pointer class', () => {
      mountComponent({ status: 'backlog' })

      const root = wrapper.find('[data-testid="run-status-indicator"]')
      expect(root.classes()).not.toContain('cursor-pointer')
    })

    it('does not emit errorClick when clicked', async () => {
      mountComponent({ status: 'backlog' })

      await wrapper.find('[data-testid="run-status-indicator"]').trigger('click')
      expect(wrapper.emitted('errorClick')).toBeUndefined()
    })
  })

  describe('null status', () => {
    it('falls back to backlog display', () => {
      mountComponent({ status: null })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi-minus-circle')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.text()).toBe('Backlog')
    })
  })
})
