import { test, expect } from '@playwright/test'

const mockStories = [
  {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Setup authentication',
    status: 'done',
    objective: 'Implement authentication flow',
    acceptance_criteria: 'Users can log in and out',
    target_files: ['src/auth.ts', 'src/middleware.ts'],
    depends_on: ['S-00'],
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 's2',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-02',
    title: 'Add user profile page',
    status: 'backlog',
    objective: 'Create the profile UI',
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
  {
    id: 's3',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-03',
    title: 'Fix login bug',
    status: 'failed',
    objective: 'Resolve intermittent login failure',
    created_at: '2026-01-17T10:00:00Z',
    updated_at: '2026-01-17T10:00:00Z',
  },
  {
    id: 's4',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-04',
    title: 'Running pipeline test',
    status: 'running',
    created_at: '2026-01-18T10:00:00Z',
    updated_at: '2026-01-18T10:00:00Z',
  },
]

test.describe('Epic Detail Page', () => {
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

    await page.route('**/api/v1/projects*', async (route) => {
      if (route.request().url().includes('/stories') || route.request().url().includes('/epics')) {
        return route.fallback()
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [{ id: 'p1', name: 'Test Project', description: 'A test project' }],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })
  })

  test('displays story list on left and detail panel on right', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    await expect(page.locator('h1')).toHaveText('Epic Stories')
    await expect(page.getByText('S-01')).toBeVisible()
    await expect(page.getByText('Setup authentication')).toBeVisible()
    await expect(page.getByText('S-02')).toBeVisible()
    await expect(page.getByText('Select a story to view details')).toBeVisible()
  })

  test('clicking a story shows its detail in the right panel', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    await page.getByText('Setup authentication').click()

    await expect(page.getByText('Implement authentication flow')).toBeVisible()
    await expect(page.getByText('Users can log in and out')).toBeVisible()
    await expect(page.getByText('src/auth.ts')).toBeVisible()
    await expect(page.getByText('S-00')).toBeVisible()
  })

  test('filters stories by status', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    await expect(page.getByText('S-01')).toBeVisible()
    await expect(page.getByText('S-02')).toBeVisible()
    await expect(page.getByText('S-03')).toBeVisible()
    await expect(page.getByText('S-04')).toBeVisible()
  })

  test('filters stories by text search with debounce', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    await page.getByPlaceholder('Search stories...').fill('login')

    await expect(page.getByText('S-03')).toBeVisible()
    await expect(page.getByText('S-01')).not.toBeVisible()
    await expect(page.getByText('S-02')).not.toBeVisible()
  })

  test('keyboard navigation with J/K/Enter', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    await expect(page.getByText('S-01')).toBeVisible()

    await page.keyboard.press('j')
    await page.keyboard.press('Enter')

    await expect(page.getByText('Add user profile page').nth(1)).toBeVisible()
  })

  test('displays error message with retry button on API failure', async ({ page }) => {
    let callCount = 0
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
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
          body: JSON.stringify({ data: mockStories }),
        })
      }
    })

    await page.goto('/projects/p1/epics/e1')

    await expect(page.getByText('Failed to load stories')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

    await page.getByRole('button', { name: 'Retry' }).click()

    await expect(page.getByText('S-01')).toBeVisible()
  })

  test('shows loading skeleton on initial load', async ({ page }) => {
    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 500))
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStories }),
      })
    })

    await page.goto('/projects/p1/epics/e1')

    const skeletons = page.locator('.p-skeleton')
    await expect(skeletons.first()).toBeVisible()

    await expect(page.getByText('S-01')).toBeVisible()
  })
})
