# Story 9.4: [FRONT] Cost Dashboard Page

Status: ready-for-dev

## Story

As a project owner, I want to view cost metrics for my project on a dedicated dashboard page, So that I can monitor spending trends, understand cost per story, and track usage over time.

## Acceptance Criteria (BDD)

**AC1: Cost tab exists in project navigation**
- **Given** I am viewing a project detail page
- **When** the tabs render
- **Then** a "Costs" tab is visible with icon `pi pi-dollar`
- **And** clicking it navigates to `/projects/:id/costs` and renders the cost dashboard

**AC2: Summary cards display period totals**
- **Given** I am on the cost dashboard
- **When** the page loads with the default 7d period
- **Then** three summary cards render: "Total cost this week" (formatted `$X.XX`), "Total cost this month", "Average cost per story"
- **And** while loading, PrimeVue `Skeleton` placeholders occupy each card

**AC3: Cost over time line chart renders**
- **Given** cost data is returned by the API
- **When** the chart renders
- **Then** a PrimeVue `Chart` component with `type="line"` displays daily cost totals
- **And** the x-axis shows date labels (e.g., "Feb 10"), y-axis shows USD values
- **And** an empty state message renders if there are no data points

**AC4: Period toggle changes the data range**
- **Given** I am on the cost dashboard
- **When** I click "30d"
- **Then** the API is re-fetched with `?period=30d`
- **And** all summary cards and the chart update to reflect the new period
- **And** the active period button is visually highlighted

**AC5: Recent runs table lists run costs**
- **Given** cost data includes run-level aggregates
- **When** the recent runs DataTable renders
- **Then** columns show: story key, status (Tag with severity), started_at (relative time), total cost (`$X.XXXXX`)
- **And** rows are sorted by `started_at DESC`
- **And** clicking a row navigates to `/runs/{runId}`

**AC6: Budget limit display**
- **Given** a project has a configured budget limit (informational)
- **When** the dashboard renders
- **Then** a budget bar displays "Budget: $X.XX / $Y.00 used" as a ProgressBar
- **And** no enforcement occurs — display only

