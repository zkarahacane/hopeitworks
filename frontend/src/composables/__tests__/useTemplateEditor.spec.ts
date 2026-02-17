import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { useTemplateEditor } from '../useTemplateEditor'

const mockGet = vi.fn()
const mockPost = vi.fn()
const mockPut = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: (...args: unknown[]) => mockGet(...args),
    POST: (...args: unknown[]) => mockPost(...args),
    PUT: (...args: unknown[]) => mockPut(...args),
  },
}))

const mockTemplate = {
  id: 't1',
  project_id: 'p1',
  name: 'Implement Template',
  template_content: 'You are working on {{story_key}}: {{story_title}}',
  type: 'implement' as const,
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

describe('useTemplateEditor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockGet.mockReset()
    mockPost.mockReset()
    mockPut.mockReset()
  })

  it('fetches template on mount for existing template', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(mockGet).toHaveBeenCalledWith('/projects/{projectId}/templates/{templateId}', {
      params: { path: { projectId: 'p1', templateId: 't1' } },
    })
    expect(result!.content.value).toBe(mockTemplate.template_content)
    expect(result!.loading.value).toBe(false)
    expect(result!.isNewTemplate.value).toBe(false)
  })

  it('does not fetch on mount for new template', async () => {
    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 'new')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(mockGet).not.toHaveBeenCalled()
    expect(result!.isNewTemplate.value).toBe(true)
    expect(result!.content.value).toBe('')
  })

  it('handles fetch error', async () => {
    mockGet.mockResolvedValue({ data: undefined, error: { message: 'not found' } })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(result!.error.value).toBe('Failed to load template')
    expect(result!.loading.value).toBe(false)
  })

  it('handles fetch exception', async () => {
    mockGet.mockRejectedValue(new Error('Network error'))

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(result!.error.value).toBe('Network error')
  })

  it('tracks isDirty when content changes', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(result!.isDirty.value).toBe(false)

    result!.content.value = 'modified content'
    expect(result!.isDirty.value).toBe(true)

    result!.content.value = mockTemplate.template_content
    expect(result!.isDirty.value).toBe(false)
  })

  it('canSave requires isDirty and non-empty content', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    expect(result!.canSave.value).toBe(false)

    result!.content.value = 'new content'
    expect(result!.canSave.value).toBe(true)

    result!.content.value = '   '
    expect(result!.canSave.value).toBe(false)
  })

  it('saves existing template with PUT', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })
    mockPut.mockResolvedValue({ data: { ...mockTemplate, template_content: 'updated' }, error: undefined })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = 'updated content'
    const success = await result!.saveTemplate()

    expect(success).toBe(true)
    expect(mockPut).toHaveBeenCalledWith('/projects/{projectId}/templates/{templateId}', {
      params: { path: { projectId: 'p1', templateId: 't1' } },
      body: { template_content: 'updated content' },
    })
    expect(result!.isDirty.value).toBe(false)
  })

  it('creates new template with POST', async () => {
    mockPost.mockResolvedValue({ data: { ...mockTemplate, id: 'new-id' }, error: undefined })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 'new')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = 'new template content'
    const success = await result!.saveTemplate('New Template', 'implement')

    expect(success).toBe(true)
    expect(mockPost).toHaveBeenCalledWith('/projects/{projectId}/templates', {
      params: { path: { projectId: 'p1' } },
      body: {
        name: 'New Template',
        type: 'implement',
        template_content: 'new template content',
      },
    })
  })

  it('handles save error for existing template', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })
    mockPut.mockResolvedValue({ data: undefined, error: { message: 'forbidden' } })

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = 'updated content'
    const success = await result!.saveTemplate()

    expect(success).toBe(false)
    expect(result!.error.value).toBe('Failed to save template')
    expect(result!.saving.value).toBe(false)
  })

  it('handles save exception', async () => {
    mockGet.mockResolvedValue({ data: mockTemplate, error: undefined })
    mockPut.mockRejectedValue(new Error('Network error'))

    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 't1')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = 'updated content'
    const success = await result!.saveTemplate()

    expect(success).toBe(false)
    expect(result!.error.value).toBe('Network error')
  })

  it('previews template with client-side Handlebars', async () => {
    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 'new')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = 'Working on {{story_key}}: {{story_title}}'
    await result!.previewTemplate()

    expect(result!.previewContent.value).toBe('Working on S-14: Add user authentication')
    expect(result!.previewError.value).toBeNull()
    expect(result!.previewLoading.value).toBe(false)
  })

  it('handles preview error for invalid template syntax', async () => {
    let result: ReturnType<typeof useTemplateEditor> | undefined

    mount({
      setup() {
        result = useTemplateEditor('p1', 'new')
        return {}
      },
      template: '<div></div>',
    })

    await nextTick()

    result!.content.value = '{{#if}}'
    await result!.previewTemplate()

    expect(result!.previewError.value).toBeTruthy()
    expect(result!.previewLoading.value).toBe(false)
  })
})
