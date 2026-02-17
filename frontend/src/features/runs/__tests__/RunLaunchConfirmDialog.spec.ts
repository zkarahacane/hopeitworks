import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { h, defineComponent } from 'vue'
import PrimeVue from 'primevue/config'
import RunLaunchConfirmDialog from '../RunLaunchConfirmDialog.vue'

/** Stub Dialog to render inline instead of teleporting. */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header'],
  emits: ['update:visible'],
  setup(props, { slots, emit }) {
    return () => {
      if (!props.visible) return null
      return h('div', { class: 'p-dialog' }, [
        h('div', { class: 'p-dialog-header' }, props.header),
        h('div', { class: 'p-dialog-content' }, slots.default?.()),
        h('div', { class: 'p-dialog-footer' }, slots.footer?.()),
      ])
    }
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountComponent(props: {
  visible?: boolean
  storyKey?: string
  storyTitle?: string
  loading?: boolean
}) {
  wrapper = mount(RunLaunchConfirmDialog, {
    props: {
      visible: true,
      storyKey: 'S-01',
      storyTitle: 'Test Story',
      loading: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
      stubs: {
        Dialog: DialogStub,
      },
    },
  })
  return wrapper
}

describe('RunLaunchConfirmDialog', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders dialog with story key, title, and warning when visible', () => {
    mountComponent({})
    expect(wrapper.text()).toContain('Launch Story Run')
    expect(wrapper.text()).toContain('S-01')
    expect(wrapper.text()).toContain('Test Story')
    expect(wrapper.text()).toContain('Claude API credits')
  })

  it('does not render content when not visible', () => {
    mountComponent({ visible: false })
    expect(wrapper.find('.p-dialog').exists()).toBe(false)
  })

  it('renders Cancel and Confirm buttons in footer', () => {
    mountComponent({})
    const buttons = wrapper.findAll('button')
    const cancelBtn = buttons.find((b) => b.text().includes('Cancel'))
    const confirmBtn = buttons.find((b) => b.text().includes('Confirm'))
    expect(cancelBtn).toBeDefined()
    expect(confirmBtn).toBeDefined()
  })

  it('emits confirm when Confirm button is clicked', async () => {
    mountComponent({})
    const confirmBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('Confirm'))
    await confirmBtn!.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('emits cancel and update:visible(false) when Cancel is clicked', async () => {
    mountComponent({})
    const cancelBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('Cancel'))
    await cancelBtn!.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.emitted('update:visible')).toBeDefined()
    expect(wrapper.emitted('update:visible')![0]![0]).toBe(false)
  })

  it('disables Confirm button when loading is true', () => {
    mountComponent({ loading: true })
    const confirmBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('Confirm'))
    expect(confirmBtn!.attributes('disabled')).toBeDefined()
  })

  it('enables Confirm button when loading is false', () => {
    mountComponent({ loading: false })
    const confirmBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('Confirm'))
    expect(confirmBtn!.attributes('disabled')).toBeUndefined()
  })

  it('displays resource usage warning text', () => {
    mountComponent({})
    expect(wrapper.text()).toContain(
      'Launching this run will start an AI agent container',
    )
    expect(wrapper.text()).toContain('Do you want to proceed?')
  })
})
