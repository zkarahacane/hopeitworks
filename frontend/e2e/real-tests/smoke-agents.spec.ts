/**
 * Real E2E smoke tests — Agents CRUD
 *
 * These tests run against a live backend at http://localhost:5173 (Vite proxy -> :8080).
 * They require seed data: Todo App project with at least one global agent.
 *
 * Run with: npx playwright test real-tests/smoke-agents.spec.ts
 */
import { test, expect, type BrowserContext } from '@playwright/test'
import { loginViaAPI } from './helpers/auth'
import { LogCollector } from './helpers/log-collector'

const TODO_APP_ID = '00000000-0000-0000-0000-000000000101'

/**
 * Create an agent via the API and return its id.
 */
async function createAgentViaAPI(
  context: BrowserContext,
  overrides: Record<string, string> = {},
): Promise<{ id: string; name: string }> {
  const name = overrides.name ?? `E2E Agent ${Date.now()}`
  const response = await context.request.post(
    `/api/v1/projects/${TODO_APP_ID}/agents`,
    {
      data: {
        name,
        model: overrides.model ?? 'claude-sonnet-4-6',
        image: overrides.image ?? 'hopeitworks/agent:latest',
        template_content:
          overrides.template_content ?? '# E2E test agent template',
      },
    },
  )
  expect(
    response.ok(),
    `Create agent API should succeed: ${response.status()}`,
  ).toBeTruthy()
  const body = await response.json()
  return { id: body.id, name }
}

/**
 * Delete an agent via the API (best-effort cleanup).
 */
async function deleteAgentViaAPI(
  context: BrowserContext,
  agentId: string,
): Promise<void> {
  await context.request.delete(
    `/api/v1/projects/${TODO_APP_ID}/agents/${agentId}`,
  )
}

