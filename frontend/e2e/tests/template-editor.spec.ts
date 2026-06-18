import { test, expect } from './fixtures'

const PROJECT_ID = 'p1'

const mockAgent = {
  id: 'a1',
  project_id: PROJECT_ID,
  name: 'Implement Feature',
  model: 'claude-opus-4-6',
  image: 'ghcr.io/org/agent:latest',
  template_content: 'You are working on {{story_key}}: {{story_title}}',
  scope: 'project',
  created_at: '2026-02-10T10:00:00Z',
  updated_at: '2026-02-10T10:00:00Z',
}

function setupProjectRoute(page: import('@playwright/test').Page) {
  return page.route(`**/api/v1/projects/${PROJECT_ID}`, async (route) => {
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
}

test.describe('Agent Editor', () => {
  test.describe('as admin', () => {
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

      await setupProjectRoute(page)
    })

    test('displays editor with agent content for existing agent', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents/t1`, async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(mockAgent),
          })
        } else {
          await route.fallback()
        }
      })

      await page.goto(`/projects/${PROJECT_ID}/agents/t1`)

      // Toolbar buttons visible for admin
      await expect(page.getByRole('button', { name: 'Preview' })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Save' })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()

      // Variable sidebar visible
      await expect(page.getByText('Context Variables')).toBeVisible()
      await expect(page.getByRole('button', { name: '{{story_key}} Unique story' })).toBeVisible()
    })

    test('shows empty editor for create mode', async ({ page }) => {
      await page.goto(`/projects/${PROJECT_ID}/agents/new`)

      await expect(page.getByRole('button', { name: 'Preview' })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Save' })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    })

    test('cancel navigates back to agent list', async ({ page }) => {
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

      await page.goto(`/projects/${PROJECT_ID}/agents/new`)

      await page.getByRole('button', { name: 'Cancel' }).click()

      await expect(page).toHaveURL(new RegExp(`/projects/${PROJECT_ID}/agents$`))
    })

    test('displays error state with retry when agent fetch fails', async ({ page }) => {
      let callCount = 0
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents/t1`, async (route) => {
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
            body: JSON.stringify(mockAgent),
          })
        }
      })

      await page.goto(`/projects/${PROJECT_ID}/agents/t1`)

      await expect(page.getByText('Failed to load agent')).toBeVisible()
      await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()

      await page.getByRole('button', { name: 'Retry' }).click()

      await expect(page.getByText('Context Variables')).toBeVisible()
    })

    test('preview dialog shows rendered agent output', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents/t1`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockAgent),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents/t1`)

      await expect(page.getByText('Context Variables')).toBeVisible()

      await page.getByRole('button', { name: 'Preview' }).click()

      await expect(page.getByText('Agent Preview')).toBeVisible()
      await expect(
        page.getByText('You are working on S-14: Add user authentication'),
      ).toBeVisible()

      await page.locator('.p-dialog-footer').getByRole('button', { name: 'Close' }).click()

      await expect(page.getByText('Agent Preview')).not.toBeVisible()
    })

    test('variable sidebar shows all context variables', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents/t1`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockAgent),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents/t1`)

      await expect(page.getByText('Context Variables')).toBeVisible()
      // Use button role selectors to target sidebar buttons specifically,
      // avoiding strict mode violations when Monaco editor also renders the same text
      await expect(page.getByRole('button', { name: '{{story_key}} Unique story' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{story_title}} Story title' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{story_objective}} Story objective' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{target_files}} Array of target' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{acceptance_criteria}} Story acceptance' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{error_context}} Error output' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{diff_content}} Git diff' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{branch_name}} Git branch' })).toBeVisible()
      await expect(page.getByRole('button', { name: '{{repo_url}} Git repository' })).toBeVisible()
    })
  })

  test.describe('as non-admin', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'u2',
            email: 'user@test.com',
            name: 'Regular User',
            role: 'member',
          }),
        })
      })

      await setupProjectRoute(page)
    })

    test('editor is read-only and Save button is hidden for non-admin', async ({ page }) => {
      await page.route(`**/api/v1/projects/${PROJECT_ID}/agents/t1`, async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockAgent),
        })
      })

      await page.goto(`/projects/${PROJECT_ID}/agents/t1`)

      // Preview and Cancel visible
      await expect(page.getByRole('button', { name: 'Preview' })).toBeVisible()
      await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()

      // Save not visible for non-admin
      await expect(page.getByRole('button', { name: 'Save' })).not.toBeVisible()
    })

    test('redirects non-admin from create route', async ({ page }) => {
      await page.goto(`/projects/${PROJECT_ID}/agents/new`)

      // Admin guard should redirect non-admin to dashboard
      await expect(page).toHaveURL('/')
    })
  })
})
