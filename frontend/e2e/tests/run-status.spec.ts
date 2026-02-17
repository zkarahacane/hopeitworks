import { test, expect } from '@playwright/test'

const mockStoriesWithRuns = [
  {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Setup authentication',
    status: 'done',
    latest_run: {
      id: 'run-1',
      status: 'completed',
      started_at: '2026-02-17T10:00:00Z',
      completed_at: '2026-02-17T11:30:00Z',
    },
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: 's2',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-02',
    title: 'Add user profile page',
    status: 'backlog',
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
  {
    id: 's3',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-03',
    title: 'Fix login bug',
    status: 'failed',
    latest_run: {
      id: 'run-3',
      status: 'failed',
      started_at: '2026-02-17T09:00:00Z',
      completed_at: '2026-02-17T09:15:00Z',
      error_message: 'Build step failed: exit code 1',
    },
    created_at: '2026-01-17T10:00:00Z',
    updated_at: '2026-01-17T10:00:00Z',
  },
  {
    id: 's4',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-04',
    title: 'Running pipeline test',
    status: 'running',
    latest_run: {
      id: 'run-4',
      status: 'running',
      started_at: '2026-02-17T11:55:00Z',
    },
    created_at: '2026-01-18T10:00:00Z',
    updated_at: '2026-01-18T10:00:00Z',
  },
]

test.describe('Run Status Display on Story Cards', () => {
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

    await page.route('**/api/v1/projects*', async (route) => {
      if (
        route.request().url().includes('/stories') ||
        route.request().url().includes('/epics')
      ) {
        return route.fallback()
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [{ id: 'p1', name: 'Test Project', description: 'A test project' }],
          pagination: { total: 1, page: 1, per_page: 20 },
        }),
      })
    })

    await page.route('**/api/v1/projects/p1/stories*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: mockStoriesWithRuns }),
      })
    })
  })

  test('displays running status with spinner and "Running..." text', async ({ page }) => {
    await page.goto('/projects/p1/epics/e1')

    const runningCard = page.getByLabel(/Story: S-04/)
    await expect(runningCard).toBeVisible()

    const indicator = runningCard.getByTestId('run-status-indicator')
    await expect(indicator).toBeVisible()
    await expect(indicator.getByTestId('run-status-text')).toHaveText('Running...')
  })

  test('displays completed status with check icon and relative time', async ({ page }) => {
    await page.goto('/projects/p1/epics/e1')

    const completedCard = page.getByLabel(/Story: S-01/)
    await expect(completedCard).toBeVisible()

    const indicator = completedCard.getByTestId('run-status-indicator')
    await expect(indicator).toBeVisible()

    const icon = indicator.getByTestId('run-status-icon')
    await expect(icon).toBeVisible()
    await expect(icon).toHaveClass(/pi-check-circle/)

    const text = indicator.getByTestId('run-status-text')
    await expect(text).toBeVisible()
    // Text should contain relative time (e.g., "1h ago", "2d ago")
    await expect(text).toHaveText(/\d+[mhdw] ago|just now/)
  })

  test('displays failed status with X icon and "Failed" text', async ({ page }) => {
    await page.goto('/projects/p1/epics/e1')

    const failedCard = page.getByLabel(/Story: S-03/)
    await expect(failedCard).toBeVisible()

    const indicator = failedCard.getByTestId('run-status-indicator')
    await expect(indicator).toBeVisible()

    const icon = indicator.getByTestId('run-status-icon')
    await expect(icon).toBeVisible()
    await expect(icon).toHaveClass(/pi-times-circle/)

    await expect(indicator.getByTestId('run-status-text')).toHaveText('Failed')
  })

  test('displays backlog status with dash icon and "Backlog" text', async ({ page }) => {
    await page.goto('/projects/p1/epics/e1')

    const backlogCard = page.getByLabel(/Story: S-02/)
    await expect(backlogCard).toBeVisible()

    const indicator = backlogCard.getByTestId('run-status-indicator')
    await expect(indicator).toBeVisible()

    const icon = indicator.getByTestId('run-status-icon')
    await expect(icon).toBeVisible()
    await expect(icon).toHaveClass(/pi-minus-circle/)

    await expect(indicator.getByTestId('run-status-text')).toHaveText('Backlog')
  })

  test('clicking failed status shows error details in toast', async ({ page }) => {
    await page.goto('/projects/p1/epics/e1')

    const failedCard = page.getByLabel(/Story: S-03/)
    const indicator = failedCard.getByTestId('run-status-indicator')
    await indicator.click()

    await expect(page.getByText('Run Failed')).toBeVisible()
    await expect(page.getByText('Build step failed: exit code 1')).toBeVisible()
  })
})
