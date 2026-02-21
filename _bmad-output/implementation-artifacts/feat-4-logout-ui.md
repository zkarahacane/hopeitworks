# Story feat-4: [FRONT] Logout button and UI flow

Status: ready-for-dev

## Story

As an authenticated user,
I want a logout option accessible from the user menu in the header,
so that I can securely end my session and be redirected to the login page.

## Acceptance Criteria (BDD)

**AC1: Logout menu item triggers session termination**
- **Given** I am authenticated and viewing any page inside AppShell
- **When** I click the user menu button (`data-testid="user-menu-button"`) and select "Logout"
- **Then** the frontend calls `POST /api/v1/auth/logout` with `credentials: 'include'`
- **And** the auth store clears `user` and `error` state (`authStore.user = null`)
- **And** I am redirected to `/login`

**AC2: Logout button is accessible from the header user menu**
- **Given** I am on any authenticated route
- **When** I look at the AppHeader
- **Then** I see a user icon button (`data-testid="user-menu-button"`) in the top-right area
- **And** clicking it opens a PrimeVue `Menu` popup containing a "Logout" item with icon `pi pi-sign-out`

**AC3: Backend errors do not block logout**
- **Given** the `POST /api/v1/auth/logout` call returns a network error or non-2xx response
- **When** I click Logout
- **Then** the error is silently swallowed (best-effort logout)
- **And** the auth store state is still cleared
- **And** I am still redirected to `/login`

**AC4: After logout, protected routes redirect to /login**
- **Given** I have just logged out
- **When** I navigate to any route with `meta: { requiresAuth: true }` (e.g. `/`, `/projects`)
- **Then** the Vue Router auth guard redirects me to `/login?redirect=<original-path>`

