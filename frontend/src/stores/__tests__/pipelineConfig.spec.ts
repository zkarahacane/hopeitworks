import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePipelineConfigStore, type PipelineStep } from '../pipelineConfig'

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
    action_type: 'agent_run',
    model: 'claude-opus-4-6',
    auto_approve: false,
    retry_policy: { max_retries: 2, retry_type: 'on-failure' },
    ...overrides,
  }
}

const mockConfig = {
  project_id: 'proj-1',
  groups: [
    {
      id: 'dev',
      name: 'Development',
      steps: [
        makeStep({ id: 's1', name: 'implement', action_type: 'agent_run' }),
        makeStep({ id: 's2', name: 'review', action_type: 'agent_run' }),
        makeStep({ id: 's3', name: 'merge', action_type: 'git_pr' }),
      ],
    },
  ],
  updated_at: '2026-02-15T10:30:00Z',
}

describe('usePipelineConfigStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPut.mockReset()
  })

  it('starts with default state', () => {
    const store = usePipelineConfigStore()
    expect(store.config).toBeNull()
    expect(store.steps).toEqual([])
    expect(store.groups).toEqual([])
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.isDirty).toBe(false)
    expect(store.isSaving).toBe(false)
  })

  describe('fetchConfig', () => {
    it('fetches config successfully', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      expect(store.config).toEqual(mockConfig)
      expect(store.steps).toHaveLength(3)
      expect(store.groups).toHaveLength(1)
      expect(store.isLoading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.isDirty).toBe(false)
      expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
        params: { path: { projectId: 'proj-1' } },
      })
    })

    it('sets error when API returns an error', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'NOT_FOUND', message: 'Not found' } },
      })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      expect(store.config).toBeNull()
      expect(store.error).toBe('Failed to load pipeline configuration')
      expect(store.isLoading).toBe(false)
    })

    it('sets error when API call throws', async () => {
      mockGet.mockRejectedValue(new Error('Network error'))

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      expect(store.config).toBeNull()
      expect(store.error).toBe('Network error')
      expect(store.isLoading).toBe(false)
    })

    it('resets isDirty after successful fetch', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addStep(makeStep())
      expect(store.isDirty).toBe(true)

      await store.fetchConfig('proj-1')
      expect(store.isDirty).toBe(false)
    })
  })

  describe('local mutations', () => {
    beforeEach(async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    })

    it('addStep adds a step and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      const newStep = makeStep({ id: 's4', name: 'ci-check', action_type: 'ci_poll' })
      store.addStep(newStep)

      expect(store.steps).toHaveLength(4)
      expect(store.steps[3]).toEqual(newStep)
      expect(store.isDirty).toBe(true)
    })

    it('removeStep removes a step by index and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.removeStep(1)

      expect(store.steps).toHaveLength(2)
      expect(store.steps[0]!.id).toBe('s1')
      expect(store.steps[1]!.id).toBe('s3')
      expect(store.isDirty).toBe(true)
    })

    it('reorderSteps swaps steps and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.reorderSteps(0, 2)

      expect(store.steps[0]!.id).toBe('s2')
      expect(store.steps[1]!.id).toBe('s3')
      expect(store.steps[2]!.id).toBe('s1')
      expect(store.isDirty).toBe(true)
    })

    it('updateStep updates a step at index and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      const step = store.steps[0]!
      const updated: PipelineStep = { ...step, model: 'claude-haiku-4-5' }
      store.updateStep(0, updated)

      expect(store.steps[0]!.model).toBe('claude-haiku-4-5')
      expect(store.isDirty).toBe(true)
    })

    it('updateGroups replaces all groups and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      const newGroups = [{ id: 'new', name: 'New Group', steps: [makeStep({ id: 'new1', name: 'only-step' })] }]
      store.updateGroups(newGroups)

      expect(store.groups).toHaveLength(1)
      expect(store.steps).toHaveLength(1)
      expect(store.steps[0]!.id).toBe('new1')
      expect(store.isDirty).toBe(true)
    })

    it('mutations do nothing when config is null', () => {
      const store = usePipelineConfigStore()
      store.addStep(makeStep())
      store.removeStep(0)
      store.reorderSteps(0, 1)
      store.updateStep(0, makeStep())
      store.updateGroups([])
      expect(store.config).toBeNull()
      expect(store.isDirty).toBe(false)
    })
  })

  describe('saveConfig', () => {
    it('saves config successfully and resets isDirty', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addStep(makeStep({ id: 's4', name: 'ci-check' }))
      expect(store.isDirty).toBe(true)

      const updatedConfig = { ...mockConfig, groups: store.groups }
      mockPut.mockResolvedValue({ data: updatedConfig, error: undefined })

      const result = await store.saveConfig('proj-1')

      expect(result).toBe(true)
      expect(store.isDirty).toBe(false)
      expect(store.isSaving).toBe(false)
      expect(store.error).toBeNull()
      expect(mockPut).toHaveBeenCalledWith('/projects/{projectId}/pipeline', {
        params: { path: { projectId: 'proj-1' } },
        body: { groups: expect.any(Array) },
      })
    })

    it('returns false and sets error when API returns error', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addStep(makeStep())

      mockPut.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'BAD_REQUEST', message: 'Invalid config' } },
      })

      const result = await store.saveConfig('proj-1')

      expect(result).toBe(false)
      expect(store.error).toBe('Invalid config')
      expect(store.isSaving).toBe(false)
    })

    it('returns false and sets error when API call throws', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      mockPut.mockRejectedValue(new Error('Network error'))

      const result = await store.saveConfig('proj-1')

      expect(result).toBe(false)
      expect(store.error).toBe('Network error')
      expect(store.isSaving).toBe(false)
    })

    it('returns false when config is null', async () => {
      const store = usePipelineConfigStore()
      const result = await store.saveConfig('proj-1')
      expect(result).toBe(false)
    })
  })

  describe('reset', () => {
    it('resets all state to initial values', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addStep(makeStep())

      store.reset()

      expect(store.config).toBeNull()
      expect(store.steps).toEqual([])
      expect(store.groups).toEqual([])
      expect(store.isLoading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.isDirty).toBe(false)
      expect(store.isSaving).toBe(false)
    })
  })
})
