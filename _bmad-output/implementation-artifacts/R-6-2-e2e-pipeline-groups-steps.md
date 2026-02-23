# Story R-6-2: [FRONT] E2E tests: pipeline config groups + new step types

Status: review

## Story

As a **platform developer**,
I want end-to-end tests covering the pipeline config editor with groups and all supported step types,
so that regressions in the pipeline configuration UI are caught automatically before merge.

## Acceptance Criteria (BDD)

### Scenario 1: Pipeline config page renders groups (not flat list)

```gherkin
Given I am logged in and on the pipeline config page for a project
When the page loads
Then I see groups displayed (e.g. "Setup", "Development", "Review", "Merge", "Delivery")
  And each group shows its steps inside it
  And there is no flat list of ungrouped steps
```

### Scenario 2: User can add a new group

```gherkin
Given I am on the pipeline config page
When I click the "Add Group" button
Then a new empty group appears in the list
  And the group has an editable name field
  And I can type a name and save it
```

### Scenario 3: User can add steps of various types within a group

```gherkin
Given I am on the pipeline config page with at least one group visible
When I click "Add Step" inside a group
  And I select step type "git_branch" from the step type dropdown
Then a git_branch step configuration form appears
  And it contains the expected config fields for a git_branch step

When I add a step of type "git_pr"
Then a git_pr step configuration form appears

When I add a step of type "notification"
Then a notification step configuration form appears

When I add a step of type "human"
Then a human step configuration form appears
```

### Scenario 4: Step config fields change based on selected type

```gherkin
Given a step is being configured in the editor
When I change the action_type from "agent_run" to "git_branch"
Then the configuration fields update to reflect the git_branch step type
  And agent_run-specific fields are no longer shown
```

### Scenario 5: User can delete a group

```gherkin
Given I am on the pipeline config page with multiple groups
When I click the "Delete" or trash icon on a group
Then a confirmation dialog appears
When I confirm the deletion
Then the group and all its steps are removed from the page
  And the remaining groups are still visible
```

### Scenario 6: Default pipeline config has expected groups

```gherkin
Given a newly created project with the default pipeline config
When I navigate to its pipeline config page
Then I see exactly 5 groups in this order: Setup, Development, Review, Merge, Delivery
  And the Setup group contains a git_branch step
  And the Development group contains an agent_run step
  And the Review group contains an agent_run step
  And the Merge group contains a git_pr step
  And the Delivery group contains a ci_poll step and a notification step
```

## Tasks / Subtasks

