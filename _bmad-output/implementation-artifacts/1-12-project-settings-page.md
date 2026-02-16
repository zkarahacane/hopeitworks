# Story 1.12: [FRONT] Project settings page

Status: ready-for-dev

## Story

As an admin,
I want to configure project settings,
So that I can control pipeline behavior.

## Acceptance Criteria (BDD)

**AC1: Settings page loads with project data**
- **Given** user navigates to `/projects/:id/settings`
- **When** the page loads
- **Then** GET `/api/v1/projects/{id}` is called and the form displays current project name and description

**AC2: Successful save**
- **Given** admin edits the name or description and clicks Save
- **When** PUT `/api/v1/projects/{id}` returns 200
- **Then** a success Toast is displayed with message "Project settings saved"
- **And** the store is updated with the new values

**AC3: Validation errors on invalid input**
- **Given** admin clears the name field (required, 1-255 chars) or exceeds description limit (1000 chars)
- **When** they blur the field or click Save
- **Then** inline validation errors are displayed via vee-validate + zod

**AC4: Error state on API failure**
- **Given** admin submits the form
- **When** PUT `/api/v1/projects/{id}` returns a non-200 response
- **Then** an error Toast is displayed with an actionable message

**AC5: Loading state while fetching project**
- **Given** user navigates to `/projects/:id/settings`
- **When** the GET call is in progress
- **Then** a loading skeleton or spinner is displayed in place of the form

**AC6: Error state on fetch failure**
- **Given** user navigates to `/projects/:id/settings`
- **When** GET `/api/v1/projects/{id}` returns a non-200 response
- **Then** an error message is displayed with a retry action

**AC7: Breadcrumb navigation**
- **Given** user is on `/projects/:id/settings`
- **When** the page renders
- **Then** a breadcrumb or back link allows navigation to `/projects`

**AC8: Future tabs placeholder**
- **Given** user views the settings page
- **When** the page renders
- **Then** a disabled TabView or info banner indicates that Git, Agent, and Budget settings will be available in future releases

