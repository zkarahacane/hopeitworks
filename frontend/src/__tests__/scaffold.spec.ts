import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import PrimeVue from 'primevue/config'
import { HopeTheme } from '@/theme'
import App from '@/App.vue'
import TestView from '@/views/TestView.vue'

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div>Home</div>' } },
      { path: '/test', component: TestView },
    ],
  })
}

describe('Vue 3 Scaffold', () => {
  it('mounts App component with router', async () => {
    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [createPinia(), router, [PrimeVue, { theme: { preset: HopeTheme } }]],
      },
    })

    expect(wrapper.exists()).toBe(true)
  })

  it('renders TestView with PrimeVue Button', async () => {
    const router = createTestRouter()
    await router.push('/test')
    await router.isReady()

    const wrapper = mount(TestView, {
      global: {
        plugins: [createPinia(), router, [PrimeVue, { theme: { preset: HopeTheme } }]],
      },
    })

    expect(wrapper.text()).toContain('hopeitworks Frontend Scaffold')
    expect(wrapper.text()).toContain('PrimeVue Button')
    expect(wrapper.text()).toContain('Secondary Button')
  })

  it('TestView contains Tailwind utility classes', async () => {
    const router = createTestRouter()
    await router.push('/test')
    await router.isReady()

    const wrapper = mount(TestView, {
      global: {
        plugins: [createPinia(), router, [PrimeVue, { theme: { preset: HopeTheme } }]],
      },
    })

    const rootDiv = wrapper.find('div')
    expect(rootDiv.classes()).toContain('flex')
    expect(rootDiv.classes()).toContain('flex-col')
    expect(rootDiv.classes()).toContain('items-center')
    expect(rootDiv.classes()).toContain('min-h-screen')
  })
})

describe('PrimeVue Configuration', () => {
  it('HopeTheme is defined and exports correctly', () => {
    expect(HopeTheme).toBeDefined()
    expect(typeof HopeTheme).toBe('object')
  })
})

describe('Project Structure', () => {
  it('imports from @/theme resolve correctly', async () => {
    const themeModule = await import('@/theme')
    expect(themeModule.HopeTheme).toBeDefined()
  })

  it('imports from @/theme/tokens resolve correctly', async () => {
    const tokensModule = await import('@/theme/tokens')
    expect(tokensModule.tokenReference).toBeDefined()
    expect(tokensModule.tokenReference.primitive).toBeDefined()
    expect(tokensModule.tokenReference.semantic).toBeDefined()
    expect(tokensModule.tokenReference.component).toBeDefined()
  })
})
