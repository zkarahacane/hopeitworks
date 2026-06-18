import { test, expect } from './fixtures'

test.describe('Run Launch', () => {
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

    // Mock story detail API
    await page.route('**/api/v1/projects/*/stories/*', async (route) => {
      if (route.request().url().includes('/runs')) {
        return route.fallback()
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'story-1',
          key: 'S-1',
          title: 'Test Story',
          status: 'backlog',
          description: 'A test story',
          epic_id: 'epic-1',
          project_id: 'proj-1',
        }),
      })
    })
  })

  test('shows Launch Run button on story detail page', async ({ page }) => {
    await page.goto('/projects/proj-1/stories/story-1')

    await expect(page.getByRole('button', { name: 'Launch Run' })).toBeVisible()
  })

  test('clicking Launch Run opens confirmation dialog', async ({ page }) => {
    await page.goto('/projects/proj-1/stories/story-1')

    await page.getByRole('button', { name: 'Launch Run' }).click()

    await expect(page.getByText('Launch Story Run')).toBeVisible()
    await expect(page.getByText('Claude API credits')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Confirm' })).toBeVisible()
  })

  test('confirming launch calls API and shows success toast', async ({ page }) => {
    await page.route('**/api/v1/projects/*/stories/*/runs', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 202,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'run-1',
            status: 'scheduling',
            story_id: 'story-1',
          }),
        })
      }
    })

    await page.goto('/projects/proj-1/stories/story-1')

    await page.getByRole('button', { name: 'Launch Run' }).click()
    await expect(page.getByText('Launch Story Run')).toBeVisible()

    await page.getByRole('button', { name: 'Confirm' }).click()

    // Success toast should appear
    await expect(page.getByText('Run launched')).toBeVisible()

    // Dialog should close after successful launch
    await expect(page.getByText('Launch Story Run')).not.toBeVisible()
  })

  test('409 conflict shows warning toast and keeps dialog open', async ({ page }) => {
    await page.route('**/api/v1/projects/*/stories/*/runs', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            error: {
              code: 'STORY_ALREADY_RUNNING',
              message: 'Story already has an active run',
            },
          }),
        })
      }
    })

    await page.goto('/projects/proj-1/stories/story-1')

    await page.getByRole('button', { name: 'Launch Run' }).click()
    await expect(page.getByText('Launch Story Run')).toBeVisible()

    await page.getByRole('button', { name: 'Confirm' }).click()

    // Warning toast should appear
    await expect(page.getByText('Already running')).toBeVisible()

    // Dialog should stay open
    await expect(page.getByText('Launch Story Run')).toBeVisible()
  })

  test('cancel button closes dialog without making API call', async ({ page }) => {
    let apiCalled = false
    await page.route('**/api/v1/projects/*/stories/*/runs', async (route) => {
      apiCalled = true
      await route.continue()
    })

    await page.goto('/projects/proj-1/stories/story-1')

    await page.getByRole('button', { name: 'Launch Run' }).click()
    await expect(page.getByText('Launch Story Run')).toBeVisible()

    await page.getByRole('button', { name: 'Cancel' }).click()

    // Dialog should close
    await expect(page.getByText('Launch Story Run')).not.toBeVisible()

    // No API call should have been made
    expect(apiCalled).toBe(false)
  })
})
