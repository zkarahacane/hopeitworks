# Story fix-12: [FRONT] Regenerate API schema and fix type safety issues

Status: ready-for-dev

## Story

As a frontend developer,
I want the generated `schema.d.ts` to match the current OpenAPI spec and all API calls to use properly typed paths,
so that TypeScript catches contract mismatches at compile time instead of at runtime.

## Context

The frontend `schema.d.ts` was generated before several auth/user endpoints were added to `api/openapi.yaml` (fix-10 adds project fields; feat-2/feat-3 added `/auth/forgot-password`, `/auth/reset-password`, `/users/me`, `/users/me/password`). The schema is also missing the pause/resume run paths. As a result, several composables and stores cast paths `as never` to bypass the type checker, and the `User` interface uses `role: 'admin' | 'member'` while the spec defines `role: 'admin' | 'user'`.

This story depends on fix-10 landing first (the OpenAPI spec must be up to date before regenerating).

## Acceptance Criteria (BDD)

**AC1: schema.d.ts includes all paths from the OpenAPI spec**
- **Given** fix-10 has been merged and `api/openapi.yaml` is up to date
- **When** the agent runs `npm run generate:api` in `frontend/`
- **Then** `frontend/src/api/schema.d.ts` contains entries for `/auth/forgot-password`, `/auth/reset-password`, `/users/me` (GET+PUT), `/users/me/password` (PUT), `/projects/{projectId}/runs/{runId}/pause`, and `/projects/{projectId}/runs/{runId}/resume`

**AC2: No `as never` casts on API paths in composables**
- **Given** the regenerated schema includes all relevant paths
- **When** `useRunLauncher.ts`, `useRunDetail.ts`, and `useStoryDetail.ts` are updated
- **Then** each `apiClient.GET/POST` call uses the correct path string literal from the schema with no `as never` cast

**AC3: `stores/stories.ts` `fetchStoriesByEpic` uses a valid typed path**
- **Given** the regenerated schema includes `/projects/{projectId}/stories`
- **When** `fetchStoriesByEpic` constructs its API call
- **Then** the path and params are properly typed — no cast to a wrong path and no `as Parameters<...>[1]` workaround

**AC4: `User` interface role type matches the OpenAPI spec**
- **Given** the OpenAPI spec defines `role: 'admin' | 'user'`
- **When** `stores/auth.ts` declares the `User` interface
- **Then** `role` is typed as `'admin' | 'user'`, not `'admin' | 'member'`
- **And** all fallback role assignments (`json.role ?? 'member'`) use `'user'` as the default
- **And** all references to `'member'` in tests, components, and router guards are updated to `'user'`

**AC5: `stores/auth.ts` auth methods use `apiClient` where appropriate**
- **Given** `apiClient` is configured with cookie credentials and the 401 redirect middleware
- **When** `login`, `logout`, and `checkAuth` are evaluated
- **Then** `login` is migrated to `apiClient.POST('/auth/login', ...)` using the typed schema
- **And** `logout` is migrated to `apiClient.POST('/auth/logout', ...)`
- **And** `checkAuth` is migrated to `apiClient.GET('/auth/me', ...)`
- **And** `forgotPassword` and `resetPassword` remain as raw `fetch` (these paths are absent from schema.d.ts unless fix-10 adds them — see Dev Notes)

**AC6: `npm run type-check` passes with no errors**
- **Given** all the above changes are applied
- **When** the agent runs `npm run type-check` in `frontend/`
- **Then** `vue-tsc --build` exits 0 with no type errors

