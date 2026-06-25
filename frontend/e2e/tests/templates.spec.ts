import { test, expect } from './fixtures'

const PROJECT_ID = 'p1'

const mockAgents = [
  {
    id: 'a1',
    project_id: PROJECT_ID,
    name: 'Implement Feature',
    model: 'claude-opus-4-6',
    image: 'ghcr.io/org/agent:latest',
    template_content: 'You are a developer...',
    scope: 'project',
    created_at: '2026-02-10T10:00:00Z',
    updated_at: '2026-02-10T10:00:00Z',
  },
  {
    id: 'a2',
    project_id: PROJECT_ID,
    name: 'Code Review',
    model: 'claude-sonnet-4-6',
    image: 'ghcr.io/org/reviewer:latest',
    template_content: 'You are a code reviewer...',
    scope: 'project',
    created_at: '2026-02-11T10:00:00Z',
    updated_at: '2026-02-12T10:00:00Z',
  },
  {
    id: 'a3',
    project_id: PROJECT_ID,
    name: 'Merge Strategy',
    model: 'claude-haiku-3-5',
    image: 'ghcr.io/org/merger:latest',
    template_content: 'You are a merge specialist...',
    scope: 'global',
    created_at: '2026-02-13T10:00:00Z',
    updated_at: '2026-02-14T10:00:00Z',
  },
]

test.describe('Agent List Page', () => {
  test.describe('as regular user', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'u1',
            email: 'user@test.com',
            name: 'Test User',
            role: 'member',
          }),
        })
      })

      await page.route(`**/api/v1/projects/${PROJECT_ID}`, async (route) => {
        if (route.request().url().includes('/agents')) return route.fallback()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: PROJECT_ID,
            name: 'Test Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })
    })

    test('displays template list in DataTable when API returns templates', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(page.getByRole('heading', { name: 'Agents' })).toBeVisible()
      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByText('Code Review')).toBeVisible()
      await expect(page.getByText('Merge Strategy')).toBeVisible()

      // Redesign: DataTable columns are Agent (name+model), Scope, Image
      await expect(page.getByRole('columnheader', { name: 'Agent' })).toBeVisible()
      await expect(page.getByRole('columnheader', { name: 'Scope' })).toBeVisible()
      await expect(page.getByRole('columnheader', { name: 'Image' })).toBeVisible()
    })

    test('does not show New Agent button for non-admin user', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByRole('button', { name: 'New Agent' })).not.toBeVisible()
    })

    test('displays empty state when API returns no templates', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(
        page.getByText('No agents found for this project.'),
      ).toBeVisible()
      await expect(page.getByRole('button', { name: 'New Agent' })).not.toBeVisible()
    })

    test('displays error state with retry button on API failure', async ({ page }) => {
      let callCount = 0
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
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
              data: mockAgents,
              pagination: { total: 3, page: 1, per_page: 20 },
            }),
          })
        }
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(page.getByText('Failed to load agents')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

      await page.getByRole('button', { name: 'Retry' }).click()

      await expect(page.getByText('Implement Feature')).toBeVisible()
    })

    test('navigates to agent editor when clicking a row', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await page.getByText('Implement Feature').click()

      await expect(page).toHaveURL(`/projects/${PROJECT_ID}/agents/a1`)
    })

    test('filters agents by scope', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByText('Code Review')).toBeVisible()
      await expect(page.getByText('Merge Strategy')).toBeVisible()

      await page.locator('#scope-filter').click()
      await page.getByText('Global', { exact: true }).click()

      await expect(page.getByText('Merge Strategy')).toBeVisible()
      await expect(page.getByText('Implement Feature')).not.toBeVisible()
      await expect(page.getByText('Code Review')).not.toBeVisible()
    })
  })

  test.describe('as admin user', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'u1',
            email: 'admin@test.com',
            name: 'Admin User',
            role: 'admin',
          }),
        })
      })

      await page.route(`**/api/v1/projects/${PROJECT_ID}`, async (route) => {
        if (route.request().url().includes('/agents')) return route.fallback()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: PROJECT_ID,
            name: 'Test Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })
    })

    test('shows New Agent button for admin user', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByRole('button', { name: 'New Agent' })).toBeVisible()
    })

    test('navigates to create page when clicking New Agent', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockAgents,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await page.getByRole('button', { name: 'New Agent' }).click()

      await expect(page).toHaveURL(`/projects/${PROJECT_ID}/agents/new`)
    })

    test('shows New Agent CTA in empty state for admin', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents`)

      await expect(
        page.getByText('No agents found for this project.'),
      ).toBeVisible()
      // In empty state the header button is hidden, so the EmptyState CTA is the
      // single "New Agent" button — getByRole no longer hits strict-mode (#304).
      const cta = page.getByTestId('empty-create-agent-button')
      await expect(cta).toBeVisible()
      await expect(page.getByRole('button', { name: 'New Agent' })).toHaveCount(1)
      await cta.click()
      await expect(page).toHaveURL(`/projects/${PROJECT_ID}/agents/new`)
    })
  })
})
