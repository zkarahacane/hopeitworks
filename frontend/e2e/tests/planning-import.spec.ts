/**
 * E2E spec — Planning connector import dialog + board provenance.
 *
 * All scenarios are fully mocked (no live backend). File uploads use inline
 * Buffer payloads (same content as frontend/e2e/fixtures/planning-stories.md)
 * so the client-side markdown parser produces deterministic output.
 *
 * Live GitHub path is guarded behind E2E_GITHUB_TOKEN + E2E_GH_PROJECT_URL:
 * test.skip is called inside the test body when those vars are absent, so CI
 * stays green without secrets.
 */
import { test, expect } from './fixtures'

// ── Fixture content (mirrors frontend/e2e/fixtures/planning-stories.md) ───────
// 1 epic (Authentication) + 3 stories:
//   AUTH-1  status: done       → mapped done
//   AUTH-2  status: in_progress → mapped backlog (not an execution state)
//   AUTH-3  (no status)        → mapped backlog
const FIXTURE_CONTENT = `---
key: AUTH-1
epic: Authentication
status: done
scope: backend
---
# Implement JWT authentication

Implement JWT-based authentication for the API using HS256 tokens.
---
key: AUTH-2
epic: Authentication
status: in_progress
scope: backend
---
# Add refresh token endpoint

Add an endpoint that issues a new access token given a valid refresh token.
---
key: AUTH-3
epic: Authentication
scope: frontend
---
# Build login page

Create the login page component with form validation and error feedback.
`

// ── Shared mock data ──────────────────────────────────────────────────────────

const MOCK_ADMIN = {
  id: 'u1',
  email: 'admin@hopeitworks.dev',
  name: 'Admin',
  role: 'admin',
}

const MOCK_PROJECT = {
  id: 'p1',
  name: 'Test Project',
  description: 'Test',
  owner_id: 'u1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const MOCK_EPIC_MARKDOWN = {
  id: 'e1',
  project_id: 'p1',
  name: 'Authentication',
  description: null,
  status: 'backlog',
  source: 'markdown',
  external_id: 'Authentication',
  source_url: null,
  synced_at: '2026-06-26T00:00:00Z',
  story_counts: { backlog: 2, running: 0, done: 1, failed: 0 },
  created_at: '2026-06-26T00:00:00Z',
  updated_at: '2026-06-26T00:00:00Z',
}

const MOCK_STORIES_MARKDOWN = [
  {
    id: 's1',
    key: 'AUTH-1',
    title: 'Implement JWT authentication',
    status: 'done',
    source: 'markdown',
    source_url: null,
    synced_at: '2026-06-26T00:00:00Z',
    epic_id: 'e1',
    project_id: 'p1',
    scope: 'backend',
    current_stage: null,
    depends_on: [],
    created_at: '2026-06-26T00:00:00Z',
    updated_at: '2026-06-26T00:00:00Z',
  },
  {
    id: 's2',
    key: 'AUTH-2',
    title: 'Add refresh token endpoint',
    status: 'backlog',
    source: 'markdown',
    source_url: null,
    synced_at: '2026-06-26T00:00:00Z',
    epic_id: 'e1',
    project_id: 'p1',
    scope: 'backend',
    current_stage: null,
    depends_on: [],
    created_at: '2026-06-26T00:00:00Z',
    updated_at: '2026-06-26T00:00:00Z',
  },
  {
    id: 's3',
    key: 'AUTH-3',
    title: 'Build login page',
    status: 'backlog',
    source: 'markdown',
    source_url: null,
    synced_at: '2026-06-26T00:00:00Z',
    epic_id: 'e1',
    project_id: 'p1',
    scope: 'frontend',
    current_stage: null,
    depends_on: [],
    created_at: '2026-06-26T00:00:00Z',
    updated_at: '2026-06-26T00:00:00Z',
  },
]

// PlanningImportResult — dry-run preview (1 epic + 3 stories created)
const MOCK_PREVIEW_RESULT = {
  source: 'markdown',
  dry_run: true,
  source_url: '',
  epics_created: 1,
  epics_updated: 0,
  stories_created: 3,
  stories_updated: 0,
  skipped: 0,
  locked: 0,
  failed: 0,
  errors: [],
  warnings: [],
  items: [
    {
      key: 'Authentication',
      kind: 'epic',
      action: 'create',
      source_url: null,
      mapped_status: 'backlog',
      reason: '',
    },
    {
      key: 'AUTH-1',
      kind: 'story',
      action: 'create',
      source_url: null,
      mapped_status: 'done',
      reason: '',
    },
    {
      key: 'AUTH-2',
      kind: 'story',
      action: 'create',
      source_url: null,
      mapped_status: 'backlog',
      reason: 'status: in_progress → backlog (not an execution state)',
    },
    {
      key: 'AUTH-3',
      kind: 'story',
      action: 'create',
      source_url: null,
      mapped_status: 'backlog',
      reason: '',
    },
  ],
}

