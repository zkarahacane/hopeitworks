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

  it('approve calls correct endpoint with hitlRequestId', async () => {
    const responseData = { id: 'hitl-1', status: 'approved' }
    mockPOST.mockResolvedValue({ data: responseData })

    const { approveAction } = useApprovalActions()
    await approveAction.execute('hitl-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/approve', {
      params: { path: { hitlRequestId: 'hitl-1' } },
    })
    expect(approveAction.data.value).toEqual(responseData)
    expect(approveAction.error.value).toBeNull()
  })

  it('reject calls correct endpoint with hitlRequestId and reason', async () => {
    const responseData = { id: 'hitl-1', status: 'rejected' }
    mockPOST.mockResolvedValue({ data: responseData })

    const { rejectAction } = useApprovalActions()
    await rejectAction.execute('hitl-1', 'this is my rejection reason')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/reject', {
      params: { path: { hitlRequestId: 'hitl-1' } },
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
    await approveAction.execute('hitl-1')

    expect(approveAction.error.value).toBeInstanceOf(Error)
    expect(approveAction.error.value?.message).toBe('Not found')
  })

  it('propagates API error to rejectAction error ref', async () => {
    mockPOST.mockResolvedValue({
      error: { error: { message: 'Already processed' } },
    })

    const { rejectAction } = useApprovalActions()
    await rejectAction.execute('hitl-1', 'some reason here')

    expect(rejectAction.error.value).toBeInstanceOf(Error)
    expect(rejectAction.error.value?.message).toBe('Already processed')
  })

  it('uses fallback message when error has no message', async () => {
    mockPOST.mockResolvedValue({ error: {} })

    const { approveAction } = useApprovalActions()
    await approveAction.execute('hitl-1')

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

    const approvePromise = approveAction.execute('hitl-1')
    expect(approveAction.isLoading.value).toBe(true)
    expect(rejectAction.isLoading.value).toBe(false)

    resolveApprove!({ data: { id: 'hitl-1' } })
    await approvePromise

    expect(approveAction.isLoading.value).toBe(false)

    mockPOST.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveReject = resolve
      }),
    )

    const rejectPromise = rejectAction.execute('hitl-2', 'reason text here')
    expect(rejectAction.isLoading.value).toBe(true)
    expect(approveAction.isLoading.value).toBe(false)

    resolveReject!({ data: { id: 'hitl-2' } })
    await rejectPromise

    expect(rejectAction.isLoading.value).toBe(false)
  })
})
