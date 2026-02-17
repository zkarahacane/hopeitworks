<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'
import Message from 'primevue/message'
import Skeleton from 'primevue/skeleton'
import { usePromptTemplates } from '@/composables/usePromptTemplates'
import { useAuth } from '@/composables/useAuth'
import PromptTemplateTable from '@/features/templates/PromptTemplateTable.vue'
import PromptTemplateEmptyState from '@/features/templates/PromptTemplateEmptyState.vue'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string

const { user } = useAuth()
const isAdmin = computed(() => user.value?.role === 'admin')

const { templates, isLoading, error, fetchTemplates, retry } = usePromptTemplates(projectId)

onMounted(() => {
  fetchTemplates()
})

function handleRowClick(templateId: string) {
  router.push(`/projects/${projectId}/templates/${templateId}`)
}

function handleCreateClick() {
  router.push(`/projects/${projectId}/templates/new`)
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Prompt Templates</h1>
      <Button
        v-if="isAdmin && templates.length > 0"
        label="Create Template"
        icon="pi pi-plus"
        severity="success"
        @click="handleCreateClick"
      />
    </div>

    <!-- Loading state -->
    <div v-if="isLoading && templates.length === 0" class="flex flex-col gap-4">
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
      <Skeleton height="2rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ error }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="retry()" />
      </div>
    </Message>

    <!-- Empty state -->
    <PromptTemplateEmptyState
      v-else-if="!isLoading && !error && templates.length === 0"
      :is-admin="isAdmin"
      @create-click="handleCreateClick"
    />

    <!-- Data state -->
    <PromptTemplateTable
      v-else
      :templates="templates"
      @row-click="handleRowClick"
    />
  </div>
</template>
