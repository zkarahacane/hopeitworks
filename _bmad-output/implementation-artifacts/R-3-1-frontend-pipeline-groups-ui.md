# Story R-3-1: [FRONT] UI for pipeline groups

Status: ready-for-dev

## Story

As a project developer configuring a pipeline,
I want to organize pipeline steps into named groups (stages),
so that I can structure my pipeline with clear stage separation and manage steps more intuitively.

## Acceptance Criteria (BDD)

**AC1: Groups are displayed as collapsible sections**
- **Given** a pipeline config has one or more `groups[]` defined
- **When** I navigate to the Pipeline tab of a project
- **Then** I see each group rendered as a `PipelineGroupCard` with its name as a header
- **And** each group's steps are displayed as `PipelineStepCard` items inside that group
- **And** each group can be collapsed/expanded by clicking its header

**AC2: Add Group button**
- **Given** I am on the Pipeline configuration page
- **When** I view the page
- **Then** I see an "Add Group" button above the group list
- **When** I click "Add Group"
- **Then** a new group is added (with a default name like "New Group") at the end of the list
- **And** the new group is displayed as a `PipelineGroupCard` with no steps inside

**AC3: Add steps within a group**
- **Given** I am viewing a `PipelineGroupCard` for a specific group
- **When** I click "Add Step" inside that group's card
- **Then** the `AddStepDialog` opens scoped to that group
- **And** on confirmation, the new step is added to that group only

**AC4: Remove a group**
- **Given** I am viewing a `PipelineGroupCard`
- **When** I click the remove/delete button on the group header
- **Then** a confirmation dialog appears
- **And** on confirmation, the group and all its steps are removed from the config

**AC5: Rename a group**
- **Given** I am viewing a `PipelineGroupCard`
- **When** I click the group name (inline edit or edit icon)
- **Then** the name becomes editable (inline `InputText`)
- **And** on blur or Enter, the name is saved to the store

**AC6: Backward compatibility — no groups in config**
- **Given** a pipeline config has no `groups[]` (legacy flat `steps[]` format)
- **When** I navigate to the Pipeline tab
- **Then** the steps are displayed under a single implicit "Default" group
- **And** I can still add steps to that default group

