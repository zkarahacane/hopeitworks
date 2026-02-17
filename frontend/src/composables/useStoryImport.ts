import { ref } from 'vue'
import { apiClient } from '@/api/client'
import type { components } from '@/api/schema'

/** Local preview of a parsed story block */
export interface ParsedStoryPreview {
  key: string
  title: string
  scope?: string
  valid: boolean
  error?: string
}

/** Import result returned from the backend API */
export type ImportResult = components['schemas']['ImportStoriesResult']

/**
 * Composable for importing stories from a markdown file.
 * Handles file selection, local markdown parsing preview, API import call, and state reset.
 */
export function useStoryImport() {
  const fileContent = ref<string | null>(null)
  const fileName = ref<string | null>(null)
  const parsedPreview = ref<ParsedStoryPreview[]>([])
  const importResult = ref<ImportResult | null>(null)
  const fileError = ref<string | null>(null)
  const apiError = ref<string | null>(null)
  const isImporting = ref(false)

  /**
   * Parse markdown content into a preview of story blocks.
   * Splits on `---` delimiters and extracts key/title from frontmatter + H1.
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

  /** Validate and read the selected file */
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

  /** Send the file content to the backend import endpoint */
  async function importStories(projectId: string): Promise<void> {
    if (!fileContent.value) return
    isImporting.value = true
    apiError.value = null
    try {
      const { data, error } = await apiClient.POST('/projects/{projectId}/stories/import', {
        params: { path: { projectId } },
        body: { content: fileContent.value },
      })
      if (error) {
        apiError.value = 'Import failed. Please try again.'
        return
      }
      importResult.value = data as ImportResult
    } finally {
      isImporting.value = false
    }
  }

  /** Reset all state to initial values */
  function reset() {
    fileContent.value = null
    fileName.value = null
    parsedPreview.value = []
    importResult.value = null
    fileError.value = null
    apiError.value = null
    isImporting.value = false
  }

  return {
    fileContent,
    fileName,
    parsedPreview,
    importResult,
    fileError,
    apiError,
    isImporting,
    parseMarkdownPreview,
    selectFile,
    importStories,
    reset,
  }
}
