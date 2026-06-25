import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import LogStreamPanel from '../LogStreamPanel.vue'
import type { LogLine } from '../LogViewer.vue'
import type { SSEStatus } from '@/composables/useSSE'

const LINES: LogLine[] = [
  { text: 'building…', timestamp: new Date('2026-06-17T10:00:00Z') },
]

type StepStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

function mountPanel(props: {
  lines: LogLine[]
  status: SSEStatus
  active?: boolean
  stepStatus?: StepStatus | null
}) {
  return mount(LogStreamPanel, { props, global: { plugins: [PrimeVue] } })
}

describe('LogStreamPanel lifecycle', () => {
  it('is idle when not active (no step selected) — U1 fix', () => {
    const w = mountPanel({ lines: [], status: 'connecting', active: false })
    expect(w.attributes('data-lifecycle')).toBe('idle')
    expect(w.find('[data-testid="log-stream-status"]').text()).toContain('No step selected')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Select a step')
  })

  it('shows connecting when active + connecting + no lines', () => {
    const w = mountPanel({ lines: [], status: 'connecting' })
    expect(w.attributes('data-lifecycle')).toBe('connecting')
    expect(w.find('[data-testid="log-stream-status"]').text()).toContain('Connecting')
  })

  it('shows "waiting for output" (NOT "no output") when open but empty — U1 fix', () => {
    const w = mountPanel({ lines: [], status: 'open' })
    expect(w.attributes('data-lifecycle')).toBe('waiting')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Waiting')
    // explicitly NOT the misleading old message
    expect(w.text()).not.toContain('No log output yet')
  })

  it('streams with a blinking caret when open + lines present', () => {
    const w = mountPanel({ lines: LINES, status: 'open' })
    expect(w.attributes('data-lifecycle')).toBe('streaming')
    expect(w.find('[data-testid="log-stream-caret"]').exists()).toBe(true)
    expect(w.find('[data-testid="log-stream-caret"]').classes()).toContain('blink-caret')
  })

  it('renders captured lines even while still "connecting"', () => {
    const w = mountPanel({ lines: LINES, status: 'connecting' })
    expect(w.attributes('data-lifecycle')).toBe('streaming')
    expect(w.findAll('.log-line')).toHaveLength(1)
  })

  it('shows closed state with whatever was captured', () => {
    const w = mountPanel({ lines: LINES, status: 'closed' })
    expect(w.attributes('data-lifecycle')).toBe('closed')
    expect(w.find('[data-testid="log-stream-caret"]').exists()).toBe(false)
    expect(w.findAll('.log-line')).toHaveLength(1)
  })

  it('shows an error state on connection error', () => {
    const w = mountPanel({ lines: [], status: 'error' })
    expect(w.attributes('data-lifecycle')).toBe('error')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Could not connect')
  })

  it('renders ANSI-converted HTML in log lines', () => {
    const lines: LogLine[] = [
      { text: '\u001b[32mOK\u001b[0m', timestamp: new Date('2026-06-17T10:00:00Z') },
    ]
    const w = mountPanel({ lines, status: 'open' })
    const line = w.find('.log-line')
    expect(line.html()).toContain('OK')
    expect(line.html()).not.toContain('\u001b')
  })

  it('emits clear when the Clear button is clicked', async () => {
    const w = mountPanel({ lines: LINES, status: 'open' })
    await w.find('button').trigger('click')
    expect(w.emitted('clear')).toHaveLength(1)
  })

  it('hides the Clear button when there are no lines', () => {
    const w = mountPanel({ lines: [], status: 'open' })
    expect(w.find('button').exists()).toBe(false)
  })
})

describe('LogStreamPanel selected-step empty states (#297)', () => {
  it('RG1: selected non-terminal step with no lines shows "No logs available" (NOT "No step selected")', () => {
    // A seed/pending step is selected (active) but not running and has no logs.
    const w = mountPanel({ lines: [], status: 'closed', active: true, stepStatus: 'pending' })
    expect(w.attributes('data-lifecycle')).toBe('empty')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('No logs available')
    expect(w.find('[data-testid="log-stream-status"]').text()).not.toContain('No step selected')
  })

  it('RG1: empty state holds even while the SSE socket is "open" (non-running selected step)', () => {
    const w = mountPanel({ lines: [], status: 'open', active: true, stepStatus: 'pending' })
    expect(w.attributes('data-lifecycle')).toBe('empty')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('No logs available')
  })

  it('RG2: running selected step with lines streams + shows the live caret', () => {
    const w = mountPanel({ lines: LINES, status: 'open', active: true, stepStatus: 'running' })
    expect(w.attributes('data-lifecycle')).toBe('streaming')
    expect(w.find('[data-testid="log-stream-caret"]').exists()).toBe(true)
  })

  it('RG2: running selected step without lines yet shows the live "waiting" state', () => {
    const w = mountPanel({ lines: [], status: 'open', active: true, stepStatus: 'running' })
    expect(w.attributes('data-lifecycle')).toBe('waiting')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Waiting')
  })

  it('RG2: running selected step while still connecting shows "Connecting…"', () => {
    const w = mountPanel({ lines: [], status: 'connecting', active: true, stepStatus: 'running' })
    expect(w.attributes('data-lifecycle')).toBe('connecting')
    expect(w.find('[data-testid="log-stream-status"]').text()).toContain('Connecting')
  })

  it('RG3: no step selected (not active) shows "No step selected"', () => {
    const w = mountPanel({ lines: [], status: 'closed', active: false, stepStatus: null })
    expect(w.attributes('data-lifecycle')).toBe('idle')
    expect(w.find('[data-testid="log-stream-status"]').text()).toContain('No step selected')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Select a step')
  })

  it('RG4: SSE error on a selected step shows "Connection error" (NOT "No step selected")', () => {
    const w = mountPanel({ lines: [], status: 'error', active: true, stepStatus: 'running' })
    expect(w.attributes('data-lifecycle')).toBe('error')
    expect(w.find('[data-testid="log-stream-status"]').text()).toContain('Connection error')
    expect(w.find('[data-testid="log-stream-status"]').text()).not.toContain('No step selected')
  })

  it('RG4: SSE error wins even for a terminal selected step with no lines', () => {
    const w = mountPanel({ lines: [], status: 'error', active: true, stepStatus: 'failed' })
    expect(w.attributes('data-lifecycle')).toBe('error')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('Could not connect')
  })

  it('Q1: terminal step (completed) with no captured output shows "No output was captured"', () => {
    const w = mountPanel({ lines: [], status: 'closed', active: true, stepStatus: 'completed' })
    expect(w.attributes('data-lifecycle')).toBe('closed')
    expect(w.find('[data-testid="log-stream-empty"]').text()).toContain('No output was captured')
  })

  it('Q1: terminal step (failed) with captured lines recaps them (closed, no caret)', () => {
    const w = mountPanel({ lines: LINES, status: 'closed', active: true, stepStatus: 'failed' })
    expect(w.attributes('data-lifecycle')).toBe('closed')
    expect(w.find('[data-testid="log-stream-caret"]').exists()).toBe(false)
    expect(w.findAll('.log-line')).toHaveLength(1)
  })
})
