import { test, expect } from './fixtures'

// ── Pipeline config — transition policy + guards ──────────────────────────────
// The stage refonte adds two per-stage controls to the pipeline editor:
//   - a transition policy Select (auto = advances alone, manual = waits for a Go,
//     gate = HITL validation);
//   - a Guards editor where probes (log_silence / wallclock / cost_batch) are
//     configured with a threshold and an on_fail policy (halt-gate by default).
// Editing either marks the config dirty (enables Save) and is persisted via PUT.
//
// Fully mocked (see ./fixtures): GET/PUT /projects/proj-1/pipeline.

const mockPipelineConfig = {
  project_id: 'proj-1',
  groups: [
    {
      id: 'g-dev',
      name: 'Development',
      transition: 'auto',
      steps: [
        {
          id: 's-dev-1',
          name: 'implement',
          action_type: 'agent_run',
          model: 'claude-opus-4-6',
          auto_approve: false,
          retry_policy: { max_retries: 2, retry_type: 'on-failure' },
        },
      ],
      guards: [],
    },
    {
      id: 'g-review',
      name: 'Review',
      transition: 'gate',
      steps: [
        {
          id: 's-review-1',
          name: 'code-review',
          action_type: 'agent_run',
          model: 'claude-sonnet-4-6',
          auto_approve: false,
          retry_policy: { max_retries: 1, retry_type: 'on-failure' },
        },
      ],
      // A stage already carrying one guard so we can assert it renders.
      guards: [{ kind: 'log_silence', threshold: 120, on_fail: 'halt-gate' }],
    },
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

function setupAuthMock(page: import('@playwright/test').Page) {
  return page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ id: '1', email: 'admin@test.com', name: 'Admin User', role: 'admin' }),
    })
  })
}

function setupProjectMock(page: import('@playwright/test').Page) {
  return page.route('**/api/v1/projects/proj-1', async (route) => {
    if (route.request().url().includes('/pipeline') || route.request().url().includes('/agents')) {
      return route.fallback()
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'proj-1',
        name: 'Test Project',
        owner_id: 'u1',
        created_at: '2026-01-01T00:00:00Z',
        updated_at: '2026-01-01T00:00:00Z',
      }),
    })
  })
}

/** Capture the PUT body so we can assert what was persisted. */
function setupPipelineMock(
  page: import('@playwright/test').Page,
  onPut: (body: { groups: typeof mockPipelineConfig.groups }) => void,
) {
  return page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockPipelineConfig),
      })
    } else if (route.request().method() === 'PUT') {
      const body = route.request().postDataJSON()
      onPut(body)
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...mockPipelineConfig,
          groups: body.groups,
          updated_at: new Date().toISOString(),
        }),
      })
    }
  })
}

test.describe('Pipeline config — transitions & guards', () => {
  test('each stage exposes a transition policy with auto/manual/gate options', async ({ page }) => {
    await setupAuthMock(page)
    await setupProjectMock(page)
    await setupPipelineMock(page, () => {})

    await page.goto('/projects/proj-1/pipeline')

    await expect(page.getByTestId('pipeline-group-card')).toHaveCount(2)

    // One transition Select per stage.
    const selects = page.getByTestId('transition-select')
    await expect(selects).toHaveCount(2)

    // The hint spells out what each policy means.
    await expect(page.getByTestId('transition-hint').first()).toContainText('auto = avance seule')

    // The three policies are offered.
    await selects.first().click()
    await expect(page.getByRole('option', { name: 'Auto' })).toBeVisible()
    await expect(page.getByRole('option', { name: 'Manual' })).toBeVisible()
    await expect(page.getByRole('option', { name: 'Gate' })).toBeVisible()
  })

  test('setting a stage to manual marks dirty and persists the policy', async ({ page }) => {
    await setupAuthMock(page)
    await setupProjectMock(page)
    let putBody: { groups: typeof mockPipelineConfig.groups } | null = null
    await setupPipelineMock(page, (body) => {
      putBody = body
    })

    await page.goto('/projects/proj-1/pipeline')

    await expect(page.getByTestId('save-config-btn')).toBeDisabled()

    // Set the first stage (Development) from auto → manual.
    await page.getByTestId('transition-select').first().click()
    await page.getByRole('option', { name: 'Manual' }).click()

    // Editing the policy enables Save.
    await expect(page.getByTestId('save-config-btn')).toBeEnabled()

    await page.getByTestId('save-config-btn').click()
    await expect(page.getByText('Configuration saved')).toBeVisible()

    // The persisted payload carries the new policy on the first stage.
    expect(putBody).not.toBeNull()
    expect(putBody!.groups[0]!.transition).toBe('manual')
  })

  test('renders an existing guard and lets an admin add a guard to a stage', async ({ page }) => {
    await setupAuthMock(page)
    await setupProjectMock(page)
    let putBody: { groups: typeof mockPipelineConfig.groups } | null = null
    await setupPipelineMock(page, (body) => {
      putBody = body
    })

    await page.goto('/projects/proj-1/pipeline')

    // The Development stage starts with no guards (empty hint); Review has one.
    const groups = page.getByTestId('pipeline-group-card')
    await expect(groups.nth(0).getByTestId('guard-empty')).toBeVisible()
    await expect(groups.nth(1).getByTestId('guard-row')).toHaveCount(1)

    // Add a guard to the Development stage.
    await groups.nth(0).getByTestId('add-guard').click()

    const devGuardRow = groups.nth(0).getByTestId('guard-row')
    await expect(devGuardRow).toHaveCount(1)

    // A fresh guard defaults to log_silence / halt-gate; the on_fail Select offers
    // the halt-gate, fail and retry policies.
    await devGuardRow.getByTestId('guard-on-fail-select').click()
    await expect(page.getByRole('option', { name: 'Halt-gate' })).toBeVisible()
    await expect(page.getByRole('option', { name: 'Fail' })).toBeVisible()
    await expect(page.getByRole('option', { name: 'Retry' })).toBeVisible()
    // Close the overlay without changing the value.
    await page.getByRole('option', { name: 'Halt-gate' }).click()

    await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    await page.getByTestId('save-config-btn').click()
    await expect(page.getByText('Configuration saved')).toBeVisible()

    // The persisted payload added a guard to the first stage.
    expect(putBody).not.toBeNull()
    expect(putBody!.groups[0]!.guards).toHaveLength(1)
    expect(putBody!.groups[0]!.guards![0]!.kind).toBe('log_silence')
    expect(putBody!.groups[0]!.guards![0]!.on_fail).toBe('halt-gate')
  })

  test('changing a guard kind re-homes its threshold and persists', async ({ page }) => {
    await setupAuthMock(page)
    await setupProjectMock(page)
    let putBody: { groups: typeof mockPipelineConfig.groups } | null = null
    await setupPipelineMock(page, (body) => {
      putBody = body
    })

    await page.goto('/projects/proj-1/pipeline')

    // The Review stage's existing guard is log_silence. Switch it to cost_batch.
    const reviewGuard = page.getByTestId('pipeline-group-card').nth(1).getByTestId('guard-row')
    await reviewGuard.getByTestId('guard-kind-select').click()
    await page.getByRole('option', { name: 'Cost batch' }).click()

    // Unit label flips to USD for a cost guard.
    await expect(reviewGuard.getByTestId('guard-unit')).toHaveText('USD')

    await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    await page.getByTestId('save-config-btn').click()
    await expect(page.getByText('Configuration saved')).toBeVisible()

    expect(putBody).not.toBeNull()
    expect(putBody!.groups[1]!.guards![0]!.kind).toBe('cost_batch')
  })
})
