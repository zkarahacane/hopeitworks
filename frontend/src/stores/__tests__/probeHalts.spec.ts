import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useProbeHaltsStore } from '../probeHalts'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('useProbeHaltsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useProbeHaltsStore()
    expect(store.items).toEqual([])
    expect(store.count).toBe(0)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.byReason).toEqual({})
  })

  describe('fetchPending', () => {
    it('fetches pending items and maps fields to camelCase', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'ph-1',
              run_step_id: 'rs-1',
              run_id: 'r-1',
              project_id: 'p-1',
              story_key: 'S-01',
              story_title: 'Test Story',
              step_name: 'code',
              stage_name: 'dev',
              halt_reason: {
                probe: 'log_silence',
                observed: 300,
                threshold: 120,
                unit: 'seconds',
              },
              created_at: '2026-06-23T10:00:00Z',
            },
          ],
          total: 1,
        },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.items).toHaveLength(1)
      expect(store.items[0]).toEqual({
        id: 'ph-1',
        runStepId: 'rs-1',
        runId: 'r-1',
        projectId: 'p-1',
        storyKey: 'S-01',
        storyTitle: 'Test Story',
        stepName: 'code',
        stageName: 'dev',
        haltReason: {
          probe: 'log_silence',
          observed: 300,
          threshold: 120,
          unit: 'seconds',
        },
        pendingSince: '2026-06-23T10:00:00Z',
      })
      expect(store.isLoading).toBe(false)
      expect(store.error).toBeNull()
      expect(mockGet).toHaveBeenCalledWith('/probe-halts', {
        params: { query: {} },
      })
    })

    it('passes project_id query param when provided', async () => {
      mockGet.mockResolvedValue({
        data: { data: [], total: 0 },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending('proj-abc')

      expect(mockGet).toHaveBeenCalledWith('/probe-halts', {
        params: { query: { project_id: 'proj-abc' } },
      })
    })

    it('sets error state when API returns an error', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.items).toEqual([])
      expect(store.error).toBe('Failed to load probe halts')
      expect(store.isLoading).toBe(false)
    })

    it('sets error state when API call throws', async () => {
      mockGet.mockRejectedValue(new Error('Network error'))

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.error).toBe('Network error')
      expect(store.isLoading).toBe(false)
    })

    it('sets fallback error message for non-Error thrown values', async () => {
      mockGet.mockRejectedValue('unknown error')

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.error).toBe('Failed to load probe halts')
    })

    it('clears previous error on new fetch', async () => {
      mockGet
        .mockResolvedValueOnce({
          data: undefined,
          error: { error: { code: 'INTERNAL', message: 'fail' } },
        })
        .mockResolvedValueOnce({
          data: { data: [], total: 0 },
          error: undefined,
        })

      const store = useProbeHaltsStore()

      await store.fetchPending()
      expect(store.error).toBe('Failed to load probe halts')

      await store.fetchPending()
      expect(store.error).toBeNull()
    })

    it('maps null halt_reason to undefined', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'ph-2',
              run_step_id: 'rs-2',
              run_id: 'r-2',
              project_id: 'p-2',
              story_key: 'S-02',
              story_title: 'Another Story',
              step_name: 'test',
              halt_reason: null,
              created_at: '2026-06-23T11:00:00Z',
            },
          ],
          total: 1,
        },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.items[0]!.haltReason).toBeUndefined()
    })
  })

  describe('byReason getter', () => {
    it('groups items by halt_reason.probe', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'ph-1',
              run_step_id: 'rs-1',
              run_id: 'r-1',
              project_id: 'p-1',
              story_key: 'S-01',
              story_title: 'Story 1',
              step_name: 'code',
              halt_reason: { probe: 'log_silence', observed: 300, threshold: 120, unit: 'seconds' },
              created_at: '2026-06-23T10:00:00Z',
            },
            {
              id: 'ph-2',
              run_step_id: 'rs-2',
              run_id: 'r-2',
              project_id: 'p-2',
              story_key: 'S-02',
              story_title: 'Story 2',
              step_name: 'test',
              halt_reason: { probe: 'wallclock', observed: 2400, threshold: 1800, unit: 'seconds' },
              created_at: '2026-06-23T10:01:00Z',
            },
            {
              id: 'ph-3',
              run_step_id: 'rs-3',
              run_id: 'r-3',
              project_id: 'p-3',
              story_key: 'S-03',
              story_title: 'Story 3',
              step_name: 'deploy',
              halt_reason: { probe: 'log_silence', observed: 400, threshold: 120, unit: 'seconds' },
              created_at: '2026-06-23T10:02:00Z',
            },
          ],
          total: 3,
        },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(Object.keys(store.byReason)).toHaveLength(2)
      expect(store.byReason['log_silence']).toHaveLength(2)
      expect(store.byReason['wallclock']).toHaveLength(1)
      expect(store.byReason['log_silence']![0]!.id).toBe('ph-1')
      expect(store.byReason['log_silence']![1]!.id).toBe('ph-3')
    })

    it('groups items with no probe under "unknown"', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'ph-1',
              run_step_id: 'rs-1',
              run_id: 'r-1',
              project_id: 'p-1',
              story_key: 'S-01',
              story_title: 'Story 1',
              step_name: 'code',
              halt_reason: null,
              created_at: '2026-06-23T10:00:00Z',
            },
          ],
          total: 1,
        },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      expect(store.byReason['unknown']).toHaveLength(1)
    })
  })

  describe('handlePendingEvent', () => {
    it('ignores events where gate_type is not probe_halt', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        gate_type: 'approval',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })
      expect(store.items).toHaveLength(0)
    })

    it('adds item when gate_type is probe_halt', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        probe: 'log_silence',
        observed: 300,
        threshold: 120,
        unit: 'seconds',
        story_key: 'S-01',
      })
      expect(store.items).toHaveLength(1)
      expect(store.items[0]!.id).toBe('hr-1')
      expect(store.items[0]!.haltReason?.probe).toBe('log_silence')
    })

    it('deduplicates by hitl_request_id', () => {
      const store = useProbeHaltsStore()
      const payload = {
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        probe: 'wallclock',
      }
      store.handlePendingEvent(payload)
      store.handlePendingEvent(payload)
      expect(store.items).toHaveLength(1)
    })

    it('does not add item when gate_type is missing', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })
      expect(store.items).toHaveLength(0)
    })

    it('increments count when item is added', () => {
      const store = useProbeHaltsStore()
      expect(store.count).toBe(0)
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })
      expect(store.count).toBe(1)
    })
  })

  describe('handleResolvedEvent', () => {
    it('removes item by id', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-2',
        run_id: 'r-2',
        step_id: 's-2',
      })

      expect(store.items).toHaveLength(2)

      store.handleResolvedEvent('hr-1')

      expect(store.items).toHaveLength(1)
      expect(store.items[0]!.id).toBe('hr-2')
    })

    it('does nothing when id is not found', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })

      store.handleResolvedEvent('non-existent')

      expect(store.items).toHaveLength(1)
    })

    it('decrements count when item is removed', () => {
      const store = useProbeHaltsStore()
      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
      })

      expect(store.count).toBe(1)
      store.handleResolvedEvent('hr-1')
      expect(store.count).toBe(0)
    })
  })

  describe('SSE dedup with fetched data', () => {
    it('does not duplicate when SSE event arrives for already-fetched item', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'ph-1',
              run_step_id: 'rs-1',
              run_id: 'r-1',
              project_id: 'p-1',
              story_key: 'S-01',
              story_title: 'Story 1',
              step_name: 'code',
              halt_reason: { probe: 'log_silence' },
              created_at: '2026-06-23T10:00:00Z',
            },
          ],
          total: 1,
        },
        error: undefined,
      })

      const store = useProbeHaltsStore()
      await store.fetchPending()

      store.handlePendingEvent({
        gate_type: 'probe_halt',
        hitl_request_id: 'ph-1',
        run_id: 'r-1',
        step_id: 'rs-1',
      })

      expect(store.items).toHaveLength(1)
    })
  })
})