// PlanningImportResult — committed (dry_run: false)
const MOCK_COMMIT_RESULT = { ...MOCK_PREVIEW_RESULT, dry_run: false }

// PlanningImportResult — idempotent re-import (all 3 stories skipped)
const MOCK_IDEMPOTENT_RESULT = {
  source: 'markdown',
  dry_run: false,
  source_url: '',
  epics_created: 0,
  epics_updated: 0,
  stories_created: 0,
  stories_updated: 0,
  skipped: 3,
  locked: 0,
  failed: 0,
  errors: [],
  warnings: [],
  items: [
    {
      key: 'AUTH-1',
      kind: 'story',
      action: 'skip',
      source_url: null,
      mapped_status: 'done',
      reason: 'unchanged (hash match)',
    },
    {
      key: 'AUTH-2',
      kind: 'story',
      action: 'skip',
      source_url: null,
      mapped_status: 'backlog',
      reason: 'unchanged (hash match)',
    },
    {
      key: 'AUTH-3',
      kind: 'story',
      action: 'skip',
      source_url: null,
      mapped_status: 'backlog',
      reason: 'unchanged (hash match)',
    },
  ],
}

// PlanningImportResult — GitHub Projects canned (1 locked story with deep-link)
const MOCK_GITHUB_RESULT = {
  source: 'github_projects',
  dry_run: true,
  source_url: 'https://github.com/orgs/acme/projects/42',
  epics_created: 0,
  epics_updated: 0,
  stories_created: 0,
  stories_updated: 0,
  skipped: 0,
  locked: 1,
  failed: 0,
  errors: [],
  warnings: [],
  items: [
    {
      key: 'ACME-1',
      kind: 'story',
      action: 'lock',
      source_url: 'https://github.com/acme/repo/issues/1',
      mapped_status: 'backlog',
      reason: 'running — status & spec frozen',
    },
  ],
}

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Upload the fixture content as a .md file to the hidden file input. */
async function uploadFixture(page: import('@playwright/test').Page) {
  await page.locator('[data-testid="file-input"]').setInputFiles({
    name: 'planning-stories.md',
    mimeType: 'text/markdown',
    buffer: Buffer.from(FIXTURE_CONTENT),
  })
}

