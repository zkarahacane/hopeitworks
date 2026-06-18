import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { useTheme, THEME_STORAGE_KEY, __resetThemeForTests } from '../useTheme'

type Theme = ReturnType<typeof useTheme>

/** Stub window.matchMedia so usePreferredDark() resolves to the given value. */
function stubMatchMedia(prefersDark: boolean) {
  window.matchMedia = (query: string) =>
    ({
      matches: query === '(prefers-color-scheme: dark)' ? prefersDark : false,
      media: query,
      addEventListener() {},
      removeEventListener() {},
      addListener() {},
      removeListener() {},
      dispatchEvent() {
        return false
      },
      onchange: null,
    }) as unknown as MediaQueryList
}

/** Mount the composable host and return the captured theme instance. */
function mountTheme(): Theme {
  let theme!: Theme
  const Host = defineComponent({
    setup() {
      theme = useTheme()
      return () => h('div')
    },
  })
  mount(Host)
  return theme
}

const isDark = () => document.documentElement.classList.contains('dark')

describe('useTheme', () => {
  beforeEach(() => {
    __resetThemeForTests()
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    // Default stub: system preference = light
    stubMatchMedia(false)
  })

  afterEach(() => {
    __resetThemeForTests()
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  // ─── auto mode: follows system preference ────────────────────────────────

  describe('auto mode (default) follows system preference', () => {
    it('applies dark when system prefers dark', async () => {
      stubMatchMedia(true)
      __resetThemeForTests() // re-mount picks up the new matchMedia stub
      const theme = mountTheme()
      await nextTick()
      expect(theme.mode.value).toBe('auto')
      expect(isDark()).toBe(true)
      expect(theme.resolvedScheme.value).toBe('dark')
    })

    it('applies light when system prefers light', async () => {
      stubMatchMedia(false)
      __resetThemeForTests()
      const theme = mountTheme()
      await nextTick()
      expect(theme.mode.value).toBe('auto')
      expect(isDark()).toBe(false)
      expect(theme.resolvedScheme.value).toBe('light')
    })
  })

  // ─── forced modes ignore system preference ────────────────────────────────

  describe('forced modes ignore the system preference', () => {
    it('dark stays dark even when system prefers light', async () => {
      stubMatchMedia(false)
      __resetThemeForTests()
      const theme = mountTheme()
      theme.setMode('dark')
      await nextTick()
      expect(isDark()).toBe(true)
      expect(theme.resolvedScheme.value).toBe('dark')
    })

    it('light stays light even when system prefers dark', async () => {
      stubMatchMedia(true)
      __resetThemeForTests()
      const theme = mountTheme()
      theme.setMode('light')
      await nextTick()
      expect(isDark()).toBe(false)
      expect(theme.resolvedScheme.value).toBe('light')
    })
  })

  // ─── cycle ────────────────────────────────────────────────────────────────

  describe('cycle()', () => {
    it('cycles auto → dark → light → auto', async () => {
      stubMatchMedia(false) // system = light so auto resolves to light
      __resetThemeForTests()
      const theme = mountTheme()
      await nextTick()
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
      // auto + system light → light
      expect(theme.resolvedScheme.value).toBe('light')
    })
  })

  // ─── persistence ─────────────────────────────────────────────────────────

  describe('persistence', () => {
    it('writes the chosen mode to localStorage', async () => {
      const theme = mountTheme()
      theme.setMode('light')
      await nextTick()
      expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light')
    })

    it('restores the persisted mode on load', async () => {
      localStorage.setItem(THEME_STORAGE_KEY, 'dark')
      __resetThemeForTests() // drop singleton so next mount re-reads localStorage
      const theme = mountTheme()
      await nextTick()
      expect(theme.mode.value).toBe('dark')
      expect(isDark()).toBe(true)
    })

    it('persisted light mode wins over dark system preference', async () => {
      localStorage.setItem(THEME_STORAGE_KEY, 'light')
      stubMatchMedia(true) // system prefers dark
      __resetThemeForTests()
      const theme = mountTheme()
      await nextTick()
      expect(theme.mode.value).toBe('light')
      expect(isDark()).toBe(false)
    })
  })

  // ─── isolation between cases ──────────────────────────────────────────────

  describe('__resetThemeForTests() isolation', () => {
    it('starts fresh after reset (no bleed from previous mode)', async () => {
      const t1 = mountTheme()
      t1.setMode('dark')
      await nextTick()
      expect(isDark()).toBe(true)

      // Simulate what beforeEach does
      __resetThemeForTests()
      localStorage.clear()
      document.documentElement.classList.remove('dark')

      stubMatchMedia(false)
      const t2 = mountTheme()
      await nextTick()
      expect(t2.mode.value).toBe('auto')
      expect(isDark()).toBe(false)
    })
  })
})
