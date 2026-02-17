import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { ref, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { usePipelineConfig } from '../usePipelineConfig'
import type { PipelineStep } from '@/stores/pipelineConfig'

const mockGet = vi.fn()
const mockPut = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
  },
}))

function makeStep(overrides: Partial<PipelineStep> = {}): PipelineStep {
  return {
    id: crypto.randomUUID(),
    name: 'implement',
    action_type: 'implement',
    model: 'claude-opus-4-6',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
    ...overrides,
  }
}

const mockConfig = {
  project_id: 'proj-1',
  steps: [
    makeStep({ id: 's1', name: 'implement' }),
    makeStep({ id: 's2', name: 'review', action_type: 'review' }),
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

describe('usePipelineConfig', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPut.mockReset()
  })

  it('exposes reactive computed properties from the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    expect(composableResult).toBeDefined()

    // onMounted triggers fetch automatically, wait for it to complete
    await nextTick()

    expect(composableResult!.config.value).toEqual(mockConfig)
    expect(composableResult!.steps.value).toHaveLength(2)
    expect(composableResult!.isLoading.value).toBe(false)
    expect(composableResult!.isSaving.value).toBe(false)
    expect(composableResult!.error.value).toBeNull()
    expect(composableResult!.isDirty.value).toBe(false)
  })

  it('provides retry that re-fetches with the same project ID', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
      params: { path: { projectId: 'proj-1' } },
    })
  })

  it('saveConfig calls store save with correct project ID', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    mockPut.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    composableResult!.addStep(makeStep({ id: 's3', name: 'test' }))
    const result = await composableResult!.saveConfig()

    expect(result).toBe(true)
    expect(mockPut).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
      params: { path: { projectId: 'proj-1' } },
      body: { steps: expect.any(Array) },
    })
  })

  it('addStep adds step to the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    const newStep = makeStep({ id: 's3', name: 'test' })
    composableResult!.addStep(newStep)

    expect(composableResult!.steps.value).toHaveLength(3)
    expect(composableResult!.steps.value[2]!.id).toBe('s3')
  })

  it('removeStep removes step from the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    composableResult!.removeStep(0)

    expect(composableResult!.steps.value).toHaveLength(1)
    expect(composableResult!.steps.value[0]!.id).toBe('s2')
  })

  it('reorderSteps swaps steps in the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    composableResult!.reorderSteps(0, 1)

    expect(composableResult!.steps.value[0]!.id).toBe('s2')
    expect(composableResult!.steps.value[1]!.id).toBe('s1')
  })

  it('updateStep updates a step in the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    let composableResult: ReturnType<typeof usePipelineConfig> | undefined

    mount({
      setup() {
        const projectId = ref('proj-1')
        composableResult = usePipelineConfig(projectId)
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()
    await composableResult!.retry()

    const step = composableResult!.steps.value[0]!
    const updated: PipelineStep = { ...step, model: 'claude-haiku-4-3' }
    composableResult!.updateStep(0, updated)

    expect(composableResult!.steps.value[0]!.model).toBe('claude-haiku-4-3')
  })
})
