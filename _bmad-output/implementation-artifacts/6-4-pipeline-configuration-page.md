# Story 6.4: [FRONT] Pipeline Configuration Page

Status: ready-for-dev

## Story

As an admin, I want a visual pipeline configuration page, So that I can customize pipeline steps without editing raw YAML.

## Acceptance Criteria (BDD)

**AC1: Display pipeline steps as ordered list**
- **Given** I am on `/projects/:id/pipeline` as an admin
- **When** the page loads
- **Then** I see the current pipeline steps as an ordered list showing step name, action type, model, and auto_approve toggle

**AC2: Expandable step details**
- **Given** I am viewing the pipeline configuration page
- **When** I click on a pipeline step
- **Then** the step expands to show model selector, auto_approve checkbox, and retry policy (max_retries, retry_type)

**AC3: Reorder steps (admin only)**
- **Given** I am viewing the pipeline configuration page as an admin
- **When** I use move up/down buttons on a step
- **Then** the step moves in the list and the order is updated locally

**AC4: Add new step (admin only)**
- **Given** I am viewing the pipeline configuration page as an admin
- **When** I click "Add Step"
- **Then** a dialog opens with fields: name, action_type, model, auto_approve, retry policy
- **When** I fill the form and click "Add"
- **Then** the new step is added to the list locally

**AC5: Remove step (admin only)**
- **Given** I am viewing the pipeline configuration page as an admin
- **When** I click the remove button on a step
- **Then** the step is removed from the list locally

**AC6: Save configuration**
- **Given** I have made changes to the pipeline configuration
- **When** I click "Save"
- **Then** PUT /api/v1/projects/{projectId}/pipeline is called with the updated config
- **And** on success, I see a success toast message
- **And** the save button is disabled until further changes are made

**AC7: Read-only mode (non-admin)**
- **Given** I am on `/projects/:id/pipeline` as a non-admin user
- **When** the page loads
- **Then** I see the pipeline steps in read-only mode with no edit controls (no add/remove/reorder buttons, no save button)

