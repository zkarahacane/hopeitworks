import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import LogStreamPanel from '../LogStreamPanel.vue'
import type { LogLine } from '../LogViewer.vue'
import type { SSEStatus } from '@/composables/useSSE'

const LINES: LogLine[] = [
  { text: 'building…', timestamp: new Date('2026-06-17T10:00:00Z') },
]

function mountPanel(props: { lines: LogLine[]; status: SSEStatus; active?: boolean }) {
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
