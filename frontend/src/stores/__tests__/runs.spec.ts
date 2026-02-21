import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useRunsStore } from '../runs'

describe('useRunsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('updateRunStatus', () => {
    it('updates status of a matching item in items array', () => {
      const store = useRunsStore()
      store.items = [
        { id: 'run-1', status: 'running' },
        { id: 'run-2', status: 'pending' },
      ]

      store.updateRunStatus('run-1', 'paused')

      expect(store.items[0]!.status).toBe('paused')
      expect(store.items[1]!.status).toBe('pending')
    })

    it('updates current run status when ids match', () => {
      const store = useRunsStore()
      store.current = { id: 'run-1', status: 'running', steps: [] }

      store.updateRunStatus('run-1', 'paused')

      expect(store.current.status).toBe('paused')
    })

    it('does not update current run status when ids differ', () => {
      const store = useRunsStore()
      store.current = { id: 'run-1', status: 'running', steps: [] }

      store.updateRunStatus('run-2', 'paused')

      expect(store.current.status).toBe('running')
    })

    it('handles empty items array', () => {
      const store = useRunsStore()
      store.items = []

      store.updateRunStatus('run-1', 'paused')

      expect(store.items).toHaveLength(0)
    })
  })

  describe('initial state', () => {
    it('has correct initial values', () => {
      const store = useRunsStore()

      expect(store.items).toEqual([])
      expect(store.current).toBeNull()
      expect(store.isLoading).toBe(false)
      expect(store.isPausing).toBe(false)
      expect(store.isResuming).toBe(false)
    })
  })
})
