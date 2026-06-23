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

import GuardEditor from '../GuardEditor.vue'
import type { Guard } from '@/stores/pipelineConfig'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(guards: Guard[] = [], props: Record<string, unknown> = {}) {
  wrapper = mount(GuardEditor, {
    props: { guards, isAdmin: true, ...props },
    global: { plugins: [PrimeVue] },
  })
  return wrapper
}

describe('GuardEditor', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  describe('rendering', () => {
    it('shows empty hint when there are no guards', () => {
      mountComponent([])
      expect(wrapper.find('[data-testid="guard-empty"]').exists()).toBe(true)
      expect(wrapper.findAll('[data-testid="guard-row"]')).toHaveLength(0)
    })

    it('renders a row per guard', () => {
      mountComponent([
        { kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' },
        { kind: 'wallclock', max: 1800, on_fail: 'fail' },
      ])
      expect(wrapper.findAll('[data-testid="guard-row"]')).toHaveLength(2)
    })

    it('shows the + Guard button for admin', () => {
      mountComponent([])
      expect(wrapper.find('[data-testid="add-guard"]').exists()).toBe(true)
    })

    it('hides the + Guard and remove buttons for non-admin', () => {
      mountComponent([{ kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' }], {
        isAdmin: false,
      })
      expect(wrapper.find('[data-testid="add-guard"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="remove-guard"]').exists()).toBe(false)
    })
  })

  describe('per-kind unit + hint', () => {
    it('shows seconds for log_silence', () => {
      mountComponent([{ kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' }])
      expect(wrapper.find('[data-testid="guard-unit"]').text()).toBe('s')
      expect(wrapper.find('[data-testid="guard-hint"]').text()).toContain('no agent output')
    })

    it('shows seconds for wallclock', () => {
      mountComponent([{ kind: 'wallclock', max: 1800, on_fail: 'fail' }])
      expect(wrapper.find('[data-testid="guard-unit"]').text()).toBe('s')
      expect(wrapper.find('[data-testid="guard-hint"]').text()).toContain('running longer')
    })

    it('shows USD for cost_batch', () => {
      mountComponent([{ kind: 'cost_batch', max: 5, on_fail: 'halt-gate' }])
      expect(wrapper.find('[data-testid="guard-unit"]').text()).toBe('USD')
      expect(wrapper.find('[data-testid="guard-hint"]').text()).toContain('cumulative run cost')
    })
  })

  describe('events', () => {
    it('emits add on + Guard click', async () => {
      mountComponent([])
      await wrapper.find('[data-testid="add-guard"]').trigger('click')
      expect(wrapper.emitted('add')).toBeTruthy()
    })

    it('emits remove with index on trash click', async () => {
      mountComponent([
        { kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' },
        { kind: 'wallclock', max: 1800, on_fail: 'fail' },
      ])
      const removeButtons = wrapper.findAll('[data-testid="remove-guard"]')
      await removeButtons[1]!.trigger('click')
      expect(wrapper.emitted('remove')).toBeTruthy()
      expect(wrapper.emitted('remove')![0]).toEqual([1])
    })

    it('re-homes the numeric value onto the new kind field when kind changes', () => {
      mountComponent([{ kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' }])
      const vm = wrapper.vm as unknown as {
        onKindChange: (i: number, g: Guard, k: Guard['kind']) => void
      }
      vm.onKindChange(0, { kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' }, 'wallclock')
      expect(wrapper.emitted('update')).toBeTruthy()
      expect(wrapper.emitted('update')![0]).toEqual([
        0,
        { kind: 'wallclock', max: 120, on_fail: 'halt-gate' },
      ])
    })

    it('preserves on_fail when the numeric field changes', () => {
      mountComponent([{ kind: 'wallclock', max: 1800, on_fail: 'fail' }])
      const vm = wrapper.vm as unknown as {
        onNumericChange: (i: number, g: Guard, v: number | null) => void
      }
      vm.onNumericChange(0, { kind: 'wallclock', max: 1800, on_fail: 'fail' }, 600)
      expect(wrapper.emitted('update')![0]).toEqual([
        0,
        { kind: 'wallclock', max: 600, on_fail: 'fail' },
      ])
    })

    it('updates on_fail without touching the numeric field', () => {
      mountComponent([{ kind: 'cost_batch', max: 5, on_fail: 'halt-gate' }])
      const vm = wrapper.vm as unknown as {
        onFailChange: (i: number, g: Guard, f: Guard['on_fail']) => void
      }
      vm.onFailChange(0, { kind: 'cost_batch', max: 5, on_fail: 'halt-gate' }, 'retry')
      expect(wrapper.emitted('update')![0]).toEqual([
        0,
        { kind: 'cost_batch', max: 5, on_fail: 'retry' },
      ])
    })
  })
})
