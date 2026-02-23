import { test, expect } from '@playwright/test'

const mockSteps = [
  {
    id: 's1',
    name: 'implement',
    action_type: 'implement',
    model: 'claude-opus-4-6',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
  },
  {
    id: 's2',
    name: 'review',
    action_type: 'review',
    model: 'claude-sonnet-4-6',
    auto_approve: true,
    retry_policy: { max_retries: 1, retry_type: 'on-failure' },
  },
  {
    id: 's3',
    name: 'merge',
    action_type: 'merge',
    model: 'claude-sonnet-4-6',
    auto_approve: false,
    retry_policy: { max_retries: 0, retry_type: 'none' },
  },
]

const mockPipelineConfig = {
  project_id: 'proj-1',
  groups: [
    {
      id: 'default',
      name: 'Default',
      steps: mockSteps,
    },
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

test.describe('Pipeline Configuration Page', () => {
  test.describe('Admin user', () => {
    test.beforeEach(async ({ page }) => {
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

      await page.route('**/api/v1/projects/proj-1', async (route) => {
        if (route.request().url().includes('/pipeline')) return route.fallback()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'proj-1',
            name: 'Test Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })

      await page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
        if (route.request().method() === 'GET') {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(mockPipelineConfig),
          })
        } else if (route.request().method() === 'PUT') {
          const body = route.request().postDataJSON()
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              ...mockPipelineConfig,
              groups: body.groups,
              updated_at: new Date().toISOString(),
            }),
          })
        }
      })
    })

    test('displays pipeline steps as an ordered list', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByRole('heading', { name: 'Pipeline Configuration' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'implement' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'review' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'merge' })).toBeVisible()
    })

    test('shows admin controls: Add Step and Save buttons', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByTestId('add-step-btn')).toBeVisible()
      await expect(page.getByTestId('save-config-btn')).toBeVisible()
      await expect(page.getByTestId('save-config-btn')).toBeDisabled()
    })

    test('shows move up/down and remove buttons for each step', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      const moveUpButtons = page.getByTestId('move-up')
      const moveDownButtons = page.getByTestId('move-down')
      const removeButtons = page.getByTestId('remove-step')

      await expect(moveUpButtons).toHaveCount(3)
      await expect(moveDownButtons).toHaveCount(3)
      await expect(removeButtons).toHaveCount(3)
    })

    test('removes a step when remove button is clicked', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByTestId('remove-step')).toHaveCount(3)

      await page.getByTestId('remove-step').first().click()

      await expect(page.getByTestId('remove-step')).toHaveCount(2)
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('reorders steps when move down is clicked', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      // Click move down on first step
      await page.getByTestId('move-down').first().click()

      // Save button should be enabled since config is dirty
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('adds a new step via the dialog', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-btn').click()

      // Dialog should appear
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      // Fill form
      await page.getByTestId('step-name-input').fill('test-step')
      await page.getByTestId('add-step-submit').click()

      // Dialog should close and save button should be enabled
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('validates required step name in add dialog', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-btn').click()
      await page.getByTestId('add-step-submit').click()

      await expect(page.locator('small').getByText('Step name is required')).toBeVisible()
    })

    test('saves configuration and shows success toast', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      // Make a change to enable save
      await page.getByTestId('remove-step').first().click()

      // Click save
      await page.getByTestId('save-config-btn').click()

      // Success toast should appear
      await expect(page.getByText('Configuration saved')).toBeVisible()

      // Save button should be disabled again
      await expect(page.getByTestId('save-config-btn')).toBeDisabled()
    })
  })

  test.describe('Non-admin user', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '2',
            email: 'user@test.com',
            name: 'Regular User',
            role: 'user',
          }),
        })
      })

      await page.route('**/api/v1/projects/proj-1', async (route) => {
        if (route.request().url().includes('/pipeline')) return route.fallback()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'proj-1',
            name: 'Test Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })

      await page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockPipelineConfig),
        })
      })
    })

    test('displays pipeline steps in read-only mode', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByRole('heading', { name: 'Pipeline Configuration' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'implement' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'review' })).toBeVisible()
      await expect(page.locator('.font-semibold').filter({ hasText: 'merge' })).toBeVisible()
    })

    test('does not show admin controls', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByTestId('add-step-btn')).not.toBeVisible()
      await expect(page.getByTestId('save-config-btn')).not.toBeVisible()
      await expect(page.getByTestId('move-up')).not.toBeVisible()
      await expect(page.getByTestId('move-down')).not.toBeVisible()
      await expect(page.getByTestId('remove-step')).not.toBeVisible()
    })
  })

  test.describe('Error state', () => {
    test.beforeEach(async ({ page }) => {
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

      await page.route('**/api/v1/projects/proj-1', async (route) => {
        if (route.request().url().includes('/pipeline')) return route.fallback()
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'proj-1',
            name: 'Test Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })
    })

    test('displays error message with retry button when API fails', async ({ page }) => {
      await page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: { code: 'INTERNAL', message: 'Server error' },
          }),
        })
      })

      await page.goto('/projects/proj-1/pipeline')

      await expect(page.getByTestId('error-message')).toBeVisible()
      await expect(page.getByText('Retry')).toBeVisible()
    })

    test('displays loading skeleton while fetching', async ({ page }) => {
      await page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
        // Delay response to see loading state
        await new Promise((r) => setTimeout(r, 500))
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockPipelineConfig),
        })
      })

      await page.goto('/projects/proj-1/pipeline')

      // Loading skeleton should be visible briefly
      await expect(page.getByTestId('loading-skeleton')).toBeVisible()

      // Then steps should appear
      await expect(page.locator('.font-semibold').filter({ hasText: 'implement' })).toBeVisible()
    })
  })
})
