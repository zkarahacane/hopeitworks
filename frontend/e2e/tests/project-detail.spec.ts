import { test, expect } from '@playwright/test'

const mockProject = {
  id: 'p1',
  name: 'Test Project',
  description: 'A test project for e2e testing',
  owner_id: 'u1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const mockEpics = [
  {
    id: 'e1',
    project_id: 'p1',
    name: 'Epic One',
    description: 'First epic',
    status: 'in_progress',
    story_counts: { backlog: 2, running: 1, done: 3, failed: 0 },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
]

test.describe('Project Detail — Tabbed Navigation', () => {
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

    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().url().includes('/epics')) return route.fallback()
      if (route.request().url().includes('/pipeline')) return route.fallback()
      if (route.request().url().includes('/templates')) return route.fallback()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockProject),
      })
    })

    await page.route('**/api/v1/projects/p1/epics*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockEpics,
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.route('**/api/v1/projects/p1/pipeline-configs*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })

    await page.route('**/api/v1/projects/p1/prompt-templates*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], pagination: { total: 0, page: 1, per_page: 20 } }),
      })
    })
  })

  test('shows project name and tabs on project detail page', async ({ page }) => {
    await page.goto('/projects/p1')

    await expect(page.getByTestId('project-name')).toHaveText('Test Project')
    await expect(page.getByTestId('project-tabs')).toBeVisible()

    const tabMenu = page.getByTestId('project-tabs')
    await expect(tabMenu.getByText('Overview')).toBeVisible()
    await expect(tabMenu.getByText('Board')).toBeVisible()
    await expect(tabMenu.getByText('Pipeline')).toBeVisible()
    await expect(tabMenu.getByText('Templates')).toBeVisible()
  })

  test('Overview tab is active by default at /projects/:id', async ({ page }) => {
    await page.goto('/projects/p1')

    await expect(page.getByTestId('project-overview-card')).toBeVisible()
    await expect(page.getByText('Test Project')).toBeVisible()
    await expect(page.getByText('Jan 1, 2026')).toBeVisible()
  })

  test('clicking Board tab navigates to board sub-page', async ({ page }) => {
    await page.goto('/projects/p1')

    await page.getByTestId('project-tabs').getByText('Board').click()

    await expect(page).toHaveURL('/projects/p1/board')
    await expect(page.getByRole('heading', { name: 'Story Board' })).toBeVisible()
  })

  test('direct navigation to /projects/:id/board shows Board tab active', async ({ page }) => {
    await page.goto('/projects/p1/board')

    await expect(page.getByTestId('project-name')).toHaveText('Test Project')
    await expect(page.getByRole('heading', { name: 'Story Board' })).toBeVisible()
  })

  test('clicking Pipeline tab navigates to pipeline sub-page', async ({ page }) => {
    await page.goto('/projects/p1')

    await page.getByTestId('project-tabs').getByText('Pipeline').click()

    await expect(page).toHaveURL('/projects/p1/pipeline')
    await expect(page.getByRole('heading', { name: 'Pipeline Configuration' })).toBeVisible()
  })

  test('clicking Templates tab navigates to templates sub-page', async ({ page }) => {
    await page.goto('/projects/p1')

    await page.getByTestId('project-tabs').getByText('Templates').click()

    await expect(page).toHaveURL('/projects/p1/templates')
    await expect(page.getByRole('heading', { name: 'Prompt Templates' })).toBeVisible()
  })

  test('back button navigates to projects list', async ({ page }) => {
    await page.route('**/api/v1/projects', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [mockProject],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1')
    await page.getByTestId('back-to-projects').click()

    await expect(page).toHaveURL('/projects')
  })

  test('shows error message when project fetch fails', async ({ page }) => {
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().url().includes('/epics')) return route.fallback()
      if (route.request().url().includes('/pipeline')) return route.fallback()
      if (route.request().url().includes('/templates')) return route.fallback()
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({
          error: { code: 'NOT_FOUND', message: 'Project not found' },
        }),
      })
    })

    await page.goto('/projects/p1')

    await expect(page.getByTestId('project-error')).toBeVisible()
    await expect(page.getByText('Failed to load project')).toBeVisible()
  })
})
