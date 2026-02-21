# Story feat-6: [FRONT] Reset password pages (forgot + reset)

Status: ready-for-dev

## Story

As an unauthenticated user,
I want to request a password reset and set a new password via a secure link,
so that I can regain access to my account without admin intervention.

## Acceptance Criteria (BDD)

**AC1: Forgot Password — form submission**
- **Given** I am on `/forgot-password`
- **When** I enter a valid email address and click "Send reset link"
- **Then** the form calls `POST /auth/forgot-password` with the email
- **And** a success message is displayed regardless of whether the email is registered
- **And** the form is hidden (replaced by the success message)

**AC2: Forgot Password — validation**
- **Given** I am on `/forgot-password`
- **When** I submit the form with an empty or malformed email
- **Then** an inline validation error appears under the email field
- **And** no API call is made

**AC3: Forgot Password — navigation**
- **Given** I am on `/forgot-password`
- **When** I click "Back to login"
- **Then** I am redirected to `/login`

**AC4: Login page — "Forgot password?" link**
- **Given** I am on `/login`
- **When** I look at the form
- **Then** I can see a "Forgot password?" link
- **And** clicking it navigates me to `/forgot-password`

**AC5: Reset Password — successful submission**
- **Given** I arrive on `/reset-password?token=<valid_token>`
- **When** I enter a valid new password and matching confirmation, then submit
- **Then** `POST /auth/reset-password` is called with `{ token, password }`
- **And** on success I am redirected to `/login` with a query param `?reset=success`
- **And** the login page displays a success banner "Password reset successfully. Please sign in."

**AC6: Reset Password — validation**
- **Given** I am on `/reset-password`
- **When** I submit with mismatched passwords or a password shorter than 8 characters
- **Then** inline validation errors appear and no API call is made

