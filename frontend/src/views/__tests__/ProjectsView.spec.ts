import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed, h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import ProjectsView from '../ProjectsView.vue'

const mockPush = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockPush,
  }),
}))

const mockFetchProjects = vi.fn()
const mockRetry = vi.fn()
const mockCreateExecute = vi.fn()
const mockProjects = ref<unknown[]>([])
const mockIsLoading = ref(false)
const mockError = ref<string | null>(null)
const mockPagination = ref<{ total: number; page: number; per_page: number } | null>(null)
const mockCreateIsLoading = ref(false)
const mockCreateError = ref<Error | null>(null)

vi.mock('@/composables/useProjects', () => ({
  useProjects: () => ({
    projects: computed(() => mockProjects.value),
    pagination: computed(() => mockPagination.value),
    isLoading: computed(() => mockIsLoading.value),
    error: computed(() => mockError.value),
    fetchProjects: mockFetchProjects,
    retry: mockRetry,
    createProject: {
      execute: mockCreateExecute,
      isLoading: mockCreateIsLoading,
      error: mockCreateError,
    },
  }),
}))

/** Stub child components to keep tests focused */
const ProjectListTableStub = defineComponent({
  name: 'ProjectListTable',
  props: ['projects', 'totalRecords', 'rows', 'loading', 'first'],
  emits: ['page', 'row-click'],
  setup(_, { slots }) {
    return () => h('div', { 'data-testid': 'project-list-table' }, slots.default?.())
  },
})

const ProjectEmptyStateStub = defineComponent({
  name: 'ProjectEmptyState',
  emits: ['create'],
  setup(_, { emit }) {
    return () =>
      h(
        'div',
        {
          'data-testid': 'empty-state',
          onClick: () => emit('create'),
        },
        'No projects yet',
      )
  },
})

const CreateProjectDialogStub = defineComponent({
  name: 'CreateProjectDialog',
  props: ['visible'],
  emits: ['update:visible', 'created'],
  setup(props) {
    return () =>
      h('div', {
        'data-testid': 'create-dialog',
        'data-visible': String(props.visible),
      })
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent() {
  wrapper = mount(ProjectsView, {
    global: {
      plugins: [PrimeVue, ToastService, createPinia()],
      stubs: {
        ProjectListTable: ProjectListTableStub,
        ProjectEmptyState: ProjectEmptyStateStub,
        CreateProjectDialog: CreateProjectDialogStub,
        Toast: true,
        ProgressSpinner: true,
      },
    },
  })
  return wrapper
}

describe('ProjectsView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPush.mockReset()
    mockFetchProjects.mockReset()
    mockRetry.mockReset()
    mockCreateExecute.mockReset()
    mockProjects.value = []
    mockIsLoading.value = false
    mockError.value = null
    mockPagination.value = null
    mockCreateIsLoading.value = false
    mockCreateError.value = null
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders page title and New Project button', () => {
    mountComponent()
    expect(wrapper.text()).toContain('Projects')
    expect(wrapper.text()).toContain('New Project')
  })

  it('calls fetchProjects on mount', () => {
    mountComponent()
    expect(mockFetchProjects).toHaveBeenCalledWith({ page: 1, per_page: 20 })
  })

  it('shows empty state when no projects and not loading', async () => {
    mockProjects.value = []
    mockIsLoading.value = false
    mountComponent()
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="empty-state"]').exists()).toBe(true)
  })

  it('opens create dialog when New Project button is clicked', async () => {
    mountComponent()

    const newProjectBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('New Project'))
    await newProjectBtn!.trigger('click')

    await wrapper.vm.$nextTick()

    const dialog = wrapper.find('[data-testid="create-dialog"]')
    expect(dialog.attributes('data-visible')).toBe('true')
  })

  it('opens create dialog when empty state emits create', async () => {
    mockProjects.value = []
    mockIsLoading.value = false
    mountComponent()
    await wrapper.vm.$nextTick()

    const emptyState = wrapper.find('[data-testid="empty-state"]')
    await emptyState.trigger('click')

    await wrapper.vm.$nextTick()

    const dialog = wrapper.find('[data-testid="create-dialog"]')
    expect(dialog.attributes('data-visible')).toBe('true')
  })

  it('navigates to project detail on created event', async () => {
    mountComponent()

    // Open dialog first
    const newProjectBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('New Project'))
    await newProjectBtn!.trigger('click')
    await wrapper.vm.$nextTick()

    // Simulate created event from dialog
    const dialog = wrapper.findComponent(CreateProjectDialogStub)
    dialog.vm.$emit('created', {
      id: 'p1',
      name: 'Test Project',
      owner_id: 'u1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    })

    await flushPromises()

    expect(mockPush).toHaveBeenCalledWith({
      name: 'project-detail',
      params: { id: 'p1' },
    })
  })
})
