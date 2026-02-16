# Story 1.7: [FRONT] Vue scaffolding + PrimeVue + Tailwind setup

Status: review

## Story

As a developer,
I want a Vue 3 project with PrimeVue and Tailwind configured,
So that I can build features with correct UI framework and styling foundation.

## Acceptance Criteria (BDD)

**Given** frontend setup is complete
**When** I run `npm run dev`
**Then** dev server starts on port 5173, PrimeVue renders with Aura unstyled preset, Tailwind layout utilities work, CSS layers are configured in correct order (tailwind-base → primevue → tailwind-utilities)

## Tasks / Subtasks

- [x] Initialize Vue 3 project with TypeScript (AC: dev server starts)
  - [x] Run `npm create vue@latest frontend` with TypeScript, Vitest, ESLint, Prettier options
  - [x] Verify package.json contains correct Vue 3 dependencies
  - [x] Verify tsconfig.json is configured correctly
  - [x] Test that `npm run dev` starts Vite dev server
  - [x] Configure Vite proxy to `/api/v1` → `http://localhost:8080/api/v1`

- [x] Create project structure following architecture (AC: dev server starts)
  - [x] Create `src/ui/primitives/` directory
  - [x] Create `src/ui/composed/` directory
  - [x] Create `src/ui/layout/` directory
  - [x] Create `src/features/` directory
  - [x] Create `src/composables/` directory
  - [x] Create `src/stores/` directory
  - [x] Create `src/views/` directory
  - [x] Create `src/utils/` directory
  - [x] Create `src/theme/` directory
  - [x] Create `src/router/` directory
  - [x] Create `src/api/` directory

- [x] Install and configure PrimeVue 4 with Aura preset (AC: PrimeVue renders with Aura unstyled)
  - [x] Install PrimeVue 4.x and primeicons via npm
  - [x] Create theme configuration in `src/theme/index.ts` with Aura preset
  - [x] Configure PrimeVue in unstyled mode with darkModeSelector: '.dark'
  - [x] Import PrimeVue in main.ts with theme config
  - [x] Create design tokens file `src/theme/tokens.ts` with 3-level token hierarchy (primitive → semantic → component)

- [x] Install and configure Tailwind CSS v4 (AC: Tailwind layout utilities work)
  - [x] Install Tailwind CSS v4 and @tailwindcss/vite plugin
  - [x] Configure @tailwindcss/vite plugin in vite.config.ts (Tailwind v4 uses Vite plugin instead of postcss/config file)
  - [x] No separate tailwind.config.js or postcss.config.js needed (Tailwind v4 CSS-first approach)
  - [x] Configure Tailwind to work alongside PrimeVue

- [x] Configure CSS layers for proper style precedence (AC: CSS layers are configured)
  - [x] Create `src/assets/main.css` with @layer directives
  - [x] Define layer order: `tailwind-base, primevue, tailwind-utilities`
  - [x] Import main.css in main.ts before App.vue
  - [x] Verify layer order via build output
  - [x] Create smoke test view `src/views/TestView.vue` to verify PrimeVue Button renders and Tailwind utilities apply

## Dev Notes

### Architecture Requirements

This story implements the **frontend foundation shell** for hopeitworks, creating a Vue 3 SPA with UI frameworks configured. The frontend is **completely independent** from the backend — different directory, different build, different agent.

**Key Architectural Principles:**
- **Strict separation:** Frontend agents NEVER touch `backend/`, backend agents NEVER touch `frontend/`
- **Shared contract:** `api/openapi.yaml` is the single coupling point (used in story 1-16)
- **Hybrid structure:** Shared UI components in `ui/`, domain features in `features/`
- **Composition API only:** No Options API usage
- **Progressive disclosure:** Components are visual assemblers, composables contain logic

**Frontend Architecture Model:**
- **Component Library:** PrimeVue 4 (unstyled mode with Aura preset)
- **Styling:** Tailwind CSS v4 for layout utilities only

### Technical Specifications

**Exact Versions to Install:**

```json
{
  "dependencies": {
    "vue": "^3.5.0",
    "primevue": "^4.3.0",
    "primeicons": "^7.0.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.0",
    "vite": "^6.0.0",
    "typescript": "~5.7.0",
    "@vue/tsconfig": "^0.7.0",
    "vitest": "^3.0.0",
    "@vue/test-utils": "^2.4.0",
    "eslint": "^9.17.0",
    "@vue/eslint-config-typescript": "^14.1.0",
    "@vue/eslint-config-prettier": "^10.1.0",
    "prettier": "^3.4.0",
    "tailwindcss": "^4.0.0",
    "postcss": "^8.4.0",
    "autoprefixer": "^10.4.0"
  }
}
```

**Node/npm Requirements:**
- Node.js: 20.x or 22.x LTS
- Package manager: npm (default from create-vue)
- Dev server port: 5173 (Vite default)
- API proxy: Configure Vite proxy to `/api/v1` → `http://localhost:8080/api/v1` (backend API)

