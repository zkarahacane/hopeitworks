/**
 * Real E2E smoke tests — Story Board
 *
 * These tests run against a live backend at http://localhost:5173 (Vite proxy → :8080).
 * They require seed data: Todo App project with Foundation + Task Management epics,
 * and stories seeded beneath them.
 *
 * Run with: npx playwright test real-tests/smoke-board.spec.ts
 */
import { test, expect } from '@playwright/test'
import { loginViaUI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const TODO_APP_ID = '00000000-0000-0000-0000-000000000101'
const TASK_MGMT_EPIC_ID = '00000000-0000-0000-0000-000000000202'

test.describe('Board smoke tests (real backend)', () => {
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

  test('board shows seed epics', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}/board`)

    // On load, the first epic (Foundation) is auto-selected and visible in the subtitle
    await expect(page.getByText(/Epic · Foundation/i)).toBeVisible({ timeout: 10000 })

    // Click the epic select combobox to open the dropdown
    const epicSelect = page.getByRole('combobox', { name: /Foundation/i })
    await epicSelect.click()

    // The second epic (Task Management) appears as an option in the dropdown
    await expect(page.getByRole('option', { name: /Task Management/i })).toBeVisible({ timeout: 5000 })
  })

  test('epic detail shows stories', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}/epics/${TASK_MGMT_EPIC_ID}`)

    // The page should render without error (not redirected to login)
    await expect(page).not.toHaveURL(/\/login/)

    // Story keys from the Task Management epic (S-04, S-05) should be visible
    const storyKeyPattern = page.getByText(/S-0[0-9]/)
    await expect(storyKeyPattern.first()).toBeVisible({ timeout: 10000 })
  })

  test('stories show correct status badges', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}/board`)

    // Wait for the board (Foundation epic auto-selected) to fully render
    await expect(page.getByText(/Epic · Foundation/i)).toBeVisible({ timeout: 10000 })

    // The seed data includes stories in various statuses.
    // Foundation stories are all in 'backlog' status.
    // Verify that at least one status badge from the expected set is visible.
    const statusTexts = ['completed', 'running', 'backlog', 'failed', 'pending', 'in_progress']

    // Build a locator that matches ANY of the status texts (case-insensitive)
    const anyStatusLocator = page.getByText(
      new RegExp(statusTexts.join('|'), 'i'),
    )

    await expect(
      anyStatusLocator.first(),
      'Expected at least one story with a recognizable status badge',
    ).toBeVisible({ timeout: 8000 })
  })
})
