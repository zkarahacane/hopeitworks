import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'

const postMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    POST: (...args: unknown[]) => postMock(...args),
  },
}))

import { usePlanningImport, type PlanningImportResult } from '../usePlanningImport'

function okResult(overrides: Partial<PlanningImportResult> = {}): PlanningImportResult {
  return {
    source: 'markdown',
    dry_run: true,
    epics_created: 0,
    epics_updated: 0,
    stories_created: 0,
    stories_updated: 0,
    skipped: 0,
    locked: 0,
    failed: 0,
    errors: [],
    warnings: [],
    items: [],
    ...overrides,
  }
}

beforeEach(() => {
  postMock.mockReset()
})

describe('usePlanningImport', () => {
  it('builds a markdown body for a dry-run preview', async () => {
    postMock.mockResolvedValue({ data: okResult(), error: undefined, response: { status: 200 } })
    const p = usePlanningImport()
    p.fileContent.value = '---\nkey: S-1\n---\n# Title\n'

    await p.preview('proj-1')

    expect(postMock).toHaveBeenCalledWith('/projects/{projectId}/planning/import', {
      params: { path: { projectId: 'proj-1' } },
      body: {
        source: 'markdown',
        dry_run: true,
        markdown: { content: '---\nkey: S-1\n---\n# Title\n' },
      },
    })
    expect(p.committed.value).toBe(false)
    expect(p.result.value).not.toBeNull()
  })

  it('builds a github_projects body for a commit (trims url, keeps done options)', async () => {
    postMock.mockResolvedValue({
      data: okResult({ source: 'github_projects', dry_run: false }),
      error: undefined,
      response: { status: 200 },
    })
    const p = usePlanningImport()
    p.source.value = 'github_projects'
    p.projectUrl.value = '  https://github.com/orgs/acme/projects/3  '
    p.statusField.value = 'State'
    p.doneOptions.value = ['Done', 'Shipped']
    p.epicIssueType.value = 'Epic'

    await p.commit('proj-9')

    expect(postMock).toHaveBeenCalledWith('/projects/{projectId}/planning/import', {
      params: { path: { projectId: 'proj-9' } },
      body: {
        source: 'github_projects',
        dry_run: false,
        github_projects: {
          project_url: 'https://github.com/orgs/acme/projects/3',
          status_field: 'State',
          done_options: ['Done', 'Shipped'],
          epic_issue_type: 'Epic',
        },
      },
    })
    expect(p.committed.value).toBe(true)
  })

  it('falls back to default status_field / epic_issue_type when blank', async () => {
    postMock.mockResolvedValue({ data: okResult(), error: undefined, response: { status: 200 } })
    const p = usePlanningImport()
    p.source.value = 'github_projects'
    p.projectUrl.value = 'https://github.com/users/me/projects/1'
    p.statusField.value = '   '
    p.epicIssueType.value = ''

    await p.preview('p')

    const body = (postMock.mock.calls[0]![1] as { body: { github_projects: Record<string, unknown> } }).body
    expect(body.github_projects.status_field).toBe('Status')
    expect(body.github_projects.epic_issue_type).toBe('Epic')
    expect(body.github_projects.done_options).toEqual([])
  })

  it('surfaces the PAT-scope hint on a 422 and returns null', async () => {
    postMock.mockResolvedValue({
      data: undefined,
      error: { error: { message: 'unusable' } },
      response: { status: 422 },
    })
    const p = usePlanningImport()
    p.source.value = 'github_projects'
    p.projectUrl.value = 'https://github.com/orgs/acme/projects/3'

    const r = await p.commit('p')

    expect(r).toBeNull()
    expect(p.apiError.value).toMatch(/read:project/)
    expect(p.committed.value).toBe(false)
    expect(p.result.value).toBeNull()
  })

  it('surfaces an admin-permission message on a 403', async () => {
    postMock.mockResolvedValue({
      data: undefined,
      error: { error: { message: 'forbidden' } },
      response: { status: 403 },
    })
    const p = usePlanningImport()
    p.fileContent.value = '# x'

    const r = await p.preview('p')

    expect(r).toBeNull()
    expect(p.apiError.value).toMatch(/Admin role required/)
  })

  it('canSubmit reflects the required input per source', () => {
    const p = usePlanningImport()
    expect(p.canSubmit.value).toBe(false) // markdown, no file yet
    p.fileContent.value = 'x'
    expect(p.canSubmit.value).toBe(true)

    p.source.value = 'github_projects'
    expect(p.canSubmit.value).toBe(false) // url empty
    p.projectUrl.value = 'https://github.com/orgs/acme/projects/3'
    expect(p.canSubmit.value).toBe(true)
  })

  it('clears a stale result when the source switches', async () => {
    postMock.mockResolvedValue({ data: okResult(), error: undefined, response: { status: 200 } })
    const p = usePlanningImport()
    p.fileContent.value = '# t'
    await p.preview('p')
    expect(p.result.value).not.toBeNull()

    p.source.value = 'github_projects'
    await nextTick()
    expect(p.result.value).toBeNull()
    expect(p.committed.value).toBe(false)
  })

  it('parses a local markdown preview (valid + invalid blocks)', () => {
    const p = usePlanningImport()
    const parsed = p.parseMarkdownPreview(
      '---\nkey: S-1\nscope: backend\n---\n# Story One\n---\nnokey: true\n---\n# Second\n',
    )
    expect(parsed[0]).toMatchObject({ key: 'S-1', title: 'Story One', scope: 'backend', valid: true })
    expect(parsed.some((s) => !s.valid)).toBe(true)
  })
})
