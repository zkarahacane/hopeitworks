import { test, expect } from './fixtures'

const mockEpics = [
  {
    id: 'e1',
    project_id: 'p1',
    name: 'User Authentication',
    description: 'Implement user authentication and authorization',
    status: 'in_progress',
    story_counts: { backlog: 3, running: 1, done: 5, failed: 0 },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 'e2',
    project_id: 'p1',
    name: 'Pipeline Execution',
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

    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().url().includes('/epics')) {
        return route.fallback()
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockProjects[0]),
      })
    })
  })

  test('loads the board with epics available in the epic selector', async ({ page }) => {
    // Redesign: the board no longer renders a grid of epic cards. Epics populate
    // an epic dropdown; the first epic is auto-selected and its kanban is shown.
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

    await expect(page.getByRole('heading', { name: 'Story Board' })).toBeVisible()

    // First epic auto-selected → its name shows in the epic selector
    const epicSelect = page.locator('#epic-select')
    await expect(epicSelect).toContainText('User Authentication')

    // Both epics are offered as options in the selector
    await epicSelect.click()
    await expect(page.getByRole('option', { name: 'User Authentication' })).toBeVisible()
    await expect(page.getByRole('option', { name: 'Pipeline Execution' })).toBeVisible()
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

    // Redesign: empty board prompts importing stories instead of creating an epic
    await expect(page.getByText('No epics found. Import stories to get started.')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Create Epic' })).not.toBeVisible()
  })

  // TODO(redesign): epics are no longer rendered as clickable cards on the board.
  // Selecting an epic in the dropdown loads its kanban in place (no navigation to
  // /projects/:id/epics/:epicId), so this card-click → epic-detail navigation
  // flow no longer exists.
  test.skip('navigates to epic detail when clicking an epic card', async ({ page }) => {
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

  // TODO(redesign): an epics-fetch failure no longer surfaces a "Failed to load
  // epics" banner with a Retry button on the board. The redesigned board shows an
  // inline warn Message next to the epic selector (no retry) and falls back to the
  // "No epics found" empty state, so this retry flow no longer exists here.
  test.skip('displays error message with retry button on API failure', async ({ page }) => {
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

  // TODO(redesign): the board no longer renders an admin-gated "Create Epic" CTA
  // nor a non-admin "Contact an administrator" hint. The empty state is a single
  // "No epics found. Import stories to get started." message for all roles, so the
  // role-specific empty-state copy this asserted no longer exists.
  test.skip('shows informational text for non-admin when no epics', async ({ page }) => {
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
