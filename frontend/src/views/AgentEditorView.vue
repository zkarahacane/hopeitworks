<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { useAuth } from '@/composables/useAuth'
import { useAgentEditor } from '@/composables/useAgentEditor'
import AgentEditorLayout from '@/features/agents/AgentEditorLayout.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const projectId = computed(() => (route.params.id ?? route.params.projectId) as string)
const agentId = computed(() => (route.params.agentId as string) ?? 'new')

const { user } = useAuth()
const isAdmin = computed(() => user.value?.role === 'admin')

const {
  agent,
  content,
  name,
  model,
  image,
  scope,
  provider,
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
} = useAgentEditor(projectId.value, agentId.value)

/** Whether the editor should be read-only (global agent + non-admin) */
const isReadOnly = computed(() => agent.value?.scope === 'global' && !isAdmin.value)

const previewVisible = ref(false)

async function handleSave() {
  const success = await saveAgent()
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: isNewAgent.value ? 'Agent created successfully' : 'Agent saved successfully',
      life: 3000,
    })
    // Return to the agents list after a successful save (consistent for create and edit)
    router.push({ name: 'project-agents', params: { id: projectId.value } })
  } else {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: error.value ?? 'Failed to save agent',
      life: 5000,
    })
  }
}

function handleCancel() {
  router.push({ name: 'project-agents', params: { id: projectId.value } })
}

async function handlePreview() {
  await previewTemplate()
  previewVisible.value = true
}
</script>

<template>
  <Toast />

  <!-- Loading state -->
  <div v-if="loading" class="flex flex-col gap-4 p-6">
    <Skeleton height="2rem" width="30%" />
    <Skeleton height="60vh" />
  </div>

  <!-- Error state -->
  <div v-else-if="error && !content" class="p-6">
    <Message severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="fetchAgent()" />
      </div>
    </Message>
  </div>

  <!-- Editor -->
  <AgentEditorLayout
    v-else
    :content="content"
    :is-admin="isAdmin"
    :is-dirty="isDirty"
    :is-saving="saving"
    :can-save="canSave"
    :is-new-agent="isNewAgent"
    :is-read-only="isReadOnly"
    :agent-name="name"
    :agent-model="model"
    :agent-image="image"
    :agent-scope="scope"
    :agent-provider="provider"
    :preview-visible="previewVisible"
    :preview-content="previewContent"
    :preview-loading="previewLoading"
    :preview-error="previewError"
    @update:content="content = $event"
    @update:agent-name="name = $event"
    @update:agent-model="model = $event"
    @update:agent-image="image = $event"
    @update:agent-scope="scope = $event"
    @update:agent-provider="provider = $event"
    @update:preview-visible="previewVisible = $event"
    @save="handleSave"
    @cancel="handleCancel"
    @preview="handlePreview"
  />
</template>
