import { test as base, expect } from '@playwright/test'

/**
 * Shared E2E fixture that makes the default Playwright suite fully
 * backend-independent.
 *
 * The CI "Frontend → Run E2E tests" job runs `npm run test:e2e` with NO
 * backend behind the Vite dev server. The Vite proxy forwards any unmocked
 * `/api/v1/*` request to `localhost:8080`, which is not running in CI, so the
 * request fails with ECONNREFUSED and the UI hangs / times out.
 *
 * This fixture registers two catch-all routes on the page BEFORE each test
 * body (and before each spec's own `test.beforeEach`) runs:
 *
 *   1. `**​/api/v1/events/stream**` (SSE) → returns an empty, already-closed
 *      `text/event-stream` body so EventSource resolves immediately and never
 *      proxies to :8080 or hangs waiting for events.
 *
 *   2. `**​/api/v1/**` (everything else) → a benign `200` with an empty list
 *      envelope so any request a spec did not explicitly mock still resolves.
 *
 * Playwright matches routes in reverse registration order (last registered
 * wins). Because this fixture registers its catch-alls first, every spec's own
 * `page.route(...)` calls take precedence over them. Specs therefore keep their
 * specific mocks and only fall back to these stubs for otherwise-unmocked
 * calls.
 */
export const test = base.extend({
  page: async ({ page }, use) => {
    // Playwright matches routes in reverse registration order (last registered
    // wins). Register the broadest catch-all FIRST so the more specific SSE
    // stub below — and each spec's own `page.route(...)` registered after this
    // fixture — take precedence over it.

    // Catch-all for any API call a spec did not explicitly mock. Returns a
    // benign empty response that satisfies both list-envelope
    // (`{ data: [], pagination }`) and bare-array/object consumers.
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          pagination: { total: 0, page: 1, per_page: 20 },
        }),
      })
    })

    // SSE stream: respond with an empty, closed event-stream so EventSource
    // opens and closes without hanging or proxying to a live backend.
    // Registered AFTER the catch-all so it wins for event-stream URLs.
    await page.route('**/api/v1/events/stream**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        headers: {
          'Cache-Control': 'no-cache',
          Connection: 'keep-alive',
        },
        body: '',
      })
    })

    await use(page)
  },
})

export { expect }
