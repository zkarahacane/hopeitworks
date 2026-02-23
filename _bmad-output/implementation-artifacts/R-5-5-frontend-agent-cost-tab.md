# Story R-5-5: [FRONT] Add "By Agent" tab in cost dashboard

Status: ready-for-dev

## Story

As a **platform user**,
I want to see a dedicated "By Agent" tab in the cost dashboard,
so that I can understand which agents consume the most tokens and cost the most across a project's runs.

## Acceptance Criteria (BDD)

### Scenario 1: "By Agent" tab exists in the cost dashboard

```gherkin
Given I am on the cost dashboard page for a project
When the page loads
Then I see a tab or section labeled "By Agent"
  And clicking it shows the AgentCostTable
  And the previously existing content is still accessible in its own tab
```

### Scenario 2: AgentCostTable displays correct columns

```gherkin
Given the "By Agent" tab is active and agent cost data is loaded
When the table renders
Then the following columns are visible:
  | column          | notes                              |
  | Agent Name      | string                             |
  | Runs            | integer count                      |
  | Tokens In       | formatted with formatTokenCount    |
  | Tokens Out      | formatted with formatTokenCount    |
  | Cost (USD)      | formatted as currency              |
  | % of Total      | percentage calculated client-side  |
  And the table is sorted by Cost (USD) descending by default
```

### Scenario 3: Percentage of total is calculated client-side

```gherkin
Given agent cost data with multiple agents summing to $10.00 total
When the table renders
Then each agent row shows its share as a percentage (e.g. agent with $3.00 shows "30.0%")
  And the percentages are calculated from the sum of cost_usd across all agents in the response
```

### Scenario 4: Empty state when no agent-linked costs exist

```gherkin
Given a project with no cost records linked to agents
When the "By Agent" tab is active
Then the table shows an empty state message (e.g. "No agent cost data available")
  And no error is thrown
```

### Scenario 5: Data is fetched via useCosts composable

```gherkin
Given the AgentCostTable is rendered
When it mounts
Then it calls fetchAgentCosts(projectId) from useCosts.ts
  And the GET /projects/{projectId}/costs/agents endpoint is called
  And loading state is shown while fetching
```

### Scenario 6: Lint and type-check pass

```gherkin
Given all new and modified files
When I run "cd frontend && npm run lint && npm run type-check"
Then no errors are reported
```

## Tasks / Subtasks

- [ ] **1.1** [FRONT] Add `fetchAgentCosts(projectId: string)` method to `frontend/src/composables/useCosts.ts` (AC: #5)
  - [ ] Call `GET /api/v1/projects/{projectId}/costs/agents` using the typed openapi-fetch client
  - [ ] Return reactive `agentCosts`, `agentCostsLoading`, `agentCostsError` refs

- [ ] **1.2** [FRONT] Create `AgentCostTable.vue` in `frontend/src/features/` (costs or pipeline area) (AC: #2, #3, #4)
  - [ ] PrimeVue DataTable with columns: Agent Name, Runs, Tokens In, Tokens Out, Cost (USD), % of Total
  - [ ] Accept `data: AgentCostBreakdown[]` as prop
  - [ ] Calculate total cost and derive percentage per row
  - [ ] Default sort by cost_usd descending
  - [ ] Empty state via DataTable's `emptyMessage` prop
  - [ ] Use `formatTokenCount` for token columns
  - [ ] Use `formatCost` or `Intl.NumberFormat` for USD column

- [ ] **1.3** [FRONT] Update `CostDashboardView.vue` to add a "By Agent" tab (AC: #1)
  - [ ] Use PrimeVue `TabView` / `Tabs` component (whichever is used elsewhere in the app)
  - [ ] Wrap existing content in a "By Run" or "Overview" tab
  - [ ] Add "By Agent" tab that renders `AgentCostTable`
  - [ ] Call `fetchAgentCosts(projectId)` when the tab activates (lazy load)

- [ ] **1.4** [FRONT] Lint and type-check (AC: #6)
  - [ ] `cd frontend && npm run lint`
  - [ ] `cd frontend && npm run type-check`

## Dev Notes

### Dependencies

- **R-5-2** — the `GET /projects/{projectId}/costs/agents` endpoint and `AgentCostBreakdown` TypeScript type must exist (generated from the updated OpenAPI spec) before this story can fetch real data.

### Architecture Requirements

- `AgentCostBreakdown` TypeScript type comes from the generated API client at `frontend/src/api/generated/` — do not manually define it
- Use `useCosts.ts` composable as the single data-fetching layer; do not call the API client directly from components
- Vue 3 Composition API (`<script setup>` with `defineProps`, `computed`, `ref`)
- Tailwind CSS v4 for layout; PrimeVue 4 for DataTable, Tab components

### Technical Specifications

**useCosts.ts addition:**

```ts
const agentCosts = ref<AgentCostBreakdown[]>([])
const agentCostsLoading = ref(false)
const agentCostsError = ref<string | null>(null)

async function fetchAgentCosts(projectId: string) {
  agentCostsLoading.value = true
  agentCostsError.value = null
  const { data, error } = await apiClient.GET('/api/v1/projects/{projectId}/costs/agents', {
    params: { path: { projectId } }
  })
  if (error) {
    agentCostsError.value = 'Failed to load agent costs'
  } else {
    agentCosts.value = data ?? []
  }
  agentCostsLoading.value = false
}
```

**Percentage calculation (client-side in AgentCostTable.vue):**

```ts
const totalCost = computed(() => props.data.reduce((sum, r) => sum + r.cost_usd, 0))

function percentOf(cost: number): string {
  if (totalCost.value === 0) return '0.0%'
  return ((cost / totalCost.value) * 100).toFixed(1) + '%'
}
```

**AgentCostTable DataTable columns:**

```vue
<DataTable :value="props.data" :defaultSortOrder="-1" sortField="cost_usd" emptyMessage="No agent cost data available.">
  <Column field="agent_name" header="Agent Name" sortable />
  <Column field="runs_count" header="Runs" sortable />
  <Column field="tokens_input" header="Tokens In" sortable>
    <template #body="{ data }">{{ formatTokenCount(data.tokens_input) }}</template>
  </Column>
  <Column field="tokens_output" header="Tokens Out" sortable>
    <template #body="{ data }">{{ formatTokenCount(data.tokens_output) }}</template>
  </Column>
  <Column field="cost_usd" header="Cost (USD)" sortable>
    <template #body="{ data }">{{ formatUSD(data.cost_usd) }}</template>
  </Column>
  <Column header="% of Total">
    <template #body="{ data }">{{ percentOf(data.cost_usd) }}</template>
  </Column>
</DataTable>
```

### Testing Requirements

- No new unit test files required; this is a display component
- Verify TypeScript compilation passes with no `any` types
- Manual verification: tab switching works, data loads, empty state shows correctly

### References

- `frontend/src/composables/useCosts.ts` — add fetchAgentCosts method
- `frontend/src/views/CostDashboardView.vue` — add "By Agent" tab
- `frontend/src/utils/formatCost.ts` — formatTokenCount, formatCost utilities
- `frontend/src/api/generated/` — AgentCostBreakdown type (generated from spec in R-5-2)
- Story R-5-2 — backend endpoint + OpenAPI schema (required dependency)

## Dev Agent Record

## Change Log
