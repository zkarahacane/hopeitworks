import { watch } from 'vue'
import { useRoute } from 'vue-router'

/**
 * useRouteTheme — applies the active route's color scheme to the document.
 *
 * Each route declares `meta.theme: 'dark' | 'light'` (see router/index.ts):
 *  - `dark`  → runtime / observability surfaces (dashboard, runs, DAG, costs…)
 *  - `light` → management / config surfaces (projects, board, agents, settings…)
 *
 * It toggles the `.dark` class on `<html>`, which is PrimeVue's configured
 * `darkModeSelector` AND the selector our scheme-aware design tokens key off
 * (assets/main.css + theme/tokens.ts). Routes without `meta.theme` default to
 * `dark`, matching the product's runtime-first default surface.
 *
 * Wire once at the app shell root. Call this from a component `setup()` so the
 * watcher is bound to that component's lifecycle.
 *
 * Follow-up (out of scope): a persisted manual user override (light/dark/system)
 * that takes precedence over the per-route default.
 */
export function useRouteTheme() {
  const route = useRoute()

  const apply = (theme: 'dark' | 'light' | undefined) => {
    document.documentElement.classList.toggle('dark', theme !== 'light')
  }

  watch(
    () => route.meta?.theme as 'dark' | 'light' | undefined,
    apply,
    { immediate: true },
  )
}
