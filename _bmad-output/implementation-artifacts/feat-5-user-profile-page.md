# Story feat-5: [FRONT] User profile page

Status: ready-for-dev

## Story

As an authenticated user,
I want to view and edit my profile (name, email) and change my password,
so that I can keep my account information up to date without requiring admin intervention.

## Acceptance Criteria (BDD)

**AC1: Profile page loads with current user data**
- **Given** an authenticated user navigates to `/profile`
- **When** the page mounts
- **Then** GET `/api/v1/users/me` is called and the profile form is pre-filled with the user's current `name` and `email`
- **And** the user's `role` and `member_since` (formatted `created_at`) are displayed as read-only metadata

**AC2: Profile info update succeeds**
- **Given** the user edits their name or email and clicks "Save Changes"
- **When** PUT `/api/v1/users/me` returns 200
- **Then** a success Toast is shown with message "Profile updated"
- **And** the auth store's `user` is updated to reflect the new values

**AC3: Profile info validation errors**
- **Given** the user clears the name field or enters an invalid email
- **When** they blur the field or click "Save Changes"
- **Then** inline validation errors appear beneath the relevant field
- **And** the Save button remains disabled while the form is invalid

**AC4: Password change succeeds**
- **Given** the user fills in a valid current password and a new password (min 8 chars, confirmed)
- **When** they click "Update Password" and PUT `/api/v1/users/me/password` returns 204
- **Then** a success Toast is shown with message "Password updated"
- **And** the password fields are reset to empty

**AC5: Password change validation errors**
- **Given** the user submits the password form with an empty current password, a new password shorter than 8 characters, or mismatched confirmation
- **When** they blur the field or click "Update Password"
- **Then** inline validation errors appear beneath the relevant field
- **And** the Update Password button remains disabled while the form is invalid

