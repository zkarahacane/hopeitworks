import { describe, it, expect, vi, beforeEach } from 'vitest'

// Mock router before importing client
const mockPush = vi.fn().mockResolvedValue(undefined)
const mockAfterEach = vi.fn()
const mockCurrentRoute = { value: { meta: { requiresAuth: true } } }

vi.mock('@/router', () => ({
  default: {
    push: (...args: unknown[]) => mockPush(...args),
    afterEach: (...args: unknown[]) => mockAfterEach(...args),
    currentRoute: mockCurrentRoute,
  },
}))

const mockAuthStore = {
  user: null as null | { id: string },
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mockAuthStore,
}))

// Mock openapi-fetch to capture the middleware
type MiddlewareFn = {
  onResponse: (ctx: { request: Request; response: Response }) => Promise<Response>
}

let capturedMiddleware: MiddlewareFn | null = null

vi.mock('openapi-fetch', () => ({
  default: () => ({
    use: (mw: MiddlewareFn) => { capturedMiddleware = mw },
    GET: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
    DELETE: vi.fn(),
  }),
}))

describe('authMiddleware', () => {
  beforeEach(async () => {
    capturedMiddleware = null
    mockAuthStore.user = null
    mockPush.mockReset()
    mockPush.mockResolvedValue(undefined)
    mockCurrentRoute.value.meta = { requiresAuth: true }

    // Re-import to trigger middleware registration
    vi.resetModules()
    await import('../client')
  })

  function makeRequest(url: string): Request {
    return new Request(url)
  }

  function makeResponse(status: number): Response {
    return new Response(null, { status })
  }

  it('clears auth.user and redirects to login on 401 from non-auth endpoint', async () => {
    mockAuthStore.user = { id: '1' }

    const request = makeRequest('http://localhost/api/v1/projects')
    const response = makeResponse(401)

    await capturedMiddleware!.onResponse({ request, response })

    expect(mockAuthStore.user).toBeNull()
    expect(mockPush).toHaveBeenCalledWith({ name: 'login' })
  })

  it('does NOT redirect on 401 from /api/v1/auth/ endpoints', async () => {
    const request = makeRequest('http://localhost/api/v1/auth/me')
    const response = makeResponse(401)

    await capturedMiddleware!.onResponse({ request, response })

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does NOT redirect on non-401 responses', async () => {
    const request = makeRequest('http://localhost/api/v1/projects')
    const response = makeResponse(200)

    await capturedMiddleware!.onResponse({ request, response })

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does NOT redirect when current route is public (requiresAuth: false)', async () => {
    mockCurrentRoute.value.meta = { requiresAuth: false }

    const request = makeRequest('http://localhost/api/v1/projects')
    const response = makeResponse(401)

    await capturedMiddleware!.onResponse({ request, response })

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('does NOT redirect twice if already redirecting', async () => {
    mockAuthStore.user = { id: '1' }

    const request = makeRequest('http://localhost/api/v1/projects')
    const response = makeResponse(401)

    // Fire two concurrent 401s
    await Promise.all([
      capturedMiddleware!.onResponse({ request, response }),
      capturedMiddleware!.onResponse({ request, response }),
    ])

    expect(mockPush).toHaveBeenCalledTimes(1)
  })

  it('returns the original response unchanged', async () => {
    const request = makeRequest('http://localhost/api/v1/projects')
    const response = makeResponse(200)

    const result = await capturedMiddleware!.onResponse({ request, response })

    expect(result).toBe(response)
  })
})