**TypeScript Configuration:**
- `strict: true`
- `moduleResolution: "bundler"`
- Path aliases: `@/` → `./src/`
- Vue 3 reactivity types enabled

**Vite Configuration:**
- Dev server port: 5173
- Proxy configuration for backend API
- Environment variables prefix: `VITE_`

**PrimeVue Configuration:**
```typescript
// src/theme/index.ts
import { definePreset } from '@primevue/themes';
import Aura from '@primevue/themes/aura';

export const HopeTheme = definePreset(Aura, {
  semantic: {
    primary: {
      50: '{blue.50}',
      100: '{blue.100}',
      // ... semantic mappings
    },
    // ... other semantic tokens
  }
});
```

**PrimeVue Setup in main.ts:**
```typescript
import PrimeVue from 'primevue/config';
import { HopeTheme } from '@/theme';

app.use(PrimeVue, {
  theme: {
    preset: HopeTheme,
    options: {
      darkModeSelector: '.dark'
    }
  },
  unstyled: true
});
```

**CSS Layer Configuration:**
```css
/* src/assets/main.css */
@layer tailwind-base, primevue, tailwind-utilities;

@layer tailwind-base {
  @tailwind base;
}

@layer primevue {
  /* PrimeVue component styles will be injected here */
}

@layer tailwind-utilities {
  @tailwind components;
  @tailwind utilities;
}
```

### File Structure

Create the following directory structure in `frontend/`:

```
frontend/
├── src/
│   ├── ui/                          # Atomic layer (shared only)
│   │   ├── primitives/              # PrimeVue wrappers, base components
│   │   ├── composed/                # Reusable combinations
│   │   └── layout/                  # Page structure
│   ├── features/                    # By business domain
│   │   ├── projects/
│   │   ├── stories/
│   │   ├── runs/
│   │   ├── dag/
│   │   ├── approvals/
│   │   └── pipeline-editor/
│   ├── composables/                 # Shared functional (pure)
│   ├── stores/                      # Pinia stores
│   ├── api/                         # openapi-fetch client (populated in 1-16)
│   ├── theme/                       # PrimeVue tokens + config
│   │   ├── tokens.ts
│   │   └── index.ts
│   ├── assets/                      # main.css
│   │   └── main.css
│   ├── router/                      # Routes with auth guards (configured in 1-16)
│   ├── views/                       # 1 view = 1 route
│   ├── utils/                       # Pure functions
│   ├── App.vue
│   └── main.ts
├── e2e/                             # Playwright tests (future)
├── public/
├── package.json
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.vitest.json
├── vite.config.ts
├── eslint.config.js
├── .prettierrc
├── tailwind.config.js
└── index.html
```

**Files to Create in This Story:**

1. **Project initialization:**
   - `package.json` (via create-vue)
   - `tsconfig.json`, `tsconfig.app.json`, `tsconfig.vitest.json`
   - `vite.config.ts`
   - `index.html`

2. **Configuration files:**
   - `tailwind.config.js`
   - `eslint.config.js`
   - `.prettierrc`
   - `postcss.config.js` (if needed)

3. **Source files:**
   - `src/main.ts` — Application entry point with PrimeVue setup
   - `src/App.vue` — Root component (minimal shell for now)
   - `src/assets/main.css` — CSS layers configuration
   - `src/theme/index.ts` — PrimeVue theme configuration
   - `src/theme/tokens.ts` — Design tokens

4. **Directory structure:**
   - All directories listed in "File Structure" section above

5. **Smoke test component:**
   - `src/views/TestView.vue` — Simple view to verify PrimeVue + Tailwind work

### Testing Requirements

**At Scaffolding Stage:**

1. **Dev server runs successfully:**
   - `npm install` completes without errors
   - `npm run dev` starts Vite server on port 5173
   - Browser opens to `http://localhost:5173` and shows app

2. **PrimeVue renders correctly:**
   - Test view displays a PrimeVue Button component
   - Button renders with Aura theme styling
   - Button is interactive (clickable)

3. **Tailwind utilities work:**
   - Flex, grid, spacing utilities apply correctly
   - Layout classes do not conflict with PrimeVue

4. **CSS layers are correct:**
   - Inspect computed styles in browser DevTools
   - Verify layer order: tailwind-base → primevue → tailwind-utilities
   - Verify Tailwind utilities can override PrimeVue styles

5. **TypeScript compilation works:**
   - `npm run build` succeeds
   - No TypeScript errors in IDE
   - Type checking with `vue-tsc` passes

**Manual Smoke Test:**

Create a simple test component that verifies all integrations:

```vue
<!-- src/views/TestView.vue -->
<script setup lang="ts">
import Button from 'primevue/button';
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-4 p-8">
    <h1 class="text-4xl font-bold">
      hopeitworks Frontend Scaffold
    </h1>

    <div class="flex gap-4">
      <Button severity="primary">
        PrimeVue Button
      </Button>

      <Button severity="secondary">
        Secondary Button
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
  </div>
</template>
```

