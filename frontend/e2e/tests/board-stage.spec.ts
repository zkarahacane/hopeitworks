import { test, expect } from './fixtures'

// ── Board "Stage" redesign ────────────────────────────────────────────────────
// Covers the user-visible behaviour of the stage-pipeline board refonte:
//   - the détail view derives one column per pipeline stage (the macro view shows
//     the lifecycle), and a card sits in its `current_stage` column;
//   - a card idle in a `manual` stage shows a "Go · start stage" affordance that
//     POSTs to the stage/start endpoint;
//   - a fresh Backlog card shows "Go" which opens the launch confirm dialog.
//
// The suite is fully mocked (see ./fixtures) so it runs without a backend. The
// board fetches three endpoints we stub here: the pipeline config (stage columns),
// the epics list (epic selector + auto-select), and the stories of the first epic.

const PROJECT_ID = 'p1'

const mockProject = {
  id: PROJECT_ID,
  name: 'Test Project',
  description: 'A test project',
  owner_id: 'u1',
  git_provider: 'github',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const mockEpics = [
  {
    id: 'e1',
    project_id: PROJECT_ID,
    name: 'User Authentication',
    description: 'Implement user authentication',
    status: 'in_progress',
    story_counts: { backlog: 1, running: 1, done: 1, failed: 0 },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
]

// Pipeline groups become the détail board's stage columns (in order). The
// `transition` policy drives the manual-idle affordance.
const mockPipelineConfig = {
  project_id: PROJECT_ID,
  groups: [
    { id: 'g-setup', name: 'Setup', transition: 'auto', steps: [] },
    { id: 'g-dev', name: 'Development', transition: 'auto', steps: [] },
    { id: 'g-review', name: 'Review', transition: 'manual', steps: [] },
    { id: 'g-merge', name: 'Merge', transition: 'gate', steps: [] },
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

// Stories spread across lifecycle states + a manual-idle card parked at "Review".
const mockStories = [
  {
    id: 'story-backlog',
    epic_id: 'e1',
    project_id: PROJECT_ID,
    key: 'S-01',
    title: 'Backlog story',
    status: 'backlog',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'story-running',
    epic_id: 'e1',
    project_id: PROJECT_ID,
    key: 'S-02',
    title: 'Running in Development',
    status: 'running',
    current_stage: 'Development',
    latest_run: {
      id: 'run-2',
      status: 'running',
      current_step: {
        id: 'st-2',
        name: 'implement',
        action_type: 'agent_run',
        status: 'running',
        index: 0,
        total: 1,
      },
    },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'story-manual-idle',
    epic_id: 'e1',
    project_id: PROJECT_ID,
    key: 'S-03',
    title: 'Parked at manual Review',
    status: 'running',
    // Parked at the entry of the manual "Review" stage: run paused, no waiting_approval
    // step → the board surfaces the "Go · start stage" affordance.
    current_stage: 'Review',
    latest_run: { id: 'run-3', status: 'paused', current_step: null },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'story-done',
    epic_id: 'e1',
    project_id: PROJECT_ID,
    key: 'S-04',
    title: 'Completed story',
    status: 'done',
    latest_run: { id: 'run-4', status: 'completed' },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
]

/** Stub auth + project + pipeline + epics + stories so the board renders fully. */
async function setupBoardMocks(
  page: import('@playwright/test').Page,
  opts: { stories?: typeof mockStories } = {},
) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: '1', email: 'admin@test.com', name: 'Admin', role: 'admin' }),
    })
  })

  await page.route('**/api/v1/projects/p1', async (route) => {
    // Let sub-resource routes (/epics, /stories, /pipeline) fall through to their handlers.
    const url = route.request().url()
    if (url.includes('/epics') || url.includes('/stories') || url.includes('/pipeline')) {
      return route.fallback()
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockProject),
    })
  })

  await page.route('**/api/v1/projects/p1/pipeline', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockPipelineConfig),
    })
  })

  await page.route('**/api/v1/projects/p1/epics*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: mockEpics, pagination: { total: 1, page: 1, per_page: 20 } }),
    })
  })

  await page.route('**/api/v1/projects/p1/stories*', async (route) => {
    // The stage/start POST lives under /stories/{id}/stage/start — never answer it here.
    if (route.request().url().includes('/stage/')) return route.fallback()
    const stories = opts.stories ?? mockStories
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: stories,
        pagination: { total: stories.length, page: 1, per_page: 50 },
      }),
    })
  })
}

