# Story 2.8: [FRONT] Story Import from Markdown

Status: ready-for-dev

## Story

As a project user, I want to import stories from a markdown file, So that I can populate the board quickly from existing story documentation.

## Acceptance Criteria (BDD)

**AC1: Import dialog opens from Story Board page**
- **Given** I am viewing the Story Board page (`/projects/:id/board`)
- **When** I click "Import Stories" button
- **Then** a Dialog opens with a file upload zone

**AC2: File upload accepts .md files**
- **Given** the import dialog is open
- **When** I drag & drop a `.md` file onto the upload zone, or click to select from filesystem
- **Then** only `.md` files are accepted
- **And** non-markdown files are rejected with an inline error ("Only .md files are supported")
- **And** the selected file name is displayed in the upload zone

**AC3: Local markdown preview before importing**
- **Given** a `.md` file is selected
- **When** the file is parsed locally (FileReader API)
- **Then** a preview table is shown listing detected stories with: key, title, scope (if present)
- **And** stories with parse errors are listed in an "Invalid stories" section with the error reason
- **And** a count summary is shown: "X stories detected, Y invalid"
- **And** the "Import" button is enabled if at least one valid story was detected

**AC4: Import sends POST to backend API**
- **Given** the preview is shown and I click "Import"
- **When** the request is processed
- **Then** POST `/api/v1/projects/{projectId}/stories/import` is called with `{ "content": "<raw file content>" }`
- **And** a loading state is shown on the Import button during the request

**AC5: Import result display**
- **Given** the POST request completes successfully
- **When** the response is received
- **Then** a result summary is shown within the dialog: "X created, Y updated, Z failed"
- **And** per-story errors (if any) are listed with key and error message
- **And** a "Close" button is available
- **And** on close, the story board refreshes (re-fetches stories list)

**AC6: Import API error feedback**
- **Given** the POST request fails (network error or 500)
- **When** the error response arrives
- **Then** a PrimeVue Message with severity "error" is shown inside the dialog
- **And** the dialog stays open so the user can retry or cancel

