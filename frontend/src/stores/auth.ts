import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'member'
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
        const json = await res.json()
        this.user = { ...json, role: json.role ?? 'member' } as User
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

    async forgotPassword(email: string): Promise<boolean> {
      this.loading = true
      this.error = null
      try {
        await fetch('/api/v1/auth/forgot-password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email }),
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
        const res = await fetch('/api/v1/auth/reset-password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token, password }),
        })
        if (!res.ok) {
          const body = await res.json().catch(() => null)
          this.error =
            body?.error?.message ?? 'Token expired or invalid. Please request a new link.'
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
        const res = await fetch('/api/v1/auth/me', {
          credentials: 'include',
        })
        if (res.ok) {
          const json = await res.json()
          this.user = { ...json, role: json.role ?? 'member' } as User
        } else {
          this.user = null
        }
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
        this.user = { ...data, role: data.role ?? 'member' } as User
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
      const updated = { ...data, role: data.role ?? 'member' } as User
      this.user = updated
      return updated
    },
  },
})
