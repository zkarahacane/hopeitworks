<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import { useAuth } from '@/composables/useAuth'
import { useTemplateEditor } from '@/composables/useTemplateEditor'
import TemplateEditorLayout from '@/features/templates/TemplateEditorLayout.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const projectId = computed(() => (route.params.id ?? route.params.projectId) as string)
const templateId = computed(() => (route.params.templateId as string) ?? 'new')

const { user } = useAuth()
const isAdmin = computed(() => user.value?.role === 'admin')

const {
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
} = useTemplateEditor(projectId.value, templateId.value)

const previewVisible = ref(false)

async function handleSave() {
  const success = await saveTemplate()
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: isNewTemplate.value ? 'Template created successfully' : 'Template saved successfully',
      life: 3000,
    })
    if (isNewTemplate.value) {
      router.push({ name: 'project-templates', params: { id: projectId.value } })
    }
  } else {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: error.value ?? 'Failed to save template',
      life: 5000,
    })
  }
}

function handleCancel() {
  router.push({ name: 'project-templates', params: { id: projectId.value } })
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
        <Button label="Retry" severity="secondary" text size="small" @click="fetchTemplate()" />
      </div>
    </Message>
  </div>

  <!-- Editor -->
  <TemplateEditorLayout
    v-else
    :content="content"
    :is-admin="isAdmin"
    :is-dirty="isDirty"
    :is-saving="saving"
    :can-save="canSave"
    :is-new-template="isNewTemplate"
    :template-name="name"
    :template-type="type"
    :preview-visible="previewVisible"
    :preview-content="previewContent"
    :preview-loading="previewLoading"
    :preview-error="previewError"
    @update:content="content = $event"
    @update:template-name="name = $event"
    @update:template-type="type = $event"
    @update:preview-visible="previewVisible = $event"
    @save="handleSave"
    @cancel="handleCancel"
    @preview="handlePreview"
  />
</template>
