# Story 1.13: [FRONT] User management page (admin only)

Status: ready-for-dev

## Story

As an admin,
I want to manage user accounts,
So that I can control access.

## Acceptance Criteria (BDD)

**AC1: User list displayed in DataTable**
- **Given** admin navigates to `/admin/users`
- **When** page loads
- **Then** DataTable shows email, role (as Tag), created date columns with server-side pagination

**AC2: Create user via dialog**
- **Given** admin clicks "Create User" button
- **When** dialog opens
- **Then** form with email, password, name, role dropdown is displayed; submit calls POST `/api/v1/auth/register` and refreshes the table

**AC3: Edit user via dialog**
- **Given** admin clicks the edit (pencil) button on a user row
- **When** dialog opens
- **Then** form pre-filled with name, email, role is displayed; submit calls PUT `/api/v1/users/{id}` and refreshes the table

**AC4: Delete user with confirmation**
- **Given** admin clicks the delete (trash) button on a user row
- **When** ConfirmDialog is confirmed
- **Then** DELETE `/api/v1/users/{id}` is called, user is removed, and table refreshes

**AC5: Non-admin redirect**
- **Given** non-admin user navigates to `/admin/users`
- **When** route guard checks `user.role`
- **Then** user is redirected to `/` (dashboard)

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Add `role` field to auth store User type and extend OpenAPI User schema (AC: #1, #5)
  - [ ] Update `frontend/src/stores/auth.ts` — add `role: string` to the `User` interface (values: `'admin' | 'member'`)
  - [ ] NOTE: The `role` field on the OpenAPI `User` schema will be added by Story 1-4 (backend peer). Until then, the auth store can default `role` to `'member'` when it is absent from API responses, using `(json.role ?? 'member')` in `login()` and `checkAuth()` actions
  - [ ] This unblocks AC5 (admin guard) and AC1 (role column) without waiting for backend

- [ ] [FRONT] Task 2: Create admin route guard in `frontend/src/router/guards.ts` (AC: #5)
  - [ ] Add `setupAdminGuard(router: Router)` function alongside existing `setupAuthGuard`
  - [ ] Register a `beforeEach` guard that checks: if `to.meta.requiresAdmin === true` AND `auth.user?.role !== 'admin'`, redirect to `{ path: '/' }`
  - [ ] Guard must run AFTER `setupAuthGuard` (so user is already authenticated)
  - [ ] Call `setupAdminGuard(router)` in `frontend/src/router/index.ts` after `setupAuthGuard(router)`

- [ ] [FRONT] Task 3: Create Pinia users store `frontend/src/stores/users.ts` (AC: #1, #2, #3, #4)
  - [ ] Use Composition API (setup) style with `defineStore('users', () => { ... })`
  - [ ] State: `users: ref<User[]>([])`, `pagination: ref<Pagination>({ total: 0, page: 1, per_page: 20 })`, `isLoading: ref(false)`
  - [ ] Action `fetchUsers(params?: { page?: number; per_page?: number })` — calls `apiClient.GET('/users', { params: { query } })`, populates `users` and `pagination`
  - [ ] Action `createUser(payload: { email: string; password: string; name: string })` — calls `apiClient.POST('/auth/register', { body: payload })`; on success, calls `fetchUsers()` to refresh
  - [ ] Action `updateUser(id: string, payload: { name?: string; email?: string })` — calls `apiClient.PUT('/users/{id}', { params: { path: { id } }, body: payload })`; on success, calls `fetchUsers()` to refresh
  - [ ] Action `deleteUser(id: string)` — calls `apiClient.DELETE('/users/{id}', { params: { path: { id } } })`; on success, calls `fetchUsers()` to refresh
  - [ ] Use `apiClient` from `@/api/client` for all calls (typed via openapi-fetch)

- [ ] [FRONT] Task 4: Create `useUsers` composable `frontend/src/composables/useUsers.ts` (AC: #1, #2, #3, #4)
  - [ ] Import `useUsersStore` and `useAsyncAction`
  - [ ] Expose computed refs: `users`, `pagination`, `isLoading`
  - [ ] Expose `fetchUsers` wrapped in `useAsyncAction` — returns `{ execute, isLoading, error }`
  - [ ] Expose `createUser` wrapped in `useAsyncAction`
  - [ ] Expose `updateUser` wrapped in `useAsyncAction`
  - [ ] Expose `deleteUser` wrapped in `useAsyncAction`
  - [ ] Signature:
    ```typescript
    export function useUsers() {
      const store = useUsersStore()
      const fetch = useAsyncAction(store.fetchUsers)
      const create = useAsyncAction(store.createUser)
      const update = useAsyncAction(store.updateUser)
      const remove = useAsyncAction(store.deleteUser)
      return {
        users: computed(() => store.users),
        pagination: computed(() => store.pagination),
        isLoading: computed(() => store.isLoading),
        fetchUsers: fetch,
        createUser: create,
        updateUser: update,
        deleteUser: remove,
      }
    }
    ```

- [ ] [FRONT] Task 5: Create `UserTable.vue` feature component `frontend/src/features/admin/UserTable.vue` (AC: #1)
  - [ ] Props: `users: User[]`, `loading: boolean`, `pagination: Pagination`
  - [ ] Emits: `edit: [user: User]`, `delete: [user: User]`, `page-change: [page: number]`
  - [ ] PrimeVue `DataTable` with `:value="users"`, `:loading="loading"`, `stripedRows`
  - [ ] Columns: `email` (header "Email"), `name` (header "Name"), `role` (header "Role" with `<Tag>` template — severity `danger` for `admin`, `info` for `member`), `created_at` (header "Created" — format with `date-fns` `format()`), Actions column with edit (pencil) and delete (trash) buttons
  - [ ] Pagination: use PrimeVue DataTable built-in paginator with `:paginator="true"` `:rows="pagination.per_page"` `:totalRecords="pagination.total"` `:first="(pagination.page - 1) * pagination.per_page"` and `@page` event emitting `page-change`
  - [ ] Zero business logic — just renders data and emits events

- [ ] [FRONT] Task 6: Create `CreateUserDialog.vue` feature component `frontend/src/features/admin/CreateUserDialog.vue` (AC: #2)
  - [ ] Props: `visible: boolean`
  - [ ] Emits: `update:visible: [value: boolean]`, `created: []`
  - [ ] PrimeVue `Dialog` with `v-model:visible`, header "Create User", modal
  - [ ] Form fields using vee-validate `useForm` + zod schema:
    - `email`: `z.string().min(1, 'Email is required').email('Invalid email')` — PrimeVue `InputText`
    - `password`: `z.string().min(8, 'Password must be at least 8 characters')` — PrimeVue `Password`
    - `name`: `z.string().min(1, 'Name is required').max(255)` — PrimeVue `InputText`
    - `role`: `z.enum(['admin', 'member'])` — PrimeVue `Select` with options `[{ label: 'Admin', value: 'admin' }, { label: 'Member', value: 'member' }]`
  - [ ] Submit button calls `useUsers().createUser.execute(formValues)`; on success, emit `created` and close dialog
  - [ ] Display inline validation errors below each field
  - [ ] Footer: Cancel button (closes dialog) + Create button (submits, shows loading state)

- [ ] [FRONT] Task 7: Create `EditUserDialog.vue` feature component `frontend/src/features/admin/EditUserDialog.vue` (AC: #3)
  - [ ] Props: `visible: boolean`, `user: User | null`
  - [ ] Emits: `update:visible: [value: boolean]`, `updated: []`
  - [ ] PrimeVue `Dialog` with `v-model:visible`, header "Edit User", modal
  - [ ] Form fields (pre-filled from `user` prop via `watch` + `resetForm`):
    - `name`: `z.string().min(1).max(255)` — PrimeVue `InputText`
    - `email`: `z.string().min(1).email()` — PrimeVue `InputText`
  - [ ] NOTE: `role` update is not supported by the current `UpdateUserRequest` schema. When Story 1-4 adds role to the API, add a `Select` field here. For now, display role as a read-only `Tag` in the dialog
  - [ ] Submit calls `useUsers().updateUser.execute(user.id, formValues)`; on success, emit `updated` and close
  - [ ] Footer: Cancel + Save buttons

- [ ] [FRONT] Task 8: Create `UserManagementView.vue` view `frontend/src/views/admin/UserManagementView.vue` (AC: #1, #2, #3, #4)
  - [ ] Route-level component for `/admin/users`
  - [ ] Composes: `UserTable`, `CreateUserDialog`, `EditUserDialog`, PrimeVue `ConfirmDialog`
  - [ ] On mount: call `useUsers().fetchUsers.execute()`
  - [ ] "Create User" button in page header opens `CreateUserDialog`
  - [ ] `UserTable` `@edit` opens `EditUserDialog` with selected user
  - [ ] `UserTable` `@delete` triggers PrimeVue `useConfirm()` with message "Are you sure you want to delete {user.email}?"; on accept, calls `useUsers().deleteUser.execute(user.id)`
  - [ ] `UserTable` `@page-change` calls `fetchUsers.execute({ page: newPage })`
  - [ ] Display `Toast` for success/error feedback via PrimeVue `useToast()`
  - [ ] Layout: page title "User Management", "Create User" button top-right, table below

- [ ] [FRONT] Task 9: Register route and update sidebar (AC: #1, #5)
  - [ ] Add route to `frontend/src/router/index.ts`:
    ```typescript
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/admin/UserManagementView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    }
    ```
  - [ ] Update `frontend/src/ui/layout/AppSidebar.vue` — add a "Users" nav item (`pi pi-users`, route `/admin/users`) in a separate admin section, conditionally rendered only when `useAuthStore().user?.role === 'admin'`
  - [ ] Add a visual separator (divider) before admin nav items in sidebar

- [ ] [FRONT] Task 10: Unit tests (AC: #1, #2, #3, #4, #5)
  - [ ] Create `frontend/src/stores/__tests__/users.spec.ts`:
    - Test `fetchUsers` populates state from mocked API response
    - Test `createUser` calls register endpoint and refreshes
    - Test `deleteUser` calls delete endpoint and refreshes
  - [ ] Create `frontend/src/composables/__tests__/useUsers.spec.ts`:
    - Test that composable exposes correct reactive interface
    - Test loading/error state propagation from `useAsyncAction`
  - [ ] Create `frontend/src/features/admin/__tests__/UserTable.spec.ts`:
    - Test renders columns (email, name, role tag, created date, actions)
    - Test emits `edit` when pencil button clicked
    - Test emits `delete` when trash button clicked
  - [ ] Create `frontend/src/router/__tests__/adminGuard.spec.ts`:
    - Test admin user can access `/admin/users`
    - Test non-admin is redirected to `/`
    - Test unauthenticated user is redirected to `/login` (handled by auth guard)

## Dev Notes

### Dependencies

**Story dependencies (must be complete):**
- Story 1-7: Vue 3 scaffolding, PrimeVue 4, Tailwind CSS v4
- Story 1-8: App shell (AppSidebar to update)
- Story 1-9: Auth store, useAuth composable, router guards, vee-validate + zod
- Story 1-16: Router setup, Pinia stores, openapi-fetch client

**Story dependencies (peer, can proceed in parallel):**
- Story 1-4: Backend user management API (provides endpoints this page consumes). Frontend can be built against mock/stub responses. The `role` field on `User` will be added by 1-4 — this story includes a temporary fallback.

**npm packages (already installed):**
- `pinia`, `vee-validate`, `@vee-validate/zod`, `zod`, `openapi-fetch`, `primevue`, `date-fns`

**npm packages to install if missing:**
```bash
cd frontend && npm install date-fns
```

### Architecture Requirements

**Component Hierarchy:**
```
UserManagementView.vue (route: /admin/users)
├── Page header ("User Management" + "Create User" button)
├── UserTable.vue
│   └── PrimeVue DataTable with columns
├── CreateUserDialog.vue
│   └── PrimeVue Dialog with vee-validate form
├── EditUserDialog.vue
│   └── PrimeVue Dialog with vee-validate form
├── PrimeVue ConfirmDialog (delete confirmation)
└── PrimeVue Toast (success/error feedback)
```

**Data Flow:**
```
UserManagementView (orchestrator)
  │
  ├─ useUsers() composable
  │    └─ useUsersStore() Pinia store
  │         └─ apiClient (openapi-fetch)
  │
  ├─ useConfirm() — PrimeVue confirm service
  └─ useToast() — PrimeVue toast service
```

### File Paths (exact)

| File | Action |
|------|--------|
| `frontend/src/stores/auth.ts` | Update (add `role` to User interface) |
| `frontend/src/stores/users.ts` | Create |
| `frontend/src/composables/useUsers.ts` | Create |
| `frontend/src/router/guards.ts` | Update (add `setupAdminGuard`) |
| `frontend/src/router/index.ts` | Update (add `/admin/users` route) |
| `frontend/src/views/admin/UserManagementView.vue` | Create |
| `frontend/src/features/admin/UserTable.vue` | Create |
| `frontend/src/features/admin/CreateUserDialog.vue` | Create |
| `frontend/src/features/admin/EditUserDialog.vue` | Create |
| `frontend/src/ui/layout/AppSidebar.vue` | Update (add admin nav section) |
| `frontend/src/stores/__tests__/users.spec.ts` | Create |
| `frontend/src/composables/__tests__/useUsers.spec.ts` | Create |
| `frontend/src/features/admin/__tests__/UserTable.spec.ts` | Create |
| `frontend/src/router/__tests__/adminGuard.spec.ts` | Create |

### Technical Specifications

**Updated User Interface (auth store):**
```typescript
// frontend/src/stores/auth.ts
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'member'  // added by this story
  created_at?: string
  updated_at?: string
}
```

**Users Store Signature:**
```typescript
// frontend/src/stores/users.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '@/api/client'
import type { User } from '@/stores/auth'

interface Pagination {
  total: number
  page: number
  per_page: number
}

export const useUsersStore = defineStore('users', () => {
  const users = ref<User[]>([])
  const pagination = ref<Pagination>({ total: 0, page: 1, per_page: 20 })
  const isLoading = ref(false)

  async function fetchUsers(params?: { page?: number; per_page?: number }): Promise<void> { /* ... */ }
  async function createUser(payload: { email: string; password: string; name: string }): Promise<void> { /* ... */ }
  async function updateUser(id: string, payload: { name?: string; email?: string }): Promise<void> { /* ... */ }
  async function deleteUser(id: string): Promise<void> { /* ... */ }

  return { users, pagination, isLoading, fetchUsers, createUser, updateUser, deleteUser }
})
```

**Admin Guard Signature:**
```typescript
// frontend/src/router/guards.ts (addition)
export function setupAdminGuard(router: Router) {
  router.beforeEach((to) => {
    if (to.meta.requiresAdmin !== true) return

    const auth = useAuthStore()
    if (auth.user?.role !== 'admin') {
      return { path: '/' }
    }
  })
}
```

**Route Meta Extension:**
```typescript
// Extend vue-router RouteMeta type
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    requiresAdmin?: boolean
  }
}
```

**Component Props/Emits:**

| Component | Props | Emits |
|-----------|-------|-------|
| `UserTable` | `users: User[]`, `loading: boolean`, `pagination: Pagination` | `edit: [User]`, `delete: [User]`, `page-change: [number]` |
| `CreateUserDialog` | `visible: boolean` | `update:visible: [boolean]`, `created: []` |
| `EditUserDialog` | `visible: boolean`, `user: User \| null` | `update:visible: [boolean]`, `updated: []` |

**Zod Schemas:**
```typescript
// CreateUserDialog internal schema
const createUserSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email format'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  name: z.string().min(1, 'Name is required').max(255, 'Name too long'),
  role: z.enum(['admin', 'member']),
})

// EditUserDialog internal schema
const editUserSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name too long'),
  email: z.string().min(1, 'Email is required').email('Invalid email format'),
})
```

### Testing Requirements

**Unit tests (Vitest):**

| Test file | What to test | Coverage target |
|-----------|-------------|----------------|
| `stores/__tests__/users.spec.ts` | fetchUsers, createUser, updateUser, deleteUser actions | 90%+ |
| `composables/__tests__/useUsers.spec.ts` | Reactive interface, loading/error propagation | 90%+ |
| `features/admin/__tests__/UserTable.spec.ts` | Column rendering, event emission on button clicks | As needed |
| `router/__tests__/adminGuard.spec.ts` | Admin access, non-admin redirect, meta flag checking | 100% |

**Manual verification checklist:**
1. Log in as admin user, navigate to `/admin/users` -- DataTable renders with user list
2. Click "Create User" -- dialog opens with form, fill and submit -- user appears in table
3. Click pencil icon on a row -- edit dialog opens pre-filled, modify and save -- table updates
4. Click trash icon on a row -- confirm dialog appears, confirm -- user removed from table
5. Log in as non-admin, navigate to `/admin/users` -- redirected to `/`
6. Sidebar shows "Users" link only for admin users
7. `npm run lint` passes
8. `npm run type-check` passes
9. `npm run test:unit` passes

### API Endpoints Used

| Method | Path | Request Body | Response | Used In |
|--------|------|-------------|----------|---------|
| GET | `/api/v1/users` | (query: `page`, `per_page`) | 200: `UserList` | `fetchUsers` |
| GET | `/api/v1/users/{id}` | (none) | 200: `User` | (reserved, not used directly) |
| PUT | `/api/v1/users/{id}` | `UpdateUserRequest` | 200: `User` | `updateUser` |
| DELETE | `/api/v1/users/{id}` | (none) | 204 | `deleteUser` |
| POST | `/api/v1/auth/register` | `RegisterRequest` | 201: `User` | `createUser` |

### Role Field Strategy

The OpenAPI `User` schema does not yet include `role`. Story 1-4 (backend) will add it. Until then:
- The frontend `User` interface includes `role: 'admin' | 'member'`
- When parsing API responses, default to `'member'` if `role` is absent: `role: json.role ?? 'member'`
- The admin guard, role Tag in DataTable, and sidebar visibility all work against this field
- When Story 1-4 lands, remove the fallback default

### PrimeVue Services Required

The view needs PrimeVue `ConfirmationService` and `ToastService` to be registered in `main.ts`. If not already registered:
```typescript
import ConfirmationService from 'primevue/confirmationservice'
import ToastService from 'primevue/toastservice'
app.use(ConfirmationService)
app.use(ToastService)
```

### References

- [Source: api/openapi.yaml -- /users endpoints, /auth/register, User schema, UserList schema]
- [Source: _bmad-output/planning-artifacts/epics.md -- Epic 1, Story 1.13]
- [Source: _bmad-output/planning-artifacts/architecture.md -- Frontend component organization]
- [Source: frontend/src/stores/auth.ts -- Existing User type and auth store]
- [Source: frontend/src/composables/useAsyncAction.ts -- useAsyncAction pattern]
- [Source: frontend/src/router/guards.ts -- Existing setupAuthGuard implementation]
- [Source: frontend/src/ui/layout/AppSidebar.vue -- Existing sidebar nav items structure]
- [Source: frontend/CLAUDE.md -- PrimeVue patterns, composable conventions, testing patterns]

## Dev Agent Record

## Change Log
