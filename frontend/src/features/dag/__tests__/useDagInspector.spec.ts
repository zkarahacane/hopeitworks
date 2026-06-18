import { describe, it, expect, beforeEach } from 'vitest'
import { computed, ref } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { useDagInspector } from '../composables/useDagInspector'
import type { DagNodeData } from '../composables/useDagLayout'
import type { SSEStatus } from '@/composables/useSSE'

function makeNode(overrides: Partial<DagNodeData> = {}): DagNodeData {
  return {
    key: 'S-02',
    title: 'Setup CI pipeline',
    status: 'running',
    restStatus: 'running',
    layer: 0,
    runId: null,
    active: true,
    containerId: 'a3f9',
    elapsedSeconds: 198,
    costUsd: 0.11,
    exitMessage: null,
    waitingOn: [],
    ...overrides,
  }
}

describe('useDagInspector', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('returns empty pipeline + logs when nothing is selected', () => {
    const selected = computed<DagNodeData | null>(() => null)
    const sse = ref<SSEStatus>('open')
    const { pipelineSteps, logLines, isActive } = useDagInspector(selected, computed(() => sse.value))
    expect(pipelineSteps.value).toHaveLength(0)
    expect(logLines.value).toHaveLength(0)
    expect(isActive.value).toBe(false)
  })

  it('derives a four-phase pipeline with the running phase active', () => {
    const selected = computed<DagNodeData | null>(() => makeNode({ status: 'running' }))
    const sse = ref<SSEStatus>('open')
    const { pipelineSteps } = useDagInspector(selected, computed(() => sse.value))
    expect(pipelineSteps.value.map((s) => s.name)).toEqual(['Setup', 'Develop', 'Review', 'Deliver'])
    expect(pipelineSteps.value[0]!.status).toBe('completed')
    expect(pipelineSteps.value[1]!.status).toBe('running')
    expect(pipelineSteps.value[2]!.status).toBe('queued')
  })

  it('marks all phases completed for a done node', () => {
    const selected = computed<DagNodeData | null>(() => makeNode({ status: 'done', active: false }))
    const sse = ref<SSEStatus>('closed')
    const { pipelineSteps } = useDagInspector(selected, computed(() => sse.value))
    expect(pipelineSteps.value.every((s) => s.status === 'completed')).toBe(true)
  })

  it('marks the develop phase failed for a failed node', () => {
    const selected = computed<DagNodeData | null>(() => makeNode({ status: 'failed', active: false }))
    const sse = ref<SSEStatus>('open')
    const { pipelineSteps } = useDagInspector(selected, computed(() => sse.value))
    expect(pipelineSteps.value[1]!.status).toBe('failed')
  })

  it('produces a non-empty live log buffer for the selected node', () => {
    const selected = computed<DagNodeData | null>(() => makeNode())
    const sse = ref<SSEStatus>('open')
    const { logLines } = useDagInspector(selected, computed(() => sse.value))
    expect(logLines.value.length).toBeGreaterThan(0)
    expect(logLines.value[0]!.text).toContain('a3f9')
    expect(logLines.value.every((l) => l.timestamp instanceof Date)).toBe(true)
  })

  it('exposes isActive + container from the selected node', () => {
    const selected = computed<DagNodeData | null>(() => makeNode())
    const sse = ref<SSEStatus>('open')
    const { isActive, containerId } = useDagInspector(selected, computed(() => sse.value))
    expect(isActive.value).toBe(true)
    expect(containerId.value).toBe('a3f9')
  })
})
