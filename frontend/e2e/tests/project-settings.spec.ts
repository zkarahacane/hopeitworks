import { test, expect } from '@playwright/test'

const mockProject = {
  id: 'p1',
  name: 'Alpha Project',
  description: 'A great project',
  owner_id: 'u1',
  created_at: '2026-02-10T10:00:00Z',
  updated_at: '2026-02-10T10:00:00Z',
}

test.describe('Project Settings Page', () => {
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

  test('displays form with project name and description from API', async ({ page }) => {
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockProject),
        })
      }
    })

    await page.goto('/projects/p1/settings')

    // Page header is visible
    await expect(page.locator('h1')).toHaveText('Project Settings')

    // Form fields are pre-filled
    await expect(page.locator('input#name')).toHaveValue('Alpha Project')
    await expect(page.locator('textarea#description')).toHaveValue('A great project')

    // Future tabs info message is visible
    await expect(
      page.getByText('Git, Agent, and Budget settings will be available in a future release.'),
    ).toBeVisible()
  })

  test('shows success toast on successful save', async ({ page }) => {
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockProject),
        })
      } else if (route.request().method() === 'PUT') {
        const body = route.request().postDataJSON()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...mockProject, ...body }),
        })
      }
    })

    await page.goto('/projects/p1/settings')

    // Wait for form to load
    await expect(page.locator('input#name')).toHaveValue('Alpha Project')

    // Edit the name
    await page.locator('input#name').fill('Updated Project')

    // Click Save
    await page.getByRole('button', { name: 'Save' }).click()

    // Success toast should appear
    await expect(page.getByText('Project settings saved')).toBeVisible()
  })

  test('shows error toast on save failure', async ({ page }) => {
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockProject),
        })
      } else if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: { code: 'INTERNAL', message: 'Server error' },
          }),
        })
      }
    })

    await page.goto('/projects/p1/settings')

    // Wait for form to load
    await expect(page.locator('input#name')).toHaveValue('Alpha Project')

    // Edit the name
    await page.locator('input#name').fill('Updated Project')

    // Click Save
    await page.getByRole('button', { name: 'Save' }).click()

    // Error toast should appear
    await expect(page.getByText('Failed to save project settings')).toBeVisible()
  })

  test('breadcrumb "Projects" link navigates to /projects', async ({ page }) => {
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockProject),
        })
      }
    })

    // Mock projects list for navigation target
    await page.route('**/api/v1/projects', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/settings')

    // Click the "Projects" breadcrumb link
    await page.getByRole('menuitem', { name: 'Projects' }).click()

    // Should navigate to /projects
    await expect(page).toHaveURL('/projects')
  })
})