**AC5: Display user identity in the user menu**
- **Given** I am authenticated and the auth store contains `user.name` and `user.email`
- **When** I open the user menu from the header
- **Then** the menu shows the user's name and email as a non-clickable header item above the Logout entry

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Enhance `AppHeader.vue` to display user identity in the user menu (AC: #2, #5)
  - [ ] Add a non-clickable `MenuItem` at the top of `menuItems` with the user's `name` and `email` (use `label` with a template or a separator)
  - [ ] Import `useAuthStore` (already imported) and bind `authStore.user.name` / `authStore.user.email` reactively — change `menuItems` from a static `const` to a `computed<MenuItem[]>` using `computed()` from Vue
  - [ ] Keep `data-testid="user-menu-button"` on the trigger button and `data-testid="user-menu"` on the `<Menu>` component (already present — do not remove)

- [ ] [FRONT] Task 2: Verify `authStore.logout()` implementation is correct and robust (AC: #1, #3)
  - [ ] Confirm `stores/auth.ts` `logout()` action: calls `POST /api/v1/auth/logout` with `credentials: 'include'`, swallows errors via `.catch(() => {})`, and sets `this.user = null` and `this.error = null` — this is already implemented; no change needed if correct
  - [ ] Confirm the `AppHeader.vue` `command` handler calls `await authStore.logout()` then `router.push('/login')` — this is already implemented; verify it matches AC3 (error-swallowing happens inside `logout()`, so the `command` handler is safe)

- [ ] [FRONT] Task 3: Verify auth guard behavior post-logout (AC: #4)
  - [ ] In `router/guards.ts`, confirm `setupAuthGuard` redirects unauthenticated users to `/login?redirect=<path>` — already implemented; no change needed
  - [ ] Confirm `authChecked` flag does not prevent re-checking after logout: the guard re-evaluates `auth.isAuthenticated` (a computed from `auth.user`) on every navigation — since `user` is set to `null` in `logout()`, the guard correctly redirects after logout without needing to re-call `checkAuth`

- [ ] [FRONT] Task 4: Add E2E tests for the logout flow (AC: #1, #3, #4)
  - [ ] Create or extend `frontend/e2e/tests/app-shell.spec.ts` with a new `describe('Logout flow')` block
  - [ ] Test: open user menu, click Logout, mock `POST /api/v1/auth/logout` returning 204 — assert redirect to `/login`
  - [ ] Test: mock `POST /api/v1/auth/logout` returning 500 — assert logout still redirects to `/login` (error-swallowing)
  - [ ] Test: after logout, navigate to `/` — mock `GET /api/v1/auth/me` returning 401 — assert redirect to `/login?redirect=/`
  - [ ] Test: open user menu — assert user name and email are visible in the menu header

- [ ] [FRONT] Task 5: Add unit test for `authStore.logout()` (AC: #1, #3)
  - [ ] Create `frontend/src/stores/__tests__/auth.spec.ts` (or add to existing if present)
  - [ ] Test: `logout()` clears `user` and `error` even when fetch throws
  - [ ] Test: `logout()` calls `POST /api/v1/auth/logout` with `credentials: 'include'`

## Dev Notes

### Dependencies (feat-1-logout-api)

This story depends on `feat-1-logout-api` which provides the backend `POST /auth/logout` endpoint. The OpenAPI contract is already defined in `api/openapi.yaml`:

```yaml
/auth/logout:
  post:
    operationId: logoutUser
    summary: Log out the current user
    tags: [auth]
    responses:
      "204":
        description: Logged out successfully
```

The backend invalidates the httpOnly session cookie. The frontend call must use `credentials: 'include'` to send the cookie. **Do NOT start this story until `feat-1-logout-api` is merged to develop.**

### File Paths

| File | Action |
|------|--------|
| `frontend/src/ui/layout/AppHeader.vue` | Modify — make `menuItems` reactive, add user identity header item |
| `frontend/src/stores/auth.ts` | Verify only — `logout()` already implemented |
| `frontend/src/composables/useAuth.ts` | No change needed |
| `frontend/src/router/guards.ts` | Verify only — guard already redirects unauthenticated users |
| `frontend/e2e/tests/app-shell.spec.ts` | Extend — add logout flow E2E tests |
| `frontend/src/stores/__tests__/auth.spec.ts` | Create or extend — unit tests for `logout()` |

### Component Structure

The logout UI is fully contained in `AppHeader.vue`. The component already has:
- PrimeVue `Menu` component with `:popup="true"` bound to `ref userMenu`
- A trigger `Button` with `@click="toggleUserMenu"` and `data-testid="user-menu-button"`
- A `menuItems` array with the Logout item wired to `authStore.logout()` + `router.push('/login')`

The only required change to `AppHeader.vue` is converting `menuItems` from a static `const` to a `computed<MenuItem[]>` so that user identity can be shown reactively:

```typescript
// Before (static)
const menuItems: MenuItem[] = [
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      router.push('/login')
    },
  },
]

// After (reactive)
const menuItems = computed<MenuItem[]>(() => [
  {
    label: authStore.user?.name ?? 'User',
    items: [
      { separator: true },
    ],
    // Use a separator + sub-label pattern or a disabled item for identity display
  },
  {
    label: authStore.user?.email ?? '',
    disabled: true,
    class: 'text-surface-500 text-sm',
  },
  { separator: true },
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      router.push('/login')
    },
  },
])
```

Note: PrimeVue `Menu` supports `disabled: true` items and `separator: true` items natively. Use these for the identity display — no custom template needed.

### PrimeVue Components Used

| Component | Import | Usage |
|-----------|--------|-------|
| `Button` | `primevue/button` | User menu trigger (`pi pi-user` icon, text+rounded) |
| `Menu` | `primevue/menu` | Popup menu with `MenuItem[]` model |
| `MenuItem` | `primevue/menuitem` | Type for menu items (disabled, separator, command) |

PrimeVue `Menu` `MenuItem` interface supports:
- `label: string` — display text
- `icon: string` — PrimeIcons class
- `command: (event) => void` — click handler
- `disabled: boolean` — non-interactive item
- `separator: boolean` — renders a `<hr>` divider

### Testing Requirements

**E2E (Playwright) — extend `frontend/e2e/tests/app-shell.spec.ts`:**

```typescript
test.describe('Logout flow', () => {
  test.beforeEach(async ({ page }) => {
    // Mock authenticated user
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: '1', email: 'test@test.com', name: 'Test User', role: 'member' }),
      })
    })
    await page.goto('/')
  })

  test('should logout and redirect to /login on success', async ({ page }) => {
    await page.route('**/api/v1/auth/logout', async (route) => {
      await route.fulfill({ status: 204 })
    })
    // Re-mock /me as 401 after logout for guard check
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({ status: 401 })
    })
    await page.getByTestId('user-menu-button').click()
    await page.getByRole('menuitem', { name: 'Logout' }).click()
    await expect(page).toHaveURL('/login')
  })

  test('should logout even when backend returns 500', async ({ page }) => {
    await page.route('**/api/v1/auth/logout', async (route) => {
      await route.fulfill({ status: 500 })
    })
    await page.getByTestId('user-menu-button').click()
    await page.getByRole('menuitem', { name: 'Logout' }).click()
    await expect(page).toHaveURL('/login')
  })

  test('should display user name and email in user menu', async ({ page }) => {
    await page.getByTestId('user-menu-button').click()
    await expect(page.getByTestId('user-menu')).toContainText('Test User')
    await expect(page.getByTestId('user-menu')).toContainText('test@test.com')
  })
})
```

**Unit tests — `frontend/src/stores/__tests__/auth.spec.ts`:**

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('logout()', () => {
    it('clears user and error state on success', async () => {
      global.fetch = vi.fn().mockResolvedValue({ ok: true })
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'member' }
      await store.logout()
      expect(store.user).toBeNull()
      expect(store.error).toBeNull()
    })

    it('clears user state even when fetch throws', async () => {
      global.fetch = vi.fn().mockRejectedValue(new Error('Network error'))
      const store = useAuthStore()
      store.user = { id: '1', email: 'a@b.com', name: 'A', role: 'member' }
      await store.logout()
      expect(store.user).toBeNull()
    })

    it('calls POST /api/v1/auth/logout with credentials include', async () => {
      const fetchMock = vi.fn().mockResolvedValue({ ok: true })
      global.fetch = fetchMock
      const store = useAuthStore()
      await store.logout()
      expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      })
    })
  })
})
```

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-21 | Story SM | Initial story creation |
