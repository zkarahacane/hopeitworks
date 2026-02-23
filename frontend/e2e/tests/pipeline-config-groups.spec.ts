import { test, expect } from '@playwright/test'

const mockDefaultGroups = [
  {
    id: 'g-setup',
    name: 'Setup',
    steps: [
      {
        id: 's-setup-1',
        name: 'create-branch',
        action_type: 'git_branch',
        auto_approve: true,
        retry_policy: { max_retries: 0, retry_type: 'none' },
        config: { branch_pattern: 'feat/{story_key}-{slug}' },
      },
    ],
  },
  {
    id: 'g-dev',
    name: 'Development',
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
  },
  {
    id: 'g-review',
    name: 'Review',
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
  },
  {
    id: 'g-merge',
    name: 'Merge',
    steps: [
      {
        id: 's-merge-1',
        name: 'create-pr',
        action_type: 'git_pr',
        auto_approve: true,
        retry_policy: { max_retries: 1, retry_type: 'on-failure' },
        config: { title_template: 'feat({scope}): {summary}', target_branch: 'develop' },
      },
    ],
  },
  {
    id: 'g-delivery',
    name: 'Delivery',
    steps: [
      {
        id: 's-delivery-1',
        name: 'wait-ci',
        action_type: 'ci_poll',
        auto_approve: true,
        retry_policy: { max_retries: 3, retry_type: 'on-failure' },
      },
      {
        id: 's-delivery-2',
        name: 'notify',
        action_type: 'notification',
        auto_approve: true,
        retry_policy: { max_retries: 0, retry_type: 'none' },
        config: { message: 'Pipeline complete' },
      },
    ],
  },
]

const mockPipelineConfig = {
  project_id: 'proj-1',
  groups: mockDefaultGroups,
  updated_at: '2026-02-15T10:30:00Z',
}

const mockMultiGroupConfig = {
  project_id: 'proj-1',
  groups: [
    {
      id: 'g1',
      name: 'Group A',
      steps: [
        {
          id: 's1',
          name: 'step-a',
          action_type: 'agent_run',
          model: 'claude-sonnet-4-6',
          auto_approve: false,
          retry_policy: { max_retries: 1, retry_type: 'on-failure' },
        },
      ],
    },
    {
      id: 'g2',
      name: 'Group B',
      steps: [
        {
          id: 's2',
          name: 'step-b',
          action_type: 'git_branch',
          auto_approve: true,
          retry_policy: { max_retries: 0, retry_type: 'none' },
        },
      ],
    },
    {
      id: 'g3',
      name: 'Group C',
      steps: [
        {
          id: 's3',
          name: 'step-c',
          action_type: 'notification',
          auto_approve: true,
          retry_policy: { max_retries: 0, retry_type: 'none' },
        },
      ],
    },
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

function setupAuthMock(page: import('@playwright/test').Page) {
  return page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: '1',
        email: 'admin@test.com',
        name: 'Admin User',
        role: 'admin',
      }),
    })
  })
}

function setupProjectMock(page: import('@playwright/test').Page) {
  return page.route('**/api/v1/projects/proj-1', async (route) => {
    if (route.request().url().includes('/pipeline')) return route.fallback()
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

function setupPipelineMock(
  page: import('@playwright/test').Page,
  config: typeof mockPipelineConfig,
) {
  return page.route('**/api/v1/projects/proj-1/pipeline', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(config),
      })
    } else if (route.request().method() === 'PUT') {
      const body = route.request().postDataJSON()
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...config,
          groups: body.groups,
          updated_at: new Date().toISOString(),
        }),
      })
    }
  })
}

