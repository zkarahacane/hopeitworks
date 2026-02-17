import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import PrimeVue from 'primevue/config'
import DagGraph from '../DagGraph.vue'

// Stub @vue-flow/core — VueFlow has complex internals not suitable for unit tests
vi.mock('@vue-flow/core', () => ({
  VueFlow: defineComponent({
    name: 'VueFlowStub',
    props: ['nodes', 'edges', 'nodeTypes', 'fitViewOnInit'],
    setup(props, { slots }) {
      return () => h('div', { class: 'vue-flow-stub' }, slots.default?.())
    },
  }),
  Handle: defineComponent({
    name: 'HandleStub',
    props: ['type', 'position'],
    setup() {
      return () => h('div', { class: 'handle-stub' })
    },
  }),
  Position: { Top: 'top', Bottom: 'bottom' },
}))

vi.mock('@vue-flow/controls', () => ({
  Controls: defineComponent({
    name: 'ControlsStub',
    setup() {
      return () => h('div', { class: 'controls-stub' })
    },
  }),
}))

vi.mock('@vue-flow/minimap', () => ({
  MiniMap: defineComponent({
    name: 'MiniMapStub',
    setup() {
      return () => h('div', { class: 'minimap-stub' })
    },
  }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mountComponent(props: Record<string, any>) {
  wrapper = mount(DagGraph as never, {
    props: {
      nodes: [],
      edges: [],
      isLoading: false,
      error: null,
      ...props,
    } as never,
    global: {
      plugins: [PrimeVue],
    },
  })
  return wrapper
}

describe('DagGraph', () => {
  afterEach(() => {
    wrapper?.unmount()
  })

  it('renders Skeleton when isLoading is true', () => {
    mountComponent({ isLoading: true })
    expect(wrapper.find('[data-pc-name="skeleton"]').exists() || wrapper.html().includes('skeleton')).toBe(true)
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
  })

  it('renders Message when error is not null', () => {
    mountComponent({ error: new Error('fetch failed') })
    expect(wrapper.text()).toContain('fetch failed')
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
  })

  it('emits retry when retry button is clicked', async () => {
    mountComponent({ error: new Error('fetch failed') })
    const retryBtn = wrapper.findAll('button').find((b) => b.text().includes('Retry'))
    expect(retryBtn).toBeDefined()
    await retryBtn!.trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('renders VueFlow when nodes and edges are provided', () => {
    mountComponent({
      nodes: [
        { id: 'S-01', type: 'story', position: { x: 0, y: 0 }, data: { key: 'S-01', title: 'Test', status: 'backlog' } },
      ],
      edges: [{ id: 'S-01-S-02', source: 'S-01', target: 'S-02' }],
    })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(true)
  })

  it('does not render VueFlow during loading', () => {
    mountComponent({ isLoading: true })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
  })

  it('does not render VueFlow during error', () => {
    mountComponent({ error: new Error('oops') })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
  })
})
