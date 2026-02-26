# Story F-2.3: [FRONT] Add role selector to Edit User dialog

Status: ready-for-dev

## Story

As an admin,
I want to change a user's role from the edit dialog,
so that I can promote users to admin or demote them.

## Acceptance Criteria (BDD)

**AC1: Role dropdown present in edit dialog**
- **Given** admin opens the Edit User dialog
- **When** the dialog renders
- **Then** a Select dropdown for Role is visible with options "Member" (value: `user`) and "Admin" (value: `admin`)

**AC2: Current role pre-selected**
- **Given** admin edits a user with role "admin"
- **When** the dialog opens
- **Then** "Admin" is pre-selected in the Role dropdown

**AC3: Role change persists**
- **Given** admin changes role from "user" to "admin"
- **When** they save the edit
- **Then** the API is called with the new role, and the user list reflects the change

**AC4: Cannot change own role**
- **Given** admin is editing their own account
- **When** the dialog renders
- **Then** the Role dropdown is disabled (prevent self-demotion)

## Tasks / Subtasks

### Task 1 — Extend the Zod schema in `EditUserDialog.vue` to include `role`

File: `frontend/src/features/admin/EditUserDialog.vue`

- Add `role: z.enum(['admin', 'user'])` to the `editUserSchema` object (currently only has `name` and `email`)
- Add a `useField<string>('role')` call alongside the existing `name` and `email` field extractions
- Update the `watch` on `props.user` to also set `role: user.role` in the `resetForm` values
- Update `onSubmit` to pass `role` in the `values` object forwarded to `updateUser.execute`

### Task 2 — Replace the read-only `Tag` with a `Select` dropdown in `EditUserDialog.vue`

File: `frontend/src/features/admin/EditUserDialog.vue`

- Import `Select` from `primevue/select` (remove the `Tag` import — it is no longer needed in the role field)
- Define a `roleOptions` constant: `[{ label: 'Member', value: 'user' }, { label: 'Admin', value: 'admin' }]`
- Import `useAuthStore` from `@/stores/auth` and derive `isSelf = computed(() => authStore.user?.id === props.user?.id)`
- Replace the `<Tag>` block with a PrimeVue `<Select>` bound to the `role` vee-validate field (`v-model="role"`), using `option-label="label"` / `option-value="value"`, `:disabled="isSelf"`, and a helper text rendered conditionally when `isSelf` is true (e.g. "You cannot change your own role")

### Task 3 — Extend `updateUser` in the users store to accept `role`

File: `frontend/src/stores/users.ts`

- Widen the `payload` type of `updateUser` from `{ name?: string; email?: string }` to `{ name?: string; email?: string; role?: 'admin' | 'user' }` so the `role` field is passed through to `PUT /users/{id}`
- The `UpdateUserRequest` schema in `api/openapi.yaml` already includes an optional `role: enum [admin, user]` field, so no spec change is required — the generated type already covers this

### Task 4 — Refresh the user list after a role update

No additional changes required: `updateUser` in the store already calls `fetchUsers()` after a successful PUT, so the updated role will be reflected immediately in the `UserTable`.

## Dev Notes

- Priority: P2
- Use PrimeVue `Select` component for the role dropdown
- Role options: `[{ label: 'Member', value: 'user' }, { label: 'Admin', value: 'admin' }]`
- The PUT `/api/v1/users/{id}` endpoint already accepts `role` in the request body (`UpdateUserRequest` schema at line 2070 of `api/openapi.yaml`) — no backend or spec changes needed
- Self-detection: compare `props.user.id` with `useAuthStore().user?.id`
- The `User` interface in `frontend/src/stores/auth.ts` already types `role` as `'admin' | 'user'`
- No API regeneration needed: the generated client already exposes `role` on `UpdateUserRequest`
