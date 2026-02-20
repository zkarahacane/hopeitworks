import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useApprovalActions } from '../composables/useApprovalActions'

const mockPOST = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => mockPOST(...args),
  },
}))

describe('useApprovalActions', () => {
  beforeEach(() => {
    mockPOST.mockReset()
  })

  it('approve calls correct endpoint with projectId and runId', async () => {
    const responseData = { run_id: 'run-1', hitl_request_id: 'hitl-1', status: 'running' }
    mockPOST.mockResolvedValue({ data: responseData })

    const { approveAction } = useApprovalActions()
    await approveAction.execute('proj-1', 'run-1')

    expect(mockPOST).toHaveBeenCalledWith('/projects/{projectId}/runs/{runId}/hitl/approve', {
      params: { path: { projectId: 'proj-1', runId: 'run-1' } },
    })
    expect(approveAction.data.value).toEqual(responseData)
    expect(approveAction.error.value).toBeNull()
  })

  it('reject calls correct endpoint with projectId, runId and reason', async () => {
    const responseData = { run_id: 'run-1', hitl_request_id: 'hitl-1', status: 'failed' }
    mockPOST.mockResolvedValue({ data: responseData })

    const { rejectAction } = useApprovalActions()
    await rejectAction.execute('proj-1', 'run-1', 'this is my rejection reason')

    expect(mockPOST).toHaveBeenCalledWith('/projects/{projectId}/runs/{runId}/hitl/reject', {
      params: { path: { projectId: 'proj-1', runId: 'run-1' } },
      body: { reason: 'this is my rejection reason' },
    })
    expect(rejectAction.data.value).toEqual(responseData)
    expect(rejectAction.error.value).toBeNull()
  })

  it('propagates API error to approveAction error ref', async () => {
    mockPOST.mockResolvedValue({
      error: { error: { message: 'Not found' } },
    })

    const { approveAction } = useApprovalActions()
    await approveAction.execute('proj-1', 'run-1')

    expect(approveAction.error.value).toBeInstanceOf(Error)
    expect(approveAction.error.value?.message).toBe('Not found')
  })

  it('propagates API error to rejectAction error ref', async () => {
    mockPOST.mockResolvedValue({
      error: { error: { message: 'Already processed' } },
    })

    const { rejectAction } = useApprovalActions()
    await rejectAction.execute('proj-1', 'run-1', 'some reason here')

    expect(rejectAction.error.value).toBeInstanceOf(Error)
    expect(rejectAction.error.value?.message).toBe('Already processed')
  })

  it('uses fallback message when error has no message', async () => {
    mockPOST.mockResolvedValue({ error: {} })

    const { approveAction } = useApprovalActions()
    await approveAction.execute('proj-1', 'run-1')

    expect(approveAction.error.value?.message).toBe('Approve failed')
  })

  it('approve and reject actions have independent loading states', async () => {
    let resolveApprove: (value: unknown) => void
    let resolveReject: (value: unknown) => void

    mockPOST.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveApprove = resolve
      }),
    )

    const { approveAction, rejectAction } = useApprovalActions()

    const approvePromise = approveAction.execute('proj-1', 'run-1')
    expect(approveAction.isLoading.value).toBe(true)
    expect(rejectAction.isLoading.value).toBe(false)

    resolveApprove!({ data: { run_id: 'run-1', hitl_request_id: 'hitl-1', status: 'running' } })
    await approvePromise

    expect(approveAction.isLoading.value).toBe(false)

    mockPOST.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveReject = resolve
      }),
    )

    const rejectPromise = rejectAction.execute('proj-1', 'run-2', 'reason text here')
    expect(rejectAction.isLoading.value).toBe(true)
    expect(approveAction.isLoading.value).toBe(false)

    resolveReject!({ data: { run_id: 'run-2', hitl_request_id: 'hitl-2', status: 'failed' } })
    await rejectPromise

    expect(rejectAction.isLoading.value).toBe(false)
  })
})
