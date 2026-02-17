import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { useStoryEditor } from '../useStoryEditor'
import type { Story } from '@/stores/stories'
import { useStoriesStore } from '@/stores/stories'

const mockPut = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    GET: vi.fn(),
    PUT: (...args: unknown[]) => mockPut(...args),
    POST: vi.fn(),
  },
}))

function makeStory(overrides: Partial<Story> = {}): Story {
  return {
    id: 's1',
    epic_id: 'e1',
    project_id: 'p1',
    key: 'S-01',
    title: 'Test Story',
    status: 'backlog',
    objective: 'Build the feature',
    acceptance_criteria: '- Item 1\n- Item 2',
    target_files: ['src/foo.ts', 'src/bar.ts'],
    depends_on: ['S-02'],
    scope: 'backend',
    created_at: '2026-01-15T10:00:00Z',
    updated_at: '2026-01-15T10:00:00Z',
    ...overrides,
  }
}

describe('useStoryEditor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockPut.mockReset()
  })

  it('starts with isEditing false', () => {
    const story = ref<Story | null>(makeStory())
    const { isEditing } = useStoryEditor('p1', story)
    expect(isEditing.value).toBe(false)
  })

  describe('startEdit', () => {
    it('copies story fields into draftFields and sets isEditing true', () => {
      const story = ref<Story | null>(makeStory())
      const { startEdit, isEditing, draftFields } = useStoryEditor('p1', story)

      startEdit()

      expect(isEditing.value).toBe(true)
      expect(draftFields.value.title).toBe('Test Story')
      expect(draftFields.value.objective).toBe('Build the feature')
      expect(draftFields.value.acceptance_criteria).toBe('- Item 1\n- Item 2')
      expect(draftFields.value.target_files).toEqual(['src/foo.ts', 'src/bar.ts'])
      expect(draftFields.value.depends_on).toEqual(['S-02'])
      expect(draftFields.value.scope).toBe('backend')
    })

    it('deep clones target_files so mutations do not affect original', () => {
      const story = ref<Story | null>(makeStory({ target_files: ['a.ts'] }))
      const { startEdit, draftFields } = useStoryEditor('p1', story)

      startEdit()
      draftFields.value.target_files!.push('b.ts')

      expect(story.value!.target_files).toEqual(['a.ts'])
    })

    it('clears previous validation errors and api error', () => {
      const story = ref<Story | null>(makeStory())
      const { startEdit, validationErrors, apiError } = useStoryEditor('p1', story)

      // Simulate previous errors
      validationErrors.value = { title: 'Required' }
      apiError.value = 'Server error'

      startEdit()

      expect(validationErrors.value).toEqual({})
      expect(apiError.value).toBeNull()
    })

    it('does nothing when story is null', () => {
      const story = ref<Story | null>(null)
      const { startEdit, isEditing } = useStoryEditor('p1', story)

      startEdit()

      expect(isEditing.value).toBe(false)
    })
  })

  describe('cancelEdit', () => {
    it('resets state and sets isEditing false', () => {
      const story = ref<Story | null>(makeStory())
      const { startEdit, cancelEdit, isEditing, draftFields, validationErrors, apiError } =
        useStoryEditor('p1', story)

      startEdit()
      draftFields.value.title = 'Modified'
      validationErrors.value = { title: 'Error' }
      apiError.value = 'API error'

      cancelEdit()

      expect(isEditing.value).toBe(false)
      expect(draftFields.value).toEqual({})
      expect(validationErrors.value).toEqual({})
      expect(apiError.value).toBeNull()
    })
  })

  describe('saveEdit', () => {
    it('calls store.updateStory and sets isEditing false on success', async () => {
      const story = ref<Story | null>(makeStory())
      const { startEdit, saveEdit, isEditing } = useStoryEditor('p1', story)

      startEdit()

      const updatedStory = makeStory({ title: 'Updated' })
      mockPut.mockResolvedValue({ data: updatedStory, error: undefined })

      // Populate store items so updateStory can find the story
      const store = useStoriesStore()
      store.items = [makeStory()]

      const result = await saveEdit('s1')

      expect(result).toEqual(updatedStory)
      expect(isEditing.value).toBe(false)
    })

    it('sets validation error when title is empty', async () => {
      const story = ref<Story | null>(makeStory())
      const { startEdit, saveEdit, validationErrors, isEditing } = useStoryEditor('p1', story)

      startEdit()
      // Clear the title
      const { draftFields } = useStoryEditor('p1', story)
      draftFields.value = { ...draftFields.value }

      // Use the original editor instance
      const editor = useStoryEditor('p1', story)
      editor.startEdit()
      editor.draftFields.value.title = ''

      const result = await editor.saveEdit('s1')

      expect(result).toBeNull()
      expect(editor.validationErrors.value.title).toBe('Title is required')
      expect(editor.isEditing.value).toBe(true)
    })

    it('sets validation error when title is whitespace only', async () => {
      const story = ref<Story | null>(makeStory())
      const editor = useStoryEditor('p1', story)

      editor.startEdit()
      editor.draftFields.value.title = '   '

      const result = await editor.saveEdit('s1')

      expect(result).toBeNull()
      expect(editor.validationErrors.value.title).toBe('Title is required')
    })

    it('does not call store when validation fails', async () => {
      const story = ref<Story | null>(makeStory())
      const editor = useStoryEditor('p1', story)

      editor.startEdit()
      editor.draftFields.value.title = ''

      await editor.saveEdit('s1')

      expect(mockPut).not.toHaveBeenCalled()
    })

    it('populates apiError and keeps isEditing true on API error', async () => {
      const story = ref<Story | null>(makeStory())
      const editor = useStoryEditor('p1', story)

      editor.startEdit()

      mockPut.mockResolvedValue({
        data: undefined,
        error: { error: { code: 'INTERNAL', message: 'Server error' } },
      })

      const result = await editor.saveEdit('s1')

      expect(result).toBeNull()
      expect(editor.apiError.value).toBeTruthy()
      expect(editor.isEditing.value).toBe(true)
    })

    it('sets isSaving during save operation', async () => {
      const story = ref<Story | null>(makeStory())
      const editor = useStoryEditor('p1', story)

      editor.startEdit()

      const store = useStoriesStore()
      store.items = [makeStory()]

      let savingDuringCall = false
      mockPut.mockImplementation(() => {
        savingDuringCall = editor.isSaving.value
        return Promise.resolve({ data: makeStory(), error: undefined })
      })

      await editor.saveEdit('s1')

      expect(savingDuringCall).toBe(true)
      expect(editor.isSaving.value).toBe(false)
    })
  })
})
