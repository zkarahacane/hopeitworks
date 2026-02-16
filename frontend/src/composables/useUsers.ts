import { computed } from 'vue'
import { useUsersStore } from '@/stores/users'
import { useAsyncAction } from '@/composables/useAsyncAction'

/** Composable for user management operations with async state tracking */
export function useUsers() {
  const store = useUsersStore()
  const fetch = useAsyncAction(store.fetchUsers)
  const create = useAsyncAction(store.createUser)
  const update = useAsyncAction(store.updateUser)
  const remove = useAsyncAction(store.deleteUser)

  return {
    users: computed(() => store.users),
    pagination: computed(() => store.pagination),
    isLoading: computed(() => store.isLoading),
    fetchUsers: fetch,
    createUser: create,
    updateUser: update,
    deleteUser: remove,
  }
}
