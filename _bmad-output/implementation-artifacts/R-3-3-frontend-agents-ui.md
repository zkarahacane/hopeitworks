# Story R-3-3: [FRONT] Rename Templates → Agents + Agent management UI

Status: ready-for-dev

## Story

As a project developer,
I want to manage AI agents (with model, Docker image, scope, and prompt) through an "Agents" tab instead of "Templates",
so that the UI accurately reflects the domain concept and I can configure all agent properties in one place.

## Acceptance Criteria (BDD)

**AC1: Tab renamed to "Agents"**
- **Given** I navigate to a project's detail page
- **When** I view the tab bar
- **Then** I see "Agents" instead of "Templates"
- **And** clicking "Agents" takes me to the agents list view (previously templates list)

**AC2: Agent list shows model and image columns**
- **Given** I am on the Agents list view
- **When** agents are loaded
- **Then** the DataTable has columns: Name, Scope, Model, Image, Actions
- **And** the Scope column shows a PrimeVue `Tag` with severity:
  - `global` → severity `info` (blue)
  - `project` → severity `success` (green)

**AC3: Agent editor has model, image, and scope fields**
- **Given** I click "New Agent" or "Edit" on an existing agent
- **When** the editor opens (AgentEditorView)
- **Then** I see all existing prompt/template fields
- **And** I additionally see:
  - "Model" — PrimeVue `Select` with LLM model options
  - "Docker Image" — `InputText` (e.g., `ghcr.io/my-org/my-agent:latest`)
  - "Scope" — `Select` with options: `global`, `project`
- **And** when Scope is `global`, a read-only notice is shown: "Global agents can only be edited by administrators"
- **And** saving updates the agent via the API

**AC4: Global agents are read-only for non-admin project users**
- **Given** I am a non-admin project user
- **And** an agent has scope `global`
- **When** I view that agent in the list
- **Then** the "Edit" button is disabled or hidden
- **And** clicking the agent row shows the agent in read-only mode

**AC5: Store and composables renamed**
- **Given** the implementation is complete
- **Then** `stores/promptTemplates.ts` is renamed to `stores/agents.ts`
- **And** the store is exported as `useAgentsStore` (was `useTemplatesStore` / `usePromptTemplatesStore`)
- **And** all imports in views and components reference the new store name
- **And** the old store file is deleted

**AC6: Router updated**
- **Given** the implementation is complete
- **Then** the route `project-templates` is renamed to `project-agents`
- **And** the path remains compatible (or a redirect is added from the old path)
- **And** the router guard and meta are preserved

**AC7: Components renamed**
- **Given** the implementation is complete
- **Then** template-related components are renamed to agent equivalents:
  - `PromptTemplateTable.vue` → `AgentTable.vue`
  - `PromptTemplateEmptyState.vue` → `AgentEmptyState.vue`
  - `TemplateEditorLayout.vue` → `AgentEditorLayout.vue`
  - `TemplateEditorToolbar.vue` → `AgentEditorToolbar.vue`
  - `TemplatePreviewDialog.vue` → `AgentPreviewDialog.vue`
  - `TemplateVariableSidebar.vue` → `AgentVariableSidebar.vue`
- **And** views renamed:
  - `PromptTemplatesView.vue` → `AgentListView.vue`
  - `TemplateEditorView.vue` → `AgentEditorView.vue`