**AC6: Profile page accessible from user menu**
- **Given** any authenticated user is on any page
- **When** they open the user menu in the header and click "My Profile"
- **Then** they are navigated to `/profile`

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `GET /users/me` and `PUT /users/me` to the OpenAPI spec (AC: #1, #2)
  - [ ] Edit `api/openapi.yaml` — add paths `/users/me` (GET) and `/users/me` (PUT) and `/users/me/password` (PUT) under the `users` tag
  - [ ] GET `/users/me`: no params, response 200 `User` schema (reuse existing `$ref`)
  - [ ] PUT `/users/me`: request body `UpdateMeRequest` (name?: string, email?: string), response 200 `User`
  - [ ] PUT `/users/me/password`: request body `ChangePasswordRequest` (current_password: string, new_password: string), response 204 no content
  - [ ] Add `UpdateMeRequest` and `ChangePasswordRequest` schemas to `components/schemas`
  - [ ] Regenerate frontend types: `cd frontend && npm run generate-api`

- [ ] [FRONT] Task 2: Extend auth store with `fetchMe` and `updateMe` actions (AC: #1, #2)
  - [ ] Edit `frontend/src/stores/auth.ts`
  - [ ] Convert store to setup syntax (Composition API style) to align with other stores — or add actions to existing Options API store, whichever keeps the diff minimal
  - [ ] Add action `fetchMe(): Promise<void>` — calls `apiClient.GET('/users/me')`, sets `user` on success
  - [ ] Add action `updateMe(payload: { name?: string; email?: string }): Promise<User>` — calls `apiClient.PUT('/users/me', { body: payload })`, updates `user` in store on success, throws on API error
  - [ ] Export `User` type from the store file (it is already defined there)

- [ ] [FRONT] Task 3: Create `useProfile` composable (AC: #1, #2, #4)
  - [ ] Create `frontend/src/composables/useProfile.ts`
  - [ ] Import `useAuthStore` and `useAsyncAction`
  - [ ] Expose `fetchMe` wrapped in `useAsyncAction` (loading + error state)
  - [ ] Expose `updateMe` wrapped in `useAsyncAction`
  - [ ] Expose `changePassword` wrapped in `useAsyncAction` — calls `apiClient.PUT('/users/me/password', { body: payload })`, throws on API error
  - [ ] Expose `user` as `computed(() => store.user)`
  - [ ] Return signature: `{ user, fetchMe, updateMe, changePassword }`

- [ ] [FRONT] Task 4: Create `ProfileInfoForm.vue` feature component (AC: #1, #2, #3)
  - [ ] Create `frontend/src/features/profile/ProfileInfoForm.vue`
  - [ ] Props: `user: User`, `isSaving: boolean`
  - [ ] Emits: `save: [payload: { name: string; email: string }]`
  - [ ] Zod schema:
    ```typescript
    z.object({
      name: z.string().min(1, 'Name is required').max(255, 'Name must be 255 characters or less'),
      email: z.string().min(1, 'Email is required').email('Invalid email format'),
    })
    ```
  - [ ] Use `useForm` + `useField` from vee-validate with `toTypedSchema(zod schema)`
  - [ ] Watch `user` prop and call `resetForm({ values: { name: user.name, email: user.email } })` on change
  - [ ] PrimeVue `InputText` for name (id="profile-name"), `InputText` type="email" for email (id="profile-email")
  - [ ] Standard label + input + error pattern (no FloatLabel, match LoginView style for consistency)
  - [ ] Save button: `Button` label="Save Changes" severity="primary" `:loading="isSaving"` `:disabled="!meta.dirty || !meta.valid"`
  - [ ] Display read-only metadata: role (PrimeVue `Tag` with severity mapping) and member since (`formatDate(user.created_at)`)

- [ ] [FRONT] Task 5: Create `ChangePasswordForm.vue` feature component (AC: #4, #5)
  - [ ] Create `frontend/src/features/profile/ChangePasswordForm.vue`
  - [ ] Props: `isSaving: boolean`
  - [ ] Emits: `save: [payload: { current_password: string; new_password: string }]`
  - [ ] Zod schema:
    ```typescript
    z.object({
      current_password: z.string().min(1, 'Current password is required'),
      new_password: z.string().min(8, 'Password must be at least 8 characters'),
      confirm_password: z.string().min(1, 'Please confirm your new password'),
    }).refine((data) => data.new_password === data.confirm_password, {
      message: 'Passwords do not match',
      path: ['confirm_password'],
    })
    ```
  - [ ] Use `useForm` + `useField` pattern, all three fields
  - [ ] PrimeVue `Password` component for all three fields (toggle-mask, :feedback="false"), use `inputId` prop for label association
  - [ ] Standard label + input + error pattern
  - [ ] Update Password button: `Button` label="Update Password" severity="secondary" `:loading="isSaving"` `:disabled="!meta.dirty || !meta.valid"`
  - [ ] On successful emit, the parent resets the form — expose `resetForm` via defineExpose OR accept a `resetKey` prop

- [ ] [FRONT] Task 6: Create `ProfileView.vue` view (AC: #1, #2, #3, #4, #5)
  - [ ] Create `frontend/src/views/ProfileView.vue`
  - [ ] On mount: call `fetchMe.execute()` to load fresh data from the API
  - [ ] Page header: `<h1>My Profile</h1>`
  - [ ] Loading state: PrimeVue `Skeleton` (3 lines at heights 2rem, 2.5rem, 2rem) while `fetchMe.isLoading && !user`
  - [ ] Error state: `Message` severity="error" + retry `Button` when `fetchMe.error && !user`
  - [ ] Two `Card` sections side by side (responsive: single column on mobile, two columns on md+):
    - Card 1 header "Profile Information" — contains `ProfileInfoForm`
    - Card 2 header "Change Password" — contains `ChangePasswordForm`
  - [ ] On `ProfileInfoForm@save`: call `updateMe.execute(payload)`, on success show Toast (severity="success", summary="Saved", detail="Profile updated"), update handled by store, on error show Toast (severity="error", summary="Error", detail=error.message)
  - [ ] On `ChangePasswordForm@save`: call `changePassword.execute(payload)`, on success show Toast (severity="success", summary="Done", detail="Password updated") and reset password form, on error show Toast (severity="error", detail=error.message)
  - [ ] Use PrimeVue `useToast()` and include `<Toast />` in template
  - [ ] Layout: `<div class="flex flex-col gap-6 p-6">`

- [ ] [FRONT] Task 7: Add `/profile` route (AC: #6)
  - [ ] Edit `frontend/src/router/index.ts`
  - [ ] Add route before the catch-all:
    ```typescript
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { requiresAuth: true },
    }
    ```

- [ ] [FRONT] Task 8: Add "My Profile" item to user menu in AppHeader (AC: #6)
  - [ ] Edit `frontend/src/ui/layout/AppHeader.vue`
  - [ ] Add `{ label: 'My Profile', icon: 'pi pi-user-edit', command: () => router.push({ name: 'profile' }) }` as the first item in `menuItems` (before Logout)

- [ ] [FRONT] Task 9: Unit tests for `useProfile` composable (AC: #1, #2, #4)
  - [ ] Create `frontend/src/composables/__tests__/useProfile.spec.ts`
  - [ ] Mock `apiClient` with `vi.mock('@/api/client')`
  - [ ] Test `fetchMe` success: `user` ref populated
  - [ ] Test `fetchMe` error: `fetchMe.error` set, `user` stays null
  - [ ] Test `updateMe` success: store `user` updated, returns updated User
  - [ ] Test `updateMe` error: throws, `updateMe.error` set
  - [ ] Test `changePassword` success: resolves without error
  - [ ] Test `changePassword` error: `changePassword.error` set

- [ ] [FRONT] Task 10: Unit tests for Zod schemas (AC: #3, #5)
  - [ ] Create `frontend/src/features/profile/__tests__/profileSchemas.spec.ts`
  - [ ] Test profile info schema: valid, empty name, invalid email
  - [ ] Test password schema: valid, short new_password, mismatched confirm_password, empty current_password

- [ ] [FRONT] Task 11: E2E test with Playwright (AC: #1, #2, #4, #6)
  - [ ] Create `frontend/e2e/tests/profile.spec.ts`
  - [ ] Mock `GET /api/v1/users/me` returning a user fixture
  - [ ] Test: page loads at `/profile` with name and email pre-filled
  - [ ] Test: edit name, click "Save Changes" with mocked PUT 200 — success Toast visible
  - [ ] Test: click "Save Changes" with mocked PUT 400 — error Toast visible
  - [ ] Test: open user menu in header — "My Profile" item is visible and navigates to `/profile`

## Dev Notes

### Dependencies (feat-2-user-profile-api)

This story depends on `feat-2-user-profile-api` which adds the following backend endpoints:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/users/me` | Returns the current authenticated user's profile |
| PUT | `/api/v1/users/me` | Updates the current user's name and/or email |
| PUT | `/api/v1/users/me/password` | Changes the current user's password |

These endpoints do NOT exist in the current `api/openapi.yaml` as of this writing. **Task 1 of this story adds them to the spec.** If feat-2 has already merged and added these endpoints, skip the spec edits in Task 1 and only run `npm run generate-api`.

The current spec has `/users/{id}` (GET/PUT/DELETE) which is admin-only. The `/users/me` endpoints are scoped to the authenticated user and require no admin role. The backend implementation is responsible for:
- Returning only the authenticated user's data (from JWT claims)
- Preventing password change if `current_password` is incorrect (return 400)
- Preventing email change to an already-taken email (return 409)

### File Paths

| File | Action |
|------|--------|
| `api/openapi.yaml` | Update — add `/users/me` GET, PUT and `/users/me/password` PUT paths + schemas |
| `frontend/src/stores/auth.ts` | Update — add `fetchMe`, `updateMe` actions |
| `frontend/src/composables/useProfile.ts` | Create |
| `frontend/src/features/profile/ProfileInfoForm.vue` | Create |
| `frontend/src/features/profile/ChangePasswordForm.vue` | Create |
| `frontend/src/views/ProfileView.vue` | Create |
| `frontend/src/router/index.ts` | Update — add `/profile` route |
| `frontend/src/ui/layout/AppHeader.vue` | Update — add "My Profile" menu item |
| `frontend/src/composables/__tests__/useProfile.spec.ts` | Create |
| `frontend/src/features/profile/__tests__/profileSchemas.spec.ts` | Create |
| `frontend/e2e/tests/profile.spec.ts` | Create |

### OpenAPI spec additions (Task 1)

```yaml
# Add to paths section in api/openapi.yaml

  /users/me:
    get:
      operationId: getMe
      summary: Get current authenticated user's profile
      tags: [users]
      responses:
        "200":
          description: Current user profile
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "401":
          $ref: "#/components/responses/Unauthorized"

    put:
      operationId: updateMe
      summary: Update current authenticated user's profile
      tags: [users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateMeRequest"
      responses:
        "200":
          description: Profile updated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "409":
          $ref: "#/components/responses/Conflict"

  /users/me/password:
    put:
      operationId: changeMyPassword
      summary: Change current authenticated user's password
      tags: [users]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/ChangePasswordRequest"
      responses:
        "204":
          description: Password changed successfully
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"

# Add to components/schemas section in api/openapi.yaml

    UpdateMeRequest:
      type: object
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 255
          example: Jane Smith
        email:
          type: string
          format: email
          example: jane@example.com

    ChangePasswordRequest:
      type: object
      required:
        - current_password
        - new_password
      properties:
        current_password:
          type: string
          format: password
          example: oldP@ss1
        new_password:
          type: string
          format: password
          minLength: 8
          example: newP@ss1
```

### Zod Validation Schemas

```typescript
// ProfileInfoForm.vue
const profileInfoSchema = toTypedSchema(
  z.object({
    name: z.string().min(1, 'Name is required').max(255, 'Name must be 255 characters or less'),
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
  })
)

// ChangePasswordForm.vue
const changePasswordSchema = toTypedSchema(
  z
    .object({
      current_password: z.string().min(1, 'Current password is required'),
      new_password: z.string().min(8, 'Password must be at least 8 characters'),
      confirm_password: z.string().min(1, 'Please confirm your new password'),
    })
    .refine((data) => data.new_password === data.confirm_password, {
      message: 'Passwords do not match',
      path: ['confirm_password'],
    })
)
```

### Auth Store Additions

```typescript
// frontend/src/stores/auth.ts — additions to existing actions object

async fetchMe(): Promise<void> {
  this.loading = true
  this.error = null
  try {
    const { data, error: apiError } = await apiClient.GET('/users/me')
    if (apiError) {
      this.error = 'Failed to load profile'
      return
    }
    this.user = { ...data, role: data.role ?? 'member' } as User
  } catch {
    this.error = 'Network error. Please try again.'
  } finally {
    this.loading = false
  }
},

async updateMe(payload: { name?: string; email?: string }): Promise<User> {
  const { data, error: apiError } = await apiClient.PUT('/users/me', {
    body: payload,
  })
  if (apiError || !data) {
    throw new Error('Failed to update profile')
  }
  const updated = { ...data, role: data.role ?? 'member' } as User
  this.user = updated
  return updated
},
```

### useProfile Composable

```typescript
// frontend/src/composables/useProfile.ts
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useProfile() {
  const store = useAuthStore()

  const fetchMe = useAsyncAction(() => store.fetchMe())
  const updateMe = useAsyncAction((payload: { name?: string; email?: string }) =>
    store.updateMe(payload)
  )
  const changePassword = useAsyncAction(
    async (payload: { current_password: string; new_password: string }) => {
      const { error: apiError } = await apiClient.PUT('/users/me/password', {
        body: payload,
      })
      if (apiError) {
        throw new Error('Failed to change password. Check your current password and try again.')
      }
    }
  )

  return {
    user: computed(() => store.user),
    fetchMe,
    updateMe,
    changePassword,
  }
}
```

### Component Structure / Layout

```
ProfileView.vue  (route: /profile)
├── <h1>My Profile</h1>
├── Skeleton x3  (v-if="fetchMe.isLoading && !user")
├── Message severity="error" + Button retry  (v-else-if="fetchMe.error && !user")
└── div.grid.grid-cols-1.md:grid-cols-2.gap-6  (v-else-if="user")
    ├── Card header="Profile Information"
    │   └── ProfileInfoForm
    │       ├── label "Name" + InputText#profile-name
    │       ├── label "Email" + InputText#profile-email type="email"
    │       ├── read-only: Tag (role) + member since
    │       └── Button "Save Changes"
    └── Card header="Change Password"
        └── ChangePasswordForm
            ├── label "Current Password" + Password#current-password
            ├── label "New Password" + Password#new-password
            ├── label "Confirm New Password" + Password#confirm-password
            └── Button "Update Password"
└── Toast
```

**Responsive layout:** use `class="grid grid-cols-1 gap-6 md:grid-cols-2"` for the cards container. On small screens both cards stack vertically; on medium+ they sit side by side.

### PrimeVue Components Used

| Component | Import path | Usage |
|-----------|-------------|-------|
| `Card` | `primevue/card` | Section containers for profile info and password change |
| `InputText` | `primevue/inputtext` | Name and email fields |
| `Password` | `primevue/password` | Current, new, and confirm password fields (use `inputId` prop) |
| `Button` | `primevue/button` | Save Changes, Update Password, Retry |
| `Tag` | `primevue/tag` | Read-only role display (severity="info" for member, severity="danger" for admin) |
| `Message` | `primevue/message` | Error state display |
| `Skeleton` | `primevue/skeleton` | Loading state |
| `Toast` | `primevue/toast` | Success/error notifications |

**Password component note:** always use `inputId` (not `id`) for PrimeVue `Password` to correctly associate the `<label for="...">`. Example:
```vue
<label for="current-password" class="text-sm font-medium">Current Password</label>
<Password inputId="current-password" v-model="currentPassword" :feedback="false" toggle-mask :invalid="!!currentPasswordError" input-class="w-full" class="w-full" />
```

### API Calls (openapi-fetch patterns)

```typescript
// GET /users/me
const { data, error } = await apiClient.GET('/users/me')

// PUT /users/me
const { data, error } = await apiClient.PUT('/users/me', {
  body: { name: 'Jane Smith', email: 'jane@example.com' },
})

// PUT /users/me/password
const { error } = await apiClient.PUT('/users/me/password', {
  body: { current_password: 'old', new_password: 'new' },
})
// 204 No Content — only check for error, data is undefined
```

### Router Additions

```typescript
// frontend/src/router/index.ts
// Add before the catch-all route { path: '/:pathMatch(.*)*', ... }
{
  path: '/profile',
  name: 'profile',
  component: () => import('@/views/ProfileView.vue'),
  meta: { requiresAuth: true },
},
```

### AppHeader User Menu Update

```typescript
// frontend/src/ui/layout/AppHeader.vue — update menuItems
const menuItems: MenuItem[] = [
  {
    label: 'My Profile',
    icon: 'pi pi-user-edit',
    command: () => router.push({ name: 'profile' }),
  },
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      router.push('/login')
    },
  },
]
```

The user's display name (from `authStore.user?.name`) can optionally be shown above the menu items using the `label` at the top of the `Menu` model — but this is optional for MVP.

### Testing Requirements

**Unit tests (Vitest):**

| Test file | Coverage target | What to test |
|-----------|----------------|--------------|
| `composables/__tests__/useProfile.spec.ts` | 95%+ | `fetchMe` success/error, `updateMe` success/error/store update, `changePassword` success/error |
| `features/profile/__tests__/profileSchemas.spec.ts` | 100% | All validation rules for both Zod schemas |

**E2E tests (Playwright) — `frontend/e2e/tests/profile.spec.ts`:**

```typescript
// Pattern: mock /api/v1/auth/me (auth guard) + mock /api/v1/users/me (profile data)

const userFixture = {
  id: '1',
  email: 'test@example.com',
  name: 'Test User',
  role: 'member',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ status: 200, json: userFixture })
  )
  await page.route('**/api/v1/users/me', (route) =>
    route.fulfill({ status: 200, json: userFixture })
  )
})
```

Test cases:
1. Navigate to `/profile` — page heading "My Profile" visible, name and email inputs pre-filled
2. Edit name, click "Save Changes" (mock PUT 200) — Toast "Profile updated" appears
3. Click "Save Changes" with mock PUT 400 — Toast error appears
4. Open user menu from header — "My Profile" item visible, click navigates to `/profile`
5. Fill valid current + new + confirm password, click "Update Password" (mock PUT 204) — Toast "Password updated" appears, password fields cleared
6. Submit password form with mismatched passwords — "Passwords do not match" error shown, button disabled

**Manual verification checklist:**
1. `npm run dev` — navigate to `/profile` via user menu "My Profile" item
2. Verify form pre-filled with current user name and email
3. Edit name, click "Save Changes" — success Toast; reload page, new name persists
4. Clear name field — "Name is required" error, Save button disabled
5. Enter invalid email — "Invalid email format" error, Save button disabled
6. Fill password change form with mismatched passwords — "Passwords do not match" error
7. Fill password change form correctly — Toast "Password updated", fields clear
8. `npm run build` — no TypeScript errors
9. `npm run lint` — no ESLint errors
10. `npm run type-check` — no type errors
11. `npm run test:unit` — all new tests pass

## Dev Agent Record

## Change Log
