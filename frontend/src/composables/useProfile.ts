import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

/** Composable for user profile self-service operations. */
export function useProfile() {
  const store = useAuthStore()

  const fetchMe = useAsyncAction(() => store.fetchMe())
  const updateMe = useAsyncAction((payload: { name?: string; email?: string }) =>
    store.updateMe(payload),
  )
  const changePassword = useAsyncAction(
    async (payload: { current_password: string; new_password: string }) => {
      const { error: apiError } = await apiClient.PUT('/users/me/password', {
        body: payload,
      })
      if (apiError) {
        throw new Error('Failed to change password. Check your current password and try again.')
      }
    },
  )

  return {
    user: computed(() => store.user),
    fetchMe,
    updateMe,
    changePassword,
  }
}
