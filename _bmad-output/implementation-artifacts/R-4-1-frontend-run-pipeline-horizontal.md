# Story R-4-1: [FRONT] Horizontal pipeline view for runs (GitLab CI style)

Status: ready-for-dev

## Story

As a developer monitoring a pipeline run,
I want to see the run's stages and jobs laid out horizontally like GitLab CI,
so that I can immediately understand parallelism, stage progression, and individual step status at a glance.

## Acceptance Criteria (BDD)

**AC1: Horizontal stage/column layout**
- **Given** a run has a pipeline config with multiple groups (stages)
- **When** I view the run detail page
- **Then** I see one column per group/stage displayed side by side (left to right)
- **And** the columns are labeled with the group/stage name
- **And** columns are separated with a visual connector (arrow or line) indicating sequential execution

**AC2: Steps displayed as rows within each stage column**
- **Given** a stage/group column is rendered
- **When** I view that column
- **Then** I see one job row per step in that stage
- **And** each row shows: step name, status icon, and duration (if completed)

**AC3: Status icons and running animation**
- **Given** a step has a specific status
- **When** I view that step's job row
- **Then** the status is indicated by a PrimeVue `Tag` or icon:
  - `pending` → clock icon, neutral severity
  - `running` → spinner/animated indicator, info severity
  - `completed` → check icon, success severity
  - `failed` → X icon, danger severity
  - `cancelled` → dash icon, secondary severity
  - `skipped` → forward icon, secondary severity
- **And** running steps have a pulsing/animated visual indicator (CSS animation)

**AC4: Click step to open log panel**
- **Given** I click on a step's job row
- **When** the click is handled
- **Then** a `step-selected` event is emitted with the step's ID and data
- **And** the clicked step row is visually highlighted (selected state)

**AC5: Derives layout from pipeline config snapshot**
- **Given** the run has a `pipeline_config_snapshot` field containing the config at time of launch
- **When** `RunPipelineView` renders
- **Then** the column structure is derived from `pipeline_config_snapshot.groups`
- **And** each run step is matched to its group by step order or step name
- **And** live step status is taken from the run's `steps[]` array (real-time, via SSE)

**AC6: Fallback — no groups in config**
- **Given** the run's pipeline config snapshot has no groups (legacy flat steps)
- **When** the view renders
- **Then** all steps are displayed in a single column labeled "Pipeline"

**AC7: Responsive horizontal scroll**
- **Given** there are more stages than fit on the screen
- **When** I view the pipeline on a smaller screen
- **Then** the stage columns overflow horizontally with a scroll bar
- **And** the layout does not wrap to a vertical stack on overflow (it stays horizontal and scrolls)

