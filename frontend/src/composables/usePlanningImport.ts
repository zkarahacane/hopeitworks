import { computed, ref, watch } from 'vue'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'
import type { components } from '@/api/schema'

/** Planning source discriminator — mirrors the OpenAPI `PlanningImportRequest.source` enum. */
export type PlanningSource = 'markdown' | 'github_projects'

/** Request/result types straight from the frozen OpenAPI contract. */
export type PlanningImportRequest = components['schemas']['PlanningImportRequest']
export type PlanningImportResult = components['schemas']['PlanningImportResult']
export type PlanningImportItem = components['schemas']['PlanningImportItem']

/** Local preview of a parsed markdown story block (client-side, before any API call). */
export interface ParsedStoryPreview {
  key: string
  title: string
  scope?: string
  valid: boolean
  error?: string
}

/**
 * Hint surfaced on a 422 (source reachable but unusable): the most common cause is a
 * token-scope / project-URL mismatch on the GitHub Projects path.
 */
const SOURCE_ERROR_HINT =
  'Source reachable but unusable. Check the project URL and that the configured token has read:project scope (user-owned projects require a classic PAT).'

/**
 * usePlanningImport — drives the planning connector dialog (markdown + GitHub Projects).
 *
 * Owns the source selection, the per-source form state, the local markdown preview, and
 * the two API calls against `POST /projects/{projectId}/planning/import`:
 *   - `preview(projectId)` → `dry_run: true`  (compute decisions, no writes)
 *   - `commit(projectId)`  → `dry_run: false` (apply the import)
 *
 * Both return the `PlanningImportResult` (or `null` on error, with `apiError` populated).
 * Refreshing the board after a commit is the store's responsibility (`runPlanningImport`).
 */
export function usePlanningImport() {
  // ── Source selection ──────────────────────────────────────────────────────────
  const source = ref<PlanningSource>('markdown')

  // ── Markdown source state ───────────────────────────────────────────────────────
  const fileContent = ref<string | null>(null)
  const fileName = ref<string | null>(null)
  const parsedPreview = ref<ParsedStoryPreview[]>([])
  const fileError = ref<string | null>(null)

  // ── GitHub Projects source state ──────────────────────────────────────────────
  const projectUrl = ref('')
  const statusField = ref('Status')
  const doneOptions = ref<string[]>([])
  const epicIssueType = ref('Epic')

  // ── Shared call state ─────────────────────────────────────────────────────────
  /** Last result returned (dry-run preview or committed import). */
  const result = ref<PlanningImportResult | null>(null)
  /** True when `result` came from a commit (dry_run: false). */
  const committed = ref(false)
  const apiError = ref<string | null>(null)
  const isLoading = ref(false)

  /**
   * Parse markdown content into a preview of story blocks. Splits on `---` delimiters
   * and extracts key/title/scope from frontmatter + H1 (purely local, no API call).
   */
  function parseMarkdownPreview(content: string): ParsedStoryPreview[] {
    const blocks = content.split(/^---$/m).filter((b) => b.trim())
    const stories: ParsedStoryPreview[] = []

    for (let i = 0; i < blocks.length - 1; i += 2) {
      const frontmatter = blocks[i]!
      const body = blocks[i + 1] ?? ''

      const keyMatch = frontmatter.match(/^key:\s*(.+)$/m)
      const titleMatch = body.match(/^#\s+(.+)$/m)
      const scopeMatch = frontmatter.match(/^scope:\s*(.+)$/m)

      const key = keyMatch?.[1]?.trim() ?? ''
      const title = titleMatch?.[1]?.trim() ?? ''
      const scope = scopeMatch?.[1]?.trim()

      if (!key) {
        stories.push({ key: '(unknown)', title, scope, valid: false, error: 'Missing key in frontmatter' })
      } else if (!title) {
        stories.push({ key, title: '(no title)', scope, valid: false, error: 'Missing H1 title in body' })
      } else {
        stories.push({ key, title, scope, valid: true })
      }
    }
    return stories
  }

  /** Validate and read a selected markdown file into `fileContent` + a local preview. */
  function selectFile(file: File) {
    fileError.value = null
    if (!file.name.endsWith('.md')) {
      fileError.value = 'Only .md files are supported'
      return
    }
    fileName.value = file.name
    const reader = new FileReader()
    reader.onload = (e) => {
      fileContent.value = e.target?.result as string
      parsedPreview.value = parseMarkdownPreview(fileContent.value)
    }
    reader.readAsText(file)
  }

  /** Whether the current source has enough input to call the API. */
  const canSubmit = computed(() => {
    if (source.value === 'markdown') return !!fileContent.value
    return projectUrl.value.trim() !== ''
  })

  /** Build the discriminated request body for the active source. */
  function buildBody(dryRun: boolean): PlanningImportRequest {
    if (source.value === 'markdown') {
      return {
        source: 'markdown',
        dry_run: dryRun,
        markdown: { content: fileContent.value ?? '' },
      }
    }
    return {
      source: 'github_projects',
      dry_run: dryRun,
      github_projects: {
        project_url: projectUrl.value.trim(),
        status_field: statusField.value.trim() || 'Status',
        done_options: doneOptions.value,
        epic_issue_type: epicIssueType.value.trim() || 'Epic',
      },
    }
  }

  /** POST the import (dry-run or commit). Returns the result or null (with apiError set). */
  async function run(projectId: string, dryRun: boolean): Promise<PlanningImportResult | null> {
    apiError.value = null
    isLoading.value = true
    try {
      const { data, error, response } = await apiClient.POST('/projects/{projectId}/planning/import', {
        params: { path: { projectId } },
        body: buildBody(dryRun),
      })

      if (error || !data) {
        if (response?.status === 422) {
          apiError.value = SOURCE_ERROR_HINT
        } else if (response?.status === 403) {
          apiError.value = 'You do not have permission to import. Admin role required.'
        } else {
          apiError.value = getApiErrorMessage(error, 'Import failed. Please try again.')
        }
        return null
      }

      result.value = data
      committed.value = !dryRun
      return data
    } finally {
      isLoading.value = false
    }
  }

  /** Dry-run preview: compute the per-node decisions without writing anything. */
  function preview(projectId: string): Promise<PlanningImportResult | null> {
    return run(projectId, true)
  }

  /** Commit: apply the import (creates/updates/locks rows). */
  function commit(projectId: string): Promise<PlanningImportResult | null> {
    return run(projectId, false)
  }

  /** Reset every piece of state to its initial value. */
  function reset() {
    source.value = 'markdown'
    fileContent.value = null
    fileName.value = null
    parsedPreview.value = []
    fileError.value = null
    projectUrl.value = ''
    statusField.value = 'Status'
    doneOptions.value = []
    epicIssueType.value = 'Epic'
    result.value = null
    committed.value = false
    apiError.value = null
    isLoading.value = false
  }

  // Switching source invalidates any stale preview/result so the table never shows a
  // decision set computed for the other source.
  watch(source, () => {
    result.value = null
    committed.value = false
    apiError.value = null
  })

  return {
    // source
    source,
    canSubmit,
    // markdown
    fileContent,
    fileName,
    parsedPreview,
    fileError,
    selectFile,
    parseMarkdownPreview,
    // github
    projectUrl,
    statusField,
    doneOptions,
    epicIssueType,
    // shared
    result,
    committed,
    apiError,
    isLoading,
    buildBody,
    preview,
    commit,
    reset,
  }
}
