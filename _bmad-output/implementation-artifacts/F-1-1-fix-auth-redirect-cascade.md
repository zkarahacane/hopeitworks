# Story F-1.1: [FRONT] Fix auth redirect cascade on public pages

Status: ready-for-dev

## Story

As a user,
I want the app to not redirect me chaotically between pages,
so that I can actually use the application without being interrupted after a few seconds.

## Acceptance Criteria (BDD)

**AC1: Public pages stay stable**
- **Given** user is on `/login`, `/forgot-password`, or `/reset-password`
- **When** an API call returns 401
- **Then** no redirect occurs — the page remains stable

**AC2: Protected pages redirect once**
- **Given** user session expires while on a protected route
- **When** an API call returns 401
- **Then** user is redirected to `/login` exactly once (no bounce loop)

**AC3: SSE reconnection does not trigger redirect**
- **Given** user is authenticated and SSE connection drops
- **When** SSE reconnects and gets a 401
- **Then** only a single redirect to `/login` happens, not a cascade

**AC4: No redirect on initial page load of public routes**
- **Given** user navigates directly to `/login` via URL bar
- **When** the page loads
- **Then** the login form renders immediately, no flicker or redirect

## Tasks / Subtasks

### Task 1 — Fix `authMiddleware` in `frontend/src/api/client.ts`

The middleware at lines 5-16 redirects on any 401 regardless of the current route. Two problems:

1. **No public-route check:** `router.push({ name: 'login' })` is called unconditionally.
2. **No deduplication:** multiple concurrent 401 responses each fire `router.push`, causing a cascade.

**Changes:**

- Import a `PUBLIC_ROUTE_NAMES` constant (defined in `frontend/src/router/index.ts` or a new `frontend/src/router/constants.ts`) containing `['login', 'forgot-password', 'reset-password', 'not-found']`.
- Before calling `router.push`, check `router.currentRoute.value.meta.requiresAuth !== false`. If the current route is already public, skip the redirect entirely.
- Add a module-level `let redirecting = false` flag. Set it to `true` before calling `router.push`, reset it to `false` inside a `router.afterEach` hook (registered once at module load) so that only one redirect fires even when multiple 401s arrive in the same tick.

Resulting logic in `onResponse`:

```ts
if (response.status === 401) {
  const url = new URL(request.url, window.location.origin)
  if (!url.pathname.startsWith('/api/v1/auth/')) {
    const currentRoute = router.currentRoute.value
    const isPublic = currentRoute.meta.requiresAuth === false
    if (!isPublic && !redirecting) {
      redirecting = true
      await router.push({ name: 'login' })
    }
  }
}
```

Reset `redirecting` inside a one-time `router.afterEach` registered at module init:

```ts
router.afterEach(() => {
  redirecting = false
})
```

### Task 2 — Fix race condition in `frontend/src/router/guards.ts`

The `authChecked` flag at line 11 is module-level. On fast navigation, `checkAuth()` is awaited correctly; however if `checkAuth()` itself triggers a redirect (e.g. via the `authMiddleware` above), the guard fires again before `authChecked` is set, causing double evaluation.

**Changes:**

- Replace the boolean `authChecked` with a `Promise<void> | null` pattern so concurrent guard invocations wait on the same in-flight promise rather than calling `checkAuth()` multiple times:

```ts
let authCheckPromise: Promise<void> | null = null

export function setupAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()

    if (!authCheckPromise) {
      authCheckPromise = auth.checkAuth()
    }
    await authCheckPromise

    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !auth.isAuthenticated) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (to.path === '/login' && auth.isAuthenticated) {
      const redirect = to.query.redirect as string
      return { path: redirect || '/' }
    }
  })
}
```

This ensures `checkAuth()` runs exactly once regardless of how many guard invocations are triggered concurrently during initial page load.

### Task 3 — Suppress SSE connection when auth state is invalid (`frontend/src/composables/useSSE.ts`)

Currently `useSSE` (line 15) opens `EventSource` immediately and its `onerror` handler (lines 20-22) only sets `status.value = 'error'` — it never closes the connection. The browser auto-retries `EventSource` on error, firing repeated reconnects that may each produce a 401.

**Changes:**

- On `onerror`, call `es.close()` and set `status.value = 'closed'` if the auth store reports the user is unauthenticated. Import `useAuthStore` from `@/stores/auth`.
- Expose a `reconnect()` function that the caller can invoke after auth is restored (useful for future re-auth flow).

```ts
es.onerror = () => {
  const auth = useAuthStore()
  if (!auth.isAuthenticated) {
    es.close()
    status.value = 'closed'
  } else {
    status.value = 'error'
  }
}
```

This stops the reconnect loop that generates repeated 401 SSE requests while the user is logged out.

### Task 4 — Export public route names constant from router (`frontend/src/router/index.ts` or new `frontend/src/router/constants.ts`)

To avoid duplicating the list of public routes between the middleware and the guard, extract them as a single exported constant. Add to `frontend/src/router/index.ts` (or a new `frontend/src/router/constants.ts` if preferred):

```ts
export const PUBLIC_ROUTE_NAMES = new Set(['login', 'forgot-password', 'reset-password', 'not-found'])
```

This constant is used by `authMiddleware` in Task 1 and can be referenced by future auth-aware composables.

## Dev Notes

- Priority: P0 — this blocks testing of all other features
- The fix must NOT break the normal auth flow (redirect to login when session expires on protected pages)
- The `redirecting` flag in `client.ts` must be reset via `router.afterEach`, not via `setTimeout`, to avoid timing-dependent behaviour
- `authCheckPromise` in `guards.ts` is intentionally never reset to `null` after resolution — the session check is a one-time startup operation; the Pinia store handles auth state updates from that point on
- SSE `EventSource` does not go through `openapi-fetch`, so the `authMiddleware` does not cover it — the `onerror` fix in `useSSE.ts` is the only guard for SSE-triggered redirect loops
- No new dependencies required — all changes are within existing files
- After implementing, run `npm run type-check` and `npm run lint` to verify no regressions
