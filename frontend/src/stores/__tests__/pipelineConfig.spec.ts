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

      const newGroups = [{ id: 'new', name: 'New Group', transition: 'auto' as const, steps: [makeStep({ id: 'new1', name: 'only-step' })] }]
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
      store.addGroup()
      store.removeGroup('x')
      store.renameGroup('x', 'y')
      store.addStepToGroup('x', makeStep())
      store.removeStepFromGroup('x', 'y')
      store.updateStepInGroup('x', 'y', makeStep())
      store.reorderStepsInGroup('x', 0, 1)
      store.reorderGroups(0, 1)
      expect(store.config).toBeNull()
      expect(store.isDirty).toBe(false)
    })
  })

  describe('group CRUD', () => {
    beforeEach(async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    })

    it('addGroup adds a new empty group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGroup('Review')

      expect(store.groups).toHaveLength(2)
      expect(store.groups[1]!.name).toBe('Review')
      expect(store.groups[1]!.steps).toEqual([])
      expect(store.isDirty).toBe(true)
    })

    it('addGroup uses default name', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGroup()

      expect(store.groups).toHaveLength(2)
      expect(store.groups[1]!.name).toBe('New Group')
    })

    it('removeGroup removes a group by id', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGroup('Review')
      const reviewGroupId = store.groups[1]!.id

      store.removeGroup(reviewGroupId)

      expect(store.groups).toHaveLength(1)
      expect(store.groups[0]!.name).toBe('Development')
      expect(store.isDirty).toBe(true)
    })

    it('renameGroup updates group name', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.renameGroup('dev', 'Setup')

      expect(store.groups[0]!.name).toBe('Setup')
      expect(store.isDirty).toBe(true)
    })

    it('addStepToGroup adds a step to a specific group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGroup('Review')
      const reviewGroupId = store.groups[1]!.id
      const newStep = makeStep({ id: 's-new', name: 'lint' })
      store.addStepToGroup(reviewGroupId, newStep)

      expect(store.groups[1]!.steps).toHaveLength(1)
      expect(store.groups[1]!.steps[0]!.id).toBe('s-new')
      // Original group unchanged
      expect(store.groups[0]!.steps).toHaveLength(3)
    })

    it('removeStepFromGroup removes a step from a specific group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.removeStepFromGroup('dev', 's2')

      expect(store.groups[0]!.steps).toHaveLength(2)
      expect(store.groups[0]!.steps.find((s: PipelineStep) => s.id === 's2')).toBeUndefined()
      expect(store.isDirty).toBe(true)
    })

    it('updateStepInGroup updates a step within a group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      const step = store.groups[0]!.steps[0]!
      const updated = { ...step, model: 'claude-haiku-4-5' as const }
      store.updateStepInGroup('dev', 's1', updated)

      expect(store.groups[0]!.steps[0]!.model).toBe('claude-haiku-4-5')
      expect(store.isDirty).toBe(true)
    })

    it('reorderStepsInGroup reorders steps within a group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.reorderStepsInGroup('dev', 0, 2)

      expect(store.groups[0]!.steps[0]!.id).toBe('s2')
      expect(store.groups[0]!.steps[1]!.id).toBe('s3')
      expect(store.groups[0]!.steps[2]!.id).toBe('s1')
      expect(store.isDirty).toBe(true)
    })

    it('reorderGroups moves a group to a new position', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGroup('Review')
      store.addGroup('Deploy')

      // Move 'Deploy' (index 2) to position 0
      store.reorderGroups(2, 0)

      expect(store.groups[0]!.name).toBe('Deploy')
      expect(store.groups[1]!.name).toBe('Development')
      expect(store.groups[2]!.name).toBe('Review')
      expect(store.isDirty).toBe(true)
    })
  })

  describe('transition policy', () => {
    beforeEach(async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    })

    it('updateGroupTransition sets transition and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.updateGroupTransition('dev', 'manual')

      expect(store.groups[0]!.transition).toBe('manual')
      expect(store.isDirty).toBe(true)
    })

    it('updateGroupTransition only affects the target group', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addGroup('Review')
      const reviewId = store.groups[1]!.id

      store.updateGroupTransition(reviewId, 'gate')

      expect(store.groups[0]!.transition).not.toBe('gate')
      expect(store.groups[1]!.transition).toBe('gate')
    })

    it('updateGroupTransition does nothing when config is null', () => {
      const store = usePipelineConfigStore()
      store.updateGroupTransition('dev', 'gate')
      expect(store.config).toBeNull()
      expect(store.isDirty).toBe(false)
    })
  })

  describe('guards', () => {
    beforeEach(async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })
    })

    it('addGuard appends a default guard to a stage and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGuard('dev')

      expect(store.groups[0]!.guards).toHaveLength(1)
      expect(store.groups[0]!.guards![0]).toEqual({
        kind: 'log_silence',
        threshold: 120,
        on_fail: 'halt-gate',
      })
      expect(store.isDirty).toBe(true)
    })

    it('addGuard appends to existing stage guards', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGuard('dev')
      store.addGuard('dev')

      expect(store.groups[0]!.guards).toHaveLength(2)
    })

    it('updateGuard replaces the guard at index and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addGuard('dev')

      store.updateGuard('dev', 0, { kind: 'wallclock', max: 1800, on_fail: 'fail' })

      expect(store.groups[0]!.guards![0]).toEqual({
        kind: 'wallclock',
        max: 1800,
        on_fail: 'fail',
      })
      expect(store.isDirty).toBe(true)
    })

    it('removeGuard removes the guard at index and marks dirty', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addGuard('dev')
      store.updateGuard('dev', 0, { kind: 'cost_batch', max: 5, on_fail: 'halt-gate' })
      store.addGuard('dev')

      store.removeGuard('dev', 0)

      expect(store.groups[0]!.guards).toHaveLength(1)
      // the surviving guard is the second (default) one
      expect(store.groups[0]!.guards![0]!.kind).toBe('log_silence')
      expect(store.isDirty).toBe(true)
    })

    it('addGuard targets a step when stepId is provided', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')

      store.addGuard('dev', 's1')

      expect(store.groups[0]!.steps[0]!.guards).toHaveLength(1)
      // group-level guards untouched
      expect(store.groups[0]!.guards ?? []).toHaveLength(0)
    })

    it('updateGuard and removeGuard target a step when stepId is provided', async () => {
      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.addGuard('dev', 's1')

      store.updateGuard('dev', 0, { kind: 'wallclock', max: 600, on_fail: 'retry' }, 's1')
      expect(store.groups[0]!.steps[0]!.guards![0]!.kind).toBe('wallclock')

      store.removeGuard('dev', 0, 's1')
      expect(store.groups[0]!.steps[0]!.guards).toHaveLength(0)
    })

    it('guard mutations do nothing when config is null', () => {
      const store = usePipelineConfigStore()
      store.addGuard('dev')
      store.updateGuard('dev', 0, { kind: 'wallclock', max: 1, on_fail: 'fail' })
      store.removeGuard('dev', 0)
      expect(store.config).toBeNull()
      expect(store.isDirty).toBe(false)
    })
  })

  describe('saveConfig', () => {
    it('persists transition and guards via the PUT body', async () => {
      mockGet.mockResolvedValue({ data: mockConfig, error: undefined })

      const store = usePipelineConfigStore()
      await store.fetchConfig('proj-1')
      store.updateGroupTransition('dev', 'gate')
      store.addGuard('dev')

      mockPut.mockResolvedValue({ data: { ...mockConfig, groups: store.groups }, error: undefined })

      await store.saveConfig('proj-1')

      const sentGroups = mockPut.mock.calls[0]![1].body.groups
      expect(sentGroups[0].transition).toBe('gate')
      expect(sentGroups[0].guards).toHaveLength(1)
    })

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
