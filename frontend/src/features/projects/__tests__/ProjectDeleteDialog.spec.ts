import { describe, it, expect, afterEach } from 'vitest'
import { mount, type VueWrapper, flushPromises } from '@vue/test-utils'
import { h, defineComponent } from 'vue'
import PrimeVue from 'primevue/config'
import ProjectDeleteDialog from '../ProjectDeleteDialog.vue'

/** Stub Dialog to render inline (header + content + footer) instead of teleporting. */
const DialogStub = defineComponent({
  name: 'DialogStub',
  props: ['visible', 'modal', 'header'],
  emits: ['update:visible'],
  setup(props, { slots }) {
    return () => {
      if (!props.visible) return null
      return h('div', { class: 'p-dialog', 'data-testid': 'project-delete-dialog' }, [
        h('div', { class: 'p-dialog-content' }, slots.default?.()),
        h('div', { class: 'p-dialog-footer' }, slots.footer?.()),
      ])
    }
  },
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function mountDialog(
  props: Partial<{ visible: boolean; projectName: string; loading: boolean }> = {},
) {
  wrapper = mount(ProjectDeleteDialog, {
    props: {
      visible: true,
      projectName: 'Test Project',
      loading: false,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
      stubs: { Dialog: DialogStub },
    },
  })
  return wrapper
}

function confirmBtn() {
  return wrapper.find('[data-testid="delete-confirm-btn"]')
}

async function typeName(value: string) {
  const input = wrapper.find('[data-testid="delete-confirm-input"]')
  await input.setValue(value)
  await flushPromises()
}

describe('ProjectDeleteDialog', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  // RG3: cascade warning is explicit about permanent deletion of linked data
  it('shows an explicit cascade warning (RG3)', () => {
    mountDialog()
    const warning = wrapper.find('[data-testid="delete-cascade-warning"]')
    expect(warning.exists()).toBe(true)
    const text = warning.text()
    expect(text).toContain('runs')
    expect(text).toContain('stories')
    expect(text).toContain('epics')
    expect(text).toContain('configs')
    expect(text.toLowerCase()).toContain('cannot be undone')
  })

  // RG2: confirm button disabled until the typed name matches exactly
  it('keeps the confirm button disabled while the typed name is empty (RG2)', () => {
    mountDialog()
    expect(confirmBtn().attributes('disabled')).toBeDefined()
  })

  it('keeps the confirm button disabled on a partial / wrong name (RG2)', async () => {
    mountDialog()
    await typeName('Test Projec')
    expect(confirmBtn().attributes('disabled')).toBeDefined()
  })

  it('enables the confirm button when the typed name matches exactly (RG2)', async () => {
    mountDialog()
    await typeName('Test Project')
    expect(confirmBtn().attributes('disabled')).toBeUndefined()
  })

  // RG1 happy path: confirm emits when name matches
  it('emits confirm when the name matches and confirm is clicked (RG1)', async () => {
    mountDialog()
    await typeName('Test Project')
    await confirmBtn().trigger('click')
    expect(wrapper.emitted('confirm')).toBeDefined()
  })

  it('does not emit confirm when the name does not match', async () => {
    mountDialog()
    await typeName('wrong')
    await confirmBtn().trigger('click')
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  // RG7: cancel emits cancel + closes, never confirm
  it('emits cancel and closes on Cancel without emitting confirm (RG7)', async () => {
    mountDialog()
    await wrapper.find('[data-testid="delete-cancel-btn"]').trigger('click')
    expect(wrapper.emitted('cancel')).toBeDefined()
    expect(wrapper.emitted('update:visible')?.[0]).toEqual([false])
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  it('disables the confirm button while loading even when the name matches (anti double-click)', async () => {
    mountDialog({ loading: true })
    await typeName('Test Project')
    expect(confirmBtn().attributes('disabled')).toBeDefined()
  })

  it('resets the typed name when the dialog reopens', async () => {
    mountDialog({ visible: false })
    await wrapper.setProps({ visible: true })
    await typeName('Test Project')
    // close then reopen — the field must be cleared, re-disabling confirm
    await wrapper.setProps({ visible: false })
    await wrapper.setProps({ visible: true })
    const input = wrapper.find('[data-testid="delete-confirm-input"]')
    expect((input.element as HTMLInputElement).value).toBe('')
    expect(confirmBtn().attributes('disabled')).toBeDefined()
  })
})
