import { ref, computed, onMounted } from 'vue'
import Handlebars from 'handlebars'
import { apiClient } from '@/api/client'
import type { PromptTemplate, PromptTemplateType } from '@/stores/promptTemplates'

/** Sample context data used for template preview rendering */
const sampleContext = {
  story_key: 'S-14',
  story_title: 'Add user authentication',
  story_objective: 'Implement JWT-based authentication with refresh tokens',
  target_files: [
    'backend/internal/api/middleware/auth.go',
    'backend/internal/domain/service/auth_service.go',
  ],
  acceptance_criteria:
    '- Given a valid JWT token, the user can access protected endpoints\n- When the token expires, the user receives a 401 error\n- When the user logs out, the token is invalidated',
  error_context: 'Error: test failed in auth_test.go line 42: expected status 200, got 401',
  diff_content:
    'diff --git a/auth.go b/auth.go\nindex 1234567..abcdefg 100644\n--- a/auth.go\n+++ b/auth.go\n@@ -10,6 +10,7 @@ func Login() {\n+    validateToken()\n',
  branch_name: 'feat/1-3-auth',
  repo_url: 'https://github.com/user/repo',
}

/**
 * Composable for template editor state and actions.
 * Handles fetching, saving, and previewing prompt templates.
 */
export function useTemplateEditor(projectId: string, templateId: string) {
  const template = ref<PromptTemplate | null>(null)
  const content = ref('')
  const name = ref('')
  const type = ref<PromptTemplateType>('custom')
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)
  const previewLoading = ref(false)
  const previewError = ref<string | null>(null)
  const previewContent = ref('')
  const originalContent = ref('')

  const isNewTemplate = computed(() => templateId === 'new')
  const isDirty = computed(() => content.value !== originalContent.value)
  const canSave = computed(() => isDirty.value && content.value.trim() !== '')

  /** Fetch an existing template from the API */
  async function fetchTemplate() {
    if (isNewTemplate.value) return
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/templates/{templateId}',
        {
          params: { path: { projectId, templateId } },
        },
      )
      if (apiError) {
        error.value = 'Failed to load template'
        return
      }
      const t = data as PromptTemplate
      template.value = t
      content.value = t.template_content
      originalContent.value = t.template_content
      name.value = t.name
      type.value = t.type
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load template'
    } finally {
      loading.value = false
    }
  }

  /** Save or create a template via the API */
  async function saveTemplate(templateName?: string, templateType?: PromptTemplateType) {
    saving.value = true
    error.value = null
    try {
      if (isNewTemplate.value) {
        const { error: apiError } = await apiClient.POST(
          '/projects/{projectId}/templates',
          {
            params: { path: { projectId } },
            body: {
              name: templateName ?? name.value,
              type: templateType ?? type.value,
              template_content: content.value,
            },
          },
        )
        if (apiError) {
          error.value = 'Failed to create template'
          return false
        }
      } else {
        const { error: apiError } = await apiClient.PUT(
          '/projects/{projectId}/templates/{templateId}',
          {
            params: { path: { projectId, templateId } },
            body: { template_content: content.value },
          },
        )
        if (apiError) {
          error.value = 'Failed to save template'
          return false
        }
      }
      originalContent.value = content.value
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to save template'
      return false
    } finally {
      saving.value = false
    }
  }

  /** Render the template with sample context data using client-side Handlebars */
  async function previewTemplate() {
    previewLoading.value = true
    previewError.value = null
    try {
      const compiled = Handlebars.compile(content.value)
      previewContent.value = compiled(sampleContext)
    } catch (e) {
      previewError.value = e instanceof Error ? e.message : 'Failed to render preview'
    } finally {
      previewLoading.value = false
    }
  }

  onMounted(() => {
    if (!isNewTemplate.value) {
      fetchTemplate()
    }
  })

  return {
    template,
    content,
    name,
    type,
    loading,
    saving,
    error,
    isDirty,
    canSave,
    isNewTemplate,
    previewLoading,
    previewError,
    previewContent,
    fetchTemplate,
    saveTemplate,
    previewTemplate,
  }
}
