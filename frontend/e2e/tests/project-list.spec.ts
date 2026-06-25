import { test, expect } from './fixtures'

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
          role: 'user',
        }),
      })
    })
  })

  test('displays project cards when API returns projects', async ({ page }) => {
    const projects = [
      {
        id: 'p1',
        name: 'Alpha Project',
        description: 'First project',
        git_provider: 'github',
        agent_runtime: 'docker',
        owner_id: 'u1',
        created_at: '2026-02-10T10:00:00Z',
        updated_at: '2026-02-10T10:00:00Z',
      },
      {
        id: 'p2',
        name: 'Beta Project',
        description: 'Second project',
        git_provider: 'gitea',
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

    // Project cards are visible with project names
    // (redesign: cards show name + chips + story count, not the description text)
    await expect(page.getByText('Alpha Project')).toBeVisible()
    await expect(page.getByText('Beta Project')).toBeVisible()
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

  test('displays the real story count per card, with singular/plural and empty states (#289)', async ({
    page,
  }) => {
    const projects = [
      {
        id: 'p-many',
        name: 'Many Stories',
        git_provider: 'github',
        owner_id: 'u1',
        story_count: 5,
        created_at: '2026-02-10T10:00:00Z',
        updated_at: '2026-02-10T10:00:00Z',
      },
      {
        id: 'p-one',
        name: 'One Story',
        git_provider: 'github',
        owner_id: 'u1',
        story_count: 1,
        created_at: '2026-02-11T10:00:00Z',
        updated_at: '2026-02-11T10:00:00Z',
      },
      {
        id: 'p-none',
        name: 'Empty Project',
        git_provider: 'github',
        owner_id: 'u1',
        story_count: 0,
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
          pagination: { total: 3, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects')

    // RG1: a project with 5 stories shows "5 stories".
    await expect(page.getByText('5 stories')).toBeVisible()
    // RG3: a project with exactly 1 story uses the singular "1 story".
    await expect(page.getByText('1 story', { exact: true })).toBeVisible()
    // RG2: a project with no stories shows the distinct "no stories" state.
    await expect(page.getByText('no stories', { exact: true })).toBeVisible()
  })

  test('navigates to project detail when clicking a card', async ({ page }) => {
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

    // Click the project card
    await page.getByText('Alpha Project').click()

    // Should navigate to project detail
    await expect(page).toHaveURL('/projects/p1')
  })
})
