# Story 6.6: [FRONT] Prompt Template Editor

Status: ready-for-dev

## Story

As an admin, I want to edit prompt templates with syntax support, So that I can customize agent prompts effectively.

## Acceptance Criteria (BDD)

**AC1: Display template editor with Monaco editor**
- **Given** I am on `/projects/:id/templates/:templateId`
- **When** the page loads and the template exists
- **Then** I see a Monaco editor displaying the Handlebars template content with syntax highlighting

**AC2: Variable sidebar with context descriptions**
- **Given** I am viewing the template editor
- **When** the page is loaded
- **Then** I see a right sidebar showing available context variables with names and descriptions
- **When** I click on a variable
- **Then** the variable placeholder (e.g., `{{story_key}}`) is inserted at the cursor position in the editor

**AC3: Preview template with sample data**
- **Given** I am viewing the template editor
- **When** I click the "Preview" button
- **Then** I see a dialog displaying the rendered template output using sample context data

**AC4: Save template (admin only)**
- **Given** I am viewing the template editor as an admin
- **When** I modify the template content and click "Save"
- **Then** the template is updated via PUT API, I see a success toast, and the dirty flag is cleared

**AC5: Cancel navigation**
- **Given** I am viewing the template editor
- **When** I click "Cancel"
- **Then** I navigate back to the template list without saving changes

**AC6: Read-only mode for non-admin**
- **Given** I am viewing the template editor as a non-admin user
- **When** the page loads
- **Then** the Monaco editor is read-only and the "Save" button is not visible
- **And** the "Preview" button and "Cancel" button remain visible

**AC7: Create new template**
- **Given** I am on `/projects/:id/templates/new` as an admin
- **When** the page loads
- **Then** I see an empty Monaco editor with all editing capabilities
- **When** I enter content and click "Save"
- **Then** a new template is created via POST API

