# Story 2.7: [FRONT] Story Editor (Create + Edit Mode)

Status: ready-for-dev

## Story

As a project user, I want to create and edit stories inline, So that I can manage story content without leaving the board.

## Acceptance Criteria (BDD)

**AC1: Edit mode activates from StoryDetailPanel**
- **Given** a story is displayed in StoryDetailPanel with status "backlog" or "failed"
- **When** I click the "Edit" button in the panel header
- **Then** the panel switches to edit mode
- **And** the title field becomes an InputText
- **And** the objective field becomes a Textarea
- **And** the acceptance_criteria field becomes a Textarea
- **And** the target_files field becomes an editable list (one input per file, with Add/Remove controls)
- **And** the scope field becomes a Select dropdown (backend, frontend, shared)

**AC2: Save with valid data persists via PUT**
- **Given** I am in edit mode and have modified story fields
- **When** I click "Save"
- **Then** PUT `/api/v1/projects/{projectId}/stories/{storyId}` is called with updated fields
- **And** on success, edit mode exits
- **And** the story data in the store is updated with the response
- **And** a success Toast is shown ("Story updated")

**AC3: Inline validation on save**
- **Given** I am in edit mode
- **When** I clear the title field and click "Save"
- **Then** an inline error message appears below the title input ("Title is required")
- **And** the form is NOT submitted
- **And** edit mode stays active

**AC4: Cancel discards changes**
- **Given** I am in edit mode and have modified fields
- **When** I click "Cancel"
- **Then** edit mode exits
- **And** the story reverts to its original values (no API call made)

**AC5: Create Story opens empty editor form**
- **Given** I am viewing the epic detail page
- **When** I click the "Create Story" button in the StoryListPanel header
- **Then** a Dialog opens with an empty story form
- **And** the form has fields: key (required), title (required), objective, acceptance_criteria, target_files (editable list), scope (dropdown), epic_id (pre-filled with current epic, hidden)
- **And** on submit, POST `/api/v1/projects/{projectId}/stories` is called
- **And** on success, the dialog closes, the story list refreshes, and a success Toast is shown ("Story created")

**AC6: Create Story validation**
- **Given** the Create Story dialog is open
- **When** I submit with key or title empty
- **Then** inline errors appear ("Key is required", "Title is required")
- **And** the form is NOT submitted

