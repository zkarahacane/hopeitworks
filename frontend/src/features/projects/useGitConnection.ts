import { apiClient } from '@/api/client'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { getApiErrorMessage } from '@/utils/apiError'
import type { components } from '@/api/schema'

/** Advisory git connection status (never carries the token). From the frozen OpenAPI contract. */
export type GitConnectionStatus = components['schemas']['GitConnectionStatus']
/** Live test-connection result (200 path). */
export type GitConnectionTestResult = components['schemas']['GitConnectionTestResult']
/** The discrete status values the UI maps to a severity. */
export type GitConnectionStatusValue = GitConnectionStatus['status']
/** Supported providers for a connection. */
export type GitProvider = GitConnectionStatus['provider']
/** Origin of the active token. */
export type GitConnectionSource = NonNullable<GitConnectionStatus['source']>

/** PrimeVue Tag severities the status maps to. */
export type StatusSeverity = 'success' | 'warn' | 'danger' | 'secondary'

/**
 * Map an advisory connection status to a PrimeVue Tag severity.
 *
 * - `connected`            → success
 * - `expired`              → warn   (token aged out — recoverable by re-entering)
 * - `insufficient_scope`   → warn   (reachable but missing a scope)
 * - `invalid`              → danger (rejected by the provider — 401)
 * - `unconfigured`         → secondary (nothing stored)
 *
 * Anti-déphasage: the caller MUST always render this next to `last_validated_at`
 * ("last checked …"), never as an unqualified "connected".
 */
export function statusSeverity(status: GitConnectionStatusValue): StatusSeverity {
  switch (status) {
    case 'connected':
      return 'success'
    case 'expired':
    case 'insufficient_scope':
      return 'warn'
    case 'invalid':
      return 'danger'
    case 'unconfigured':
    default:
      return 'secondary'
  }
}

/** Payload for saving (PUT) a PAT connection. `kind` is always `pat` in v1. */
export interface SaveGitConnectionPayload {
  provider?: GitProvider
  token: string
  /** Probe the provider before persisting (default true). */
  validate?: boolean
}

/** Documented error codes the API returns on the 422/4xx paths. */
const ERROR_MESSAGES: Record<string, string> = {
  GIT_CONNECTION_INVALID:
    'The provider rejected this token (401). Check that it is valid and not revoked or expired.',
  GIT_CONNECTION_INSUFFICIENT_SCOPE:
    'The token is missing a required scope. Grant read:project (plus repo / read:org for private boards) — a fine-grained PAT is recommended.',
  GIT_CONNECTION_KEY_UNSET:
    'The server encryption key is not configured, so tokens cannot be stored. Ask an operator to set ENCRYPTION_KEY.',
}

/** Extract the `error.code` from an OpenAPI-fetch error body, if present. */
function errorCode(errBody: unknown): string | undefined {
  return (errBody as { error?: { code?: string } } | undefined)?.error?.code
}

/**
 * Turn a failed response into a friendly, scope/PAT-aware message.
 * 403 → owner/admin guidance; 422 → maps the documented error code; else the envelope message.
 */
function friendlyError(status: number | undefined, errBody: unknown, fallback: string): string {
  if (status === 403) {
    return 'You must be the project owner or a global admin to manage this connection.'
  }
  const code = errorCode(errBody)
  if (code && ERROR_MESSAGES[code]) {
    return ERROR_MESSAGES[code]!
  }
  return getApiErrorMessage(errBody, fallback)
}

/**
 * useGitConnection — wraps the four `/projects/{id}/git-connection` endpoints with the
 * `useAsyncAction` (loading / error / data) pattern over the typed `apiClient`. Types come
 * straight from the generated OpenAPI schema; nothing is hand-written.
 *
 * The token is write-only end-to-end: it is only ever sent in `save`/`test` request bodies
 * and never read back — the status payload exposes only advisory metadata.
 */
export function useGitConnection() {
  /** GET — fetch the advisory connection status for a project. */
  const status = useAsyncAction(async (projectId: string): Promise<GitConnectionStatus> => {
    const res = await apiClient.GET('/projects/{id}/git-connection', {
      params: { path: { id: projectId } },
    })
    if (res?.error || !res?.data) {
      throw new Error(friendlyError(res?.response?.status, res?.error, 'Failed to load git connection status.'))
    }
    return res.data
  })

  /** PUT — set or replace the token (validated before persist by default). Returns refreshed status. */
  const save = useAsyncAction(
    async (projectId: string, payload: SaveGitConnectionPayload): Promise<GitConnectionStatus> => {
      const res = await apiClient.PUT('/projects/{id}/git-connection', {
        params: { path: { id: projectId } },
        body: {
          kind: 'pat',
          provider: payload.provider ?? 'github',
          token: payload.token,
          validate: payload.validate ?? true,
        },
      })
      if (res?.error || !res?.data) {
        throw new Error(friendlyError(res?.response?.status, res?.error, 'Failed to save the git connection.'))
      }
      return res.data
    },
  )

  /** POST /test — live-probe the stored token, or an unsaved one supplied in the body. */
  const test = useAsyncAction(
    async (projectId: string, token?: string): Promise<GitConnectionTestResult> => {
      const res = await apiClient.POST('/projects/{id}/git-connection/test', {
        params: { path: { id: projectId } },
        body: token ? { token } : {},
      })
      if (res?.error || !res?.data) {
        throw new Error(friendlyError(res?.response?.status, res?.error, 'Connection test failed.'))
      }
      return res.data
    },
  )

  /** DELETE — clear the stored token (reverts resolution to the env fallback). Idempotent. */
  const clear = useAsyncAction(async (projectId: string): Promise<void> => {
    const res = await apiClient.DELETE('/projects/{id}/git-connection', {
      params: { path: { id: projectId } },
    })
    if (res?.error) {
      throw new Error(friendlyError(res?.response?.status, res?.error, 'Failed to disconnect.'))
    }
  })

  return { status, save, test, clear, statusSeverity }
}
