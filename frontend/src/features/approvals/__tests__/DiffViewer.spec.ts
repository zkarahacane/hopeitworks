import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import DiffViewer from '../DiffViewer.vue'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(props: {
  diff: string | null | undefined
  mode: 'side-by-side' | 'line-by-line'
}) {
  wrapper = mount(DiffViewer, {
    props,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('DiffViewer', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders "No diff available" message when diff is null', () => {
    mountComponent({ diff: null, mode: 'side-by-side' })
    expect(wrapper.text()).toContain('No diff available')
  })

  it('renders "No diff available" message when diff is undefined', () => {
    mountComponent({ diff: undefined, mode: 'side-by-side' })
    expect(wrapper.text()).toContain('No diff available')
  })

  it('renders "No diff available" message when diff is empty string', () => {
    mountComponent({ diff: '', mode: 'side-by-side' })
    expect(wrapper.text()).toContain('No diff available')
  })

  it('renders diff html when diff content is provided', () => {
    const diffContent = [
      '--- a/foo.go',
      '+++ b/foo.go',
      '@@ -1,1 +1,1 @@',
      '-old line',
      '+new line',
    ].join('\n')

    mountComponent({ diff: diffContent, mode: 'side-by-side' })
    expect(wrapper.find('.d2h-wrapper').exists()).toBe(true)
  })

  it('does not show toggle button when diff is null', () => {
    mountComponent({ diff: null, mode: 'side-by-side' })
    const buttons = wrapper.findAll('button')
    expect(buttons.length).toBe(0)
  })

  it('shows toggle button when diff is provided', () => {
    mountComponent({
      diff: '--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b',
      mode: 'side-by-side',
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toContain('Unified')
  })

  it('emits update:mode when toggle button is clicked', async () => {
    mountComponent({
      diff: '--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b',
      mode: 'side-by-side',
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:mode')).toEqual([['line-by-line']])
  })

  it('shows "Side by side" label when in line-by-line mode', () => {
    mountComponent({
      diff: '--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b',
      mode: 'line-by-line',
    })
    const button = wrapper.find('button')
    expect(button.text()).toContain('Side by side')
  })
})
