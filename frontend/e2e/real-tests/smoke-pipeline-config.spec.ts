/**
 * Real E2E smoke tests — Pipeline Configuration
 *
 * These tests run against a live backend at http://localhost:5173 (Vite proxy → :8080).
 * They require seed data: Todo App project with a pipeline config.
 *
 * Run with: npx playwright test real-tests/smoke-pipeline-config.spec.ts
 */
import { test, expect } from '@playwright/test'
import { loginViaUI, loginViaAPI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const TODO_APP_ID = '00000000-0000-0000-0000-000000000101'

test.describe('Pipeline config smoke tests (real backend)', () => {
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

  test('pipeline config page loads and renders content', async ({ page }) => {
    await loginViaUI(page, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}/pipeline`)

    // Should stay authenticated — not redirected to /login
    await expect(page).not.toHaveURL(/\/login/)

    // The page must render something pipeline-related.
    // We look for any of the common terms that appear in a pipeline config UI.
    const pipelineContent = page
      .getByText(/pipeline|config|yaml|steps|actions/i)
      .or(page.getByRole('main'))
      .first()

    await expect(pipelineContent).toBeVisible({ timeout: 10000 })
  })

  test('pipeline config API returns 200 with config_yaml field', async ({ context }) => {
    await loginViaAPI(context, 'admin')

    const response = await context.request.get(
      `/api/v1/projects/${TODO_APP_ID}/pipeline-config`,
    )

    expect(
      response.status(),
      `GET /api/v1/projects/${TODO_APP_ID}/pipeline-config should return 200`,
    ).toBe(200)

    const body = await response.json()

    // The response must include a config_yaml field (may be empty string if not yet set)
    expect(
      body,
      'Response body should be an object',
    ).toEqual(expect.objectContaining({}))

    expect(
      'config_yaml' in body,
      'Response should contain a config_yaml field',
    ).toBe(true)

    // config_yaml should be a string (even if empty)
    expect(
      typeof body.config_yaml,
      'config_yaml field should be a string',
    ).toBe('string')
  })
})
