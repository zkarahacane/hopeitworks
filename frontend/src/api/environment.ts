/** Manual types for the environment API (not yet in generated schema). */

export interface EnvironmentService {
  name: string
  image: string
  env: Record<string, string>
}

export type EnvironmentSource = 'devcontainer' | 'compose' | 'makefile' | 'declared'

export interface Environment {
  id: string
  project_id: string
  stacks: string[]
  services: EnvironmentService[]
  source: string
  commands: Record<string, string>
  created_at: string
  updated_at: string
}

export interface EnvironmentInput {
  stacks: string[]
  services: EnvironmentService[]
  source: EnvironmentSource
  commands: Record<string, string>
}

const BASE = '/api/v1'

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<{ data?: T; status: number }> {
  const res = await fetch(`${BASE}${path}`, {
    method,
    credentials: 'include',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return { status: 204 }
  if (!res.ok) {
    if (res.status === 404) return { status: 404 }
    throw new Error(`Request failed: ${res.status}`)
  }
  return { data: (await res.json()) as T, status: res.status }
}

/** Fetch the environment for a project. Returns null if not yet configured (404). */
export async function getEnvironment(projectId: string): Promise<Environment | null> {
  const result = await request<Environment>('GET', `/projects/${projectId}/environment`)
  if (result.status === 404) return null
  return result.data ?? null
}

/** Upsert the environment for a project. */
export async function putEnvironment(
  projectId: string,
  input: EnvironmentInput,
): Promise<Environment> {
  const result = await request<Environment>('PUT', `/projects/${projectId}/environment`, input)
  if (!result.data) throw new Error('No data returned from PUT environment')
  return result.data
}

/** Delete the environment for a project. */
export async function deleteEnvironment(projectId: string): Promise<void> {
  await request<void>('DELETE', `/projects/${projectId}/environment`)
}
