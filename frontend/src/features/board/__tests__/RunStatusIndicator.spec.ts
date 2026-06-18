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
    it('renders a spinner and "Running..." text with running color', () => {
      mountComponent({ status: 'running' })

      const spinner = wrapper.find('[data-testid="run-status-spinner"]')
      expect(spinner.exists()).toBe(true)

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Running...')
      expect(text.attributes('style')).toContain('--status-running-color')
    })

    it('does not render an icon', () => {
      mountComponent({ status: 'running' })
      expect(wrapper.find('[data-testid="run-status-icon"]').exists()).toBe(false)
    })
  })

  describe('completed state', () => {
    it('renders a check circle icon with done color', () => {
      mountComponent({
        status: 'completed',
        completedAt: new Date().toISOString(),
      })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-check-circle')
      expect(icon.attributes('style')).toContain('--status-done-color')
    })

    it('renders relative time text with done color', () => {
      mountComponent({
        status: 'completed',
        completedAt: new Date().toISOString(),
      })

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.attributes('style')).toContain('--status-done-color')
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
    it('renders a times circle icon in failed color with "Failed" text', () => {
      mountComponent({ status: 'failed', errorMessage: 'Build error' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-times-circle')
      expect(icon.attributes('style')).toContain('--status-failed-color')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Failed')
      expect(text.attributes('style')).toContain('--status-failed-color')
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
    it('renders a pause circle icon in gate color with "Paused" text', () => {
      mountComponent({ status: 'paused' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-pause-circle')
      expect(icon.attributes('style')).toContain('--status-gate-color')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Paused')
      expect(text.attributes('style')).toContain('--status-gate-color')
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
    it('renders a minus circle icon in queued color with "Backlog" text', () => {
      mountComponent({ status: 'backlog' })

      const icon = wrapper.find('[data-testid="run-status-icon"]')
      expect(icon.exists()).toBe(true)
      expect(icon.classes()).toContain('pi')
      expect(icon.classes()).toContain('pi-minus-circle')
      expect(icon.attributes('style')).toContain('--status-queued-color')

      const text = wrapper.find('[data-testid="run-status-text"]')
      expect(text.exists()).toBe(true)
      expect(text.text()).toBe('Backlog')
      expect(text.attributes('style')).toContain('--status-queued-color')
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
