# Story 1.16: Vue App Routing, State & Tooling

Status: ready-for-dev

## Story

As a frontend developer,
I want Vue Router, Pinia state management, dark mode, API client generation, and dev tooling configured,
so that the frontend application has complete infrastructure for feature development.

## Acceptance Criteria (BDD)

**Given** story 1-7 is complete (Vue 3 + PrimeVue + Tailwind shell exists)
**When** I run `npm run dev` and `npm run generate:api`
**Then** dark mode toggles via useTheme() composable, API client types are generated from openapi.yaml, routing works with placeholder routes, Pinia stores are available, linting passes, and basic smoke test verifies all integrations

## Tasks / Subtasks

- [ ] Implement dark mode toggle composable (AC: dark mode toggles via useTheme() composable)
  - [ ] Create `src/composables/useTheme.ts`
  - [ ] Implement theme state management with localStorage persistence
  - [ ] Add `.dark` class toggle on `<html>` element
  - [ ] Load theme preference from localStorage on init
  - [ ] Respect system preference if no stored value
  - [ ] Export useTheme composable with isDark ref and toggleTheme function

- [ ] Set up OpenAPI TypeScript client generation (AC: API client types generated from openapi.yaml)
  - [ ] Install openapi-typescript and openapi-fetch
  - [ ] Create npm script `generate:api` to generate types from `../api/openapi.yaml`
  - [ ] Create `src/api/client.ts` with typed fetch client
  - [ ] Configure credentials: 'include' for JWT cookie auth
  - [ ] Configure baseUrl: '/api/v1'
  - [ ] Add generation script to package.json scripts

- [ ] Set up Pinia state management (AC: Pinia stores available)
  - [ ] Install Pinia
  - [ ] Configure Pinia in main.ts
  - [ ] Create stores directory structure (stores/ already exists from 1-7)
  - [ ] Create example store placeholder `src/stores/README.md` documenting store conventions

- [ ] Set up Vue Router (AC: routing works with placeholder routes)
  - [ ] Install Vue Router
  - [ ] Create `src/router/index.ts` with basic routes (home, test view)
  - [ ] Configure router in main.ts
  - [ ] Add navigation guards placeholder for auth (to be implemented later)
  - [ ] Update App.vue to include `<RouterView />`

- [ ] Configure linting and formatting (AC: linting passes)
  - [ ] Verify ESLint configuration from create-vue for Vue 3 + TypeScript
  - [ ] Verify Prettier configuration from create-vue
  - [ ] Add lint scripts to package.json if not present
  - [ ] Test that `npm run lint` executes without errors
  - [ ] Test that `npm run format` formats code consistently

- [ ] Create basic smoke test (AC: all integrations verified)
  - [ ] Update `src/views/TestView.vue` to include dark mode toggle
  - [ ] Verify PrimeVue Button renders in dark mode
  - [ ] Verify Tailwind utilities apply correctly
  - [ ] Verify theme toggle works and persists
  - [ ] Create `frontend/CLAUDE.md` with frontend-specific agent instructions
  - [ ] Document component rules, style conventions, and project structure

## Dev Notes

### Dependencies on Story 1-7

**Required Prerequisites:**
- Vue 3 project must exist at `frontend/`
- PrimeVue 4 must be installed and configured with Aura preset
- Tailwind CSS v4 must be installed
- CSS layers must be configured (tailwind-base → primevue → tailwind-utilities)
- TypeScript must be configured with path aliases (`@/` → `./src/`)
- Package manager (npm) must be set up
- Vite dev server must be working on port 5173
- Directory structure must exist: `src/composables/`, `src/stores/`, `src/router/`, `src/api/`, `src/views/`

**This story adds on top of 1-7:**
- Dark mode functionality
- API client code generation
- State management setup
- Routing setup
- Linting verification
- Complete smoke test

### Architecture Requirements

This story completes the **frontend infrastructure layer** by adding application-level concerns: routing, state management, dark mode, and API client generation.

**Key Architectural Principles:**
- **Code-gen first:** API client types are generated from OpenAPI spec
- **Composition API only:** All logic in composables
- **Dark mode via .dark class:** Applied to `<html>` element, managed by useTheme()
- **Shared contract:** `api/openapi.yaml` is the single coupling point with backend

**Frontend Architecture Components Added:**
- **State Management:** Pinia stores per domain
- **Routing:** Vue Router with auth guards placeholder
- **API Client:** Generated from OpenAPI via openapi-typescript + openapi-fetch
- **Dark Mode:** useTheme() composable

### Technical Specifications

**Additional Versions to Install:**

```json
{
  "dependencies": {
    "vue-router": "^4.5.0",
    "pinia": "^2.3.0",
    "openapi-fetch": "^0.13.0"
  },
  "devDependencies": {
    "openapi-typescript": "^7.4.0"
  }
}
```

