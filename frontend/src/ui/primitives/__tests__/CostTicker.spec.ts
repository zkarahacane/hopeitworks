import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import CostTicker from '../CostTicker.vue'

// Deterministic rAF/clock so the count-up tween is controllable.
let rafQueue: FrameRequestCallback[] = []
let clockNow = 0
function flushFrame() {
  const cbs = rafQueue
  rafQueue = []
  for (const cb of cbs) cb(clockNow)
}

beforeEach(() => {
  rafQueue = []
  clockNow = 0
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => {
    rafQueue.push(cb)
    return rafQueue.length
  })
  vi.stubGlobal('cancelAnimationFrame', vi.fn())
  vi.stubGlobal('performance', { now: () => clockNow })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

type TickerProps = InstanceType<typeof CostTicker>['$props']

function mountTicker(props: TickerProps) {
  return mount(CostTicker, { props, global: { plugins: [PrimeVue] } })
}

describe('CostTicker', () => {
  it('renders the value as USD in mono voice', () => {
    const w = mountTicker({ value: 1.2345, animated: false })
    expect(w.find('[data-testid="cost-ticker"]').classes()).toContain('font-mono')
    expect(w.find('[data-testid="cost-ticker-value"]').text()).toContain('$1.23')
  })

  it('shows small AI costs with extra precision', () => {
    const w = mountTicker({ value: 0.00042, animated: false })
    expect(w.find('[data-testid="cost-ticker-value"]').text()).toContain('$0.00042')
  })

  it('counts up toward a new target when animated', async () => {
    const w = mountTicker({ value: 0, animated: true, durationMs: 1000 })
    await w.setProps({ value: 10 })

    clockNow = 0
    flushFrame()
    const startText = w.find('[data-testid="cost-ticker-value"]').text()

    clockNow = 500
    flushFrame()
    await w.vm.$nextTick()
    const midText = w.find('[data-testid="cost-ticker-value"]').text()

    clockNow = 1000
    flushFrame()
    await w.vm.$nextTick()
    const endText = w.find('[data-testid="cost-ticker-value"]').text()

    expect(endText).toContain('$10.00')
    // mid-animation value differs from final (it counted up)
    expect(midText).not.toBe(endText)
    expect(startText).not.toBe(endText)
  })

  it('shows the raw value immediately when not animated', () => {
    const w = mountTicker({ value: 5, animated: false })
    expect(w.find('[data-testid="cost-ticker-value"]').text()).toContain('$5.00')
  })
})