**AC7: Error and empty states**
- **Given** the API returns an error
- **When** the page renders
- **Then** an inline `Message` with severity "error" is shown with a Retry button
- **And** if the API succeeds but returns no data, an `EmptyState` component renders with message "No cost data yet"

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add "Costs" tab + route to ProjectDetailView (AC: #1)
  - [ ] In `frontend/src/views/ProjectDetailView.vue`: add `{ label: 'Costs', icon: 'pi pi-dollar', route: 'project-costs' }` to `tabs` array
  - [ ] In `frontend/src/router/index.ts`: add child route `{ path: 'costs', name: 'project-costs', component: () => import('@/views/CostDashboardView.vue') }`

- [ ] [FRONT] Task 2: Define expected API contract in openapi.yaml (AC: #2, #3, #4, #5)
  - [ ] Add to `api/openapi.yaml`: `GET /projects/{projectId}/costs/summary` returning `CostSummary` schema
  - [ ] Add `GET /projects/{projectId}/costs/chart` returning array of `CostDataPoint` (date + total_cost_usd)
  - [ ] Add `GET /projects/{projectId}/costs/runs` returning paginated list of `RunCostRow` (run_id, story_key, status, started_at, total_cost_usd)
  - [ ] Run `cd frontend && npm run generate-api` to regenerate types

- [ ] [FRONT] Task 3: Create `useCosts` composable (AC: #2, #3, #4, #5)
  - [ ] File: `frontend/src/composables/useCosts.ts`
  - [ ] Accept `projectId: string`
  - [ ] Expose: `period` (ref: `'7d' | '30d'`, default `'7d'`), `summary`, `chartData`, `runs`, `isLoading`, `error`
  - [ ] `fetchAll()` calls all three endpoints with current `period` value in parallel via `Promise.all`
  - [ ] `setPeriod(p)` updates `period` and calls `fetchAll()`
  - [ ] Unit test: `frontend/src/composables/__tests__/useCosts.spec.ts`

- [ ] [FRONT] Task 4: Create summary card sub-component (AC: #2)
  - [ ] File: `frontend/src/features/costs/CostSummaryCard.vue`
  - [ ] Props: `label: string`, `value: string`, `isLoading: boolean`
  - [ ] Renders Skeleton when loading, formatted value otherwise
  - [ ] Used three times in CostDashboardView with different label/value props

- [ ] [FRONT] Task 5: Create cost chart sub-component (AC: #3)
  - [ ] File: `frontend/src/features/costs/CostChart.vue`
  - [ ] Props: `data: CostDataPoint[]`, `isLoading: boolean`
  - [ ] Uses PrimeVue `Chart` with `type="line"`, `chartjs` datasets built from `data` prop
  - [ ] X-axis: date labels formatted as `"MMM d"` (date-fns `format`)
  - [ ] Y-axis: USD values, step size auto
  - [ ] Empty state: `<EmptyState message="No cost data yet" />` when `data.length === 0`

- [ ] [FRONT] Task 6: Create recent runs cost table sub-component (AC: #5)
  - [ ] File: `frontend/src/features/costs/RunCostTable.vue`
  - [ ] Props: `runs: RunCostRow[]`, `isLoading: boolean`
  - [ ] PrimeVue `DataTable` with columns: Story Key, Status (`Tag`), Started (relative via `useRelativeTime`), Cost (`$X.XXXXX`)
  - [ ] Row click emits `navigate` event with `run_id`; parent calls `router.push({ name: 'run-detail', params: { runId } })`
  - [ ] Skeleton rows (3) when `isLoading`

- [ ] [FRONT] Task 7: Create `CostDashboardView.vue` (AC: #1, #2, #3, #4, #5, #6, #7)
  - [ ] File: `frontend/src/views/CostDashboardView.vue`
  - [ ] Composes: `CostSummaryCard` x3, period toggle buttons, `CostChart`, budget `ProgressBar`, `RunCostTable`
  - [ ] Period toggle: two `Button` components (`7d`, `30d`), `:outlined="period !== '7d'"` pattern
  - [ ] Budget bar: only renders if `summary.budget_limit_usd > 0`
  - [ ] Error state: `Message` severity="error" + Retry `Button`
  - [ ] Row click → `router.push({ name: 'run-detail', params: { runId: event.run_id } })`

- [ ] [FRONT] Task 8: Unit tests for useCosts composable (AC: #2, #4)
  - [ ] `frontend/src/composables/__tests__/useCosts.spec.ts`
  - [ ] `setPeriod('30d')` triggers re-fetch with updated period in query param
  - [ ] Parallel fetch: all three endpoints called on `fetchAll()`
  - [ ] API error → `error.value` populated
  - [ ] `isLoading` transitions correctly

## Dev Notes

### Dependencies

- **Story 9.1 (wave 10):** `cost_records` table exists — backend provides data
- **Story 9.2 (wave 11 — NOT YET IMPLEMENTED):** Cost aggregation API endpoints. This story defines the expected API contract in `openapi.yaml` but the backend implementation comes in wave 11. Frontend must function with mock/empty data until wave 11 is merged.
- **Existing composable patterns:** `useRelativeTime`, `useAsyncAction` — use these directly

### Architecture Requirements

Component hierarchy:

```
CostDashboardView.vue
├── CostSummaryCard.vue  x3  (Total week, Total month, Avg/story)
├── [Period toggle buttons]
├── CostChart.vue
├── [Budget ProgressBar]   — conditional
└── RunCostTable.vue
```

`useCosts` composable owns all state: summary, chart data, runs, period, isLoading, error. Views and sub-components are purely presentational.

Route: child under `/projects/:id` — registered as `project-costs` in Vue Router, same shell as Board/Pipeline/Templates tabs.

### File Paths (exact)

```
api/openapi.yaml                                                (extend: cost summary/chart/runs endpoints + schemas)
frontend/src/views/CostDashboardView.vue                        (new)
frontend/src/views/ProjectDetailView.vue                        (extend: add Costs tab)
frontend/src/router/index.ts                                    (extend: add project-costs child route)
frontend/src/composables/useCosts.ts                            (new)
frontend/src/composables/__tests__/useCosts.spec.ts             (new)
frontend/src/features/costs/CostSummaryCard.vue                 (new)
frontend/src/features/costs/CostChart.vue                       (new)
frontend/src/features/costs/RunCostTable.vue                    (new)
```

### Technical Specifications

**OpenAPI schemas to add:**
```yaml
CostSummary:
  type: object
  required: [total_cost_usd, period_start, period_end, avg_cost_per_story]
  properties:
    total_cost_usd: { type: number, format: double }
    total_cost_week_usd: { type: number, format: double }
    total_cost_month_usd: { type: number, format: double }
    avg_cost_per_story_usd: { type: number, format: double }
    budget_limit_usd: { type: number, format: double }
    period_start: { type: string, format: date-time }
    period_end: { type: string, format: date-time }

CostDataPoint:
  type: object
  required: [date, total_cost_usd]
  properties:
    date: { type: string, format: date }
    total_cost_usd: { type: number, format: double }

RunCostRow:
  type: object
  required: [run_id, story_key, status, started_at, total_cost_usd]
  properties:
    run_id: { type: string, format: uuid }
    story_key: { type: string }
    status: { type: string }
    started_at: { type: string, format: date-time }
    total_cost_usd: { type: number, format: double }
```

**useCosts composable sketch:**
```typescript
export function useCosts(projectId: string) {
  const period = ref<'7d' | '30d'>('7d')
  const summary = ref<CostSummary | null>(null)
  const chartData = ref<CostDataPoint[]>([])
  const runs = ref<RunCostRow[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    isLoading.value = true
    error.value = null
    try {
      const [sumRes, chartRes, runsRes] = await Promise.all([
        apiClient.GET('/api/v1/projects/{projectId}/costs/summary', {
          params: { path: { projectId }, query: { period: period.value } }
        }),
        apiClient.GET('/api/v1/projects/{projectId}/costs/chart', {
          params: { path: { projectId }, query: { period: period.value } }
        }),
        apiClient.GET('/api/v1/projects/{projectId}/costs/runs', {
          params: { path: { projectId }, query: { period: period.value } }
        }),
      ])
      if (sumRes.error) throw sumRes.error
      summary.value = sumRes.data ?? null
      chartData.value = chartRes.data ?? []
      runs.value = runsRes.data?.data ?? []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load cost data'
    } finally {
      isLoading.value = false
    }
  }

  function setPeriod(p: '7d' | '30d') {
    period.value = p
    fetchAll()
  }

  onMounted(fetchAll)

  return { period, summary, chartData, runs, isLoading, error, fetchAll, setPeriod }
}
```

**CostChart dataset construction:**
```typescript
const chartDataset = computed(() => ({
  labels: props.data.map(d => format(parseISO(d.date), 'MMM d')),
  datasets: [{
    label: 'Daily Cost (USD)',
    data: props.data.map(d => d.total_cost_usd),
    fill: false,
    tension: 0.3,
    borderColor: '#6366f1',   // PrimeVue primary color
  }]
}))
```

**Currency formatting utility:**
```typescript
// frontend/src/utils/formatCost.ts
export function formatCostUSD(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 5,
  }).format(value)
}
```

**ProjectDetailView tab addition:**
```typescript
const tabs = [
  { label: 'Overview',   icon: 'pi pi-home',    route: 'project-overview' },
  { label: 'Board',      icon: 'pi pi-th-large', route: 'project-board' },
  { label: 'Pipeline',   icon: 'pi pi-cog',      route: 'project-pipeline' },
  { label: 'Templates',  icon: 'pi pi-file',     route: 'project-templates' },
  { label: 'Costs',      icon: 'pi pi-dollar',   route: 'project-costs' },   // NEW
]
```

**Router child route addition:**
```typescript
{
  path: 'costs',
  name: 'project-costs',
  component: () => import('@/views/CostDashboardView.vue'),
},
```

**Budget ProgressBar:**
```vue
<ProgressBar
  v-if="summary && summary.budget_limit_usd > 0"
  :value="(summary.total_cost_usd / summary.budget_limit_usd) * 100"
/>
<span class="text-sm text-surface-500">
  Budget: {{ formatCostUSD(summary.total_cost_usd) }} / {{ formatCostUSD(summary.budget_limit_usd) }} used
</span>
```

### Testing Requirements

**useCosts unit tests:**
- `fetchAll()` calls all three endpoints with correct `period` query param
- `setPeriod('30d')` updates `period` ref and triggers `fetchAll()`
- API error populates `error.value` and clears `isLoading`
- `isLoading` is true during fetch, false after

**Note:** `CostChart` and `RunCostTable` are presentational — no unit tests required unless logic is complex.

### References

- Tab pattern: `frontend/src/views/ProjectDetailView.vue` — existing `tabs` array and `activeIndex` logic
- Relative time: `frontend/src/composables/useRelativeTime.ts`
- Async composable pattern: `frontend/src/composables/useAsyncAction.ts`
- Run detail route: `run-detail` in `frontend/src/router/index.ts`
- PrimeVue Chart: https://primevue.org/chart/ — requires `chart.js` peer dependency (already installed if used elsewhere; check `frontend/package.json`)
- PrimeVue ProgressBar: https://primevue.org/progressbar/

## Dev Agent Record

(Agent execution logs will be appended here)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-17 | Claude Sonnet 4.5 | Initial story creation |
