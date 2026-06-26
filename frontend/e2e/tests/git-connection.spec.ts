/**
 * E2E spec — Git connection (GitHub PAT) management.
 *
 * All network calls to the four git-connection endpoints are mocked via
 * page.route() — the outbound GitHub probe runs server-side and cannot be
 * intercepted by Playwright. The live test case is guarded behind
 * E2E_GITHUB_TOKEN; CI stays green without secrets.
 *
 * Endpoints mocked:
 *   GET    /api/v1/projects/:id/git-connection         → GitConnectionStatus
 *   PUT    /api/v1/projects/:id/git-connection         → GitConnectionStatus
 *   POST   /api/v1/projects/:id/git-connection/test    → GitConnectionTestResult
 *   DELETE /api/v1/projects/:id/git-connection         → 204
 *
 * Route-registration order: the fixture's catch-all is registered first;
 * setupCommonMocks() second; setupGitConnectionMock() last — last registered
 * wins in Playwright, so git-connection routes take precedence over everything.
 */
import { test, expect } from './fixtures'

// ── Shared mock payloads ──────────────────────────────────────────────────────

const MOCK_ADMIN = {
  id: 'u1',
  email: 'admin@hopeitworks.dev',
  name: 'Admin',
  role: 'admin',
}

/** A plain user who is NOT the project owner (owner_id = 'u1'). */
const MOCK_NON_OWNER_USER = {
  id: 'u999',
  email: 'user@hopeitworks.dev',
  name: 'Regular User',
  role: 'user',
}

const MOCK_PROJECT = {
  id: 'p1',
  name: 'Git Test Project',
  description: 'A project for git connection tests',
  repo_url: 'https://github.com/org/repo',
  git_provider: 'github',
  owner_id: 'u1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

/** GitConnectionStatus — no stored connection. */
const STATUS_UNCONFIGURED = {
  configured: false,
  source: 'none',
  kind: 'pat',
  provider: 'github',
  status: 'unconfigured',
  secret_last4: null,
  token_type: null,
  account_login: null,
  scopes: [],
  expires_at: null,
  last_validated_at: null,
  validation_error: null,
}

/** GitConnectionStatus — PAT stored and validated. */
const STATUS_CONNECTED = {
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
  last_validated_at: '2026-06-27T10:00:00Z',
  validation_error: null,
}

/** GitConnectionTestResult — probe succeeded. */
const TEST_RESULT_OK = {
  ok: true,
  status: 'connected',
  account_login: 'octocat',
  scopes: ['read:project'],
  missing_scopes: [],
  token_type: 'fine_grained',
  expires_at: null,
  message: 'Connection OK.',
}

/** GitConnectionTestResult — token lacks required scopes. */
const TEST_RESULT_INSUFFICIENT_SCOPE = {
  ok: false,
  status: 'insufficient_scope',
  account_login: 'octocat',
  scopes: ['repo'],
  missing_scopes: ['read:project'],
  token_type: 'classic',
  expires_at: null,
  message: 'Token is missing required scopes. Add read:project.',
}

// ── Helpers ───────────────────────────────────────────────────────────────────

type Page = import('@playwright/test').Page

/** Register auth/me and bare project mocks shared by every test. */
async function setupCommonMocks(page: Page, user = MOCK_ADMIN) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user),
    })
  })

  // Match the bare project URL; delegate git-connection sub-paths to the
  // more specific mock registered in setupGitConnectionMock() (last wins).
  await page.route('**/api/v1/projects/p1', async (route) => {
    const url = route.request().url()
    if (
      url.includes('/git-connection') ||
      url.includes('/epics') ||
      url.includes('/pipeline') ||
      url.includes('/agents') ||
      url.includes('/notifications') ||
      url.includes('/prompt-templates') ||
      url.includes('/planning')
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
 * Register git-connection route handlers with state tracking.
 *
 * All four verbs are handled by a single page.route() discriminated on HTTP
 * method and URL suffix. This mock is registered LAST, so it wins over both
 * setupCommonMocks() and the fixture catch-all.
 *
 * When `disconnectThenUnconfigured: true`, the GET status flips to
 * STATUS_UNCONFIGURED after the DELETE is served — simulating the server
 * state change without requiring a real backend.
 */
async function setupGitConnectionMock(
  page: Page,
  opts: {
    initialStatus?: typeof STATUS_UNCONFIGURED | typeof STATUS_CONNECTED
    putResponse?: typeof STATUS_CONNECTED
    testResponse?: typeof TEST_RESULT_OK | typeof TEST_RESULT_INSUFFICIENT_SCOPE
    disconnectThenUnconfigured?: boolean
  } = {},
) {
  const {
    initialStatus = STATUS_UNCONFIGURED,
    putResponse = STATUS_CONNECTED,
    testResponse = TEST_RESULT_OK,
    disconnectThenUnconfigured = false,
  } = opts

  let deleteCalled = false

  await page.route('**/api/v1/projects/p1/git-connection*', async (route) => {
    const url = route.request().url()
    const method = route.request().method()

    // POST .../git-connection/test
    if (url.includes('/git-connection/test')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(testResponse),
      })
      return
    }

    // PUT /git-connection (save & verify)
    if (method === 'PUT') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(putResponse),
      })
      return
    }

    // DELETE /git-connection (disconnect)
    if (method === 'DELETE') {
      deleteCalled = true
      await route.fulfill({ status: 204, body: '' })
      return
    }

    // GET /git-connection (status) — flips after DELETE when flag is set.
    const currentStatus =
      disconnectThenUnconfigured && deleteCalled ? STATUS_UNCONFIGURED : initialStatus
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(currentStatus),
    })
  })

  return { getDeleteCalled: () => deleteCalled }
}

