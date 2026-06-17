import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import { useRouteTheme } from '../useRouteTheme'

const Host = defineComponent({
  setup() {
    useRouteTheme()
    return () => h('div')
  },
})

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

describe('useRouteTheme', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark')
  })

  afterEach(() => {
    document.documentElement.classList.remove('dark')
  })

  it('applies .dark on <html> for a dark route on mount', async () => {
    const router = makeRouter()
    await router.push('/dark')
    await router.isReady()

    mount(Host, { global: { plugins: [router] } })
    await nextTick()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('removes .dark for a light route', async () => {
    const router = makeRouter()
    await router.push('/light')
    await router.isReady()

    mount(Host, { global: { plugins: [router] } })
    await nextTick()

    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('defaults to dark when meta.theme is missing', async () => {
    const router = makeRouter()
    await router.push('/no-meta')
    await router.isReady()

    mount(Host, { global: { plugins: [router] } })
    await nextTick()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('flips the class reactively on navigation', async () => {
    const router = makeRouter()
    await router.push('/light')
    await router.isReady()

    mount(Host, { global: { plugins: [router] } })
    await nextTick()
    expect(document.documentElement.classList.contains('dark')).toBe(false)

    await router.push('/dark')
    await nextTick()
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    await router.push('/light')
    await nextTick()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })
})
