import { test, expect } from './fixtures'

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

// Bug #288: deleting an API key returned 501 and double-fired the request, with
// no user feedback. These specs cover the acceptance criteria RG1-RG6.
// RG5 (non-UUID id -> 400) and RG6 (route no longer 501) are server concerns
// proven by the backend tests; the UI only ever sends valid ids and consumes a
// 204, so they are not re-asserted at the e2e layer.
test.describe('Profile — API Keys (#288)', () => {
  const apiKeyFixture = {
    id: '11111111-1111-1111-1111-111111111111',
    provider: 'claude',
    key_name: 'default',
    key_hint: '...1234',
    created_at: '2026-01-01T00:00:00Z',
  }
  // Stateful list so the GET mock stays faithful to the server across a delete:
  // a successful DELETE mutates it, so any refetch reflects the real state.
  let apiKeys: Array<typeof apiKeyFixture>

  test.beforeEach(async ({ page }) => {
    apiKeys = [{ ...apiKeyFixture }]

    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(userFixture) })
    })
    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(userFixture) })
      } else {
        await route.fallback()
      }
    })
    await page.route('**/api/v1/users/me/api-keys', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(apiKeys) })
      } else {
        await route.fallback()
      }
    })
  })

  /** Mocks DELETE: 204 (and mutates the stateful list) or a failure status. */
  async function mockDelete(page: import('@playwright/test').Page, status: number) {
    await page.route('**/api/v1/users/me/api-keys/*', async (route) => {
      if (route.request().method() !== 'DELETE') {
        await route.fallback()
        return
      }
      if (status === 204) {
        const id = route.request().url().split('/').pop()
        apiKeys = apiKeys.filter((k) => k.id !== id)
        await route.fulfill({ status: 204, body: '' })
      } else {
        await route.fulfill({
          status,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'INTERNAL_ERROR', message: 'boom' } }),
        })
      }
    })
  }

  /** Tracks every DELETE issued against the api-keys collection. */
  function trackDeletes(page: import('@playwright/test').Page): string[] {
    const calls: string[] = []
    page.on('request', (req) => {
      if (req.method() === 'DELETE' && req.url().includes('/users/me/api-keys/')) {
        calls.push(req.url())
      }
    })
    return calls
  }

  // PrimeVue ConfirmDialog uses role="alertdialog" and default labels Yes/No.
  // Anchor on the button labels (^...$) so they never match the trash button
  // (aria-label "Delete API key"); the accept button being visible is the
  // "dialog open" signal.
  const acceptBtn = (page: import('@playwright/test').Page) =>
    page.getByRole('button', { name: /^(yes|accept|confirm|ok)$/i })
  const rejectBtn = (page: import('@playwright/test').Page) =>
    page.getByRole('button', { name: /^(no|cancel|reject)$/i })

  async function openDeleteConfirm(page: import('@playwright/test').Page) {
    await expect(page.getByRole('cell', { name: 'default' })).toBeVisible()
    await page.getByRole('button', { name: 'Delete API key' }).click()
    await expect(acceptBtn(page)).toBeVisible()
  }

  test('RG1: confirming a delete returns 204, removes the row and shows a success toast', async ({ page }) => {
    await mockDelete(page, 204)

    await page.goto('/profile')
    await openDeleteConfirm(page)
    await acceptBtn(page).click()

    await expect(page.getByText('API key deleted')).toBeVisible()
    await expect(page.getByText('No API keys configured yet.')).toBeVisible()
    await expect(page.getByRole('cell', { name: 'default' })).toHaveCount(0)
  })

  test('RG2: cancelling the dialog emits no DELETE and keeps the key', async ({ page }) => {
    const deletes = trackDeletes(page)

    await page.goto('/profile')
    await openDeleteConfirm(page)
    await rejectBtn(page).click()

    await expect(acceptBtn(page)).toBeHidden()
    await expect(page.getByRole('cell', { name: 'default' })).toBeVisible()
    expect(deletes).toHaveLength(0)
  })

  test('RG3: confirming once emits exactly one DELETE (no double-fire)', async ({ page }) => {
    const deletes = trackDeletes(page)
    await mockDelete(page, 204)

    await page.goto('/profile')
    await openDeleteConfirm(page)
    await acceptBtn(page).click()

    await expect(page.getByText('API key deleted')).toBeVisible()
    expect(deletes).toHaveLength(1)
  })

  test('RG4: an API failure shows a visible error and keeps the key (no optimistic removal)', async ({ page }) => {
    await mockDelete(page, 500)

    await page.goto('/profile')
    await openDeleteConfirm(page)
    await acceptBtn(page).click()

    await expect(page.getByText('Delete failed')).toBeVisible()
    await expect(page.getByRole('cell', { name: 'default' })).toBeVisible()
  })
})