> **Note on MVP scope:** The epic AC references TabView with General, Git, Agent, Budget tabs and read-only mode for non-admins. The current `Project` and `UpdateProjectRequest` schemas only have `name` and `description` fields. This story implements against the actual schema. TabView with additional tabs (Git, Agent, Budget) and role-based read-only mode will be added in future stories when those backend fields and RBAC per-project are available.

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `getProject` and `updateProject` actions to projects store (AC: #1, #2, #5, #6)
  - [ ] Update `frontend/src/stores/projects.ts`
  - [ ] Add state: `currentProject: ref<Project | null>(null)`
  - [ ] Add action: `getProject(id: string)` — calls `apiClient.GET('/projects/{id}', { params: { path: { id } } })`, sets `currentProject` on success, sets `error` on failure
  - [ ] Add action: `updateProject(id: string, payload: { name?: string; description?: string })` — calls `apiClient.PUT('/projects/{id}', { params: { path: { id } }, body: payload })`, returns the updated `Project` on success, throws on failure
  - [ ] Add action: `clearCurrentProject()` — resets `currentProject` to null
  - [ ] Export `UpdateProjectPayload` interface matching `UpdateProjectRequest` from OpenAPI spec

- [ ] [FRONT] Task 2: Extend `useProjects` composable with single-project operations (AC: #1, #2, #5, #6)
  - [ ] Update `frontend/src/composables/useProjects.ts`
  - [ ] Import `useAsyncAction` from `@/composables/useAsyncAction`
  - [ ] Wrap `store.getProject` in `useAsyncAction` — expose as `getProject: { execute, isLoading, error }`
  - [ ] Wrap `store.updateProject` in `useAsyncAction` — expose as `updateProject: { execute, isLoading, error }`
  - [ ] Expose `currentProject` as computed ref from store
  - [ ] Expose `clearCurrentProject` from store

- [ ] [FRONT] Task 3: Create `ProjectSettingsForm.vue` feature component (AC: #1, #2, #3, #8)
  - [ ] Create at `frontend/src/features/projects/ProjectSettingsForm.vue`
  - [ ] Props: `project: Project`, `isSaving: boolean`
  - [ ] Emits: `save: [payload: { name: string; description: string }]`
  - [ ] Use vee-validate `useForm` with zod schema:
    - `name`: `z.string().min(1, 'Project name is required').max(255, 'Name must be 255 characters or less')`
    - `description`: `z.string().max(1000, 'Description must be 1000 characters or less').optional().default('')`
  - [ ] Initialize form values from `project` prop via `watch` + `resetForm`
  - [ ] PrimeVue `InputText` for name, `Textarea` for description (rows: 4)
  - [ ] PrimeVue `FloatLabel` wrappers for both fields
  - [ ] PrimeVue `Button` with label "Save", severity="success", `:loading="isSaving"`, `:disabled="!meta.dirty || !meta.valid"`
  - [ ] Display inline validation errors below each field using PrimeVue `Message` or small text
  - [ ] Future tabs placeholder: a PrimeVue `Message` (severity="info") below the form reading "Git, Agent, and Budget settings will be available in a future release."

- [ ] [FRONT] Task 4: Create `ProjectSettingsView.vue` view (AC: #1, #2, #3, #4, #5, #6, #7, #8)
  - [ ] Create at `frontend/src/views/ProjectSettingsView.vue`
  - [ ] Extract `id` from `useRoute().params.id`
  - [ ] Use `useProjects()` composable — call `getProject.execute(id)` on mount
  - [ ] Cleanup: call `clearCurrentProject()` on `onUnmounted`
  - [ ] Render breadcrumb: PrimeVue `Breadcrumb` with items `[{ label: 'Projects', route: '/projects' }, { label: project.name, route: '/projects/:id' }, { label: 'Settings' }]`
  - [ ] Page header: `<h1>Project Settings</h1>`
  - [ ] Conditional rendering:
    - Loading: PrimeVue `Skeleton` (3 lines) when `getProject.isLoading` and no `currentProject`
    - Error: PrimeVue `Message` severity="error" with retry button when `getProject.error`
    - Data: `ProjectSettingsForm` with `:project="currentProject"` and `:isSaving="updateProject.isLoading"`
  - [ ] On `@save` from form: call `updateProject.execute(id, payload)`, on success show Toast (severity="success", summary="Saved", detail="Project settings saved"), on error show Toast (severity="error", summary="Error", detail=error message)
  - [ ] Use PrimeVue `useToast()` for notifications
  - [ ] Layout: `<div class="flex flex-col gap-6 p-6">`

- [ ] [FRONT] Task 5: Add `/projects/:id/settings` route (AC: #1, #7)
  - [ ] Update `frontend/src/router/index.ts`
  - [ ] Add route:
    ```typescript
    {
      path: '/projects/:id/settings',
      name: 'project-settings',
      component: () => import('@/views/ProjectSettingsView.vue'),
      meta: { requiresAuth: true },
    }
    ```
  - [ ] Place after the `/projects/:id` route to avoid route conflicts
  - [ ] Note: the route uses lazy-loaded import for `ProjectSettingsView`

- [ ] [FRONT] Task 6: Add settings navigation from project detail (AC: #7)
  - [ ] Update `frontend/src/views/ProjectDetailView.vue` — add a PrimeVue `Button` (label="Settings", icon="pi pi-cog", severity="secondary") that routes to `/projects/:id/settings`
  - [ ] Use `router.push({ name: 'project-settings', params: { id } })`

- [ ] [FRONT] Task 7: Unit tests for store additions (AC: #1, #2, #5, #6)
  - [ ] Update `frontend/src/stores/__tests__/projects.spec.ts`
  - [ ] Test `getProject` success: `currentProject` populated with correct data
  - [ ] Test `getProject` error: `error` set, `currentProject` stays null
  - [ ] Test `updateProject` success: returns updated project
  - [ ] Test `updateProject` error: throws with error message
  - [ ] Test `clearCurrentProject`: resets `currentProject` to null
  - [ ] Mock `apiClient` using `vi.mock('@/api/client')`
  - [ ] Use `createPinia()` + `setActivePinia()` in each test

- [ ] [FRONT] Task 8: Unit tests for composable additions (AC: #1, #2)
  - [ ] Update `frontend/src/composables/__tests__/useProjects.spec.ts`
  - [ ] Test `getProject` loading state transitions: `isLoading` true during fetch, false after
  - [ ] Test `updateProject` loading state transitions
  - [ ] Test `currentProject` reactive ref updates when store changes
  - [ ] Test `clearCurrentProject` resets computed value

- [ ] [FRONT] Task 9: Component unit test for ProjectSettingsForm (AC: #1, #2, #3)
  - [ ] Create `frontend/src/features/projects/__tests__/ProjectSettingsForm.spec.ts`
  - [ ] Test form renders with project name and description pre-filled
  - [ ] Test Save button disabled when form is pristine (not dirty)
  - [ ] Test Save button disabled when validation fails (empty name)
  - [ ] Test Save button enabled when form is dirty and valid
  - [ ] Test submitting emits `save` event with form values
  - [ ] Test `isSaving` prop shows loading state on Save button
  - [ ] Test future tabs info message is displayed
  - [ ] Use `@vue/test-utils` `mount` with PrimeVue plugin configured

- [ ] [FRONT] Task 10: E2E test with Playwright (AC: #1, #2, #4)
  - [ ] Create `frontend/e2e/tests/project-settings.spec.ts`
  - [ ] Test: navigate to `/projects/:id/settings` with mocked GET returning project -> form displays name and description
  - [ ] Test: edit name, click Save with mocked PUT returning success -> success toast visible
  - [ ] Test: edit name, click Save with mocked PUT returning 500 -> error toast visible
  - [ ] Test: breadcrumb "Projects" link navigates to `/projects`
  - [ ] Use Playwright route interception (`page.route()`) to mock API responses

## Dev Notes

This story adds the project settings page, establishing the pattern for detail/edit views. It extends the existing projects store and composable with single-project CRUD operations (get by ID, update).

### Dependencies

**Story dependencies (already implemented):**
- Story 1-7: Vue 3 scaffold, PrimeVue 4, Tailwind CSS v4
- Story 1-8: App shell with AppSidebar
- Story 1-9: Auth guard, vee-validate + zod, useAuth composable
- Story 1-10: Project list page, `useProjectsStore` with `fetchProjects`, `useProjects` composable
- Story 1-16: Router with `/projects/:id` route, `useAsyncAction`, `apiClient`

**Backend dependency (peer):**
- Story 1-5: Projects CRUD API (provides GET `/projects/{id}` and PUT `/projects/{id}` endpoints). Frontend can be built against mock API responses.

**npm packages (already installed):**
- `pinia`, `vee-validate`, `@vee-validate/zod`, `zod`, `primevue`, `date-fns`, `openapi-fetch`, `vue-router`

### Architecture Requirements

**Component Hierarchy:**
```
ProjectSettingsView.vue (route: /projects/:id/settings)
├── PrimeVue Breadcrumb
│   └── Projects > {project.name} > Settings
├── <h1> "Project Settings"
├── ProgressSpinner / Skeleton (v-if="getProject.isLoading && !currentProject")
├── Message severity="error" (v-if="getProject.error")
│   └── retry Button
├── ProjectSettingsForm (v-if="currentProject")
│   ├── FloatLabel + InputText (name)
│   ├── FloatLabel + Textarea (description)
│   ├── Button "Save" (severity="success", loading state)
│   └── Message severity="info" (future tabs placeholder)
└── PrimeVue Toast (success/error feedback)
```

**Data Flow:**
```
ProjectSettingsView (orchestrator)
  │
  ├─ useProjects() composable
  │    ├─ getProject: useAsyncAction(store.getProject)
  │    ├─ updateProject: useAsyncAction(store.updateProject)
  │    ├─ currentProject: computed(store.currentProject)
  │    └─ clearCurrentProject: store.clearCurrentProject
  │
  └─ useToast() — PrimeVue toast service
```

### File Paths (exact)

| File | Action |
|------|--------|
| `frontend/src/stores/projects.ts` | Update (add getProject, updateProject, clearCurrentProject, currentProject) |
| `frontend/src/composables/useProjects.ts` | Update (add getProject, updateProject, currentProject, clearCurrentProject) |
| `frontend/src/features/projects/ProjectSettingsForm.vue` | Create |
| `frontend/src/views/ProjectSettingsView.vue` | Create |
| `frontend/src/views/ProjectDetailView.vue` | Update (add settings nav button) |
| `frontend/src/router/index.ts` | Update (add /projects/:id/settings route) |
| `frontend/src/stores/__tests__/projects.spec.ts` | Update (add getProject, updateProject tests) |
| `frontend/src/composables/__tests__/useProjects.spec.ts` | Update (add getProject, updateProject tests) |
| `frontend/src/features/projects/__tests__/ProjectSettingsForm.spec.ts` | Create |
| `frontend/e2e/tests/project-settings.spec.ts` | Create |

### Technical Specifications

**Projects Store — additions:**

```typescript
// frontend/src/stores/projects.ts — additions to existing store
import { ref } from 'vue'
import { apiClient } from '@/api/client'
import type { Project } from './projects' // existing interface

export interface UpdateProjectPayload {
  name?: string
  description?: string
}

// Inside defineStore('projects', () => { ... })

const currentProject = ref<Project | null>(null)

/** Fetch a single project by ID */
async function getProject(id: string) {
  isLoading.value = true
  error.value = null
  try {
    const { data, error: apiError } = await apiClient.GET('/projects/{id}', {
      params: { path: { id } },
    })
    if (apiError) {
      error.value = 'Failed to load project'
      return
    }
    currentProject.value = data as Project
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load project'
  } finally {
    isLoading.value = false
  }
}

/** Update an existing project */
async function updateProject(id: string, payload: UpdateProjectPayload): Promise<Project> {
  const { data, error: apiError } = await apiClient.PUT('/projects/{id}', {
    params: { path: { id } },
    body: payload,
  })
  if (apiError) {
    throw new Error('Failed to update project')
  }
  const updated = data as Project
  currentProject.value = updated
  // Also update in items list if present
  const idx = items.value.findIndex((p) => p.id === id)
  if (idx >= 0) {
    items.value[idx] = updated
  }
  return updated
}

/** Clear the currently loaded project */
function clearCurrentProject() {
  currentProject.value = null
}

// Add to return: currentProject, getProject, updateProject, clearCurrentProject
```

**useProjects Composable — additions:**

```typescript
// frontend/src/composables/useProjects.ts — additions
import { useAsyncAction } from '@/composables/useAsyncAction'

// Inside useProjects():

const getProjectAction = useAsyncAction(
  (id: string) => store.getProject(id)
)

const updateProjectAction = useAsyncAction(
  (id: string, payload: UpdateProjectPayload) => store.updateProject(id, payload)
)

// Add to return:
// currentProject: computed(() => store.currentProject),
// getProject: getProjectAction,
// updateProject: updateProjectAction,
// clearCurrentProject: store.clearCurrentProject,
```

**ProjectSettingsForm.vue:**

```vue
<script setup lang="ts">
import { watch } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import FloatLabel from 'primevue/floatlabel'
import Message from 'primevue/message'
import type { Project } from '@/stores/projects'

const props = defineProps<{
  project: Project
  isSaving: boolean
}>()

const emit = defineEmits<{
  save: [payload: { name: string; description: string }]
}>()

const schema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Project name is required').max(255, 'Name must be 255 characters or less'),
    description: z.string().max(1000, 'Description must be 1000 characters or less').default(''),
  })
)

const { handleSubmit, resetForm, meta } = useForm({
  validationSchema: schema,
  initialValues: {
    name: props.project.name,
    description: props.project.description ?? '',
  },
})

const { value: name, errorMessage: nameError } = useField<string>('name')
const { value: description, errorMessage: descriptionError } = useField<string>('description')

watch(
  () => props.project,
  (newProject) => {
    resetForm({
      values: {
        name: newProject.name,
        description: newProject.description ?? '',
      },
    })
  }
)

const onSubmit = handleSubmit((values) => {
  emit('save', { name: values.name, description: values.description })
})
</script>

<template>
  <form class="flex flex-col gap-6 max-w-xl" @submit.prevent="onSubmit">
    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText id="name" v-model="name" class="w-full" />
        <label for="name">Project Name</label>
      </FloatLabel>
      <small v-if="nameError" class="text-red-500">{{ nameError }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Textarea id="description" v-model="description" rows="4" class="w-full" />
        <label for="description">Description</label>
      </FloatLabel>
      <small v-if="descriptionError" class="text-red-500">{{ descriptionError }}</small>
    </div>

    <div class="flex justify-end">
      <Button
        type="submit"
        label="Save"
        severity="success"
        :loading="isSaving"
        :disabled="!meta.dirty || !meta.valid"
      />
    </div>

    <Message severity="info" :closable="false">
      Git, Agent, and Budget settings will be available in a future release.
    </Message>
  </form>
</template>
```

**ProjectSettingsView.vue:**

```vue
<script setup lang="ts">
import { onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Breadcrumb from 'primevue/breadcrumb'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import Toast from 'primevue/toast'
import ProjectSettingsForm from '@/features/projects/ProjectSettingsForm.vue'
import { useProjects } from '@/composables/useProjects'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const projectId = route.params.id as string

const { currentProject, getProject, updateProject, clearCurrentProject } = useProjects()

const breadcrumbItems = computed(() => [
  { label: 'Projects', route: '/projects' },
  {
    label: currentProject.value?.name ?? 'Project',
    route: `/projects/${projectId}`,
  },
  { label: 'Settings' },
])

const breadcrumbHome = { icon: 'pi pi-home', route: '/' }

onMounted(() => {
  getProject.execute(projectId)
})

onUnmounted(() => {
  clearCurrentProject()
})

async function handleSave(payload: { name: string; description: string }) {
  try {
    await updateProject.execute(projectId, payload)
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: 'Project settings saved',
      life: 3000,
    })
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to save project settings',
      life: 5000,
    })
  }
}

function handleRetry() {
  getProject.execute(projectId)
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <Toast />

    <Breadcrumb :model="breadcrumbItems" :home="breadcrumbHome" />

    <h1 class="text-2xl font-bold">Project Settings</h1>

    <!-- Loading state -->
    <div v-if="getProject.isLoading.value && !currentProject" class="flex flex-col gap-4 max-w-xl">
      <Skeleton height="2.5rem" />
      <Skeleton height="6rem" />
      <Skeleton width="6rem" height="2.5rem" />
    </div>

    <!-- Error state -->
    <div v-else-if="getProject.error.value" class="flex flex-col gap-4 max-w-xl">
      <Message severity="error" :closable="false">
        Failed to load project. Please try again.
      </Message>
      <Button label="Retry" severity="secondary" icon="pi pi-refresh" @click="handleRetry" />
    </div>

    <!-- Settings form -->
    <ProjectSettingsForm
      v-else-if="currentProject"
      :project="currentProject"
      :is-saving="updateProject.isLoading.value"
      @save="handleSave"
    />
  </div>
</template>
```

**Router Addition:**

```typescript
// frontend/src/router/index.ts — add after /projects/:id route
{
  path: '/projects/:id/settings',
  name: 'project-settings',
  component: () => import('@/views/ProjectSettingsView.vue'),
  meta: { requiresAuth: true },
}
```

**ProjectDetailView Update (settings nav button):**

```vue
<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string

function goToSettings() {
  router.push({ name: 'project-settings', params: { id: projectId } })
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Project Detail</h1>
      <Button label="Settings" icon="pi pi-cog" severity="secondary" @click="goToSettings" />
    </div>
  </div>
</template>
```

### Zod Schema

```typescript
// Used inside ProjectSettingsForm.vue
const projectSettingsSchema = z.object({
  name: z.string().min(1, 'Project name is required').max(255, 'Name must be 255 characters or less'),
  description: z.string().max(1000, 'Description must be 1000 characters or less').default(''),
})

type ProjectSettingsFormValues = z.infer<typeof projectSettingsSchema>
```

### Component Props/Emits

| Component | Props | Emits |
|-----------|-------|-------|
| `ProjectSettingsForm` | `project: Project`, `isSaving: boolean` | `save: [{ name: string; description: string }]` |

### PrimeVue Components Used

- `InputText` -- project name field
- `Textarea` -- project description field
- `FloatLabel` -- floating label wrappers
- `Button` -- Save action, Settings nav, Retry
- `Message` -- error display, future tabs info
- `Skeleton` -- loading state
- `Breadcrumb` -- navigation breadcrumb
- `Toast` -- success/error notifications

### PrimeVue Services Required

The view uses PrimeVue `ToastService`. Verify it is registered in `main.ts`:
```typescript
import ToastService from 'primevue/toastservice'
app.use(ToastService)
```

### API Endpoints Used

| Method | Path | Request Body | Response | Used In |
|--------|------|-------------|----------|---------|
| GET | `/api/v1/projects/{id}` | (none) | 200: `Project` | `getProject` |
| PUT | `/api/v1/projects/{id}` | `UpdateProjectRequest` | 200: `Project` | `updateProject` |

### Style Conventions

- PrimeVue components for all interactive/display elements
- Tailwind for layout only (flex, gap, padding, max-w)
- Zero `<style scoped>` blocks
- No custom CSS classes
- No inline styles

### Testing Requirements

**Unit tests (Vitest):**

| Test file | What to test | Coverage target |
|-----------|-------------|----------------|
| `stores/__tests__/projects.spec.ts` | getProject success/error, updateProject success/error, clearCurrentProject | 90%+ |
| `composables/__tests__/useProjects.spec.ts` | getProject/updateProject loading state transitions, currentProject reactivity | 90%+ |
| `features/projects/__tests__/ProjectSettingsForm.spec.ts` | Form pre-fill, validation, dirty state, save emit, isSaving prop, info message | As needed |

**E2E tests (Playwright):**
- Settings form renders with project data (mocked API)
- Save success flow with toast notification (mocked API)
- Save error flow with error toast (mocked API)
- Breadcrumb navigates to project list

**Manual verification checklist:**
1. `npm run dev` -- navigate to `/projects/:id/settings`, see form with project name and description
2. Edit name, click Save -- success toast appears
3. Clear name field -- validation error appears, Save button disabled
4. Enter description > 1000 chars -- validation error appears
5. Breadcrumb "Projects" link navigates to `/projects`
6. Settings button on project detail page navigates to settings
7. Future tabs info message is visible
8. Reload page -- data loads correctly (loading state shown briefly)
9. `npm run build` -- no TypeScript errors
10. `npm run lint` -- no lint errors
11. `npm run test:unit` -- all new tests pass

### References

- [Source: api/openapi.yaml -- GET /projects/{id}, PUT /projects/{id}, Project schema, UpdateProjectRequest schema]
- [Source: _bmad-output/planning-artifacts/epics.md -- Epic 1, Story 1.12]
- [Source: _bmad-output/planning-artifacts/architecture.md -- Frontend hybrid structure, features/ directory]
- [Source: _bmad-output/implementation-artifacts/1-10-project-list-page.md -- Projects store, useProjects composable patterns]
- [Source: _bmad-output/implementation-artifacts/1-16-vue-app-routing-state-tooling.md -- useAsyncAction, apiClient, router setup]
- [Source: _bmad-output/implementation-artifacts/1-9-login-page-auth-guard.md -- vee-validate + zod form pattern]
- [Source: frontend/CLAUDE.md -- PrimeVue patterns, composable conventions, testing patterns]

## Dev Agent Record

## Change Log
