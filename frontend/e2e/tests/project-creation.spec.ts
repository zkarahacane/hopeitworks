import { test, expect } from '@playwright/test'

test.describe('Project Creation', () => {
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

    // Mock empty project list by default
    await page.route('**/api/v1/projects*', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      } else {
        await route.continue()
      }
    })
  })

  test('clicking New Project button opens creation dialog', async ({ page }) => {
    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()

    // Dialog should be visible with form fields
    await expect(page.getByText('Create Project')).toBeVisible()
    await expect(page.locator('#project-name')).toBeVisible()
    await expect(page.locator('#project-description')).toBeVisible()
  })

  test('submitting empty form shows validation error on Name field', async ({ page }) => {
    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Click Create without filling the form
    await page.getByRole('button', { name: 'Create' }).click()

    // Validation error should appear
    await expect(page.getByText('Project name is required')).toBeVisible()
  })

  test('successful form submission calls API, closes dialog, and navigates', async ({ page }) => {
    const createdProject = {
      id: 'p-new-1',
      name: 'My New Project',
      description: 'A test project',
      owner_id: '1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    }

    // Mock POST /projects
    await page.route('**/api/v1/projects', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(createdProject),
        })
      } else {
        await route.continue()
      }
    })

    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Fill in the form
    await page.locator('#project-name').fill('My New Project')
    await page.locator('#project-description').fill('A test project')

    // Submit
    await page.getByRole('button', { name: 'Create' }).click()

    // Should navigate to project detail page
    await expect(page).toHaveURL('/projects/p-new-1')
  })

  test('empty state CTA opens creation dialog', async ({ page }) => {
    await page.goto('/projects')

    // Empty state should be visible
    await expect(page.getByText('No projects yet')).toBeVisible()

    // Click the CTA button
    await page.getByRole('button', { name: 'Create your first project' }).click()

    // Dialog should open
    await expect(page.getByText('Create Project')).toBeVisible()
    await expect(page.locator('#project-name')).toBeVisible()
  })
})
