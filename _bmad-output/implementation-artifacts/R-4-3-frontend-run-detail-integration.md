# Story R-4-3: [FRONT] Integrate new pipeline view into RunDetailView

Status: ready-for-dev

## Story

As a developer monitoring a pipeline run,
I want the run detail page to show the new horizontal pipeline view with a slide-in log panel,
so that I get a modern, GitLab CI-like experience for all my pipeline runs.

## Acceptance Criteria (BDD)

**AC1: RunDetailView uses RunPipelineView**
- **Given** I navigate to a run detail page
- **When** the page loads
- **Then** I see the horizontal pipeline view (`RunPipelineView`) in the main content area
- **And** the old `RunTimeline` component is no longer rendered

**AC2: Clicking a step opens the log panel**
- **Given** I am on the run detail page
- **When** I click on a step in the `RunPipelineView`
- **Then** `RunStepLogPanel` slides in from the right
- **And** the panel displays that step's logs and metadata

**AC3: Clicking a different step switches the panel**
- **Given** the log panel is open for step A
- **When** I click on step B in the pipeline view
- **Then** the panel content updates to show step B's logs and metadata
- **And** no close/reopen animation occurs (it's a content switch)

**AC4: Closing the panel**
- **Given** the log panel is open
- **When** I click the X button in the panel header or press Escape
- **Then** the panel closes
- **And** the pipeline view returns to full width

**AC5: Existing controls preserved — cancel, retry, cost summary**
- **Given** I am on the run detail page
- **When** the run is in progress
- **Then** I still see the Cancel button (from story 3-16)
- **And** I still see the Retry button on failed steps (from story 3-18)
- **And** I still see the cost summary area (from story 3-17)
- **And** these controls behave identically to their current implementation

**AC6: SSE real-time updates preserved**
- **Given** I am on the run detail page for an active run
- **When** a step status changes (via SSE event)
- **Then** the `RunJobRow` for that step updates its status icon immediately
- **And** no manual page refresh is required

**AC7: Page header preserved**
- **Given** I am on the run detail page
- **When** the page loads
- **Then** I still see the run header: run ID, story title, overall status Tag, progress bar, started_at timestamp

**AC8: Backward compatibility — runs without pipeline group info**
- **Given** a run was launched before pipeline groups were introduced (legacy config snapshot)
- **When** I navigate to that run's detail page
- **Then** all steps display in a single "Pipeline" column (no crash, no empty state)
- **And** the log panel still works for each step

**AC9: Old components deprecated gracefully**
- **Given** the implementation is complete
- **Then** `RunTimeline.vue` is either removed or kept as a dead file with a deprecation comment
- **And** `RunLogViewer.vue` is either removed or kept as a dead file with a deprecation comment
- **And** no import or usage of the old components remains in `RunDetailView.vue`

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Modify `RunDetailView.vue` — replace timeline with pipeline view (AC: #1)
  - [ ] Remove `import RunTimeline from ...` and its `<RunTimeline>` usage in the template
  - [ ] Remove `import RunLogViewer from ...` and its `<RunLogViewer>` usage in the template
  - [ ] Add `import RunPipelineView from '@/features/runs/RunPipelineView.vue'`
  - [ ] Add `import RunStepLogPanel from '@/features/runs/RunStepLogPanel.vue'`
  - [ ] Place `<RunPipelineView>` in the main content area with `:run="run"` and `:steps="steps"` props
  - [ ] Wire `@step-selected="handleStepSelected"` on `RunPipelineView`

- [ ] [FRONT] Task 2: Add step selection state and panel wiring (AC: #2, #3, #4)
  - [ ] Add `selectedStep = ref<RunStep | null>(null)` and `isPanelOpen = ref(false)` to `RunDetailView.vue` script
  - [ ] Add `handleStepSelected(step: RunStep)` function: sets `selectedStep.value = step`, sets `isPanelOpen.value = true`
  - [ ] Add `<RunStepLogPanel>` to the template: `:step="selectedStep"` `:run-id="runId"` `:visible="isPanelOpen"` `@close="isPanelOpen = false"`
  - [ ] Layout: use a `flex flex-row` wrapper so the pipeline view and panel coexist horizontally on desktop

- [ ] [FRONT] Task 3: Preserve existing controls (AC: #5)
  - [ ] Verify Cancel button (`RunCancelConfirmDialog.vue` / direct button) still renders
  - [ ] Verify Retry button still renders on failed step rows (wire through `RunPipelineView` or keep as overlay in `RunDetailView`)
  - [ ] Verify cost summary card still renders
  - [ ] If retry/cancel actions were inside `RunTimeline`'s template, move them to `RunJobRow` or to a step action overlay in `RunPipelineView`

- [ ] [FRONT] Task 4: Ensure SSE wiring is intact (AC: #6)
  - [ ] Verify `useRunDetail.ts` composable still drives `steps` reactively (no change needed if composable is preserved)
  - [ ] Confirm `RunPipelineView` receives updated `steps` prop on each SSE event
  - [ ] Confirm `RunJobRow` re-renders with new status when step status changes

- [ ] [FRONT] Task 5: Preserve page header (AC: #7)
  - [ ] Keep the run header section in `RunDetailView.vue` unchanged: run ID, story title, status Tag, progress bar, started_at
  - [ ] Confirm progress bar still computes from step completion ratio

- [ ] [FRONT] Task 6: Deprecate old components (AC: #9)
  - [ ] Add deprecation comment at the top of `RunTimeline.vue`: `// DEPRECATED: replaced by RunPipelineView in R-4-3. Remove after wave is merged.`
  - [ ] Add deprecation comment at the top of `RunLogViewer.vue`: `// DEPRECATED: replaced by RunStepLogPanel in R-4-3. Remove after wave is merged.`
  - [ ] Alternatively, delete both files if no other consumers exist (check imports across codebase before deleting)

- [ ] [FRONT] Task 7: Tests
  - [ ] Unit test `RunDetailView.vue`: `RunPipelineView` rendered, `RunTimeline` not rendered, `RunStepLogPanel` appears when step is selected, panel closes on close event
  - [ ] Unit test step selection flow: `handleStepSelected` sets `selectedStep` and `isPanelOpen=true`
  - [ ] Unit test backward compat: run with no groups in config snapshot → single column rendered (delegated to `RunPipelineView`)

- [ ] [FRONT] Task 8: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- **R-4-1** (RunPipelineView): must be implemented first
- **R-4-2** (RunStepLogPanel): must be implemented first
- **Story 3-16** (cancel), **Story 3-17** (costs), **Story 3-18** (retry): already implemented — this story must preserve all those UI elements without regression

### Architecture Requirements

- `RunDetailView.vue` is the single orchestrator: it owns `selectedStep` and `isPanelOpen` state
- `RunPipelineView` and `RunStepLogPanel` are children — they receive data via props and communicate via events
- No store is added for panel state — local `ref` in `RunDetailView.vue` is sufficient
- The `useRunDetail.ts` composable is not modified — it continues to provide `run`, `steps`, and SSE-driven reactivity

### Technical Specifications

#### RunDetailView.vue structure after integration

```vue
<script setup lang="ts">
import RunPipelineView from '@/features/runs/RunPipelineView.vue'
import RunStepLogPanel from '@/features/runs/RunStepLogPanel.vue'
// ... existing imports (cancel, costs, etc.)

const { run, steps, isLoading } = useRunDetail(projectId, runId)

const selectedStep = ref<RunStep | null>(null)
const isPanelOpen = ref(false)

function handleStepSelected(step: RunStep) {
  selectedStep.value = step
  isPanelOpen.value = true
}
</script>

<template>
  <div class="flex flex-col gap-4 h-full">
    <!-- Header: run info, status, progress, cancel button -->
    <RunHeader :run="run" ... />

    <!-- Cost summary (from 3-17) -->
    <RunCostSummary :run-id="runId" :project-id="projectId" />

    <!-- Main area: pipeline + log panel side by side -->
    <div class="flex flex-row flex-1 min-h-0 relative">
      <!-- Pipeline view: shrinks when panel is open -->
      <div :class="isPanelOpen ? 'flex-1 md:w-1/2' : 'flex-1'">
        <RunPipelineView
          :run="run"
          :steps="steps"
          @step-selected="handleStepSelected"
        />
      </div>

      <!-- Log panel (fixed overlay on mobile, side panel on desktop) -->
      <RunStepLogPanel
        :step="selectedStep"
        :run-id="runId"
        :visible="isPanelOpen"
        @close="isPanelOpen = false"
      />
    </div>
  </div>
</template>
```

#### Retry button placement after RunTimeline removal

The retry button (from story 3-18) was inside `RunTimeline`'s step slot. After this integration, it should be moved to `RunJobRow.vue`:

```vue
<!-- In RunJobRow.vue -->
<Button
  v-if="step.status === 'failed'"
  label="Retry"
  icon="pi pi-refresh"
  severity="warn"
  size="small"
  v-tooltip="'Retry step'"
  @click.stop="emit('retry', step)"
/>
```

Wire `@retry` from `RunJobRow` → `RunStageColumn` → `RunPipelineView` → `RunDetailView` (event bubble chain), then `RunDetailView` calls `runsStore.retryStep(...)`.

Alternatively, keep retry as a button inside the `RunStepLogPanel` header (triggered from the selected step), which is simpler and avoids event bubbling through multiple layers.

**Recommendation:** Place the retry button in `RunStepLogPanel`'s header section. This is more discoverable (user clicks step → sees logs + retry in one place).

#### Cancel button stays in RunDetailView header

The cancel button (from story 3-16) is in the run header area — no change needed.

#### Progress bar computation (preserve)

```typescript
// In useRunDetail.ts or RunDetailView.vue (existing logic, do not change)
const completedCount = computed(() => steps.value.filter(s => s.status === 'completed').length)
const progressPercent = computed(() => steps.value.length > 0 ? (completedCount.value / steps.value.length) * 100 : 0)
```

#### Checking for existing consumers before deleting old components

Before deleting `RunTimeline.vue` and `RunLogViewer.vue`, search for imports:

```bash
grep -r "RunTimeline\|RunLogViewer" frontend/src/ --include="*.vue" --include="*.ts"
```

If no other consumers exist, delete both files. If other consumers exist (e.g., E2E tests, other views), add deprecation comments only.

### Testing Requirements

- Vitest unit tests for `RunDetailView.vue` after integration:
  - `RunPipelineView` rendered (snapshot or component existence check)
  - `RunTimeline` NOT rendered
  - Click on step via `@step-selected` event → `RunStepLogPanel` becomes visible
  - `isPanelOpen = false` after `@close` event from panel
  - Cancel button still present in header
  - Cost summary still rendered
- Regression check: existing `RunDetailView` tests must still pass (update mocks/stubs as needed)

### References

- `frontend/src/views/RunDetailView.vue` — file to modify
- `frontend/src/features/runs/RunPipelineView.vue` — from R-4-1
- `frontend/src/features/runs/RunStepLogPanel.vue` — from R-4-2
- `frontend/src/features/runs/RunTimeline.vue` — to deprecate/remove
- `frontend/src/features/runs/RunLogViewer.vue` — to deprecate/remove
- `frontend/src/features/runs/composables/useRunDetail.ts` — unchanged, provides run + steps + SSE
- `frontend/src/stores/runs.ts` — `retryStep`, `cancelRun` actions (from 3-16, 3-18)
- `frontend/CLAUDE.md` — component placement, props-down events-up, no business logic in views

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
