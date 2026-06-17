import { test, expect } from './fixtures'

/**
 * E2E tests for Story 1-8: App Shell Layout
 *
 * Tests the complete app shell integration including:
 * - AppHeader: brand, user menu, mobile hamburger
 * - AppSidebar: navigation items, collapsible behavior
 * - AppStatusBar: connection status, version display
 * - AppShell: layout integration, routing
 */

test.describe('App Shell Layout', () => {
  test.beforeEach(async ({ page }) => {
    // Mock auth API to simulate authenticated user
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          email: 'test@test.com',
          name: 'Test',
          role: 'user',
        }),
      })
    })

    // Mock Dashboard API calls (Dashboard loads projects, HITL requests, and runs on mount)
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

    // Navigate to the root route (Dashboard)
    await page.goto('/')
  })

  test.describe('Header', () => {
    test('should display header with brand name "Hope"', async ({ page }) => {
      const brandText = page.locator('header span.text-lg.font-semibold')
      await expect(brandText).toBeVisible()
      await expect(brandText).toHaveText('Hope')
    })

    test('should show user menu button', async ({ page }) => {
      const userMenuButton = page.getByRole('button', { name: 'User menu' })
      await expect(userMenuButton).toBeVisible()
    })
  })

  test.describe('Sidebar (desktop)', () => {
    test('should display sidebar with navigation items (Dashboard, Projects, Runs, Settings)', async ({
      page,
    }) => {
      // Set viewport to desktop size
      await page.setViewportSize({ width: 1280, height: 720 })

      // Check for all navigation items in the sidebar specifically
      const sidebar = page.locator('aside')
      const dashboardButton = sidebar.getByRole('button', { name: 'Dashboard' })
      const projectsButton = sidebar.getByRole('button', { name: 'Projects' })
      const runsButton = sidebar.getByRole('button', { name: 'Runs' })
      const settingsButton = sidebar.getByRole('button', { name: 'Settings' })

      await expect(dashboardButton).toBeVisible()
      await expect(projectsButton).toBeVisible()
      await expect(runsButton).toBeVisible()
      await expect(settingsButton).toBeVisible()
    })

    test('should navigate to correct route when clicking nav items', async ({
      page,
    }) => {
      // Set viewport to desktop size
      await page.setViewportSize({ width: 1280, height: 720 })

      // Click on Projects navigation item in sidebar
      const sidebar = page.locator('aside')
      const projectsButton = sidebar.getByRole('button', { name: 'Projects' })
      await projectsButton.click()

      // Verify URL changed to /projects
      await expect(page).toHaveURL('/projects')

      // Navigate back to Dashboard
      const dashboardButton = sidebar.getByRole('button', { name: 'Dashboard' })
      await dashboardButton.click()

      // Verify URL changed back to /
      await expect(page).toHaveURL('/')
    })

    test('should show main content area with router-view', async ({ page }) => {
      // Set viewport to desktop size
      await page.setViewportSize({ width: 1280, height: 720 })

      // Verify main content area exists
      const mainContent = page.locator('main#main-content')
      await expect(mainContent).toBeVisible()

      // Verify router-view renders content (Dashboard heading)
      const heading = mainContent.locator('h1')
      await expect(heading).toBeVisible()
      await expect(heading).toHaveText('Dashboard')
    })
  })

  test.describe('Status Bar', () => {
    test('should display status bar with "Connected" text', async ({
      page,
    }) => {
      // Set viewport to desktop size (status bar hidden on mobile)
      await page.setViewportSize({ width: 1280, height: 720 })

      const statusBar = page.locator('footer')
      await expect(statusBar).toBeVisible()

      await expect(statusBar.getByText('Connected')).toBeVisible()
    })

    test('should display version "v0.0.0" in status bar', async ({ page }) => {
      // Set viewport to desktop size (status bar hidden on mobile)
      await page.setViewportSize({ width: 1280, height: 720 })

      const statusBar = page.locator('footer')
      await expect(statusBar).toBeVisible()

      await expect(statusBar.getByText('v0.0.0')).toBeVisible()
    })
  })

  test.describe('Logout flow', () => {
    test('should logout and redirect to /login on success', async ({ page }) => {
      await page.route('**/api/v1/auth/logout', async (route) => {
        await route.fulfill({ status: 204 })
      })
      // After logout, /me should return 401 so guard redirects
      await page.unroute('**/api/v1/auth/me')
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({ status: 401 })
      })
      await page.getByTestId('user-menu-button').click()
      await page.getByRole('menuitem', { name: 'Logout' }).click()
      await expect(page).toHaveURL('/login')
    })

    test('should logout even when backend returns 500', async ({ page }) => {
      await page.route('**/api/v1/auth/logout', async (route) => {
        await route.fulfill({ status: 500 })
      })
      await page.unroute('**/api/v1/auth/me')
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({ status: 401 })
      })
      await page.getByTestId('user-menu-button').click()
      await page.getByRole('menuitem', { name: 'Logout' }).click()
      await expect(page).toHaveURL('/login')
    })

    test('should redirect to /login with redirect param after logout', async ({ page }) => {
      await page.route('**/api/v1/auth/logout', async (route) => {
        await route.fulfill({ status: 204 })
      })
      await page.unroute('**/api/v1/auth/me')
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({ status: 401 })
      })
      // Logout first
      await page.getByTestId('user-menu-button').click()
      await page.getByRole('menuitem', { name: 'Logout' }).click()
      await expect(page).toHaveURL('/login')
      // Try to navigate to a protected route
      await page.goto('/')
      await expect(page).toHaveURL(/\/login\?redirect=/)
    })

    test('should display user name and email in user menu', async ({ page }) => {
      await page.getByTestId('user-menu-button').click()
      const userMenu = page.getByTestId('user-menu')
      await expect(userMenu).toContainText('Test')
      await expect(userMenu).toContainText('test@test.com')
    })
  })

  test.describe('Layout Integration', () => {
    test('should render Dashboard view at root route', async ({ page }) => {
      // Already at root route from beforeEach
      await expect(page).toHaveURL('/')

      const heading = page.locator('h1')
      await expect(heading).toBeVisible()
      await expect(heading).toHaveText('Dashboard')
    })

    test('should render Projects view at /projects route', async ({ page }) => {
      // Navigate to /projects
      await page.goto('/projects')
      await expect(page).toHaveURL('/projects')

      const heading = page.locator('h1')
      await expect(heading).toBeVisible()
      await expect(heading).toHaveText('Projects')
    })
  })
})
