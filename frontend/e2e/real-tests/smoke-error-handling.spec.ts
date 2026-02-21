import { test, expect } from '@playwright/test'
import { loginViaAPI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const NONEXISTENT_PROJECT_ID = '00000000-0000-0000-0000-999999999999'

test.describe('smoke: error handling', () => {
  test('non-existent project shows error', async ({ page, context }) => {
    const logs = new LogCollector()
    logs.attach(page)

    await loginViaAPI(context, 'admin')
    await page.goto(`/projects/${NONEXISTENT_PROJECT_ID}`)

    // The page should show a 404 or error message rather than crash silently.
    // Accept either: a visible error message, a "not found" text, or a network 404
    const bodyText = await page.locator('body').innerText()
    const has404 = bodyText.toLowerCase().includes('not found') ||
      bodyText.toLowerCase().includes('404') ||
      bodyText.toLowerCase().includes('error')

    // Alternatively the page might redirect to /projects with an error toast
    const isRedirected = page.url().includes('/projects') && !page.url().includes(NONEXISTENT_PROJECT_ID)

    const report = logs.getReport()
    const hasNetworkError = report.networkErrors.some(
      (e) => e.message.includes('404') || e.message.includes(NONEXISTENT_PROJECT_ID),
    )

    expect(has404 || isRedirected || hasNetworkError).toBe(true)
  })

  test('unauthenticated API call returns 401', async ({ context }) => {
    // Make a direct API call without any auth cookie
    const freshContext = context
    const response = await freshContext.request.get('/api/v1/projects', {
      headers: {
        // Explicitly omit any cookies — rely on a fresh context with no session
        Cookie: '',
      },
    })

    expect(response.status()).toBe(401)
  })

  test('invalid route shows fallback', async ({ page, context }) => {
    const logs = new LogCollector()
    logs.attach(page)

    await loginViaAPI(context, 'admin')
    await page.goto('/this-does-not-exist')

    // The app should either redirect (e.g. to / or /projects) or render a fallback/404 view.
    // It should not crash with an unhandled JS error.
    const report = logs.getReport()
    const jsErrors = report.errors.filter((e) => e.type === 'js-error')
    expect(jsErrors).toHaveLength(0)

    // Verify the page has some content (not a blank screen)
    await expect(page.locator('body')).not.toBeEmpty()

    // Accept a redirect to any valid route OR a fallback page containing "not found" / "404"
    const url = page.url()
    const bodyText = await page.locator('body').innerText()
    const isRedirectedToValid =
      url.endsWith('/') ||
      url.includes('/projects') ||
      url.includes('/dashboard')
    const showsFallback =
      bodyText.toLowerCase().includes('not found') ||
      bodyText.toLowerCase().includes('404') ||
      bodyText.toLowerCase().includes("page doesn't exist")

    expect(isRedirectedToValid || showsFallback).toBe(true)
  })
})
