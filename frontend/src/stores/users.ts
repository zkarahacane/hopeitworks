import { ref } from 'vue'
import { defineStore } from 'pinia'
import { apiClient } from '@/api/client'
import type { User } from '@/stores/auth'

/** Pagination metadata from API list responses */
export interface Pagination {
  total: number
  page: number
  per_page: number
}

/** Parameters for fetching paginated user lists */
export interface FetchUsersParams {
  page?: number
  per_page?: number
}

/**
 * Pinia store for user management (admin).
 * Handles CRUD operations for the user list with server-side pagination.
 */
export const useUsersStore = defineStore('users', () => {
  const users = ref<User[]>([])
  const pagination = ref<Pagination>({ total: 0, page: 1, per_page: 20 })
  const isLoading = ref(false)

  /** Fetch paginated user list from the API */
  async function fetchUsers(params?: FetchUsersParams): Promise<void> {
    isLoading.value = true
    try {
      const { data, error: apiError } = await apiClient.GET('/users', {
        params: {
          query: {
            page: params?.page ?? pagination.value.page,
            per_page: params?.per_page ?? pagination.value.per_page,
          },
        },
      })
      if (apiError) {
        throw new Error('Failed to load users')
      }
      users.value = (data?.data as User[]) ?? []
      pagination.value = (data?.pagination as Pagination) ?? { total: 0, page: 1, per_page: 20 }
    } finally {
      isLoading.value = false
    }
  }

  /** Create a new user via the register endpoint */
  async function createUser(payload: {
    email: string
    password: string
    name: string
  }): Promise<void> {
    const { error: apiError } = await apiClient.POST('/auth/register', {
      body: payload,
    })
    if (apiError) {
      throw new Error('Failed to create user')
    }
    await fetchUsers()
  }

  /** Update an existing user */
  async function updateUser(
    id: string,
    payload: { name?: string; email?: string },
  ): Promise<void> {
    const { error: apiError } = await apiClient.PUT('/users/{id}', {
      params: { path: { id } },
      body: payload,
    })
    if (apiError) {
      throw new Error('Failed to update user')
    }
    await fetchUsers()
  }

  /** Delete a user by ID */
  async function deleteUser(id: string): Promise<void> {
    const { error: apiError } = await apiClient.DELETE('/users/{id}', {
      params: { path: { id } },
    })
    if (apiError) {
      throw new Error('Failed to delete user')
    }
    await fetchUsers()
  }

  return { users, pagination, isLoading, fetchUsers, createUser, updateUser, deleteUser }
})
