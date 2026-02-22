import { test, expect } from '@playwright/test'

const userFixture = {
  id: '1',
  email: 'test@example.com',
  name: 'Test User',
  role: 'member',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

test.describe('Profile Page', () => {
  test.beforeEach(async ({ page }) => {
    // Mock auth check as authenticated
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(userFixture),
      })
    })

    // Mock GET /users/me for profile data
    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(userFixture),
        })
      } else {
        // Let PUT requests fall through to specific test handlers
        await route.fallback()
      }
    })

    // Mock Dashboard API calls (for tests that navigate to /)
    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 5 },
        }),
      })
    })

    await page.route('**/api/v1/hitl-requests*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })
  })

  test('should display profile page with pre-filled data', async ({ page }) => {
    await page.goto('/profile')

    // Check heading
    await expect(page.locator('h1')).toHaveText('My Profile')

    // Check name field is pre-filled
    const nameInput = page.locator('#profile-name')
    await expect(nameInput).toBeVisible()
    await expect(nameInput).toHaveValue('Test User')

    // Check email field is pre-filled
    const emailInput = page.locator('#profile-email')
    await expect(emailInput).toBeVisible()
    await expect(emailInput).toHaveValue('test@example.com')
  })

  test('should show success toast on profile update', async ({ page }) => {
    const updatedUser = { ...userFixture, name: 'Updated Name' }

    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(updatedUser),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(userFixture),
        })
      }
    })

    await page.goto('/profile')

    // Edit name
    const nameInput = page.locator('#profile-name')
    await nameInput.clear()
    await nameInput.fill('Updated Name')

    // Click save
    await page.getByRole('button', { name: 'Save Changes' }).click()

    // Check success toast
    await expect(page.getByText('Profile updated')).toBeVisible()
  })

  test('should show error toast on profile update failure', async ({ page }) => {
    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'BAD_REQUEST', message: 'Invalid email' } }),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(userFixture),
        })
      }
    })

    await page.goto('/profile')

    // Edit name to trigger dirty state
    const nameInput = page.locator('#profile-name')
    await nameInput.clear()
    await nameInput.fill('New Name')

    // Click save
    await page.getByRole('button', { name: 'Save Changes' }).click()

    // Check error toast
    await expect(page.getByText('Error')).toBeVisible()
  })

  test('should navigate to profile from user menu', async ({ page }) => {
    await page.goto('/')

    // Open user menu
    await page.getByTestId('user-menu-button').click()

    // Click "My Profile"
    await page.getByText('My Profile').click()

    // Should navigate to /profile
    await expect(page).toHaveURL('/profile')
    await expect(page.locator('h1')).toHaveText('My Profile')
  })

  test('should show success toast on password change', async ({ page }) => {
    await page.route('**/api/v1/users/me/password', async (route) => {
      await route.fulfill({ status: 204, body: '' })
    })

    await page.goto('/profile')

    // Fill password fields
    await page.locator('#current-password').fill('oldpassword')
    await page.locator('#new-password').fill('newpassword123')
    await page.locator('#confirm-password').fill('newpassword123')

    // Click update password
    await page.getByRole('button', { name: 'Update Password' }).click()

    // Check success toast
    await expect(page.getByText('Password updated')).toBeVisible()

    // Password fields should be cleared
    await expect(page.locator('#current-password')).toHaveValue('')
    await expect(page.locator('#new-password')).toHaveValue('')
    await expect(page.locator('#confirm-password')).toHaveValue('')
  })

  test('should show validation error for mismatched passwords', async ({ page }) => {
    await page.goto('/profile')

    // Fill password fields with mismatch
    await page.locator('#current-password').fill('oldpassword')
    await page.locator('#new-password').fill('newpassword123')
    await page.locator('#confirm-password').fill('differentpassword')

    // Trigger validation by blurring the field
    await page.locator('#confirm-password').blur()

    // Check validation error
    await expect(page.getByText('Passwords do not match')).toBeVisible()

    // Button should be disabled
    await expect(page.getByRole('button', { name: 'Update Password' })).toBeDisabled()
  })
})
