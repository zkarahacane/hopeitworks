# Story 3.11: [FRONT] Run Launch Button & Confirmation Dialog

Status: ready-for-dev

## Story

As a user, I want to launch a story run from the UI, So that I can trigger the AI pipeline for a story.

## Acceptance Criteria (BDD)

**AC1: Show launch button for backlog stories**
- **Given** I am viewing a story detail page
- **When** the story status is 'backlog'
- **Then** I see a "Launch Run" primary button

**AC2: Hide launch button for non-backlog stories**
- **Given** I am viewing a story detail page
- **When** the story status is 'running', 'done', or 'failed'
- **Then** I do not see a "Launch Run" button
- **And** if status is 'running', I see a disabled button with text "Running..." and tooltip "A run is already in progress"

**AC3: Confirmation dialog on launch**
- **Given** I am viewing a story with status 'backlog'
- **When** I click "Launch Run"
- **Then** I see a PrimeVue ConfirmDialog with story key, title, and resource usage warning
- **And** the dialog has "Cancel" and "Confirm" buttons

**AC4: Successful run launch**
- **Given** the confirmation dialog is open
- **When** I click "Confirm"
- **Then** POST /api/v1/projects/{id}/stories/{storyId}/runs is called
- **And** on success, I see a success toast message
- **And** the button changes to disabled state with text "Running..."
- **And** the dialog closes

**AC5: Error handling for already running story**
- **Given** the confirmation dialog is open
- **When** I click "Confirm" and the API returns 409 (story already running)
- **Then** I see an error toast with message "This story already has a run in progress"
- **And** the dialog stays open
- **And** the button remains enabled

