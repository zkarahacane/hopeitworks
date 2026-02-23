# Story R-5-4: [FRONT] Display tokens (input/output) in cost dashboard

Status: ready-for-dev

## Story

As a **platform user**,
I want to see token input and output counts alongside USD costs in the cost dashboard,
so that I can understand the token consumption breakdown for runs and steps without having to infer it from cost figures alone.

## Acceptance Criteria (BDD)

### Scenario 1: CostSummaryCard displays total token counts

```gherkin
Given I am on the cost dashboard page
When the summary cards are rendered
Then each cost summary card shows a token count in addition to the USD amount
  And the token count displays inputs and outputs separately or combined (formatted with formatTokenCount)
  And zero token counts are displayed as "0" not as blank
```

### Scenario 2: RunCostTable shows tokens_input and tokens_output columns

```gherkin
Given I am on the cost dashboard page and the run cost table is visible
When the table renders
Then there is a "Tokens In" column displaying tokens_input formatted with formatTokenCount
  And there is a "Tokens Out" column displaying tokens_output formatted with formatTokenCount
  And the columns appear next to the existing cost USD column
```

### Scenario 3: Step cost breakdown also shows tokens per step

```gherkin
Given I am viewing a run detail page with step cost breakdown visible
When the step cost rows are rendered
Then each step row shows tokens_input and tokens_output values
  And the values are formatted using formatTokenCount
```

### Scenario 4: formatTokenCount utility is used (not raw numbers)

```gherkin
Given a token count of 150000
When it is displayed in any cost component
Then the formatted value uses the existing formatTokenCount utility (e.g. "150K" or "150,000")
  And it is never displayed as a raw unformatted integer
```

## Tasks / Subtasks

- [ ] **1.1** [FRONT] Update `CostSummaryCard.vue` to display token counts (AC: #1, #4)
  - [ ] Import `formatTokenCount` from `frontend/src/utils/formatCost.ts`
  - [ ] Add a tokens display section below the USD amount
  - [ ] Show `tokens_input` as "In: {formatted}" and `tokens_output` as "Out: {formatted}"
  - [ ] Use a secondary text style (muted/smaller font) consistent with PrimeVue design

- [ ] **1.2** [FRONT] Update `RunCostTable.vue` to add tokens columns (AC: #2, #4)
  - [ ] Add a `<Column>` for `tokens_input` with header "Tokens In"
  - [ ] Add a `<Column>` for `tokens_output` with header "Tokens Out"
  - [ ] Apply `formatTokenCount` in the column body template
  - [ ] Place columns before or adjacent to the cost USD column

- [ ] **1.3** [FRONT] Update step cost breakdown component (wherever it exists in run detail view) to show tokens_input and tokens_output per step (AC: #3, #4)
  - [ ] Locate the step cost display in `frontend/src/features/` (runs or pipeline area)
  - [ ] Add tokens_input and tokens_output display using `formatTokenCount`

- [ ] **1.4** [FRONT] Lint and type-check (AC: #4)
  - [ ] `cd frontend && npm run lint`
  - [ ] `cd frontend && npm run type-check`

## Dev Notes

### Dependencies

None. This story is purely a frontend display change. The API already returns `tokens_input` and `tokens_output` in the cost record responses. No backend changes required.

### Architecture Requirements

- Use the existing `formatTokenCount` utility from `frontend/src/utils/formatCost.ts` — do not inline formatting logic in components
- Follow PrimeVue 4 DataTable Column conventions already used in `RunCostTable.vue`
- Tailwind CSS v4 for layout classes only — no custom CSS
- All components use Vue 3 Composition API (`<script setup>`)

### Technical Specifications

**formatTokenCount import:**

```ts
import { formatTokenCount } from '@/utils/formatCost'
```

**PrimeVue Column with slot (tokens example):**

```vue
<Column field="tokens_input" header="Tokens In">
  <template #body="{ data }">
    {{ formatTokenCount(data.tokens_input) }}
  </template>
</Column>
```

**CostSummaryCard token display (below USD amount):**

```vue
<div class="text-sm text-muted-color mt-1">
  <span>In: {{ formatTokenCount(props.tokensInput) }}</span>
  <span class="ml-2">Out: {{ formatTokenCount(props.tokensOutput) }}</span>
</div>
```

### Testing Requirements

- No new test files required for this story (pure display logic)
- Verify visually that formatTokenCount output is consistent with existing usage elsewhere in the codebase
- Ensure TypeScript type-check passes with no new `any` types introduced

### References

- `frontend/src/utils/formatCost.ts` — formatTokenCount utility (already exists)
- `frontend/src/features/` — CostSummaryCard.vue, RunCostTable.vue locations
- `frontend/src/composables/useCosts.ts` — data source for cost components
- `frontend/src/views/` — CostDashboardView.vue (entry point for cost page)

## Dev Agent Record

## Change Log
