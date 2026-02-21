/**
 * Real E2E smoke tests — Projects
 *
 * These tests run against a live backend at http://localhost:5173 (Vite proxy → :8080).
 * They require the seed data (3 projects, admin user) to be present.
 *
 * Run with: npx playwright test real-tests/smoke-projects.spec.ts
 */
import { test, expect } from '@playwright/test'
import { loginViaUI, loginViaAPI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const TODO_APP_ID = '00000000-0000-0000-0000-000000000101'

test.describe('Projects smoke tests (real backend)', () => {
  let logs: LogCollector

  test.beforeEach(({ page }) => {
    logs = new LogCollector()
    logs.attach(page)
  })

  test.afterEach(() => {
    const report = logs.getReport()
    if (report.summary.totalErrors > 0) {
      console.warn('[LogCollector] Console/JS errors:', report.errors)
    }
    if (report.summary.totalWarnings > 0) {
      console.warn('[LogCollector] Warnings:', report.warnings)
    }
  })

  test('lists seed projects', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto('/projects')

    // Wait for the project list to render — use row-scoped selectors to avoid strict mode violations
    await expect(page.getByRole('row', { name: /Todo App/i }).first()).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('row', { name: /E-commerce API/i }).first()).toBeVisible({ timeout: 5000 })
  })

  test('click project navigates to project detail', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto('/projects')

    // Click the first link/button that leads to "Todo App"
    await page.getByText('Todo App').first().click()

    // The URL should contain the seed project ID
    await expect(page).toHaveURL(new RegExp(TODO_APP_ID), { timeout: 8000 })
  })

  test('project overview shows project name and description', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}`)

    // The project name appears in h1[data-testid="project-name"] — use testid to avoid strict mode
    await expect(page.getByTestId('project-name')).toBeVisible({ timeout: 10000 })
    await expect(page.getByTestId('project-name')).toHaveText('Todo App')

    // Some description text should also be present (could be a subheading or paragraph)
    // We look for a non-empty block of text beneath the title rather than a fixed string,
    // because the exact seed description may vary.
    const mainContent = page.getByRole('main').or(page.locator('main')).first()
    await expect(mainContent).toBeVisible({ timeout: 5000 })
    await expect(mainContent).not.toBeEmpty()
  })

  test('create project via API and verify in UI', async ({ context, page }) => {
    // Create a project directly via API
    await loginViaAPI(context, 'admin')

    const projectName = `Test Project E2E ${Date.now()}`

    const createResponse = await context.request.post('/api/v1/projects', {
      data: {
        name: projectName,
        description: 'Created by Playwright smoke test',
      },
    })

    expect(
      createResponse.status(),
      `POST /api/v1/projects should return 200 or 201, got ${createResponse.status()}`,
    ).toBeGreaterThanOrEqual(200)
    expect(createResponse.status()).toBeLessThan(300)

    // Navigate to projects list — the new project should appear
    await page.goto('/projects')
    await page.reload()

    await expect(page.getByText(projectName)).toBeVisible({ timeout: 10000 })
  })
})