**Dark Mode Implementation:**
```typescript
// src/composables/useTheme.ts
import { ref, watch } from 'vue';

const isDark = ref(false);

export function useTheme() {
  // Load from localStorage on init
  const stored = localStorage.getItem('theme');
  if (stored === 'dark' || (!stored && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true;
  }

  // Apply theme class to html element
  const applyTheme = () => {
    if (isDark.value) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  };

  applyTheme();

  // Watch for changes and persist
  watch(isDark, (newVal) => {
    localStorage.setItem('theme', newVal ? 'dark' : 'light');
    applyTheme();
  });

  const toggleTheme = () => {
    isDark.value = !isDark.value;
  };

  return {
    isDark,
    toggleTheme
  };
}
```

**OpenAPI Client Generation:**
```json
// package.json scripts
{
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc && vite build",
    "preview": "vite preview",
    "test:unit": "vitest",
    "lint": "eslint . --ext .vue,.js,.jsx,.cjs,.mjs,.ts,.tsx,.cts,.mts --fix --ignore-path .gitignore",
    "format": "prettier --write src/",
    "generate:api": "openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts"
  }
}
```

```typescript
// src/api/client.ts
import createClient from 'openapi-fetch';
import type { paths } from './schema';

export const apiClient = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include' // Important for JWT cookie auth
});
```

**Vue Router Configuration:**
```typescript
// src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router';
import TestView from '@/views/TestView.vue';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: TestView
    },
    {
      path: '/test',
      name: 'test',
      component: TestView
    }
    // More routes will be added in feature stories
  ]
});

// Placeholder for auth guards (to be implemented in auth story)
// router.beforeEach((to, from) => {
//   // Check auth state, redirect to login if needed
// });

export default router;
```

**Pinia Configuration:**
```typescript
// src/main.ts
import { createPinia } from 'pinia';

const pinia = createPinia();
app.use(pinia);
```

**Store Convention (README.md):**
```markdown
# Pinia Stores

## Naming Convention
- File: domain noun (e.g., `auth.ts`, `projects.ts`, `runs.ts`)
- Store ID: `use<Domain>Store` (e.g., `useAuthStore`, `useProjectsStore`)

## Structure
- State: reactive data
- Getters: computed values
- Actions: methods that modify state or call API

## Example
// src/stores/auth.ts
import { defineStore } from 'pinia';

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    isAuthenticated: false
  }),
  getters: {
    userName: (state) => state.user?.name ?? 'Guest'
  },
  actions: {
    async login(credentials) {
      // Call API, update state
    }
  }
});
```

### File Structure

**Files Created by This Story:**

1. **Composables:**
   - `src/composables/useTheme.ts` — Dark mode toggle

2. **API Client:**
   - `src/api/client.ts` — OpenAPI fetch client
   - `src/api/schema.d.ts` — Generated types (via npm script)

3. **Router:**
   - `src/router/index.ts` — Router setup with placeholder routes

4. **Stores:**
   - `src/stores/README.md` — Store conventions documentation

5. **Updated Files:**
   - `src/main.ts` — Add Pinia and Router setup
   - `src/App.vue` — Add `<RouterView />`
   - `src/views/TestView.vue` — Add dark mode toggle and verify all integrations

6. **Agent Instructions:**
   - `frontend/CLAUDE.md` — Frontend-specific agent instructions

7. **Package.json:**
   - Add `generate:api` script
   - Add router and pinia dependencies

### Testing Requirements

**Verification Steps:**

1. **Dark mode works:**
   - useTheme() composable is callable
   - toggleTheme() switches between light and dark
   - `.dark` class is added/removed from `<html>` element
   - Theme preference persists in localStorage
   - System preference is respected if no stored value

2. **API client types are generated:**
   - `npm run generate:api` executes successfully
   - `src/api/schema.d.ts` is created
   - TypeScript recognizes API types in client.ts
   - No TypeScript errors when importing from `./schema`

3. **Routing works:**
   - `npm run dev` starts without errors
   - Navigating to `/` shows TestView
   - Navigating to `/test` shows TestView
   - Router is configured in main.ts
   - App.vue contains `<RouterView />`

4. **Pinia state management works:**
   - Pinia is configured in main.ts
   - Stores can be defined following conventions
   - No console errors related to Pinia

5. **Linting and formatting work:**
   - `npm run lint` executes without errors
   - `npm run format` formats code consistently
   - ESLint and Prettier configurations are compatible

6. **Smoke test passes:**
   - TestView renders PrimeVue Button
   - Dark mode toggle button works
   - Theme persists across page reloads
   - Tailwind utilities apply correctly in both themes

**Updated Smoke Test:**

