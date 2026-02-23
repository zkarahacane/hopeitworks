import { test, expect } from '@playwright/test'
import { loginViaAPI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const TODO_APP_ID = '00000000-0000-0000-0000-000000000101'

test.describe('smoke: navigation', () => {
  test('dashboard loads after login', async ({ page, context }) => {
    const logs = new LogCollector()
    logs.attach(page)

    await loginViaAPI(context, 'admin')
    await page.goto('/')

    await expect(page).not.toHaveURL(/\/login/)
    // Dashboard should have some visible content
    await expect(page.locator('body')).not.toBeEmpty()
    // No JS errors expected on dashboard load
    const report = logs.getReport()
    expect(report.errors).toHaveLength(0)
  })

  test('sidebar navigation to projects', async ({ page, context }) => {
    await loginViaAPI(context, 'admin')
    await page.goto('/')

    // Sidebar uses PrimeVue Button components, not <a> links
    await page.getByRole('button', { name: /projects/i }).first().click()

    await expect(page).toHaveURL(/\/projects/)
  })

  test('sidebar navigation to approvals', async ({ page, context }) => {
    await loginViaAPI(context, 'admin')
    await page.goto('/')

    // Sidebar uses PrimeVue Button components, not <a> links
    await page.getByRole('button', { name: /approvals/i }).first().click()

    await expect(page).toHaveURL(/\/approvals/)
  })

  test('project tabs navigation', async ({ page, context }) => {
    await loginViaAPI(context, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}`)

    // Verify we land on the project detail page
    await expect(page).toHaveURL(new RegExp(TODO_APP_ID))

    // PrimeVue TabMenu renders items as <a> with role="menuitem" inside a role="menubar"
    // Navigate to board tab
    const boardTab = page.getByRole('menuitem', { name: /board/i })
    await boardTab.first().click()
    await expect(page).toHaveURL(new RegExp(`${TODO_APP_ID}/board`))

    // Navigate to pipeline tab
    const pipelineTab = page.getByRole('menuitem', { name: /pipeline/i })
    await pipelineTab.first().click()
    await expect(page).toHaveURL(new RegExp(`${TODO_APP_ID}/pipeline`))

    // Navigate to agents tab
    const agentsTab = page.getByRole('menuitem', { name: /agents/i })
    await agentsTab.first().click()
    await expect(page).toHaveURL(new RegExp(`${TODO_APP_ID}/agents`))
  })

  test('deep link to project board', async ({ page, context }) => {
    await loginViaAPI(context, 'admin')
    await page.goto(`/projects/${TODO_APP_ID}/board`)

    // Board page should load without redirect to login
    await expect(page).not.toHaveURL(/\/login/)
    await expect(page).toHaveURL(new RegExp(`${TODO_APP_ID}/board`))
    await expect(page.locator('body')).not.toBeEmpty()
  })

  test('browser back/forward works', async ({ page, context }) => {
    await loginViaAPI(context, 'admin')

    // Navigate through a sequence of pages
    await page.goto('/')
    await page.goto('/projects')
    await page.goto(`/projects/${TODO_APP_ID}`)

    // Go back to /projects
    await page.goBack()
    await expect(page).toHaveURL(/\/projects$/)

    // Go back to dashboard
    await page.goBack()
    await expect(page).toHaveURL(/localhost:\d+\/?$|\/dashboard/)

    // Go forward to /projects
    await page.goForward()
    await expect(page).toHaveURL(/\/projects$/)
  })
})