**AC8: Loading and error states**
- **Given** I am on `/projects/:id/templates/:templateId`
- **When** the API request is pending
- **Then** I see a loading skeleton
- **When** the API request fails
- **Then** I see an error message with a retry button

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create useTemplateEditor composable (AC: #1, #4, #7, #8)
  - [ ] State: template, content (v-model for Monaco), isDirty, loading, saving, error, previewLoading, previewError
  - [ ] Actions: fetchTemplate, saveTemplate, previewTemplate
  - [ ] Computed: isNewTemplate (templateId === 'new'), canSave (admin + dirty)
  - [ ] On mount: if templateId !== 'new', fetch template and populate content
  - [ ] isDirty: compare current content with fetched template_content
  - [ ] Use apiClient GET/PUT /api/v1/projects/{projectId}/templates/{templateId}
  - [ ] Use apiClient POST /api/v1/projects/{projectId}/templates (create mode)

- [ ] [FRONT] Task 2: Build MonacoEditorWrapper.vue component (AC: #1, #6)
  - [ ] Props: modelValue (string), readonly (boolean), language (default 'handlebars')
  - [ ] Emits: update:modelValue
  - [ ] Install @guolao/vue-monaco-editor
  - [ ] Monaco config: language=handlebars, minimap disabled, wordWrap on, theme matching app theme
  - [ ] Expose insertAtCursor(text: string) method for variable insertion
  - [ ] Handle v-model two-way binding

- [ ] [FRONT] Task 3: Build TemplateVariableSidebar.vue component (AC: #2)
  - [ ] Props: editorRef (MonacoEditorWrapper component ref)
  - [ ] Display list of available context variables with PrimeVue Card or Panel
  - [ ] Variables: story_key, story_title, story_objective, target_files, acceptance_criteria, error_context, diff_content, branch_name, repo_url
  - [ ] Each variable: name, description, click handler
  - [ ] Click handler: call editorRef.insertAtCursor(`{{${variable.name}}}`)
  - [ ] Scrollable sidebar, fixed width 250px

- [ ] [FRONT] Task 4: Build TemplateEditorToolbar.vue component (AC: #3, #4, #5, #6)
  - [ ] Props: isAdmin, canSave, isSaving, isDirty
  - [ ] Emits: preview, save, cancel
  - [ ] "Preview" button (always visible)
  - [ ] "Save" button (admin only, enabled when canSave)
  - [ ] "Cancel" button (always visible)
  - [ ] Use PrimeVue Button components
  - [ ] Show loading state on "Save" button when isSaving

- [ ] [FRONT] Task 5: Build TemplatePreviewDialog.vue component (AC: #3)
  - [ ] Props: visible, renderedContent, loading, error
  - [ ] Emits: update:visible
  - [ ] PrimeVue Dialog with "Template Preview" title
  - [ ] Display loading skeleton while loading
  - [ ] Display error message if preview fails
  - [ ] Display rendered content in a read-only text area or pre block with syntax highlighting
  - [ ] "Close" button in footer

- [ ] [FRONT] Task 6: Build TemplateEditorLayout.vue component (AC: #1, #2, #3, #4, #5)
  - [ ] Props: template, content (v-model), isAdmin, isDirty, isSaving, previewVisible (v-model), previewContent, previewLoading
  - [ ] Emits: update:content, update:previewVisible, save, cancel, preview
  - [ ] Split layout: MonacoEditorWrapper (flex-1) + TemplateVariableSidebar (250px fixed right)
  - [ ] TemplateEditorToolbar at top
  - [ ] TemplatePreviewDialog conditional render
  - [ ] Use Flexbox for layout (no custom CSS)

- [ ] [FRONT] Task 7: Build TemplateEditorView.vue route view (AC: #1, #4, #5, #6, #7, #8)
  - [ ] Use useTemplateEditor composable
  - [ ] Use useAuth to check isAdmin
  - [ ] Route params: projectId, templateId from useRoute()
  - [ ] Loading state: PrimeVue Skeleton
  - [ ] Error state: PrimeVue Message with retry button
  - [ ] Success: render TemplateEditorLayout
  - [ ] Handle save: call saveTemplate, show success toast, navigate back on success
  - [ ] Handle cancel: router.push back to template list
  - [ ] Handle preview: call previewTemplate, show dialog with result
  - [ ] Support create mode (templateId === 'new')

- [ ] [FRONT] Task 8: Register routes in router/index.ts (AC: #1, #7)
  - [ ] Path: /projects/:id/templates/:templateId (edit existing)
  - [ ] Path: /projects/:id/templates/new (create new, admin only)
  - [ ] Name: template-editor
  - [ ] Component: TemplateEditorView
  - [ ] Meta: requiresAuth: true

- [ ] [FRONT] Task 9: Write unit tests and E2E test (AC: all)
  - [ ] useTemplateEditor.spec.ts: test fetchTemplate, saveTemplate, previewTemplate, isDirty tracking, create vs edit mode
  - [ ] template-editor.spec.ts (E2E): test editor display, variable insertion, preview, save, cancel, admin vs non-admin, create mode

## Dev Notes

### Dependencies

- Story 1-8: App shell (AppLayout)
- Story 1-9: Login/auth guard (requiresAuth meta)
- Story 1-16: Routing, stores, apiClient setup
- Story 6-5: Prompt template list page (navigation from list to editor)
- Backend peer: Story 6-2 (prompt templates API) — consumes GET/PUT /api/v1/projects/{projectId}/templates/{templateId}, POST /api/v1/projects/{projectId}/templates
- Backend peer: Story 6-3 (Handlebars rendering engine) — consumes POST /api/v1/projects/{projectId}/templates/preview (optional, or use client-side Handlebars)

### Architecture Requirements

Component hierarchy:
```
TemplateEditorView.vue (route view)
├── PrimeVue Skeleton (loading state)
├── PrimeVue Message (error state with retry)
└── TemplateEditorLayout.vue (split layout)
    ├── TemplateEditorToolbar.vue (top bar)
    │   ├── "Preview" button
    │   ├── "Save" button (admin only)
    │   └── "Cancel" button
    ├── MonacoEditorWrapper.vue (main area, flex-1)
    │   └── Monaco Editor (handlebars language mode)
    ├── TemplateVariableSidebar.vue (right sidebar, 250px)
    │   └── List of variables with name + description + click to insert
    └── TemplatePreviewDialog.vue (modal)
        └── Rendered template output
```

State management:
```
stores/promptTemplates.ts (from 6-5) → composables/useTemplateEditor.ts → TemplateEditorView.vue
```

### File Paths (exact)

```
frontend/src/features/templates/MonacoEditorWrapper.vue
frontend/src/features/templates/TemplateVariableSidebar.vue
frontend/src/features/templates/TemplateEditorToolbar.vue
frontend/src/features/templates/TemplateEditorLayout.vue
frontend/src/features/templates/TemplatePreviewDialog.vue
frontend/src/composables/useTemplateEditor.ts
frontend/src/views/TemplateEditorView.vue
frontend/src/router/index.ts (append route)
frontend/src/__tests__/composables/useTemplateEditor.spec.ts
frontend/e2e/tests/template-editor.spec.ts
```

### Technical Specifications

**useTemplateEditor composable:**
```typescript
export function useTemplateEditor(projectId: string, templateId: string | 'new') {
  // State
  const template = ref<PromptTemplate | null>(null)
  const content = ref<string>('')
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<Error | null>(null)
  const previewLoading = ref(false)
  const previewError = ref<Error | null>(null)
  const previewContent = ref<string>('')

  // Computed
  const isNewTemplate = computed(() => templateId === 'new')
  const originalContent = ref<string>('')
  const isDirty = computed(() => content.value !== originalContent.value)
  const canSave = computed(() => isDirty.value && content.value.trim() !== '')

  // Actions
  const fetchTemplate = async () => {
    if (isNewTemplate.value) return
    loading.value = true
    error.value = null
    try {
      const response = await apiClient.GET('/api/v1/projects/{projectId}/templates/{templateId}', {
        params: { path: { projectId, templateId } }
      })
      template.value = response.data
      content.value = response.data.template_content
      originalContent.value = response.data.template_content
    } catch (e) {
      error.value = e
    } finally {
      loading.value = false
    }
  }

  const saveTemplate = async (name: string, type: string) => {
    saving.value = true
    try {
      if (isNewTemplate.value) {
        await apiClient.POST('/api/v1/projects/{projectId}/templates', {
          params: { path: { projectId } },
          body: { name, type, template_content: content.value }
        })
      } else {
        await apiClient.PUT('/api/v1/projects/{projectId}/templates/{templateId}', {
          params: { path: { projectId, templateId } },
          body: { template_content: content.value }
        })
      }
      originalContent.value = content.value
    } finally {
      saving.value = false
    }
  }

  const previewTemplate = async () => {
    previewLoading.value = true
    previewError.value = null
    try {
      // Option 1: Backend preview endpoint
      const response = await apiClient.POST('/api/v1/projects/{projectId}/templates/preview', {
        params: { path: { projectId } },
        body: { template_content: content.value, context: sampleContext }
      })
      previewContent.value = response.data.rendered_content

      // Option 2: Client-side Handlebars (if backend preview not available)
      // import Handlebars from 'handlebars'
      // const template = Handlebars.compile(content.value)
      // previewContent.value = template(sampleContext)
    } catch (e) {
      previewError.value = e
    } finally {
      previewLoading.value = false
    }
  }

  // Auto-fetch on mount
  onMounted(() => {
    if (!isNewTemplate.value) {
      fetchTemplate()
    }
  })

  return {
    template,
    content,
    loading,
    saving,
    error,
    isDirty,
    canSave,
    isNewTemplate,
    previewLoading,
    previewError,
    previewContent,
    fetchTemplate,
    saveTemplate,
    previewTemplate
  }
}
```

**Sample context for preview:**
```typescript
const sampleContext = {
  story_key: 'S-14',
  story_title: 'Add user authentication',
  story_objective: 'Implement JWT-based authentication with refresh tokens',
  target_files: [
    'backend/internal/api/middleware/auth.go',
    'backend/internal/domain/service/auth_service.go'
  ],
  acceptance_criteria: '- Given a valid JWT token, the user can access protected endpoints\n- When the token expires, the user receives a 401 error\n- When the user logs out, the token is invalidated',
  error_context: 'Error: test failed in auth_test.go line 42: expected status 200, got 401',
  diff_content: 'diff --git a/auth.go b/auth.go\nindex 1234567..abcdefg 100644\n--- a/auth.go\n+++ b/auth.go\n@@ -10,6 +10,7 @@ func Login() {\n+    validateToken()\n',
  branch_name: 'feat/1-3-auth',
  repo_url: 'https://github.com/user/repo'
}
```

**MonacoEditorWrapper.vue props/emits:**
```typescript
interface Props {
  modelValue: string       // template content (v-model)
  readonly?: boolean       // non-admin mode
  language?: string        // default 'handlebars'
}
const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// Expose method for variable insertion
const insertAtCursor = (text: string) => {
  // Get Monaco editor instance and insert text at cursor position
  const editor = editorRef.value?.getEditor()
  if (!editor) return

  const selection = editor.getSelection()
  const range = new monaco.Range(
    selection.startLineNumber,
    selection.startColumn,
    selection.endLineNumber,
    selection.endColumn
  )

  editor.executeEdits('', [
    { range, text, forceMoveMarkers: true }
  ])

  editor.focus()
}

defineExpose({ insertAtCursor })
```

**Monaco editor configuration:**
```typescript
const editorOptions = {
  language: props.language || 'handlebars',
  theme: 'vs-dark', // or 'vs-light' based on app theme
  readOnly: props.readonly || false,
  minimap: { enabled: false },
  wordWrap: 'on',
  automaticLayout: true,
  scrollBeyondLastLine: false,
  fontSize: 14,
  lineNumbers: 'on',
  renderWhitespace: 'boundary',
  tabSize: 2
}
```

**TemplateVariableSidebar.vue:**
```typescript
const variables = [
  { name: 'story_key', description: 'Unique story identifier (e.g., S-14)' },
  { name: 'story_title', description: 'Story title/summary' },
  { name: 'story_objective', description: 'Story objective text' },
  { name: 'target_files', description: 'Array of target file paths' },
  { name: 'acceptance_criteria', description: 'Story acceptance criteria text' },
  { name: 'error_context', description: 'Error output from previous failed run (retry only)' },
  { name: 'diff_content', description: 'Git diff from previous attempt (retry/review only)' },
  { name: 'branch_name', description: 'Git branch name for this run' },
  { name: 'repo_url', description: 'Git repository URL' }
]

interface Props {
  editorRef: { insertAtCursor: (text: string) => void } | null
}

const handleVariableClick = (variableName: string) => {
  if (!props.editorRef) return
  props.editorRef.insertAtCursor(`{{${variableName}}}`)
}
```

**TemplateEditorToolbar.vue props/emits:**
```typescript
interface Props {
  isAdmin: boolean
  canSave: boolean
  isSaving: boolean
  isDirty: boolean
}
const emit = defineEmits<{
  preview: []
  save: []
  cancel: []
}>()
```

**TemplatePreviewDialog.vue props/emits:**
```typescript
interface Props {
  visible: boolean
  renderedContent: string
  loading: boolean
  error: Error | null
}
const emit = defineEmits<{
  'update:visible': [visible: boolean]
}>()
```

**TemplateEditorLayout.vue props/emits:**
```typescript
interface Props {
  content: string           // v-model for Monaco
  isAdmin: boolean
  isDirty: boolean
  isSaving: boolean
  previewVisible: boolean   // v-model for dialog
  previewContent: string
  previewLoading: boolean
  previewError: Error | null
}
const emit = defineEmits<{
  'update:content': [content: string]
  'update:previewVisible': [visible: boolean]
  save: []
  cancel: []
  preview: []
}>()
```

**Layout structure (Tailwind only):**
```html
<div class="flex flex-col h-full">
  <!-- Toolbar -->
  <TemplateEditorToolbar class="border-b" />

  <!-- Main content area -->
  <div class="flex flex-1 overflow-hidden">
    <!-- Monaco editor (flex-1) -->
    <MonacoEditorWrapper class="flex-1" />

    <!-- Variable sidebar (fixed 250px) -->
    <TemplateVariableSidebar class="w-[250px] border-l overflow-y-auto" />
  </div>

  <!-- Preview dialog -->
  <TemplatePreviewDialog />
</div>
```

**RBAC rules:**
- Admin: full edit, Save visible, Preview works, can create new templates
- Non-admin: readonly Monaco editor, no Save button, Preview still works, cannot access /new route

**Routes:**
- `/projects/:id/templates/:templateId` — edit existing template
- `/projects/:id/templates/new` — create new template (admin only, navigate guard)

**Install Monaco:**
```bash
cd frontend
npm install @guolao/vue-monaco-editor monaco-editor
```

### Testing Requirements

**Unit tests:**
- useTemplateEditor composable:
  - fetchTemplate success/error for existing template
  - create mode (isNewTemplate = true, no fetch)
  - saveTemplate for create vs update
  - previewTemplate success/error
  - isDirty tracking (content vs originalContent)
  - canSave computed (isDirty + non-empty content)

**E2E tests:**
- Load template editor → see Monaco editor with template content
- Click variable in sidebar → see variable inserted in editor
- Click Preview → see preview dialog with rendered content
- Edit content → Save button enabled (admin), save works, success toast shown
- Cancel → navigate back to template list
- Non-admin → editor read-only, no Save button
- Create mode (/new) → empty editor, save creates new template
- API error → see error message + retry button

### References

- Epic 6: Pipeline Configuration & Prompt Templates
- Story 6-5: Prompt template list page (navigation to editor)
- Backend Story 6-2: Prompt templates API (GET/PUT/POST)
- Backend Story 6-3: Handlebars rendering engine (preview endpoint)
- @guolao/vue-monaco-editor: https://github.com/imguolao/monaco-vue
- Monaco Editor: https://microsoft.github.io/monaco-editor/
- Handlebars: https://handlebarsjs.com/
- PrimeVue Dialog: https://primevue.org/dialog/
- PrimeVue Button: https://primevue.org/button/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
