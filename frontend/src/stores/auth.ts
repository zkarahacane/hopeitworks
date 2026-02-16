import { defineStore } from 'pinia'

export interface User {
  id: string
  email: string
  name: string
}

interface AuthState {
  user: User | null
  loading: boolean
  error: string | null
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    loading: false,
    error: null,
  }),
  getters: {
    isAuthenticated: (state) => state.user !== null,
  },
  actions: {
    async login(email: string, password: string): Promise<boolean> {
      this.loading = true
      this.error = null
      try {
        const res = await fetch('/api/v1/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ email, password }),
        })
        if (!res.ok) {
          const body = await res.json().catch(() => null)
          this.error = body?.message ?? 'Invalid email or password'
          return false
        }
        this.user = (await res.json()) as User
        return true
      } catch {
        this.error = 'Network error. Please try again.'
        return false
      } finally {
        this.loading = false
      }
    },

    async logout(): Promise<void> {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include',
      }).catch(() => {})
      this.user = null
      this.error = null
    },

    async checkAuth(): Promise<void> {
      this.loading = true
      try {
        const res = await fetch('/api/v1/auth/me', {
          credentials: 'include',
        })
        if (res.ok) {
          this.user = (await res.json()) as User
        } else {
          this.user = null
        }
      } catch {
        this.user = null
      } finally {
        this.loading = false
      }
    },
  },
})
