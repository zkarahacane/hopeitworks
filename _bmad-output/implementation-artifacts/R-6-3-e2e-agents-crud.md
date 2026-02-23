# Story R-6-3: [FRONT] E2E tests: agents CRUD + scope management

Status: ready-for-dev

## Story

As a **platform developer**,
I want end-to-end tests covering the agents management UI,
so that regressions in agent CRUD operations and scope display are caught automatically before merge.

## Acceptance Criteria (BDD)

### Scenario 1: Agents page is labeled "Agents" (not "Templates")

```gherkin
Given I am logged in and on the project detail page
When I navigate to the agents tab
Then the tab label is "Agents" (not "Templates")
  And the page heading or breadcrumb reflects "Agents"
```

### Scenario 2: Agents list shows scope badges

```gherkin
Given I am on the project agents page
When the list of agents loads
Then each agent row shows a scope badge
  And global agents display a "Global" badge
  And project-scoped agents display a "Project" badge
```

### Scenario 3: Agents list shows model and image columns

```gherkin
Given I am on the project agents page with at least one agent listed
When the table renders
Then a "Model" column is visible showing the agent's model identifier
  And an "Image" column is visible showing the agent's Docker image reference
```

### Scenario 4: User can create a new project-scoped agent

```gherkin
Given I am on the project agents page
When I click the "New Agent" or "Create Agent" button
Then a form or dialog appears with fields: name, model, image, template_content
When I fill in valid values and submit
Then the new agent appears in the agents list
  And it has a "Project" scope badge
  And it shows the name, model, and image I entered
```

### Scenario 5: User can edit a project-scoped agent

```gherkin
Given I am on the project agents page with a project-scoped agent visible
When I click the "Edit" button on that agent
Then an edit form appears pre-filled with the agent's current values
When I change the name and save
Then the updated name is reflected in the agents list
```

### Scenario 6: User can delete a project-scoped agent

```gherkin
Given I am on the project agents page with a project-scoped agent visible
When I click the "Delete" button on that agent
Then a confirmation dialog appears
When I confirm the deletion
Then the agent is removed from the list
  And no error message is shown
```

### Scenario 7: Global agents are not editable from the project scope

```gherkin
Given I am on the project agents page and a global agent is listed
When I inspect the actions available for that agent row
Then no "Edit" button is visible for global agents
  And no "Delete" button is visible for global agents
```

## Tasks / Subtasks

