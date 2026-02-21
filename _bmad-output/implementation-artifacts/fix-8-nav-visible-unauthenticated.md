# Story fix-8: [FRONT] Hide navigation when user is not authenticated

Status: ready-for-dev

## Story

As an unauthenticated user visiting the login page,
I want the navigation sidebar and header to be hidden,
so that the UI does not expose application chrome before I have signed in.

## Acceptance Criteria (BDD)

**AC1: Navigation hidden on /login (unauthenticated)**
- **Given** the user is not authenticated (auth store `user` is null)
- **When** the user visits `/login`
- **Then** `AppHeader`, `AppSidebar`, the mobile bottom nav, and `AppStatusBar` are not rendered — only the login form is visible

**AC2: Navigation visible after successful login**
- **Given** the user was on the login page with no navigation visible
- **When** the user submits valid credentials and is redirected to a protected route
- **Then** the full `AppShell` (header, sidebar, status bar) is visible

**AC3: Navigation visible on all authenticated routes**
- **Given** the user is authenticated (`auth store user !== null`)
- **When** the user navigates to any protected route (e.g. `/`, `/projects`)
- **Then** `AppHeader`, `AppSidebar`, and `AppStatusBar` are rendered normally

**AC4: Login page uses a bare, full-screen layout**
- **Given** the user is on `/login`
- **Then** the page occupies the full viewport with no sidebar/header offset — the existing `LoginView.vue` centred layout (`flex min-h-screen items-center justify-center`) fills the screen without being constrained inside the shell's `<main>` element

**AC5: Navigation hidden on the 404 catch-all route when unauthenticated**
- **Given** the user is not authenticated
- **When** the user lands on an unknown route that maps to `NotFoundView`
- **Then** navigation chrome is not shown

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Conditionally render app chrome in `AppShell.vue` based on auth state (AC: #1, #2, #3, #4)
  - [ ] Import `useAuth` composable (or `useAuthStore`) in `AppShell.vue`
  - [ ] Add `const { isAuthenticated } = useAuth()` (uses the existing `useAuthStore` getter)
  - [ ] Wrap `<AppHeader>`, `<AppSidebar>`, the mobile bottom `<nav>`, and `<AppStatusBar>` with `v-if="isAuthenticated"`
  - [ ] When unauthenticated, render `<router-view />` directly inside the root `<div>` without the shell layout structure (no sidebar offset, no header height offset) — `LoginView.vue` already owns `min-h-screen` so it fills the viewport

- [ ] [FRONT] Task 2: Add Playwright E2E test for unauthenticated shell visibility (AC: #1, #5)
  - [ ] In `frontend/e2e/tests/login.spec.ts`, add a test: "should not render AppHeader or AppSidebar on /login when unauthenticated"
  - [ ] Mock `**/api/v1/auth/me` as 401
  - [ ] Navigate to `/login`
  - [ ] Assert `header` element is not visible (or not present)
  - [ ] Assert sidebar nav element is not visible (or not present)
  - [ ] Add a test: "should render AppHeader and AppSidebar after successful login"
  - [ ] Mock login 200 + `/auth/me` 200, submit credentials, assert header visible at `/`

- [ ] [FRONT] Task 3: Verify existing app-shell E2E tests still pass (AC: #2, #3)
  - [ ] Run `npm run test:e2e` in `frontend/` and confirm `app-shell.spec.ts` is green
  - [ ] Confirm `login.spec.ts` is green including new tests

## Dev Notes

### Dependencies

- `useAuth` composable: `frontend/src/composables/useAuth.ts` — exposes `isAuthenticated: ComputedRef<boolean>` sourced from `useAuthStore().isAuthenticated` (getter: `state.user !== null`)
- `useAuthStore`: `frontend/src/stores/auth.ts` — Options API store, `isAuthenticated` getter already defined
- Auth guard: `frontend/src/router/guards.ts` performs `auth.checkAuth()` on first navigation; by the time `AppShell` renders, `isAuthenticated` is already resolved — no additional async guard needed inside the component

### File Paths

| File | Change |
|------|--------|
| `frontend/src/ui/layout/AppShell.vue` | Add `isAuthenticated` check; conditionally render shell chrome |
| `frontend/e2e/tests/login.spec.ts` | Add two new E2E tests for shell visibility |

### Architecture / Implementation Details

**Current `AppShell.vue` template structure (lines 91-158):**

```html
<div class="flex h-screen flex-col overflow-hidden">
  <a href="#main-content" ...>Skip to main content</a>
  <AppHeader ... />
  <div class="flex min-h-0 flex-1">
    <AppSidebar ... />
    <main id="main-content" class="flex-1 overflow-auto bg-surface-100 p-4">
      <router-view />
    </main>
  </div>
  <!-- Mobile bottom nav (v-if="isMobile") -->
  <AppStatusBar v-if="!isMobile" />
  <Toast ... />
</div>
```

**Target structure after fix:**

```html
<div class="flex h-screen flex-col overflow-hidden">
  <template v-if="isAuthenticated">
    <a href="#main-content" ...>Skip to main content</a>
    <AppHeader ... />
    <div class="flex min-h-0 flex-1">
      <AppSidebar ... />
      <main id="main-content" class="flex-1 overflow-auto bg-surface-100 p-4">
        <router-view />
      </main>
    </div>
    <!-- Mobile bottom nav (v-if="isMobile") -->
    <AppStatusBar v-if="!isMobile" />
    <Toast ... />
  </template>

  <template v-else>
    <router-view />
  </template>
</div>
```

Key points:
- Using `<template v-if>` blocks avoids adding wrapper DOM nodes.
- The `<Toast>` component is inside the authenticated block intentionally — no toast notifications are expected in the unauthenticated state. If future requirements need toasts on the login page, move it outside the `v-if`.
- `LoginView.vue` already wraps itself in `<div class="flex min-h-screen items-center justify-center p-4">` so removing the shell's `<main>` wrapper causes no visual regression — it fills the viewport naturally.
- The skip-navigation link is intentionally hidden in unauthenticated mode (no navigable content to skip to).
- The `useKeyboard` shortcut (`[` to toggle sidebar) still fires when unauthenticated — this is harmless since the sidebar is not rendered, but if desired, the keyboard handler can be wrapped with an `isAuthenticated` check inside `useKeyboard`.

**Script addition in `AppShell.vue`:**

```typescript
import { useAuth } from '@/composables/useAuth'

const { isAuthenticated } = useAuth()
```

This uses the existing composable — no new state, no new API call.

### Testing Requirements

**Manual verification:**
1. Start the dev stack: `cd deploy && docker-compose up -d`
2. Open the app in the browser without being logged in — confirm no sidebar, no header
3. Log in with valid credentials — confirm sidebar and header appear immediately after redirect to `/`
4. Log out (if logout is wired in the header) — confirm sidebar disappears and the login page has no chrome
5. Navigate directly to a protected route while unauthenticated (e.g., `/projects`) — confirm the auth guard redirects to `/login` with no nav chrome

**E2E (Playwright):**

```bash
cd frontend && npm run test:e2e -- --grep "login"
cd frontend && npm run test:e2e -- --grep "App Shell"
```

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-21 | story-writer | Initial story created |