**AC7: Reset and retry**
- **Given** the import result is shown
- **When** I click "Import Another File"
- **Then** the dialog resets to the initial file upload state
- **And** I can select a different file

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `useStoryImport` composable (AC: #2, #3, #4, #5, #6, #7)
  - [ ] File: `frontend/src/composables/useStoryImport.ts`
  - [ ] State: `fileContent`, `fileName`, `parsedPreview`, `importResult`, `fileError`, `apiError`, `isImporting`
  - [ ] Define `ParsedStoryPreview`: `{ key, title, scope?, valid, error? }`
  - [ ] Define `ImportResult`: `{ imported, updated, failed, errors: { key, message, code }[] }`
  - [ ] `selectFile(file: File)` — validates `.md` extension, reads with `FileReader.readAsText`, on load: parse preview
  - [ ] `parseMarkdownPreview(content)` — lightweight regex: split on `---`, extract key/title from frontmatter + H1, mark invalid if missing
  - [ ] `importStories(projectId)` — calls `apiClient.POST('/projects/{projectId}/stories/import', { body: { content } })`, sets `importResult` or `apiError`
  - [ ] `reset()` — clears all state to initial values

- [ ] [FRONT] Task 2: Create `StoryImportDialog.vue` component (AC: #1, #2, #3, #4, #5, #6, #7)
  - [ ] File: `frontend/src/features/board/StoryImportDialog.vue`
  - [ ] Props: `visible`, `projectId`; Emits: `update:visible`, `imported`
  - [ ] **Step 1 — Upload zone** (when `!fileContent && !importResult`):
    - Drag & drop div with `@dragover.prevent`, `@drop.prevent`, dashed border, `pi-upload` icon
    - Hidden `<input type="file" accept=".md">` triggered on zone click
    - Visual feedback on drag over (`isDragging` ref)
    - File error `<small class="text-red-500">`
  - [ ] **Step 2 — Preview** (when `fileContent && !importResult`):
    - Summary: "X stories detected, Y invalid" with Tags
    - PrimeVue DataTable: columns Key, Title, Scope (valid stories only)
    - Invalid stories `<ul>` if any
    - Footer: Cancel + Import button (`:loading="isImporting"`, `:disabled` if no valid stories)
  - [ ] **Step 3 — Result** (when `importResult`):
    - Summary Tags: "X created" (success), "Y updated" (info), "Z failed" (danger)
    - Per-story errors list if any
    - Footer: "Import Another File" (secondary, calls `reset()`) + "Close" (primary, emits `imported` + closes)
  - [ ] API error `<Message severity="error">` between content and buttons
  - [ ] On dialog close: `reset()`, emit `update:visible: false`

- [ ] [FRONT] Task 3: Add "Import Stories" button to BoardView + wire dialog (AC: #1)
  - [ ] In `frontend/src/views/BoardView.vue`: add `importDialogVisible` ref
  - [ ] Add "Import Stories" Button (icon `pi-upload`, secondary) in header row
  - [ ] Import `StoryImportDialog`, include in template with `:visible` and `:project-id`
  - [ ] On `@imported`: call `retry()` from `useEpics` to refresh epic list with updated story counts

- [ ] [FRONT] Task 4: Handle drag & drop and file input (AC: #2)
  - [ ] In `StoryImportDialog.vue`:
    - `handleDrop(event: DragEvent)` — extract first file, call `selectFile`
    - `handleFileInput(event: Event)` — extract file from input, call `selectFile`
    - `isDragging` ref toggled by `@dragenter`/`@dragleave` for visual highlight
  - [ ] Drop zone styling: dashed border with PrimeVue surface tokens, highlighted on drag

- [ ] [FRONT] Task 5: Unit tests for `useStoryImport` composable (AC: #2, #3, #4, #5, #6)
  - [ ] File: `frontend/src/composables/__tests__/useStoryImport.spec.ts`
  - [ ] `selectFile` with `.txt` → sets `fileError`
  - [ ] `selectFile` with `.md` → reads content, sets `fileContent` (mock FileReader)
  - [ ] `parseMarkdownPreview` with valid block → `[{ key, title, valid: true }]`
  - [ ] `parseMarkdownPreview` with missing key → `valid: false, error set`
  - [ ] `parseMarkdownPreview` with 2 valid stories → returns 2 previews
  - [ ] `importStories` success → `importResult` set
  - [ ] `importStories` API error → `apiError` set, `importResult` null
  - [ ] `reset()` → all state cleared

- [ ] [FRONT] Task 6: Unit tests for `StoryImportDialog.vue` (AC: #2, #3, #5, #7)
  - [ ] File: `frontend/src/features/board/__tests__/StoryImportDialog.spec.ts`
  - [ ] Step 1 renders: upload zone visible when no file selected
  - [ ] File error shown when `fileError` set
  - [ ] Step 2 renders: preview table when `fileContent` set
  - [ ] Import button disabled when all parsed stories invalid
  - [ ] Step 3 renders: summary tags when `importResult` set
  - [ ] "Import Another File" triggers reset (Step 1 reappears)
  - [ ] "Close" emits `imported` and `update:visible: false`

## Dev Notes

### Dependencies

- **Story 2-3 (DONE after wave 7):** Backend `POST /api/v1/projects/{projectId}/stories/import` endpoint — accepts `{ "content": "<raw markdown>" }`, returns `ImportStoriesResult`
- **Story 2-7 (DONE after wave 8):** No direct code dependency — both are additive features on the board page
- **Story 1-16 (DONE):** `apiClient` fully typed from OpenAPI spec

### Architecture Requirements

**Component placement:** `StoryImportDialog.vue` goes in `features/board/` (board domain, not shared).

**Three-step dialog state machine:**
```
Step 1: Upload  →  (file selected and read)  →  Step 2: Preview
Step 2: Preview →  (POST succeeds)           →  Step 3: Result
Step 3: Result  →  ("Import Another File")   →  Step 1: Upload
Step 3: Result  →  ("Close")                 →  emit imported, close dialog
Any step        →  (dialog closed / ESC)     →  reset() + close
```

**Local parsing (`parseMarkdownPreview`) — purpose:** Lightweight local preview only. Does NOT replace the backend parser (Story 2-3). Extracts `key` (from frontmatter) and `title` (from first H1) for display. The actual import runs server-side.

**Board refresh after import:** BoardView uses `useEpics` which fetches epic summaries with story counts. Calling `retry()` refreshes the epic cards. Full story list refresh happens when navigating to an epic detail.

### File Paths (exact)

```
frontend/src/composables/useStoryImport.ts                              (new)
frontend/src/composables/__tests__/useStoryImport.spec.ts               (new)
frontend/src/features/board/StoryImportDialog.vue                       (new)
frontend/src/features/board/__tests__/StoryImportDialog.spec.ts         (new)
frontend/src/views/BoardView.vue                                        (enhance: Import button + StoryImportDialog)
```

### Technical Specifications

**Local parsing implementation:**
```typescript
function parseMarkdownPreview(content: string): ParsedStoryPreview[] {
  const blocks = content.split(/^---$/m).filter(b => b.trim())
  const stories: ParsedStoryPreview[] = []

  for (let i = 0; i < blocks.length - 1; i += 2) {
    const frontmatter = blocks[i]
    const body = blocks[i + 1] ?? ''

    const keyMatch = frontmatter.match(/^key:\s*(.+)$/m)
    const titleMatch = body.match(/^#\s+(.+)$/m)
    const scopeMatch = frontmatter.match(/^scope:\s*(.+)$/m)

    const key = keyMatch?.[1]?.trim() ?? ''
    const title = titleMatch?.[1]?.trim() ?? ''
    const scope = scopeMatch?.[1]?.trim()

    if (!key) {
      stories.push({ key: '(unknown)', title, scope, valid: false, error: 'Missing key in frontmatter' })
    } else if (!title) {
      stories.push({ key, title: '(no title)', scope, valid: false, error: 'Missing H1 title in body' })
    } else {
      stories.push({ key, title, scope, valid: true })
    }
  }
  return stories
}
```

**FileReader usage:**
```typescript
function selectFile(file: File) {
  fileError.value = null
  if (!file.name.endsWith('.md')) {
    fileError.value = 'Only .md files are supported'
    return
  }
  fileName.value = file.name
  const reader = new FileReader()
  reader.onload = (e) => {
    fileContent.value = e.target?.result as string
    parsedPreview.value = parseMarkdownPreview(fileContent.value)
  }
  reader.readAsText(file)
}
```

**API call:**
```typescript
async function importStories(projectId: string): Promise<void> {
  if (!fileContent.value) return
  isImporting.value = true
  apiError.value = null
  try {
    const { data, error } = await apiClient.POST(
      '/projects/{projectId}/stories/import',
      { params: { path: { projectId } }, body: { content: fileContent.value } }
    )
    if (error) {
      apiError.value = 'Import failed. Please try again.'
      return
    }
    importResult.value = data as ImportResult
  } finally {
    isImporting.value = false
  }
}
```

**Drop zone styling:**
```html
<div
  class="flex flex-col items-center justify-center gap-3 p-8 border-2 border-dashed rounded-lg cursor-pointer"
  :style="{
    borderColor: isDragging ? 'var(--p-primary-color)' : 'var(--p-surface-300)',
    backgroundColor: isDragging ? 'var(--p-primary-50)' : 'transparent'
  }"
  @click="triggerFileInput"
  @dragover.prevent
  @dragenter="isDragging = true"
  @dragleave="isDragging = false"
  @drop.prevent="handleDrop"
>
  <i class="pi pi-upload" style="font-size: 2rem; color: var(--p-text-muted-color)" />
  <p style="color: var(--p-text-muted-color)">Drag & drop a .md file here, or click to browse</p>
</div>
<input ref="fileInputRef" type="file" accept=".md" class="hidden" @change="handleFileInput" />
```

**Result summary:**
```html
<div class="flex items-center gap-2">
  <Tag :value="`${importResult.imported} created`" severity="success" />
  <Tag :value="`${importResult.updated} updated`" severity="info" />
  <Tag v-if="importResult.failed > 0" :value="`${importResult.failed} failed`" severity="danger" />
</div>
```

**Mock FileReader in tests:**
```typescript
const mockFileReader = {
  readAsText: vi.fn(),
  onload: null as ((e: ProgressEvent) => void) | null,
  result: null as string | null,
}
vi.stubGlobal('FileReader', vi.fn(() => mockFileReader))
```

### Testing Requirements

See Tasks 5 and 6 for detailed test cases.

### References

- Story 2-3: Backend import endpoint + `ImportStoriesResult` schema
- OpenAPI spec: `api/openapi.yaml` — `ImportStoriesRequest`, `ImportStoriesResult`, `ImportStoryError`
- PrimeVue DataTable, Dialog, Tag, Message components
- FileReader API: https://developer.mozilla.org/en-US/docs/Web/API/FileReader
- Existing BoardView: `frontend/src/views/BoardView.vue`
- Existing useEpics composable: `frontend/src/composables/useEpics.ts`

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Opus 4.6 | Initial story creation |
