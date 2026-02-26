# Story F-3.2: [FRONT] Improve auth error messages and user feedback

Status: ready-for-dev

## Story

As a user,
I want clear feedback when authentication errors occur,
so that I understand why I was redirected or why an action failed.

## Acceptance Criteria (BDD)

**AC1: Session expired toast**
- **Given** user's session expires mid-session (a non-auth API call returns 401)
- **When** `authMiddleware` in `api/client.ts` intercepts the response and redirects to `/login`
- **Then** a Toast notification says "Session expired. Please log in again." before or during the redirect

**AC2: Login redirect reason displayed**
- **Given** user was redirected to `/login` with `?reason=session_expired` in the query string
- **When** `LoginView.vue` renders
- **Then** a `Message` component above the form says "Your session has expired. Please sign in again." (similar to the existing `?reset=success` message pattern)

**AC3: 403 Forbidden shows meaningful message**
- **Given** user tries to access a resource they don't have permission for
- **When** any API call returns 403
- **Then** `authMiddleware` (or a dedicated 403 handler in `api/client.ts`) triggers a Toast notification saying "Access denied. You don't have permission to perform this action."

**AC4: Network error feedback**
- **Given** the API is unreachable (fetch throws a network-level error, not an HTTP error)
- **When** any API call in `authMiddleware.onRequest` or a global fetch wrapper fails with a `TypeError`
- **Then** a Toast notification says "Unable to connect to the server. Please check your connection."

## Tasks / Subtasks

### Task 1 — Extend `authMiddleware` in `frontend/src/api/client.ts`

The current middleware only handles 401 with a silent redirect. Extend `onResponse` to:

1. On **401** (non-auth endpoint): push to `/login` with query `{ reason: 'session_expired' }` instead of bare `{ name: 'login' }`. Fire a Toast *before* the redirect using the PrimeVue `useToast()` service. Because `useToast` is a composable tied to the Vue app context, inject the toast instance at client initialisation time (export a `setToastService(toast: ToastServiceMethods)` function called from `AppShell.vue` or `main.ts`, then call it inside the middleware).

2. On **403**: fire a Toast with `severity: 'error'`, `summary: 'Access Denied'`, `detail: "You don't have permission to perform this action."`, `life: 5000`. Do NOT redirect.

3. Add an `onRequest` error hook (or wrap `fetch` globally) to catch `TypeError` (network failure) and fire a Toast with `severity: 'error'`, `summary: 'Connection Error'`, `detail: 'Unable to connect to the server. Please check your connection.'`, `life: 5000`.

File to edit: `frontend/src/api/client.ts`

### Task 2 — Wire Toast service into `client.ts` from `AppShell.vue`

`AppShell.vue` already imports `useToast` and holds the `<Toast>` component. It is the earliest authenticated component that has access to the Vue app context.

1. In `AppShell.vue` (`frontend/src/ui/layout/AppShell.vue`), on `onMounted`, call the `setToastService(toast)` function exported by `client.ts`.
2. Add a second `<Toast position="top-right" group="auth" />` element inside the `<template v-if="isAuthenticated">` block alongside the existing HITL Toast, so auth error toasts render independently from HITL toasts.
3. For the unauthenticated layout (`<template v-else>`), add a standalone `<Toast position="top-right" group="auth" />` so the session-expired toast fires even before the shell is fully shown.

File to edit: `frontend/src/ui/layout/AppShell.vue`

### Task 3 — Display redirect reason on `LoginView.vue`

`LoginView.vue` (`frontend/src/views/LoginView.vue`) already reads `route.query.reset` to show a success `Message`. Apply the same pattern for the new `reason` query param:

1. Read `route.query.reason` (type: `string | undefined`).
2. Define a computed `redirectMessage` that maps known reason values to human-readable strings:
   - `'session_expired'` → `"Your session has expired. Please sign in again."`
   - Any other non-empty value → `"Please sign in to continue."`
3. Render a `<Message severity="warn" :closable="false">{{ redirectMessage }}</Message>` above the form, conditionally on `redirectMessage` being non-null (same position as the existing `reset=success` Message).

File to edit: `frontend/src/views/LoginView.vue`

### Task 4 — Add unit tests

Add/extend tests for the changed modules:

1. `frontend/src/api/__tests__/client.spec.ts` (create if absent):
   - Mock `router.push` and the toast service.
   - Assert 401 on a non-auth URL → `router.push` called with `{ path: '/login', query: { reason: 'session_expired' } }` and toast fired.
   - Assert 401 on `/api/v1/auth/*` → neither redirect nor toast.
   - Assert 403 → toast fired with correct severity/summary, no redirect.

2. `frontend/src/views/__tests__/LoginView.spec.ts` (create if absent):
   - Mount `LoginView` with `route.query.reason = 'session_expired'`.
   - Assert the `Message` with `"Your session has expired"` is rendered.
   - Mount with no query params → assert Message is absent.

## Dev Notes

- Priority: P2
- Use PrimeVue Toast service (`useToast` / `ToastServiceMethods`) for all transient notifications; `life: 5000` for errors, `life: 0` (sticky) only for HITL.
- The `?reason=session_expired` query param on login redirect is the canonical mechanism for passing redirect context — keep it extensible for future reasons (e.g. `reason=forbidden`, `reason=logout`).
- `authMiddleware` in `client.ts` runs outside Vue component context so `useToast()` cannot be called directly there. Use the exported `setToastService()` initialisation pattern (module-level variable, set once from `AppShell.vue` on mount).
- The existing HITL Toast uses `group="hitl"` — use `group="auth"` for auth-related toasts to keep them separate and independently dismissible.
- Do not show raw HTTP status codes or internal error objects to users — map to plain English as specified above.
- The `guards.ts` redirect at line 26 (`{ path: '/login', query: { redirect: to.fullPath } }`) already preserves the intended destination via `?redirect=`. The new `?reason=` param is additive: both query params can coexist on the login URL.
