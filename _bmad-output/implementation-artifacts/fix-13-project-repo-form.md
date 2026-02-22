# Story fix-13: [FRONT] Expose repo URL and pipeline config fields in project forms

Status: done

## Story

As a user creating or managing a project,
I want to set the GitHub repo URL, git provider, agent runtime, and default model in the project forms,
so that the pipeline has all the configuration it needs to clone the repo and run agents.

## Context

The backend `Project` model has `repo_url`, `git_provider`, `agent_runtime`, and `default_model` fields. Without `repo_url`, no pipeline run can succeed (the agent container has nothing to clone). These fields were never surfaced in the UI. This story adds them to both the creation dialog and the project settings form.

**Dependencies:**
- `fix-10` — adds `repo_url`, `git_provider`, `agent_runtime`, `default_model` to `Project`, `CreateProjectRequest`, and `UpdateProjectRequest` in `api/openapi.yaml`
- `fix-12` — regenerates `frontend/src/api/generated/` from the updated spec so TypeScript types include the new fields

This story must be implemented **after** fix-10 and fix-12 are merged and CI is green on `develop`.

## Acceptance Criteria (BDD)

**AC1: CreateProjectDialog includes the new fields**
- **Given** the user opens the "Create Project" dialog
- **When** the dialog renders
- **Then** four new fields are visible below the existing name/description fields:
  - Repo URL (text input, required, labeled "Repository URL")
  - Git Provider (dropdown, required, options: `github`, default: `github`)
  - Agent Runtime (dropdown, required, options: `docker`, default: `docker`)
  - Default Model (text input, optional, placeholder: `claude-opus-4-5`)

**AC2: CreateProjectDialog validates the new fields**
- **Given** the user submits the create form with an empty Repo URL
- **When** the form is submitted
- **Then** an inline validation error "Repository URL is required" is shown and submission is blocked
- **And** no API call is made

**AC3: CreateProjectDialog sends the new fields to the API**
- **Given** the user fills in all required fields including a valid repo URL
- **When** the user clicks "Create"
- **Then** `POST /api/v1/projects` is called with the body including `repo_url`, `git_provider`, `agent_runtime`, and `default_model`
- **And** on 201, the `created` event is emitted and the dialog closes

**AC4: ProjectOverview displays the new fields**
- **Given** the user navigates to the project overview tab
- **When** the overview card renders
- **Then** the card displays `repo_url`, `git_provider`, `agent_runtime`, and `default_model` in the existing `<dl>` grid alongside name and description

**AC5: ProjectSettingsForm includes the new fields**
- **Given** the user navigates to `/projects/:id/settings`
- **When** the form renders
- **Then** it pre-fills the four new fields from the loaded project data alongside the existing name and description

**AC6: ProjectSettingsForm saves the new fields**
- **Given** the user edits the Repo URL in the settings form and clicks Save
- **When** `PUT /api/v1/projects/{id}` is called
- **Then** the request body includes the updated `repo_url` (and the current values of `git_provider`, `agent_runtime`, `default_model`)
- **And** a success Toast "Project settings saved" is displayed

