# Story F-3.1: [FRONT] Add dark mode toggle to app header

Status: ready-for-dev

## Story

As a user,
I want to toggle between light and dark mode,
so that I can use the app comfortably in different lighting conditions.

## Acceptance Criteria (BDD)

**AC1: Toggle button visible in header**
- **Given** user is authenticated
- **When** viewing any page
- **Then** a dark/light mode toggle button is visible in the app header (right side, before the user menu button)

**AC2: Clicking toggle switches theme**
- **Given** app is in light mode
- **When** user clicks the toggle
- **Then** `.dark` class is added to `<html>`, PrimeVue switches to dark palette (Aura dark surface tokens activate)

**AC3: Preference persisted**
- **Given** user has selected dark mode
- **When** they refresh the page or return later
- **Then** dark mode is still active (preference key `hope-theme` persisted in localStorage)

**AC4: System preference respected as default**
- **Given** user has not set a preference in localStorage
- **When** they load the app for the first time
- **Then** the theme matches their OS preference (`prefers-color-scheme: dark`)

## Tasks / Subtasks

### Task 1: Create `useTheme` composable

File to create: `frontend/src/composables/useTheme.ts`

- Use `@vueuse/core` `usePreferredDark` and `useLocalStorage` (both already available — see `frontend/package.json`)
- localStorage key: `hope-theme`, values: `'light' | 'dark' | null` (null = unset, fall back to OS preference)
- Expose:
  - `isDark: ComputedRef<boolean>` — derived from stored preference or OS fallback
  - `toggle(): void` — flips the stored preference between `'light'` and `'dark'`
- On composable init (and on `isDark` change), sync the `.dark` class on `document.documentElement`:
  - `isDark` is `true` → add class `dark`
  - `isDark` is `false` → remove class `dark`
- Use `watchEffect` or `watch` with `immediate: true` to apply the class on every change
- The composable is stateless (no Pinia store needed — localStorage is the source of truth)

### Task 2: Add toggle button to `AppHeader.vue`

File to modify: `frontend/src/ui/layout/AppHeader.vue`

- Import and call `useTheme()` in `<script setup>`
- Add a PrimeVue `Button` component in the right section of the header (between the left logo area and the existing user menu button), using:
  - `:icon="isDark ? 'pi pi-sun' : 'pi pi-moon'"`
  - `text` + `rounded` props (matches the existing hamburger and user-menu button style)
  - `aria-label` toggling between `'Switch to light mode'` and `'Switch to dark mode'`
  - `data-testid="theme-toggle-button"`
  - `@click="toggle"`
- No new imports beyond `useTheme` and `Button` (already imported)

### Task 3: Apply dark class on app init

File to modify: `frontend/src/main.ts`

- Import `useTheme` and call it once before `app.mount('#app')` so the `.dark` class is applied immediately on page load (before first render), preventing a flash of wrong theme
- Note: composables normally require an active Vue instance; use `app.runWithContext(() => useTheme())` or restructure as a plain function call that reads localStorage + `window.matchMedia` directly at module level if the composable pattern is incompatible outside a component

  **Preferred alternative (avoids complexity):** extract the "apply class on load" logic into a small standalone function `applyInitialTheme()` exported from `useTheme.ts`, callable outside Vue context, and call it at the top of `main.ts` before `createApp`.

### Task 4: Write unit tests for `useTheme`

File to create: `frontend/src/composables/__tests__/useTheme.spec.ts`

- Use Vitest + `@vueuse/core` test utilities
- Test cases:
  - No localStorage entry + OS dark preference → `isDark` is `true`, `.dark` on `<html>`
  - No localStorage entry + OS light preference → `isDark` is `false`, no `.dark` on `<html>`
  - localStorage `'dark'` overrides OS light preference → `isDark` is `true`
  - localStorage `'light'` overrides OS dark preference → `isDark` is `false`
  - `toggle()` from light → sets localStorage to `'dark'`, adds `.dark` to `<html>`
  - `toggle()` from dark → sets localStorage to `'light'`, removes `.dark` from `<html>`

## Dev Notes

- Priority: P2
- `@vueuse/core` is already a dependency — use `usePreferredDark` and `useLocalStorage`
- PrimeVue 4 Aura preset handles dark palette automatically via the `.dark` selector on `<html>` — no PrimeVue API calls needed, just DOM class manipulation
- `darkModeSelector: '.dark'` is already configured in `frontend/src/main.ts` (line 24)
- Dark surface tokens are already defined in `frontend/src/theme/tokens.ts` — no theme changes needed
- `main.css` has no dark-mode CSS to add — PrimeVue CSS layers handle it
- The `AppHeader.vue` right-side section currently only has a `Button` + `Menu` for user menu; add the toggle button before it in the same `div.flex.items-center.gap-2`
- No backend changes, no API spec changes, no Pinia store changes required
