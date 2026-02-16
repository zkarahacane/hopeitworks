import { test, expect } from '@playwright/test'

test.describe('Project List Page', () => {
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

  test('displays project list in DataTable when API returns projects', async ({ page }) => {
    const projects = [
      {
        id: 'p1',
        name: 'Alpha Project',
        description: 'First project',
        owner_id: 'u1',
        created_at: '2026-02-10T10:00:00Z',
        updated_at: '2026-02-10T10:00:00Z',
      },
      {
        id: 'p2',
        name: 'Beta Project',
        description: 'Second project',
        owner_id: 'u1',
        created_at: '2026-02-12T10:00:00Z',
        updated_at: '2026-02-12T10:00:00Z',
      },
    ]

    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: projects,
          pagination: { total: 2, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects')

    // Page header is visible
    await expect(page.locator('h1')).toHaveText('Projects')

    // DataTable rows are visible with project names
    await expect(page.getByText('Alpha Project')).toBeVisible()
    await expect(page.getByText('Beta Project')).toBeVisible()

    // Column headers are visible
    await expect(page.getByText('Name')).toBeVisible()
    await expect(page.getByText('Description')).toBeVisible()
    await expect(page.getByText('Created')).toBeVisible()
  })

  test('displays empty state when API returns no projects', async ({ page }) => {
    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects')

    // Empty state is visible
    await expect(page.getByText('No projects yet')).toBeVisible()
    await expect(page.getByText('Create your first project')).toBeVisible()
  })

  test('navigates to project detail when clicking a row', async ({ page }) => {
    const projects = [
      {
        id: 'p1',
        name: 'Alpha Project',
        description: 'First project',
        owner_id: 'u1',
        created_at: '2026-02-10T10:00:00Z',
        updated_at: '2026-02-10T10:00:00Z',
      },
    ]

    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: projects,
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects')

    // Click the project row
    await page.getByText('Alpha Project').click()

    // Should navigate to project detail
    await expect(page).toHaveURL('/projects/p1')
  })
})
