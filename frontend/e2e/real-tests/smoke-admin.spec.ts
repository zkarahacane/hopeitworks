import { test, expect } from '@playwright/test'
import { loginViaAPI, SEED_USERS } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

test.describe('smoke: admin', () => {
  test('admin users page loads for admin', async ({ page, context }) => {
    const logs = new LogCollector()
    logs.attach(page)

    await loginViaAPI(context, 'admin')
    await page.goto('/admin/users')

    // The page should not redirect to login
    await expect(page).not.toHaveURL(/\/login/)

    // Some content should be visible (user list, heading, etc.)
    await expect(page.locator('body')).not.toBeEmpty()

    // No JS crashes expected
    const report = logs.getReport()
    const jsErrors = report.errors.filter((e) => e.type === 'js-error')
    expect(jsErrors).toHaveLength(0)
  })

  test('AUDIT: admin role not enforced due to missing role in API', async ({ request, page, context }) => {
    test.info().annotations.push({
      type: 'known-bug',
      description:
        'The /api/v1/auth/me response does not include a `role` field. ' +
        'As a result, the frontend cannot distinguish admin from member users based on the API response alone. ' +
        'All users default to the "member" role in the frontend auth store. ' +
        'This means any role-based UI guard relying on the API-provided role is ineffective.',
    })

    // Login as admin and inspect the /api/v1/auth/me response
    await loginViaAPI(context, 'admin')
    const meResponse = await context.request.get('/api/v1/auth/me')
    expect(meResponse.ok()).toBe(true)

    const meBody = await meResponse.json()

    // KNOWN BUG: the `role` field is absent from the auth/me response
    const hasRoleField = 'role' in meBody
    test.info().annotations.push({
      type: 'audit-result',
      description: `auth/me response has 'role' field: ${hasRoleField}. Body keys: ${Object.keys(meBody).join(', ')}`,
    })

    // Document the bug — we expect this assertion to fail once the bug is fixed
    // For now we assert the current (broken) behavior so CI catches any regression
    expect(hasRoleField).toBe(false)

    // Now check what happens when a non-admin (dev user) hits /admin/users
    const devContext = await page.context().browser()!.newContext()
    const devPage = await devContext.newPage()
    try {
      await loginViaAPI(devContext, 'dev')
      await devPage.goto('/admin/users')

      const devUrl = devPage.url()
      const devBodyText = await devPage.locator('body').innerText()

      const isBlocked =
        devUrl.includes('/login') ||
        devUrl.endsWith('/') ||
        devUrl.includes('/projects') ||
        devBodyText.toLowerCase().includes('forbidden') ||
        devBodyText.toLowerCase().includes('not authorized') ||
        devBodyText.toLowerCase().includes('access denied')

      test.info().annotations.push({
        type: 'audit-result',
        description: `Non-admin user navigating to /admin/users lands at: ${devUrl}. Blocked: ${isBlocked}`,
      })

      // Document current behavior — does not assert a specific outcome
      // because the behavior depends on the frontend guard implementation
    } finally {
      await devContext.close()
    }
  })

  test('non-admin access to admin page', async ({ page, context }) => {
    const logs = new LogCollector()
    logs.attach(page)

    // Login as a regular dev user
    await loginViaAPI(context, 'dev')
    await page.goto('/admin/users')

    test.info().annotations.push({
      type: 'known-bug',
      description:
        'Because the API does not return a `role` field in the auth response, ' +
        'the frontend auth store defaults every user to "member". ' +
        'If the /admin/users route guard checks for role === "admin", it will block even real admins. ' +
        'If the guard is absent or lenient, non-admin users can access the admin page. ' +
        'Either way, the root cause is the missing `role` in the API response.',
    })

    const currentUrl = page.url()
    const bodyText = await page.locator('body').innerText()

    // Document the actual behavior
    const wasBlocked =
      currentUrl.includes('/login') ||
      currentUrl.endsWith('/') ||
      currentUrl.includes('/projects') ||
      bodyText.toLowerCase().includes('forbidden') ||
      bodyText.toLowerCase().includes('not authorized')

    const wasAllowedThrough =
      currentUrl.includes('/admin/users') &&
      !bodyText.toLowerCase().includes('forbidden')

    test.info().annotations.push({
      type: 'audit-result',
      description:
        `Dev user (role: member) accessing /admin/users — ` +
        `blocked: ${wasBlocked}, allowed through: ${wasAllowedThrough}, ` +
        `final URL: ${currentUrl}`,
    })

    // No JS crashes should occur regardless of whether the route is guarded or not
    const report = logs.getReport()
    const jsErrors = report.errors.filter((e) => e.type === 'js-error')
    expect(jsErrors).toHaveLength(0)

    // The user should end up somewhere meaningful (not a blank crash page)
    await expect(page.locator('body')).not.toBeEmpty()
  })
})
