import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import RunsView from '../RunsView.vue'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

// RunsView is the global /runs view: it only navigates via useRouter and has no
// project scope (no route params), unlike ProjectRunsView.
vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...(actual as object),
    useRouter: () => ({ push: vi.fn() }),
  }
})

// useRecentRuns is the data source the Cost column renders from.
const mockRuns = ref<RunSummary[]>([])
const mockIsLoading = ref(false)
const mockError = ref<Error | null>(null)
vi.mock('@/features/runs/composables/useRecentRuns', () => ({
  useRecentRuns: () => ({
    runs: mockRuns,
    isLoading: mockIsLoading,
    error: mockError,
    refresh: vi.fn(),
  }),
}))

function mountView() {
  return mount(RunsView, {
    global: { plugins: [[PrimeVue, { unstyled: true }]] },
  })
}

// Cell text for the run row identified by its truncated id (the "Run ID" column
// renders id.substring(0, 8)). Returns the trimmed text of the whole row.
function rowText(wrapper: ReturnType<typeof mountView>, runId: string): string {
  const row = wrapper
    .findAll('tbody tr')
    .find((tr) => tr.text().includes(runId.substring(0, 8)))
  return row ? row.text() : ''
}

describe('RunsView — global Cost column (#290)', () => {
  beforeEach(() => {
    mockIsLoading.value = false
    mockError.value = null
    mockRuns.value = []
  })

  it('renders the aggregated cost as $0.8145 for a run with a cost', async () => {
    mockRuns.value = [
      {
        id: 'run-cost',
        project_id: 'proj-1',
        story_id: 'story-1',
        status: 'completed',
        progress: 100,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T11:00:00Z',
        project_name: 'Alpha',
        story_key: 'S-01',
        cost_usd: 0.8145,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-cost')).toContain('$0.8145')
  })

  it('renders an em dash when the run has no cost record', async () => {
    mockRuns.value = [
      {
        id: 'run-nocost',
        project_id: 'proj-1',
        story_id: 'story-2',
        status: 'pending',
        progress: 0,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T10:00:00Z',
        project_name: 'Alpha',
        story_key: 'S-02',
        cost_usd: null,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    const text = rowText(wrapper, 'run-nocost')
    expect(text).toContain('—')
    expect(text).not.toContain('$')
  })

  it('renders a real $0.00 distinct from no-data', async () => {
    mockRuns.value = [
      {
        id: 'run-zero',
        project_id: 'proj-1',
        story_id: 'story-3',
        status: 'completed',
        progress: 100,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T11:00:00Z',
        project_name: 'Alpha',
        story_key: 'S-03',
        cost_usd: 0,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-zero')).toContain('$0.00')
  })
})
