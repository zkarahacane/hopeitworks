import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ProjectRunsView from '../ProjectRunsView.vue'
import type { RunSummary } from '@/features/runs/composables/useRecentRuns'

// Route stub: ProjectRunsView reads route.params.id for the project scope.
vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal()
  return {
    ...(actual as object),
    useRoute: () => ({ params: { id: 'proj-1' } }),
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
  return mount(ProjectRunsView, {
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

describe('ProjectRunsView — Cost column (#290)', () => {
  beforeEach(() => {
    mockIsLoading.value = false
    mockError.value = null
    mockRuns.value = []
  })

  it('renders the aggregated cost as $0.8145 for a run with a cost (RG1)', async () => {
    mockRuns.value = [
      {
        id: 'run-cost',
        project_id: 'proj-1',
        story_id: 'story-1',
        status: 'completed',
        progress: 100,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T11:00:00Z',
        story_key: 'S-01',
        cost_usd: 0.8145,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-cost')).toContain('$0.8145')
  })

  it('renders an em dash when the run has no cost record (RG2)', async () => {
    mockRuns.value = [
      {
        id: 'run-nocost',
        project_id: 'proj-1',
        story_id: 'story-2',
        status: 'pending',
        progress: 0,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T10:00:00Z',
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

  it('renders a real $0.00 distinct from no-data (RG2 boundary)', async () => {
    mockRuns.value = [
      {
        id: 'run-zero',
        project_id: 'proj-1',
        story_id: 'story-3',
        status: 'completed',
        progress: 100,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T11:00:00Z',
        story_key: 'S-03',
        cost_usd: 0,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-zero')).toContain('$0.00')
  })

  it('renders the server-summed multi-step cost as a single value (RG3)', async () => {
    // The backend already sums all steps; the view renders the one aggregated
    // number it receives. 0.5000 + 0.3145 = 0.8145.
    mockRuns.value = [
      {
        id: 'run-multi',
        project_id: 'proj-1',
        story_id: 'story-4',
        status: 'completed',
        progress: 100,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T11:00:00Z',
        story_key: 'S-04',
        cost_usd: 0.5 + 0.3145,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-multi')).toContain('$0.8145')
  })

  it('renders the cumulative cost of a running run (RG5)', async () => {
    mockRuns.value = [
      {
        id: 'run-live',
        project_id: 'proj-1',
        story_id: 'story-5',
        status: 'running',
        progress: 50,
        created_at: '2026-02-17T10:00:00Z',
        updated_at: '2026-02-17T10:30:00Z',
        story_key: 'S-05',
        cost_usd: 0.2,
      },
    ]
    const wrapper = mountView()
    await flushPromises()

    expect(rowText(wrapper, 'run-live')).toContain('$0.20')
  })
})
