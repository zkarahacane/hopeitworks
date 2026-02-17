import { ref, type Ref } from 'vue'
import { useStoriesStore, type Story, type UpdateStoryFields } from '@/stores/stories'

/**
 * Composable for managing inline story edit mode.
 * Owns all edit-mode state: isEditing, draftFields, validationErrors, apiError, isSaving.
 */
export function useStoryEditor(projectId: string, story: Ref<Story | null>) {
  const store = useStoriesStore()
  const isEditing = ref(false)
  const draftFields = ref<UpdateStoryFields>({})
  const validationErrors = ref<Record<string, string>>({})
  const apiError = ref<string | null>(null)
  const isSaving = ref(false)

  /** Enter edit mode by copying story fields into draft */
  function startEdit() {
    if (!story.value) return
    draftFields.value = {
      title: story.value.title,
      objective: story.value.objective,
      acceptance_criteria: story.value.acceptance_criteria,
      target_files: [...(story.value.target_files ?? [])],
      depends_on: [...(story.value.depends_on ?? [])],
      scope: story.value.scope,
    }
    validationErrors.value = {}
    apiError.value = null
    isEditing.value = true
  }

  /** Exit edit mode and discard draft changes */
  function cancelEdit() {
    isEditing.value = false
    draftFields.value = {}
    validationErrors.value = {}
    apiError.value = null
  }

  /** Validate and save the draft fields via the store */
  async function saveEdit(storyId: string): Promise<Story | null> {
    validationErrors.value = {}
    if (!draftFields.value.title?.trim()) {
      validationErrors.value.title = 'Title is required'
      return null
    }
    isSaving.value = true
    apiError.value = null
    try {
      const updated = await store.updateStory(projectId, storyId, draftFields.value)
      if (updated) {
        isEditing.value = false
        return updated
      }
      apiError.value = store.error ?? 'Failed to save story'
      return null
    } finally {
      isSaving.value = false
    }
  }

  return {
    isEditing,
    draftFields,
    validationErrors,
    apiError,
    isSaving,
    startEdit,
    cancelEdit,
    saveEdit,
  }
}
