import { ref, computed, onMounted } from 'vue'
import Handlebars from 'handlebars'
import { apiClient } from '@/api/client'
import type { Agent, AgentScope } from '@/stores/agents'

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
 * Composable for agent editor state and actions.
 * Handles fetching, saving, and previewing agents.
 */
export function useAgentEditor(projectId: string, agentId: string) {
  const agent = ref<Agent | null>(null)
  const content = ref('')
  const name = ref('')
  const model = ref('claude-sonnet-4-6')
  const image = ref('')
  const scope = ref<AgentScope>('project')
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)
  const previewLoading = ref(false)
  const previewError = ref<string | null>(null)
  const previewContent = ref('')
  const originalContent = ref('')

  const isNewAgent = computed(() => agentId === 'new')
  const isDirty = computed(() => content.value !== originalContent.value)
  const canSave = computed(
    () =>
      isDirty.value &&
      content.value.trim() !== '' &&
      (!isNewAgent.value || name.value.trim() !== ''),
  )

  /** Fetch an existing agent from the API */
  async function fetchAgent() {
    if (isNewAgent.value) return
    loading.value = true
    error.value = null
    try {
      const { data, error: apiError } = await apiClient.GET(
        '/projects/{projectId}/agents/{agentId}',
        {
          params: { path: { projectId, agentId } },
        },
      )
      if (apiError) {
        error.value = 'Failed to load agent'
        return
      }
      const a = data as Agent
      agent.value = a
      content.value = a.template_content
      originalContent.value = a.template_content
      name.value = a.name
      model.value = a.model
      image.value = a.image
      scope.value = a.scope
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load agent'
    } finally {
      loading.value = false
    }
  }

  /** Save or create an agent via the API */
  async function saveAgent(
    agentName?: string,
    agentModel?: string,
    agentImage?: string,
    agentScope?: AgentScope,
  ) {
    saving.value = true
    error.value = null
    try {
      if (isNewAgent.value) {
        const { error: apiError } = await apiClient.POST(
          '/projects/{projectId}/agents',
          {
            params: { path: { projectId } },
            body: {
              name: agentName ?? name.value,
              model: agentModel ?? model.value,
              image: agentImage ?? image.value,
              template_content: content.value,
              scope: agentScope ?? scope.value,
              provider: 'claude',
            },
          },
        )
        if (apiError) {
          error.value = 'Failed to create agent'
          return false
        }
      } else {
        const { error: apiError } = await apiClient.PUT(
          '/projects/{projectId}/agents/{agentId}',
          {
            params: { path: { projectId, agentId } },
            body: {
              template_content: content.value,
              name: name.value,
              model: model.value,
              image: image.value,
            },
          },
        )
        if (apiError) {
          error.value = 'Failed to save agent'
          return false
        }
      }
      originalContent.value = content.value
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to save agent'
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
    if (!isNewAgent.value) {
      fetchAgent()
    }
  })

  return {
    agent,
    content,
    name,
    model,
    image,
    scope,
    loading,
    saving,
    error,
    isDirty,
    canSave,
    isNewAgent,
    previewLoading,
    previewError,
    previewContent,
    fetchAgent,
    saveAgent,
    previewTemplate,
  }
}
