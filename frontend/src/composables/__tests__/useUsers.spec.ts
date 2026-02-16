import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUsers } from '../useUsers'

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn().mockResolvedValue({ data: { data: [], pagination: { total: 0, page: 1, per_page: 20 } }, error: undefined }),
    POST: vi.fn().mockResolvedValue({ data: {}, error: undefined }),
    PUT: vi.fn().mockResolvedValue({ data: {}, error: undefined }),
    DELETE: vi.fn().mockResolvedValue({ data: undefined, error: undefined }),
  },
}))

describe('useUsers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('exposes reactive users, pagination, and isLoading', () => {
    const { users, pagination, isLoading } = useUsers()
    expect(users.value).toEqual([])
    expect(pagination.value).toEqual({ total: 0, page: 1, per_page: 20 })
    expect(isLoading.value).toBe(false)
  })

  it('exposes fetchUsers with execute, isLoading, and error', () => {
    const { fetchUsers } = useUsers()
    expect(fetchUsers.execute).toBeTypeOf('function')
    expect(fetchUsers.isLoading.value).toBe(false)
    expect(fetchUsers.error.value).toBeNull()
  })

  it('exposes createUser with execute, isLoading, and error', () => {
    const { createUser } = useUsers()
    expect(createUser.execute).toBeTypeOf('function')
    expect(createUser.isLoading.value).toBe(false)
    expect(createUser.error.value).toBeNull()
  })

  it('exposes updateUser with execute, isLoading, and error', () => {
    const { updateUser } = useUsers()
    expect(updateUser.execute).toBeTypeOf('function')
    expect(updateUser.isLoading.value).toBe(false)
    expect(updateUser.error.value).toBeNull()
  })

  it('exposes deleteUser with execute, isLoading, and error', () => {
    const { deleteUser } = useUsers()
    expect(deleteUser.execute).toBeTypeOf('function')
    expect(deleteUser.isLoading.value).toBe(false)
    expect(deleteUser.error.value).toBeNull()
  })

  it('fetchUsers.execute calls the store and updates loading', async () => {
    const { fetchUsers } = useUsers()
    await fetchUsers.execute()
    expect(fetchUsers.isLoading.value).toBe(false)
    expect(fetchUsers.error.value).toBeNull()
  })
})
