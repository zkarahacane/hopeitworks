import { describe, it, expect } from 'vitest'
import { z } from 'zod'

// Define schemas inline to test independently of Vue components
const profileInfoSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be 255 characters or less'),
  email: z.string().min(1, 'Email is required').email('Invalid email format'),
})

const changePasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(8, 'Password must be at least 8 characters'),
    confirm_password: z.string().min(1, 'Please confirm your new password'),
  })
  .refine((data) => data.new_password === data.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  })

function getErrors(result: z.SafeParseReturnType<unknown, unknown>, field: string): string[] {
  if (result.success) return []
  return result.error.issues
    .filter((i) => i.path.includes(field))
    .map((i) => i.message)
}

describe('profileInfoSchema', () => {
  it('accepts valid input', () => {
    const result = profileInfoSchema.safeParse({ name: 'Jane Doe', email: 'jane@example.com' })
    expect(result.success).toBe(true)
  })

  it('rejects empty name', () => {
    const result = profileInfoSchema.safeParse({ name: '', email: 'jane@example.com' })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'name')).toContain('Name is required')
  })

  it('rejects name exceeding 255 characters', () => {
    const result = profileInfoSchema.safeParse({
      name: 'a'.repeat(256),
      email: 'jane@example.com',
    })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'name')).toContain('Name must be 255 characters or less')
  })

  it('rejects empty email', () => {
    const result = profileInfoSchema.safeParse({ name: 'Jane', email: '' })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'email')).toContain('Email is required')
  })

  it('rejects invalid email format', () => {
    const result = profileInfoSchema.safeParse({ name: 'Jane', email: 'not-an-email' })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'email')).toContain('Invalid email format')
  })
})

describe('changePasswordSchema', () => {
  it('accepts valid input', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'oldpassword',
      new_password: 'newpassword123',
      confirm_password: 'newpassword123',
    })
    expect(result.success).toBe(true)
  })

  it('rejects empty current password', () => {
    const result = changePasswordSchema.safeParse({
      current_password: '',
      new_password: 'newpassword123',
      confirm_password: 'newpassword123',
    })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'current_password')).toContain('Current password is required')
  })

  it('rejects new password shorter than 8 characters', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'oldpassword',
      new_password: 'short',
      confirm_password: 'short',
    })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'new_password')).toContain('Password must be at least 8 characters')
  })

  it('rejects mismatched confirm password', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'oldpassword',
      new_password: 'newpassword123',
      confirm_password: 'differentpassword',
    })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'confirm_password')).toContain('Passwords do not match')
  })

  it('rejects empty confirm password', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'oldpassword',
      new_password: 'newpassword123',
      confirm_password: '',
    })
    expect(result.success).toBe(false)
    expect(getErrors(result, 'confirm_password').length).toBeGreaterThan(0)
  })
})
