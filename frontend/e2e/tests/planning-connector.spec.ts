/**
 * E2E spec — Planning connector write-back (GitHub Projects v2).
 *
 * All network calls to the planning/connector endpoints are mocked via
 * page.route() — the live tracker probe runs server-side and cannot be
 * intercepted by Playwright. The test suite is therefore backend-independent
 * and runs in CI without secrets.
 *
 * Endpoints mocked:
 *   GET    /api/v1/projects/:id/planning/connector                → PlanningConnector | 404
 *   PUT    /api/v1/projects/:id/planning/connector                → PlanningConnector | 422
 *   GET    /api/v1/projects/:id/planning/connector/status-options → PlanningStatusOptions
 *   GET    /api/v1/projects/:id/git-connection                    → GitConnectionStatus
 *   GET    /api/v1/projects/:projectId/stories/:storyId           → Story
 *   GET    /api/v1/projects/:id/pipeline                         → PipelineConfig stub
 *
 * Route registration order: fixture catch-all first → setupCommonMocks() →
 * setupConnectorMock() last (last registered wins in Playwright).
 */
import { test, expect } from './fixtures'

// ── Shared mock payloads ──────────────────────────────────────────────────────

const MOCK_ADMIN = {
  id: 'u1',
  email: 'admin@hopeitworks.dev',
  name: 'Admin',
  role: 'admin',
}

const MOCK_PROJECT = {
  id: 'p1',
  name: 'Connector Test Project',
  description: 'A project for planning connector tests',
  repo_url: 'https://github.com/acme/repo',
  git_provider: 'github',
  owner_id: 'u1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

/** GitConnectionStatus — PAT stored and validated (needed so the no-git warning is suppressed). */
const GIT_STATUS_CONNECTED = {
  configured: true,
  source: 'connection',
  kind: 'pat',
  provider: 'github',
  status: 'connected',
  secret_last4: 'abcd',
  token_type: 'fine_grained',
  account_login: 'octocat',
  scopes: [],
  expires_at: null,
  last_validated_at: '2026-06-29T10:00:00Z',
  validation_error: null,
}

/** PlanningConnector — not configured yet (404 body, for the empty-state test). */
// GET returns 404 → composable maps to null; we fulfill with 404.

/** PlanningConnector — fully configured. */
const CONNECTOR_CONFIGURED = {
  project_id: 'p1',
  source: 'github_projects',
  project_url: 'https://github.com/orgs/acme/projects/42',
  status_field: 'Status',
  done_options: ['Done', 'Closed'],
  epic_issue_type: 'Epic',
  status_mapping: {
    backlog: 'opt-backlog-id',
    running: 'opt-progress-id',
    done: 'opt-done-id',
    failed: null,
  },
  writeback_enabled: true,
  post_run_comment: true,
}

/** PlanningStatusOptions — live-probed from the GitHub board. */
const STATUS_OPTIONS = {
  field_id: 'PVTSSF_lADOAcmeBoardField',
  field_name: 'Status',
  options: [
    { id: 'opt-backlog-id', name: 'Backlog' },
    { id: 'opt-progress-id', name: 'In Progress' },
    { id: 'opt-done-id', name: 'Done' },
    { id: 'opt-blocked-id', name: 'Blocked' },
  ],
}

/** Story with writeback_status = 'synced'. */
const STORY_SYNCED = {
  id: 's1',
  key: 'AUTH-1',
  title: 'Implement JWT authentication',
  status: 'done',
  writeback_status: 'synced',
  source: 'github_projects',
  source_url: 'https://github.com/acme/repo/issues/1',
  synced_at: '2026-06-29T10:00:00Z',
  epic_id: 'e1',
  project_id: 'p1',
  scope: 'backend',
  current_stage: null,
  depends_on: [],
  created_at: '2026-06-29T00:00:00Z',
  updated_at: '2026-06-29T00:00:00Z',
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type Page = import('@playwright/test').Page

/**
 * Register auth/me and project mocks shared by every test.
 * Sub-paths (git-connection, planning, pipeline, epics, stories) fall through
 * to the more specific mocks registered afterwards (last registered wins).
 */
async function setupCommonMocks(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_ADMIN),
    })
  })

  await page.route('**/api/v1/projects/p1', async (route) => {
    const url = route.request().url()
    if (
      url.includes('/git-connection') ||
      url.includes('/planning') ||
      url.includes('/pipeline') ||
      url.includes('/epics') ||
      url.includes('/stories') ||
      url.includes('/agents') ||
      url.includes('/notifications') ||
      url.includes('/prompt-templates') ||
      url.includes('/environments')
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

/**
 * Register planning/connector route handlers.
 * Must be registered AFTER setupCommonMocks() so it wins for connector URLs.
 *
 * @param opts.getConnector      Response for the initial GET (null → 404)
 * @param opts.putConnector      Response for PUT (defaults to CONNECTOR_CONFIGURED)
 * @param opts.putStatus         HTTP status for PUT (defaults to 200)
 * @param opts.putErrorBody      Error body if putStatus is 4xx
 * @param opts.statusOptions     Response for GET status-options
 * @param opts.gitConnected      Whether git-connection should appear connected (default true)
 */
async function setupConnectorMock(
  page: Page,
  opts: {
    getConnector?: typeof CONNECTOR_CONFIGURED | null
    putConnector?: typeof CONNECTOR_CONFIGURED
    putStatus?: number
    putErrorBody?: unknown
    statusOptions?: typeof STATUS_OPTIONS
    gitConnected?: boolean
  } = {},
) {
  const {
    getConnector = null,
    putConnector = CONNECTOR_CONFIGURED,
    putStatus = 200,
    putErrorBody = null,
    statusOptions = STATUS_OPTIONS,
    gitConnected = true,
  } = opts

  // Git-connection status (used by PlanningConnectorCard to show/hide the no-git warning).
  await page.route('**/api/v1/projects/p1/git-connection', async (route) => {
    const url = route.request().url()
    if (url.includes('/test')) return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(
        gitConnected
          ? GIT_STATUS_CONNECTED
          : { ...GIT_STATUS_CONNECTED, configured: false, status: 'unconfigured' },
      ),
    })
  })

  // Connector endpoints (GET / PUT / status-options).
  await page.route('**/api/v1/projects/p1/planning/connector*', async (route) => {
    const url = route.request().url()
    const method = route.request().method()

    // GET status-options
    if (url.includes('/status-options')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(statusOptions),
      })
      return
    }

    // PUT /planning/connector
    if (method === 'PUT') {
      if (putStatus !== 200) {
        await route.fulfill({
          status: putStatus,
          contentType: 'application/json',
          body: JSON.stringify(putErrorBody),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(putConnector),
        })
      }
      return
    }

    // GET /planning/connector
    if (getConnector === null) {
      await route.fulfill({ status: 404, contentType: 'application/json', body: '{}' })
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(getConnector),
      })
    }
  })
}

