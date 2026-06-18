import { describe, it, expect, afterEach, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import DagGraph from '../DagGraph.vue'

// Stub @vue-flow/core — VueFlow has complex internals not suitable for unit tests.
vi.mock('@vue-flow/core', () => ({
  VueFlow: defineComponent({
    name: 'VueFlowStub',
    props: ['nodes', 'edges', 'nodeTypes', 'edgeTypes', 'fitViewOnInit'],
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    setup(props: any, { slots }) {
      // Render each node through the #node-story slot so the real DagStoryNode
      // gets the full DagNodeData (as VueFlow does via v-bind="storyProps").
      return () =>
        h('div', { class: 'vue-flow-stub' }, [
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ...(props.nodes ?? []).map((n: any) =>
            slots['node-story']?.({ id: n.id, data: n.data, selected: false }),
          ),
          slots.default?.(),
        ])
    },
  }),
  Panel: defineComponent({
    name: 'PanelStub',
    props: ['position'],
    setup(_props, { slots }) {
      return () => h('div', { class: 'panel-stub' }, slots.default?.())
    },
  }),
  useVueFlow: () => ({
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    fitView: vi.fn(),
  }),
  Handle: defineComponent({
    name: 'HandleStub',
    props: ['type', 'position'],
    setup() {
      return () => h('div', { class: 'handle-stub' })
    },
  }),
  BaseEdge: defineComponent({ name: 'BaseEdgeStub', setup: () => () => h('path') }),
  getBezierPath: () => ['M0,0', 0, 0],
  Position: { Top: 'top', Bottom: 'bottom' },
}))

vi.mock('@vue-flow/minimap', () => ({
  MiniMap: defineComponent({
    name: 'MiniMapStub',
    props: ['pannable', 'zoomable', 'position'],
    setup() {
      return () => h('div', { class: 'minimap-stub' })
    },
  }),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let wrapper: VueWrapper<any>

function makeNode(key: string, status = 'running') {
  return {
    id: key,
    type: 'story',
    position: { x: 0, y: 0 },
    data: {
      key,
      title: 'Test',
      status,
      restStatus: status,
      layer: 0,
      runId: null,
      active: status === 'running',
      containerId: 'a3f9',
      elapsedSeconds: 12,
      costUsd: 0.1,
      exitMessage: status === 'failed' ? 'exit 1' : null,
      waitingOn: [],
    },
  }
}

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
      plugins: [PrimeVue, createPinia()],
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
    expect(
      wrapper.find('[data-pc-name="skeleton"]').exists() || wrapper.html().includes('skeleton'),
    ).toBe(true)
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
      nodes: [makeNode('S-01')],
      edges: [{ id: 'S-01-S-02', source: 'S-01', target: 'S-02', data: { active: true } }],
    })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(true)
  })

  it('renders zoom controls + minimap', () => {
    mountComponent({ nodes: [makeNode('S-01')], edges: [] })
    expect(wrapper.find('[data-testid="dag-zoom-in"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dag-zoom-out"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dag-fit-view"]').exists()).toBe(true)
    expect(wrapper.find('.minimap-stub').exists()).toBe(true)
  })

  it('emits toggle-theme when the theme button is clicked', async () => {
    mountComponent({ nodes: [makeNode('S-01')], edges: [] })
    await wrapper.find('[data-testid="dag-theme-toggle"]').trigger('click')
    expect(wrapper.emitted('toggle-theme')).toHaveLength(1)
  })

  it('applies the dark class to the canvas by default', () => {
    mountComponent({ nodes: [makeNode('S-01')], edges: [] })
    expect(wrapper.find('[data-testid="dag-graph"]').classes()).toContain('dark')
  })

  it('does not render VueFlow during loading or error', () => {
    mountComponent({ isLoading: true })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
    wrapper.unmount()
    mountComponent({ error: new Error('oops') })
    expect(wrapper.find('.vue-flow-stub').exists()).toBe(false)
  })
})