```vue
<!-- src/views/TestView.vue -->
<script setup lang="ts">
import { useTheme } from '@/composables/useTheme';
import Button from 'primevue/button';

const { isDark, toggleTheme } = useTheme();
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-4 p-8">
    <h1 class="text-4xl font-bold">
      hopeitworks Frontend Scaffold
    </h1>

    <div class="flex gap-4">
      <Button @click="toggleTheme" severity="secondary">
        Toggle Theme ({{ isDark ? 'Dark' : 'Light' }})
      </Button>

      <Button severity="primary">
        PrimeVue Button
      </Button>
    </div>

    <div class="grid grid-cols-2 gap-4 mt-8">
      <div class="p-4 border rounded">
        <p>Tailwind: flex, gap, grid work</p>
      </div>
      <div class="p-4 border rounded">
        <p>CSS Layers: utilities override PrimeVue</p>
      </div>
    </div>

    <p class="mt-4 text-sm opacity-70">
      Reload page to verify theme persistence
    </p>
  </div>
</template>
```

**Acceptance Validation:**

After completing all tasks, verify:
- [ ] Dark mode toggle works and persists
- [ ] API client types are generated successfully
- [ ] Routing works (navigate to / and /test)
- [ ] Pinia is configured and ready for use
- [ ] Linting passes without errors
- [ ] All integrations verified in smoke test

### CLAUDE.md Content

Create `frontend/CLAUDE.md` with:

```markdown
# Frontend Agent Instructions

## Scope Boundary
- **ONLY** work in `frontend/` directory
- **NEVER** touch `backend/` or `api/` (except reading openapi.yaml for types)
- If you need backend changes, ask user to delegate to backend agent

## Component Rules
1. **PrimeVue first** — Use PrimeVue components for everything they provide
2. **Tailwind for layout** — flex, grid, gap, padding, margin only
3. **Zero custom CSS** — No `<style scoped>` blocks except for complex animations or SVG
4. **No inline styles** — Use PrimeVue severity props instead of inline color styles
5. **Dark mode via .dark class** — Applied to `<html>` element, managed by useTheme()

## Project Structure
- `src/ui/primitives/` — PrimeVue wrappers, base components
- `src/ui/composed/` — Reusable combinations
- `src/ui/layout/` — Page structure components
- `src/features/` — By business domain (projects, stories, runs, dag, approvals, pipeline-editor)
- `src/composables/` — Shared functional composables (pure)
- `src/stores/` — Pinia stores (one per domain)
- `src/api/` — OpenAPI client (generated, do not edit schema.d.ts)
- `src/theme/` — PrimeVue tokens + config
- `src/router/` — Routes with auth guards
- `src/views/` — 1 view = 1 route
- `src/utils/` — Pure utility functions

## Naming Conventions
- Components: PascalCase.vue (e.g., `ProjectList.vue`, `RunTimeline.vue`)
- Composables: camelCase with `use` prefix (e.g., `useTheme.ts`, `useSSE.ts`)
- Stores: domain noun (e.g., `auth.ts`, `projects.ts`, `runs.ts`)
- Utils: camelCase (e.g., `formatDate.ts`, `parseNdjson.ts`)

## Anti-Patterns (DO NOT DO)
- ❌ No Options API (use Composition API only)
- ❌ No custom CSS for colors/sizes (use PrimeVue theme tokens)
- ❌ No inline styles (use Tailwind classes or PrimeVue props)
- ❌ No touching backend code
- ❌ No manually editing `src/api/schema.d.ts` (it's generated)

## API Client Usage
- Run `npm run generate:api` to regenerate types from openapi.yaml
- Import `apiClient` from `@/api/client`
- All API calls use typed endpoints from schema

## State Management
- One Pinia store per domain
- Store ID: `use<Domain>Store` (e.g., `useAuthStore`)
- Actions for API calls, getters for computed values

## Dark Mode
- Use `useTheme()` composable to access `isDark` and `toggleTheme()`
- PrimeVue automatically switches based on `.dark` class on `<html>`
- Theme persists in localStorage
```

### References

- [Source: architecture.md, Section "Frontend Architecture"]
  - PrimeVue setup: Lines 978-989
  - Style conventions: Lines 984-989
  - Dark mode pattern: Line 290 (inferred)

- [Source: architecture.md, Section "Project Structure Decision"]
  - Boundary rules: Lines 167-172
  - Component structure: Lines 100-127

- [Source: architecture.md, Section "Stack Decisions"]
  - Frontend stack: Lines 237-243

- [Source: prd.md, Section "Technical Architecture"]
  - Integration points: Lines 156-166

- [Source: ux-design-specification.md, Section "Design Opportunities"]
  - Progressive disclosure pattern: Lines 62-71

## Dev Agent Record

### Agent Model Used

(To be filled by implementation agent)

### Debug Log References

(To be filled by implementation agent)

### Completion Notes List

(To be filled by implementation agent)

### File List

(To be filled by implementation agent)
