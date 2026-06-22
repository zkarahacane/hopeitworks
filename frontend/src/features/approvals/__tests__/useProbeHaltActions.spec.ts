import { describe, it, expect, vi, beforeEach } from 'vitest'
import { suggestedRemedy, useProbeHaltActions } from '../composables/useProbeHaltActions'

const mockPOST = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => mockPOST(...args),
  },
}))

describe('suggestedRemedy', () => {
  it('cost_batch → resume with budget hint', () => {
    const r = suggestedRemedy('cost_batch')
    expect(r.action).toBe('resume')
    expect(r.label).toBe('Resume with more budget')
    expect(r.hint).toContain('budget')
  })

  it('log_silence → resume with retry hint', () => {
    const r = suggestedRemedy('log_silence')
    expect(r.action).toBe('resume')
    expect(r.label).toBe('Retry fresh')
    expect(r.hint).toContain('silence')
  })

  it('wallclock → resume with retry hint', () => {
    const r = suggestedRemedy('wallclock')
    expect(r.action).toBe('resume')
    expect(r.label).toBe('Retry fresh')
    expect(r.hint).toContain('wall-clock')
  })

  it('unknown probe → resume with generic hint', () => {
    const r = suggestedRemedy('something_else')
    expect(r.action).toBe('resume')
    expect(r.label).toBe('Resume')
  })

  it('empty string probe → resume with generic hint', () => {
    const r = suggestedRemedy('')
    expect(r.action).toBe('resume')
  })
})

describe('useProbeHaltActions', () => {
  beforeEach(() => {
    mockPOST.mockReset()
  })

  it('resumeAction calls resolve endpoint with action=resume', async () => {
    const responseData = { id: 'hr-1', status: 'resolved' }
    mockPOST.mockResolvedValue({ data: responseData })

    const { resumeAction } = useProbeHaltActions()
    await resumeAction.execute('hr-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'resume' },
    })
    expect(resumeAction.data.value).toEqual(responseData)
    expect(resumeAction.error.value).toBeNull()
  })

  it('overrideAction calls resolve endpoint with action=override', async () => {
    mockPOST.mockResolvedValue({ data: { id: 'hr-1', status: 'resolved' } })

    const { overrideAction } = useProbeHaltActions()
    await overrideAction.execute('hr-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'override' },
    })
  })

  it('sendBackAction calls resolve endpoint with action=send_back', async () => {
    mockPOST.mockResolvedValue({ data: { id: 'hr-1', status: 'resolved' } })

    const { sendBackAction } = useProbeHaltActions()
    await sendBackAction.execute('hr-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'send_back' },
    })
  })

  it('skipAction calls resolve endpoint with action=skip', async () => {
    mockPOST.mockResolvedValue({ data: { id: 'hr-1', status: 'resolved' } })

    const { skipAction } = useProbeHaltActions()
    await skipAction.execute('hr-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'skip' },
    })
  })

  it('abortAction calls resolve endpoint with action=abort', async () => {
    mockPOST.mockResolvedValue({ data: { id: 'hr-1', status: 'resolved' } })

    const { abortAction } = useProbeHaltActions()
    await abortAction.execute('hr-1')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'abort' },
    })
  })

  it('passes optional reason when provided', async () => {
    mockPOST.mockResolvedValue({ data: { id: 'hr-1', status: 'resolved' } })

    const { resumeAction } = useProbeHaltActions()
    await resumeAction.execute('hr-1', 'trying again after config fix')

    expect(mockPOST).toHaveBeenCalledWith('/hitl-requests/{hitlRequestId}/resolve', {
      params: { path: { hitlRequestId: 'hr-1' } },
      body: { action: 'resume', reason: 'trying again after config fix' },
    })
  })

  it('propagates API error to action error ref', async () => {
    mockPOST.mockResolvedValue({
      error: { error: { message: 'Not allowed' } },
    })

    const { resumeAction } = useProbeHaltActions()
    await resumeAction.execute('hr-1')

    expect(resumeAction.error.value).toBeInstanceOf(Error)
    expect(resumeAction.error.value?.message).toBe('Not allowed')
  })

  it('uses fallback message when error has no message', async () => {
    mockPOST.mockResolvedValue({ error: {} })

    const { abortAction } = useProbeHaltActions()
    await abortAction.execute('hr-1')

    expect(abortAction.error.value?.message).toBe('abort failed')
  })
})
