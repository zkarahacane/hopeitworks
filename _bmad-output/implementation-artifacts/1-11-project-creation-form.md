# Story 1.11: [FRONT] Project creation form

Status: ready-for-dev

## Story

As an admin,
I want to create a new project via a form dialog,
So that I can set up a project and later connect a Git repository.

## Acceptance Criteria (BDD)

**AC1: "New Project" button opens creation dialog**
- **Given** user is authenticated and on the `/projects` page
- **When** they click the "New Project" button in the page header (or the CTA in empty state)
- **Then** a PrimeVue Dialog opens with a form containing "Name" (required) and "Description" (optional) fields

**AC2: Form validation with zod schema**
- **Given** the Create Project dialog is open
- **When** user blurs the "Name" field while it is empty, or enters more than 255 characters
- **Then** inline validation error is displayed below the field
- **And** the "Create" button remains disabled until the form is valid

**AC3: Successful project creation**
- **Given** user fills in a valid project name (and optional description)
- **When** they click "Create"
- **Then** POST `/api/v1/projects` is called with `{ name, description }`
- **And** on 201 success, the dialog closes, a success Toast is shown, and user is redirected to `/projects/:id`

**AC4: API error feedback**
- **Given** user submits the creation form
- **When** POST `/api/v1/projects` returns a non-2xx response (e.g., 400 validation error, 500 server error)
- **Then** an error message is displayed inside the dialog (not a Toast) so user can correct and retry
- **And** the form remains open with values preserved

**AC5: Empty state CTA wires to creation dialog**
- **Given** user is on `/projects` with no projects (empty state is displayed)
- **When** they click "Create your first project" button in `ProjectEmptyState`
- **Then** the Create Project dialog opens (same dialog as AC1)

**AC6: Dialog cancel and close behavior**
- **Given** the Create Project dialog is open
- **When** user clicks "Cancel" or the dialog close icon
- **Then** the dialog closes without making an API call
- **And** form state is reset for next open

