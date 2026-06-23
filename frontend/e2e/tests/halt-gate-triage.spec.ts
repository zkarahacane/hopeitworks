import { test, expect } from './fixtures'

// ── Halt-gate triage (/halts) ─────────────────────────────────────────────────
// A guard probe (timeout / heartbeat / cost) can park a run on a "halt-gate". The
// /halts view lists every parked halt grouped by probe reason, with the resolution
// actions (resume / override / skip / send back / abort) on each card and a
// "Resume all" bulk action per group.
//
// Fully mocked (see ./fixtures): GET /probe-halts lists the parked halts;
// POST /hitl-requests/{id}/resolve resolves a single halt.

// Two probe reasons → two groups (log_silence ×2, cost_batch ×1).
const mockHalts = [
  {
    id: 'h-1',
    run_step_id: 'rs-1',
    run_id: 'run-1',
    project_id: 'p1',
    story_key: 'S-01',
    story_title: 'Implement auth',
    step_name: 'implement',
    stage_name: 'Development',
    halt_reason: { probe: 'log_silence', observed: 240, threshold: 120, unit: 'seconds' },
    created_at: '2026-02-15T10:00:00Z',
  },
  {
    id: 'h-2',
    run_step_id: 'rs-2',
    run_id: 'run-2',
    project_id: 'p1',
    story_key: 'S-02',
    story_title: 'Add profile page',
    step_name: 'review',
    stage_name: 'Review',
    halt_reason: { probe: 'log_silence', observed: 300, threshold: 120, unit: 'seconds' },
    created_at: '2026-02-15T10:05:00Z',
  },
  {
    id: 'h-3',
    run_step_id: 'rs-3',
    run_id: 'run-3',
    project_id: 'p1',
    story_key: 'S-03',
    story_title: 'Build pipeline',
    step_name: 'implement',
    stage_name: 'Development',
    halt_reason: { probe: 'cost_batch', observed: 12.5, threshold: 10, unit: 'usd' },
    created_at: '2026-02-15T10:10:00Z',
  },
]

async function setupAuthMock(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: '1', email: 'admin@test.com', name: 'Admin', role: 'admin' }),
    })
  })
}

test.describe('Halt-gate triage', () => {
  test('lists parked halts grouped by probe reason with resolution actions', async ({ page }) => {
    await setupAuthMock(page)
    await page.route('**/api/v1/probe-halts*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockHalts, total: mockHalts.length }),
      })
    })

    await page.goto('/halts')

    await expect(page.getByRole('heading', { name: 'Halt-gate triage' })).toBeVisible()

    // One card per parked halt.
    await expect(page.getByTestId('halt-gate-card-item')).toHaveCount(3)

    // Grouped by probe reason → a "Resume all" bulk action per group (2 groups).
    await expect(page.getByTestId('halt-group-bulk-resume')).toHaveCount(2)

    // Each card surfaces the structured halt context + the resolution actions.
    const firstCard = page.getByTestId('halt-gate-card-item').first()
    await expect(firstCard.getByTestId('halt-gate-story')).toHaveText('S-01')
    await expect(firstCard.getByTestId('halt-gate-stage')).toHaveText('Development')
    await expect(firstCard.getByTestId('halt-gate-description')).toContainText('240')
    await expect(firstCard.getByTestId('halt-gate-resume')).toBeVisible()
    await expect(firstCard.getByTestId('halt-gate-override')).toBeVisible()
    await expect(firstCard.getByTestId('halt-gate-skip')).toBeVisible()
    await expect(firstCard.getByTestId('halt-gate-send-back')).toBeVisible()
    await expect(firstCard.getByTestId('halt-gate-abort')).toBeVisible()
  })

  test('resuming a halt POSTs the resolve action and removes the card', async ({ page }) => {
    await setupAuthMock(page)
    await page.route('**/api/v1/probe-halts*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockHalts, total: mockHalts.length }),
      })
    })

    let resolvedId = ''
    let resolvedAction = ''
    await page.route('**/api/v1/hitl-requests/*/resolve', async (route) => {
      expect(route.request().method()).toBe('POST')
      const body = route.request().postDataJSON()
      resolvedAction = body.action
      resolvedId = route.request().url()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'resolved' }),
      })
    })

    await page.goto('/halts')
    await expect(page.getByTestId('halt-gate-card-item')).toHaveCount(3)

    // Resolve the first halt by resuming it.
    await page.getByTestId('halt-gate-card-item').first().getByTestId('halt-gate-resume').click()

    // Success toast + the resolved card is dropped from the list.
    await expect(page.getByText('S-01 resolved')).toBeVisible()
    await expect(page.getByTestId('halt-gate-card-item')).toHaveCount(2)

    expect(resolvedAction).toBe('resume')
    expect(resolvedId).toContain('/hitl-requests/h-1/resolve')
  })

  test('shows the empty state when no runs are parked', async ({ page }) => {
    await setupAuthMock(page)
    await page.route('**/api/v1/probe-halts*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], total: 0 }),
      })
    })

    await page.goto('/halts')

    await expect(page.getByText('No parked halts')).toBeVisible()
    await expect(page.getByTestId('halt-gate-card-item')).toHaveCount(0)
  })

  test('shows an error with retry when the halts fetch fails', async ({ page }) => {
    await setupAuthMock(page)
    let callCount = 0
    await page.route('**/api/v1/probe-halts*', async (route) => {
      callCount++
      if (callCount === 1) {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'INTERNAL', message: 'boom' } }),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: mockHalts, total: mockHalts.length }),
        })
      }
    })

    await page.goto('/halts')

    await expect(page.getByText('Failed to load probe halts')).toBeVisible()
    await page.getByRole('button', { name: 'Retry' }).click()

    await expect(page.getByTestId('halt-gate-card-item')).toHaveCount(3)
  })
})
