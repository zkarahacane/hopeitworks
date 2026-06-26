/**
 * Screenshot capture utility (NOT a test assertion).
 *
 * Captures fresh UI screenshots into ../docs/screenshots/ for the project README.
 * Run against a live, seeded stack (seed.sql data — project Todo App with runs/costs):
 *   npx playwright test --config playwright.e2e-real.config.ts capture-screenshots
 *
 * Seed IDs are fixed in backend/testdata/seed.sql.
 */
import { test } from '@playwright/test'
import { loginViaUI } from './helpers/auth'

const PROJECT = '00000000-0000-0000-0000-000000000101' // Todo App
const EPIC = '00000000-0000-0000-0000-000000000201' // Foundation
const RUN = '00000000-0000-0000-0000-000000000501' // completed run with full step timeline
const OUT = '../docs/screenshots'

const SHOTS: Array<[string, string]> = [
  ['02-dashboard', '/'],
  ['07-board', `/projects/${PROJECT}/board`],
  ['09-dag-view', `/projects/${PROJECT}/epics/${EPIC}/dag`],
  ['10-pipeline', `/projects/${PROJECT}/pipeline`],
  ['11-agents', `/projects/${PROJECT}/agents`],
  ['12-costs', `/projects/${PROJECT}/costs`],
  ['03-run-detail', `/runs/${RUN}`],
  ['13-approvals', '/approvals'],
]

test.use({ viewport: { width: 1440, height: 900 } })

test('capture README screenshots', async ({ page }) => {
  // Utility, not an assertion — skipped in normal e2e runs to avoid churn.
  // Regenerate with: CAPTURE=1 npx playwright test --config playwright.e2e-real.config.ts capture-screenshots
  test.skip(!process.env.CAPTURE, 'set CAPTURE=1 to regenerate README screenshots')
  test.setTimeout(120000)
  await loginViaUI(page, 'admin')

  for (const [name, path] of SHOTS) {
    try {
      // domcontentloaded (NOT networkidle — SSE keeps a connection open forever)
      await page.goto(path, { waitUntil: 'domcontentloaded', timeout: 30000 })
      // give charts / Vue Flow DAG / Monaco time to render
      await page.waitForTimeout(2500)
      await page.screenshot({
        path: `${OUT}/${name}.png`,
        animations: 'disabled',
        timeout: 20000,
      })
       
      console.log(`OK ${name} <- ${path}`)
    } catch (e) {
       
      console.log(`FAIL ${name} <- ${path}: ${(e as Error).message.split('\n')[0]}`)
    }
  }
})