// ─────────────────────────────────────────────────────────────────────────────
// GS1 — Admin saves a PAT → connected status with last_validated_at
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS1 — Save token → connected status', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
  })

  test('entering a PAT and saving → status connected + last validated shown', async ({ page }) => {
    await setupGitConnectionMock(page, {
      initialStatus: STATUS_UNCONFIGURED,
      putResponse: STATUS_CONNECTED,
    })

    await page.goto('/projects/p1/settings')

    // Card is visible; initial status is unconfigured.
    await expect(page.getByTestId('git-connection-card')).toBeVisible()
    await expect(page.getByTestId('git-connection-status')).toContainText('unconfigured')

    // Fill the PAT into the inner input of the PrimeVue Password component.
    // The component renders <div data-testid="git-connection-token"><input id="git-connection-token-input"></div>
    await page.locator('#git-connection-token-input').fill('github_pat_testtoken1234567890abcdef')

    // Save & verify — PUT is mocked to return STATUS_CONNECTED.
    const saveBtn = page.getByTestId('git-connection-save')
    await expect(saveBtn).toBeEnabled()
    await saveBtn.click()

    // Status tag flips to connected.
    await expect(page.getByTestId('git-connection-status')).toContainText('connected')

    // Anti-déphasage: last_validated_at is always shown next to the status.
    // The card shows "Last checked <relative date>" — must NOT read "never".
    await expect(page.getByTestId('git-connection-card')).not.toContainText('Last checked never')

    // Token hint (secret_last4) and token_type are displayed.
    await expect(page.getByTestId('git-connection-card')).toContainText('abcd')
    await expect(page.getByTestId('git-connection-card')).toContainText('fine_grained')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// GS2 — Test connection
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS2 — Test connection', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
  })

  test('testing the stored token → test-result rendered (ok = true)', async ({ page }) => {
    await setupGitConnectionMock(page, {
      initialStatus: STATUS_CONNECTED,
      testResponse: TEST_RESULT_OK,
    })

    await page.goto('/projects/p1/settings')
    await expect(page.getByTestId('git-connection-card')).toBeVisible()

    await page.getByTestId('git-connection-test').click()

    await expect(page.getByTestId('git-connection-test-result')).toBeVisible()
    await expect(page.getByTestId('git-connection-test-result')).toContainText('Connection OK.')
  })

  test('insufficient_scope → git-connection-missing-scopes warning is rendered', async ({
    page,
  }) => {
    await setupGitConnectionMock(page, {
      initialStatus: STATUS_CONNECTED,
      testResponse: TEST_RESULT_INSUFFICIENT_SCOPE,
    })

    await page.goto('/projects/p1/settings')
    await expect(page.getByTestId('git-connection-card')).toBeVisible()

    await page.getByTestId('git-connection-test').click()

    // Missing-scopes panel must appear; test-result must NOT appear.
    await expect(page.getByTestId('git-connection-missing-scopes')).toBeVisible()
    await expect(page.getByTestId('git-connection-missing-scopes')).toContainText('read:project')
    await expect(page.getByTestId('git-connection-test-result')).not.toBeVisible()
  })

  // Live GitHub probe — skipped unless E2E_GITHUB_TOKEN is set.
  // CI stays green without a secret; run locally to validate a real token.
  test('live: test with real GitHub token', async ({ page }) => {
    test.skip(
      !process.env.E2E_GITHUB_TOKEN,
      'Requires E2E_GITHUB_TOKEN environment variable. Set it locally to run the live probe.',
    )

    // No git-connection mock — let requests reach a real backend.
    await setupCommonMocks(page)
    await page.goto('/projects/p1/settings')

    // Enter the real token and click Test connection.
    await page.locator('#git-connection-token-input').fill(process.env.E2E_GITHUB_TOKEN!)
    await page.getByTestId('git-connection-test').click()

    // At minimum one of the two result panels must appear.
    const result = page.getByTestId('git-connection-test-result')
    const missing = page.getByTestId('git-connection-missing-scopes')
    await expect(result.or(missing)).toBeVisible({ timeout: 15_000 })
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// GS3 — Import-flow guard
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS3 — Import-flow guard', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonMocks(page)
    // Empty board so the empty-state import CTA is visible.
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })
  })

  test('unconfigured → GitHub Projects tab shows connection-guard with settings link', async ({
    page,
  }) => {
    await setupGitConnectionMock(page, { initialStatus: STATUS_UNCONFIGURED })

    await page.goto('/projects/p1/board')
    await page.getByTestId('board-empty-import-button').click()

    // Switch to GitHub Projects via the source-picker SelectButton.
    await page.getByTestId('source-picker').getByText('GitHub Projects').click()

    // Guard message is visible; link toward project settings is rendered.
    await expect(page.getByTestId('github-connection-guard')).toBeVisible()
    await expect(page.getByTestId('github-connection-guard-link')).toBeVisible()
  })

  test('connected → GitHub Projects tab shows no guard; URL input is accessible', async ({
    page,
  }) => {
    await setupGitConnectionMock(page, { initialStatus: STATUS_CONNECTED })

    await page.goto('/projects/p1/board')
    await page.getByTestId('board-empty-import-button').click()

    await page.getByTestId('source-picker').getByText('GitHub Projects').click()

    // Guard must be absent (v-if removes it from DOM when connected).
    await expect(page.getByTestId('github-connection-guard')).not.toBeVisible()
    // Primary controls are accessible.
    await expect(page.getByTestId('github-project-url')).toBeVisible()
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// GS4 — Disconnect
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS4 — Disconnect', () => {
  test('disconnect via ConfirmDialog → DELETE 204 → status reverts to unconfigured', async ({
    page,
  }) => {
    await setupCommonMocks(page)

    const { getDeleteCalled } = await setupGitConnectionMock(page, {
      initialStatus: STATUS_CONNECTED,
      disconnectThenUnconfigured: true,
    })

    await page.goto('/projects/p1/settings')

    // Start: status is connected, disconnect button is enabled.
    await expect(page.getByTestId('git-connection-status')).toContainText('connected')
    const disconnectBtn = page.getByTestId('git-connection-clear')
    await expect(disconnectBtn).toBeEnabled()
    await disconnectBtn.click()

    // PrimeVue ConfirmDialog appears — wait for its specific message text.
    await expect(page.getByText('Disconnect this project from its git host?')).toBeVisible()

    // Click the accept button ("Disconnect") inside the dialog.
    // The dialog role scopes the search away from the trigger button.
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: 'Disconnect' }).click()

    // The DELETE request must have been served.
    await expect.poll(() => getDeleteCalled()).toBe(true)

    // After deletion, the card re-fetches and status reverts to unconfigured.
    await expect(page.getByTestId('git-connection-status')).toContainText('unconfigured', {
      timeout: 5000,
    })
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// GS5 — Authorization: non-owner/admin sees read-only view
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS5 — Read-only view for non-owner/admin', () => {
  test('plain user who is not the project owner sees git-connection-readonly; controls hidden', async ({
    page,
  }) => {
    // Authenticate as a plain user (role: 'user', id: 'u999', owner_id of project is 'u1').
    await setupCommonMocks(page, MOCK_NON_OWNER_USER)
    // canManage = false → card skips refresh() on mount; mock is a safety net.
    await setupGitConnectionMock(page, { initialStatus: STATUS_CONNECTED })

    await page.goto('/projects/p1/settings')
    await expect(page.getByTestId('git-connection-card')).toBeVisible()

    // The read-only message is shown.
    await expect(page.getByTestId('git-connection-readonly')).toBeVisible()

    // Action controls must NOT be rendered (v-else removes them from the DOM).
    await expect(page.getByTestId('git-connection-save')).not.toBeVisible()
    await expect(page.getByTestId('git-connection-test')).not.toBeVisible()
    await expect(page.getByTestId('git-connection-clear')).not.toBeVisible()
  })

  // Skipped: the e2e harness does not yet have a mechanism to create a
  // second user account and log in as that user within a single spec. When
  // a multi-user fixture is added, this test should verify that a project
  // MEMBER (role: 'user', id !== owner_id) without admin rights is also
  // read-only, and that the project OWNER (role: 'user', id === owner_id)
  // CAN manage the connection.
  test.skip(
    'project owner (non-admin) can manage; project member cannot',
    // Requires a multi-user e2e fixture that creates real accounts and logs
    // in as different users. Track with a future e2e infrastructure ticket.
    async () => {},
  )
})

// ─────────────────────────────────────────────────────────────────────────────
// GS6 — Legacy git_token_env is demoted to an advanced disclosure
// ─────────────────────────────────────────────────────────────────────────────
test.describe('GS6 — Legacy env-var fallback is in advanced section', () => {
  test('git-token-env-advanced Panel is present; git-connection-card is the primary path', async ({
    page,
  }) => {
    await setupCommonMocks(page)
    await setupGitConnectionMock(page, { initialStatus: STATUS_UNCONFIGURED })

    await page.goto('/projects/p1/settings')

    // The git connection card (primary path) is rendered and visible.
    await expect(page.getByTestId('git-connection-card')).toBeVisible()

    // The legacy Panel is in the DOM (collapsed by default; the header is still
    // rendered and accessible). It must NOT be the primary form element.
    await expect(page.getByTestId('git-token-env-advanced')).toBeAttached()
  })
})
