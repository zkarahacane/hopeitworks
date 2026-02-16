# Story 1.16: Vue app routing + state + tooling

Status: ready-for-dev

## Story

As a frontend developer,
I want Vue Router routes, Pinia stores, openapi-fetch client, and base composables configured,
so that the frontend has complete application infrastructure for feature development.

## Acceptance Criteria (BDD)

**AC1: Route definitions work with placeholder views**
- **Given** the Vue app is running via `npm run dev`
- **When** I navigate to `/`, `/login`, `/projects`, `/projects/:id`, `/runs/:id`, `/approvals`
- **Then** each route renders its placeholder view component

**AC2: Pinia stores are scaffolded and functional**
- **Given** Pinia is installed and configured in `main.ts`
- **When** I import `useAuthStore`, `useProjectsStore`, `useStoriesStore`, `useRunsStore` in a component
- **Then** each store is accessible with its typed state shape (empty shells)

**AC3: openapi-fetch client is generated and typed**
- **Given** `api/openapi.yaml` exists
- **When** I run `npm run generate:api`
- **Then** `src/api/schema.d.ts` is generated and `apiClient` in `src/api/client.ts` is typed against it

**AC4: Base composables exist with correct signatures**
- **Given** the composables directory exists
- **When** I import `useAsyncAction` and `usePagination`
- **Then** they export the documented reactive interface (loading, error, execute / page, perPage, total)

**AC5: API error interceptor handles common errors**
- **Given** the API client is configured
- **When** a request returns 401
- **Then** the interceptor redirects to `/login`

