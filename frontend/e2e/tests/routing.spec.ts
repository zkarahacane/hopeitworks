import { test, expect } from '@playwright/test'

test.describe('Application Routing', () => {
  test.describe('Route rendering (authenticated)', () => {
    test.beforeEach(async ({ page }) => {
      // Mock authenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '1',
            email: 'test@test.com',
            name: 'Test User',
          }),
        })
      })
    })

    test('should render Dashboard view at /', async ({ page }) => {
      await page.goto('/')

      await expect(page.locator('h1')).toHaveText('Dashboard')
      await expect(page).toHaveURL('/')
    })

    test('should render Projects view at /projects', async ({ page }) => {
      await page.goto('/projects')

      await expect(page.locator('h1')).toHaveText('Projects')
      await expect(page).toHaveURL('/projects')
    })

    test('should render Project Detail view at /projects/123', async ({ page }) => {
      await page.goto('/projects/123')

      await expect(page.locator('h1')).toHaveText('Project Detail')
      await expect(page).toHaveURL('/projects/123')
    })

    test('should render Run Detail view at /runs/456', async ({ page }) => {
      await page.goto('/runs/456')

      await expect(page.locator('h1')).toHaveText('Run Detail')
      await expect(page).toHaveURL('/runs/456')
    })

    test('should render Approvals view at /approvals', async ({ page }) => {
      await page.goto('/approvals')

      await expect(page.locator('h1')).toHaveText('Approvals')
      await expect(page).toHaveURL('/approvals')
    })
  })

  test.describe('Navigation tests', () => {
    test.beforeEach(async ({ page }) => {
      // Mock authenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '1',
            email: 'test@test.com',
            name: 'Test User',
          }),
        })
      })
    })

    test('should navigate between routes using sidebar links (Dashboard → Projects)', async ({
      page,
    }) => {
      // Start at dashboard
      await page.goto('/')
      await expect(page.locator('h1')).toHaveText('Dashboard')

      // Navigate to Projects using sidebar button
      await page.getByRole('button', { name: 'Projects' }).click()
      await expect(page).toHaveURL('/projects')
      await expect(page.locator('h1')).toHaveText('Projects')
    })
  })

  test.describe('Auth guard integration (unauthenticated)', () => {
    test.beforeEach(async ({ page }) => {
      // Mock unauthenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ message: 'Unauthorized' }),
        })
      })
    })

    test('should redirect /projects to /login when unauthenticated', async ({ page }) => {
      await page.goto('/projects')

      // Should be redirected to login with redirect param
      await expect(page).toHaveURL('/login?redirect=/projects')
      await expect(page.locator('h1')).toHaveText('hopeitworks')
    })

    test('should redirect /approvals to /login when unauthenticated', async ({ page }) => {
      await page.goto('/approvals')

      // Should be redirected to login with redirect param
      await expect(page).toHaveURL('/login?redirect=/approvals')
      await expect(page.locator('h1')).toHaveText('hopeitworks')
    })
  })
})