/** Register auth/me (admin) and project detail routes. */
async function setupCommonMocks(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_ADMIN),
    })
  })
  // Only intercept the bare project URL, not sub-paths.
  await page.route('**/api/v1/projects/p1', async (route) => {
    const url = route.request().url()
    if (
      url.includes('/epics') ||
      url.includes('/stories') ||
      url.includes('/planning') ||
      url.includes('/pipeline')
    ) {
      return route.fallback()
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_PROJECT),
    })
  })
}

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 1 — Markdown happy path
// ─────────────────────────────────────────────────────────────────────────────
test.describe('Markdown happy path', () => {
  test('upload fixture → preview dry-run → commit → board shows markdown badges', async ({
    page,
  }) => {
    await setupCommonMocks(page)

    // Board starts empty → empty-state CTA visible
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })

    await page.goto('/projects/p1/board')

    // Admin sees the empty-state import CTA (board-empty-import-button)
    await expect(page.getByTestId('board-empty-import-button')).toBeVisible()
    await page.getByTestId('board-empty-import-button').click()

    // Dialog opens: source picker + markdown panel visible by default
    await expect(page.getByTestId('source-picker')).toBeVisible()
    await expect(page.getByTestId('markdown-panel')).toBeVisible()
    await expect(page.getByTestId('drop-zone')).toBeVisible()

    // Upload fixture → client-side parse preview (3 valid stories)
    await uploadFixture(page)
    await expect(page.getByText('3 stories detected')).toBeVisible()
    await expect(page.getByTestId('preview-table')).toBeVisible()

    // Preview (dry_run: true) — shows per-item decisions
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PREVIEW_RESULT),
      })
    })
    await page.getByTestId('preview-button').click()

    // Tally: 4 created (1 epic + 3 stories), label "Preview"
    await expect(page.getByTestId('import-result')).toBeVisible()
    await expect(page.getByTestId('preview-tally')).toContainText('4 created')
    await expect(page.getByTestId('preview-result-table')).toBeVisible()

    // Wire up the commit route and the post-import board refresh mocks
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_COMMIT_RESULT),
      })
    })
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [MOCK_EPIC_MARKDOWN],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: MOCK_STORIES_MARKDOWN,
          pagination: { total: 3, page: 1, per_page: 20 },
        }),
      })
    })

    await page.getByTestId('import-button').click()

    // Committed state: tally label switches to "Imported"; close/import-another appear
    await expect(page.getByTestId('preview-tally')).toContainText('Imported')
    await expect(page.getByTestId('close-button')).toBeVisible()
    await expect(page.getByTestId('import-another-button')).toBeVisible()

    // Close dialog → board shows story cards with markdown source badge
    await page.getByTestId('close-button').click()
    await expect(page.locator('[data-testid="source-badge"]').first()).toBeVisible()
    await expect(page.locator('[data-testid="source-badge"]').first()).toHaveAttribute(
      'data-source',
      'markdown',
    )
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 2 — Idempotency
// ─────────────────────────────────────────────────────────────────────────────
test.describe('Idempotency', () => {
  test('re-importing the same unchanged file → 0 created, 3 unchanged', async ({ page }) => {
    await setupCommonMocks(page)

    // Board already has an epic (post-first-import state)
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [MOCK_EPIC_MARKDOWN],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_IDEMPOTENT_RESULT),
      })
    })

    await page.goto('/projects/p1/board')

    // Non-empty board → header Re-import button
    await expect(page.getByTestId('board-import-button')).toBeVisible()
    await page.getByTestId('board-import-button').click()

    await uploadFixture(page)
    await expect(page.getByText('3 stories detected')).toBeVisible()

    // Import directly (skip preview step)
    await page.getByTestId('import-button').click()

    // Tally: 0 created, 3 unchanged (skipped=3)
    await expect(page.getByTestId('preview-tally')).toContainText('0 created')
    await expect(page.getByTestId('preview-tally')).toContainText('3 unchanged')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 3 — Status mapping
// ─────────────────────────────────────────────────────────────────────────────
test.describe('Status mapping', () => {
  test('done → done; in_progress → backlog; absent status → backlog', async ({ page }) => {
    await setupCommonMocks(page)

    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_PREVIEW_RESULT),
      })
    })

    await page.goto('/projects/p1/board')
    await page.getByTestId('board-empty-import-button').click()

    await uploadFixture(page)
    await page.getByTestId('preview-button').click()

    await expect(page.getByTestId('preview-result-table')).toBeVisible()

    // AUTH-1 (frontmatter status: done) → mapped_status: done
    const auth1Row = page.getByTestId('preview-result-table').locator('tr', { hasText: 'AUTH-1' })
    await expect(auth1Row).toContainText('done')

    // AUTH-2 (frontmatter status: in_progress) → mapped_status: backlog
    // in_progress is an execution-axis state — the importer must never produce it.
    const auth2Row = page.getByTestId('preview-result-table').locator('tr', { hasText: 'AUTH-2' })
    await expect(auth2Row).not.toContainText('in_progress')
    await expect(auth2Row).toContainText('backlog')

    // AUTH-3 (no status field) → mapped_status: backlog
    const auth3Row = page.getByTestId('preview-result-table').locator('tr', { hasText: 'AUTH-3' })
    await expect(auth3Row).toContainText('backlog')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 4 — Lock-while-running (skipped: requires a live run)