> **Note on MVP scope:** The epic acceptance criteria reference `repo_url`, `git_provider`, and `git_token` fields. The current `CreateProjectRequest` schema only has `name` and `description`. Git integration fields will be added in a future story. This implementation covers the current OpenAPI spec.

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `createProject` action to projects store (AC: #3, #4)
  - [ ] Update `frontend/src/stores/projects.ts`
  - [ ] Add `createProject(payload: { name: string; description?: string }): Promise<Project>` action
  - [ ] Call `apiClient.POST('/projects', { body: payload })`
  - [ ] On success, return the created `Project` object (do NOT auto-refresh list — the redirect handles this)
  - [ ] On API error, throw an `Error` with the message from the error response body
  - [ ] Export `CreateProjectPayload` interface: `{ name: string; description?: string }`

- [ ] [FRONT] Task 2: Expose `createProject` in `useProjects` composable (AC: #3, #4)
  - [ ] Update `frontend/src/composables/useProjects.ts`
  - [ ] Import `useAsyncAction` from `@/composables/useAsyncAction`
  - [ ] Wrap `store.createProject` in `useAsyncAction` — exposes `{ execute, isLoading, error }`
  - [ ] Return `createProject` alongside existing exposed values
  - [ ] Signature: `createProject: { execute: (payload) => Promise<Project | null>, isLoading: Ref<boolean>, error: Ref<Error | null> }`

- [ ] [FRONT] Task 3: Create `CreateProjectDialog.vue` component (AC: #1, #2, #3, #4, #6)
  - [ ] Create `frontend/src/features/projects/CreateProjectDialog.vue`
  - [ ] Props: `visible: boolean`
  - [ ] Emits: `update:visible: [value: boolean]`, `created: [project: Project]`
  - [ ] Use PrimeVue `Dialog` with `v-model:visible`, `modal`, header "Create Project"
  - [ ] Form validation via `useForm` from `vee-validate` with `@vee-validate/zod` resolver and zod schema
  - [ ] Zod schema:
    - `name`: `z.string().min(1, 'Project name is required').max(255, 'Name must be 255 characters or fewer')`
    - `description`: `z.string().max(1000, 'Description must be 1000 characters or fewer').optional().or(z.literal(''))`
  - [ ] Form fields: PrimeVue `InputText` for name (with `FloatLabel`), PrimeVue `Textarea` for description (with `FloatLabel`)
  - [ ] Display inline validation errors below each field using a `<small>` with error styling
  - [ ] Footer: "Cancel" button (`severity="secondary"`, `text`) and "Create" button (`severity="success"`, with `loading` prop)
  - [ ] On submit: call `useProjects().createProject.execute(formValues)`, on success emit `created` with returned project, close dialog
  - [ ] On API error: display error message inside dialog (below form, above footer) via PrimeVue `Message`
  - [ ] On dialog close or cancel: reset form via `resetForm()` from vee-validate

- [ ] [FRONT] Task 4: Wire dialog into `ProjectsView.vue` (AC: #1, #3, #5)
  - [ ] Update `frontend/src/views/ProjectsView.vue`
  - [ ] Add `CreateProjectDialog` import and template
  - [ ] Add `showCreateDialog` ref (boolean, default `false`)
  - [ ] Replace the disabled "New Project" button with an enabled one that sets `showCreateDialog = true`
  - [ ] Wire `ProjectEmptyState` `@create` to set `showCreateDialog = true` (replace the existing `router.push` handler)
  - [ ] Wire `CreateProjectDialog` `@created` handler: show success Toast via `useToast()`, then `router.push({ name: 'project-detail', params: { id: project.id } })`
  - [ ] Import and use PrimeVue `Toast` component in template and `useToast()` composable

- [ ] [FRONT] Task 5: Ensure PrimeVue ToastService is registered (AC: #3)
  - [ ] Check `frontend/src/main.ts` for `ToastService` registration
  - [ ] If not already present, add `import ToastService from 'primevue/toastservice'` and `app.use(ToastService)`
  - [ ] This may already be done by Story 1-13 — verify before duplicating

- [ ] [FRONT] Task 6: Unit tests for store `createProject` action (AC: #3, #4)
  - [ ] Update `frontend/src/stores/__tests__/projects.spec.ts` (or create if not yet present)
  - [ ] Test `createProject` success: mock `apiClient.POST` returning 201 with project data, assert returned project matches
  - [ ] Test `createProject` API error: mock `apiClient.POST` returning 400 with error body, assert thrown error contains message
  - [ ] Test `createProject` network error: mock `apiClient.POST` throwing, assert error propagates
  - [ ] Use `createPinia()` + `setActivePinia()` in `beforeEach`
  - [ ] Mock `apiClient` via `vi.mock('@/api/client')`

- [ ] [FRONT] Task 7: Unit tests for composable `createProject` wrapper (AC: #3, #4)
  - [ ] Update `frontend/src/composables/__tests__/useProjects.spec.ts` (or create if not yet present)
  - [ ] Test `createProject.execute` success: `isLoading` transitions from `false` -> `true` -> `false`, `error` is `null`, returns project
  - [ ] Test `createProject.execute` failure: `isLoading` transitions, `error` is set, returns `null`
  - [ ] Mock store via `vi.mock('@/stores/projects')`

- [ ] [FRONT] Task 8: Component unit tests for `CreateProjectDialog` (AC: #1, #2, #4, #6)
  - [ ] Create `frontend/src/features/projects/__tests__/CreateProjectDialog.spec.ts`
  - [ ] Test dialog renders when `visible=true` with Name and Description fields
  - [ ] Test Name field validation: submit with empty name shows error message
  - [ ] Test successful submission: fill name, click Create, assert `created` event emitted with project data
  - [ ] Test cancel button emits `update:visible` with `false`
  - [ ] Test API error is displayed inside dialog (not thrown)
  - [ ] Use `@vue/test-utils` `mount` with PrimeVue plugin configured
  - [ ] Mock `useProjects` composable via `vi.mock('@/composables/useProjects')`

- [ ] [FRONT] Task 9: Update `ProjectsView` tests (AC: #1, #5)
  - [ ] Update `frontend/src/views/__tests__/ProjectsView.spec.ts` (or create)
  - [ ] Test "New Project" button click opens dialog (`showCreateDialog` becomes true)
  - [ ] Test `ProjectEmptyState` `@create` opens dialog
  - [ ] Test `CreateProjectDialog` `@created` triggers navigation to `/projects/:id`

- [ ] [FRONT] Task 10: E2E test with Playwright (AC: #1, #2, #3, #5)
  - [ ] Create `frontend/e2e/tests/project-creation.spec.ts`
  - [ ] Test: click "New Project" button -> dialog opens with Name and Description fields
  - [ ] Test: submit empty form -> validation error on Name field
  - [ ] Test: fill name, submit -> API called, dialog closes, navigated to `/projects/:id` (mock API)
  - [ ] Test: on empty state, click CTA -> dialog opens
  - [ ] Use Playwright route interception (`page.route()`) to mock POST `/api/v1/projects`

## Dev Notes

This story replaces the disabled "New Project" button in `ProjectsView` with a functional creation dialog. It establishes the pattern for all future form dialogs in the application (vee-validate + zod in a PrimeVue Dialog).

### Dependencies

**Story dependencies (already implemented):**
- Story 1-7: Vue 3 scaffold, PrimeVue 4, Tailwind CSS v4
- Story 1-8: App shell with AppSidebar
- Story 1-9: Auth guard, vee-validate + @vee-validate/zod + zod installed
- Story 1-10: ProjectsView with DataTable, ProjectEmptyState (has `@create` emit), useProjects composable, projectsStore
- Story 1-16: Router (with `/projects/:id` route as `project-detail`), Pinia, apiClient, useAsyncAction

**Backend peer (can proceed in parallel):**
- Story 1-5: POST `/api/v1/projects` endpoint. Frontend can be built against mock API responses if backend is not ready yet.

**npm packages (already installed):**
- `pinia`, `vee-validate`, `@vee-validate/zod`, `zod`, `openapi-fetch`, `primevue`, `date-fns`

**No new npm packages required.**

### Architecture Requirements

**Component Hierarchy:**
```
ProjectsView.vue (route: /projects)
├── Page header ("Projects" + "New Project" button)
├── ProgressSpinner (v-if loading)
├── Message (v-if error)
├── ProjectEmptyState (v-if empty) — @create opens dialog
├── ProjectListTable (v-if data)
├── CreateProjectDialog (v-model:visible="showCreateDialog")
│   ├── PrimeVue Dialog (modal)
│   │   ├── InputText (name) with FloatLabel
│   │   ├── Textarea (description) with FloatLabel
│   │   ├── Message (API error, v-if)
│   │   └── Footer: Cancel + Create buttons
└── Toast (success feedback)
```

**Data Flow:**
```
ProjectsView (orchestrator)
  │
  ├─ showCreateDialog ref (boolean)
  │
  ├─ CreateProjectDialog
  │    ├─ useForm (vee-validate) + zod schema
  │    ├─ useProjects().createProject (useAsyncAction)
  │    │    └─ useProjectsStore().createProject
  │    │         └─ apiClient.POST('/projects')
  │    └─ emits: created → ProjectsView → router.push + toast
  │
  └─ useToast() — PrimeVue toast service
```

### File Paths (exact)

| File | Action |
|------|--------|
| `frontend/src/stores/projects.ts` | Update (add `createProject` action + `CreateProjectPayload` type) |
| `frontend/src/composables/useProjects.ts` | Update (expose `createProject` via `useAsyncAction`) |
| `frontend/src/features/projects/CreateProjectDialog.vue` | Create |
| `frontend/src/views/ProjectsView.vue` | Update (wire dialog, enable button, add Toast) |
| `frontend/src/main.ts` | Update if needed (ensure ToastService registered) |
| `frontend/src/stores/__tests__/projects.spec.ts` | Update (add createProject tests) |
| `frontend/src/composables/__tests__/useProjects.spec.ts` | Update (add createProject tests) |
| `frontend/src/features/projects/__tests__/CreateProjectDialog.spec.ts` | Create |
| `frontend/src/views/__tests__/ProjectsView.spec.ts` | Create or update |
| `frontend/e2e/tests/project-creation.spec.ts` | Create |

### Technical Specifications

**Store — `createProject` action addition:**

```typescript
// frontend/src/stores/projects.ts — additions to existing store

export interface CreateProjectPayload {
  name: string
  description?: string
}

// Inside defineStore('projects', () => { ... })

async function createProject(payload: CreateProjectPayload): Promise<Project> {
  const { data, error: apiError } = await apiClient.POST('/projects', {
    body: payload,
  })
  if (apiError) {
    const message = (apiError as { error?: { message?: string } })?.error?.message
      ?? 'Failed to create project'
    throw new Error(message)
  }
  return data as Project
}

// Add createProject to the return statement
return { items, pagination, isLoading, error, fetchProjects, createProject, reset }
```

**Composable — `createProject` exposure:**

```typescript
// frontend/src/composables/useProjects.ts — additions

import { useAsyncAction } from '@/composables/useAsyncAction'

export function useProjects() {
  const store = useProjectsStore()
  const lastParams = ref<FetchProjectsParams>({})

  async function fetchProjects(params: FetchProjectsParams = {}) {
    lastParams.value = params
    await store.fetchProjects(params)
  }

  async function retry() {
    await store.fetchProjects(lastParams.value)
  }

  const createProject = useAsyncAction(store.createProject)

  return {
    projects: computed(() => store.items),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    error: computed(() => store.error),
    fetchProjects,
    retry,
    createProject,
  }
}
```

**CreateProjectDialog.vue — full component:**

```vue
<script setup lang="ts">
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useProjects } from '@/composables/useProjects'
import type { Project } from '@/stores/projects'

defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  created: [project: Project]
}>()

const createProjectSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Project name is required').max(255, 'Name must be 255 characters or fewer'),
    description: z.string().max(1000, 'Description must be 1000 characters or fewer').optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: createProjectSchema,
})

const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')

const { createProject } = useProjects()

const onSubmit = handleSubmit(async (values) => {
  const project = await createProject.execute(values.name, values.description || undefined)
  if (project) {
    emit('created', project as Project)
    close()
  }
})

function close() {
  resetForm()
  createProject.error.value = null
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Create Project"
    class="w-full max-w-lg"
    @update:visible="close"
  >
    <form class="flex flex-col gap-6" @submit.prevent="onSubmit">
      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="project-name"
            v-model="name"
            v-bind="nameAttrs"
            class="w-full"
            :invalid="!!errors.name"
          />
          <label for="project-name">Name *</label>
        </FloatLabel>
        <small v-if="errors.name" class="text-red-500">{{ errors.name }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Textarea
            id="project-description"
            v-model="description"
            v-bind="descriptionAttrs"
            class="w-full"
            rows="3"
            :invalid="!!errors.description"
          />
          <label for="project-description">Description</label>
        </FloatLabel>
        <small v-if="errors.description" class="text-red-500">{{ errors.description }}</small>
      </div>

      <Message v-if="createProject.error.value" severity="error" :closable="false">
        {{ createProject.error.value.message }}
      </Message>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="close" />
        <Button
          label="Create"
          severity="success"
          icon="pi pi-check"
          :loading="createProject.isLoading.value"
          @click="onSubmit"
        />
      </div>
    </template>
  </Dialog>
</template>
```

**ProjectsView.vue — updated wiring:**

```vue
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import type { DataTablePageEvent } from 'primevue/datatable'
import Button from 'primevue/button'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'
import Toast from 'primevue/toast'
import ProjectListTable from '@/features/projects/ProjectListTable.vue'
import ProjectEmptyState from '@/features/projects/ProjectEmptyState.vue'
import CreateProjectDialog from '@/features/projects/CreateProjectDialog.vue'
import { useProjects } from '@/composables/useProjects'
import type { Project } from '@/stores/projects'

const router = useRouter()
const toast = useToast()
const { projects, pagination, isLoading, error, fetchProjects, retry } = useProjects()

const perPage = 20
const first = ref(0)
const showCreateDialog = ref(false)

onMounted(() => {
  fetchProjects({ page: 1, per_page: perPage })
})

function handlePage(event: DataTablePageEvent) {
  const newPage = Math.floor(event.first / event.rows) + 1
  first.value = event.first
  fetchProjects({ page: newPage, per_page: event.rows })
}

function handleRowClick(project: Project) {
  router.push({ name: 'project-detail', params: { id: project.id } })
}

function handleCreated(project: Project) {
  toast.add({
    severity: 'success',
    summary: 'Project created',
    detail: `"${project.name}" has been created successfully`,
    life: 3000,
  })
  router.push({ name: 'project-detail', params: { id: project.id } })
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Projects</h1>
      <Button
        label="New Project"
        icon="pi pi-plus"
        severity="success"
        @click="showCreateDialog = true"
      />
    </div>

    <ProgressSpinner
      v-if="isLoading && projects.length === 0"
      class="flex justify-center"
    />

    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <ProjectEmptyState
      v-else-if="!isLoading && !error && projects.length === 0"
      @create="showCreateDialog = true"
    />

    <ProjectListTable
      v-else
      :projects="projects"
      :total-records="pagination?.total ?? 0"
      :rows="perPage"
      :loading="isLoading"
      :first="first"
      @page="handlePage"
      @row-click="handleRowClick"
    />

    <CreateProjectDialog
      v-model:visible="showCreateDialog"
      @created="handleCreated"
    />

    <Toast />
  </div>
</template>
```

### PrimeVue Components Used

- `Dialog` — modal form container
- `InputText` — project name field
- `Textarea` — project description field
- `FloatLabel` — animated labels for form fields
- `Button` — Cancel, Create, New Project actions
- `Message` — API error display inside dialog (severity="error")
- `Toast` — success feedback after creation
- `ProgressSpinner` — loading state (existing)

### PrimeVue Services Required

`ToastService` must be registered in `main.ts`:
```typescript
import ToastService from 'primevue/toastservice'
app.use(ToastService)
```

### API Endpoints

| Method | Path | Request Body | Response |
|--------|------|-------------|----------|
| POST | /api/v1/projects | `{ name: string, description?: string }` | 201: `Project` |
| GET | /api/v1/projects | (query: `page`, `per_page`, `sort_by`) | 200: `{ data: Project[], pagination }` |

### Zod Schema (matching OpenAPI CreateProjectRequest)

```typescript
const createProjectSchema = z.object({
  name: z.string()
    .min(1, 'Project name is required')
    .max(255, 'Name must be 255 characters or fewer'),
  description: z.string()
    .max(1000, 'Description must be 1000 characters or fewer')
    .optional()
    .or(z.literal('')),
})
```

### Style Conventions

- PrimeVue components for all form elements — no native HTML inputs
- Tailwind for layout only (flex, gap, padding, width)
- Zero `<style scoped>` blocks
- No custom CSS classes
- Validation error text uses PrimeVue error color via `text-red-500` or `p-error` class

### Future Git Integration Note

When the Git integration story lands, the following changes will be needed in this dialog:
- Add `repo_url` field (InputText) with URL format validation
- Add `git_provider` field (auto-detected from URL or Select dropdown)
- Add `git_token` field (Password component)
- Add "Validate Connection" button that calls a connection test endpoint
- Update the zod schema and OpenAPI spec accordingly

These fields are NOT part of this story's scope.

### Testing Requirements

**Unit tests (Vitest):**

| Test file | What to test | Coverage target |
|-----------|-------------|----------------|
| `stores/__tests__/projects.spec.ts` | `createProject` success returns Project, error throws with message | 90%+ |
| `composables/__tests__/useProjects.spec.ts` | `createProject.execute` loading transitions, error propagation | 90%+ |
| `features/projects/__tests__/CreateProjectDialog.spec.ts` | Renders fields, validation errors, submit emits, cancel resets, API error display | As needed |
| `views/__tests__/ProjectsView.spec.ts` | Button opens dialog, empty state opens dialog, created event navigates | As needed |

**E2E tests (Playwright):**
- Open dialog via "New Project" button
- Submit empty form -> validation error displayed
- Submit valid form -> API called, dialog closes, navigation occurs
- Open dialog via empty state CTA

**Manual verification checklist:**
1. `npm run dev` -> navigate to `/projects` -> click "New Project" -> dialog opens
2. Submit with empty name -> validation error below name field
3. Fill name "Test Project", optionally add description -> click Create -> dialog closes, toast shows, navigated to `/projects/:id`
4. Navigate back to `/projects` -> new project appears in list
5. With empty project list -> click "Create your first project" CTA -> same dialog opens
6. Open dialog -> click Cancel -> dialog closes, no API call made
7. Open dialog -> close via X button -> dialog closes, form reset
8. `npm run build` -> no TypeScript errors
9. `npm run lint` -> no lint errors
10. `npm run test:unit` -> all new tests pass

### References

- [Source: api/openapi.yaml -- POST /projects endpoint, CreateProjectRequest schema, Project schema]
- [Source: _bmad-output/planning-artifacts/epics.md -- Epic 1, Story 1.11]
- [Source: _bmad-output/planning-artifacts/architecture.md -- Frontend component organization]
- [Source: _bmad-output/implementation-artifacts/1-10-project-list-page.md -- ProjectsView, ProjectEmptyState, useProjects, projectsStore]
- [Source: _bmad-output/implementation-artifacts/1-13-user-management-page-admin-only.md -- CreateUserDialog pattern with vee-validate + zod]
- [Source: _bmad-output/implementation-artifacts/1-9-login-page-auth-guard.md -- vee-validate + zod form validation pattern]
- [Source: _bmad-output/implementation-artifacts/1-16-vue-app-routing-state-tooling.md -- useAsyncAction, apiClient, router]
- [Source: frontend/CLAUDE.md -- PrimeVue Dialog patterns, useAsyncAction pattern, composable conventions, zero style blocks]

## Dev Agent Record

## Change Log
