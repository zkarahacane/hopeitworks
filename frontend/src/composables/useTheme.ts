import { computed, watch, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { useLocalStorage } from '@vueuse/core'

/**
 * useTheme — the single, canonical theme controller.
 *
 * Three persisted modes:
 *  - `auto`  → follow the active route's `route.meta.theme` (runtime surfaces
 *              are dark, management surfaces are light). Missing meta ⇒ dark,
 *              matching the product's runtime-first default. This folds in the
 *              former `useRouteTheme` logic — there is no second toggler.
 *  - `dark`  → force dark globally, regardless of route.
 *  - `light` → force light globally, regardless of route.
 *
 * The resolved scheme drives the `.dark` class on <html>, which is PrimeVue's
 * configured `darkModeSelector` AND the selector our scheme-aware design tokens
 * key off (assets/main.css). Applied reactively: on mode change always, and on
 * route change when in `auto`.
 *
 * The chosen MODE (not the resolved scheme) is persisted in localStorage so the
 * user's intent survives reloads; `auto` then re-derives per route on load.
 *
 * Wire once at the app shell root (AppShell calls useTheme()). The toggle
 * control lives in AppHeader and binds to the returned `mode` ref.
 */

export type ThemeMode = 'auto' | 'dark' | 'light'

export const THEME_STORAGE_KEY = 'hope-theme-mode'

/**
 * Singleton mode ref so the header toggle and the shell share ONE source of
 * truth. Lazily created on first useTheme() call (not at import) so it reads the
 * current localStorage value at that point — and so tests can reset it via
 * `__resetThemeForTests()` between cases.
 */
let modeSingleton: Ref<ThemeMode> | null = null

function getMode(): Ref<ThemeMode> {
  if (!modeSingleton) {
    modeSingleton = useLocalStorage<ThemeMode>(THEME_STORAGE_KEY, 'auto')
  }
  return modeSingleton
}

/** Test-only: drop the singleton so the next useTheme() re-reads localStorage. */
export function __resetThemeForTests() {
  modeSingleton = null
}

/**
 * Resolve a route's declared scheme. Missing meta ⇒ dark (runtime-first).
 * `light` is the only value that maps to light; anything else is dark.
 */
function routeScheme(theme: unknown): 'dark' | 'light' {
  return theme === 'light' ? 'light' : 'dark'
}

export function useTheme() {
  const route = useRoute()
  const mode = getMode()

  /** The scheme actually applied to <html>, given the mode + current route. */
  const resolvedScheme = computed<'dark' | 'light'>(() => {
    if (mode.value === 'dark') return 'dark'
    if (mode.value === 'light') return 'light'
    // auto → follow the route's declared theme.
    return routeScheme(route.meta?.theme)
  })

  const apply = (scheme: 'dark' | 'light') => {
    document.documentElement.classList.toggle('dark', scheme === 'dark')
  }

  // React to BOTH mode changes (force) and route changes (auto branch): both
  // feed resolvedScheme, so a single watcher covers everything. immediate so the
  // class is applied on mount / first restore.
  watch(resolvedScheme, apply, { immediate: true })

  /** Cycle order for a single-button control: auto → dark → light → auto. */
  function cycle() {
    mode.value = mode.value === 'auto' ? 'dark' : mode.value === 'dark' ? 'light' : 'auto'
  }

  function setMode(next: ThemeMode) {
    mode.value = next
  }

  return { mode, resolvedScheme, cycle, setMode }
}
