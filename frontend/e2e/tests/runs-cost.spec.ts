import { test, expect } from './fixtures'

// #290 — the runs list shows each run's aggregated cost in the Cost column,
// without opening the run detail. A run with cost records shows its summed value,
// a run without any shows an em dash (distinct from a real $0.00).
const mockRuns = [
  {
    id: 'run-cost-1',
    project_id: 'proj-1',
    story_id: 'story-1',
    status: 'completed',
    progress: 100,
    story_key: 'S-01',
    started_at: '2026-02-17T10:00:00Z',
    completed_at: '2026-02-17T11:00:00Z',
    created_at: '2026-02-17T10:00:00Z',
    updated_at: '2026-02-17T11:00:00Z',
    cost_usd: 0.8145,
  },
  {
    id: 'run-nocost-2',
    project_id: 'proj-1',
    story_id: 'story-2',
    status: 'pending',
    progress: 0,
    story_key: 'S-02',
    created_at: '2026-02-16T10:00:00Z',
    updated_at: '2026-02-16T10:00:00Z',
    cost_usd: null,
  },
]

test.describe('Runs list — Cost column (#290)', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          email: 'test@test.com',
          name: 'Test User',
          role: 'admin',
        }),
      })
    })

    await page.route('**/api/v1/projects/proj-1/runs*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockRuns,
          pagination: { total: mockRuns.length, page: 1, per_page: 20 },
        }),
      })
    })
  })

  test('displays the aggregated cost and an em dash for a run without cost', async ({
    page,
  }) => {
    await page.goto('/projects/proj-1/runs')

    const table = page.getByTestId('project-runs-table')
    await expect(table).toBeVisible()

    // Run with cost records → exact aggregated value, identical to the detail view.
    const costRow = table.locator('tbody tr', { hasText: 'run-cost' })
    await expect(costRow).toContainText('$0.8145')

    // Run without any cost record → em dash, never a dollar amount.
    const noCostRow = table.locator('tbody tr', { hasText: 'run-noco' })
    await expect(noCostRow).toContainText('—')
    await expect(noCostRow).not.toContainText('$')
  })
})
