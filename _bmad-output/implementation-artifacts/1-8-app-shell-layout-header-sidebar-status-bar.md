# Story 1.8: [FRONT] App shell layout — Header, Sidebar, Status Bar

Status: ready-for-dev

## Story

As a logged-in user,
I want a responsive app shell with header, sidebar, and status bar,
So that I can navigate efficiently on all devices.

## Acceptance Criteria (BDD)

**AC1: Layout structure renders correctly**
- **Given** the user is logged in
- **When** they access any page
- **Then** header (48px), sidebar (240px collapsible), and status bar (24px) display correctly

**AC2: Desktop sidebar toggle**
- **Given** desktop viewport (>=1024px)
- **When** I press the `[` key
- **Then** sidebar toggles between 240px (full) and 48px (icons only)

**AC3: Mobile responsive layout**
- **Given** mobile viewport (<1024px)
- **When** the page loads
- **Then** sidebar is hidden, hamburger menu appears in header, bottom nav with 4 tabs shows

**AC4: Semantic HTML and accessibility**
- **Given** the layout renders
- **When** I inspect the HTML
- **Then** semantic elements (`<nav>`, `<main>`, `<aside>`) and a skip navigation link are present

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create Pinia `useLayoutStore` (AC: #2)
  - [ ] Create `frontend/src/stores/layout.ts`
  - [ ] Define state: `sidebarCollapsed: boolean` (default `false`)
  - [ ] Define action: `toggleSidebar()` to flip `sidebarCollapsed`
  - [ ] Persist `sidebarCollapsed` to `localStorage` key `layout-sidebar-collapsed`
  - [ ] Export typed store

- [ ] [FRONT] Task 2: Create composables `useKeyboard` and `useBreakpoint` (AC: #2, #3)
  - [ ] Create `frontend/src/composables/useKeyboard.ts`
  - [ ] Accept a map of `{ key: handler }` bindings, register on `keydown`, cleanup on unmount
  - [ ] Ignore shortcuts when focus is inside `<input>`, `<textarea>`, or `[contenteditable]`
  - [ ] Create `frontend/src/composables/useBreakpoint.ts`
  - [ ] Expose reactive `isMobile` ref (true when `window.innerWidth < 1024`)
  - [ ] Use `matchMedia('(max-width: 1023px)')` listener, cleanup on unmount

- [ ] [FRONT] Task 3: Create `AppHeader.vue` (AC: #1, #3, #4)
  - [ ] Create `frontend/src/ui/layout/AppHeader.vue`
  - [ ] Render a `<header>` element, fixed height 48px (`h-12`)
  - [ ] Desktop: show app logo/name on the left, placeholder user menu on the right
  - [ ] Mobile: show hamburger button (PrimeVue `Button` icon `pi pi-bars`) that emits `toggle-sidebar`
  - [ ] Use PrimeVue `Toolbar` or `Button` components, Tailwind for layout (`flex`, `items-center`, `justify-between`)

- [ ] [FRONT] Task 4: Create `AppSidebar.vue` (AC: #1, #2, #3, #4)
  - [ ] Create `frontend/src/ui/layout/AppSidebar.vue`
  - [ ] Render an `<aside>` with `<nav>` inside
  - [ ] Desktop: width transitions between 240px (`w-60`) and 48px (`w-12`) based on `useLayoutStore().sidebarCollapsed`
  - [ ] Collapsed mode: show icons only (PrimeVue `Button` with `icon` prop, no label)
  - [ ] Expanded mode: show icons + labels using PrimeVue `Menu` or `PanelMenu`
  - [ ] Mobile: render as overlay/drawer, controlled by parent via prop `mobileOpen: boolean`
  - [ ] Navigation items: Dashboard, Projects, Runs, Settings (placeholder routes)
  - [ ] CSS transition on width change (`transition-all duration-200`)

- [ ] [FRONT] Task 5: Create `AppStatusBar.vue` (AC: #1)
  - [ ] Create `frontend/src/ui/layout/AppStatusBar.vue`
  - [ ] Render a `<footer>` element, fixed height 24px (`h-6`)
  - [ ] Show placeholder text: connection status indicator (left), version string (right)
  - [ ] Use Tailwind for layout (`flex`, `items-center`, `justify-between`, `text-xs`)

- [ ] [FRONT] Task 6: Create `AppShell.vue` and wire everything (AC: #1, #2, #3, #4)
  - [ ] Create `frontend/src/ui/layout/AppShell.vue`
  - [ ] Compose AppHeader, AppSidebar, AppStatusBar, and `<router-view>` inside `<main>`
  - [ ] Add skip navigation link (`<a href="#main-content" class="sr-only focus:not-sr-only">`) before header
  - [ ] Wire `useKeyboard` to bind `[` key to `layoutStore.toggleSidebar()` on desktop
  - [ ] Wire `useBreakpoint` to control mobile vs desktop rendering
  - [ ] Mobile: hide sidebar, show bottom nav (4 tabs: Dashboard, Projects, Runs, Settings) using PrimeVue `TabMenu` or `Button` group
  - [ ] Desktop: grid layout — sidebar left, content right, header top, status bar bottom
  - [ ] Set `id="main-content"` on the `<main>` element for skip-nav target
  - [ ] Update `frontend/src/App.vue` or router to use `AppShell` as the layout wrapper

## Dev Notes

This story builds the **responsive app shell** that wraps all authenticated routes. It creates the persistent layout (header, sidebar, status bar) and the composables for keyboard shortcuts and responsive breakpoints. No actual navigation logic or auth guards are implemented here — routing comes from Story 1-16, auth from Story 1-9.

### Architecture Requirements

**Component Hierarchy:**
```
AppShell.vue
├── <a> (skip navigation link)
├── AppHeader.vue
│   ├── Logo / App name
│   ├── Hamburger button (mobile only)
│   └── User menu placeholder
├── AppSidebar.vue
│   └── <nav> with menu items
├── <main id="main-content">
│   └── <router-view />
└── AppStatusBar.vue
    ├── Connection status
    └── Version string
```

**Desktop Layout (CSS Grid):**
```
┌──────────────────────────────────────┐
│            AppHeader (48px)          │
├──────────┬───────────────────────────┤
│          │                           │
│ Sidebar  │       <main>             │
│ 240/48px │     <router-view />      │
│          │                           │
├──────────┴───────────────────────────┤
│          AppStatusBar (24px)         │
└──────────────────────────────────────┘
```

**Mobile Layout (<1024px):**
```
┌──────────────────────────────────────┐
│    AppHeader (48px) + hamburger      │
├──────────────────────────────────────┤
│                                      │
│           <main>                     │
│         <router-view />             │
│                                      │
├──────────────────────────────────────┤
│       Bottom Nav (4 tabs)            │
└──────────────────────────────────────┘
  + Sidebar as overlay drawer when open
```

### Technical Specifications

**Exact File Paths:**
```
frontend/src/
├── stores/
│   └── layout.ts                  # Pinia store — sidebarCollapsed state
├── composables/
│   ├── useKeyboard.ts             # Keyboard shortcut registration
│   └── useBreakpoint.ts           # Reactive isMobile breakpoint
└── ui/layout/
    ├── AppShell.vue               # Root layout wrapper
    ├── AppHeader.vue              # Top bar (48px)
    ├── AppSidebar.vue             # Left sidebar (240/48px)
    └── AppStatusBar.vue           # Bottom status bar (24px)
```

**Pinia Store — `useLayoutStore`:**
```typescript
// frontend/src/stores/layout.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLayoutStore = defineStore('layout', () => {
  const sidebarCollapsed = ref(
    localStorage.getItem('layout-sidebar-collapsed') === 'true'
  )

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    localStorage.setItem(
      'layout-sidebar-collapsed',
      String(sidebarCollapsed.value)
    )
  }

  return { sidebarCollapsed, toggleSidebar }
})
```

**Composable — `useKeyboard`:**
```typescript
// frontend/src/composables/useKeyboard.ts
// Signature: useKeyboard(bindings: Record<string, () => void>): void
// - Registers keydown listeners on mount, removes on unmount
// - Ignores events when activeElement is input/textarea/contenteditable
// - Key values use KeyboardEvent.key (e.g., '[', 'Escape', 'k')
```

**Composable — `useBreakpoint`:**
```typescript
// frontend/src/composables/useBreakpoint.ts
// Signature: useBreakpoint(): { isMobile: Ref<boolean> }
// - Uses window.matchMedia('(max-width: 1023px)')
// - Listens to 'change' event on MediaQueryList
// - Cleans up listener on onUnmounted
```

**Component Props & Emits:**

| Component | Props | Emits |
|---|---|---|
| `AppShell` | none | none |
| `AppHeader` | `showHamburger: boolean` | `toggle-sidebar` |
| `AppSidebar` | `collapsed: boolean`, `mobileOpen: boolean` | `close` (mobile overlay) |
| `AppStatusBar` | none | none |

**PrimeVue Components to Use:**
- `Button` — hamburger toggle, sidebar collapse button, bottom nav items
- `Menu` or `PanelMenu` — sidebar navigation items
- `Toolbar` — header bar (optional, can use plain `<header>` + Tailwind)
- `TabMenu` — mobile bottom navigation (optional, can use Button group)

**Responsive Breakpoint:**
- Mobile: `< 1024px` (Tailwind `lg` breakpoint)
- Desktop: `>= 1024px`

**Sidebar Dimensions:**
- Expanded: `240px` (`w-60`)
- Collapsed: `48px` (`w-12`)
- Transition: `transition-all duration-200 ease-in-out`

**Header/Status Bar:**
- Header height: `48px` (`h-12`)
- Status bar height: `24px` (`h-6`)

**Keyboard Shortcut:**
- Key: `[` (left bracket)
- Action: toggle `layoutStore.sidebarCollapsed`
- Only active on desktop (ignore when `isMobile` is true)

**Style Rules:**
- Zero custom CSS (`<style>` blocks) — Tailwind utility classes only
- PrimeVue components for interactive elements
- Tailwind for layout (flex, grid, gap, padding, sizing)

### Testing Requirements

**Manual verification checklist:**
1. `npm run dev` — layout renders with header (48px), sidebar (240px), status bar (24px)
2. Press `[` key — sidebar toggles to 48px (icons only), press again — back to 240px
3. Refresh page — sidebar state persists (localStorage)
4. Resize browser to <1024px — sidebar disappears, hamburger appears in header, bottom nav shows
5. Click hamburger — sidebar opens as overlay, click outside or close — it dismisses
6. Inspect HTML — `<header>`, `<aside>`, `<nav>`, `<main>`, `<footer>` elements present
7. Tab from top of page — skip navigation link becomes visible on focus
8. `npm run build` — no TypeScript errors

### Navigation Items (Placeholder)

The sidebar and mobile bottom nav should include these 4 items with PrimeIcons:
1. **Dashboard** — `pi pi-home` — route: `/`
2. **Projects** — `pi pi-folder` — route: `/projects`
3. **Runs** — `pi pi-play` — route: `/runs`
4. **Settings** — `pi pi-cog` — route: `/settings`

Routes do not need to exist yet. Use `router-link` or `to` prop where applicable.

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]
  - Package layout: `ui/layout/` with AppShell.vue
  - Composables: `useKeyboard.ts`
  - PrimeVue components: Menubar + PanelMenu (sidebar)

- [Source: _bmad-output/planning-artifacts/architecture.md#Component Library]
  - Navigation: Menubar + PanelMenu (sidebar)
  - Style convention: PrimeVue first, Tailwind for layout, zero custom CSS

- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.8]
  - AC: header 48px, sidebar 240px collapsible, status bar 24px
  - Keyboard: `[` toggles sidebar
  - Mobile: <1024px, hamburger, bottom nav 4 tabs
  - Semantic HTML + skip nav

- [Source: _bmad-output/implementation-artifacts/1-7-vue-scaffolding-primevue-tailwind-setup.md]
  - Dependency: PrimeVue 4.5, Tailwind v4, Pinia, vue-router already configured

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6 (claude-opus-4-6)

### Debug Log References

- Build: `npm run build` — passes with 0 errors (252 modules transformed)
- Lint: `npm run lint` — 0 warnings, 0 errors (oxlint + eslint)

### Completion Notes List

- Installed Pinia as a new dependency and registered it in `main.ts`
- Created `useLayoutStore` Pinia store with `sidebarCollapsed` state persisted to localStorage
- Created `useKeyboard` composable for keydown bindings with input/textarea guard
- Created `useBreakpoint` composable using `matchMedia` for reactive `isMobile` ref
- Created `AppHeader.vue` with PrimeVue Button for hamburger and user menu placeholder
- Created `AppSidebar.vue` with collapsible desktop sidebar (240/48px) and mobile overlay drawer
- Created `AppStatusBar.vue` with connection status indicator and version string
- Created `AppShell.vue` wiring all components with skip-nav link, keyboard shortcut (`[`), responsive layout
- Updated `App.vue` to use `AppShell` as the root layout wrapper
- Added placeholder routes for `/projects`, `/runs`, `/settings` in router
- All semantic HTML elements used: `<header>`, `<aside>`, `<nav>`, `<main>`, `<footer>`
- Zero custom CSS — Tailwind utility classes only, PrimeVue components for interactivity

### File List

- `frontend/src/stores/layout.ts` — Pinia store for sidebar collapsed state
- `frontend/src/composables/useKeyboard.ts` — Keyboard shortcut composable
- `frontend/src/composables/useBreakpoint.ts` — Responsive breakpoint composable
- `frontend/src/ui/layout/AppHeader.vue` — Header component (48px)
- `frontend/src/ui/layout/AppSidebar.vue` — Sidebar component (240/48px collapsible)
- `frontend/src/ui/layout/AppStatusBar.vue` — Status bar component (24px)
- `frontend/src/ui/layout/AppShell.vue` — Root layout shell
- `frontend/src/App.vue` — Updated to use AppShell
- `frontend/src/router/index.ts` — Updated with placeholder routes
- `frontend/src/main.ts` — Updated to register Pinia

## Change Log

- 2026-02-16: Initial implementation of Story 1-8