- **And** old files are deleted (not kept as stubs)

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Rename store file and export (AC: #5)
  - [ ] Copy `stores/promptTemplates.ts` → `stores/agents.ts`
  - [ ] Rename `defineStore('promptTemplates', ...)` → `defineStore('agents', ...)`
  - [ ] Rename exported function: `useAgentsStore`
  - [ ] Add `model`, `image`, and `scope` fields to the Agent state type (or use generated type from R-1-2)
  - [ ] Add `fetchAgents`, `createAgent`, `updateAgent`, `deleteAgent` actions referencing `/agents` endpoints (from R-1-2 API)
  - [ ] Delete `stores/promptTemplates.ts`

- [ ] [FRONT] Task 2: Rename all component files (AC: #7)
  - [ ] Rename files in `frontend/src/features/templates/`:
    - `PromptTemplateTable.vue` → `AgentTable.vue`
    - `PromptTemplateEmptyState.vue` → `AgentEmptyState.vue`
    - `TemplateEditorLayout.vue` → `AgentEditorLayout.vue`
    - `TemplateEditorToolbar.vue` → `AgentEditorToolbar.vue`
    - `TemplatePreviewDialog.vue` → `AgentPreviewDialog.vue`
    - `TemplateVariableSidebar.vue` → `AgentVariableSidebar.vue`
  - [ ] Update internal component imports in each file
  - [ ] Update store references from old store to `useAgentsStore`
  - [ ] Consider renaming the feature directory: `features/templates/` → `features/agents/`

- [ ] [FRONT] Task 3: Rename view files (AC: #7)
  - [ ] Rename `views/PromptTemplatesView.vue` → `views/AgentListView.vue`
  - [ ] Rename `views/TemplateEditorView.vue` → `views/AgentEditorView.vue`
  - [ ] Update all internal references in each view

- [ ] [FRONT] Task 4: Update router (AC: #6)
  - [ ] In `frontend/src/router/index.ts`, rename route name `project-templates` → `project-agents`
  - [ ] Update component imports to new view file names
  - [ ] Add redirect: `{ path: '...templates', redirect: '...agents' }` for backward compat (optional, only if deep links exist)
  - [ ] Ensure route meta (auth guards, etc.) is preserved

- [ ] [FRONT] Task 5: Update ProjectDetailView.vue tab (AC: #1)
  - [ ] Change tab label from `'Templates'` to `'Agents'`
  - [ ] Update tab route reference to `'project-agents'`

- [ ] [FRONT] Task 6: Add model, image, scope columns to AgentTable.vue (AC: #2)
  - [ ] Add `Model` column: display `agent.model` as plain text
  - [ ] Add `Image` column: display `agent.image` as monospace text (use a `code` element or `CodeBlock` ui primitive)
  - [ ] Add `Scope` column: render PrimeVue `Tag` with value = scope label, severity = `info` (global) or `success` (project)
  - [ ] Disable/hide edit action for `global` agents when user is not admin (AC: #4)

- [ ] [FRONT] Task 7: Add model, image, scope fields to AgentEditorView.vue (AC: #3, #4)
  - [ ] Add "Model" `Select` with LLM model options (see Technical Specifications for list)
  - [ ] Add "Docker Image" `InputText` with placeholder `ghcr.io/org/agent-name:latest`
  - [ ] Add "Scope" `Select` with options `[{ label: 'Project', value: 'project' }, { label: 'Global', value: 'global' }]`
  - [ ] When scope is `global` and user is not admin: render all fields as read-only, show info `Message` component
  - [ ] Bind new fields to form state and include in save payload

- [ ] [FRONT] Task 8: Fix all broken imports across the codebase
  - [ ] Search for all imports referencing old store, old view names, old component names
  - [ ] Update each import to the new names
  - [ ] Verify no dead imports remain

- [ ] [FRONT] Task 9: Tests
  - [ ] Unit test `useAgentsStore`: fetch, create, update, delete actions
  - [ ] Unit test `AgentTable.vue`: scope Tag severity, edit button disabled for global agents
  - [ ] Unit test `AgentEditorView.vue`: model/image/scope fields rendered, read-only state for global agents

- [ ] [FRONT] Task 10: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- **R-1-2** (Agent in OpenAPI): the `Agent` schema must be available in the generated TypeScript types with `model`, `image`, and `scope` fields. If not yet merged, extend `PromptTemplate` type locally.
- **R-2-5** (AgentService backend): `/agents` endpoints must be implemented for create/update/delete to work end-to-end. The list view can work with GET only in the meantime.

### Architecture Requirements

- The rename is a pure refactor — no behavioral change except adding new fields
- All files under `features/templates/` should move to `features/agents/` to maintain feature directory conventions
- The `stores/promptTemplates.ts` store is fully replaced — no aliasing or re-export shim
- Use `useAuthStore` or project-level RBAC to determine if the current user is admin (for global agent read-only enforcement)

### Technical Specifications

#### LLM model options for Select

```typescript
const LLM_MODEL_OPTIONS = [
  { label: 'Claude Opus 4 (claude-opus-4-6)', value: 'claude-opus-4-6' },
  { label: 'Claude Sonnet 4 (claude-sonnet-4-6)', value: 'claude-sonnet-4-6' },
  { label: 'Claude Haiku 3.5 (claude-haiku-3-5)', value: 'claude-haiku-3-5' },
]
```

Add new models here as they are released. This list can be moved to a shared constant in `utils/models.ts` so it is reused in `AddStepDialog.vue`.

#### Agent type (from OpenAPI R-1-2, stub if not generated)

```typescript
interface Agent {
  id: string
  name: string
  description?: string
  prompt_template: string
  model: string              // e.g. 'claude-opus-4-6'
  image?: string             // Docker image name
  scope: 'global' | 'project'
  project_id?: string        // null for global agents
  variables?: AgentVariable[]
  created_at: string
  updated_at: string
}
```

#### Scope Tag severity mapping

```typescript
function scopeSeverity(scope: 'global' | 'project'): string {
  return scope === 'global' ? 'info' : 'success'
}
```

#### Read-only check for global agents

```typescript
const authStore = useAuthStore()
const isReadOnly = computed(() =>
  agent.value?.scope === 'global' && !authStore.isAdmin
)
```

#### Router rename example

```typescript
// Before
{ name: 'project-templates', path: 'templates', component: () => import('@/views/PromptTemplatesView.vue') }

// After
{ name: 'project-agents', path: 'agents', component: () => import('@/views/AgentListView.vue') }
// Optional redirect for old path
{ path: 'templates', redirect: { name: 'project-agents' } }
```

### Testing Requirements

- Unit tests cover:
  - `useAgentsStore`: all CRUD actions, error handling
  - `AgentTable.vue`: scope badge severity, edit disabled for global + non-admin
  - `AgentEditorView.vue` (or layout): new fields render, read-only when global + non-admin
- Existing template unit tests should be migrated/renamed to agent equivalents
- No new E2E tests required for this story — it is a rename with field additions

### References

- `frontend/src/views/PromptTemplatesView.vue` — rename to `AgentListView.vue`
- `frontend/src/views/TemplateEditorView.vue` — rename to `AgentEditorView.vue`
- `frontend/src/features/templates/` — rename directory to `features/agents/`
- `frontend/src/stores/promptTemplates.ts` — replace with `stores/agents.ts`
- `frontend/src/router/index.ts` — update route name and component imports
- `frontend/src/views/ProjectDetailView.vue` — update tab label and route
- `frontend/CLAUDE.md` — naming conventions, PrimeVue Tag usage

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
