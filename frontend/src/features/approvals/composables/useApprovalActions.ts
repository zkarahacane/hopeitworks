import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

export function useApprovalActions() {
  const approveAction = useAsyncAction(async (hitlRequestId: string) => {
    const { data, error } = await apiClient.POST('/hitl-requests/{hitlRequestId}/approve', {
      params: { path: { hitlRequestId } },
    })
    if (error) throw new Error(getApiErrorMessage(error, 'Approve failed'))
    return data
  })

  const rejectAction = useAsyncAction(async (hitlRequestId: string, reason: string) => {
    const { data, error } = await apiClient.POST('/hitl-requests/{hitlRequestId}/reject', {
      params: { path: { hitlRequestId } },
      body: { reason },
    })
    if (error) throw new Error(getApiErrorMessage(error, 'Reject failed'))
    return data
  })

  return { approveAction, rejectAction }
}