**AC8: Loading and error states**
- **Given** I am on `/projects/:id/pipeline`
- **When** the API request is pending
- **Then** I see a loading skeleton
- **When** the API request fails
- **Then** I see an error message with a retry button

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create pipelineConfig Pinia store (AC: #1, #6, #8)
  - [ ] State: config (PipelineConfig type), loading, error, isDirty
  - [ ] Actions: fetchConfig, updateConfig, saveConfig, resetDirty
  - [ ] Use apiClient GET /api/v1/projects/{projectId}/pipeline and PUT
  - [ ] Track isDirty flag for unsaved changes

- [ ] [FRONT] Task 2: Create usePipelineConfig composable (AC: #1, #6, #8)
  - [ ] Wrap useAsyncAction for fetch and save
  - [ ] Export: config, loading, error, isDirty, fetchConfig, saveConfig, retry
  - [ ] Auto-fetch on mount with projectId param

- [ ] [FRONT] Task 3: Build PipelineStepCard.vue component (AC: #1, #2)
  - [ ] Props: step (PipelineStep), index, isAdmin, expanded
  - [ ] Emits: toggle, update, remove, moveUp, moveDown
  - [ ] Collapsed view: show name, action_type, model, auto_approve badge
  - [ ] Expanded view: show model selector (PrimeVue Dropdown), auto_approve checkbox, retry policy fields
  - [ ] Admin mode: show move up/down buttons, remove button
  - [ ] Read-only mode: no interactive controls
  - [ ] Use PrimeVue Card + PrimeVue Dropdown + PrimeVue Checkbox + PrimeVue InputNumber

- [ ] [FRONT] Task 4: Build PipelineStepList.vue component (AC: #1, #3, #5)
  - [ ] Props: steps (PipelineStep[]), isAdmin
  - [ ] Emits: update, remove, reorder
  - [ ] Render PipelineStepCard for each step
  - [ ] Handle moveUp/moveDown events: swap steps in array, emit reorder
  - [ ] Handle remove event: filter step, emit remove
  - [ ] Expandable state: track expanded step index locally

- [ ] [FRONT] Task 5: Build AddStepDialog.vue component (AC: #4)
  - [ ] Props: visible
  - [ ] Emits: add, cancel, update:visible
  - [ ] PrimeVue Dialog with form fields: name, action_type (dropdown), model (dropdown), auto_approve (checkbox), max_retries (InputNumber), retry_type (dropdown)
  - [ ] Validate required fields
  - [ ] Emit add event with new step object
  - [ ] Reset form on close

- [ ] [FRONT] Task 6: Build PipelineConfigView.vue route view (AC: #1, #6, #7, #8)
  - [ ] Use usePipelineConfig composable
  - [ ] Use useAuth to check isAdmin
  - [ ] Loading state: PrimeVue Skeleton
  - [ ] Error state: PrimeVue Message with retry button
  - [ ] Success: render PipelineStepList
  - [ ] Admin mode: show "Add Step" button, "Save" button (disabled when !isDirty)
  - [ ] Non-admin mode: no edit controls
  - [ ] Handle save: call saveConfig, show toast on success/error
  - [ ] Handle add step: open AddStepDialog, push new step to local config

- [ ] [FRONT] Task 7: Register route in router/index.ts (AC: #1)
  - [ ] Path: /projects/:id/pipeline
  - [ ] Name: project-pipeline
  - [ ] Component: PipelineConfigView
  - [ ] Meta: requiresAuth: true

- [ ] [FRONT] Task 8: Write unit tests for store and composable (AC: #1, #6, #8)
  - [ ] pipelineConfig.spec.ts: test fetchConfig, saveConfig, isDirty tracking
  - [ ] usePipelineConfig.spec.ts: test auto-fetch, save, retry

- [ ] [FRONT] Task 9: Write E2E test for pipeline config page (AC: #1, #3, #4, #5, #6, #7)
  - [ ] pipeline-config.spec.ts: test view as admin (add/remove/reorder/save), view as non-admin (read-only)

## Dev Notes

### Dependencies

- Story 1-8: App shell (AppLayout)
- Story 1-9: Login/auth guard (requiresAuth meta)
- Story 1-16: Routing, stores, apiClient setup
- Backend peer: Story 6-1 (pipeline configs API) — consumes GET/PUT /api/v1/projects/{projectId}/pipeline

### Architecture Requirements

Component hierarchy:
```
PipelineConfigView.vue (route view)
├── PrimeVue Skeleton (loading state)
├── PrimeVue Message (error state with retry button)
└── PipelineStepList.vue
    └── PipelineStepCard.vue (repeated, expandable)
        ├── Collapsed view:
        │   ├── name, action_type, model
        │   ├── auto_approve badge
        │   └── move up/down buttons (admin only)
        └── Expanded view:
            ├── PrimeVue Dropdown (model selector)
            ├── PrimeVue Checkbox (auto_approve)
            ├── PrimeVue InputNumber (max_retries)
            └── PrimeVue Dropdown (retry_type)
└── AddStepDialog.vue (admin only)
    └── PrimeVue Dialog with form fields
```

State management:
```
stores/pipelineConfig.ts → composables/usePipelineConfig.ts → PipelineConfigView.vue
```

### File Paths (exact)

```
frontend/src/stores/pipelineConfig.ts
frontend/src/composables/usePipelineConfig.ts
frontend/src/features/pipeline/PipelineStepCard.vue
frontend/src/features/pipeline/PipelineStepList.vue
frontend/src/features/pipeline/AddStepDialog.vue
frontend/src/views/PipelineConfigView.vue
frontend/src/router/index.ts (append route)
frontend/src/__tests__/stores/pipelineConfig.spec.ts
frontend/src/__tests__/composables/usePipelineConfig.spec.ts
frontend/e2e/tests/pipeline-config.spec.ts
```

### Technical Specifications

**PipelineStepCard.vue props/emits:**
```typescript
interface Props {
  step: PipelineStep // from generated API types
  index: number
  isAdmin: boolean
  expanded: boolean
}
const emit = defineEmits<{
  toggle: []
  update: [step: PipelineStep]
  remove: []
  moveUp: []
  moveDown: []
}>()
```

**PipelineStepList.vue props/emits:**
```typescript
interface Props {
  steps: PipelineStep[]
  isAdmin: boolean
}
const emit = defineEmits<{
  update: [steps: PipelineStep[]]
  remove: [index: number]
  reorder: [fromIndex: number, toIndex: number]
}>()
```

**AddStepDialog.vue props/emits:**
```typescript
interface Props {
  visible: boolean
}
const emit = defineEmits<{
  add: [step: Omit<PipelineStep, 'id'>]
  cancel: []
  'update:visible': [value: boolean]
}>()
```

**Model options (dropdown):**
```typescript
const modelOptions = [
  { label: 'Claude Opus 4.6', value: 'claude-opus-4-6' },
  { label: 'Claude Sonnet 4.5', value: 'claude-sonnet-4-5' },
  { label: 'Claude Haiku 4.3', value: 'claude-haiku-4-3' }
]
```

**Action type options (dropdown):**
```typescript
const actionTypeOptions = [
  { label: 'Implement', value: 'implement' },
  { label: 'Review', value: 'review' },
  { label: 'Merge', value: 'merge' },
  { label: 'Test', value: 'test' },
  { label: 'Custom', value: 'custom' }
]
```

**Retry type options (dropdown):**
```typescript
const retryTypeOptions = [
  { label: 'None', value: 'none' },
  { label: 'On Failure', value: 'on-failure' },
  { label: 'Always', value: 'always' }
]
```

**isDirty tracking:**
```typescript
// In store: set isDirty = true when config is modified locally
// Reset isDirty after successful save
```

### Testing Requirements

**Unit tests:**
- pipelineConfig store: fetchConfig success/error, saveConfig success/error, isDirty tracking
- usePipelineConfig composable: auto-fetch on mount, save with toast feedback

**E2E tests:**
- Load pipeline config page as admin → see step list, add/remove/reorder buttons
- Add new step → see step in list
- Reorder step → see order change
- Remove step → see step removed
- Save changes → see success toast, isDirty resets
- Load as non-admin → no edit controls visible

### References

- Epic 6: Pipeline Configuration & Prompt Templates
- Backend Story 6-1: Pipeline configs API (GET/PUT /api/v1/projects/{projectId}/pipeline)
- PrimeVue Card: https://primevue.org/card/
- PrimeVue Dropdown: https://primevue.org/dropdown/
- PrimeVue Checkbox: https://primevue.org/checkbox/
- PrimeVue InputNumber: https://primevue.org/inputnumber/
- PrimeVue Dialog: https://primevue.org/dialog/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
