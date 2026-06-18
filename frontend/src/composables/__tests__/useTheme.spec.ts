import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import { useTheme, THEME_STORAGE_KEY, __resetThemeForTests } from '../useTheme'

type Theme = ReturnType<typeof useTheme>

/** Host that captures the composable return so tests can assert on the refs. */
function makeHost(capture: (t: Theme) => void) {
  return defineComponent({
    setup() {
      capture(useTheme())
      return () => h('div')
    },
  })
}

function makeRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dark', component: { template: '<div/>' }, meta: { theme: 'dark' } },
      { path: '/light', component: { template: '<div/>' }, meta: { theme: 'light' } },
      { path: '/no-meta', component: { template: '<div/>' } },
    ],
  })
}

async function mountAt(path: string) {
  const router = makeRouter()
  await router.push(path)
  await router.isReady()
  let theme!: Theme
  mount(makeHost((t) => (theme = t)), { global: { plugins: [router] } })
  await nextTick()
  return { theme, router }
}

const isDark = () => document.documentElement.classList.contains('dark')

describe('useTheme', () => {
  beforeEach(() => {
    // The mode ref is a lazily-created singleton; drop it AND clear storage +
    // the document class so each test starts from a clean slate.
    __resetThemeForTests()
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  afterEach(() => {
    __resetThemeForTests()
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  describe('auto mode (default) follows route.meta.theme', () => {
    it('applies dark for a dark route', async () => {
      const { theme } = await mountAt('/dark')
      expect(theme.mode.value).toBe('auto')
      expect(isDark()).toBe(true)
    })

    it('applies light for a light route', async () => {
      await mountAt('/light')
      expect(isDark()).toBe(false)
    })

    it('defaults to dark when meta.theme is missing', async () => {
      await mountAt('/no-meta')
      expect(isDark()).toBe(true)
    })

    it('flips reactively on navigation', async () => {
      const { router } = await mountAt('/light')
      expect(isDark()).toBe(false)

      await router.push('/dark')
      await nextTick()
      expect(isDark()).toBe(true)

      await router.push('/light')
      await nextTick()
      expect(isDark()).toBe(false)
    })
  })

  describe('forced modes ignore the route', () => {
    it('dark stays dark on a light route', async () => {
      const { theme, router } = await mountAt('/light')
      theme.setMode('dark')
      await nextTick()
      expect(isDark()).toBe(true)

      await router.push('/light')
      await nextTick()
      expect(isDark()).toBe(true)
    })

    it('light stays light on a dark route', async () => {
      const { theme, router } = await mountAt('/dark')
      theme.setMode('light')
      await nextTick()
      expect(isDark()).toBe(false)

      await router.push('/dark')
      await nextTick()
      expect(isDark()).toBe(false)
    })
  })

  describe('resolution + cycle', () => {
    it('cycles auto → dark → light → auto', async () => {
      const { theme } = await mountAt('/light')
      expect(theme.mode.value).toBe('auto')

      theme.cycle()
      await nextTick()
      expect(theme.mode.value).toBe('dark')
      expect(theme.resolvedScheme.value).toBe('dark')

      theme.cycle()
      await nextTick()
      expect(theme.mode.value).toBe('light')
      expect(theme.resolvedScheme.value).toBe('light')

      theme.cycle()
      await nextTick()
      expect(theme.mode.value).toBe('auto')
      // auto on /light → light
      expect(theme.resolvedScheme.value).toBe('light')
    })
  })

  describe('persistence', () => {
    it('writes the chosen mode to localStorage', async () => {
      const { theme } = await mountAt('/no-meta')
      theme.setMode('light')
      await nextTick()
      // useLocalStorage stores string values raw (no JSON quoting).
      expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
    })

    it('restores the persisted mode on load and forces its scheme', async () => {
      // Simulate a prior session that chose dark (raw string, as VueUse writes).
      localStorage.setItem(THEME_STORAGE_KEY, 'dark')
      const { theme } = await mountAt('/light')
      expect(theme.mode.value).toBe('dark')
      // forced dark wins over the light route
      expect(isDark()).toBe(true)
    })
  })
})