**AC8: Empty/loading states**
- **Given** the run has no steps yet (just launched)
- **When** the pipeline view renders
- **Then** stage columns are still shown (from config snapshot) with steps in `pending` state
- **And** a loading skeleton is shown while the config snapshot is being fetched

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create `RunPipelineView.vue` component (AC: #1, #5, #6, #7, #8)
  - [ ] Place in `frontend/src/features/runs/RunPipelineView.vue`
  - [ ] Props: `run: Run`, `steps: RunStep[]`
  - [ ] Emits: `step-selected: [step: RunStep]`
  - [ ] Computed `stages`: derives stage columns from `run.pipeline_config_snapshot.groups` (or falls back to single "Pipeline" stage)
  - [ ] Computed `stepsByStage`: maps `steps[]` into their respective stage (by step_order range or step_name matching — see Technical Specifications)
  - [ ] Layout: `flex flex-row gap-0 overflow-x-auto` container wrapping `RunStageColumn` components
  - [ ] Handle loading state: show `Skeleton` while `run` is null/undefined
  - [ ] Handle empty steps: show pending placeholders from config snapshot

- [ ] [FRONT] Task 2: Create `RunStageColumn.vue` component (AC: #1, #2)
  - [ ] Place in `frontend/src/features/runs/RunStageColumn.vue`
  - [ ] Props: `stageName: string`, `steps: RunStep[]`, `selectedStepId: string | null`
  - [ ] Emits: `step-selected: [step: RunStep]`
  - [ ] Render stage name in a header row
  - [ ] Render a `RunJobRow` for each step in the stage
  - [ ] Visual connector between columns: right-side arrow or divider (CSS border-right + arrow overlay)
  - [ ] Min width per column: `min-w-48` or `min-w-56` (Tailwind)

- [ ] [FRONT] Task 3: Create `RunJobRow.vue` component (AC: #2, #3, #4)
  - [ ] Place in `frontend/src/features/runs/RunJobRow.vue`
  - [ ] Props: `step: RunStep`, `selected: boolean`
  - [ ] Emits: `click: [step: RunStep]`
  - [ ] Render step name as text
  - [ ] Render status icon/Tag (use `StatusBadge` from `ui/primitives/` if it exists, or a local computed)
  - [ ] Render duration: `formatDuration(step.started_at, step.completed_at)` — show `--` if not yet completed
  - [ ] Apply `selected` styling (border or background highlight) when `selected === true`
  - [ ] Running steps: add CSS class for pulse animation (see Technical Specifications)
  - [ ] Cursor pointer, `@click` emits `click` event

- [ ] [FRONT] Task 4: Add duration formatter utility (AC: #2)
  - [ ] Add `formatDuration(startedAt?: string, completedAt?: string): string` to `frontend/src/utils/formatDuration.ts` (or extend existing date utils)
  - [ ] Returns `mm:ss` format for completed steps, `--` for pending, elapsed `mm:ss` for running (use `Date.now()` - `startedAt`)

- [ ] [FRONT] Task 5: Add `stepsByStage` mapping logic (AC: #5, #6)
  - [ ] Implement `groupStepsByStage(groups: PipelineGroup[], steps: RunStep[]): Map<string, RunStep[]>` in `frontend/src/utils/pipelineStageUtils.ts`
  - [ ] Matching strategy: use step index ranges derived from group step counts in snapshot (see Technical Specifications)
  - [ ] Export and use in `RunPipelineView.vue`

- [ ] [FRONT] Task 6: CSS animation for running steps (AC: #3)
  - [ ] Add `@keyframes pulse-run` in `frontend/src/assets/main.css` or as a scoped style in `RunJobRow.vue`
  - [ ] Apply via class `running-indicator` on the status icon element when `step.status === 'running'`

- [ ] [FRONT] Task 7: Tests
  - [ ] Unit test `RunPipelineView.vue`: correct number of `RunStageColumn` rendered, fallback single column
  - [ ] Unit test `RunStageColumn.vue`: renders step rows, emits `step-selected`
  - [ ] Unit test `RunJobRow.vue`: status icon mapping, selected styling, click emit
  - [ ] Unit test `groupStepsByStage` utility: correct grouping, edge cases (empty groups, extra steps)
  - [ ] Unit test `formatDuration`: pending, running, completed cases

- [ ] [FRONT] Task 8: Lint and type check
  - [ ] `cd frontend && npm run lint` — must pass
  - [ ] `cd frontend && npm run type-check` — must pass
  - [ ] `cd frontend && npm run test:unit` — must pass

## Dev Notes

### Dependencies

- **R-1-1** (PipelineGroup in OpenAPI): `Run.pipeline_config_snapshot` must include `groups[]` for the multi-column layout to work. Without R-1-1, only the single-column fallback can be shown.
- **R-4-3** (RunDetailView integration): this component is consumed by `RunDetailView.vue` in R-4-3.
- No backend dependency beyond what already exists: `Run.pipeline_config_snapshot` and `RunStep[]` are already available from existing endpoints.

### Architecture Requirements

- All three new components are feature-level → `frontend/src/features/runs/`
- `RunPipelineView.vue` is the orchestrator — it owns state and passes data down
- `RunStageColumn.vue` and `RunJobRow.vue` are purely presentational — they receive data via props and emit events
- No direct store access inside `RunStageColumn` or `RunJobRow` — only `RunPipelineView` interacts with stores
- Duration formatting is a pure utility → `utils/` (not in component)

### Technical Specifications

#### Step-to-stage mapping strategy

Each pipeline group in the snapshot has `steps[]` with a count. When matching live `RunStep` objects (which have `step_order: number`) to stages, use the cumulative step count from the config snapshot:

```typescript
// pipelineStageUtils.ts
export function groupStepsByStage(
  groups: PipelineGroup[],
  steps: RunStep[],
): Map<string, RunStep[]> {
  const result = new Map<string, RunStep[]>()
  let offset = 0

  for (const group of groups) {
    const groupStepCount = group.steps.length
    const stageSteps = steps.filter(
      (s) => s.step_order >= offset && s.step_order < offset + groupStepCount,
    )
    result.set(group.id, stageSteps)
    offset += groupStepCount
  }

  return result
}
```

If `groups` is empty (legacy config), return all steps under a single `'default'` key.

#### Status icon/severity mapping

```typescript
const STATUS_CONFIG: Record<string, { icon: string; severity: string }> = {
  pending:           { icon: 'pi pi-clock',        severity: 'secondary' },
  running:           { icon: 'pi pi-spin pi-spinner', severity: 'info' },
  completed:         { icon: 'pi pi-check-circle',  severity: 'success' },
  failed:            { icon: 'pi pi-times-circle',  severity: 'danger' },
  cancelled:         { icon: 'pi pi-minus-circle',  severity: 'secondary' },
  skipped:           { icon: 'pi pi-forward',       severity: 'secondary' },
  waiting_approval:  { icon: 'pi pi-pause-circle',  severity: 'warn' },
}
```

#### Running step animation

```css
/* In main.css or scoped style */
@keyframes pulse-run {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}
.running-indicator {
  animation: pulse-run 1.2s ease-in-out infinite;
}
```

#### Column layout structure (Tailwind)

```vue
<!-- RunPipelineView.vue -->
<div class="flex flex-row overflow-x-auto gap-0 min-h-0">
  <RunStageColumn
    v-for="(stage, idx) in stages"
    :key="stage.id"
    :stage-name="stage.name"
    :steps="stepsByStage.get(stage.id) ?? []"
    :selected-step-id="selectedStepId"
    :is-last="idx === stages.length - 1"
    @step-selected="emit('step-selected', $event)"
  />
</div>
```

```vue
<!-- RunStageColumn.vue: right-side connector -->
<div class="flex flex-col min-w-52 border-r border-surface-200 pr-2 mr-2 relative">
  <!-- header + rows -->
  <!-- connector arrow (absolute position) on right edge, hidden on last column -->
</div>
```

#### formatDuration utility

```typescript
// utils/formatDuration.ts
export function formatDuration(startedAt?: string | null, completedAt?: string | null): string {
  if (!startedAt) return '--'
  const start = new Date(startedAt).getTime()
  const end = completedAt ? new Date(completedAt).getTime() : Date.now()
  const secs = Math.floor((end - start) / 1000)
  const m = Math.floor(secs / 60).toString().padStart(2, '0')
  const s = (secs % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}
```

### Testing Requirements

- Vitest unit tests (all deterministic, no time-dependent assertions):
  - `groupStepsByStage`: 3 groups × 2 steps each → correct mapping; empty groups → single-column fallback
  - `formatDuration`: `('2026-01-01T10:00:00Z', '2026-01-01T10:01:30Z')` → `'01:30'`; no startedAt → `'--'`
  - `RunJobRow.vue`: `selected=true` applies highlight class; click emits step; `running` status has animation class
  - `RunStageColumn.vue`: renders correct number of `RunJobRow` elements; emits `step-selected` on row click
  - `RunPipelineView.vue`: 3 groups → 3 columns; no groups → 1 column labeled "Pipeline"; emits `step-selected`

### References

- `frontend/src/features/runs/RunTimeline.vue` — existing component (to be replaced in R-4-3, use as reference for step data access patterns)
- `frontend/src/features/runs/composables/useRunTimeline.ts` — existing step grouping logic (reference)
- `frontend/src/ui/composed/LogViewer.vue` — referenced in R-4-2
- `frontend/src/utils/` — place new utilities here
- `frontend/CLAUDE.md` — Tailwind layout-only, PrimeVue severity usage

## Dev Agent Record

## Change Log

| Date | Version | Description | Author |
|------|---------|-------------|--------|
| 2026-02-23 | 1.0 | Initial story creation | Arch |