- [ ] **1.1** [FRONT] Create `frontend/e2e/tests/agents.spec.ts` (AC: #1–#7)
  - [ ] Import Playwright test helpers from existing E2E setup
  - [ ] Add `beforeEach` hook: login as admin and navigate to a test project's agents page

- [ ] **1.2** [FRONT] Write test: tab is labeled "Agents" not "Templates" (AC: #1)
  - [ ] Assert the tab element with `data-testid="agents-tab"` or role tab has text "Agents"
  - [ ] Assert no visible element contains "Templates" as the primary navigation label

- [ ] **1.3** [FRONT] Write test: agents list shows scope badges (AC: #2)
  - [ ] Assert at least one `[data-testid="scope-badge"]` element is visible
  - [ ] Verify global agent rows contain text "Global" in their badge
  - [ ] Verify project agent rows contain text "Project" in their badge

- [ ] **1.4** [FRONT] Write test: model and image columns are visible (AC: #3)
  - [ ] Assert column headers "Model" and "Image" exist in the table
  - [ ] Assert at least one row shows a non-empty model and image value

- [ ] **1.5** [FRONT] Write test: create a project-scoped agent (AC: #4)
  - [ ] Click `data-testid="create-agent-button"`
  - [ ] Fill in name, model, image, template_content fields
  - [ ] Submit and assert the new agent appears in the list with "Project" badge

- [ ] **1.6** [FRONT] Write test: edit a project-scoped agent (AC: #5)
  - [ ] Create an agent first (or use one from test setup)
  - [ ] Click `data-testid="edit-agent-button"` on the agent row
  - [ ] Change the name field
  - [ ] Save and assert the updated name appears in the list

- [ ] **1.7** [FRONT] Write test: delete a project-scoped agent (AC: #6)
  - [ ] Create an agent first (or use one from test setup)
  - [ ] Click `data-testid="delete-agent-button"` on the agent row
  - [ ] Confirm in the dialog
  - [ ] Assert the agent no longer appears in the list

- [ ] **1.8** [FRONT] Write test: global agents have no edit/delete buttons (AC: #7)
  - [ ] Seed or verify at least one global agent exists (via API call in beforeEach if needed)
  - [ ] Navigate to the agents page
  - [ ] For each row with scope "Global", assert `data-testid="edit-agent-button"` is not visible
  - [ ] For each row with scope "Global", assert `data-testid="delete-agent-button"` is not visible

- [ ] **1.9** [FRONT] Ensure `data-testid` selectors needed by the tests exist in the agents UI components
  - [ ] Add `data-testid="agents-tab"` on the agents tab navigation element
  - [ ] Add `data-testid="scope-badge"` on agent scope badge elements
  - [ ] Add `data-testid="create-agent-button"` on the create/new agent trigger
  - [ ] Add `data-testid="edit-agent-button"` on per-row edit buttons (only for project-scoped)
  - [ ] Add `data-testid="delete-agent-button"` on per-row delete buttons (only for project-scoped)

## Dev Notes

### Dependencies

- **R-3-3** — the frontend agents management UI (list, create, edit, delete) must be implemented before these tests can pass. This E2E test story validates R-3-3's implementation.
- **R-1-2** — the OpenAPI spec for Agent endpoints must be merged and frontend types generated.
- **R-1-4** — the backend agents table and CRUD API must be implemented and running in the E2E stack.

### Architecture Requirements

- Tests go in `frontend/e2e/tests/agents.spec.ts`
- Use Playwright `page` fixture; reuse existing E2E auth helpers
- Use `data-testid` selectors as primary locator strategy
- Tests must not depend on each other — each test manages its own agent lifecycle (create/cleanup via API if necessary)
- Test project ID can be read from environment variables or seeded during E2E stack setup

### Technical Specifications

**Test file skeleton:**

```ts
import { test, expect } from '@playwright/test'
import { loginAs, navigateToProjectAgents } from '../helpers/navigation'

test.describe('Agents CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, 'admin')
    await navigateToProjectAgents(page, process.env.E2E_PROJECT_ID!)
  })

  test('tab is labeled Agents not Templates', async ({ page }) => {
    await expect(page.getByRole('tab', { name: 'Agents' })).toBeVisible()
    await expect(page.getByRole('tab', { name: 'Templates' })).not.toBeVisible()
  })

  test('shows scope badges for each agent', async ({ page }) => {
    await expect(page.locator('[data-testid="scope-badge"]').first()).toBeVisible()
  })

  test('model and image columns are visible', async ({ page }) => {
    await expect(page.getByRole('columnheader', { name: 'Model' })).toBeVisible()
    await expect(page.getByRole('columnheader', { name: 'Image' })).toBeVisible()
  })

  test('can create a project-scoped agent', async ({ page }) => { /* ... */ })

  test('can edit a project-scoped agent', async ({ page }) => { /* ... */ })

  test('can delete a project-scoped agent', async ({ page }) => { /* ... */ })

  test('global agents have no edit or delete buttons', async ({ page }) => { /* ... */ })
})
```

**API-based agent creation for test isolation:**

```ts
// Use Playwright request context to create an agent via API before testing edit/delete
const response = await page.request.post(`/api/v1/projects/${projectId}/agents`, {
  data: { name: 'Test Agent', model: 'claude-sonnet-4-6', image: 'hopeitworks/agent:latest', template_content: '# Test' }
})
const agent = await response.json()
// Test edit/delete on this agent, then clean up
```

### Testing Requirements

- Use `page.request` for API-side setup/teardown to avoid UI test fragility
- Each test that creates an agent should clean it up in `afterEach` or in the test body after assertions
- Tests must pass when run via `npm run test:e2e:real` against the local E2E stack
- Do not use `page.waitForTimeout()` — use `page.waitForSelector()` or `expect().toBeVisible()` with timeout instead

### References

- `frontend/e2e/tests/` — existing E2E test files for conventions and shared helpers
- `frontend/e2e/helpers/` — auth helpers (loginAs, navigateTo functions)
- `frontend/src/features/` — agents UI components (for data-testid placement)
- `scripts/e2e-stack.sh` — E2E stack lifecycle management
- Story R-3-3 — frontend agents UI implementation (tested by this story)
- Story R-1-2 — Agent OpenAPI spec and generated types
- Story R-1-4 — backend agents table and CRUD API

## Dev Agent Record

## Change Log
