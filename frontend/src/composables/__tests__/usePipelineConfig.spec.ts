import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { ref } from 'vue'
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

  it('exposes reactive computed properties from the store', () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { config, steps, isLoading, isSaving, error, isDirty } =
      usePipelineConfig(projectId)

    expect(config.value).toBeNull()
    expect(steps.value).toEqual([])
    expect(isLoading.value).toBe(false)
    expect(isSaving.value).toBe(false)
    expect(error.value).toBeNull()
    expect(isDirty.value).toBe(false)
  })

  it('provides retry that re-fetches with the same project ID', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { retry } = usePipelineConfig(projectId)

    await retry()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
      params: { path: { projectId: 'proj-1' } },
    })
  })

  it('saveConfig calls store save with correct project ID', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    mockPut.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { saveConfig, addStep, retry } = usePipelineConfig(projectId)

    // Explicitly fetch config so it is populated
    await retry()

    addStep(makeStep({ id: 's3', name: 'test' }))
    const result = await saveConfig()

    expect(result).toBe(true)
    expect(mockPut).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
      params: { path: { projectId: 'proj-1' } },
      body: { steps: expect.any(Array) },
    })
  })

  it('addStep adds step to the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { steps, addStep, retry } = usePipelineConfig(projectId)

    await retry()

    const newStep = makeStep({ id: 's3', name: 'test' })
    addStep(newStep)

    expect(steps.value).toHaveLength(3)
    expect(steps.value[2]!.id).toBe('s3')
  })

  it('removeStep removes step from the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { steps, removeStep, retry } = usePipelineConfig(projectId)

    await retry()

    removeStep(0)

    expect(steps.value).toHaveLength(1)
    expect(steps.value[0]!.id).toBe('s2')
  })

  it('reorderSteps swaps steps in the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { steps, reorderSteps, retry } = usePipelineConfig(projectId)

    await retry()

    reorderSteps(0, 1)

    expect(steps.value[0]!.id).toBe('s2')
    expect(steps.value[1]!.id).toBe('s1')
  })

  it('updateStep updates a step in the store', async () => {
    mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

    const projectId = ref('proj-1')
    const { steps, updateStep, retry } = usePipelineConfig(projectId)

    await retry()

    const step = steps.value[0]!
    const updated: PipelineStep = { ...step, model: 'claude-haiku-4-3' }
    updateStep(0, updated)

    expect(steps.value[0]!.model).toBe('claude-haiku-4-3')
  })
})
