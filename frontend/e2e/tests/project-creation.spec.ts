import { test, expect } from '@playwright/test'

test.describe('Project Creation', () => {
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
        }),
      })
    })

    // Mock empty project list by default
    await page.route('**/api/v1/projects*', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [],
            pagination: { total: 0, page: 1, per_page: 20 },
          }),
        })
      } else {
        await route.continue()
      }
    })
  })

  test('clicking New Project button opens creation dialog', async ({ page }) => {
    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()

    // Dialog should be visible with form fields
    await expect(page.getByText('Create Project')).toBeVisible()
    await expect(page.locator('#project-name')).toBeVisible()
    await expect(page.locator('#project-description')).toBeVisible()
  })

  test('creation dialog includes pipeline configuration fields', async ({ page }) => {
    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Pipeline configuration section and fields should be visible
    await expect(page.getByText('Pipeline Configuration')).toBeVisible()
    await expect(page.locator('#project-repo-url')).toBeVisible()
    await expect(page.locator('#project-git-provider')).toBeVisible()
    await expect(page.locator('#project-agent-runtime')).toBeVisible()
    await expect(page.locator('#project-default-model')).toBeVisible()
  })

  test('submitting empty form does not proceed', async ({ page }) => {
    let postRequestMade = false
    await page.route('**/api/v1/projects', async (route) => {
      if (route.request().method() === 'POST') {
        postRequestMade = true
        await route.continue()
      } else {
        await route.continue()
      }
    })

    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Click Create without filling the form (use exact match to avoid matching empty state button)
    await page.getByRole('button', { name: 'Create', exact: true }).click()

    // Wait a bit to ensure no POST request is made
    await page.waitForTimeout(1000)

    // Validation should prevent the POST request
    expect(postRequestMade).toBe(false)

    // Dialog should still be visible (not closed)
    await expect(page.getByText('Create Project')).toBeVisible()
  })

  test('submitting without repo URL shows validation error and blocks submission', async ({
    page,
  }) => {
    let postRequestMade = false
    await page.route('**/api/v1/projects', async (route) => {
      if (route.request().method() === 'POST') {
        postRequestMade = true
        await route.continue()
      } else {
        await route.continue()
      }
    })

    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Fill name but not repo URL
    await page.locator('#project-name').fill('My Project')
    await page.getByRole('button', { name: 'Create', exact: true }).click()

    // Wait for validation
    await page.waitForTimeout(1000)

    // Validation error should appear
    await expect(page.getByText('Repository URL is required')).toBeVisible()

    // No API call should be made
    expect(postRequestMade).toBe(false)

    // Dialog should still be open
    await expect(page.getByText('Create Project')).toBeVisible()
  })

  test('successful form submission calls API with repo_url and navigates', async ({ page }) => {
    const createdProject = {
      id: 'p-new-1',
      name: 'My New Project',
      description: 'A test project',
      repo_url: 'https://github.com/org/repo',
      git_provider: 'github',
      agent_runtime: 'docker',
      owner_id: '1',
      created_at: '2026-02-16T10:00:00Z',
      updated_at: '2026-02-16T10:00:00Z',
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let capturedBody: any = null

    // Mock POST /projects
    await page.route('**/api/v1/projects', async (route) => {
      if (route.request().method() === 'POST') {
        capturedBody = route.request().postDataJSON()
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(createdProject),
        })
      } else {
        await route.continue()
      }
    })

    await page.goto('/projects')

    await page.getByRole('button', { name: 'New Project' }).click()
    await expect(page.getByText('Create Project')).toBeVisible()

    // Fill in the form including pipeline config fields
    await page.locator('#project-name').fill('My New Project')
    await page.locator('#project-description').fill('A test project')
    await page.locator('#project-repo-url').fill('https://github.com/org/repo')

    // Submit (use exact match to avoid matching empty state button)
    await page.getByRole('button', { name: 'Create', exact: true }).click()

    // Should navigate to project detail page
    await expect(page).toHaveURL('/projects/p-new-1')

    // Verify the POST body includes repo_url
    expect(capturedBody).toBeDefined()
    expect(capturedBody.repo_url).toBe('https://github.com/org/repo')
    expect(capturedBody.git_provider).toBe('github')
    expect(capturedBody.agent_runtime).toBe('docker')
  })

  test('empty state CTA opens creation dialog', async ({ page }) => {
    await page.goto('/projects')

    // Empty state should be visible
    await expect(page.getByText('No projects yet')).toBeVisible()

    // Click the CTA button
    await page.getByRole('button', { name: 'Create your first project' }).click()

    // Dialog should open
    await expect(page.getByText('Create Project')).toBeVisible()
    await expect(page.locator('#project-name')).toBeVisible()
  })
})