**AC7: Reset Password — missing or invalid token**
- **Given** I navigate to `/reset-password` with no token query param
- **Then** an error message is displayed explaining the link is invalid or expired
- **And** a "Request a new link" button links back to `/forgot-password`

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `forgotPassword` and `resetPassword` actions to the auth store (AC: #1, #5)
  - [ ] Add `forgotPassword(email: string): Promise<boolean>` to `frontend/src/stores/auth.ts` — calls `POST /api/v1/auth/forgot-password`, always returns `true` (no error surfaced to user)
  - [ ] Add `resetPassword(token: string, password: string): Promise<boolean>` to `frontend/src/stores/auth.ts` — calls `POST /api/v1/auth/reset-password`, returns `true` on 200, `false` on 400/422
  - [ ] Expose both methods via `frontend/src/composables/useAuth.ts`

- [ ] [FRONT] Task 2: Create `ForgotPasswordView.vue` at `frontend/src/views/ForgotPasswordView.vue` (AC: #1, #2, #3)
  - [ ] Use `vee-validate` + `zod` for email validation (same pattern as `LoginView.vue`)
  - [ ] On submit success: hide form, show static success message (see wireframe)
  - [ ] On submit error (network): show inline `Message` with `severity="error"`
  - [ ] "Back to login" `RouterLink` to `/login`
  - [ ] No loading indicator needed when success state is shown — only on submission

- [ ] [FRONT] Task 3: Create `ResetPasswordView.vue` at `frontend/src/views/ResetPasswordView.vue` (AC: #5, #6, #7)
  - [ ] Read `token` from `route.query.token` on mount; if absent, render error state (see wireframe)
  - [ ] Two fields: new password (PrimeVue `Password` with `toggle-mask`, no strength feedback) + confirm password (`Password` with `toggle-mask`, no strength feedback)
  - [ ] Zod schema validates min length 8 and `passwords match` cross-field rule
  - [ ] On success: redirect to `/login?reset=success`
  - [ ] On API error (400/422): show inline `Message` with `severity="error"` (e.g., "Token expired or invalid")

- [ ] [FRONT] Task 4: Register routes in `frontend/src/router/index.ts` (AC: #1, #5)
  - [ ] Add `/forgot-password` route with `meta: { requiresAuth: false }`
  - [ ] Add `/reset-password` route with `meta: { requiresAuth: false }`
  - [ ] Both routes must redirect to `/` when the user is already authenticated (existing guard logic handles this automatically via `to.path === '/login'` pattern — verify it also redirects other public paths or update guard logic)

- [ ] [FRONT] Task 5: Modify `LoginView.vue` to add "Forgot password?" link and success banner (AC: #4, #5)
  - [ ] Add `RouterLink` "Forgot password?" below the password field, aligned to the right
  - [ ] Read `route.query.reset`; if `=== 'success'`, display a `Message` with `severity="success"` above the form: "Password reset successfully. Please sign in."

- [ ] [FRONT] Task 6: Unit tests (AC: #1, #2, #6)
  - [ ] `frontend/src/stores/__tests__/auth.spec.ts` — add tests for `forgotPassword` and `resetPassword` actions (mock `fetch`, cover success and error paths)
  - [ ] `frontend/src/composables/__tests__/useAuth.spec.ts` — verify new methods are exposed

## Dev Notes

### Dependencies (feat-3-reset-password-api)

The backend endpoints must exist before this story can be tested end-to-end:

- `POST /api/v1/auth/forgot-password` — body: `{ email: string }` — always returns 200 (no disclosure)
- `POST /api/v1/auth/reset-password` — body: `{ token: string, password: string }` — returns 200 on success, 400/422 on invalid/expired token

These endpoints are NOT yet in `api/openapi.yaml` — they will be added by feat-3. The frontend auth store calls the API directly via `fetch` (same pattern as the existing `login` action in `stores/auth.ts`) to avoid blocking on generated types. Once feat-3 merges and the spec is updated, the store can be migrated to `apiClient.POST(...)`.

### File Paths

| File | Action |
|------|--------|
| `frontend/src/views/ForgotPasswordView.vue` | Create |
| `frontend/src/views/ResetPasswordView.vue` | Create |
| `frontend/src/stores/auth.ts` | Modify — add `forgotPassword`, `resetPassword` actions |
| `frontend/src/composables/useAuth.ts` | Modify — expose new actions |
| `frontend/src/router/index.ts` | Modify — add two public routes |
| `frontend/src/views/LoginView.vue` | Modify — add link + success banner |
| `frontend/src/stores/__tests__/auth.spec.ts` | Create or extend |

### Zod Validation Schemas (forgot + reset forms)

**Forgot Password schema** (`ForgotPasswordView.vue`):

```typescript
const forgotSchema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
  }),
)
```

**Reset Password schema** (`ResetPasswordView.vue`):

```typescript
const resetSchema = toTypedSchema(
  z.object({
    password: z.string().min(8, 'Password must be at least 8 characters'),
    confirmPassword: z.string().min(1, 'Please confirm your password'),
  }).refine((data) => data.password === data.confirmPassword, {
    message: 'Passwords do not match',
    path: ['confirmPassword'],
  }),
)
```

### Page Layouts (ASCII wireframes)

**ForgotPasswordView — initial state:**

```
┌──────────────────────────────────────┐
│                                      │
│          hopeitworks                 │  ← h1, same as LoginView
│                                      │
│  Reset your password                 │  ← h2 or subtitle
│                                      │
│  Email                               │
│  ┌────────────────────────────────┐  │
│  │ you@example.com                │  │  ← InputText id="email"
│  └────────────────────────────────┘  │
│  <inline error if invalid>           │
│                                      │
│  ┌────────────────────────────────┐  │
│  │     Send reset link            │  │  ← Button type="submit" :loading
│  └────────────────────────────────┘  │
│                                      │
│  ← Back to login                     │  ← RouterLink to="/login"
│                                      │
└──────────────────────────────────────┘
```

**ForgotPasswordView — success state (form hidden, replaced by):**

```
┌──────────────────────────────────────┐
│                                      │
│          hopeitworks                 │
│                                      │
│  ╔══════════════════════════════╗    │
│  ║  Check your email            ║    │  ← Message severity="success"
│  ║  If an account exists for    ║    │
│  ║  that email, you will        ║    │
│  ║  receive a reset link.       ║    │
│  ╚══════════════════════════════╝    │
│                                      │
│  ← Back to login                     │
│                                      │
└──────────────────────────────────────┘
```

**ResetPasswordView — normal state:**

```
┌──────────────────────────────────────┐
│                                      │
│          hopeitworks                 │
│                                      │
│  Set new password                    │
│                                      │
│  New password                        │
│  ┌────────────────────────────────┐  │
│  │ ••••••••              [👁]     │  │  ← Password inputId="password" toggle-mask
│  └────────────────────────────────┘  │
│  <inline error if invalid>           │
│                                      │
│  Confirm new password                │
│  ┌────────────────────────────────┐  │
│  │ ••••••••              [👁]     │  │  ← Password inputId="confirmPassword" toggle-mask
│  └────────────────────────────────┘  │
│  <inline error if mismatch>          │
│                                      │
│  ┌────────────────────────────────┐  │
│  │     Set new password           │  │  ← Button type="submit" :loading
│  └────────────────────────────────┘  │
│                                      │
│  ╔══════════════════════════╗        │  ← Message v-if="error" severity="error"
│  ║ Token expired or invalid ║        │
│  ╚══════════════════════════╝        │
│                                      │
└──────────────────────────────────────┘
```

**ResetPasswordView — missing/invalid token state (shown immediately on mount):**

```
┌──────────────────────────────────────┐
│                                      │
│          hopeitworks                 │
│                                      │
│  ╔══════════════════════════════╗    │  ← Message severity="error" :closable="false"
│  ║  Invalid or expired link     ║    │
│  ║  This password reset link    ║    │
│  ║  is invalid or has expired.  ║    │
│  ╚══════════════════════════════╝    │
│                                      │
│  ┌────────────────────────────────┐  │
│  │   Request a new link           │  │  ← RouterLink styled as Button to="/forgot-password"
│  └────────────────────────────────┘  │
│                                      │
└──────────────────────────────────────┘
```

**LoginView modification — "Forgot password?" link placement:**

```
│  Password                            │
│  ┌────────────────────────────────┐  │
│  │ ••••••••              [👁]     │  │
│  └────────────────────────────────┘  │
│  <inline error>          Forgot password? →│  ← RouterLink, text-sm, aligned right
│                                      │
│  [Sign In button]                    │
│                                      │
│  ╔══════════════════╗                │  ← v-if="route.query.reset === 'success'"
│  ║  Password reset  ║                │    Message severity="success" above form
│  ║  successfully.   ║                │
│  ╚══════════════════╝                │
```

### PrimeVue Components Used

| Component | Import | Usage |
|-----------|--------|-------|
| `InputText` | `primevue/inputtext` | Email field in ForgotPasswordView |
| `Password` | `primevue/password` | Password fields in ResetPasswordView |
| `Button` | `primevue/button` | Submit buttons, "Request new link" |
| `Message` | `primevue/message` | Success/error feedback, `:closable="false"` |

All imports follow the existing pattern in `LoginView.vue`. Do NOT import `RouterLink` — it is globally registered by Vue Router.

### Router additions (public routes)

Add to `frontend/src/router/index.ts` after the `login` route:

```typescript
import ForgotPasswordView from '@/views/ForgotPasswordView.vue'
import ResetPasswordView from '@/views/ResetPasswordView.vue'

// Inside routes array, after the login route:
{
  path: '/forgot-password',
  name: 'forgot-password',
  component: ForgotPasswordView,
  meta: { requiresAuth: false },
},
{
  path: '/reset-password',
  name: 'reset-password',
  component: ResetPasswordView,
  meta: { requiresAuth: false },
},
```

Note: The existing auth guard in `frontend/src/router/guards.ts` redirects authenticated users away from `/login` specifically. It does NOT auto-redirect from other public pages. Since it is reasonable to allow authenticated users to access these pages (e.g., if they opened a stale reset link), no change to the guard is required. If desired, add an explicit redirect for authenticated users reaching `/forgot-password` or `/reset-password` — but this is optional for MVP.

### Login page modification (add "Forgot password?" link)

In `frontend/src/views/LoginView.vue`:

1. Add `useRoute` import (already present).
2. Replace the password `<div class="flex flex-col gap-1">` block to include the link:

```vue
<div class="flex flex-col gap-1">
  <div class="flex items-center justify-between">
    <label for="password" class="text-sm font-medium">Password</label>
    <RouterLink to="/forgot-password" class="text-sm">Forgot password?</RouterLink>
  </div>
  <Password
    inputId="password"
    v-model="password"
    :feedback="false"
    toggle-mask
    :invalid="!!passwordError"
    input-class="w-full"
    class="w-full"
  />
  <small v-if="passwordError" class="text-red-500">{{ passwordError }}</small>
</div>
```

3. Add success banner above the `<form>` tag:

```vue
<Message
  v-if="route.query.reset === 'success'"
  severity="success"
  :closable="false"
>
  Password reset successfully. Please sign in.
</Message>
```

### Auth store additions (`frontend/src/stores/auth.ts`)

Add to the `actions` object:

```typescript
async forgotPassword(email: string): Promise<boolean> {
  this.loading = true
  this.error = null
  try {
    await fetch('/api/v1/auth/forgot-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    })
    // Always return true — never disclose whether email is registered
    return true
  } catch {
    // Network error: still return true to avoid disclosure
    return true
  } finally {
    this.loading = false
  }
},

async resetPassword(token: string, password: string): Promise<boolean> {
  this.loading = true
  this.error = null
  try {
    const res = await fetch('/api/v1/auth/reset-password', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token, password }),
    })
    if (!res.ok) {
      const body = await res.json().catch(() => null)
      this.error = body?.error?.message ?? 'Token expired or invalid. Please request a new link.'
      return false
    }
    return true
  } catch {
    this.error = 'Network error. Please try again.'
    return false
  } finally {
    this.loading = false
  }
},
```

Expose in `frontend/src/composables/useAuth.ts`:

```typescript
export function useAuth() {
  const store = useAuthStore()
  return {
    // ... existing fields ...
    forgotPassword: store.forgotPassword.bind(store),
    resetPassword: store.resetPassword.bind(store),
  }
}
```

### Security notes (don't reveal if email exists)

- `forgotPassword` in the store ALWAYS returns `true` and displays the same success message, regardless of the API response code or network error. This prevents user enumeration attacks.
- The success message must be generic: "If an account exists for that email, you will receive a reset link shortly."
- Do NOT display different messages for "email not found" vs "email found".
- The `resetPassword` action MAY surface a generic error ("Token expired or invalid") because at that point the user already has the token — no enumeration risk.

### Testing Requirements

**Unit tests — `frontend/src/stores/__tests__/auth.spec.ts`:**

```typescript
describe('forgotPassword', () => {
  it('returns true on 200', async () => { /* mock fetch 200 */ })
  it('returns true on 404 (no disclosure)', async () => { /* mock fetch 404 */ })
  it('returns true on network error (no disclosure)', async () => { /* mock fetch throws */ })
  it('never sets error state', async () => { /* verify store.error stays null */ })
})

describe('resetPassword', () => {
  it('returns true on 200', async () => { /* mock fetch 200 */ })
  it('returns false and sets error on 400', async () => { /* mock fetch 400 */ })
  it('returns false and sets error on 422', async () => { /* mock fetch 422 */ })
  it('returns false and sets generic error on network failure', async () => { /* mock throws */ })
})
```

**No E2E tests required for this story** — the backend dependency (feat-3) must be merged first. Add E2E coverage once the full flow is testable against the real stack.

## Dev Agent Record

_This section is populated by the implementing agent._

## Change Log

| Date | Change | Author |
|------|--------|--------|
| 2026-02-21 | Story created | orchestrator |
