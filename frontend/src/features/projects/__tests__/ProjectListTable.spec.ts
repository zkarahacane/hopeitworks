import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import ProjectListTable from '../ProjectListTable.vue'
import type { Project } from '@/stores/projects'

vi.mock('@/utils/formatDate', () => ({
  formatRelativeDate: (date: string) => `mocked-${date}`,
}))

const sampleProjects: Project[] = [
  {
    id: '1',
    name: 'Alpha Project',
    description: 'First project description',
    owner_id: 'u1',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: '2',
    name: 'Beta Project',
    owner_id: 'u1',
    created_at: '2026-02-01T08:00:00Z',
    updated_at: '2026-02-01T08:00:00Z',
  },
]

function mountComponent(props: Partial<InstanceType<typeof ProjectListTable>['$props']> = {}) {
  return mount(ProjectListTable, {
    props: {
      projects: sampleProjects,
      totalRecords: sampleProjects.length,
      rows: 20,
      loading: false,
      first: 0,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('ProjectListTable', () => {
  it('renders column headers', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('Name')
    expect(text).toContain('Description')
    expect(text).toContain('Created')
  })

  it('renders project names in the table', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('Alpha Project')
    expect(text).toContain('Beta Project')
  })

  it('renders project descriptions', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('First project description')
  })

  it('renders dash for missing description', () => {
    const wrapper = mountComponent()
    // Beta Project has no description, should show '-'
    const text = wrapper.text()
    expect(text).toContain('-')
  })

  it('renders formatted dates using formatRelativeDate', () => {
    const wrapper = mountComponent()
    const text = wrapper.text()
    expect(text).toContain('mocked-2026-01-15T10:00:00Z')
    expect(text).toContain('mocked-2026-02-01T08:00:00Z')
  })

  it('emits row-click with project data when a row is clicked', async () => {
    const wrapper = mountComponent()
    const rows = wrapper.findAll('tr[data-p-index]')
    expect(rows.length).toBeGreaterThan(0)
    await rows[0]!.trigger('click')
    const emitted = wrapper.emitted('row-click')
    expect(emitted).toBeDefined()
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toEqual(sampleProjects[0])
  })

  it('shows loading state when loading prop is true', () => {
    const wrapper = mountComponent({ loading: true })
    // PrimeVue DataTable adds a loading overlay when loading is true
    const loadingIcon = wrapper.find('[data-pc-section="loadingicon"]')
    expect(loadingIcon.exists()).toBe(true)
  })
})