test.describe('Board — Stage view', () => {
  test('détail view renders one column per pipeline stage with cards in their current_stage', async ({
    page,
  }) => {
    await setupBoardMocks(page)
    await page.goto('/projects/p1/board')

    await expect(page.getByRole('heading', { name: 'Story Board' })).toBeVisible()
    // First epic auto-selected → its stories load.
    await expect(page.getByText('Running in Development')).toBeVisible()

    // Switch from the macro lifecycle to the détail (per-stage) view.
    await page.getByRole('button', { name: 'Détail' }).click()

    // Détail columns = Backlog entry lane + one lane per pipeline stage + Done/Failed.
    // Each stage in the pipeline config becomes its own column header.
    await expect(page.getByText('Setup', { exact: true })).toBeVisible()
    await expect(page.getByText('Development', { exact: true })).toBeVisible()
    await expect(page.getByText('Review', { exact: true })).toBeVisible()
    await expect(page.getByText('Merge', { exact: true })).toBeVisible()
    // Entry + terminal lanes flank the stage columns.
    await expect(page.getByText('Backlog', { exact: true })).toBeVisible()
    await expect(page.getByText('Done', { exact: true })).toBeVisible()
    await expect(page.getByText('Failed', { exact: true })).toBeVisible()

    // A card sits in the column matching its current_stage. Each board column is a
    // fixed-width lane (min-w-[240px]); assert co-location by selecting the lane that
    // holds the "Development" header and checking it carries S-02 but not the S-03 card
    // (which is parked in the "Review" stage).
    const columns = page.locator('div.min-w-\\[240px\\]')
    const devColumn = columns.filter({ has: page.getByText('Development', { exact: true }) })
    await expect(devColumn).toHaveCount(1)
    await expect(devColumn).toContainText('Running in Development')
    await expect(devColumn).not.toContainText('Parked at manual Review')

    const reviewColumn = columns.filter({ has: page.getByText('Review', { exact: true }) })
    await expect(reviewColumn).toContainText('Parked at manual Review')
  })

  test('macro view shows the lifecycle columns', async ({ page }) => {
    await setupBoardMocks(page)
    await page.goto('/projects/p1/board')

    // Default view is macro (lifecycle). Force it in case localStorage persisted détail.
    await page.getByRole('button', { name: 'Macro' }).click()

    await expect(page.getByText('Backlog', { exact: true })).toBeVisible()
    await expect(page.getByText('Running', { exact: true })).toBeVisible()
    await expect(page.getByText('In Review', { exact: true })).toBeVisible()
    await expect(page.getByText('Done', { exact: true })).toBeVisible()
    await expect(page.getByText('Failed', { exact: true })).toBeVisible()
  })

  test('a manual-idle card shows "Go · start stage" and starts the stage', async ({ page }) => {
    await setupBoardMocks(page)

    let startCalled = false
    let startUrl = ''
    await page.route(
      '**/api/v1/projects/p1/stories/story-manual-idle/stage/start',
      async (route) => {
        startCalled = true
        startUrl = route.request().url()
        expect(route.request().method()).toBe('POST')
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ id: 'run-3', status: 'running', story_id: 'story-manual-idle' }),
        })
      },
    )

    await page.goto('/projects/p1/board')

    // The parked manual-stage card shows the explicit start-stage label.
    const goButton = page.getByRole('button', { name: 'Go: S-03' })
    await expect(goButton).toHaveText(/Go · start stage/)

    await goButton.click()

    await expect(page.getByText('Stage started')).toBeVisible()
    expect(startCalled).toBe(true)
    expect(startUrl).toContain('/projects/p1/stories/story-manual-idle/stage/start')
  })

  test('a Backlog card shows "Go" which opens the launch confirm dialog', async ({ page }) => {
    await setupBoardMocks(page)
    await page.goto('/projects/p1/board')

    const goButton = page.getByRole('button', { name: 'Go: S-01' })
    await expect(goButton).toBeVisible()
    await expect(goButton).toHaveText('Go')

    await goButton.click()

    // Backlog "Go" launches a fresh run via the confirm dialog, not a direct POST.
    await expect(page.getByText('Launch Story Run')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible()
  })

  test('shows the empty state when the project has no epics', async ({ page }) => {
    await setupBoardMocks(page)
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })

    await page.goto('/projects/p1/board')

    await expect(
      page.getByText('No epics found. Import stories to get started.'),
    ).toBeVisible()
  })
})
