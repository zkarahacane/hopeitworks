import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import Tooltip from 'primevue/tooltip'
import RunLaunchButton from '../RunLaunchButton.vue'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(props: {
  storyId: string
  storyKey: string
  storyTitle: string
  status: 'backlog' | 'running' | 'done' | 'failed'
}) {
  wrapper = mount(RunLaunchButton, {
    props,
    global: {
      plugins: [PrimeVue],
      directives: { tooltip: Tooltip },
    },
  })
  return wrapper
}

describe('RunLaunchButton', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders "Launch Run" button when status is backlog', () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'backlog',
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toContain('Launch Run')
    expect(button.attributes('disabled')).toBeUndefined()
  })

  it('emits launchClick when button is clicked in backlog state', async () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'backlog',
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('launchClick')).toHaveLength(1)
  })

  it('renders disabled "Running..." button when status is running', () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'running',
    })
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect(button.text()).toContain('Running...')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('does not render any button when status is done', () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'done',
    })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('does not render any button when status is failed', () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'failed',
    })
    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('does not emit launchClick when running button is clicked', async () => {
    mountComponent({
      storyId: 's-1',
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      status: 'running',
    })
    const button = wrapper.find('button')
    await button.trigger('click')
    expect(wrapper.emitted('launchClick')).toBeUndefined()
  })
})
