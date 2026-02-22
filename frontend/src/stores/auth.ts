import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user'
  created_at?: string
  updated_at?: string
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
        const { data, error: apiError } = await apiClient.POST('/auth/login', {
          body: { email, password },
        })
        if (apiError || !data) {
          this.error = getApiErrorMessage(apiError, 'Invalid email or password')
          return false
        }
        this.user = { ...data, role: data.role ?? 'user' } as User
        return true
      } catch {
        this.error = 'Network error. Please try again.'
        return false
      } finally {
        this.loading = false
      }
    },

    async logout(): Promise<void> {
      await apiClient.POST('/auth/logout', {}).catch(() => {})
      this.user = null
      this.error = null
    },

    async forgotPassword(email: string): Promise<boolean> {
      this.loading = true
      this.error = null
      try {
        await apiClient.POST('/auth/forgot-password', {
          body: { email },
        })
        // Always return true — never disclose whether email is registered
        return true
      } catch {
        // Network error: still return true to avoid disclosure
        return true
      } finally {
        this.loading = false
      }
    },

    async resetPassword(token: string, password: string): Promise<boolean> {
      this.loading = true
      this.error = null
      try {
        const { error: apiError } = await apiClient.POST('/auth/reset-password', {
          body: { token, password },
        })
        if (apiError) {
          this.error = getApiErrorMessage(apiError, 'Token expired or invalid. Please request a new link.')
          return false
        }
        return true
      } catch {
        this.error = 'Network error. Please try again.'
        return false
      } finally {
        this.loading = false
      }
    },

    async checkAuth(): Promise<void> {
      this.loading = true
      try {
        const { data } = await apiClient.GET('/auth/me')
        this.user = data ? { ...data, role: data.role ?? 'user' } as User : null
      } catch {
        this.user = null
      } finally {
        this.loading = false
      }
    },

    async fetchMe(): Promise<void> {
      this.loading = true
      this.error = null
      try {
        const { data, error: apiError } = await apiClient.GET('/users/me')
        if (apiError) {
          this.error = 'Failed to load profile'
          return
        }
        this.user = { ...data, role: data.role ?? 'user' } as User
      } catch {
        this.error = 'Network error. Please try again.'
      } finally {
        this.loading = false
      }
    },

    async updateMe(payload: { name?: string; email?: string }): Promise<User> {
      const { data, error: apiError } = await apiClient.PUT('/users/me', {
        body: payload,
      })
      if (apiError || !data) {
        throw new Error('Failed to update profile')
      }
      const updated = { ...data, role: data.role ?? 'user' } as User
      this.user = updated
      return updated
    },
  },
})
