import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunCostTable from '../RunCostTable.vue'
import type { components } from '@/api/schema'

type RunCostRow = components['schemas']['RunCostRow']

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeRow(overrides?: Partial<RunCostRow>): RunCostRow {
  return {
    run_id: 'run-1',
    story_key: 'S-01',
    status: 'completed',
    started_at: '2026-02-17T10:00:00Z',
    total_cost_usd: 1.5,
    tokens_input: 120000,
    tokens_output: 30000,
    ...overrides,
  }
}

function mountTable(runs: RunCostRow[], isLoading = false) {
  wrapper = mount(RunCostTable, {
    props: { runs, isLoading },
    global: { plugins: [PrimeVue] },
  })
  return wrapper
}

describe('RunCostTable', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  // RG2: real tokens In/Out are rendered (formatted with thousands separators),
  // not 0.
  it('renders the real tokens in/out for a run', () => {
    mountTable([makeRow({ tokens_input: 120000, tokens_output: 30000 })])
    const text = wrapper.text()
    expect(text).toContain('120,000')
    expect(text).toContain('30,000')
  })

  // RG3: empty period shows the explicit empty state, no crash.
  it('shows an empty state when there are no runs', () => {
    mountTable([])
    expect(wrapper.text()).toContain('No runs in this period')
  })

  // Edge case: missing tokens fall back to 0 rather than rendering undefined.
  it('renders 0 when tokens are absent', () => {
    mountTable([makeRow({ tokens_input: undefined, tokens_output: undefined })])
    expect(wrapper.text()).toContain('0')
  })
})