test.describe('Pipeline Config — Groups', () => {
  test.describe('Groups display', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page)
      await setupProjectMock(page)
      await setupPipelineMock(page, mockPipelineConfig)
    })

    test('groups are displayed instead of flat steps', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      // Groups should be visible
      const groups = page.getByTestId('pipeline-group-card')
      await expect(groups).toHaveCount(5)

      // Each group should show its name
      const groupNames = page.getByTestId('group-name')
      await expect(groupNames).toHaveCount(5)

      // Steps should be nested inside groups, not as a flat list
      // Verify steps are within group containers
      for (const groupCard of await groups.all()) {
        const stepsContainer = groupCard.getByTestId('group-steps')
        await expect(stepsContainer).toBeVisible()
      }
    })
  })

  test.describe('Add group', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page)
      await setupProjectMock(page)
      await setupPipelineMock(page, mockPipelineConfig)
    })

    test('can add a new group', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      const groupsBefore = page.getByTestId('pipeline-group-card')
      await expect(groupsBefore).toHaveCount(5)

      // Click "Add Group" button
      await page.getByTestId('add-group-btn').click()

      // A new group should appear
      const groupsAfter = page.getByTestId('pipeline-group-card')
      await expect(groupsAfter).toHaveCount(6)

      // Save button should be enabled (dirty state)
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })
  })

  test.describe('Add steps of various types', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page)
      await setupProjectMock(page)
      await setupPipelineMock(page, mockMultiGroupConfig)
    })

    test('can add steps of type git_branch within a group', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      // Click "Add Step" in the first group
      await page.getByTestId('add-step-to-group').first().click()

      // Dialog should appear
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      // Fill step name
      await page.getByTestId('step-name-input').fill('new-branch-step')

      // Select git_branch type
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Create Git Branch' }).click()

      // git_branch specific fields should appear
      await expect(page.getByTestId('branch-pattern-input')).toBeVisible()

      // Model selector should NOT be visible for git_branch
      await expect(page.getByTestId('agent-select')).not.toBeVisible()

      // Submit
      await page.getByTestId('add-step-submit').click()

      // Save button should be enabled
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('can add steps of type git_pr within a group', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-to-group').first().click()
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      await page.getByTestId('step-name-input').fill('new-pr-step')

      // Select git_pr type
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Create Pull Request' }).click()

      // git_pr specific fields should appear
      await expect(page.getByTestId('title-template-input')).toBeVisible()
      await expect(page.getByTestId('target-branch-input')).toBeVisible()
      await expect(page.getByTestId('draft-toggle')).toBeVisible()

      // Model selector should NOT be visible for git_pr
      await expect(page.getByTestId('agent-select')).not.toBeVisible()

      await page.getByTestId('add-step-submit').click()
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('can add steps of type notification within a group', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-to-group').first().click()
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      await page.getByTestId('step-name-input').fill('new-notify-step')

      // Select notification type
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Send Notification' }).click()

      // notification specific fields should appear
      await expect(page.getByTestId('notification-message-input')).toBeVisible()

      // Model selector should NOT be visible for notification
      await expect(page.getByTestId('agent-select')).not.toBeVisible()

      await page.getByTestId('add-step-submit').click()
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('can add steps of type human within a group', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-to-group').first().click()
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      await page.getByTestId('step-name-input').fill('new-human-step')

      // Select human type
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Human Task' }).click()

      // human specific fields should appear
      await expect(page.getByTestId('human-message-input')).toBeVisible()
      await expect(page.getByTestId('human-instructions-input')).toBeVisible()

      // Model selector should NOT be visible for human
      await expect(page.getByTestId('agent-select')).not.toBeVisible()

      await page.getByTestId('add-step-submit').click()
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })

    test('step config fields change when type changes', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      await page.getByTestId('add-step-to-group').first().click()
      await expect(page.getByText('Add Pipeline Step')).toBeVisible()

      await page.getByTestId('step-name-input').fill('changing-step')

      // Default type is agent_run — agent selector should be visible
      await expect(page.getByTestId('agent-select')).toBeVisible()

      // Switch to git_branch
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Create Git Branch' }).click()

      // agent_run fields should be gone, git_branch fields should appear
      await expect(page.getByTestId('agent-select')).not.toBeVisible()
      await expect(page.getByTestId('branch-pattern-input')).toBeVisible()

      // Switch to notification
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Send Notification' }).click()

      // git_branch fields should be gone, notification fields should appear
      await expect(page.getByTestId('branch-pattern-input')).not.toBeVisible()
      await expect(page.getByTestId('notification-message-input')).toBeVisible()

      // Switch back to agent_run
      await page.getByTestId('action-type-select').click()
      await page.getByRole('option', { name: 'Agent Run' }).click()

      // notification fields should be gone, agent selector should reappear
      await expect(page.getByTestId('notification-message-input')).not.toBeVisible()
      await expect(page.getByTestId('agent-select')).toBeVisible()
    })
  })

  test.describe('Delete group', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page)
      await setupProjectMock(page)
      await setupPipelineMock(page, mockMultiGroupConfig)
    })

    test('can delete a group with confirmation', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      const groupsBefore = page.getByTestId('pipeline-group-card')
      await expect(groupsBefore).toHaveCount(3)

      // Verify we see all 3 group names
      await expect(page.getByTestId('group-name').nth(0)).toHaveText('Group A')
      await expect(page.getByTestId('group-name').nth(1)).toHaveText('Group B')
      await expect(page.getByTestId('group-name').nth(2)).toHaveText('Group C')

      // Click delete on the second group (Group B)
      await page.getByTestId('remove-group').nth(1).click()

      // Confirmation dialog should appear
      await expect(page.getByText('Confirm Removal')).toBeVisible()
      await expect(page.getByText(/Remove group "Group B"/)).toBeVisible()

      // Confirm deletion
      await page.getByRole('button', { name: /yes|accept|ok/i }).click()

      // Group B should be removed
      const groupsAfter = page.getByTestId('pipeline-group-card')
      await expect(groupsAfter).toHaveCount(2)

      // Remaining groups should be Group A and Group C
      await expect(page.getByTestId('group-name').nth(0)).toHaveText('Group A')
      await expect(page.getByTestId('group-name').nth(1)).toHaveText('Group C')

      // Save button should be enabled
      await expect(page.getByTestId('save-config-btn')).toBeEnabled()
    })
  })

  test.describe('Default pipeline config', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page)
      await setupProjectMock(page)
      await setupPipelineMock(page, mockPipelineConfig)
    })

    test('default config has 5 groups in expected order', async ({ page }) => {
      await page.goto('/projects/proj-1/pipeline')

      const groups = page.getByTestId('pipeline-group-card')
      await expect(groups).toHaveCount(5)

      // Verify group names in order
      const groupNames = page.getByTestId('group-name')
      await expect(groupNames.nth(0)).toHaveText('Setup')
      await expect(groupNames.nth(1)).toHaveText('Development')
      await expect(groupNames.nth(2)).toHaveText('Review')
      await expect(groupNames.nth(3)).toHaveText('Merge')
      await expect(groupNames.nth(4)).toHaveText('Delivery')

      // Verify Setup group contains a git_branch step
      const setupGroup = groups.nth(0)
      await expect(setupGroup.getByTestId('action-type-tag').first()).toHaveText('git_branch')

      // Verify Development group contains an agent_run step
      const devGroup = groups.nth(1)
      await expect(devGroup.getByTestId('action-type-tag').first()).toHaveText('agent_run')

      // Verify Review group contains an agent_run step
      const reviewGroup = groups.nth(2)
      await expect(reviewGroup.getByTestId('action-type-tag').first()).toHaveText('agent_run')

      // Verify Merge group contains a git_pr step
      const mergeGroup = groups.nth(3)
      await expect(mergeGroup.getByTestId('action-type-tag').first()).toHaveText('git_pr')

      // Verify Delivery group contains ci_poll and notification steps
      const deliveryGroup = groups.nth(4)
      const deliveryTags = deliveryGroup.getByTestId('action-type-tag')
      await expect(deliveryTags).toHaveCount(2)
      await expect(deliveryTags.nth(0)).toHaveText('ci_poll')
      await expect(deliveryTags.nth(1)).toHaveText('notification')
    })
  })
})
