import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunJobRow from '../RunJobRow.vue'
import type { RunStep } from '../composables/useRunDetail'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeStep(overrides?: Partial<RunStep>): RunStep {
  return {
    id: 'step-1',
    run_id: 'run-1',
    step_name: 'dev-story',
    step_order: 0,
    action: 'agent_run',
    status: 'completed',
    created_at: '2026-02-17T10:00:00Z',
    ...overrides,
  }
}

function mountRow(step: RunStep, selected = false) {
  wrapper = mount(RunJobRow, {
    props: { step, selected },
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('RunJobRow', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders step name', () => {
    mountRow(makeStep({ step_name: 'code-review' }))
    expect(wrapper.find('[data-testid="step-name"]').text()).toBe('code-review')
  })

  it('renders status tag', () => {
    mountRow(makeStep({ status: 'failed' }))
    const tag = wrapper.find('[data-testid="status-tag"]')
    expect(tag.exists()).toBe(true)
    expect(tag.text()).toBe('failed')
  })

  it('renders duration as "--" for pending steps', () => {
    mountRow(makeStep({ status: 'pending', started_at: undefined }))
    expect(wrapper.find('[data-testid="duration"]').text()).toBe('--')
  })

  it('renders formatted duration for completed steps', () => {
    mountRow(
      makeStep({
        status: 'completed',
        started_at: '2026-01-01T10:00:00Z',
        completed_at: '2026-01-01T10:01:30Z',
      }),
    )
    expect(wrapper.find('[data-testid="duration"]').text()).toBe('01:30')
  })

  it('emits click event when clicked', async () => {
    const step = makeStep()
    mountRow(step)
    await wrapper.find('[data-testid="job-row"]').trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
    expect(wrapper.emitted('click')![0]).toEqual([step])
  })

  it('applies selected styling when selected is true', () => {
    mountRow(makeStep(), true)
    const row = wrapper.find('[data-testid="job-row"]')
    expect(row.classes()).toContain('bg-primary-50')
  })

  it('does not apply selected styling when selected is false', () => {
    mountRow(makeStep(), false)
    const row = wrapper.find('[data-testid="job-row"]')
    expect(row.classes()).not.toContain('bg-primary-50')
  })

  it('applies running-indicator class for running status', () => {
    mountRow(makeStep({ status: 'running' }))
    const icon = wrapper.find('[data-testid="status-icon"]')
    expect(icon.classes()).toContain('running-indicator')
  })

  it('does not apply running-indicator class for non-running status', () => {
    mountRow(makeStep({ status: 'completed' }))
    const icon = wrapper.find('[data-testid="status-icon"]')
    expect(icon.classes()).not.toContain('running-indicator')
  })

  it('renders correct icon for each status', () => {
    const statuses = [
      { status: 'pending', icon: 'pi-clock' },
      { status: 'running', icon: 'pi-spinner' },
      { status: 'completed', icon: 'pi-check-circle' },
      { status: 'failed', icon: 'pi-times-circle' },
      { status: 'cancelled', icon: 'pi-minus-circle' },
    ] as const

    for (const { status, icon } of statuses) {
      mountRow(makeStep({ status }))
      const iconEl = wrapper.find('[data-testid="status-icon"]')
      expect(iconEl.classes().join(' ')).toContain(icon)
      wrapper.unmount()
    }
  })
})