// ─────────────────────────────────────────────────────────────────────────────
// PC1 — Settings section visible + Load options populates mapping selects
// ─────────────────────────────────────────────────────────────────────────────
test.describe('PC1 — Settings: Tracker & sync section visible; Load options', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
  })

  test('no connector yet → empty-state message; enter URL + Load options → selects appear', async ({
    page,
  }) => {
    await setupConnectorMock(page, { getConnector: null })

    await page.goto('/projects/p1/settings')

    // Card is rendered.
    await expect(page.getByTestId('planning-connector-card')).toBeVisible()

    // Empty-state info message is shown when no connector is configured.
    await expect(page.getByTestId('planning-connector-empty')).toBeVisible()

    // No git-warning (git is connected in this mock).
    await expect(page.getByTestId('planning-connector-no-git')).not.toBeVisible()

    // Fill in a project URL.
    await page.getByTestId('pc-project-url').fill('https://github.com/orgs/acme/projects/42')

    // "Load options" becomes enabled (URL is non-empty).
    const loadBtn = page.getByTestId('pc-load-options')
    await expect(loadBtn).toBeEnabled()

    // Placeholder text is visible before loading.
    await expect(page.getByTestId('pc-options-hint')).toBeVisible()

    // Click "Load options" — status-options endpoint is mocked.
    await loadBtn.click()

    // Mapping selects appear once options are loaded.
    await expect(page.getByTestId('pc-map-backlog')).toBeVisible()
    await expect(page.getByTestId('pc-map-running')).toBeVisible()
    await expect(page.getByTestId('pc-map-done')).toBeVisible()
    await expect(page.getByTestId('pc-map-failed')).toBeVisible()

    // Placeholder hint disappears (replaced by the mapping selects).
    await expect(page.getByTestId('pc-options-hint')).not.toBeVisible()

    // "Auto-fill" button becomes enabled once options are loaded.
    await expect(page.getByTestId('pc-auto-fill')).toBeEnabled()
  })

  test('no git connection → no-git warning is shown below the empty-state hint', async ({
    page,
  }) => {
    await setupConnectorMock(page, { getConnector: null, gitConnected: false })

    await page.goto('/projects/p1/settings')

    await expect(page.getByTestId('planning-connector-card')).toBeVisible()
    // The no-git warning banner is rendered (write-back prerequisite not met).
    await expect(page.getByTestId('planning-connector-no-git')).toBeVisible()
    await expect(page.getByTestId('planning-connector-no-git')).toContainText(
      'Write-back will not work',
    )
  })

  test('connector already configured → form pre-filled with persisted values', async ({
    page,
  }) => {
    await setupConnectorMock(page, { getConnector: CONNECTOR_CONFIGURED })

    await page.goto('/projects/p1/settings')

    await expect(page.getByTestId('planning-connector-card')).toBeVisible()

    // Empty-state message must NOT appear when a connector exists.
    await expect(page.getByTestId('planning-connector-empty')).not.toBeVisible()

    // Project URL is pre-filled with the persisted value.
    await expect(page.getByTestId('pc-project-url')).toHaveValue(
      'https://github.com/orgs/acme/projects/42',
    )
    // Status field is pre-filled.
    await expect(page.getByTestId('pc-status-field')).toHaveValue('Status')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// PC2 — Auto-fill + Save → success toast
// ─────────────────────────────────────────────────────────────────────────────
test.describe('PC2 — Auto-fill by convention then Save', () => {
  test('clicking Auto-fill then Save → PUT returns connector → success toast shown', async ({
    page,
  }) => {
    await setupCommonMocks(page)
    await setupConnectorMock(page, { getConnector: null, putConnector: CONNECTOR_CONFIGURED })

    await page.goto('/projects/p1/settings')
    await expect(page.getByTestId('planning-connector-card')).toBeVisible()

    // Enter the GitHub Projects URL.
    await page.getByTestId('pc-project-url').fill('https://github.com/orgs/acme/projects/42')

    // Load options to populate the mapping selects.
    await page.getByTestId('pc-load-options').click()
    await expect(page.getByTestId('pc-map-done')).toBeVisible()

    // Auto-fill mapping by convention: should populate matching options by keyword.
    const autoFillBtn = page.getByTestId('pc-auto-fill')
    await expect(autoFillBtn).toBeEnabled()
    await autoFillBtn.click()

    // Enable write-back toggle.
    await page.getByTestId('pc-writeback-enabled').click()

    // Enable post-run-comment toggle.
    await page.getByTestId('pc-post-run-comment').click()

    // Save → PUT is mocked to return CONNECTOR_CONFIGURED.
    await page.getByTestId('pc-save').click()

    // Success toast: "Tracker connector saved".
    await expect(page.getByText('Tracker connector saved')).toBeVisible()

    // No error message.
    await expect(page.getByTestId('pc-save-error')).not.toBeVisible()
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// PC3 — PUT 422 PLANNING_CONNECTOR_NO_GIT_CONNECTION → friendly error message
// ─────────────────────────────────────────────────────────────────────────────
test.describe('PC3 — Save 422 error: no git connection', () => {
  test('PUT 422 PLANNING_CONNECTOR_NO_GIT_CONNECTION → inline error message is shown', async ({
    page,
  }) => {
    await setupCommonMocks(page)
    await setupConnectorMock(page, {
      getConnector: null,
      putStatus: 422,
      putErrorBody: {
        error: {
          code: 'PLANNING_CONNECTOR_NO_GIT_CONNECTION',
          message: 'Write-back requires a configured git connection.',
        },
      },
    })

    await page.goto('/projects/p1/settings')
    await expect(page.getByTestId('planning-connector-card')).toBeVisible()

    await page.getByTestId('pc-project-url').fill('https://github.com/orgs/acme/projects/42')

    // Click Save immediately (no mapping needed for this error path).
    await page.getByTestId('pc-save').click()

    // Inline save-error message must appear with the friendly code-aware text.
    await expect(page.getByTestId('pc-save-error')).toBeVisible()
    await expect(page.getByTestId('pc-save-error')).toContainText(
      'Write-back requires a configured git connection',
    )

    // No success toast.
    await expect(page.getByText('Tracker connector saved')).not.toBeVisible()
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// PC4 — WritebackStatusBadge in StoryDetailPanel
// ─────────────────────────────────────────────────────────────────────────────
test.describe('PC4 — WritebackStatusBadge renders in story detail', () => {
  test('story with writeback_status="synced" → WritebackStatusBadge with severity success', async ({
    page,
  }) => {
    // Auth mock — story detail view checks auth.
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_ADMIN),
      })
    })

    // Story endpoint — return a story with writeback_status: 'synced'.
    await page.route('**/api/v1/projects/p1/stories/s1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(STORY_SYNCED),
      })
    })

    await page.goto('/projects/p1/stories/s1')

    // The WritebackStatusBadge must be visible.
    const badge = page.getByTestId('writeback-status-badge')
    await expect(badge).toBeVisible()

    // The badge carries the data-writeback-status attribute set to 'synced'.
    await expect(badge).toHaveAttribute('data-writeback-status', 'synced')

    // The badge label must read "Synced".
    await expect(badge).toContainText('Synced')
  })

  test('story with writeback_status="pending" → badge shows "Sync pending"', async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_ADMIN),
      })
    })

    await page.route('**/api/v1/projects/p1/stories/s1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...STORY_SYNCED, writeback_status: 'pending' }),
      })
    })

    await page.goto('/projects/p1/stories/s1')

    const badge = page.getByTestId('writeback-status-badge')
    await expect(badge).toBeVisible()
    await expect(badge).toHaveAttribute('data-writeback-status', 'pending')
    await expect(badge).toContainText('Sync pending')
  })

  test('story with writeback_status="failed" → badge shows "Sync failed"', async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_ADMIN),
      })
    })

    await page.route('**/api/v1/projects/p1/stories/s1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...STORY_SYNCED, writeback_status: 'failed' }),
      })
    })

    await page.goto('/projects/p1/stories/s1')

    const badge = page.getByTestId('writeback-status-badge')
    await expect(badge).toBeVisible()
    await expect(badge).toHaveAttribute('data-writeback-status', 'failed')
    await expect(badge).toContainText('Sync failed')
  })

  test('story with writeback_status="disabled" → badge is NOT rendered (noise suppressed)', async ({
    page,
  }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_ADMIN),
      })
    })

    await page.route('**/api/v1/projects/p1/stories/s1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...STORY_SYNCED, writeback_status: 'disabled' }),
      })
    })

    await page.goto('/projects/p1/stories/s1')

    // WritebackStatusBadge renders nothing for 'disabled' (showDisabled: false in StoryDetailPanel).
    await expect(page.getByTestId('writeback-status-badge')).not.toBeVisible()
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// PC5 — "Tracker & sync" button in PipelineConfigView navigates to settings
// ─────────────────────────────────────────────────────────────────────────────
test.describe('PC5 — Tracker & sync shortcut in pipeline editor', () => {
  test('clicking "Tracker & sync" button navigates to project settings page', async ({ page }) => {
    await setupCommonMocks(page)

    // Mock the pipeline config so the view renders without errors.
    await page.route('**/api/v1/projects/p1/pipeline', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ groups: [] }),
      })
    })

    await page.goto('/projects/p1/pipeline')

    // "Tracker & sync" button is visible in the pipeline header toolbar.
    const trackerBtn = page.getByTestId('tracker-sync-link')
    await expect(trackerBtn).toBeVisible()
    await expect(trackerBtn).toContainText('Tracker & sync')

    // Clicking navigates to project settings (router.push({ name: 'project-settings', ... })).
    await trackerBtn.click()

    // URL should now point at the settings page.
    await expect(page).toHaveURL(/\/projects\/p1\/settings/)
  })
})
