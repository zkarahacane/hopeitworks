import { test, expect } from './fixtures'

const mockProject = {
  id: 'p1',
  name: 'Test Project',
  description: 'A test project for e2e testing',
  repo_url: 'https://github.com/org/repo',
  git_provider: 'github',
  agent_runtime: 'docker',
  owner_id: 'u1',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

const otherProject = {
  id: 'p2',
  name: 'Surviving Project',
  owner_id: 'u1',
  created_at: '2026-01-02T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
}

/** Register the admin + project + list mocks shared by every test. */
async function setupCommonRoutes(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: '1',
        email: 'admin@test.com',
        name: 'Admin User',
        role: 'admin',
      }),
    })
  })

  await page.route('**/api/v1/projects/p1', async (route) => {
    if (route.request().url().includes('/epics')) return route.fallback()
    if (route.request().url().includes('/pipeline')) return route.fallback()
    if (route.request().url().includes('/agents')) return route.fallback()
    if (route.request().method() === 'DELETE') return route.fallback()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(mockProject),
    })
  })
}

test.describe('Project delete from Settings danger zone', () => {
  test.beforeEach(async ({ page }) => {
    await setupCommonRoutes(page)
  })

  test('admin deletes a project: retype name → confirm → redirect to list without it', async ({
    page,
  }) => {
    let deleteCalled = false

    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'DELETE') {
        deleteCalled = true
        await route.fulfill({ status: 204, body: '' })
        return
      }
      await route.fallback()
    })

    // List shown after redirect — the deleted project is gone.
    await page.route('**/api/v1/projects?*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [otherProject],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.goto('/projects/p1/settings')

    // Danger zone is visible for admins (RG1/RG6).
    await expect(page.getByTestId('project-danger-zone')).toBeVisible()
    await page.getByTestId('open-delete-dialog-btn').click()

    // Cascade warning is explicit (RG3).
    const warning = page.getByTestId('delete-cascade-warning')
    await expect(warning).toBeVisible()
    await expect(warning).toContainText('runs')
    await expect(warning).toContainText('stories')

    // Confirm stays disabled until the typed name matches exactly (RG2).
    const confirmBtn = page.getByTestId('delete-confirm-btn')
    await expect(confirmBtn).toBeDisabled()
    await page.getByTestId('delete-confirm-input').fill('Wrong Name')
    await expect(confirmBtn).toBeDisabled()
    await page.getByTestId('delete-confirm-input').fill('Test Project')
    await expect(confirmBtn).toBeEnabled()

    await confirmBtn.click()

    // Success → redirect to /projects, deleted project absent (RG4).
    await expect(page).toHaveURL('/projects')
    expect(deleteCalled).toBe(true)
    await expect(page.getByText('Surviving Project')).toBeVisible()
    await expect(page.getByText('Test Project')).toHaveCount(0)
  })

  test('cancelling the confirmation never deletes and stays on Settings (RG7)', async ({
    page,
  }) => {
    let deleteCalled = false
    await page.route('**/api/v1/projects/p1', async (route) => {
      if (route.request().method() === 'DELETE') {
        deleteCalled = true
        await route.fulfill({ status: 204, body: '' })
        return
      }
      await route.fallback()
    })

    await page.goto('/projects/p1/settings')

    await page.getByTestId('open-delete-dialog-btn').click()
    await expect(page.getByTestId('delete-confirm-btn')).toBeVisible()
    await page.getByTestId('delete-cancel-btn').click()

    await expect(page).toHaveURL('/projects/p1/settings')
    expect(deleteCalled).toBe(false)
  })
})
