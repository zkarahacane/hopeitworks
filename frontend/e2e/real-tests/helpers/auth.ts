import { type Page, type BrowserContext } from '@playwright/test'

export interface SeedUser {
  email: string
  password: string
  name: string
  role: 'admin' | 'user'
}

export const SEED_USERS = {
  admin: {
    email: 'admin@hopeitworks.dev',
    password: 'admin123',
    name: 'Admin User',
    role: 'admin' as const,
  },
  dev: {
    email: 'dev@hopeitworks.dev',
    password: 'dev123',
    name: 'Dev User',
    role: 'user' as const,
  },
  alice: {
    email: 'alice@hopeitworks.dev',
    password: 'alice123',
    name: 'Alice Developer',
    role: 'user' as const,
  },
} satisfies Record<string, SeedUser>

export type SeedUserKey = keyof typeof SEED_USERS

/**
 * Login via the UI form. Fills email/password and submits.
 * Waits for navigation away from /login.
 */
export async function loginViaUI(page: Page, userKey: SeedUserKey): Promise<void> {
  const user = SEED_USERS[userKey]
  await page.goto('/login')
  await page.getByLabel(/email/i).fill(user.email)
  await page.getByLabel(/password/i).fill(user.password)
  await page.getByRole('button', { name: /sign in|log in|login/i }).click()
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 10000 })
}

/**
 * Login via API. Sets the auth cookie on the browser context.
 */
export async function loginViaAPI(context: BrowserContext, userKey: SeedUserKey): Promise<void> {
  const user = SEED_USERS[userKey]
  const response = await context.request.post('/api/v1/auth/login', {
    data: { email: user.email, password: user.password },
  })
  if (!response.ok()) {
    throw new Error(`Login failed for ${user.email}: ${response.status()} ${await response.text()}`)
  }
}
