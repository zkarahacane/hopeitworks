import { test, expect } from './fixtures'

test.describe('Application Routing', () => {
  test.describe('Route rendering (authenticated)', () => {
    test.beforeEach(async ({ page }) => {
      // Mock authenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '1',
            email: 'test@test.com',
            name: 'Test User',
            role: 'user',
          }),
        })
      })

      // Mock Dashboard API calls
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
    })

    test('should render Dashboard view at /', async ({ page }) => {
      await page.goto('/')

      // Redesign: dashboard greets the user instead of a static "Dashboard" title
      await expect(page.locator('h1')).toHaveText(/Welcome back,/)
      await expect(page).toHaveURL('/')
    })

    test('should render Projects view at /projects', async ({ page }) => {
      await page.goto('/projects')

      await expect(page.locator('h1')).toHaveText('Projects')
      await expect(page).toHaveURL('/projects')
    })

    test('should render Project Detail view at /projects/123', async ({ page }) => {
      await page.route('**/api/v1/projects/123', async (route) => {
        if (
          route.request().url().includes('/epics') ||
          route.request().url().includes('/pipeline') ||
          route.request().url().includes('/templates')
        ) {
          return route.fallback()
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '123',
            name: 'My Project',
            owner_id: 'u1',
            created_at: '2026-01-01T00:00:00Z',
            updated_at: '2026-01-01T00:00:00Z',
          }),
        })
      })

      await page.goto('/projects/123')

      await expect(page.getByTestId('project-name')).toHaveText('My Project')
      await expect(page).toHaveURL('/projects/123')
    })

    test.describe('Run Detail view at /runs/456', () => {
      // A minimal RunWithSteps payload matching api/openapi.yaml. The Run schema
      // has no `story_title` — the Run Detail <h1> only shows a custom title when
      // a HITL request carries one, so an empty `steps` array means no human gate
      // → no HITL fetch → the header falls back to "Run Detail" (RG4).
      const runMock = () => ({
        id: '00000000-0000-0000-0000-000000000456',
        project_id: '00000000-0000-0000-0000-000000000001',
        story_id: '00000000-0000-0000-0000-000000000002',
        status: 'running' as const,
        progress: 0,
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
        steps: [],
      })

      // The view title is via getByRole('heading', { name: 'Run Detail' }) so the
      // assertion is unambiguous even if other headings exist on the page.
      const runDetailHeading = (page: import('@playwright/test').Page) =>
        page.getByRole('heading', { name: 'Run Detail' })

      // Stub the secondary run-scoped costs call so it resolves deterministically
      // instead of hanging on an unmocked network request (RG5). The SSE stream
      // is already stubbed by the shared fixture.
      const stubCosts = async (page: import('@playwright/test').Page) => {
        await page.route('**/api/v1/projects/**/runs/**/costs*', async (route) => {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({
              run_id: '00000000-0000-0000-0000-000000000456',
              total_cost: 0,
              steps: [],
            }),
          })
        })
      }

      test('RG1: mocked run (200) → renders heading and keeps URL stable', async ({ page }) => {
        await stubCosts(page)
        await page.route('**/api/v1/runs/456*', async (route) => {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(runMock()),
          })
        })

        await page.goto('/runs/456')

        await expect(runDetailHeading(page)).toBeVisible()
        await expect(page).toHaveURL('/runs/456')
      })

      test('RG2: deferred run response → heading and skeleton visible immediately', async ({
        page,
      }) => {
        await stubCosts(page)
        let release: () => void = () => {}
        const gate = new Promise<void>((resolve) => {
          release = resolve
        })
        await page.route('**/api/v1/runs/456*', async (route) => {
          // Hold the response open so the view stays in its loading state.
          await gate
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(runMock()),
          })
        })

        await page.goto('/runs/456')

        // Title is independent of loading; skeleton is shown while the run loads.
        await expect(runDetailHeading(page)).toBeVisible()
        await expect(page.locator('.p-skeleton').first()).toBeVisible()

        release()
      })

      test('RG3: run error (500) → heading stays, error Message + Retry shown', async ({
        page,
      }) => {
        await stubCosts(page)
        await page.route('**/api/v1/runs/456*', async (route) => {
          await route.fulfill({
            status: 500,
            contentType: 'application/json',
            body: JSON.stringify({ error: { code: 'INTERNAL', message: 'boom' } }),
          })
        })

        await page.goto('/runs/456')

        await expect(runDetailHeading(page)).toBeVisible()
        // useRunDetail throws Error('Failed to load run'); the view renders it in
        // an error Message with a Retry button (RunDetailView.vue error branch).
        await expect(page.getByText('Failed to load run')).toBeVisible()
        await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()
      })

      test('RG4: run without a story title → heading falls back to "Run Detail"', async ({
        page,
      }) => {
        await stubCosts(page)
        await page.route('**/api/v1/runs/456*', async (route) => {
          await route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(runMock()),
          })
        })

        await page.goto('/runs/456')

        await expect(runDetailHeading(page)).toBeVisible()
        await expect(page).toHaveURL('/runs/456')
      })
    })

    test('should render Approvals view at /approvals', async ({ page }) => {
      await page.goto('/approvals')

      await expect(page.locator('h1')).toHaveText('Approvals')
      await expect(page).toHaveURL('/approvals')
    })
  })

  test.describe('Navigation tests', () => {
    test.beforeEach(async ({ page }) => {
      // Mock authenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: '1',
            email: 'test@test.com',
            name: 'Test User',
            role: 'user',
          }),
        })
      })

      // Mock Dashboard API calls
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
    })

    test('should navigate between routes using sidebar links (Dashboard → Projects)', async ({
      page,
    }) => {
      // Start at dashboard (redesign greets the user)
      await page.goto('/')
      await expect(page.locator('h1')).toHaveText(/Welcome back,/)

      // Navigate to Projects using sidebar button
      const sidebar = page.locator('aside')
      await sidebar.getByRole('button', { name: 'Projects' }).click()
      await expect(page).toHaveURL('/projects')
      await expect(page.locator('h1')).toHaveText('Projects')
    })
  })

  test.describe('Auth guard integration (unauthenticated)', () => {
    test.beforeEach(async ({ page }) => {
      // Mock unauthenticated user
      await page.route('**/api/v1/auth/me', async (route) => {
        await route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: { code: 'UNAUTHORIZED', message: 'Unauthorized' } }),
        })
      })
    })

    test('should redirect /projects to /login when unauthenticated', async ({ page }) => {
      await page.goto('/projects')

      // Should be redirected to login with redirect param (login heading is "Sign in")
      await expect(page).toHaveURL('/login?redirect=/projects')
      await expect(page.locator('h1')).toHaveText('Sign in')
    })

    test('should redirect /approvals to /login when unauthenticated', async ({ page }) => {
      await page.goto('/approvals')

      // Should be redirected to login with redirect param (login heading is "Sign in")
      await expect(page).toHaveURL('/login?redirect=/approvals')
      await expect(page.locator('h1')).toHaveText('Sign in')
    })
  })
})
