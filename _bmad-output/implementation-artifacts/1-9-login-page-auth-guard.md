# Story 1.9: [FRONT] Login page + auth guard

Status: ready-for-dev

## Story

As a user,
I want to log in with email and password,
so that I can access protected features.

## Acceptance Criteria (BDD)

**AC1: Unauthenticated redirect**
- **Given** user is not authenticated
- **When** they navigate to a protected route (e.g. `/`)
- **Then** they are redirected to `/login`

**AC2: Successful login**
- **Given** user submits valid credentials on `/login`
- **When** POST /api/v1/auth/login returns 200 with User object
- **Then** user state is stored in Pinia auth store and user is redirected to `/` (dashboard)

**AC3: Failed login**
- **Given** user submits invalid credentials on `/login`
- **When** POST /api/v1/auth/login returns 401
- **Then** an error message displays below the form

**AC4: Form validation**
- **Given** form fields are present
- **When** user blurs with invalid input (empty email, bad email format, password < 8 chars)
- **Then** inline validation errors show via vee-validate + zod

**AC5: Auth persistence check**
- **Given** user reloads the page
- **When** the app initializes
- **Then** GET /api/v1/auth/me is called to restore session from httpOnly cookie

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create Pinia auth store `frontend/src/stores/auth.ts` (AC: #2, #3, #5)
  - [ ] Install pinia and add to main.ts if not already present
  - [ ] Install vee-validate, @vee-validate/zod, zod
  - [ ] Define User type: `{ id: string; email: string; name: string }`
  - [ ] Define store state: `user: User | null`, `isAuthenticated: boolean`, `loading: boolean`, `error: string | null`
  - [ ] Implement `login(email, password)` action — POST /api/v1/auth/login, set user on success, set error on failure
  - [ ] Implement `logout()` action — POST /api/v1/auth/logout, clear user state
  - [ ] Implement `checkAuth()` action — GET /api/v1/auth/me, set user if 200, clear if 401
  - [ ] Use openapi-fetch client from `src/api/client.ts` (credentials: 'include')

- [ ] [FRONT] Task 2: Create `useAuth` composable `frontend/src/composables/useAuth.ts` (AC: #2, #3)
  - [ ] Wrap `useAuthStore()` for component consumption
  - [ ] Expose: `user`, `isAuthenticated`, `loading`, `error` as computed refs
  - [ ] Expose: `login(email, password)`, `logout()`, `checkAuth()` methods
  - [ ] Return type-safe interface

- [ ] [FRONT] Task 3: Create router guard `frontend/src/router/guards.ts` (AC: #1, #5)
  - [ ] Export `setupAuthGuard(router)` function
  - [ ] Register global `beforeEach` guard
  - [ ] On first navigation: call `useAuthStore().checkAuth()` once (lazy init)
  - [ ] If route requires auth (`meta.requiresAuth !== false`) and `!isAuthenticated` → redirect to `/login`
  - [ ] If route is `/login` and `isAuthenticated` → redirect to `/`
  - [ ] `/login` route has `meta: { requiresAuth: false }`

- [ ] [FRONT] Task 4: Update router `frontend/src/router/index.ts` (AC: #1)
  - [ ] Add `/login` route → `LoginView` (lazy loaded)
  - [ ] Add `meta: { requiresAuth: false }` to `/login` route
  - [ ] Set default `meta: { requiresAuth: true }` on all other routes
  - [ ] Call `setupAuthGuard(router)` before export

- [ ] [FRONT] Task 5: Create `LoginView.vue` `frontend/src/views/LoginView.vue` (AC: #2, #3, #4)
  - [ ] Use vee-validate `useForm` with zod schema for validation
  - [ ] Zod schema: `email` (z.string().min(1).email()), `password` (z.string().min(8))
  - [ ] PrimeVue components: InputText (email), Password (password), Button (submit), Message (error)
  - [ ] Bind fields with `useField` — show validation errors on blur
  - [ ] On submit: call `useAuth().login()`, redirect to `/` on success
  - [ ] Display API error from store via PrimeVue Message component
  - [ ] Centered card layout using Tailwind flex utilities, zero custom CSS
  - [ ] Disable submit button while loading

## Dev Notes

### Dependencies

**Story dependencies:** 1-7 (Vue scaffold), 1-8 (app shell layout)

**npm packages to install:**
```bash
npm install pinia vee-validate @vee-validate/zod zod
```

Note: `openapi-fetch` and generated API types should already be available from story 1-16. If not yet available, use a plain `fetch` wrapper as interim and mark for update.

### File Paths

| File | Action |
|------|--------|
| `frontend/src/stores/auth.ts` | Create |
| `frontend/src/composables/useAuth.ts` | Create |
| `frontend/src/router/guards.ts` | Create |
| `frontend/src/router/index.ts` | Update |
| `frontend/src/views/LoginView.vue` | Create |
| `frontend/src/main.ts` | Update (add Pinia) |

### Zod Schema

```typescript
import { z } from 'zod'

export const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email format'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

export type LoginFormValues = z.infer<typeof loginSchema>
```

### Auth Store Signature

```typescript
// frontend/src/stores/auth.ts
import { defineStore } from 'pinia'

interface User {
  id: string
  email: string
  name: string
}

interface AuthState {
  user: User | null
  loading: boolean
  error: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    loading: false,
    error: null,
  }),
  getters: {
    isAuthenticated: (state) => state.user !== null,
  },
  actions: {
    async login(email: string, password: string): Promise<boolean> { /* ... */ },
    async logout(): Promise<void> { /* ... */ },
    async checkAuth(): Promise<void> { /* ... */ },
  },
})
```

Alternatively, use `defineStore` with Composition API (setup function) — either style is fine, but be consistent with future stores.

### useAuth Composable Signature

```typescript
// frontend/src/composables/useAuth.ts
export function useAuth() {
  const store = useAuthStore()
  return {
    user: computed(() => store.user),
    isAuthenticated: computed(() => store.isAuthenticated),
    loading: computed(() => store.loading),
    error: computed(() => store.error),
    login: store.login,
    logout: store.logout,
    checkAuth: store.checkAuth,
  }
}
```

### Router Guard Logic

```typescript
// frontend/src/router/guards.ts
let authChecked = false

export function setupAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuthStore()

    // One-time session restore on first navigation
    if (!authChecked) {
      authChecked = true
      await auth.checkAuth()
    }

    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !auth.isAuthenticated) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }
    if (to.path === '/login' && auth.isAuthenticated) {
      return { path: '/' }
    }
  })
}
```

### PrimeVue Components Used

- `InputText` — email field
- `Password` — password field (with toggle visibility)
- `Button` — submit button (with `loading` prop)
- `Message` — error display (severity="error")

### API Endpoints

| Method | Path | Request Body | Response |
|--------|------|-------------|----------|
| POST | /api/v1/auth/login | `{ email, password }` | 200: User object + Set-Cookie (httpOnly JWT) |
| POST | /api/v1/auth/logout | (empty) | 204 |
| GET | /api/v1/auth/me | (none) | 200: User object / 401 |

The JWT cookie is httpOnly — frontend never reads/stores the token. The browser handles cookie send via `credentials: 'include'` on fetch.

### LoginView Layout

Centered card, no sidebar/header (login is outside the app shell):
```
+-------------------------------------------+
|                                           |
|         [hopeitworks logo/title]          |
|                                           |
|         +-------------------------+       |
|         | Email                   |       |
|         | [___________________]   |       |
|         | Password                |       |
|         | [___________________]   |       |
|         |                         |       |
|         | [  Sign In  ]          |       |
|         |                         |       |
|         | (error message here)    |       |
|         +-------------------------+       |
|                                           |
+-------------------------------------------+
```

### Style Conventions

- PrimeVue components for all form elements — no native HTML inputs
- Tailwind for layout only (flex, gap, padding, width)
- Zero `<style scoped>` blocks
- No custom CSS classes

### Testing Requirements

**Manual verification:**
1. Navigate to `/` while not authenticated -> redirected to `/login`
2. Submit empty form -> validation errors appear on blur
3. Submit invalid credentials -> error message from API appears
4. Submit valid credentials -> redirected to `/` with user in store
5. Reload page -> session restored via GET /auth/me
6. Navigate to `/login` while authenticated -> redirected to `/`

**Unit tests (Vitest) — to add in follow-up:**
- Auth store: login success, login failure, checkAuth, logout
- Router guard: redirect logic
- LoginView: form validation, submission

### References

- [Source: api/openapi.yaml — /auth/login, /auth/logout, /auth/me endpoints]
- [Source: _bmad-output/planning-artifacts/architecture.md — JWT httpOnly cookie flow]
- [Source: _bmad-output/planning-artifacts/architecture.md — Frontend package layout]
- [Source: _bmad-output/planning-artifacts/epics.md — Epic 1, Story 1.9]

## Dev Agent Record

## Change Log
