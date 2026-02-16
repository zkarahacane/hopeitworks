import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUsersStore } from '../users'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()
const mockDelete = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
    DELETE: (...args: unknown[]) => mockDelete(...args),
  },
}))

const mockUsers = [
  {
    id: '1',
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: '2',
    email: 'user@example.com',
    name: 'Regular User',
    role: 'user',
    created_at: '2026-01-16T10:00:00Z',
    updated_at: '2026-01-16T10:00:00Z',
  },
]

const mockPagination = { total: 2, page: 1, per_page: 20 }

describe('useUsersStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
    mockDelete.mockReset()
  })

  it('starts with default state', () => {
    const store = useUsersStore()
    expect(store.users).toEqual([])
    expect(store.pagination).toEqual({ total: 0, page: 1, per_page: 20 })
    expect(store.isLoading).toBe(false)
  })

  it('fetchUsers populates state from API response', async () => {
    mockGet.mockResolvedValue({
      data: { data: mockUsers, pagination: mockPagination },
      error: undefined,
    })

    const store = useUsersStore()
    await store.fetchUsers({ page: 1, per_page: 20 })

    expect(store.users).toEqual(mockUsers)
    expect(store.pagination).toEqual(mockPagination)
    expect(store.isLoading).toBe(false)
    expect(mockGet).toHaveBeenCalledWith('/users', {
      params: { query: { page: 1, per_page: 20 } },
    })
  })

  it('fetchUsers throws when API returns an error', async () => {
    mockGet.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'FORBIDDEN', message: 'Admin only' } },
    })

    const store = useUsersStore()
    await expect(store.fetchUsers()).rejects.toThrow('Failed to load users')
    expect(store.isLoading).toBe(false)
  })

  it('createUser calls register endpoint and refreshes', async () => {
    mockPost.mockResolvedValue({
      data: { id: '3', email: 'new@example.com', name: 'New', role: 'user' },
      error: undefined,
    })
    mockGet.mockResolvedValue({
      data: { data: mockUsers, pagination: mockPagination },
      error: undefined,
    })

    const store = useUsersStore()
    await store.createUser({ email: 'new@example.com', password: 'password123', name: 'New' })

    expect(mockPost).toHaveBeenCalledWith('/auth/register', {
      body: { email: 'new@example.com', password: 'password123', name: 'New' },
    })
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('createUser throws when API returns an error', async () => {
    mockPost.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'CONFLICT', message: 'Email taken' } },
    })

    const store = useUsersStore()
    await expect(
      store.createUser({ email: 'dup@example.com', password: 'password123', name: 'Dup' }),
    ).rejects.toThrow('Failed to create user')
  })

  it('updateUser calls PUT endpoint and refreshes', async () => {
    mockPut.mockResolvedValue({
      data: { id: '1', email: 'updated@example.com', name: 'Updated', role: 'admin' },
      error: undefined,
    })
    mockGet.mockResolvedValue({
      data: { data: mockUsers, pagination: mockPagination },
      error: undefined,
    })

    const store = useUsersStore()
    await store.updateUser('1', { name: 'Updated', email: 'updated@example.com' })

    expect(mockPut).toHaveBeenCalledWith('/users/{id}', {
      params: { path: { id: '1' } },
      body: { name: 'Updated', email: 'updated@example.com' },
    })
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('deleteUser calls DELETE endpoint and refreshes', async () => {
    mockDelete.mockResolvedValue({
      data: undefined,
      error: undefined,
    })
    mockGet.mockResolvedValue({
      data: { data: [mockUsers[1]], pagination: { total: 1, page: 1, per_page: 20 } },
      error: undefined,
    })

    const store = useUsersStore()
    await store.deleteUser('1')

    expect(mockDelete).toHaveBeenCalledWith('/users/{id}', {
      params: { path: { id: '1' } },
    })
    expect(mockGet).toHaveBeenCalledTimes(1)
  })

  it('deleteUser throws when API returns an error', async () => {
    mockDelete.mockResolvedValue({
      data: undefined,
      error: { error: { code: 'NOT_FOUND', message: 'User not found' } },
    })

    const store = useUsersStore()
    await expect(store.deleteUser('999')).rejects.toThrow('Failed to delete user')
  })
})
