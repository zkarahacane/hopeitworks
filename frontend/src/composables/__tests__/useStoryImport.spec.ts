import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useStoryImport } from '../useStoryImport'

const mockPost = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: vi.fn(),
    POST: (...args: unknown[]) => mockPost(...args),
  },
}))

/** Captured instance from the most recent FileReader construction */
let capturedReader: {
  readAsText: ReturnType<typeof vi.fn>
  onload: ((e: ProgressEvent) => void) | null
}

beforeEach(() => {
  mockPost.mockReset()
  capturedReader = { readAsText: vi.fn(), onload: null }

  // Use a real function so `new FileReader()` works
  vi.stubGlobal(
    'FileReader',
    function MockFileReader(this: typeof capturedReader) {
      this.readAsText = vi.fn()
      this.onload = null
      // Save a reference so tests can trigger onload
      capturedReader = this
    } as unknown as typeof FileReader,
  )
})

describe('useStoryImport', () => {
  describe('selectFile', () => {
    it('sets fileError when file is not .md', () => {
      const { selectFile, fileError, fileContent } = useStoryImport()

      selectFile(new File(['content'], 'test.txt', { type: 'text/plain' }))

      expect(fileError.value).toBe('Only .md files are supported')
      expect(fileContent.value).toBeNull()
    })

    it('reads content and parses preview for .md file', () => {
      const { selectFile, fileContent, fileName, parsedPreview } = useStoryImport()

      const mdContent = '---\nkey: S-01\n---\n# My Story\n'
      selectFile(new File([mdContent], 'stories.md', { type: 'text/markdown' }))

      expect(fileName.value).toBe('stories.md')
      expect(capturedReader.readAsText).toHaveBeenCalled()

      // Simulate FileReader onload
      capturedReader.onload!({ target: { result: mdContent } } as unknown as ProgressEvent)

      expect(fileContent.value).toBe(mdContent)
      expect(parsedPreview.value).toHaveLength(1)
      expect(parsedPreview.value[0]).toEqual({
        key: 'S-01',
        title: 'My Story',
        scope: undefined,
        valid: true,
      })
    })

    it('clears previous fileError when selecting new file', () => {
      const { selectFile, fileError } = useStoryImport()

      selectFile(new File([''], 'test.txt', { type: 'text/plain' }))
      expect(fileError.value).toBe('Only .md files are supported')

      selectFile(new File([''], 'test.md', { type: 'text/markdown' }))
      expect(fileError.value).toBeNull()
    })
  })

  describe('parseMarkdownPreview', () => {
    it('parses valid story block with key, title, and scope', () => {
      const { parseMarkdownPreview } = useStoryImport()

      const content = '---\nkey: S-01\nscope: backend\n---\n# Implement login\n'
      const result = parseMarkdownPreview(content)

      expect(result).toHaveLength(1)
      expect(result[0]).toEqual({
        key: 'S-01',
        title: 'Implement login',
        scope: 'backend',
        valid: true,
      })
    })

    it('marks story invalid when key is missing', () => {
      const { parseMarkdownPreview } = useStoryImport()

      const content = '---\nscope: backend\n---\n# Some Title\n'
      const result = parseMarkdownPreview(content)

      expect(result).toHaveLength(1)
      expect(result[0]).toEqual({
        key: '(unknown)',
        title: 'Some Title',
        scope: 'backend',
        valid: false,
        error: 'Missing key in frontmatter',
      })
    })

    it('marks story invalid when H1 title is missing', () => {
      const { parseMarkdownPreview } = useStoryImport()

      const content = '---\nkey: S-01\n---\nNo heading here\n'
      const result = parseMarkdownPreview(content)

      expect(result).toHaveLength(1)
      expect(result[0]).toEqual({
        key: 'S-01',
        title: '(no title)',
        scope: undefined,
        valid: false,
        error: 'Missing H1 title in body',
      })
    })

    it('parses multiple story blocks', () => {
      const { parseMarkdownPreview } = useStoryImport()

      const content =
        '---\nkey: S-01\n---\n# Story One\n---\nkey: S-02\nscope: frontend\n---\n# Story Two\n'
      const result = parseMarkdownPreview(content)

      expect(result).toHaveLength(2)
      expect(result[0]!.key).toBe('S-01')
      expect(result[0]!.title).toBe('Story One')
      expect(result[0]!.valid).toBe(true)
      expect(result[1]!.key).toBe('S-02')
      expect(result[1]!.title).toBe('Story Two')
      expect(result[1]!.scope).toBe('frontend')
      expect(result[1]!.valid).toBe(true)
    })

    it('returns empty array for content with no story blocks', () => {
      const { parseMarkdownPreview } = useStoryImport()

      const content = 'Just some text without any frontmatter delimiters'
      const result = parseMarkdownPreview(content)

      expect(result).toHaveLength(0)
    })
  })

  describe('importStories', () => {
    it('sets importResult on success', async () => {
      const { selectFile, importStories, importResult } = useStoryImport()

      // Set up file content
      selectFile(new File(['content'], 'test.md'))
      capturedReader.onload!({
        target: { result: '---\nkey: S-01\n---\n# Test\n' },
      } as unknown as ProgressEvent)

      const mockResult = { imported: 2, updated: 1, failed: 0, errors: [] }
      mockPost.mockResolvedValue({ data: mockResult, error: undefined })

      await importStories('project-123')

      expect(mockPost).toHaveBeenCalledWith('/projects/{projectId}/stories/import', {
        params: { path: { projectId: 'project-123' } },
        body: { content: '---\nkey: S-01\n---\n# Test\n' },
      })
      expect(importResult.value).toEqual(mockResult)
    })

    it('sets apiError on API error', async () => {
      const { selectFile, importStories, importResult, apiError } = useStoryImport()

      selectFile(new File(['content'], 'test.md'))
      capturedReader.onload!({
        target: { result: 'content' },
      } as unknown as ProgressEvent)

      mockPost.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      await importStories('project-123')

      expect(apiError.value).toBe('Import failed. Please try again.')
      expect(importResult.value).toBeNull()
    })

    it('does nothing when fileContent is null', async () => {
      const { importStories } = useStoryImport()

      await importStories('project-123')

      expect(mockPost).not.toHaveBeenCalled()
    })

    it('sets isImporting during the request', async () => {
      const { selectFile, importStories, isImporting } = useStoryImport()

      selectFile(new File(['content'], 'test.md'))
      capturedReader.onload!({
        target: { result: 'content' },
      } as unknown as ProgressEvent)

      let importingDuringCall = false
      mockPost.mockImplementation(() => {
        importingDuringCall = isImporting.value
        return Promise.resolve({ data: { imported: 0, updated: 0, failed: 0, errors: [] }, error: undefined })
      })

      await importStories('project-123')

      expect(importingDuringCall).toBe(true)
      expect(isImporting.value).toBe(false)
    })
  })

  describe('reset', () => {
    it('clears all state to initial values', () => {
      const {
        fileContent,
        fileName,
        parsedPreview,
        importResult,
        fileError,
        apiError,
        isImporting,
        reset,
      } = useStoryImport()

      // Populate state
      fileContent.value = 'some content'
      fileName.value = 'test.md'
      parsedPreview.value = [{ key: 'S-01', title: 'Test', valid: true }]
      importResult.value = { imported: 1, updated: 0, failed: 0, errors: [] }
      fileError.value = 'error'
      apiError.value = 'api error'
      isImporting.value = true

      reset()

      expect(fileContent.value).toBeNull()
      expect(fileName.value).toBeNull()
      expect(parsedPreview.value).toEqual([])
      expect(importResult.value).toBeNull()
      expect(fileError.value).toBeNull()
      expect(apiError.value).toBeNull()
      expect(isImporting.value).toBe(false)
    })
  })
})
