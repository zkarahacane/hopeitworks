import { test, expect } from '@playwright/test'

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    // Mock auth check as unauthenticated by default
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } }),
      })
    })
  })

  test('should display login form with all elements', async ({ page }) => {
    await page.goto('/login')

    // Check heading
    await expect(page.locator('h1')).toHaveText('hopeitworks')

    // Check email field
    await expect(page.locator('label[for="email"]')).toHaveText('Email')
    const emailInput = page.locator('#email')
    await expect(emailInput).toBeVisible()
    await expect(emailInput).toHaveAttribute('placeholder', 'you@example.com')

    // Check password field
    await expect(page.locator('label[for="password"]')).toHaveText('Password')
    const passwordInput = page.locator('#password')
    await expect(passwordInput).toBeVisible()

    // Check submit button
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible()
  })

  test('should show validation errors for empty fields on submit', async ({ page }) => {
    await page.goto('/login')

    // Submit without filling fields
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check validation errors
    const errors = page.locator('small.text-red-500')
    await expect(errors).toHaveCount(2)
    await expect(errors.first()).toContainText('Required')
    await expect(errors.last()).toContainText('Required')
  })

  test('should show validation error for invalid email format', async ({ page }) => {
    await page.goto('/login')

    // Fill invalid email
    await page.locator('#email').fill('not-an-email')
    await page.locator('#password').fill('password123')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check email validation error
    const emailError = page.locator('small.text-red-500').first()
    await expect(emailError).toContainText('Invalid email format')
  })

  test('should show validation error for empty password', async ({ page }) => {
    await page.goto('/login')

    // Fill valid email but leave password empty
    await page.locator('#email').fill('test@example.com')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check password validation error
    const passwordError = page.locator('small.text-red-500').last()
    await expect(passwordError).toContainText('Password is required')
  })

  test('should successfully login and redirect to dashboard', async ({ page }) => {
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
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

    // Mock Dashboard API calls
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

    await page.goto('/login')

    // Fill valid credentials
    await page.locator('#email').fill('test@test.com')
    await page.locator('#password').fill('password123')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should redirect to dashboard
    await expect(page).toHaveURL('/')
  })

  test('should display API error message on failed login', async ({ page }) => {
    // Mock failed login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'UNAUTHORIZED', message: 'Invalid credentials' } }),
      })
    })

    await page.goto('/login')

    // Fill credentials
    await page.locator('#email').fill('wrong@test.com')
    await page.locator('#password').fill('wrongpassword123')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check error message
    const errorMessage = page.locator('[role="alert"]')
    await expect(errorMessage).toContainText('Invalid credentials')
  })

  test('should redirect to original URL after login (using ?redirect= query param)', async ({
    page,
  }) => {
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
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

    // Visit login with redirect param
    await page.goto('/login?redirect=/stories')

    // Fill valid credentials
    await page.locator('#email').fill('test@test.com')
    await page.locator('#password').fill('password123')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should redirect to the original URL
    await expect(page).toHaveURL('/stories')
  })

  test('should redirect unauthenticated users to /login (auth guard)', async ({ page }) => {
    // Try to access protected route
    await page.goto('/')

    // Should be redirected to login with redirect param
    await expect(page).toHaveURL(/\/login\?redirect=/)
  })

  test('should not render AppHeader or AppSidebar on /login when unauthenticated', async ({
    page,
  }) => {
    await page.goto('/login')

    // Login form should be visible
    await expect(page.locator('h1')).toHaveText('hopeitworks')

    // Header and sidebar should not be present
    await expect(page.locator('header')).toHaveCount(0)
    await expect(page.locator('aside')).toHaveCount(0)

    // Status bar footer should not be present
    await expect(page.locator('footer')).toHaveCount(0)

    // Mobile navigation should not be present
    await expect(page.locator('nav[aria-label="Mobile navigation"]')).toHaveCount(0)
  })

  test('should render AppHeader and AppSidebar after successful login', async ({ page }) => {
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
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

    // Mock Dashboard API calls
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

    await page.goto('/login')

    // Verify no shell chrome before login
    await expect(page.locator('header')).toHaveCount(0)

    // Fill valid credentials and submit
    await page.locator('#email').fill('test@test.com')
    await page.locator('#password').fill('password123')
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Should redirect to dashboard
    await expect(page).toHaveURL('/')

    // Header and sidebar should now be visible
    await expect(page.locator('header')).toBeVisible()
    await expect(page.locator('aside')).toBeVisible()
  })

  test('should redirect authenticated users from /login to / (already logged in)', async ({
    page,
  }) => {
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

    // Mock Dashboard API calls
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

    // Try to access login page when already authenticated
    await page.goto('/login')

    // Should redirect to dashboard
    await expect(page).toHaveURL('/')
  })
})
