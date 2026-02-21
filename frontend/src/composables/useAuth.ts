import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function useAuth() {
  const store = useAuthStore()
  return {
    user: computed(() => store.user),
    isAuthenticated: computed(() => store.isAuthenticated),
    loading: computed(() => store.loading),
    error: computed(() => store.error),
    login: store.login.bind(store),
    logout: store.logout.bind(store),
    checkAuth: store.checkAuth.bind(store),
    forgotPassword: store.forgotPassword.bind(store),
    resetPassword: store.resetPassword.bind(store),
  }
}