**AC7: Unit tests updated to reflect `'user'` role**
- **Given** tests in `stores/__tests__/auth.spec.ts`, `stores/__tests__/users.spec.ts`, `router/__tests__/adminGuard.spec.ts`, and `features/admin/__tests__/UserTable.spec.ts` use `role: 'member'`
- **When** the fix is applied
- **Then** all occurrences of `role: 'member'` in test fixtures are replaced with `role: 'user'`
- **And** `npm run test:unit` passes

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Regenerate `schema.d.ts` from the OpenAPI spec (AC: #1)
  - [ ] Run `cd frontend && npm run generate:api`
  - [ ] Verify new paths are present: `/auth/forgot-password`, `/auth/reset-password`, `/users/me`, `/users/me/password`, `/projects/{projectId}/runs/{runId}/pause`, `/projects/{projectId}/runs/{runId}/resume`
  - [ ] Commit the updated `schema.d.ts`

- [ ] [FRONT] Task 2: Fix `as never` casts in `useRunLauncher.ts` (AC: #2)
  - [ ] Change path from `'/projects/{id}/stories/{story_id}/runs' as never` to `'/projects/{projectId}/stories/{storyId}/runs'`
  - [ ] Update path params from `{ id: projectId, story_id: storyId }` to `{ projectId, storyId }`
  - [ ] Remove both `as never` casts; let TypeScript infer the options type
  - [ ] Remove the manual `res` type assertion — use the typed response directly

- [ ] [FRONT] Task 3: Fix `as never` casts in `useRunDetail.ts` (AC: #2)
  - [ ] Change path from `'/runs/{runId}' as never` to `'/runs/{runId}'` (this path exists in the schema)
  - [ ] Remove `as never` from the options argument
  - [ ] Remove `data as unknown as RunWithSteps` — cast only if the generated type is compatible or derive `RunWithSteps` from the schema type

- [ ] [FRONT] Task 4: Fix `as never` casts in `useStoryDetail.ts` (AC: #2)
  - [ ] Change path from `'/projects/{projectId}/stories/{storyId}' as never` to `'/projects/{projectId}/stories/{storyId}'` (this path exists in the schema)
  - [ ] Remove `as never` from the options argument
  - [ ] Remove `data as unknown as Story` cast

- [ ] [FRONT] Task 5: Fix `fetchStoriesByEpic` in `stores/stories.ts` (AC: #3)
  - [ ] Replace the wrong path cast `'/projects/{projectId}/stories' as '/projects/{projectId}/epics'` with the correct path `'/projects/{projectId}/stories'`
  - [ ] Remove the `as Parameters<typeof apiClient.GET>[1]` workaround
  - [ ] Verify that `epic_id` is a valid query parameter for `listStories` in the schema; if not, pass it as `Record<string, string>` with a comment referencing the schema

- [ ] [FRONT] Task 6: Fix `User` interface role type in `stores/auth.ts` (AC: #4)
  - [ ] Change `role: 'admin' | 'member'` to `role: 'admin' | 'user'`
  - [ ] Change all `json.role ?? 'member'` fallbacks to `json.role ?? 'user'`
  - [ ] Change `data.role ?? 'member'` fallbacks to `data.role ?? 'user'`

- [ ] [FRONT] Task 7: Migrate `login`, `logout`, `checkAuth` to `apiClient` in `stores/auth.ts` (AC: #5)
  - [ ] Replace raw `fetch('/api/v1/auth/login', ...)` with `apiClient.POST('/auth/login', { body: { email, password } })`
  - [ ] Replace raw `fetch('/api/v1/auth/logout', ...)` with `apiClient.POST('/auth/logout', {})`
  - [ ] Replace raw `fetch('/api/v1/auth/me', ...)` with `apiClient.GET('/auth/me')` (already done in `fetchMe` — unify or remove duplication)
  - [ ] Assess `forgotPassword` and `resetPassword`: if the new schema includes `/auth/forgot-password` and `/auth/reset-password`, migrate them; otherwise document why they remain as raw `fetch`

- [ ] [FRONT] Task 8: Update `'member'` to `'user'` in all affected files (AC: #4, #7)
  - [ ] `frontend/src/features/admin/CreateUserDialog.vue` — zod schema `z.enum(['admin', 'member'])` → `z.enum(['admin', 'user'])`, `initialValues.role: 'member'` → `'user'`, `roleOptions` label/value pair
  - [ ] `frontend/src/stores/__tests__/auth.spec.ts` — all `role: 'member'` fixture values
  - [ ] `frontend/src/stores/__tests__/users.spec.ts` — all `role: 'member'` fixture values
  - [ ] `frontend/src/router/__tests__/adminGuard.spec.ts` — all `role: 'member'` fixture values
  - [ ] `frontend/src/features/admin/__tests__/UserTable.spec.ts` — `role: 'member' as const` → `role: 'user' as const`

- [ ] [FRONT] Task 9: Run type-check and unit tests (AC: #6, #7)
  - [ ] `cd frontend && npm run type-check`
  - [ ] `cd frontend && npm run test:unit`
  - [ ] Fix any remaining errors before committing

## Dev Notes

### Dependencies

- **fix-10-openapi-project-fields** must be merged first — this story regenerates `schema.d.ts` from the spec that fix-10 updates. The agent must NOT regen until fix-10 is on `main` / the target branch.
- The `generate:api` script in `package.json` is `openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts` (note: script name is `generate:api`, not `generate-api`).

### File Paths

| File | Change |
|------|--------|
| `frontend/src/api/schema.d.ts` | Regenerated — do not manually edit |
| `frontend/src/composables/useRunLauncher.ts` | Fix path + params + remove `as never` |
| `frontend/src/features/runs/composables/useRunDetail.ts` | Fix path + remove `as never` |
| `frontend/src/composables/useStoryDetail.ts` | Fix path + remove `as never` |
| `frontend/src/stores/stories.ts` | Fix `fetchStoriesByEpic` path cast |
| `frontend/src/stores/auth.ts` | Fix role type, fallback defaults, migrate raw fetch |
| `frontend/src/features/admin/CreateUserDialog.vue` | Fix zod enum + initialValues + roleOptions |
| `frontend/src/stores/__tests__/auth.spec.ts` | Fix `role: 'member'` fixtures |
| `frontend/src/stores/__tests__/users.spec.ts` | Fix `role: 'member'` fixtures |
| `frontend/src/router/__tests__/adminGuard.spec.ts` | Fix `role: 'member'` fixtures |
| `frontend/src/features/admin/__tests__/UserTable.spec.ts` | Fix `role: 'member'` fixtures |

### Architecture / Implementation Details

#### Task 2 — `useRunLauncher.ts` before/after

The current path `'/projects/{id}/stories/{story_id}/runs'` uses `{id}` and `{story_id}`, but the schema defines `/projects/{projectId}/stories/{storyId}/runs` with params `projectId` and `storyId`.

```typescript
// BEFORE (broken — path and params do not match schema)
const response = await apiClient.POST(
  '/projects/{id}/stories/{story_id}/runs' as never,
  {
    params: { path: { id: projectId, story_id: storyId } },
  } as never,
)
const res = response as { error?: { message?: string }; response?: { status: number }; data?: unknown }
if (res.error) { ... }
return res.data

// AFTER (typed)
const { data, error: apiError, response } = await apiClient.POST(
  '/projects/{projectId}/stories/{storyId}/runs',
  {
    params: { path: { projectId, storyId } },
  },
)
if (apiError) {
  if (response?.status === 409) throw new Error(ALREADY_RUNNING_ERROR)
  throw new Error((apiError as { error?: { message?: string } })?.error?.message ?? 'Failed to launch run')
}
return data
```

#### Task 3 — `useRunDetail.ts` before/after

The path `/runs/{runId}` is already present in `schema.d.ts` (line 345). The `as never` casts are unnecessary.

```typescript
// BEFORE
const { data, error: apiError } = await apiClient.GET(
  '/runs/{runId}' as never,
  { params: { path: { runId } } } as never,
)
return data as unknown as RunWithSteps

// AFTER
const { data, error: apiError } = await apiClient.GET('/runs/{runId}', {
  params: { path: { runId } },
})
if (apiError) throw new Error('Failed to load run')
return data as RunWithSteps  // cast only because RunWithSteps is a local interface
```

#### Task 4 — `useStoryDetail.ts` before/after

The path `/projects/{projectId}/stories/{storyId}` is present in `schema.d.ts` (line 271).

```typescript
// BEFORE
const { data, error: apiError } = await apiClient.GET(
  '/projects/{projectId}/stories/{storyId}' as never,
  { params: { path: { projectId, storyId } } } as never,
)
return data as unknown as Story

// AFTER
const { data, error: apiError } = await apiClient.GET(
  '/projects/{projectId}/stories/{storyId}',
  { params: { path: { projectId, storyId } } },
)
if (apiError) throw new Error('Failed to load story')
return data as Story
```

#### Task 5 — `stores/stories.ts` `fetchStoriesByEpic` before/after

The path `/projects/{projectId}/stories` exists in the schema. The workaround cast is wrong — it casts to the epics path to silence a type error about `epic_id` not being a recognized query param.

```typescript
// BEFORE (wrong path cast + params workaround)
const { data, error: apiError } = await apiClient.GET(
  '/projects/{projectId}/stories' as '/projects/{projectId}/epics',
  {
    params: {
      path: { projectId },
      query: { epic_id: epicId } as Record<string, string>,
    },
  } as Parameters<typeof apiClient.GET>[1],
)

// AFTER
// If the regenerated schema includes epic_id as a query param for listStories:
const { data, error: apiError } = await apiClient.GET('/projects/{projectId}/stories', {
  params: {
    path: { projectId },
    query: { epic_id: epicId },
  },
})

// If epic_id is NOT in the schema query params, keep the typed path and cast only the query:
const { data, error: apiError } = await apiClient.GET('/projects/{projectId}/stories', {
  params: {
    path: { projectId },
    query: { epic_id: epicId } as Record<string, string>,
  },
})
```

Check the schema after regen to determine which variant applies.

#### Task 7 — Auth store `login` migration

The current `login` action uses raw `fetch` and loses the `apiClient` 401 redirect middleware. After migration:

```typescript
async login(email: string, password: string): Promise<boolean> {
  this.loading = true
  this.error = null
  try {
    const { data, error: apiError, response } = await apiClient.POST('/auth/login', {
      body: { email, password },
    })
    if (apiError || !data) {
      this.error = (apiError as { message?: string })?.message ?? 'Invalid email or password'
      return false
    }
    this.user = { ...data, role: data.role ?? 'user' } as User
    return true
  } catch {
    this.error = 'Network error. Please try again.'
    return false
  } finally {
    this.loading = false
  }
},
```

For `logout`, since it intentionally swallows errors:

```typescript
async logout(): Promise<void> {
  await apiClient.POST('/auth/logout', {}).catch(() => {})
  this.user = null
  this.error = null
},
```

For `checkAuth`, unify with `fetchMe` (or keep both if they serve different purposes — `checkAuth` is called by the auth guard on first navigation):

```typescript
async checkAuth(): Promise<void> {
  this.loading = true
  try {
    const { data } = await apiClient.GET('/auth/me')
    this.user = data ? { ...data, role: data.role ?? 'user' } as User : null
  } catch {
    this.user = null
  } finally {
    this.loading = false
  }
},
```

Note: `forgotPassword` and `resetPassword` currently use raw `fetch` with no credentials because these endpoints do not require authentication. After regen, if `/auth/forgot-password` and `/auth/reset-password` appear in the schema, migrate them to `apiClient` too. If they are absent, document the reason in a comment.

#### Task 8 — `CreateUserDialog.vue` role fix

The `roleOptions` array and zod enum must change from `'member'` to `'user'`. The display label can remain "Member" for user-facing text, but the internal value must be `'user'` to match the API:

```typescript
// BEFORE
role: z.enum(['admin', 'member'])
initialValues: { ..., role: 'member' as const }
const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Member', value: 'member' },
]

// AFTER
role: z.enum(['admin', 'user'])
initialValues: { ..., role: 'user' as const }
const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Member', value: 'user' },  // display "Member", API value "user"
]
```

#### `stores/runs.ts` — no change needed

The pause/resume paths in `stores/runs.ts` already use the correct typed paths `/projects/{projectId}/runs/{runId}/pause` and `/projects/{projectId}/runs/{runId}/resume`. After regeneration these will be present in the schema and will typecheck cleanly — no code change required.

#### `router/guards.ts` — no change needed

The admin guard checks `auth.user?.role !== 'admin'`, which works correctly with both `'admin' | 'member'` and `'admin' | 'user'`. No code change required; the type narrowing improves automatically when `User.role` is updated.

### Testing Requirements

```bash
# Regenerate schema
cd frontend && npm run generate:api

# Type check
npm run type-check

# Unit tests
npm run test:unit

# Optionally run E2E smoke to validate login/auth flows still work
# (requires e2e stack to be up)
./scripts/e2e-stack.sh reset
cd frontend && npm run test:e2e:real -- --grep "login"
```

## Dev Agent Record

_To be filled by the implementing agent._

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2026-02-22 | story-writer | Initial story created |
