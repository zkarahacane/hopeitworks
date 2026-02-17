import { test, expect } from '@playwright/test'

const PROJECT_ID = 'p1'

const mockTemplates = [
  {
    id: 't1',
    project_id: PROJECT_ID,
    name: 'Implement Feature',
    template_content: 'You are a developer...',
    type: 'implement',
    created_at: '2026-02-10T10:00:00Z',
    updated_at: '2026-02-10T10:00:00Z',
  },
  {
    id: 't2',
    project_id: PROJECT_ID,
    name: 'Code Review',
    template_content: 'You are a code reviewer...',
    type: 'review',
    created_at: '2026-02-11T10:00:00Z',
    updated_at: '2026-02-12T10:00:00Z',
  },
  {
    id: 't3',
    project_id: PROJECT_ID,
    name: 'Merge Strategy',
    template_content: 'You are a merge specialist...',
    type: 'merge',
    created_at: '2026-02-13T10:00:00Z',
    updated_at: '2026-02-14T10:00:00Z',
  },
]

test.describe('Prompt Template List Page', () => {
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
    })

    test('displays template list in DataTable when API returns templates', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(page.locator('h1')).toHaveText('Prompt Templates')
      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByText('Code Review')).toBeVisible()
      await expect(page.getByText('Merge Strategy')).toBeVisible()

      await expect(page.getByText('Name')).toBeVisible()
      await expect(page.getByText('Type')).toBeVisible()
      await expect(page.getByText('Last Updated')).toBeVisible()
    })

    test('does not show Create Template button for non-admin user', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Create Template' })).not.toBeVisible()
    })

    test('displays empty state when API returns no templates', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(
        page.getByText('No prompt templates found for this project.'),
      ).toBeVisible()
      await expect(page.getByRole('button', { name: 'Create Template' })).not.toBeVisible()
    })

    test('displays error state with retry button on API failure', async ({ page }) => {
      let callCount = 0
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
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
              data: mockTemplates,
              pagination: { total: 3, page: 1, per_page: 20 },
            }),
          })
        }
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(page.getByText('Failed to load templates')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

      await page.getByRole('button', { name: 'Retry' }).click()

      await expect(page.getByText('Implement Feature')).toBeVisible()
    })

    test('navigates to template detail when clicking a row', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await page.getByText('Implement Feature').click()

      await expect(page).toHaveURL(`/projects/${PROJECT_ID}/templates/t1`)
    })

    test('filters templates by type', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByText('Code Review')).toBeVisible()
      await expect(page.getByText('Merge Strategy')).toBeVisible()

      await page.locator('#type-filter').click()
      await page.getByText('Review', { exact: true }).click()

      await expect(page.getByText('Code Review')).toBeVisible()
      await expect(page.getByText('Implement Feature')).not.toBeVisible()
      await expect(page.getByText('Merge Strategy')).not.toBeVisible()
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
    })

    test('shows Create Template button for admin user', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(page.getByText('Implement Feature')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Create Template' })).toBeVisible()
    })

    test('navigates to create page when clicking Create Template', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: mockTemplates,
            pagination: { total: 3, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await page.getByRole('button', { name: 'Create Template' }).click()

      await expect(page).toHaveURL(`/projects/${PROJECT_ID}/templates/new`)
    })

    test('shows Create Template CTA in empty state for admin', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/templates*`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/templates`)

      await expect(
        page.getByText('No prompt templates found for this project.'),
      ).toBeVisible()
      await expect(page.getByRole('button', { name: 'Create Template' })).toBeVisible()
    })
  })
})
