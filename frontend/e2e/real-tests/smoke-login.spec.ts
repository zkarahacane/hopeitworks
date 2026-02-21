/**
 * Real E2E smoke tests — Authentication
 *
 * These tests run against a live backend at http://localhost:5173 (Vite proxy → :8080).
 * They require seed data to be present (admin/dev/alice users).
 *
 * Run with: npx playwright test real-tests/smoke-login.spec.ts
 */
import { test, expect } from '@playwright/test'
import { loginViaUI, loginViaAPI, SEED_USERS } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

test.describe('Auth smoke tests (real backend)', () => {
  let logs: LogCollector

  test.beforeEach(({ page }) => {
    logs = new LogCollector()
    logs.attach(page)
  })

  test.afterEach(() => {
    const report = logs.getReport()
    if (report.summary.totalErrors > 0) {
      console.warn('[LogCollector] Console/JS errors:', report.errors)
    }
    if (report.summary.totalWarnings > 0) {
      console.warn('[LogCollector] Warnings:', report.warnings)
    }
  })

  test('backend health check — /auth/me returns 401 when unauthenticated', async ({
    request,
  }) => {
    // We are NOT logged in, so the backend should return 401.
    // This just verifies the backend is reachable and responding correctly.
    const response = await request.get('/api/v1/auth/me')
    expect(
      response.status(),
      'Expected 401 Unauthorized when no session cookie is set',
    ).toBe(401)
  })

  test('login with admin credentials', async ({ page }) => {
    await loginViaUI(page, 'admin')

    // After login we should be redirected away from /login
    await expect(page).not.toHaveURL(/\/login/)

    // The dashboard or some authenticated page should render user-related content.
    // We accept any of: the user's name, a nav element, or the main landmark.
    await expect(
      page.getByText(SEED_USERS.admin.name).or(page.getByRole('navigation')).first(),
    ).toBeVisible({ timeout: 8000 })
  })

  test('login with dev credentials', async ({ page }) => {
    await loginViaUI(page, 'dev')

    await expect(page).not.toHaveURL(/\/login/)

    await expect(
      page.getByText(SEED_USERS.dev.name).or(page.getByRole('navigation')).first(),
    ).toBeVisible({ timeout: 8000 })
  })

  test('wrong password shows error message', async ({ page }) => {
    await page.goto('/login')

    await page.getByLabel(/email/i).fill(SEED_USERS.admin.email)

    // PrimeVue Password wraps the input — fall back to CSS selector if getByLabel fails
    const pwByLabel = page.getByLabel(/password/i)
    const pwBySelector = page.locator('input[type="password"]')
    const pwField = (await pwByLabel.count()) > 0 ? pwByLabel : pwBySelector
    await pwField.fill('this-is-the-wrong-password')

    await page.getByRole('button', { name: /sign in|log in|login/i }).click()

    // The form should remain on /login and show an error
    await expect(page).toHaveURL(/\/login/)

    // Error can appear as an alert role, a .error element, or any visible error text.
    // We use a broad OR to be resilient to slight UI variations.
    const errorLocator = page
      .getByRole('alert')
      .or(page.getByText(/invalid|incorrect|credentials|unauthorized/i))
      .first()

    await expect(errorLocator).toBeVisible({ timeout: 6000 })
  })

  test('auth guard redirects unauthenticated user to /login', async ({ page }) => {
    // Attempt to access a protected route without any session
    await page.goto('/projects')

    // The router guard should redirect to /login (with or without a redirect param)
    await expect(page).toHaveURL(/\/login/, { timeout: 6000 })
  })

  test('logout clears session and redirects to /login', async ({ page }) => {
    await loginViaUI(page, 'dev')

    // Wait until we are past the login page
    await expect(page).not.toHaveURL(/\/login/)

    // The logout is inside a PrimeVue popup Menu triggered by the "User menu" button.
    // Click the user menu button to open the popup.
    await page.getByTestId('user-menu-button').click()

    // PrimeVue Menu renders items as <a role="menuitem"> — click the Logout item
    await page.getByRole('menuitem', { name: /logout/i }).click()

    await expect(page).toHaveURL(/\/login/, { timeout: 8000 })
  })

  /**
   * KNOWN BUG AUDIT: /api/v1/auth/me does NOT return a `role` field.
   *
   * Root cause: `userResponse` struct in `auth_handler.go` is missing the `Role` field.
   * As a result the frontend falls back to `json.role ?? 'member'`, meaning ALL users —
   * including admin — are displayed/treated as "member" in the UI.
   *
   * This test DOCUMENTS the bug by asserting that `role` is absent from the response.
   * It should be updated to assert `role` IS present once the bug is fixed.
   */
  test('AUDIT: /auth/me does not return role field (known bug)', async ({
    context,
  }) => {
    test.info().annotations.push({
      type: 'known-bug',
      description:
        'auth_handler.go userResponse struct is missing the Role field. ' +
        'All users appear as "member" in the frontend because it falls back to ' +
        '`json.role ?? "member"`. Fix: add Role to userResponse and populate it ' +
        'from the user entity.',
    })

    // Login via API to get the session cookie set on the browser context
    await loginViaAPI(context, 'admin')

    // Use context.request so the session cookie set by loginViaAPI is applied.
    // The standalone `request` fixture has a separate cookie jar and won't be authenticated.
    const response = await context.request.get('/api/v1/auth/me')
    expect(response.status(), '/auth/me should return 200 for authenticated user').toBe(200)

    const body = await response.json()

    // BUG ASSERTION: role field should be absent from the response.
    // When this bug is fixed, remove the `toBeUndefined()` assertion and replace with:
    //   expect(body.role).toBe('admin')
    expect(
      body.role,
      'BUG: /auth/me is missing the `role` field — all users fall back to "member" in the UI',
    ).toBeUndefined()

    // Verify other expected fields ARE present so we know the response is otherwise valid
    expect(body.id, 'Response should include id').toBeDefined()
    expect(body.email, 'Response should include email').toBeDefined()
    expect(body.name, 'Response should include name').toBeDefined()
  })
})
