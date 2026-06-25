/**
 * NotFoundView — unit tests (#298).
 *
 * The 404 subtext used to be hardcoded to "/settings now lives under your
 * profile." for ANY unresolved URL, which was misleading. These tests pin the
 * fixed behaviour: a generic subtext plus the requested path, with no "/settings"
 * mention, and the path rendered via {{ }} interpolation (auto-escaped, no XSS).
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { ref } from 'vue'
import PrimeVue from 'primevue/config'
import NotFoundView from '../NotFoundView.vue'

// Configurable mock route: each test sets fullPath before mounting.
const mockFullPath = ref('/')

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => ({ fullPath: mockFullPath.value }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountAt(path: string) {
  mockFullPath.value = path
  wrapper = mount(NotFoundView, {
    global: { plugins: [PrimeVue] },
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
})

describe('NotFoundView (#298)', () => {
  it('RG1: shows a generic subtext for any unknown URL, never mentioning /settings', () => {
    mountAt('/xyz-bidon')
    const text = wrapper.text()
    expect(text).toContain("We couldn't find")
    expect(text).not.toContain('/settings')
    expect(text).not.toContain('now lives under your profile')
  })

  it('RG2: a directly-typed /overview shows no out-of-context settings migration copy', () => {
    mountAt('/overview')
    const text = wrapper.text()
    expect(text).not.toContain('/settings')
    expect(text).not.toContain('now lives under your profile')
  })

  it('RG4: includes the requested path in the subtext', () => {
    mountAt('/xyz-bidon')
    expect(wrapper.text()).toContain('/xyz-bidon')
  })

  it('RG4: renders the path via interpolation, escaping a malicious URL (no XSS)', () => {
    mountAt('/<img src=x onerror=alert(1)>')
    // The raw markup must appear as escaped text inside <code>, never as a real node.
    const code = wrapper.find('code')
    expect(code.exists()).toBe(true)
    expect(code.text()).toContain('<img src=x onerror=alert(1)>')
    // No injected <img> element should exist in the rendered DOM.
    expect(wrapper.find('img').exists()).toBe(false)
  })

  it('no longer renders the legacy "Profile & settings" action', () => {
    mountAt('/xyz-bidon')
    expect(wrapper.text()).not.toContain('Profile & settings')
    expect(wrapper.text()).toContain('Go to dashboard')
  })
})