**Acceptance Validation:**

After completing all tasks, verify:
- [ ] Dev server starts without errors
- [ ] Test view renders PrimeVue Button with Aura styling
- [ ] Tailwind layout utilities work correctly
- [ ] CSS layers are in correct order (inspect in DevTools)
- [ ] TypeScript compilation succeeds
- [ ] All directories created per architecture

### Component Naming Convention

- Components: PascalCase.vue (e.g., `ProjectList.vue`, `RunTimeline.vue`)
- Composables: camelCase with `use` prefix (e.g., `useTheme.ts`, `useSSE.ts`)
- Stores: domain noun (e.g., `auth.ts`, `projects.ts`, `runs.ts`)
- Utils: camelCase (e.g., `formatDate.ts`, `parseNdjson.ts`)

### Style Conventions for Agents

1. **PrimeVue first** — Use PrimeVue components for everything they provide
2. **Tailwind for layout** — flex, grid, gap, padding, margin only
3. **Zero custom CSS** — No `<style scoped>` blocks except for complex animations or SVG
4. **No inline styles** — Use PrimeVue severity props instead of inline color styles

### References

- [Source: architecture.md, Section "Frontend Architecture"]
  - Package layout: Lines 100-127
  - PrimeVue setup: Lines 978-989
  - Style conventions: Lines 984-989
  - CSS layers: Line 399

- [Source: architecture.md, Section "Project Structure Decision"]
  - Monorepo structure: Lines 62-165
  - Boundary rules: Lines 167-172
  - Selected approach: Lines 186-204

- [Source: architecture.md, Section "Stack Decisions"]
  - Frontend stack: Lines 237-243

- [Source: epics.md, Epic 1, Story 1.7]
  - User story: Lines 550-552
  - Acceptance criteria: Lines 554-558

- [Source: prd.md, Section "Technical Architecture"]
  - Integration points: Lines 156-166

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- Tailwind CSS v4 uses a different configuration model than v3: no `tailwind.config.js` or `postcss.config.js` required. Instead, uses `@tailwindcss/vite` plugin and CSS-first configuration via `@import "tailwindcss/..."` directives with `layer()` assignments.
- `@primevue/themes` package is deprecated in favor of `@primeuix/themes`, but still works for PrimeVue 4.x and includes `@primeuix/themes` as a transitive dependency.
- `create-vue@3.21.1` scaffolds with Vite 7.x, Vue 3.5.x, TypeScript 5.9.x, Vitest 4.x — all newer than story specs but backward-compatible.

### Completion Notes List

- Scaffolded Vue 3 project with TypeScript, Vitest, ESLint, Prettier, Vue Router, Pinia
- Created full architecture directory structure: ui/{primitives,composed,layout}, features/, composables/, stores/, views/, utils/, theme/, router/, api/
- Installed PrimeVue 4.5.4 with Aura preset, configured HopeTheme with blue primary palette
- Installed Tailwind CSS v4.1.18 with @tailwindcss/vite plugin
- Configured CSS layers: tailwind-base → primevue → tailwind-utilities
- Created TestView.vue smoke test with PrimeVue Button and Tailwind layout utilities
- Configured Vite proxy: /api/v1 → http://localhost:8080/api/v1
- All tests pass (6 unit tests), build succeeds, type-check passes, lint passes
- Removed default create-vue boilerplate (HelloWorld, TheWelcome, etc.) and replaced with minimal App shell

### Change Log

- 2026-02-16: Initial implementation of Vue 3 scaffold with PrimeVue 4 + Tailwind CSS v4

### File List

New files:
- frontend/package.json
- frontend/tsconfig.json
- frontend/tsconfig.app.json
- frontend/tsconfig.node.json
- frontend/tsconfig.vitest.json
- frontend/vite.config.ts
- frontend/vitest.config.ts
- frontend/index.html
- frontend/eslint.config.js
- frontend/.prettierrc.json
- frontend/env.d.ts
- frontend/src/main.ts
- frontend/src/App.vue
- frontend/src/assets/main.css
- frontend/src/theme/index.ts
- frontend/src/theme/tokens.ts
- frontend/src/views/HomeView.vue
- frontend/src/views/TestView.vue
- frontend/src/router/index.ts
- frontend/src/stores/counter.ts
- frontend/src/__tests__/scaffold.spec.ts
- frontend/src/ui/primitives/.gitkeep
- frontend/src/ui/composed/.gitkeep
- frontend/src/ui/layout/.gitkeep
- frontend/src/features/.gitkeep
- frontend/src/composables/.gitkeep
- frontend/src/utils/.gitkeep
- frontend/src/api/.gitkeep
- frontend/e2e/.gitkeep

Modified files:
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/1-7-vue-scaffolding-primevue-tailwind-setup.md
