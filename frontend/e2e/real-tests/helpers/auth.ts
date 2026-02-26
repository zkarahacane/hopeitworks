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
    password: 'admin1234',
    name: 'Admin User',
    role: 'admin' as const,
  },
  dev: {
    email: 'dev@hopeitworks.dev',
    password: 'user1234',
    name: 'Dev User',
    role: 'user' as const,
  },
  alice: {
    email: 'alice@hopeitworks.dev',
    password: 'user1234',
    name: 'Alice Developer',
    role: 'user' as const,
  },
} satisfies Record<string, SeedUser>

export type SeedUserKey = keyof typeof SEED_USERS

/**
 * Login via the UI form. Fills email/password and submits.
 * Waits for navigation away from /login.
 *
 * The password field uses PrimeVue Password which wraps the native input,
 * so getByLabel may not resolve correctly. We fall back to a CSS selector.
 */
export async function loginViaUI(page: Page, userKey: SeedUserKey): Promise<void> {
  const user = SEED_USERS[userKey]
  await page.goto('/login')
  await page.getByLabel(/email/i).fill(user.email)

  // PrimeVue Password component wraps the <input> — getByLabel may not find it.
  // Try label association first, fall back to the raw input[type="password"] selector.
  const passwordByLabel = page.getByLabel(/password/i)
  const passwordBySelector = page.locator('input[type="password"]')
  const passwordField = (await passwordByLabel.count()) > 0
    ? passwordByLabel
    : passwordBySelector
  await passwordField.fill(user.password)

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
