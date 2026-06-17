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

  it('routes status through StatusBadge (statusToken family)', () => {
    mountRow(makeStep({ status: 'failed' }))
    const badge = wrapper.find('[data-testid="status-tag"]')
    expect(badge.exists()).toBe(true)
    // statusToken normalizes "failed" → failed family.
    expect(badge.attributes('data-family')).toBe('failed')
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

  it('marks the row selected via data-selected when selected', () => {
    mountRow(makeStep(), true)
    const row = wrapper.find('[data-testid="job-row"]')
    expect(row.attributes('data-selected')).toBe('true')
  })

  it('does not mark the row selected when not selected', () => {
    mountRow(makeStep(), false)
    const row = wrapper.find('[data-testid="job-row"]')
    expect(row.attributes('data-selected')).toBe('false')
  })

  it('renders an AgentChip for agent_run steps', () => {
    mountRow(makeStep({ action: 'agent_run' }))
    expect(wrapper.find('[data-testid="job-agent-chip"]').exists()).toBe(true)
  })

  it('renders a ContainerChip for non-agent steps that have a container', () => {
    mountRow(makeStep({ action: 'git_branch', container_id: 'abc123def456' }))
    expect(wrapper.find('[data-testid="job-agent-chip"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="job-container-chip"]').exists()).toBe(true)
  })

  it('renders a type icon matching the step action', () => {
    const cases = [
      { action: 'git_branch', icon: 'pi-code' },
      { action: 'human', icon: 'pi-user' },
      { action: 'git_pr', icon: 'pi-github' },
      { action: 'notify', icon: 'pi-bell' },
    ] as const
    for (const { action, icon } of cases) {
      mountRow(makeStep({ action }))
      const typeIcon = wrapper.find('[data-testid="step-type-icon"] i')
      expect(typeIcon.classes().join(' ')).toContain(icon)
      wrapper.unmount()
    }
  })
})
