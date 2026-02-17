import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useRunLauncher, ALREADY_RUNNING_ERROR } from '../useRunLauncher'

const mockPOST = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => mockPOST(...args),
  },
}))

describe('useRunLauncher', () => {
  beforeEach(() => {
    mockPOST.mockReset()
  })

  it('starts with default state', () => {
    const { data, error, isLoading } = useRunLauncher()
    expect(data.value).toBeNull()
    expect(error.value).toBeNull()
    expect(isLoading.value).toBe(false)
  })

  it('returns data on successful launch', async () => {
    const runData = { id: 'run-1', status: 'scheduling' }
    mockPOST.mockResolvedValue({
      data: runData,
      response: { status: 202 },
    })

    const { data, error, isLoading, launchRun } = useRunLauncher()
    const result = await launchRun('proj-1', 'story-1')

    expect(result).toEqual(runData)
    expect(data.value).toEqual(runData)
    expect(error.value).toBeNull()
    expect(isLoading.value).toBe(false)
    expect(mockPOST).toHaveBeenCalledWith(
      '/projects/{id}/stories/{story_id}/runs',
      { params: { path: { id: 'proj-1', story_id: 'story-1' } } },
    )
  })

  it('throws ALREADY_RUNNING error on 409 response', async () => {
    mockPOST.mockResolvedValue({
      error: { message: 'Story already has an active run' },
      response: { status: 409 },
    })

    const { error, launchRun } = useRunLauncher()
    const result = await launchRun('proj-1', 'story-1')

    expect(result).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe(ALREADY_RUNNING_ERROR)
  })

  it('throws generic error on non-409 error response', async () => {
    mockPOST.mockResolvedValue({
      error: { message: 'Internal server error' },
      response: { status: 500 },
    })

    const { error, launchRun } = useRunLauncher()
    const result = await launchRun('proj-1', 'story-1')

    expect(result).toBeNull()
    expect(error.value).toBeInstanceOf(Error)
    expect(error.value?.message).toBe('Internal server error')
  })

  it('uses fallback message when error has no message', async () => {
    mockPOST.mockResolvedValue({
      error: {},
      response: { status: 500 },
    })

    const { error, launchRun } = useRunLauncher()
    await launchRun('proj-1', 'story-1')

    expect(error.value?.message).toBe('Failed to launch run')
  })

  it('sets isLoading while request is in flight', async () => {
    let resolveRequest: (value: unknown) => void
    mockPOST.mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )

    const { isLoading, launchRun } = useRunLauncher()
    expect(isLoading.value).toBe(false)

    const promise = launchRun('proj-1', 'story-1')
    expect(isLoading.value).toBe(true)

    resolveRequest!({ data: { id: 'run-1' }, response: { status: 202 } })
    await promise

    expect(isLoading.value).toBe(false)
  })

  it('resets error on subsequent successful call', async () => {
    mockPOST.mockResolvedValueOnce({
      error: { message: 'Server error' },
      response: { status: 500 },
    })

    const { data, error, launchRun } = useRunLauncher()
    await launchRun('proj-1', 'story-1')
    expect(error.value).not.toBeNull()

    mockPOST.mockResolvedValueOnce({
      data: { id: 'run-2' },
      response: { status: 202 },
    })

    await launchRun('proj-1', 'story-1')
    expect(error.value).toBeNull()
    expect(data.value).toEqual({ id: 'run-2' })
  })
})