**AC7: Reorder groups**
- **Given** I have multiple pipeline groups
- **When** I drag a group header to a new position in the list
- **Then** the group order is updated in the store
- **And** the UI reflects the new order immediately

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `PipelineGroupCard.vue` component (AC: #1, #3, #4, #5)
  - [ ] Props: `group: PipelineGroup`, `index: number`
  - [ ] Emits: `update:group`, `remove`, `add-step`
  - [ ] Render group name as editable `InputText` (inline, activates on click/focus)
  - [ ] Render collapse/expand toggle using PrimeVue `Panel` or custom accordion logic
  - [ ] Render list of `PipelineStepCard` components for `group.steps`
  - [ ] Add "Add Step" button at the bottom of each group card
  - [ ] Add remove group button (PrimeVue `Button` severity="danger", icon="pi pi-trash") in group header
  - [ ] Wire remove button to emit `remove` with confirmation dialog (`useConfirm`)

- [ ] [FRONT] Task 2: Modify `PipelineStepList.vue` to render groups (AC: #1, #2, #6)
  - [ ] Replace flat `steps[]` iteration with `groups[]` iteration
  - [ ] Render a `PipelineGroupCard` per group
  - [ ] Add "Add Group" button at the top of the list
  - [ ] Handle backward compat: if `config.groups` is empty/undefined, synthesize a single "Default" group wrapping `config.steps`

- [ ] [FRONT] Task 3: Update `usePipelineConfig.ts` composable / `stores/pipelineConfig.ts` store (AC: #2, #3, #4, #5, #7)
  - [ ] Add `addGroup()` action — pushes a new group `{ id: uuid, name: 'New Group', steps: [] }` to `config.groups`
  - [ ] Add `removeGroup(groupId)` action — removes group by id (and all its steps)
  - [ ] Add `renameGroup(groupId, name)` action — updates group name in place
  - [ ] Add `addStepToGroup(groupId, step)` action — appends step to the target group
  - [ ] Add `removeStepFromGroup(groupId, stepId)` action
  - [ ] Add `reorderGroups(fromIndex, toIndex)` action — moves group in the array
  - [ ] Ensure backward compat: if `config.groups` is missing, initialize it from flat `config.steps` as a single "Default" group on load

- [ ] [FRONT] Task 4: Wire group reordering (AC: #7)
  - [ ] Use `@vueuse/core` `useSortable` or a lightweight drag handle approach on group cards
  - [ ] On drag end, call `reorderGroups(from, to)` in the store

- [ ] [FRONT] Task 5: Tests
  - [ ] Unit test `PipelineGroupCard.vue`: renders group name, collapses on click, emits `remove` on delete, emits `add-step` on button click
  - [ ] Unit test updated `usePipelineConfig.ts` / store: `addGroup`, `removeGroup`, `renameGroup`, `addStepToGroup`, backward compat init
  - [ ] Unit test `PipelineStepList.vue`: renders groups, shows default group when no groups defined

- [ ] [FRONT] Task 6: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- **R-1-1** (PipelineGroup in OpenAPI spec): the `PipelineGroup` type and updated `PipelineConfig` schema with `groups[]` field must be available in the generated TypeScript types (`frontend/src/api/generated/schema.d.ts`) before this story can be implemented. If R-1-1 is not merged yet, stub the type locally.
- **R-1-3** (backend groups API): the backend must persist and return `groups[]` in the pipeline config response. For frontend development, mock data can be used in the store.

### Architecture Requirements

- `PipelineGroupCard.vue` is a feature component → place in `frontend/src/features/pipeline/PipelineGroupCard.vue`
- All business logic (add/remove/reorder groups, backward compat) lives in `usePipelineConfig.ts` composable or `stores/pipelineConfig.ts` — not in the component
- Components are visual assemblers only — zero business logic in `.vue` files (project CLAUDE.md rule)
- Drag & drop: if the chosen library adds significant bundle weight, scope it to this feature directory only

### Technical Specifications

#### PipelineGroup type (from OpenAPI R-1-1)

```typescript
// Expected type from generated schema — DO NOT write manually if generated
interface PipelineGroup {
  id: string          // uuid
  name: string
  steps: PipelineStep[]
  collapsed?: boolean // UI-only, not persisted to backend
}

interface PipelineConfig {
  groups: PipelineGroup[]
  // legacy: steps?: PipelineStep[]   // kept for backward compat reading
}
```

#### Backward Compatibility Logic

```typescript
// In usePipelineConfig.ts or store, on config load:
function normalizeConfig(raw: PipelineConfig): PipelineConfig {
  if (!raw.groups || raw.groups.length === 0) {
    return {
      ...raw,
      groups: [
        {
          id: 'default',
          name: 'Default',
          steps: raw.steps ?? [],
        },
      ],
    }
  }
  return raw
}
```

#### PipelineGroupCard layout sketch

```vue
<template>
  <div class="flex flex-col gap-2 border rounded p-4">
    <!-- Header row -->
    <div class="flex items-center justify-between gap-2">
      <Button icon="pi pi-chevron-down" text @click="toggleCollapse" />
      <InputText v-model="localName" size="small" @blur="emitRename" @keydown.enter="emitRename" />
      <Button icon="pi pi-trash" severity="danger" text @click="confirmRemove" />
    </div>
    <!-- Steps list (collapsible) -->
    <div v-show="!collapsed" class="flex flex-col gap-2">
      <PipelineStepCard v-for="step in group.steps" :key="step.id" :step="step" />
      <Button label="Add Step" icon="pi pi-plus" text size="small" @click="emit('add-step', group.id)" />
    </div>
  </div>
</template>
```

### Testing Requirements

- Vitest unit tests for:
  - `PipelineGroupCard`: collapse toggle, inline rename, remove confirm, add-step emit
  - `usePipelineConfig` / store: all group CRUD actions, backward compat normalization
  - `PipelineStepList`: renders N groups, "Add Group" button calls store action
- No E2E tests required for this story (covered by pipeline integration E2E in a later story)

### References

- `frontend/src/features/pipeline/PipelineStepList.vue` — file to modify
- `frontend/src/features/pipeline/PipelineStepCard.vue` — reused inside group cards
- `frontend/src/features/pipeline/AddStepDialog.vue` — opened per group
- `frontend/src/stores/pipelineConfig.ts` — store to extend
- `frontend/src/views/PipelineConfigView.vue` — parent view
- `frontend/CLAUDE.md` — component placement rules, PrimeVue usage, Tailwind layout-only

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
