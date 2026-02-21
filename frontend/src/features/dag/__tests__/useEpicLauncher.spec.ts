import { describe, it, expect, vi, beforeEach } from 'vitest'

const mockPOST = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => mockPOST(...args),
  },
}))

const { useEpicLauncher } = await import('../composables/useEpicLauncher')

describe('useEpicLauncher', () => {
  beforeEach(() => {
    mockPOST.mockReset()
  })

  it('starts with default state', () => {
    const { result, error, isLaunching } = useEpicLauncher('proj-1', 'epic-1')
    expect(result.value).toBeNull()
    expect(error.value).toBeNull()
    expect(isLaunching.value).toBe(false)
  })

  it('returns EpicRunAccepted on successful launch', async () => {
    const accepted = { epic_run_id: 'run-uuid', status: 'scheduling', stories_count: 3 }
    mockPOST.mockResolvedValue({ data: accepted })

    const { launch, result, error, isLaunching } = useEpicLauncher('proj-1', 'epic-1')
    await launch()

    expect(result.value).toEqual(accepted)
    expect(result.value!.epic_run_id).toBe('run-uuid')
    expect(error.value).toBeNull()
    expect(isLaunching.value).toBe(false)
    expect(mockPOST).toHaveBeenCalledWith('/projects/{projectId}/epics/{epicId}/runs', {
      params: { path: { projectId: 'proj-1', epicId: 'epic-1' } },
    })
  })

  it('sets error ref when API returns an error', async () => {
    mockPOST.mockResolvedValue({
      error: { message: 'Epic not found' },
    })

    const { launch, result, error } = useEpicLauncher('proj-1', 'epic-1')
    await launch()

    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Failed to launch epic run')
    expect(result.value).toBeNull()
  })

  it('sets isLaunching while request is in flight', async () => {
    let resolveRequest: (value: unknown) => void
    mockPOST.mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )

    const { launch, isLaunching } = useEpicLauncher('proj-1', 'epic-1')
    expect(isLaunching.value).toBe(false)

    const promise = launch()
    expect(isLaunching.value).toBe(true)

    resolveRequest!({ data: { epic_run_id: 'id', status: 'scheduling', stories_count: 1 } })
    await promise

    expect(isLaunching.value).toBe(false)
  })

  it('resets error on subsequent successful call', async () => {
    mockPOST.mockResolvedValueOnce({
      error: { message: 'Server error' },
    })

    const { launch, result, error } = useEpicLauncher('proj-1', 'epic-1')
    await launch()
    expect(error.value).not.toBeNull()

    mockPOST.mockResolvedValueOnce({
      data: { epic_run_id: 'run-2', status: 'scheduling', stories_count: 2 },
    })

    await launch()
    expect(error.value).toBeNull()
    expect(result.value).toEqual({ epic_run_id: 'run-2', status: 'scheduling', stories_count: 2 })
  })
})
