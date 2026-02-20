import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

export function useApprovalActions() {
  const approveAction = useAsyncAction(async (projectId: string, runId: string) => {
    const { data, error } = await apiClient.POST('/projects/{projectId}/runs/{runId}/hitl/approve', {
      params: { path: { projectId, runId } },
    })
    if (error) throw new Error((error as { error?: { message?: string } }).error?.message ?? 'Approve failed')
    return data
  })

  const rejectAction = useAsyncAction(async (projectId: string, runId: string, reason: string) => {
    const { data, error } = await apiClient.POST('/projects/{projectId}/runs/{runId}/hitl/reject', {
      params: { path: { projectId, runId } },
      body: { reason },
    })
    if (error) throw new Error((error as { error?: { message?: string } }).error?.message ?? 'Reject failed')
    return data
  })

  return { approveAction, rejectAction }
}