**AC6: Vitest runs successfully**
- **Given** Vitest is configured from story 1-7
- **When** I run `npm run test:unit`
- **Then** tests pass (including at least one smoke test for a composable)

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Vue Router route definitions with placeholder views (AC: #1)
  - [ ] Create placeholder view files: `LoginView.vue`, `DashboardView.vue`, `ProjectsView.vue`, `ProjectDetailView.vue`, `RunDetailView.vue`, `ApprovalsView.vue`
  - [ ] Update `src/router/index.ts` with all route definitions
  - [ ] Add `beforeEach` navigation guard placeholder (commented, ready for auth story)
  - [ ] Verify all routes render their placeholder view

- [ ] [FRONT] Task 2: Pinia stores scaffolding (AC: #2)
  - [ ] Install Pinia: `npm install pinia`
  - [ ] Register Pinia in `src/main.ts`
  - [ ] Create `src/stores/auth.ts` — empty shell with typed state
  - [ ] Create `src/stores/projects.ts` — empty shell with typed state
  - [ ] Create `src/stores/stories.ts` — empty shell with typed state
  - [ ] Create `src/stores/runs.ts` — empty shell with typed state

- [ ] [FRONT] Task 3: openapi-fetch client setup (AC: #3)
  - [ ] Install openapi-typescript and openapi-fetch: `npm install openapi-fetch && npm install -D openapi-typescript`
  - [ ] Add `generate:api` script to `package.json`
  - [ ] Run generation to create `src/api/schema.d.ts`
  - [ ] Create `src/api/client.ts` with typed client (`credentials: 'include'`, `baseUrl: '/api/v1'`)

- [ ] [FRONT] Task 4: Base composables — useAsyncAction + usePagination (AC: #4)
  - [ ] Create `src/composables/useAsyncAction.ts`
  - [ ] Create `src/composables/usePagination.ts`

- [ ] [FRONT] Task 5: API error interceptor (AC: #5)
  - [ ] Add middleware/interceptor in `src/api/client.ts` for 401 → redirect to `/login`
  - [ ] Add Toast-ready error shape for 4xx/5xx responses

- [ ] [FRONT] Task 6: Vitest smoke tests for composables (AC: #6)
  - [ ] Create `src/composables/__tests__/useAsyncAction.spec.ts`
  - [ ] Create `src/composables/__tests__/usePagination.spec.ts`
  - [ ] Verify `npm run test:unit` passes

## Dev Notes

This story adds **application-level infrastructure** on top of the 1-7 scaffold (Vue 3 + PrimeVue + Tailwind + Router shell). No UI features — only plumbing.

### Dependencies on Story 1-7

**Already exists from 1-7 (do NOT recreate):**
- Vue 3 project at `frontend/`
- PrimeVue 4 configured with Aura preset + CSS layers
- Tailwind CSS v4 installed
- Vue Router installed (`vue-router@^5.0.2`) and registered in `main.ts`
- Basic `src/router/index.ts` with a single `/` route pointing to `TestView`
- Directory structure: `src/composables/`, `src/stores/`, `src/router/`, `src/api/`, `src/views/`, `src/features/`, `src/ui/`, `src/utils/`
- Vitest configured
- ESLint + Prettier + oxlint configured

**This story adds:**
- Route definitions for all application views (placeholder components)
- Pinia state management (install + configure + store shells)
- openapi-fetch typed API client (install + generate + configure)
- Base composables (useAsyncAction, usePagination)
- API error interceptor
- Composable unit tests

### Architecture Requirements

**Route Definitions:**

```typescript
// src/router/index.ts
const routes = [
  { path: '/login', name: 'login', component: LoginView },
  { path: '/', name: 'dashboard', component: DashboardView },
  { path: '/projects', name: 'projects', component: ProjectsView },
  { path: '/projects/:id', name: 'project-detail', component: ProjectDetailView },
  { path: '/runs/:id', name: 'run-detail', component: RunDetailView },
  { path: '/approvals', name: 'approvals', component: ApprovalsView },
]
```

Routes for stories, DAG, pipeline-editor will be added in their respective feature stories. Keep it minimal.

**Pinia Store Interfaces (empty shells):**

```typescript
// src/stores/auth.ts
export const useAuthStore = defineStore('auth', () => {
  const user = ref<{ id: string; name: string; role: string } | null>(null)
  const isAuthenticated = computed(() => user.value !== null)
  return { user, isAuthenticated }
})

// src/stores/projects.ts
export const useProjectsStore = defineStore('projects', () => {
  const items = ref<Array<{ id: string; name: string }>>([])
  const current = ref<{ id: string; name: string } | null>(null)
  const isLoading = ref(false)
  return { items, current, isLoading }
})

// src/stores/stories.ts
export const useStoriesStore = defineStore('stories', () => {
  const items = ref<Array<{ id: string; summary: string; status: string }>>([])
  const isLoading = ref(false)
  return { items, isLoading }
})

// src/stores/runs.ts
export const useRunsStore = defineStore('runs', () => {
  const items = ref<Array<{ id: string; status: string }>>([])
  const current = ref<{ id: string; status: string; steps: Array<unknown> } | null>(null)
  const isLoading = ref(false)
  return { items, current, isLoading }
})
```

Use Composition API (`setup()` syntax) for all stores. Types are intentionally lightweight — real types will come from `schema.d.ts` in feature stories.

**Composable Signatures:**

```typescript
// src/composables/useAsyncAction.ts
export function useAsyncAction<T>(fn: (...args: any[]) => Promise<T>) {
  const data = ref<T | null>(null)
  const error = ref<Error | null>(null)
  const isLoading = ref(false)

  async function execute(...args: any[]): Promise<T | null> {
    isLoading.value = true
    error.value = null
    try {
      data.value = await fn(...args)
      return data.value
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
      return null
    } finally {
      isLoading.value = false
    }
  }

  return { data, error, isLoading, execute }
}

// src/composables/usePagination.ts
export function usePagination(options?: { perPage?: number }) {
  const page = ref(1)
  const perPage = ref(options?.perPage ?? 20)
  const total = ref(0)
  const totalPages = computed(() => Math.ceil(total.value / perPage.value))

  function setTotal(n: number) { total.value = n }
  function nextPage() { if (page.value < totalPages.value) page.value++ }
  function prevPage() { if (page.value > 1) page.value-- }
  function goToPage(n: number) { page.value = Math.max(1, Math.min(n, totalPages.value)) }
  function reset() { page.value = 1 }

  return { page, perPage, total, totalPages, setTotal, nextPage, prevPage, goToPage, reset }
}
```

**API Client:**

```typescript
// src/api/client.ts
import createClient, { type Middleware } from 'openapi-fetch'
import type { paths } from './schema'
import router from '@/router'

const authMiddleware: Middleware = {
  async onResponse({ response }) {
    if (response.status === 401) {
      await router.push({ name: 'login' })
    }
    return response
  },
}

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include',
})

apiClient.use(authMiddleware)
```

**openapi-typescript generation script:**

```json
{
  "generate:api": "openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts"
}
```

### File Structure

**Files created by this story:**

1. **Views (placeholder components, minimal `<template>` with route name):**
   - `frontend/src/views/LoginView.vue`
   - `frontend/src/views/DashboardView.vue`
   - `frontend/src/views/ProjectsView.vue`
   - `frontend/src/views/ProjectDetailView.vue`
   - `frontend/src/views/RunDetailView.vue`
   - `frontend/src/views/ApprovalsView.vue`

2. **Stores:**
   - `frontend/src/stores/auth.ts`
   - `frontend/src/stores/projects.ts`
   - `frontend/src/stores/stories.ts`
   - `frontend/src/stores/runs.ts`

3. **API Client:**
   - `frontend/src/api/client.ts`
   - `frontend/src/api/schema.d.ts` (generated, gitignored)

4. **Composables:**
   - `frontend/src/composables/useAsyncAction.ts`
   - `frontend/src/composables/usePagination.ts`

5. **Tests:**
   - `frontend/src/composables/__tests__/useAsyncAction.spec.ts`
   - `frontend/src/composables/__tests__/usePagination.spec.ts`

**Files modified by this story:**

6. **Updated:**
   - `frontend/src/router/index.ts` — full route definitions
   - `frontend/src/main.ts` — add Pinia registration
   - `frontend/package.json` — add `pinia`, `openapi-fetch`, `openapi-typescript`, `generate:api` script

### Testing Requirements

**Automated tests:**
1. `useAsyncAction` unit test — verify loading/error/data lifecycle, error handling
2. `usePagination` unit test — verify page navigation, bounds, reset

**Manual verification checklist:**
1. `npm run dev` starts without errors
2. Navigate to each defined route — placeholder view renders
3. `npm run generate:api` produces `src/api/schema.d.ts`
4. `npm run test:unit` passes all tests
5. `npm run lint` passes without errors
6. No TypeScript errors (`npm run type-check`)

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]
- [Source: _bmad-output/planning-artifacts/architecture.md#Hybrid Structure: Feature + Atomic for Shared]
- [Source: _bmad-output/planning-artifacts/architecture.md#Stack Decisions — Frontend]
- [Source: _bmad-output/planning-artifacts/architecture.md#API & Communication Patterns]
- [Source: _bmad-output/planning-artifacts/architecture.md#Key Frontend Libraries]

## Dev Agent Record

### Agent Model Used

(To be filled by implementation agent)

### Debug Log References

(To be filled by implementation agent)

### Completion Notes List

(To be filled by implementation agent)

### File List

(To be filled by implementation agent)

## Change Log