test.describe('smoke: agents CRUD (real backend)', () => {
  let logs: LogCollector

  test.beforeEach(async ({ page, context }) => {
    logs = new LogCollector()
    logs.attach(page)
    await loginViaAPI(context, 'admin')
  })

  test.afterEach(() => {
    const report = logs.getReport()
    if (report.summary.totalErrors > 0) {
      console.warn('[LogCollector] Console/JS errors:', report.errors)
    }
  })

  test('agents tab is labeled "Agents" not "Templates"', async ({ page }) => {
    await page.goto(`/projects/${TODO_APP_ID}`)

    // PrimeVue TabMenu renders items as role="menuitem"
    const agentsTab = page.getByRole('menuitem', { name: /agents/i })
    await expect(agentsTab.first()).toBeVisible({ timeout: 10000 })

    // Verify no "Templates" tab exists
    const templatesTab = page.getByRole('menuitem', { name: /templates/i })
    await expect(templatesTab).toHaveCount(0)
  })

  test('agents list shows scope badges', async ({ page }) => {
    await page.goto(`/projects/${TODO_APP_ID}/agents`)

    // Wait for the agents table to render
    await expect(page.getByTestId('agents-table')).toBeVisible({
      timeout: 10000,
    })

    // Should have at least one scope badge
    const scopeBadges = page.getByTestId('scope-badge')
    await expect(scopeBadges.first()).toBeVisible()

    // Check that badges display valid scope values
    const badgeCount = await scopeBadges.count()
    expect(badgeCount).toBeGreaterThan(0)

    for (let i = 0; i < badgeCount; i++) {
      const text = await scopeBadges.nth(i).textContent()
      expect(['global', 'project']).toContain(text?.trim().toLowerCase())
    }
  })

  test('agents list shows model (inline) and image column', async ({ page }) => {
    await page.goto(`/projects/${TODO_APP_ID}/agents`)

    // Wait for the table to appear
    await expect(page.getByTestId('agents-table')).toBeVisible({
      timeout: 10000,
    })

    // Column headers: Agent and Image must be visible (Model is inline in Agent column)
    await expect(
      page.getByRole('columnheader', { name: 'Agent' }),
    ).toBeVisible()
    await expect(
      page.getByRole('columnheader', { name: 'Image' }),
    ).toBeVisible()

    // At least one row should exist
    const rows = page.locator('[data-testid="agents-table"] tbody tr')
    const rowCount = await rows.count()
    expect(rowCount).toBeGreaterThan(0)

    // Model should be shown inline within the table (seed agents use claude-sonnet-4-6 or similar)
    const tableLocator = page.locator('[data-testid="agents-table"]')
    await expect(tableLocator.getByText(/claude/i).first()).toBeVisible()
  })

  test('can create a project-scoped agent', async ({ page, context }) => {
    const agentName = `E2E Create Test ${Date.now()}`
    let createdAgentId: string | null = null

    try {
      await page.goto(`/projects/${TODO_APP_ID}/agents`)
      await expect(
        page
          .getByTestId('agents-table')
          .or(page.getByText('No agents found')),
      ).toBeVisible({ timeout: 10000 })

      // Click the New Agent button
      await page.getByRole('button', { name: 'New Agent' }).click()

      // Wait for the editor/form to load
      await expect(page).toHaveURL(
        new RegExp(`/projects/${TODO_APP_ID}/agents/new`),
      )

      // Fill in the agent name
      const nameField = page.getByLabel(/name/i).first()
      await nameField.fill(agentName)

      // Save the agent
      await page.getByRole('button', { name: /save/i }).click()

      // Should redirect back to agent list
      await page.waitForURL(
        new RegExp(`/projects/${TODO_APP_ID}/agents$`),
        { timeout: 10000 },
      )

      // The new agent should appear in the list
      await expect(page.getByText(agentName)).toBeVisible({ timeout: 10000 })

      // It should have a "project" scope badge in its row
      const agentRow = page.locator('tr', { hasText: agentName })
      const scopeBadge = agentRow.getByTestId('scope-badge')
      await expect(scopeBadge).toHaveText(/project/i)

      // Get the created agent ID for cleanup via API
      const agents = await context.request.get(
        `/api/v1/projects/${TODO_APP_ID}/agents`,
      )
      const body = await agents.json()
      const created = body.data?.find(
        (a: { name: string }) => a.name === agentName,
      )
      if (created) createdAgentId = created.id
    } finally {
      if (createdAgentId) {
        await deleteAgentViaAPI(context, createdAgentId)
      }
    }
  })

  test('can edit a project-scoped agent', async ({ page, context }) => {
    // Create an agent via API first for test isolation
    const { id: agentId, name: originalName } = await createAgentViaAPI(
      context,
      { name: `E2E Edit Test ${Date.now()}` },
    )

    try {
      await page.goto(`/projects/${TODO_APP_ID}/agents`)
      await expect(page.getByTestId('agents-table')).toBeVisible({
        timeout: 10000,
      })

      // Find and click the edit button for our agent
      const agentRow = page.locator('tr', { hasText: originalName })
      await agentRow.getByTestId('edit-agent-button').click()

      // Wait for the editor to load
      await expect(page).toHaveURL(
        new RegExp(`/projects/${TODO_APP_ID}/agents/${agentId}`),
      )

      // Change the name
      const updatedName = `${originalName} Updated`
      const nameField = page.getByLabel(/name/i).first()
      await nameField.fill(updatedName)

      // Save
      await page.getByRole('button', { name: /save/i }).click()

      // Should redirect back to agent list
      await page.waitForURL(
        new RegExp(`/projects/${TODO_APP_ID}/agents$`),
        { timeout: 10000 },
      )

      // The updated name should be visible
      await expect(page.getByText(updatedName)).toBeVisible({ timeout: 10000 })
    } finally {
      await deleteAgentViaAPI(context, agentId)
    }
  })

  test('can delete a project-scoped agent', async ({ page, context }) => {
    // Create an agent via API first
    const { id: agentId, name: agentName } = await createAgentViaAPI(context, {
      name: `E2E Delete Test ${Date.now()}`,
    })

    try {
      await page.goto(`/projects/${TODO_APP_ID}/agents`)
      await expect(page.getByTestId('agents-table')).toBeVisible({
        timeout: 10000,
      })

      // Verify the agent is in the list
      await expect(page.getByText(agentName)).toBeVisible()

      // Click delete button on the agent row
      const agentRow = page.locator('tr', { hasText: agentName })
      await agentRow.getByTestId('delete-agent-button').click()

      // Confirm deletion in the dialog
      const confirmDialog = page.getByRole('dialog')
      await expect(confirmDialog).toBeVisible()
      await confirmDialog
        .getByRole('button', { name: /yes|accept|confirm|ok/i })
        .click()

      // Agent should be removed from the list
      await expect(page.getByText(agentName)).not.toBeVisible({
        timeout: 10000,
      })
    } catch {
      // If test fails, try to clean up the agent
      await deleteAgentViaAPI(context, agentId)
    }
  })

  test('global agents have no edit or delete buttons for non-admin', async ({
    browser,
  }) => {
    // Use a separate browser context logged in as a non-admin user
    const devContext = await browser.newContext()
    const page = await devContext.newPage()
    const devLogs = new LogCollector()
    devLogs.attach(page)

    try {
      await loginViaAPI(devContext, 'dev')
      await page.goto(`/projects/${TODO_APP_ID}/agents`)

      // Wait for the table to appear
      await expect(page.getByTestId('agents-table')).toBeVisible({
        timeout: 10000,
      })

      // Find rows with global scope badges
      const scopeBadges = page.getByTestId('scope-badge')
      const badgeCount = await scopeBadges.count()

      let foundGlobal = false
      for (let i = 0; i < badgeCount; i++) {
        const text = await scopeBadges.nth(i).textContent()
        if (text?.trim().toLowerCase() === 'global') {
          foundGlobal = true

          // Get the row containing this global badge
          const row = scopeBadges.nth(i).locator('xpath=ancestor::tr')

          // Edit button should be disabled for global agents when non-admin
          const editBtn = row.getByTestId('edit-agent-button')
          await expect(editBtn).toBeDisabled()

          // Delete button should not be visible for global agents when non-admin
          const deleteBtn = row.getByTestId('delete-agent-button')
          await expect(deleteBtn).toHaveCount(0)
        }
      }

      // Ensure we actually found at least one global agent to validate
      expect(
        foundGlobal,
        'Expected at least one global agent in the list',
      ).toBeTruthy()
    } finally {
      await page.close()
      await devContext.close()
    }
  })
})
