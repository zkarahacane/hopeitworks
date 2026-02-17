import { test, expect } from '@playwright/test'

const mockEpics = [
  {
    id: 'e1',
    project_id: 'p1',
    title: 'User Authentication',
    description: 'Implement user authentication and authorization',
    status: 'in_progress',
    story_counts: { backlog: 3, running: 1, done: 5, failed: 0 },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'e2',
    project_id: 'p1',
    title: 'Pipeline Execution',
    description: 'Build the pipeline execution engine',
    status: 'backlog',
    story_counts: { backlog: 4, running: 0, done: 0, failed: 0 },
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
]

const mockProjects = [
  {
    id: 'p1',
    name: 'Test Project',
    description: 'A test project',
    owner_id: 'u1',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
]

test.describe('Board Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          email: 'test@test.com',
          name: 'Test User',
          role: 'admin',
        }),
      })
    })

    await page.route('**/api/v1/projects?*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockProjects,
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })
  })

  test('displays epic cards with story counts', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockEpics,
          pagination: { total: 2, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/board')

    await expect(page.locator('h1')).toHaveText('Story Board')
    await expect(page.getByRole('heading', { name: 'User Authentication' })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Pipeline Execution' })).toBeVisible()

    await expect(page.getByText('Backlog')).toHaveCount(2)
    await expect(page.getByText('Running')).toHaveCount(2)
    await expect(page.getByText('Done')).toHaveCount(2)
    await expect(page.getByText('Failed')).toHaveCount(2)
  })

  test('displays empty state when no epics exist', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/board')

    await expect(page.getByText('No epics found for this project')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Create Epic' })).toBeVisible()
  })

  test('navigates to epic detail when clicking an epic card', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockEpics,
          pagination: { total: 2, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/board')

    await page.getByRole('heading', { name: 'User Authentication' }).click()

    await expect(page).toHaveURL('/projects/p1/epics/e1')
  })

  test('displays error message with retry button on API failure', async ({ page }) => {
    let callCount = 0
    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      callCount++
      if (callCount === 1) {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: { code: 'INTERNAL', message: 'Server error' },
          }),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockEpics,
            pagination: { total: 2, page: 1, per_page: 20 },
          }),
        })
      }
    })

    await page.goto('/projects/p1/board')

    await expect(page.getByText('Failed to load epics')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

    await page.getByRole('button', { name: 'Retry' }).click()

    await expect(page.getByRole('heading', { name: 'User Authentication' })).toBeVisible()
  })

  test('shows informational text for non-admin when no epics', async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          email: 'test@test.com',
          name: 'Test User',
          role: 'user',
        }),
      })
    })

    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/board')

    await expect(page.getByText('Contact an administrator')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Create Epic' })).not.toBeVisible()
  })
})
