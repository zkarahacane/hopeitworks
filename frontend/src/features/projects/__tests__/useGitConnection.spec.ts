import { describe, it, expect, vi, beforeEach } from 'vitest'

const getMock = vi.fn()
const putMock = vi.fn()
const postMock = vi.fn()
const deleteMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => getMock(...args),
    PUT: (...args: unknown[]) => putMock(...args),
    POST: (...args: unknown[]) => postMock(...args),
    DELETE: (...args: unknown[]) => deleteMock(...args),
  },
}))

import { useGitConnection, statusSeverity } from '../useGitConnection'

beforeEach(() => {
  getMock.mockReset()
  putMock.mockReset()
  postMock.mockReset()
  deleteMock.mockReset()
})

describe('statusSeverity', () => {
  it('maps each status to the documented PrimeVue severity', () => {
    expect(statusSeverity('connected')).toBe('success')
    expect(statusSeverity('expired')).toBe('warn')
    expect(statusSeverity('insufficient_scope')).toBe('warn')
    expect(statusSeverity('invalid')).toBe('danger')
    expect(statusSeverity('unconfigured')).toBe('secondary')
  })
})

describe('useGitConnection', () => {
  describe('status', () => {
    it('GETs the advisory status and returns the payload', async () => {
      getMock.mockResolvedValue({
        data: { configured: true, kind: 'pat', provider: 'github', status: 'connected' },
        error: undefined,
        response: { status: 200 },
      })
      const { status } = useGitConnection()

      const result = await status.execute('proj-1')

      expect(getMock).toHaveBeenCalledWith('/projects/{id}/git-connection', {
        params: { path: { id: 'proj-1' } },
      })
      expect(result?.status).toBe('connected')
      expect(status.error.value).toBeNull()
    })

    it('captures a friendly 403 message', async () => {
      getMock.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'FORBIDDEN', message: 'nope' } },
        response: { status: 403 },
      })
      const { status } = useGitConnection()

      const result = await status.execute('proj-1')

      expect(result).toBeNull()
      expect(status.error.value?.message).toContain('project owner or a global admin')
    })
  })

  describe('save', () => {
    it('PUTs a pat connection with provider + validate defaults', async () => {
      putMock.mockResolvedValue({
        data: { configured: true, kind: 'pat', provider: 'github', status: 'connected' },
        error: undefined,
        response: { status: 200 },
      })
      const { save } = useGitConnection()

      const result = await save.execute('proj-7', { token: 'ghp_secret' })

      expect(putMock).toHaveBeenCalledWith('/projects/{id}/git-connection', {
        params: { path: { id: 'proj-7' } },
        body: { kind: 'pat', provider: 'github', token: 'ghp_secret', validate: true },
      })
      expect(result?.status).toBe('connected')
    })

    it('forwards an explicit provider + validate flag', async () => {
      putMock.mockResolvedValue({
        data: { configured: true, kind: 'pat', provider: 'gitea', status: 'connected' },
        error: undefined,
        response: { status: 200 },
      })
      const { save } = useGitConnection()

      await save.execute('proj-7', { token: 't', provider: 'gitea', validate: false })

      expect(putMock).toHaveBeenCalledWith('/projects/{id}/git-connection', {
        params: { path: { id: 'proj-7' } },
        body: { kind: 'pat', provider: 'gitea', token: 't', validate: false },
      })
    })

    it('maps a 422 insufficient-scope code to scope guidance', async () => {
      putMock.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'GIT_CONNECTION_INSUFFICIENT_SCOPE', message: 'raw' } },
        response: { status: 422 },
      })
      const { save } = useGitConnection()

      const result = await save.execute('proj-7', { token: 't' })

      expect(result).toBeNull()
      expect(save.error.value?.message).toContain('read:project')
    })

    it('maps a 422 invalid code to a rejection message', async () => {
      putMock.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'GIT_CONNECTION_INVALID', message: 'raw' } },
        response: { status: 422 },
      })
      const { save } = useGitConnection()

      await save.execute('proj-7', { token: 't' })

      expect(save.error.value?.message).toContain('rejected this token')
    })
  })

  describe('test', () => {
    it('POSTs an empty body to test the stored token', async () => {
      postMock.mockResolvedValue({
        data: { ok: true, status: 'connected', token_type: 'fine_grained' },
        error: undefined,
        response: { status: 200 },
      })
      const { test } = useGitConnection()

      const result = await test.execute('proj-3')

      expect(postMock).toHaveBeenCalledWith('/projects/{id}/git-connection/test', {
        params: { path: { id: 'proj-3' } },
        body: {},
      })
      expect(result?.ok).toBe(true)
    })

    it('POSTs the unsaved token when provided', async () => {
      postMock.mockResolvedValue({
        data: { ok: true, status: 'connected', token_type: 'classic' },
        error: undefined,
        response: { status: 200 },
      })
      const { test } = useGitConnection()

      await test.execute('proj-3', 'ghp_unsaved')

      expect(postMock).toHaveBeenCalledWith('/projects/{id}/git-connection/test', {
        params: { path: { id: 'proj-3' } },
        body: { token: 'ghp_unsaved' },
      })
    })
  })

  describe('clear', () => {
    it('DELETEs the connection (idempotent, no body)', async () => {
      deleteMock.mockResolvedValue({ data: undefined, error: undefined, response: { status: 204 } })
      const { clear } = useGitConnection()

      await clear.execute('proj-9')

      expect(deleteMock).toHaveBeenCalledWith('/projects/{id}/git-connection', {
        params: { path: { id: 'proj-9' } },
      })
      expect(clear.error.value).toBeNull()
    })
  })
})
