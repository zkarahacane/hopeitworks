import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useHITLStore } from '../hitl'

const mockGet = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
  },
}))

describe('useHITLStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
  })

  it('starts with default state', () => {
    const store = useHITLStore()
    expect(store.pendingItems).toEqual([])
    expect(store.pendingCount).toBe(0)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('pendingCount is computed from pendingItems.length', () => {
    const store = useHITLStore()
    store.handlePendingEvent({
      hitl_request_id: 'hr-1',
      run_id: 'r-1',
      step_id: 's-1',
      project_id: 'p-1',
      story_key: 'S-01',
    })
    expect(store.pendingCount).toBe(1)

    store.handlePendingEvent({
      hitl_request_id: 'hr-2',
      run_id: 'r-2',
      step_id: 's-2',
      project_id: 'p-2',
      story_key: 'S-02',
    })
    expect(store.pendingCount).toBe(2)
  })

  describe('fetchPending', () => {
    it('fetches pending items and populates state', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'hr-1',
              run_step_id: 'rs-1',
              step_id: 's-1',
              run_id: 'r-1',
              project_id: 'p-1',
              gate_type: 'approval',
              status: 'pending',
              story_key: 'S-01',
              story_title: 'Test Story',
              created_at: '2026-02-17T10:00:00Z',
            },
          ],
          pagination: { total: 1, page: 1, per_page: 20 },
        },
        error: undefined,
      })

      const store = useHITLStore()
      await store.fetchPending()

      expect(store.pendingItems).toHaveLength(1)
      expect(store.pendingItems[0]).toEqual({
        hitlRequestId: 'hr-1',
        runId: 'r-1',
        stepId: 's-1',
        projectId: 'p-1',
        projectName: '',
        storyKey: 'S-01',
        storyTitle: 'Test Story',
        prUrl: null,
        pendingSince: '2026-02-17T10:00:00Z',
      })
      expect(store.isLoading).toBe(false)
      expect(store.error).toBeNull()
      expect(mockGet).toHaveBeenCalledWith('/hitl-requests', {
        params: { query: { status: 'pending' } },
      })
    })

    it('sets error state when API returns an error', async () => {
      mockGet.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      const store = useHITLStore()
      await store.fetchPending()

      expect(store.pendingItems).toEqual([])
      expect(store.error).toBe('Failed to load pending approvals')
      expect(store.isLoading).toBe(false)
    })

    it('sets error state when API call throws', async () => {
      mockGet.mockRejectedValue(new Error('Network error'))

      const store = useHITLStore()
      await store.fetchPending()

      expect(store.error).toBe('Network error')
      expect(store.isLoading).toBe(false)
    })

    it('sets fallback error message for non-Error thrown values', async () => {
      mockGet.mockRejectedValue('unknown error')

      const store = useHITLStore()
      await store.fetchPending()

      expect(store.error).toBe('Failed to load pending approvals')
    })

    it('clears previous error on new fetch', async () => {
      mockGet
        .mockResolvedValueOnce({
          data: undefined,
          error: { error: { code: 'INTERNAL', message: 'fail' } },
        })
        .mockResolvedValueOnce({
          data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } },
          error: undefined,
        })

      const store = useHITLStore()

      await store.fetchPending()
      expect(store.error).toBe('Failed to load pending approvals')

      await store.fetchPending()
      expect(store.error).toBeNull()
    })
  })

  describe('handlePendingEvent', () => {
    it('adds a new pending item', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })

      expect(store.pendingItems).toHaveLength(1)
      expect(store.pendingItems[0]!.hitlRequestId).toBe('hr-1')
    })

    it('deduplicates by hitlRequestId', () => {
      const store = useHITLStore()
      const payload = {
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      }

      store.handlePendingEvent(payload)
      store.handlePendingEvent(payload)

      expect(store.pendingItems).toHaveLength(1)
    })

    it('uses pr_url when provided', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
        pr_url: 'https://github.com/org/repo/pull/1',
      })

      expect(store.pendingItems[0]!.prUrl).toBe('https://github.com/org/repo/pull/1')
    })

    it('sets prUrl to null when pr_url is not provided', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })

      expect(store.pendingItems[0]!.prUrl).toBeNull()
    })
  })

  describe('handleResolvedEvent', () => {
    it('removes item by hitlRequestId', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })
      store.handlePendingEvent({
        hitl_request_id: 'hr-2',
        run_id: 'r-2',
        step_id: 's-2',
        project_id: 'p-2',
        story_key: 'S-02',
      })

      expect(store.pendingItems).toHaveLength(2)

      store.handleResolvedEvent('hr-1')

      expect(store.pendingItems).toHaveLength(1)
      expect(store.pendingItems[0]!.hitlRequestId).toBe('hr-2')
    })

    it('does nothing when hitlRequestId is not found', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })

      store.handleResolvedEvent('non-existent')

      expect(store.pendingItems).toHaveLength(1)
    })

    it('decrements pendingCount when item is removed', () => {
      const store = useHITLStore()
      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })

      expect(store.pendingCount).toBe(1)

      store.handleResolvedEvent('hr-1')

      expect(store.pendingCount).toBe(0)
    })
  })

  describe('SSE events merge with fetched data without duplication', () => {
    it('does not duplicate items when SSE event arrives for already-fetched item', async () => {
      mockGet.mockResolvedValue({
        data: {
          data: [
            {
              id: 'hr-1',
              run_step_id: 'rs-1',
              step_id: 's-1',
              run_id: 'r-1',
              project_id: 'p-1',
              gate_type: 'approval',
              status: 'pending',
              story_key: 'S-01',
              story_title: 'Test Story',
              created_at: '2026-02-17T10:00:00Z',
            },
          ],
          pagination: { total: 1, page: 1, per_page: 20 },
        },
        error: undefined,
      })

      const store = useHITLStore()
      await store.fetchPending()

      store.handlePendingEvent({
        hitl_request_id: 'hr-1',
        run_id: 'r-1',
        step_id: 's-1',
        project_id: 'p-1',
        story_key: 'S-01',
      })

      expect(store.pendingItems).toHaveLength(1)
    })
  })
})
