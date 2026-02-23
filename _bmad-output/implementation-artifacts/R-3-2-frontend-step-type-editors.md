# Story R-3-2: [FRONT] Step type editors for new action types

Status: ready-for-dev

## Story

As a project developer configuring a pipeline,
I want to configure new step action types (git_branch, git_pr, notification, human, ci_poll, hitl_gate) through dedicated form fields in the step dialog,
so that I can set up the full pipeline without manually editing YAML or JSON.

## Acceptance Criteria (BDD)

**AC1: New action types available in AddStepDialog**
- **Given** I open the "Add Step" dialog
- **When** I view the action type selector
- **Then** I see the following options in addition to existing ones:
  - `git_branch` — Create Git Branch
  - `git_pr` — Create Pull Request
  - `notification` — Send Notification
  - `human` — Human Task
  - `ci_poll` — Poll CI Status
  - `hitl_gate` — HITL Gate

**AC2: Conditional config fields — git_branch**
- **Given** I select `git_branch` as the action type
- **When** the form renders
- **Then** I see a "Branch Pattern" text input field
- **And** the model selector is hidden (not applicable for git actions)

**AC3: Conditional config fields — git_pr**
- **Given** I select `git_pr` as the action type
- **When** the form renders
- **Then** I see a "PR Title Template" text input field
- **And** I see a "Target Branch" text input field
- **And** I see a "Draft" toggle switch (PrimeVue `ToggleSwitch`)
- **And** the model selector is hidden

**AC4: Conditional config fields — notification**
- **Given** I select `notification` as the action type
- **When** the form renders
- **Then** I see a "Message" textarea field
- **And** the model selector is hidden

**AC5: Conditional config fields — human**
- **Given** I select `human` as the action type
- **When** the form renders
- **Then** I see a "Message" textarea field (what to display to the human reviewer)
- **And** I see an "Instructions" textarea field (optional detailed instructions)
- **And** the model selector is hidden

**AC6: Conditional config fields — ci_poll**
- **Given** I select `ci_poll` as the action type
- **When** the form renders
- **Then** I see no extra config fields (CI poll uses existing pipeline/run settings)
- **And** the model selector is hidden

**AC7: Conditional config fields — hitl_gate**
- **Given** I select `hitl_gate` as the action type
- **When** the form renders
- **Then** I see no extra config fields
- **And** the model selector is hidden

**AC8: Model selector only for agent_run**
- **Given** I select any action type other than `agent_run`
- **Then** the model selector is NOT displayed
- **Given** I select `agent_run`
- **Then** the model selector IS displayed (existing behavior preserved)

**AC9: Step type icon/badge in PipelineStepCard**
- **Given** a step has a specific `action_type`
- **When** I view the step card in the pipeline config list
- **Then** I see a PrimeVue `Tag` or icon badge indicating the action type
- **And** each type has a distinct icon (see Technical Specifications below)