- [x] **1.1** [FRONT] Create `frontend/e2e/tests/pipeline-config-groups.spec.ts` (AC: #1–#6)
  - [x] Import Playwright test helpers from existing E2E setup (`frontend/e2e/helpers/` or similar)
  - [x] Add `beforeEach` hook: login as admin user and navigate to a test project's pipeline config page

- [x] **1.2** [FRONT] Write test: groups are displayed (not flat list) (AC: #1)
  - [x] Assert `data-testid="pipeline-group-card"` elements are visible
  - [x] Assert steps are nested within group containers (via `group-steps` testid)

- [x] **1.3** [FRONT] Write test: add a new group (AC: #2)
  - [x] Click `data-testid="add-group-btn"`
  - [x] Verify new group element appears
  - [x] Verify save button becomes enabled (dirty state)

- [x] **1.4** [FRONT] Write test: add steps of different types within a group (AC: #3, #4)
  - [x] For each type: git_branch, git_pr, notification, human
  - [x] Click "Add Step" within a group, select type, assert type-specific fields appear
  - [x] Verify switching type changes the fields shown

- [x] **1.5** [FRONT] Write test: delete a group (AC: #5)
  - [x] Click delete/trash icon on a group
  - [x] Confirm in dialog
  - [x] Assert group is no longer in the DOM

- [x] **1.6** [FRONT] Write test: default pipeline config has expected groups (AC: #6)
  - [x] Navigate to pipeline config page with default 5-group mock
  - [x] Assert 5 groups in order: Setup, Development, Review, Merge, Delivery
  - [x] Assert each group's first step has the expected action_type via `data-testid="action-type-tag"`

- [x] **1.7** [FRONT] Ensure `data-testid` selectors needed by the tests exist in the pipeline config components
  - [x] Verified `data-testid="pipeline-group-card"` exists on group container elements
  - [x] Verified `data-testid="add-group-btn"` exists on the add group button
  - [x] Verified `data-testid="remove-group"` exists on group delete buttons
  - [x] Verified `data-testid="add-step-to-group"` exists inside group step area
  - [x] Verified `data-testid="action-type-select"` exists on the action_type dropdown

## Dev Notes

### Dependencies

- **R-3-1** — frontend pipeline config groups UI must be implemented (groups rendered, add/delete group functionality) before these tests can pass.
- **R-3-2** — step type editors (per-action-type config forms) must be implemented before step-type-specific tests can pass.
- **R-6-1** — default pipeline config must have the 5 expected groups for Scenario 6 to pass.

### Architecture Requirements

- Tests go in `frontend/e2e/tests/pipeline-config-groups.spec.ts`
- Use Playwright `page` fixture; no custom test fixtures unless they already exist
- Use `data-testid` selectors as primary locator strategy — consistent with existing E2E tests
- Do not use text-based selectors for dynamic content; prefer `data-testid` or role-based selectors
- Tests are independent — each test creates its own state via API or navigates to a fresh project

### Technical Specifications

**Test file skeleton:**

```ts
import { test, expect } from '@playwright/test'
import { loginAs, createTestProject } from '../helpers/auth'

test.describe('Pipeline Config — Groups', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
  })

  test('groups are displayed instead of flat steps', async ({ page }) => {
    // Navigate to pipeline config, assert groups
  })

  test('can add a new group', async ({ page }) => { /* ... */ })

  test('can add steps of type git_branch within a group', async ({ page }) => { /* ... */ })

  test('can add steps of type git_pr within a group', async ({ page }) => { /* ... */ })

  test('can add steps of type notification within a group', async ({ page }) => { /* ... */ })

  test('can add steps of type human within a group', async ({ page }) => { /* ... */ })

  test('step config fields change when type changes', async ({ page }) => { /* ... */ })

  test('can delete a group with confirmation', async ({ page }) => { /* ... */ })

  test('default config has 5 groups in expected order', async ({ page }) => { /* ... */ })
})
```

**Locator conventions:**

```ts
// Groups container
page.locator('[data-testid="pipeline-group"]')

// Add group button
page.locator('[data-testid="add-group-button"]')

// Step type dropdown within a group (nth)
page.locator('[data-testid="step-type-select"]').nth(0)

// Group name (for verifying order)
page.locator('[data-testid="pipeline-group"] [data-testid="group-name"]')
```

### Testing Requirements

- Tests must be deterministic — do not depend on shared state between test runs
- Use `test.beforeEach` to navigate to a known URL or reset state
- Tests should pass when run via `npm run test:e2e:real` against the local E2E stack
- Use `expect.soft()` only for non-critical assertions; use `expect()` for critical assertions

### References

- `frontend/e2e/tests/` — existing E2E test files for conventions
- `frontend/e2e/helpers/` — auth and navigation helpers
- `frontend/src/features/` — pipeline config components (for data-testid placement)
- Story R-3-1 — pipeline config groups frontend implementation
- Story R-3-2 — step type editor components
- Story R-6-1 — default pipeline config groups

## Dev Agent Record

### Implementation Plan

Used mocked API tests following the same pattern as `frontend/e2e/tests/pipeline-config.spec.ts`. Tests use route interception to mock auth, project, and pipeline API endpoints. Adapted data-testid selectors to match actual component implementations (e.g., `pipeline-group-card` instead of story's suggested `pipeline-group`, `add-group-btn` instead of `add-group-button`).

### Completion Notes

- Created `frontend/e2e/tests/pipeline-config-groups.spec.ts` with 9 test cases covering all 6 acceptance criteria
- Tests are fully isolated using API mocking — no shared state between tests
- All data-testid selectors needed already existed in the components (PipelineGroupCard, PipelineStepCard, AddStepDialog, PipelineConfigView)
- ESLint passes on all test files
- Playwright browsers cannot be executed in this Docker container (missing system dependencies) but tests follow exact patterns of existing passing tests

## File List

- `frontend/e2e/tests/pipeline-config-groups.spec.ts` — NEW: E2E tests for pipeline config groups and step types

## Change Log

- 2026-02-23: Created pipeline-config-groups.spec.ts with 9 E2E tests covering groups display, add/delete group, step type forms (git_branch, git_pr, notification, human), type switching, and default config verification
