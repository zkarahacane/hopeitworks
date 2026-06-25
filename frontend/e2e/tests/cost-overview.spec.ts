import { test, expect } from './fixtures'

/**
 * Overview → Costs E2E (ticket #292): the COST BY ROLE widget renders real
 * per-role bars (no "unavailable" message) and the Recent Runs table shows the
 * real Tokens In/Out — both wired to the project-level cost endpoints.
 */

const mockSummary = {
  total_cost_usd: 17.75,
  total_cost_week_usd: 17.75,
  total_cost_month_usd: 17.75,
  avg_cost_per_story_usd: 8.875,
  budget_limit_usd: 0,
  period_start: '2026-02-10T00:00:00Z',
  period_end: '2026-02-17T00:00:00Z',
}

const mockChart = [
  { date: '2026-02-10', total_cost_usd: 5.0 },
  { date: '2026-02-11', total_cost_usd: 12.75 },
]

const mockRuns = {
  data: [
    {
      run_id: 'run-1',
      story_key: 'S-01',
      status: 'completed',
      started_at: '2026-02-10T10:00:00Z',
      total_cost_usd: 12.5,
      tokens_input: 500000,
      tokens_output: 100000,
    },
  ],
  pagination: { total: 1, page: 1, per_page: 20 },
}

const mockByRole = {
  total_cost: 17.75,
  total_tokens_input: 700000,
  total_tokens_output: 150000,
  roles: [
    { role: 'implement', tokens_input: 500000, tokens_output: 100000, cost_usd: 12.5, runs_count: 3 },
    { role: 'review', tokens_input: 200000, tokens_output: 50000, cost_usd: 5.25, runs_count: 2 },
  ],
}

async function mockAuth(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: '1', email: 'test@test.com', name: 'Test User', role: 'admin' }),
    })
  })
}

test.describe('Overview Costs — COST BY ROLE + tokens', () => {
  test('COST BY ROLE shows per-role bars and Recent Runs shows real tokens', async ({ page }) => {
    await mockAuth(page)
    await page.route('**/api/v1/projects/p1/costs/summary*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockSummary) }),
    )
    await page.route('**/api/v1/projects/p1/costs/chart*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockChart) }),
    )
    await page.route('**/api/v1/projects/p1/costs/runs*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockRuns) }),
    )
    await page.route('**/api/v1/projects/p1/costs/by-role*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockByRole) }),
    )

    await page.goto('/projects/p1/costs')

    // RG1: per-role bars are visible, not the "unavailable" fallback.
    const panel = page.getByTestId('cost-by-role-panel')
    await expect(panel).toBeVisible()
    await expect(page.getByTestId('cost-by-role-bars')).toBeVisible()
    await expect(page.getByTestId('cost-by-role-empty')).toHaveCount(0)
    const rows = page.getByTestId('cost-by-role-row')
    await expect(rows).toHaveCount(2)
    await expect(panel).toContainText('Implement')
    await expect(panel).toContainText('Review')

    // RG2: Recent Runs shows the real tokens in/out, not 0.
    const runsTable = page.getByTestId('runs-table')
    await expect(runsTable).toBeVisible()
    await expect(runsTable).toContainText('500,000')
    await expect(runsTable).toContainText('100,000')
  })

  test('shows an error with Retry when the by-role endpoint fails (RG5)', async ({ page }) => {
    await mockAuth(page)
    await page.route('**/api/v1/projects/p1/costs/summary*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockSummary) }),
    )
    await page.route('**/api/v1/projects/p1/costs/chart*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockChart) }),
    )
    await page.route('**/api/v1/projects/p1/costs/runs*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(mockRuns) }),
    )
    await page.route('**/api/v1/projects/p1/costs/by-role*', (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: { code: 'INTERNAL', message: 'boom' } }),
      }),
    )

    await page.goto('/projects/p1/costs')

    const errorBox = page.getByTestId('cost-error')
    await expect(errorBox).toBeVisible()
    await expect(errorBox.getByRole('button', { name: 'Retry' })).toBeVisible()
  })
})