**AC7: API error feedback**
- **Given** I save (edit or create) and the API returns an error (e.g., 409 key conflict)
- **When** the error response arrives
- **Then** a PrimeVue Message with severity "error" is shown inside the form/dialog
- **And** the form stays open so the user can correct the issue

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `updateStory` and `createStory` actions to stories store (AC: #2, #5)
  - [ ] Add `UpdateStoryFields` interface: `{ title?, objective?, acceptance_criteria?, target_files?, depends_on?, scope?, status? }`
  - [ ] Add `CreateStoryFields` interface: `{ key, title, objective?, acceptance_criteria?, target_files?, depends_on?, scope?, epic_id? }`
  - [ ] `updateStory(projectId, storyId, fields)` — calls `apiClient.PUT`, updates item in `items.value` in-place on success
  - [ ] `createStory(projectId, fields)` — calls `apiClient.POST`, pushes new story to `items.value`
  - [ ] Both return `Story | null`, set `error.value` on failure

- [ ] [FRONT] Task 2: Create `useStoryEditor` composable (AC: #1, #2, #3, #4)
  - [ ] File: `frontend/src/composables/useStoryEditor.ts`
  - [ ] Accept `projectId: string`, `story: Ref<Story | null>`
  - [ ] Expose: `isEditing`, `draftFields`, `validationErrors`, `apiError`, `isSaving`
  - [ ] `startEdit()` — copies story fields into draftFields (deep clone), sets `isEditing = true`
  - [ ] `cancelEdit()` — resets draftFields, sets `isEditing = false`
  - [ ] `saveEdit(storyId)` — validates (title required), calls `store.updateStory`, exits edit mode on success
  - [ ] Unit test: `frontend/src/composables/__tests__/useStoryEditor.spec.ts`

- [ ] [FRONT] Task 3: Create `StoryEditorForm.vue` component (AC: #1, #3)
  - [ ] File: `frontend/src/features/board/StoryEditorForm.vue`
  - [ ] Props: `modelValue: UpdateStoryFields`, `errors: Record<string, string>`, `apiError: string | null`, `isSaving: boolean`
  - [ ] Emits: `update:modelValue`, `save`, `cancel`
  - [ ] Fields: Title (FloatLabel+InputText), Objective (Textarea), Acceptance Criteria (Textarea), Scope (Select), Target Files (editable list with Add/Remove)
  - [ ] Inline error `<small class="text-red-500">` below each invalid field
  - [ ] API error `<Message severity="error">` at top of form
  - [ ] Footer: Cancel (secondary, text) + Save (primary, `:loading="isSaving"`)

- [ ] [FRONT] Task 4: Enhance `StoryDetailPanel.vue` with edit mode toggle (AC: #1, #2, #4)
  - [ ] Import and use `useStoryEditor` composable
  - [ ] Add "Edit" Button (secondary, text, icon `pi-pencil`) in header — visible only if story status is `backlog` or `failed`
  - [ ] When `isEditing`: replace body with `StoryEditorForm`, wire draftFields/errors/apiError/isSaving
  - [ ] On `@save`: call `saveEdit(story.id)`, emit `story-updated` on success
  - [ ] On `@cancel`: call `cancelEdit()`
  - [ ] Update `EpicDetailLayout.vue` to pass `projectId` and wire `@story-updated`

- [ ] [FRONT] Task 5: Create `CreateStoryDialog.vue` component (AC: #5, #6, #7)
  - [ ] File: `frontend/src/features/board/CreateStoryDialog.vue`
  - [ ] Props: `visible`, `projectId`, `epicId`
  - [ ] Emits: `update:visible`, `created: [story: Story]`
  - [ ] Use `useForm` + `toTypedSchema(z.object({ key: z.string().min(1), title: z.string().min(1), objective: z.string().optional(), acceptance_criteria: z.string().optional(), scope: z.enum([...]).optional() }))`
  - [ ] On valid submit: call `store.createStory(projectId, { ...values, epic_id: epicId })`
  - [ ] Success: emit `created`, close dialog, Toast "Story created"
  - [ ] API error: `<Message severity="error">` inside dialog
  - [ ] Reset form on dialog close

- [ ] [FRONT] Task 6: Add "Create Story" button to StoryListPanel + wire dialog in EpicDetailView (AC: #5)
  - [ ] In `StoryListPanel.vue`: add `projectId`, `epicId` props + "Create Story" Button in header (icon `pi-plus`, text, secondary)
  - [ ] Emit `create-story` on click
  - [ ] In `EpicDetailLayout.vue`: wire `@create-story` emit up
  - [ ] In `EpicDetailView.vue`: import `CreateStoryDialog`, manage `createDialogVisible` ref
  - [ ] On `@create-story`: open dialog; on `@created`: show Toast, stories store already updated

- [ ] [FRONT] Task 7: Unit tests for store actions and StoryEditorForm (AC: #2, #3)
  - [ ] Extend `frontend/src/stores/__tests__/stories.spec.ts`:
    - `updateStory` updates item in-place on success, returns null on error
    - `createStory` pushes to items on success, returns null on error
  - [ ] Create `frontend/src/features/board/__tests__/StoryEditorForm.spec.ts`:
    - All fields render with initial values
    - Inline error appears when `errors.title` set
    - API error Message renders when `apiError` set
    - Save button loading when `isSaving=true`
    - Save/Cancel emit correct events

- [ ] [FRONT] Task 8: Unit tests for useStoryEditor composable (AC: #1, #2, #3, #4)
  - [ ] File: `frontend/src/composables/__tests__/useStoryEditor.spec.ts`
  - [ ] `startEdit()` copies story fields, sets `isEditing=true`
  - [ ] `cancelEdit()` resets state, sets `isEditing=false`
  - [ ] `saveEdit()` with valid fields: calls `store.updateStory`, sets `isEditing=false`
  - [ ] `saveEdit()` with empty title: sets `validationErrors.title`, does NOT call store
  - [ ] `saveEdit()` on API error: populates `apiError`, `isEditing` stays true

## Dev Notes

### Dependencies

- **Story 2-6 (DONE after wave 7):** Adds `scope` to Story interface, `allStories`/`projectId` props to EpicDetailLayout, RunLaunchButton in StoryDetailPanel
- **Story 2-2 (DONE):** Backend PUT/POST endpoints exist at `/projects/{projectId}/stories/{storyId}` and `/projects/{projectId}/stories`
- **Story 1-16 (DONE):** `apiClient` and routing infrastructure

### Architecture Requirements

Component hierarchy additions:

```
EpicDetailView.vue
├── CreateStoryDialog.vue  (new)
└── EpicDetailLayout.vue  (enhanced: story-updated, create-story)
    ├── StoryListPanel.vue  (enhanced: create-story emit, projectId+epicId props)
    └── StoryDetailPanel.vue  (enhanced: edit mode, StoryEditorForm)
        ├── [read mode] existing read-only content + Edit button
        └── [edit mode] StoryEditorForm.vue  (new)
```

**Composable boundary:**
- `useStoryEditor` owns all edit-mode state: `isEditing`, `draftFields`, `validationErrors`, `apiError`, `isSaving`
- `CreateStoryDialog` uses `vee-validate` directly (standard form dialog pattern, matching `CreateProjectDialog`)

**Store update strategy:**
- `updateStory`: find story in `items.value` by id, replace in-place — reactive update propagates automatically
- `createStory`: push to `items.value` — list updates reactively

**Edit button visibility rule:** Show only when `story.status` is `'backlog'` or `'failed'`

### File Paths (exact)

```
frontend/src/composables/useStoryEditor.ts                              (new)
frontend/src/composables/__tests__/useStoryEditor.spec.ts               (new)
frontend/src/stores/stories.ts                                          (extend: updateStory, createStory actions)
frontend/src/stores/__tests__/stories.spec.ts                           (extend: test new actions)
frontend/src/features/board/StoryEditorForm.vue                         (new)
frontend/src/features/board/__tests__/StoryEditorForm.spec.ts           (new)
frontend/src/features/board/StoryDetailPanel.vue                        (enhance: edit mode + useStoryEditor)
frontend/src/features/board/CreateStoryDialog.vue                       (new)
frontend/src/features/board/StoryListPanel.vue                          (enhance: create-story emit)
frontend/src/features/board/EpicDetailLayout.vue                        (enhance: wire create-story + story-updated)
frontend/src/views/EpicDetailView.vue                                   (enhance: CreateStoryDialog + handlers)
```

### Technical Specifications

**`useStoryEditor` composable:**
```typescript
export function useStoryEditor(projectId: string, story: Ref<Story | null>) {
  const store = useStoriesStore()
  const isEditing = ref(false)
  const draftFields = ref<UpdateStoryFields>({})
  const validationErrors = ref<Record<string, string>>({})
  const apiError = ref<string | null>(null)
  const isSaving = ref(false)

  function startEdit() {
    if (!story.value) return
    draftFields.value = {
      title: story.value.title,
      objective: story.value.objective,
      acceptance_criteria: story.value.acceptance_criteria,
      target_files: [...(story.value.target_files ?? [])],
      depends_on: [...(story.value.depends_on ?? [])],
      scope: story.value.scope,
    }
    validationErrors.value = {}
    apiError.value = null
    isEditing.value = true
  }

  function cancelEdit() {
    isEditing.value = false
    draftFields.value = {}
    validationErrors.value = {}
    apiError.value = null
  }

  async function saveEdit(storyId: string): Promise<Story | null> {
    validationErrors.value = {}
    if (!draftFields.value.title?.trim()) {
      validationErrors.value.title = 'Title is required'
      return null
    }
    isSaving.value = true
    apiError.value = null
    try {
      const updated = await store.updateStory(projectId, storyId, draftFields.value)
      if (updated) {
        isEditing.value = false
        return updated
      }
      apiError.value = store.error ?? 'Failed to save story'
      return null
    } finally {
      isSaving.value = false
    }
  }

  return { isEditing, draftFields, validationErrors, apiError, isSaving, startEdit, cancelEdit, saveEdit }
}
```

**Target file editable list pattern:**
```typescript
function addFile() {
  emit('update:modelValue', { ...props.modelValue, target_files: [...(props.modelValue.target_files ?? []), ''] })
}
function removeFile(index: number) {
  const updated = [...(props.modelValue.target_files ?? [])]
  updated.splice(index, 1)
  emit('update:modelValue', { ...props.modelValue, target_files: updated })
}
function updateFile(index: number, value: string) {
  const updated = [...(props.modelValue.target_files ?? [])]
  updated[index] = value
  emit('update:modelValue', { ...props.modelValue, target_files: updated })
}
```

**CreateStoryDialog zod schema:**
```typescript
const createStorySchema = toTypedSchema(
  z.object({
    key: z.string().min(1, 'Key is required').max(50),
    title: z.string().min(1, 'Title is required').max(255),
    objective: z.string().optional().or(z.literal('')),
    acceptance_criteria: z.string().optional().or(z.literal('')),
    scope: z.enum(['backend', 'frontend', 'shared']).optional(),
  })
)
```

### Testing Requirements

See Tasks 7 and 8 for detailed test cases.

### References

- Existing form pattern: `frontend/src/features/projects/CreateProjectDialog.vue` (vee-validate + zod)
- OpenAPI spec: `api/openapi.yaml` — `UpdateStoryRequest`, `CreateStoryRequest`, `Story` schemas
- PrimeVue Dialog, Select, FloatLabel, Message components

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Opus 4.6 | Initial story creation |
