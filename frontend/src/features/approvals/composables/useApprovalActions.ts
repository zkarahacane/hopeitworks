import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useApprovalActions() {
  const approveAction = useAsyncAction(async (hitlRequestId: string) => {
    const { data, error } = await apiClient.POST('/hitl-requests/{hitlRequestId}/approve', {
      params: { path: { hitlRequestId } },
    })
    if (error) throw new Error((error as { error?: { message?: string } }).error?.message ?? 'Approve failed')
    return data
  })

  const rejectAction = useAsyncAction(async (hitlRequestId: string, reason: string) => {
    const { data, error } = await apiClient.POST('/hitl-requests/{hitlRequestId}/reject', {
      params: { path: { hitlRequestId } },
      body: { reason },
    })
    if (error) throw new Error((error as { error?: { message?: string } }).error?.message ?? 'Reject failed')
    return data
  })

  return { approveAction, rejectAction }
}
