import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { ref, computed, h, defineComponent } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ProjectSettingsView from '../ProjectSettingsView.vue'
import type { Project } from '@/stores/projects'

const mockPush = vi.fn()

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { id: 'p1' } }),
    useRouter: () => ({ push: mockPush }),
  }
})

const mockToastAdd = vi.fn()
vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: mockToastAdd }),
}))

const mockUpdateExecute = vi.fn()
const mockDeleteProject = vi.fn()
const mockUpdateIsLoading = ref(false)

vi.mock('@/composables/useProjects', () => ({
  useProjects: () => ({
    updateProject: {
      execute: mockUpdateExecute,
      isLoading: mockUpdateIsLoading,
      error: ref<Error | null>(null),
    },
    deleteProject: mockDeleteProject,
  }),
}))

/** Stub the form: it just re-emits delete/save up to the view. */
const ProjectSettingsFormStub = defineComponent({
  name: 'ProjectSettingsForm',
  props: ['project', 'isSaving', 'isDeleting'],
  emits: ['save', 'delete'],
  setup(props, { emit }) {
    return () =>
      h('button', {
        'data-testid': 'stub-delete',
        'data-deleting': String(props.isDeleting),
        onClick: () => emit('delete'),
      })
  },
})

const project = ref<Project | null>({
  id: 'p1',
  name: 'Test Project',
  owner_id: 'u1',
  created_at: 'x',
  updated_at: 'x',
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountView() {
  wrapper = mount(ProjectSettingsView, {
    global: {
      plugins: [PrimeVue, createPinia()],
      provide: { project },
      stubs: {
        ProjectSettingsForm: ProjectSettingsFormStub,
        Toast: true,
      },
    },
  })
  return wrapper
}

describe('ProjectSettingsView delete flow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPush.mockReset()
    mockToastAdd.mockReset()
    mockDeleteProject.mockReset()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  // RG4: success → toast + redirect to /projects
  it('on delete success shows a success toast and redirects to /projects (RG4)', async () => {
    mockDeleteProject.mockResolvedValue(undefined)
    mountView()

    await wrapper.find('[data-testid="stub-delete"]').trigger('click')
    await flushPromises()

    expect(mockDeleteProject).toHaveBeenCalledWith('p1')
    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'success' }),
    )
    expect(mockPush).toHaveBeenCalledWith('/projects')
  })

  // RG5: error → error toast, NO redirect
  it('on delete error shows an error toast and does not redirect (RG5)', async () => {
    mockDeleteProject.mockRejectedValue(new Error('Project not found'))
    mountView()

    await wrapper.find('[data-testid="stub-delete"]').trigger('click')
    await flushPromises()

    expect(mockToastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', detail: 'Project not found' }),
    )
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does not fire a second delete while one is in flight (anti double-click)', async () => {
    let resolveDelete: () => void = () => {}
    mockDeleteProject.mockReturnValue(
      new Promise<void>((resolve) => {
        resolveDelete = resolve
      }),
    )
    mountView()

    const btn = wrapper.find('[data-testid="stub-delete"]')
    await btn.trigger('click')
    await flushPromises()
    // second click while in flight
    await btn.trigger('click')
    await flushPromises()

    resolveDelete()
    await flushPromises()

    expect(mockDeleteProject).toHaveBeenCalledTimes(1)
  })
})
