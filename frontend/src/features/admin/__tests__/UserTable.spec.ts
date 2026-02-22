import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import PrimeVue from 'primevue/config'
import UserTable from '../UserTable.vue'

const mockUsers = [
  {
    id: '1',
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin' as const,
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
  },
  {
    id: '2',
    email: 'user@example.com',
    name: 'Regular User',
    role: 'user' as const,
    created_at: '2026-02-01T08:00:00Z',
    updated_at: '2026-02-01T08:00:00Z',
  },
]

const mockPagination = { total: 2, page: 1, per_page: 20 }

function mountTable(props = {}) {
  setActivePinia(createPinia())
  return mount(UserTable, {
    props: {
      users: mockUsers,
      loading: false,
      pagination: mockPagination,
      ...props,
    },
    global: {
      plugins: [PrimeVue],
    },
  })
}

describe('UserTable', () => {
  it('renders email, name, role, and created columns', () => {
    const wrapper = mountTable()
    const headers = wrapper.findAll('th')
    const headerTexts = headers.map((h) => h.text())
    expect(headerTexts).toContain('Email')
    expect(headerTexts).toContain('Name')
    expect(headerTexts).toContain('Role')
    expect(headerTexts).toContain('Created')
    expect(headerTexts).toContain('Actions')
  })

  it('renders user data in table rows', () => {
    const wrapper = mountTable()
    const text = wrapper.text()
    expect(text).toContain('admin@example.com')
    expect(text).toContain('Admin User')
    expect(text).toContain('user@example.com')
    expect(text).toContain('Regular User')
  })

  it('renders role tags with correct values', () => {
    const wrapper = mountTable()
    const tags = wrapper.findAll('[data-pc-name="tag"]')
    expect(tags.length).toBeGreaterThanOrEqual(2)
  })

  it('emits edit when pencil button is clicked', async () => {
    const wrapper = mountTable()
    const editButtons = wrapper.findAll('button[aria-label="Edit user"]')
    expect(editButtons.length).toBe(2)
    await editButtons[0]!.trigger('click')
    const editEvents = wrapper.emitted('edit')
    expect(editEvents).toBeTruthy()
    expect(editEvents![0]).toEqual([mockUsers[0]])
  })

  it('emits delete when trash button is clicked', async () => {
    const wrapper = mountTable()
    const deleteButtons = wrapper.findAll('button[aria-label="Delete user"]')
    expect(deleteButtons.length).toBe(2)
    await deleteButtons[0]!.trigger('click')
    const deleteEvents = wrapper.emitted('delete')
    expect(deleteEvents).toBeTruthy()
    expect(deleteEvents![0]).toEqual([mockUsers[0]])
  })
})
