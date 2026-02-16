import { test, expect } from '@playwright/test'

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    // Mock auth check as unauthenticated by default
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'Unauthorized' }),
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
    const passwordInput = page.locator('#password input')
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
    await page.locator('#password input').fill('password123')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check email validation error
    const emailError = page.locator('small.text-red-500').first()
    await expect(emailError).toContainText('Invalid email format')
  })

  test('should show validation error for short password (< 8 chars)', async ({ page }) => {
    await page.goto('/login')

    // Fill valid email but short password
    await page.locator('#email').fill('test@example.com')
    await page.locator('#password input').fill('short')

    // Submit
    await page.getByRole('button', { name: 'Sign In' }).click()

    // Check password validation error
    const passwordError = page.locator('small.text-red-500').last()
    await expect(passwordError).toContainText('Password must be at least 8 characters')
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
        }),
      })
    })

    await page.goto('/login')

    // Fill valid credentials
    await page.locator('#email').fill('test@test.com')
    await page.locator('#password input').fill('password123')

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
        body: JSON.stringify({ message: 'Invalid credentials' }),
      })
    })

    await page.goto('/login')

    // Fill credentials
    await page.locator('#email').fill('wrong@test.com')
    await page.locator('#password input').fill('wrongpassword123')

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
        }),
      })
    })

    // Visit login with redirect param
    await page.goto('/login?redirect=/stories')

    // Fill valid credentials
    await page.locator('#email').fill('test@test.com')
    await page.locator('#password input').fill('password123')

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
        }),
      })
    })

    // Try to access login page when already authenticated
    await page.goto('/login')

    // Should redirect to dashboard
    await expect(page).toHaveURL('/')
  })
})
