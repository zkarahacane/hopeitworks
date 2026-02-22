# Story 3-17: Display Step Costs in Run Detail Page

Status: ready-for-dev

## Story

As a developer reviewing a completed run,
I want to see the cost breakdown per step directly in the Run Detail page,
so that I can understand AI spending without navigating to the separate Cost Dashboard.

## Acceptance Criteria (BDD)

**Scenario: Run has cost data**

Given I am on the Run Detail page for a completed run that incurred costs,
When the page finishes loading,
Then I see a cost summary card showing the run's total cost in USD,
And I see each step in the timeline annotated with its individual cost,
And each step annotation shows the model name, token counts (input / output), and cost in USD.

**Scenario: Run has no cost data**

Given I am on the Run Detail page for a run with no recorded costs (e.g., pending or very new run),
When the page finishes loading,
Then the cost summary card shows "$0.00",
And no cost annotations are shown on individual steps.

**Scenario: Cost fetch fails**

Given the `GET /projects/{projectId}/runs/{runId}/costs` endpoint returns an error,
When the page finishes loading,
Then the step timeline renders normally without cost annotations,
And no error is surfaced to the user for the cost fetch (fail silently — costs are supplementary data).

**Scenario: Loading state**

Given the run detail page is loading,
When cost data is still being fetched,
Then a Skeleton placeholder is displayed where the cost summary card will appear.

## Technical Notes

### Endpoint

`GET /projects/{projectId}/runs/{runId}/costs` — already implemented in backend.

Response schema (`RunCostDetail`):
```json
{
  "run_id": "uuid",
  "total_cost": 3.15,
  "steps": [
    {
      "step_id": "uuid",
      "step_name": "implement",
      "model": "claude-opus-4-6",
      "tokens_input": 100000,
      "tokens_output": 20000,
      "cost_usd": 3.00
    }
  ]
}
```

Note: `StepCostBreakdown.step_id` maps to `RunStep.id` in the timeline. Use this to join cost data to the corresponding timeline entry by `step_id`.

### What to build

#### 1. Composable: `frontend/src/features/runs/composables/useRunCosts.ts`

A new composable wrapping the API call, following the `useAsyncAction` pattern:

```typescript
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

type RunCostDetail = components['schemas']['RunCostDetail']
type StepCostBreakdown = components['schemas']['StepCostBreakdown']

export function useRunCosts(projectId: string, runId: string) {
  const { execute, isLoading, data } = useAsyncAction(async () => {
    const { data } = await apiClient.GET(
      '/api/v1/projects/{projectId}/runs/{runId}/costs',
      { params: { path: { projectId, runId } } }
    )
    return data ?? null
  })

  // Return a Map<step_id, StepCostBreakdown> for O(1) lookup in the template
  const costByStepId = computed((): Map<string, StepCostBreakdown> => {
    if (!data.value) return new Map()
    return new Map(data.value.steps.map((s) => [s.step_id, s]))
  })

  return { execute, isLoading, runCost: data, costByStepId }
}
```

Errors are intentionally swallowed — cost data is supplementary, failures must not break the primary run view.

#### 2. Updates to `frontend/src/views/RunDetailView.vue`

- Import and call `useRunCosts(projectId.value, runId.value)` after the run data is available
- Call `execute()` once on mount (or when `run` becomes defined)
- Add a cost summary card **above the step timeline**, below the `<ProgressBar>`:
  - Reuse the existing `CostSummaryCard.vue` component with props:
    - `label="Total Run Cost"`
    - `:value="formatCostUSD(runCost?.total_cost ?? 0)"`
    - `:isLoading="isCostLoading"`
- In the `#content` slot of each `<Timeline>` item, join by `costByStepId.get((item as RunStep).id)` and render cost chip inline under the step name:
  - Show: model name, `↑ {tokens_input.toLocaleString()}` tokens in, `↓ {tokens_output.toLocaleString()}` tokens out, cost in USD
  - Render only when `costByStepId.has(step.id)` — no empty placeholder per step
  - Use `formatCostUSD` from `@/utils/formatCost`

#### 3. No new store needed

Cost data for a single run is view-local (not shared across views). Use the composable directly in the view — no Pinia store required.

#### 4. No new components

- `CostSummaryCard.vue` already accepts `{ label, value, isLoading }` — use as-is.
- The per-step annotation is simple enough to be inline markup in `RunDetailView.vue`.

### Constraints

- No changes to `api/openapi.yaml` — the endpoint and schemas already exist.
- No backend changes required.
- Do NOT modify `CostSummaryCard.vue`, `CostChart.vue`, or `RunCostTable.vue`.
- The cost fetch must not block rendering of the run timeline or logs.

## Tasks / Subtasks

- [ ] **Task 1** — Create `frontend/src/features/runs/composables/useRunCosts.ts`
  - Wraps `GET /projects/{projectId}/runs/{runId}/costs` via `apiClient`
  - Returns `{ execute, isLoading, runCost, costByStepId }` (Map keyed by `step_id`)
  - Errors caught and ignored (fail silently)

- [ ] **Task 2** — Update `frontend/src/views/RunDetailView.vue`
  - Import and initialize `useRunCosts`
  - Call `execute()` once after `projectId` and `runId` are resolved (on mount, guarded by `run` being defined)
  - Add `CostSummaryCard` between `<ProgressBar>` and the Steps section
  - Add per-step cost chip inside the `#content` timeline slot, rendered conditionally when `costByStepId.has(step.id)`

- [ ] **Task 3** — Unit tests: `frontend/src/features/runs/__tests__/useRunCosts.spec.ts`
  - Test: returns `costByStepId` map keyed by `step_id`
  - Test: `runCost` is `null` when API returns empty / 404
  - Test: errors are swallowed (no thrown exception from `execute`)

- [ ] **Task 4** — Lint + type-check
  - `npm run lint`
  - `npm run type-check`
