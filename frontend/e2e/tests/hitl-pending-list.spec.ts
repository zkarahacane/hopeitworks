import { test, expect } from './fixtures'

const mockAuthResponse = {
  id: '1',
  email: 'test@test.com',
  name: 'Test User',
  role: 'admin',
}

const mockPendingItems = {
  data: [
    {
      id: 'hr-1',
      run_step_id: 'rs-1',
      step_id: 's-1',
      run_id: 'r-1',
      project_id: 'p-1',
      gate_type: 'approval',
      status: 'pending',
      story_key: 'S-01',
      story_title: 'Implement login page',
      created_at: '2026-02-17T10:00:00Z',
    },
    {
      id: 'hr-2',
      run_step_id: 'rs-2',
      step_id: 's-2',
      run_id: 'r-2',
      project_id: 'p-2',
      gate_type: 'approval',
      status: 'pending',
      story_key: 'S-02',
      story_title: 'Add dashboard',
      created_at: '2026-02-17T11:00:00Z',
    },
  ],
  pagination: { total: 2, page: 1, per_page: 20 },
}

test.describe('HITL Pending List and Notification Badge', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockAuthResponse),
      })
    })

    // Mock the SSE endpoint to prevent connection errors
    await page.route('**/api/v1/events/stream*', async (route) => {
      await route.abort()
    })
  })

  test('shows badge with count when pending items exist and list page shows rows', async ({
    page,
  }) => {
    await page.route('**/api/v1/hitl-requests?status=pending*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPendingItems),
      })
    })

    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto('/approvals')

    // Verify the Approvals nav item exists in sidebar
    const approvalsButton = page.getByRole('button', { name: 'Approvals' })
    await expect(approvalsButton).toBeVisible()

    // Verify the page heading
    const heading = page.locator('h1')
    await expect(heading).toHaveText('Approvals')

    // Redesign: pending items render as gate cards (one per pending request),
    // each showing the story key. Story title is no longer shown on the card.
    const cards = page.getByTestId('approvals-gate-card')
    await expect(cards).toHaveCount(2)
    await expect(page.getByTestId('hitl-gate-story').filter({ hasText: 'S-01' })).toBeVisible()
    await expect(page.getByTestId('hitl-gate-story').filter({ hasText: 'S-02' })).toBeVisible()
  })

  // TODO(redesign): the standalone "Review" button + navigation to
  // /projects/:projectId/runs/:runId/approve/:stepId was removed. The Approvals
  // page now exposes inline Approve / Request changes / Reject actions on each
  // gate card (navigating to /runs/:runId on action), so this Review-navigation
  // flow no longer exists and can't be salvaged without re-asserting a different intent.
  test.skip('clicking Review navigates to the approval page', async ({ page }) => {
    await page.route('**/api/v1/hitl-requests?status=pending*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPendingItems),
      })
    })

    // Mock the HITL request detail endpoint for navigation target
    await page.route('**/api/v1/hitl-requests/by-step/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPendingItems.data[0]),
      })
    })

    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto('/approvals')

    // Wait for table to load
    await expect(page.locator('[data-testid="hitl-pending-table"]').getByText('S-01')).toBeVisible()

    // Click the first Review button
    const reviewButtons = page.getByRole('button', { name: 'Review' })
    await reviewButtons.first().click()

    // Verify navigation to approval page
    await expect(page).toHaveURL(/\/projects\/p-1\/runs\/r-1\/approve\/s-1/)
  })

  test('shows empty state when no pending approvals', async ({ page }) => {
    await page.route('**/api/v1/hitl-requests?status=pending*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto('/approvals')

    // Verify empty state message
    await expect(page.getByText('No pending approvals')).toBeVisible()
  })
})