**AC7: Project store and types include the new fields**
- **Given** the updated generated types from fix-12 are present
- **When** `createProject` or `updateProject` is called with the new fields
- **Then** TypeScript does not produce type errors for the new payload fields

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Update `Project` interface and payload types in the projects store (AC: #7)
  - [ ] In `frontend/src/stores/projects.ts`, extend the local `Project` interface to add:
    - `repo_url?: string`
    - `git_provider?: string`
    - `agent_runtime?: string`
    - `default_model?: string`
  - [ ] Extend `CreateProjectPayload` to add the same four optional fields (note: `repo_url` is required at the app level but optional in the type to match the OpenAPI nullable pattern — validation is enforced in the Zod schema)
  - [ ] Extend `UpdateProjectPayload` to add the same four optional fields
  - [ ] Note: once fix-12 types are generated, the local `Project` interface should be replaced or augmented from `components['schemas']['Project']` — until then, keep the explicit interface

- [ ] [FRONT] Task 2: Update `CreateProjectDialog.vue` — add new fields and Zod schema (AC: #1, #2, #3)
  - [ ] In `frontend/src/features/projects/CreateProjectDialog.vue`, add four new fields to the Zod schema:
    ```typescript
    repo_url: z.string().min(1, 'Repository URL is required').url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
    ```
  - [ ] Add `defineField` bindings for each new field
  - [ ] In `onSubmit`, pass the new fields to `createProject.execute(...)`:
    ```typescript
    repo_url: values.repo_url,
    git_provider: values.git_provider,
    agent_runtime: values.agent_runtime,
    default_model: values.default_model || undefined,
    ```
  - [ ] Add a visual separator (e.g., `<Divider />` or `<hr />`) between the basic info fields (name, description) and the pipeline configuration fields (repo_url, git_provider, agent_runtime, default_model) with a section heading "Pipeline Configuration"
  - [ ] Repo URL: `FloatLabel` + `InputText`, `id="project-repo-url"`, `:invalid="!!errors.repo_url"`
  - [ ] Git Provider: `FloatLabel` + `Select` (PrimeVue dropdown), `id="project-git-provider"`, `:options="['github']"`, default `github`, required
  - [ ] Agent Runtime: `FloatLabel` + `Select`, `id="project-agent-runtime"`, `:options="['docker']"`, default `docker`, required
  - [ ] Default Model: `FloatLabel` + `InputText`, `id="project-default-model"`, placeholder `claude-opus-4-5`, optional
  - [ ] Add `<small>` error display below each field using `errors.{field}`
  - [ ] Widen dialog to `max-w-2xl` to accommodate the additional fields

- [ ] [FRONT] Task 3: Update `ProjectOverview.vue` — display the new fields (AC: #4)
  - [ ] In `frontend/src/features/projects/ProjectOverview.vue`, add four new `<div>` entries to the existing `<dl>` grid:
    - Repo URL: `<dt>Repository URL</dt><dd>{{ project.repo_url || '-' }}</dd>`
    - Git Provider: `<dt>Git Provider</dt><dd>{{ project.git_provider || '-' }}</dd>`
    - Agent Runtime: `<dt>Agent Runtime</dt><dd>{{ project.agent_runtime || '-' }}</dd>`
    - Default Model: `<dt>Default Model</dt><dd>{{ project.default_model || '-' }}</dd>`
  - [ ] Place these after the existing description field
  - [ ] Repo URL should render as a link if non-empty: `<a :href="project.repo_url" target="_blank" rel="noopener noreferrer">{{ project.repo_url }}</a>`

- [ ] [FRONT] Task 4: Update `ProjectSettingsForm.vue` — add new fields (AC: #5, #6)
  - [ ] In `frontend/src/features/projects/ProjectSettingsForm.vue`, extend the Zod schema to add the four new fields (same schema as CreateProjectDialog):
    ```typescript
    repo_url: z.string().min(1, 'Repository URL is required').url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
    ```
  - [ ] Initialize the new fields from the `project` prop in `initialValues`
  - [ ] Add the `watch(props.project, ...)` reset to include the new fields
  - [ ] Extend the `save` emit payload type to include the four new fields
  - [ ] Add a `<Divider />` + "Pipeline Configuration" heading section in the form template
  - [ ] Add the four new form fields (same pattern as CreateProjectDialog: FloatLabel + InputText/Select)
  - [ ] Remove the "Git, Agent, and Budget settings will be available in a future release." placeholder `Message` that was added in story 1-12 — it is now superseded

- [ ] [FRONT] Task 5: Update `useProjects` composable — pass new fields through (AC: #3, #6, #7)
  - [ ] In `frontend/src/composables/useProjects.ts`, ensure `createProject.execute(...)` and `updateProject.execute(...)` signatures accept the new fields via the updated payload types
  - [ ] No structural change needed if the store types are updated correctly in Task 1

- [ ] [FRONT] Task 6: Unit tests for store type updates (AC: #7)
  - [ ] Update `frontend/src/stores/__tests__/projects.spec.ts`
  - [ ] Add test: `createProject` with `repo_url`, `git_provider`, `agent_runtime`, `default_model` in payload — verify the API client is called with those fields
  - [ ] Add test: `updateProject` with new fields — verify body passed to API includes them
  - [ ] Mock `apiClient` using `vi.mock('@/api/client')`

- [ ] [FRONT] Task 7: Unit tests for CreateProjectDialog (AC: #1, #2, #3)
  - [ ] In `frontend/src/features/projects/__tests__/CreateProjectDialog.spec.ts` (create if it does not exist)
  - [ ] Test: repo URL field is present and labeled correctly
  - [ ] Test: submitting without repo_url shows validation error "Repository URL is required"
  - [ ] Test: submitting with an invalid URL shows "Must be a valid URL"
  - [ ] Test: git_provider Select defaults to `github`
  - [ ] Test: agent_runtime Select defaults to `docker`

- [ ] [FRONT] Task 8: Unit tests for ProjectSettingsForm (AC: #5, #6)
  - [ ] Update `frontend/src/features/projects/__tests__/ProjectSettingsForm.spec.ts`
  - [ ] Test: form pre-fills `repo_url`, `git_provider`, `agent_runtime`, `default_model` from `project` prop
  - [ ] Test: submitting emits `save` with the four new fields included
  - [ ] Test: the old "future tabs" info message is no longer rendered

- [ ] [FRONT] Task 9: E2E test — create project with repo URL (AC: #1, #2, #3)
  - [ ] Update or create `frontend/e2e/tests/projects.spec.ts`
  - [ ] Test: open create dialog, fill all fields including repo URL, submit — verify `POST /projects` body contains `repo_url`
  - [ ] Test: open create dialog, leave repo URL empty, submit — verify inline validation error is shown and no API call is made

## Dev Notes

### Dependencies on other fix stories

| Story | What it provides |
|-------|-----------------|
| fix-10 | Adds `repo_url`, `git_provider`, `agent_runtime`, `default_model` to `Project`, `CreateProjectRequest`, `UpdateProjectRequest` in `api/openapi.yaml` |
| fix-12 | Regenerates `frontend/src/api/generated/schema.d.ts` — TypeScript types will include the new fields |

Until fix-10 and fix-12 are merged, implement against the local interface additions in `stores/projects.ts` (Task 1). Once the generated types are available, remove any manual type duplication and rely on the generated `components['schemas']['Project']`.

### File Paths

| File | Action |
|------|--------|
| `frontend/src/stores/projects.ts` | Update `Project`, `CreateProjectPayload`, `UpdateProjectPayload` interfaces |
| `frontend/src/features/projects/CreateProjectDialog.vue` | Update Zod schema, add 4 new fields to form |
| `frontend/src/features/projects/ProjectOverview.vue` | Add 4 new fields to overview card |
| `frontend/src/features/projects/ProjectSettingsForm.vue` | Update Zod schema, add 4 new fields, remove old placeholder message |
| `frontend/src/composables/useProjects.ts` | No structural change expected — type inference handles it |
| `frontend/src/stores/__tests__/projects.spec.ts` | Add tests for new payload fields |
| `frontend/src/features/projects/__tests__/CreateProjectDialog.spec.ts` | Create |
| `frontend/src/features/projects/__tests__/ProjectSettingsForm.spec.ts` | Update |
| `frontend/e2e/tests/projects.spec.ts` | Add/update E2E tests for create flow |

### Architecture / Implementation Details

**Component Hierarchy (unchanged — only adding fields):**
```
ProjectsView
└── CreateProjectDialog.vue  ← add 4 fields + Zod validation

ProjectDetailView
└── router-view
    ├── ProjectOverview.vue  ← add 4 read-only display fields
    └── (project-settings route)
        └── ProjectSettingsView.vue
            └── ProjectSettingsForm.vue  ← add 4 editable fields
```

**Data Flow:**
```
CreateProjectDialog
  ├── Zod schema (repo_url required, git_provider/agent_runtime enum with defaults)
  ├── vee-validate defineField x6
  └── onSubmit → createProject.execute({ name, description, repo_url, git_provider, agent_runtime, default_model })
                    └── store.createProject → apiClient.POST('/projects', { body: payload })

ProjectSettingsForm
  ├── Zod schema (same additions)
  ├── vee-validate useForm initialValues from project prop
  └── @save emit → ProjectSettingsView → updateProject.execute(id, payload)
                    └── store.updateProject → apiClient.PUT('/projects/{id}', { body: payload })
```

### Zod Schemas

**CreateProjectDialog Zod schema (full):**

```typescript
const createProjectSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or fewer'),
    description: z
      .string()
      .max(1000, 'Description must be 1000 characters or fewer')
      .optional()
      .or(z.literal('')),
    repo_url: z
      .string()
      .min(1, 'Repository URL is required')
      .url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
  }),
)
```

**ProjectSettingsForm Zod schema (full):**

```typescript
const projectSettingsSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or less'),
    description: z
      .string()
      .max(1000, 'Description must be 1000 characters or less')
      .default(''),
    repo_url: z
      .string()
      .min(1, 'Repository URL is required')
      .url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
  }),
)
```

### Store Interface Updates

```typescript
// frontend/src/stores/projects.ts — updated interfaces

export interface Project {
  id: string
  name: string
  description?: string
  repo_url?: string
  git_provider?: string
  agent_runtime?: string
  default_model?: string
  owner_id: string
  circuit_breaker_active?: boolean
  created_at: string
  updated_at: string
}

export interface CreateProjectPayload {
  name: string
  description?: string
  repo_url?: string
  git_provider?: string
  agent_runtime?: string
  default_model?: string
}

export interface UpdateProjectPayload {
  name?: string
  description?: string
  repo_url?: string
  git_provider?: string
  agent_runtime?: string
  default_model?: string
}
```

### CreateProjectDialog Form Template (new fields section)

```vue
<!-- After the existing description field, before the footer -->

<div class="flex flex-col gap-1">
  <p class="text-sm font-semibold text-surface-600">Pipeline Configuration</p>
</div>

<div class="flex flex-col gap-2">
  <FloatLabel>
    <InputText
      id="project-repo-url"
      v-model="repoUrl"
      v-bind="repoUrlAttrs"
      class="w-full"
      :invalid="!!errors.repo_url"
    />
    <label for="project-repo-url">Repository URL *</label>
  </FloatLabel>
  <small v-if="errors.repo_url" class="text-red-500">{{ errors.repo_url }}</small>
</div>

<div class="flex flex-col gap-2">
  <FloatLabel>
    <Select
      id="project-git-provider"
      v-model="gitProvider"
      v-bind="gitProviderAttrs"
      :options="['github']"
      class="w-full"
      :invalid="!!errors.git_provider"
    />
    <label for="project-git-provider">Git Provider *</label>
  </FloatLabel>
  <small v-if="errors.git_provider" class="text-red-500">{{ errors.git_provider }}</small>
</div>

<div class="flex flex-col gap-2">
  <FloatLabel>
    <Select
      id="project-agent-runtime"
      v-model="agentRuntime"
      v-bind="agentRuntimeAttrs"
      :options="['docker']"
      class="w-full"
      :invalid="!!errors.agent_runtime"
    />
    <label for="project-agent-runtime">Agent Runtime *</label>
  </FloatLabel>
  <small v-if="errors.agent_runtime" class="text-red-500">{{ errors.agent_runtime }}</small>
</div>

<div class="flex flex-col gap-2">
  <FloatLabel>
    <InputText
      id="project-default-model"
      v-model="defaultModel"
      v-bind="defaultModelAttrs"
      class="w-full"
      placeholder="claude-opus-4-5"
    />
    <label for="project-default-model">Default Model</label>
  </FloatLabel>
  <small v-if="errors.default_model" class="text-red-500">{{ errors.default_model }}</small>
</div>
```

### ProjectOverview Template (new fields section)

```vue
<!-- Add inside the existing <dl> grid, after description -->

<div v-if="project.repo_url" class="sm:col-span-2">
  <dt class="text-sm font-medium text-surface-500">Repository URL</dt>
  <dd class="mt-1">
    <a
      :href="project.repo_url"
      target="_blank"
      rel="noopener noreferrer"
      class="underline"
    >{{ project.repo_url }}</a>
  </dd>
</div>

<div>
  <dt class="text-sm font-medium text-surface-500">Git Provider</dt>
  <dd class="mt-1">{{ project.git_provider || '-' }}</dd>
</div>

<div>
  <dt class="text-sm font-medium text-surface-500">Agent Runtime</dt>
  <dd class="mt-1">{{ project.agent_runtime || '-' }}</dd>
</div>

<div>
  <dt class="text-sm font-medium text-surface-500">Default Model</dt>
  <dd class="mt-1">{{ project.default_model || '-' }}</dd>
</div>
```

### PrimeVue Components Used

| Component | Import | Usage |
|-----------|--------|-------|
| `InputText` | `primevue/inputtext` | Repo URL, Default Model text fields |
| `Select` | `primevue/select` | Git Provider, Agent Runtime dropdowns |
| `FloatLabel` | `primevue/floatlabel` | Floating label wrappers for all new fields |
| `Button` | `primevue/button` | Existing (no change) |
| `Message` | `primevue/message` | Existing error display (no change) |
| `Dialog` | `primevue/dialog` | Existing dialog container (widen to `max-w-2xl`) |

Note: use `Select` (not `Dropdown`) — PrimeVue 4 renamed `Dropdown` to `Select`.

### API Endpoints Affected

| Method | Path | Change |
|--------|------|--------|
| `POST` | `/api/v1/projects` | Body now optionally includes `repo_url`, `git_provider`, `agent_runtime`, `default_model` |
| `PUT` | `/api/v1/projects/{id}` | Body now optionally includes `repo_url`, `git_provider`, `agent_runtime`, `default_model` |
| `GET` | `/api/v1/projects/{id}` | Response now includes the four new fields (from fix-10 spec change) |

### Testing Requirements

**Unit tests (Vitest):**

| Test file | What to test | Coverage target |
|-----------|-------------|----------------|
| `stores/__tests__/projects.spec.ts` | `createProject` and `updateProject` pass new fields to apiClient | 90%+ |
| `features/projects/__tests__/CreateProjectDialog.spec.ts` | repo_url required validation, URL format validation, Select defaults | As needed |
| `features/projects/__tests__/ProjectSettingsForm.spec.ts` | Pre-fill from project prop, save emits new fields, old placeholder gone | As needed |

**E2E tests (Playwright):**
- Create dialog: fill all fields including repo URL → POST body contains `repo_url`
- Create dialog: leave repo URL empty → validation error visible, no API call
- Settings form: loaded project with repo URL → field pre-filled, edit and save → PUT body contains updated `repo_url`

**Manual verification checklist:**
1. Open Create Project dialog — confirm four new fields appear under "Pipeline Configuration"
2. Submit without repo URL — "Repository URL is required" appears, form does not submit
3. Enter invalid URL (e.g., `not-a-url`) — "Must be a valid URL" appears
4. Enter valid URL, complete form, create — project created successfully, visible in list
5. Navigate to project overview tab — repo URL appears as clickable link, git_provider and agent_runtime show `github` / `docker`
6. Navigate to project settings — all six fields pre-filled including the four new ones
7. Edit repo URL in settings, save — success toast, fields reflect new values on reload
8. `npm run build` — no TypeScript errors
9. `npm run lint` — no lint errors
10. `npm run test:unit` — all new and existing tests pass

### Style / Convention Notes

- `Select` options are plain string arrays `['github']` / `['docker']` — no object mapping needed for single-value enums at MVP
- Both `git_provider` and `agent_runtime` are intentionally single-option dropdowns at MVP — they communicate future extensibility to the user while keeping the implementation trivial
- `default_model` is intentionally free-text (not an enum) because the model catalogue changes frequently and is not controlled by this app
- No `<style scoped>` blocks — layout via Tailwind utilities only
- The `repo_url` link in `ProjectOverview` uses `rel="noopener noreferrer"` for security on external links

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | story-writer | Initial story created |
