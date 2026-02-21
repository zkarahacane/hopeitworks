import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PrimeVue from 'primevue/config'
import LogViewer from '../LogViewer.vue'
import type { LogLine } from '../LogViewer.vue'

function mountLogViewer(props: { lines: LogLine[]; status: 'connecting' | 'open' | 'closed' | 'error' }) {
  return mount(LogViewer, {
    props,
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('LogViewer', () => {
  it('renders the correct number of log line elements', () => {
    const lines: LogLine[] = [
      { text: 'line one', timestamp: new Date('2026-01-01T10:00:00Z') },
      { text: 'line two', timestamp: new Date('2026-01-01T10:00:01Z') },
      { text: 'line three', timestamp: new Date('2026-01-01T10:00:02Z') },
    ]

    const wrapper = mountLogViewer({ lines, status: 'open' })
    const logLines = wrapper.findAll('.log-line')

    expect(logLines).toHaveLength(3)
  })

  it('renders ANSI-converted HTML in log lines', () => {
    const lines: LogLine[] = [
      { text: '\u001b[32mOK\u001b[0m', timestamp: new Date('2026-01-01T10:00:00Z') },
    ]

    const wrapper = mountLogViewer({ lines, status: 'open' })
    const logLine = wrapper.find('.log-line')

    expect(logLine.html()).toContain('OK')
    expect(logLine.html()).not.toContain('\u001b')
  })

  it('shows "Connecting..." tag when status is connecting', () => {
    const wrapper = mountLogViewer({ lines: [], status: 'connecting' })

    expect(wrapper.text()).toContain('Connecting...')
  })

  it('shows "Live" tag when status is open', () => {
    const wrapper = mountLogViewer({ lines: [], status: 'open' })

    expect(wrapper.text()).toContain('Live')
  })

  it('shows "Disconnected" tag when status is closed', () => {
    const wrapper = mountLogViewer({ lines: [], status: 'closed' })

    expect(wrapper.text()).toContain('Disconnected')
  })

  it('shows "Error" tag when status is error', () => {
    const wrapper = mountLogViewer({ lines: [], status: 'error' })

    expect(wrapper.text()).toContain('Error')
  })

  it('emits clear when Clear button is clicked', async () => {
    const wrapper = mountLogViewer({ lines: [], status: 'open' })
    const clearButton = wrapper.find('button')

    await clearButton.trigger('click')

    expect(wrapper.emitted('clear')).toBeTruthy()
    expect(wrapper.emitted('clear')).toHaveLength(1)
  })

  it('shows "No log output yet" when lines is empty', () => {
    const wrapper = mountLogViewer({ lines: [], status: 'open' })

    expect(wrapper.text()).toContain('No log output yet')
  })
})