**AC10: Form submission includes config fields**
- **Given** I have filled in config fields for a `git_pr` step (title template, target branch, draft)
- **When** I click "Add Step"
- **Then** the step is saved with `config.title_template`, `config.target_branch`, and `config.draft` populated in the store

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Update `AddStepDialog.vue` — new action type options (AC: #1)
  - [ ] Add new options to the action type `Select` component: `git_branch`, `git_pr`, `notification`, `human`, `ci_poll`, `hitl_gate`
  - [ ] Use human-readable labels (e.g., "Create Git Branch", "Create Pull Request")
  - [ ] Keep existing options: `agent_run` (and any others already present)

- [ ] [FRONT] Task 2: Implement conditional config fields (AC: #2–#7)
  - [ ] Wrap config field sections with `v-if="selectedActionType === 'git_branch'"` etc.
  - [ ] `git_branch`: `InputText` for `config.branch_pattern`
  - [ ] `git_pr`: `InputText` for `config.title_template`, `InputText` for `config.target_branch`, `ToggleSwitch` for `config.draft`
  - [ ] `notification`: `Textarea` for `config.message`
  - [ ] `human`: `Textarea` for `config.message`, `Textarea` for `config.instructions`
  - [ ] `ci_poll`: no config fields, show info message: "No additional configuration required"
  - [ ] `hitl_gate`: no config fields, show info message: "No additional configuration required"
  - [ ] All fields wrapped in `FloatLabel` for PrimeVue v4 labeling convention

- [ ] [FRONT] Task 3: Conditionally hide model selector (AC: #8)
  - [ ] Extract model selector block into `v-if="selectedActionType === 'agent_run'"`
  - [ ] Ensure model is cleared/reset when switching to a non-agent type

- [ ] [FRONT] Task 4: Bind config fields to reactive form state and emit on submit (AC: #10)
  - [ ] Extend the step form state object to include `config: Record<string, unknown>` sub-object
  - [ ] Each conditional field binds to `formState.config.{field_name}`
  - [ ] On dialog submit, include `config` in the step object passed to the store action
  - [ ] Reset `config` on action type change

- [ ] [FRONT] Task 5: Update `PipelineStepCard.vue` — action type icon/badge (AC: #9)
  - [ ] Add a computed `actionTypeIcon` returning a `pi-*` icon class per action type
  - [ ] Add a PrimeVue `Tag` or icon element showing the type badge in the card header
  - [ ] Map: `agent_run` → `pi-android`, `git_branch` → `pi-code-branch`, `git_pr` → `pi-arrow-right-arrow-left`, `notification` → `pi-bell`, `human` → `pi-user`, `ci_poll` → `pi-sync`, `hitl_gate` → `pi-shield`

- [ ] [FRONT] Task 6: Tests
  - [ ] Unit test `AddStepDialog.vue`: all new action types appear in the select, correct fields show per type, model selector hidden for non-agent types
  - [ ] Unit test form state: switching action type resets config fields
  - [ ] Unit test `PipelineStepCard.vue`: correct icon rendered per action type

- [ ] [FRONT] Task 7: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- **R-1-1** (new action_types in OpenAPI spec): the `action_type` enum in `PipelineStep` must include the new values in the generated TypeScript types. If not yet merged, use a local type union extension:
  ```typescript
  type ActionType = 'agent_run' | 'git_branch' | 'git_pr' | 'notification' | 'human' | 'ci_poll' | 'hitl_gate'
  ```

### Architecture Requirements

- All conditional rendering logic stays in `AddStepDialog.vue` — no sub-components needed for each field group (keep it simple)
- Config fields are typed loosely as `config: Record<string, unknown>` at the step level until R-1-1 generates specific config schemas
- `PipelineStepCard.vue` update is purely presentational (icon badge) — no logic change

### Technical Specifications

#### Action type options array

```typescript
const ACTION_TYPE_OPTIONS = [
  { label: 'Agent Run', value: 'agent_run' },
  { label: 'Create Git Branch', value: 'git_branch' },
  { label: 'Create Pull Request', value: 'git_pr' },
  { label: 'Send Notification', value: 'notification' },
  { label: 'Human Task', value: 'human' },
  { label: 'Poll CI Status', value: 'ci_poll' },
  { label: 'HITL Gate', value: 'hitl_gate' },
]
```

#### Action type icon map

```typescript
const ACTION_TYPE_ICONS: Record<string, string> = {
  agent_run:    'pi pi-android',
  git_branch:   'pi pi-code-branch',
  git_pr:       'pi pi-arrow-right-arrow-left',
  notification: 'pi pi-bell',
  human:        'pi pi-user',
  ci_poll:      'pi pi-sync',
  hitl_gate:    'pi pi-shield',
}
```

#### git_pr config fields (AddStepDialog template snippet)

```vue
<template v-if="formState.action_type === 'git_pr'">
  <FloatLabel class="w-full">
    <InputText id="title-template" v-model="formState.config.title_template" class="w-full" />
    <label for="title-template">PR Title Template</label>
  </FloatLabel>
  <FloatLabel class="w-full">
    <InputText id="target-branch" v-model="formState.config.target_branch" class="w-full" />
    <label for="target-branch">Target Branch</label>
  </FloatLabel>
  <div class="flex items-center gap-2">
    <ToggleSwitch v-model="formState.config.draft" inputId="draft-toggle" />
    <label for="draft-toggle">Draft PR</label>
  </div>
</template>
```

#### Reset on action type change

```typescript
watch(() => formState.action_type, () => {
  formState.config = {}
  if (formState.action_type !== 'agent_run') {
    formState.model = undefined
  }
})
```

### Testing Requirements

- Vitest unit tests for `AddStepDialog.vue`:
  - Each new action type appears in the dropdown
  - Selecting `git_branch` shows branch_pattern field, hides model
  - Selecting `git_pr` shows title_template, target_branch, draft fields, hides model
  - Selecting `notification` shows message textarea, hides model
  - Selecting `human` shows message + instructions textareas, hides model
  - Selecting `ci_poll` shows info message only, hides model
  - Selecting `hitl_gate` shows info message only, hides model
  - Selecting `agent_run` shows model selector
  - Switching type resets config fields
- Vitest unit test for `PipelineStepCard.vue`:
  - Correct icon rendered for each action type

### References

- `frontend/src/features/pipeline/AddStepDialog.vue` — file to modify
- `frontend/src/features/pipeline/PipelineStepCard.vue` — file to modify (icon badge)
- `frontend/CLAUDE.md` — PrimeVue component usage rules
- `api/openapi.yaml` — R-1-1 will define the new action_type enum values

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
