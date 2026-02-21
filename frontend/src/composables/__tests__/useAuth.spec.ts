import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuth } from '../useAuth'

describe('useAuth', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('exposes forgotPassword method', () => {
    const { forgotPassword } = useAuth()
    expect(typeof forgotPassword).toBe('function')
  })

  it('exposes resetPassword method', () => {
    const { resetPassword } = useAuth()
    expect(typeof resetPassword).toBe('function')
  })

  it('exposes all expected properties and methods', () => {
    const auth = useAuth()
    expect(auth).toHaveProperty('user')
    expect(auth).toHaveProperty('isAuthenticated')
    expect(auth).toHaveProperty('loading')
    expect(auth).toHaveProperty('error')
    expect(auth).toHaveProperty('login')
    expect(auth).toHaveProperty('logout')
    expect(auth).toHaveProperty('checkAuth')
    expect(auth).toHaveProperty('forgotPassword')
    expect(auth).toHaveProperty('resetPassword')
  })
})
