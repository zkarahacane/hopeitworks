import { computed, watch, type Ref } from 'vue'
import { useLocalStorage, usePreferredDark } from '@vueuse/core'

/**
 * useTheme — the single, canonical theme controller.
 *
 * Three persisted modes, applied GLOBALLY and stable across navigation
 * (the theme never changes just because you switched page/tab):
 *  - `auto`  → follow the OS / browser preference (`prefers-color-scheme`),
 *              reactive to system changes.
 *  - `dark`  → force dark.
 *  - `light` → force light.
 *
 * The resolved scheme drives the `.dark` class on <html>, which is PrimeVue's
 * configured `darkModeSelector` AND the selector our scheme-aware design tokens
 * key off (assets/main.css).
 *
 * The chosen MODE (not the resolved scheme) is persisted in localStorage so the
 * user's intent survives reloads. Wire once at the app shell root (AppShell
 * calls useTheme()); the toggle control in AppHeader binds to `mode`/`cycle`.
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

export function useTheme() {
  const mode = getMode()
  const prefersDark = usePreferredDark()

  /** The scheme actually applied to <html>. `auto` follows the system. */
  const resolvedScheme = computed<'dark' | 'light'>(() => {
    if (mode.value === 'dark') return 'dark'
    if (mode.value === 'light') return 'light'
    return prefersDark.value ? 'dark' : 'light'
  })

  // One watcher covers mode changes and (in auto) system-preference changes.
  // immediate so the class is applied on mount / first restore.
  watch(
    resolvedScheme,
    (scheme) => document.documentElement.classList.toggle('dark', scheme === 'dark'),
    { immediate: true },
  )

  /** Cycle order for a single-button control: auto → dark → light → auto. */
  function cycle() {
    mode.value = mode.value === 'auto' ? 'dark' : mode.value === 'dark' ? 'light' : 'auto'
  }

  function setMode(next: ThemeMode) {
    mode.value = next
  }

  return { mode, resolvedScheme, cycle, setMode }
}
