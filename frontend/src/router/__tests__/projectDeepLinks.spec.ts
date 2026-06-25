import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'

// The api client imports the configured router singleton and calls router.afterEach() at
// module load, creating an import cycle with the route table's guards. Stubbing it lets us
// import the REAL `routes` table without that side-effect firing on an undefined router.
vi.mock('@/api/client', () => ({ apiClient: {} }))

const { routes } = await import('../index')

/**
 * #293 — deep-links /projects/:id/overview and /projects/:id/notifications fell into
 * the catch-all 404 because no child route declared them. Overview's canonical URL is
 * /projects/:id ('' child) and Notifications' is /projects/:id/settings/notifications.
 *
 * These tests navigate the REAL route table (the exported `routes`, single source of
 * truth). router.push follows the redirects (router.resolve does not), so currentRoute
 * after push proves the legacy deep-links land on the canonical named routes, the URL is
 * canonical, and the existing tabs keep matching (no regression into the catch-all).
 */
function createTestRouter(): Router {
  return createRouter({ history: createMemoryHistory(), routes })
}

async function navigate(router: Router, path: string) {
  await router.push(path)
  await router.isReady()
  return router.currentRoute.value
}

describe('#293 project deep-link routes', () => {
  let router: Router

  beforeEach(() => {
    router = createTestRouter()
  })

  // RG5: legacy /projects/:id/overview redirects to the canonical Overview (/projects/:id)
  it('redirects /projects/p1/overview to the canonical project-overview (/projects/p1)', async () => {
    const route = await navigate(router, '/projects/p1/overview')
    expect(route.name).toBe('project-overview')
    expect(route.path).toBe('/projects/p1')
    expect(route.params.id).toBe('p1')
  })

  // RG6: legacy /projects/:id/notifications redirects to settings/notifications
  it('redirects /projects/p1/notifications to project-notifications (settings/notifications)', async () => {
    const route = await navigate(router, '/projects/p1/notifications')
    expect(route.name).toBe('project-notifications')
    expect(route.path).toBe('/projects/p1/settings/notifications')
    expect(route.params.id).toBe('p1')
  })

  // RG1/RG2: direct access to the canonical Overview URL stays Overview
  it('keeps the canonical /projects/p1 on project-overview', async () => {
    const route = await navigate(router, '/projects/p1')
    expect(route.name).toBe('project-overview')
    expect(route.params.id).toBe('p1')
  })

  // RG3/RG4: direct access to the canonical Notifications URL stays Notifications
  it('keeps the canonical /projects/p1/settings/notifications on project-notifications', async () => {
    const route = await navigate(router, '/projects/p1/settings/notifications')
    expect(route.name).toBe('project-notifications')
    expect(route.params.id).toBe('p1')
  })

  it('preserves a different :id through both redirects', async () => {
    expect((await navigate(router, '/projects/abc-123/overview')).path).toBe('/projects/abc-123')
    expect((await navigate(router, '/projects/abc-123/notifications')).path).toBe(
      '/projects/abc-123/settings/notifications',
    )
  })

  // RG8: non-regression — every other project tab still resolves to its own route,
  // never the catch-all.
  it.each([
    ['/projects/p1', 'project-overview'],
    ['/projects/p1/board', 'project-board'],
    ['/projects/p1/runs', 'project-runs'],
    ['/projects/p1/pipeline', 'project-pipeline'],
    ['/projects/p1/agents', 'project-agents'],
    ['/projects/p1/environment', 'project-environment'],
    ['/projects/p1/costs', 'project-costs'],
    ['/projects/p1/settings', 'project-settings'],
    ['/projects/p1/settings/notifications', 'project-notifications'],
  ])('keeps %s on %s (not the catch-all)', async (path, name) => {
    const route = await navigate(router, path)
    expect(route.name).toBe(name)
    expect(route.name).not.toBe('not-found')
  })

  it('still routes a genuinely unknown project sub-path to the catch-all', async () => {
    const route = await navigate(router, '/projects/p1/does-not-exist')
    expect(route.name).toBe('not-found')
  })
})