// ─────────────────────────────────────────────────────────────────────────────
test.describe('Lock-while-running', () => {
  test.skip(
    're-import a running story → title frozen, locked count ≥ 1',
    // Skipped: starting a run requires a live backend stack (Docker agent, CI env).
    // Manual exercise: start a run for AUTH-2, then re-import a fixture that changes
    // its title. Assert:
    //   - preview-tally shows "1 locked"
    //   - the decision row for AUTH-2 reads "running — status & spec frozen"
    //   - AUTH-2 title on the board is unchanged
    //   - source, source_url, synced_at are still refreshed (provenance update on lock)
    async () => {},
  )
})

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 5 — GitHub Projects UI + provenance (route-mocked)
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GitHub Projects — UI and mocked import', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })
  })

  test('fields visible; empty URL disables submit; 400 shows inline error; canned result shows lock + deep-link', async ({
    page,
  }) => {
    await page.goto('/projects/p1/board')
    await page.getByTestId('board-empty-import-button').click()

    // Switch to GitHub Projects via the source-picker SelectButton
    await page.getByTestId('source-picker').getByText('GitHub Projects').click()

    // GitHub panel and its four input fields are visible
    await expect(page.getByTestId('github-panel')).toBeVisible()
    await expect(page.getByTestId('github-project-url')).toBeVisible()
    await expect(page.getByTestId('github-status-field')).toBeVisible()
    await expect(page.getByTestId('github-done-options')).toBeVisible()
    await expect(page.getByTestId('github-epic-issue-type')).toBeVisible()

    // Empty URL → Preview and Import buttons are disabled (client-side guard)
    await expect(page.getByTestId('preview-button')).toBeDisabled()
    await expect(page.getByTestId('import-button')).toBeDisabled()

    // Provide an invalid URL and mock a 400 → api-error message appears inline
    await page.getByTestId('github-project-url').fill('not-a-valid-url')
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'VALIDATION', message: 'invalid project_url' } }),
      })
    })
    await page.getByTestId('preview-button').click()
    await expect(page.getByTestId('api-error')).toBeVisible()

    // Provide a valid URL and return the canned GitHub result (1 lock item + deep-link)
    await page.getByTestId('github-project-url').fill(
      'https://github.com/orgs/acme/projects/42',
    )
    await page.route('**/api/v1/projects/p1/planning/import*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_GITHUB_RESULT),
      })
    })
    await page.getByTestId('preview-button').click()

    // Preview result table: the lock row is present with the expected reason
    await expect(page.getByTestId('preview-result-table')).toBeVisible()
    const lockRow = page
      .getByTestId('preview-result-table')
      .locator('tr', { hasText: 'ACME-1' })
    await expect(lockRow).toContainText('lock')
    await expect(lockRow).toContainText('running — status & spec frozen')

    // The source link for the GitHub item is a proper deep-link to the issue
    const deepLink = page.getByTestId('item-source-link').first()
    await expect(deepLink).toBeVisible()
    await expect(deepLink).toHaveAttribute('href', 'https://github.com/acme/repo/issues/1')
  })

  // Live GitHub path — skipped when secrets are absent (CI stays green without tokens).
  test('live fetch: imports real items from the configured GitHub project', async ({ page }) => {
    test.skip(
      !process.env.E2E_GITHUB_TOKEN || !process.env.E2E_GH_PROJECT_URL,
      'Requires E2E_GITHUB_TOKEN and E2E_GH_PROJECT_URL environment variables',
    )

    // Do NOT mock planning/import — let the request reach the live backend.
    await page.goto('/projects/p1/board')
    await page.getByTestId('board-empty-import-button').click()
    await page.getByTestId('source-picker').getByText('GitHub Projects').click()

    await page.getByTestId('github-project-url').fill(process.env.E2E_GH_PROJECT_URL!)
    await page.getByTestId('preview-button').click()

    // Minimal check: preview result table appears (row count varies per live project)
    await expect(page.getByTestId('preview-result-table')).toBeVisible({ timeout: 30_000 })
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// SCENARIO 6 — Provenance regression
// ─────────────────────────────────────────────────────────────────────────────
test.describe('Provenance regression', () => {
  test('manual story shows "In-app" badge without deep-link (old git_provider heuristic removed)', async ({
    page,
  }) => {
    await setupCommonMocks(page)

    const manualEpic = {
      id: 'e-manual',
      project_id: 'p1',
      name: 'Manual Epic',
      status: 'backlog',
      source: 'manual',
      external_id: null,
      source_url: null,
      story_counts: { backlog: 1, running: 0, done: 0, failed: 0 },
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }
    const manualStory = {
      id: 's-manual',
      key: 'PROJ-1',
      title: 'A manually created story',
      status: 'backlog',
      source: 'manual',
      external_id: null,
      source_url: null,
      synced_at: null,
      epic_id: 'e-manual',
      project_id: 'p1',
      scope: null,
      current_stage: null,
      depends_on: [],
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    }

    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [manualEpic],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [manualStory],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/board')

    // Story card renders SourceBadge with data-source="manual" → label "In-app"
    const badge = page.locator('[data-testid="source-badge"]').first()
    await expect(badge).toBeVisible()
    await expect(badge).toHaveAttribute('data-source', 'manual')
    await expect(badge).toContainText('In-app')

    // No anchor deep-link for a manual source (source_url is null)
    await expect(page.locator('[data-testid="source-badge-link"]')).not.toBeVisible()
  })
})