**AC6: Error handling for other failures**
- **Given** the confirmation dialog is open
- **When** I click "Confirm" and the API returns an error (non-409)
- **Then** I see an error toast with the error message
- **And** the dialog closes
- **And** the button remains enabled

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create useRunLauncher composable (AC: #4, #5, #6)
  - [ ] Use useAsyncAction pattern
  - [ ] Action: launchRun(projectId, storyId)
  - [ ] POST /api/v1/projects/{id}/stories/{storyId}/runs via apiClient
  - [ ] Handle 409 Conflict specifically (already running)
  - [ ] Return loading, error, execute, reset

- [ ] [FRONT] Task 2: Build RunLaunchButton.vue component (AC: #1, #2, #3)
  - [ ] Props: storyId, storyKey, storyTitle, status
  - [ ] Emits: launchClick
  - [ ] Conditional rendering based on status
  - [ ] Backlog: show primary button "Launch Run"
  - [ ] Running: show disabled button "Running..." with PrimeVue Tooltip
  - [ ] Done/Failed: no button rendered
  - [ ] Click handler: emit launchClick

- [ ] [FRONT] Task 3: Build RunLaunchConfirmDialog.vue component (AC: #3, #4, #5, #6)
  - [ ] Props: visible, storyKey, storyTitle, loading
  - [ ] Emits: confirm, cancel, update:visible
  - [ ] PrimeVue ConfirmDialog with header "Launch Story Run"
  - [ ] Content: story key, title, resource usage warning text
  - [ ] Footer: Cancel + Confirm buttons
  - [ ] Confirm button disabled when loading
  - [ ] Handle confirm click: emit confirm event
  - [ ] Handle cancel/close: emit cancel + update:visible(false)

- [ ] [FRONT] Task 4: Integrate into story detail view (placeholder) (AC: #1, #2, #3, #4, #5, #6)
  - [ ] Import RunLaunchButton and RunLaunchConfirmDialog
  - [ ] Pass story props to button
  - [ ] Handle launchClick: open dialog (set dialogVisible = true)
  - [ ] Handle dialog confirm: call useRunLauncher.execute
  - [ ] On success: show success toast, close dialog
  - [ ] On 409 error: show specific error toast, keep dialog open
  - [ ] On other error: show error toast, close dialog

- [ ] [FRONT] Task 5: Add PrimeVue ToastService integration (AC: #4, #5, #6)
  - [ ] Ensure PrimeVue Toast is registered in main.ts
  - [ ] Use useToast composable in story detail view
  - [ ] Success: toast.add({ severity: 'success', summary: 'Run launched', detail: '...', life: 3000 })
  - [ ] Error 409: toast.add({ severity: 'warn', summary: 'Already running', detail: '...', life: 5000 })
  - [ ] Error other: toast.add({ severity: 'error', summary: 'Launch failed', detail: error message, life: 5000 })

- [ ] [FRONT] Task 6: Write unit tests for useRunLauncher (AC: #4, #5, #6)
  - [ ] Test successful launch
  - [ ] Test 409 error handling
  - [ ] Test generic error handling
  - [ ] Test loading state

- [ ] [FRONT] Task 7: Write unit tests for components (AC: #1, #2, #3)
  - [ ] RunLaunchButton.spec.ts: test conditional rendering, disabled states, emit
  - [ ] RunLaunchConfirmDialog.spec.ts: test open/close, confirm/cancel emit, loading state

- [ ] [FRONT] Task 8: Write E2E test for run launch flow (AC: #1, #3, #4, #5)
  - [ ] run-launch.spec.ts: navigate to story detail → click launch → confirm → verify toast + button state
  - [ ] Test 409 conflict handling

## Dev Notes

### Dependencies

- Story 1-8: App shell (AppLayout, Toast component)
- Story 1-9: Login/auth guard
- Story 1-16: apiClient setup
- Backend peer: Story 3-10 (run launch API, Wave 7) — for now use mock/placeholder endpoint or implement with real API when available

### Architecture Requirements

Component hierarchy:
```
StoryDetailView.vue (placeholder/future story)
├── RunLaunchButton.vue
│   └── PrimeVue Button + Tooltip (if running)
└── RunLaunchConfirmDialog.vue
    └── PrimeVue ConfirmDialog
        ├── Header: "Launch Story Run"
        ├── Content: story key, title, warning
        └── Footer: Cancel + Confirm buttons
```

Composable usage:
```
useRunLauncher.ts → StoryDetailView.vue
```

### File Paths (exact)

```
frontend/src/composables/useRunLauncher.ts
frontend/src/features/runs/RunLaunchButton.vue
frontend/src/features/runs/RunLaunchConfirmDialog.vue
frontend/src/views/StoryDetailView.vue (integrate, placeholder if not exists)
frontend/src/__tests__/composables/useRunLauncher.spec.ts
frontend/src/__tests__/features/runs/RunLaunchButton.spec.ts
frontend/src/__tests__/features/runs/RunLaunchConfirmDialog.spec.ts
frontend/e2e/tests/run-launch.spec.ts
```

### Technical Specifications

**RunLaunchButton.vue props/emits:**
```typescript
interface Props {
  storyId: string
  storyKey: string
  storyTitle: string
  status: 'backlog' | 'running' | 'done' | 'failed'
}
const emit = defineEmits<{
  launchClick: []
}>()
```

**RunLaunchConfirmDialog.vue props/emits:**
```typescript
interface Props {
  visible: boolean
  storyKey: string
  storyTitle: string
  loading: boolean
}
const emit = defineEmits<{
  confirm: []
  cancel: []
  'update:visible': [value: boolean]
}>()
```

**useRunLauncher composable signature:**
```typescript
export function useRunLauncher() {
  const { loading, error, execute, reset } = useAsyncAction(
    async (projectId: string, storyId: string) => {
      const response = await apiClient.POST('/api/v1/projects/{id}/stories/{storyId}/runs', {
        params: { path: { id: projectId, storyId } }
      })
      if (response.error) {
        if (response.response.status === 409) {
          throw new Error('ALREADY_RUNNING')
        }
        throw new Error(response.error.message)
      }
      return response.data
    }
  )

  return { loading, error, launchRun: execute, reset }
}
```

**Resource usage warning text:**
```
"Launching this run will start an AI agent container. The run will consume Claude API credits and Docker resources. Do you want to proceed?"
```

### Testing Requirements

**Unit tests:**
- useRunLauncher: test success, 409 error, generic error, loading states
- RunLaunchButton: test conditional rendering (backlog/running/done/failed), emit launchClick
- RunLaunchConfirmDialog: test open/close, confirm/cancel emit, loading disables confirm button

**E2E tests:**
- Navigate to story detail with backlog status → see "Launch Run" button
- Click "Launch Run" → see confirmation dialog
- Click "Confirm" → see success toast, button changes to "Running..."
- API returns 409 → see warning toast, dialog stays open

### References

- Epic 3: Pipeline Execution Engine
- Backend Story 3-10: Run launch API (POST /api/v1/projects/{id}/stories/{storyId}/runs)
- PrimeVue ConfirmDialog: https://primevue.org/confirmdialog/
- PrimeVue Toast: https://primevue.org/toast/
- PrimeVue Tooltip: https://primevue.org/tooltip/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
