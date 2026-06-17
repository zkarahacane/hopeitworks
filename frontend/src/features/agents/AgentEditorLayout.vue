<script setup lang="ts">
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AutoComplete from 'primevue/autocomplete'
import type { AutoCompleteCompleteEvent } from 'primevue/autocomplete'
import Message from 'primevue/message'
import MonacoEditorWrapper from './MonacoEditorWrapper.vue'
import AgentVariableSidebar from './AgentVariableSidebar.vue'
import AgentEditorToolbar from './AgentEditorToolbar.vue'
import AgentPreviewDialog from './AgentPreviewDialog.vue'
import type { AgentScope } from '@/stores/agents'
import { DEFAULT_MODEL_SUGGESTIONS } from '@/utils/models'

interface Props {
  content: string
  isAdmin: boolean
  isDirty: boolean
  isSaving: boolean
  canSave: boolean
  isNewAgent: boolean
  isReadOnly: boolean
  agentName: string
  agentModel: string
  agentImage: string
  agentScope: AgentScope
  agentProvider: string
  previewVisible: boolean
  previewContent: string
  previewLoading: boolean
  previewError: string | null
}

defineProps<Props>()

const emit = defineEmits<{
  'update:content': [content: string]
  'update:agentName': [name: string]
  'update:agentModel': [model: string]
  'update:agentImage': [image: string]
  'update:agentScope': [scope: AgentScope]
  'update:agentProvider': [provider: string]
  'update:previewVisible': [visible: boolean]
  save: []
  cancel: []
  preview: []
}>()

const scopeOptions = [
  { label: 'Project', value: 'project' },
  { label: 'Global', value: 'global' },
]

const providerOptions = [
  { label: 'Claude', value: 'claude' },
  { label: 'OpenCode', value: 'opencode' },
]

const filteredModels = ref<string[]>([])

function searchModels(event: AutoCompleteCompleteEvent) {
  const query = event.query.toLowerCase()
  filteredModels.value = query
    ? DEFAULT_MODEL_SUGGESTIONS.filter((m) => m.toLowerCase().includes(query))
    : [...DEFAULT_MODEL_SUGGESTIONS]
}

const editorRef = ref<InstanceType<typeof MonacoEditorWrapper> | null>(null)
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Toolbar -->
    <AgentEditorToolbar
      :style="{ borderBottom: '1px solid var(--surface-border)' }"
      :is-admin="isAdmin"
      :can-save="canSave"
      :is-saving="isSaving"
      :is-dirty="isDirty"
      :is-read-only="isReadOnly"
      @preview="emit('preview')"
      @save="emit('save')"
      @cancel="emit('cancel')"
    />

    <!-- Read-only notice for global agents -->
    <div v-if="isReadOnly" class="px-4 pt-3">
      <Message severity="info" :closable="false">
        Global agents can only be edited by administrators
      </Message>
    </div>

    <!-- Agent metadata fields -->
    <div
      class="flex flex-wrap items-end gap-4 px-4 py-3"
      :style="{ borderBottom: '1px solid var(--surface-border)' }"
    >
      <div class="flex min-w-[200px] flex-1 flex-col gap-1">
        <label class="text-xs font-medium" :style="{ color: 'var(--p-text-muted-color)' }">Agent Name</label>
        <InputText
          :value="agentName"
          placeholder="e.g. Default Implement Agent"
          size="small"
          class="w-full"
          :disabled="isReadOnly"
          @input="emit('update:agentName', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium" :style="{ color: 'var(--p-text-muted-color)' }">Model</label>
        <AutoComplete
          :model-value="agentModel"
          :suggestions="filteredModels"
          dropdown
          :disabled="isReadOnly"
          placeholder="Type or select a model..."
          class="w-64"
          @complete="searchModels"
          @update:model-value="emit('update:agentModel', $event)"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium" :style="{ color: 'var(--p-text-muted-color)' }">Provider</label>
        <Select
          :model-value="agentProvider"
          :options="providerOptions"
          option-label="label"
          option-value="value"
          size="small"
          class="w-40"
          :disabled="isReadOnly"
          @update:model-value="emit('update:agentProvider', $event)"
        />
      </div>
      <div class="flex min-w-[200px] flex-1 flex-col gap-1">
        <label class="text-xs font-medium" :style="{ color: 'var(--p-text-muted-color)' }">Docker Image</label>
        <InputText
          :value="agentImage"
          placeholder="ghcr.io/org/agent-name:latest"
          size="small"
          class="w-full"
          :disabled="isReadOnly"
          @input="emit('update:agentImage', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium" :style="{ color: 'var(--p-text-muted-color)' }">Scope</label>
        <Select
          :model-value="agentScope"
          :options="scopeOptions"
          option-label="label"
          option-value="value"
          size="small"
          class="w-36"
          :disabled="isReadOnly || !isNewAgent"
          @update:model-value="emit('update:agentScope', $event)"
        />
      </div>
    </div>

    <!-- Main content area -->
    <div class="flex flex-1 overflow-hidden">
      <!-- Monaco editor -->
      <div class="flex-1">
        <MonacoEditorWrapper
          ref="editorRef"
          :model-value="content"
          :readonly="isReadOnly"
          @update:model-value="emit('update:content', $event)"
        />
      </div>

      <!-- Variable sidebar -->
      <AgentVariableSidebar
        class="w-[250px] shrink-0 overflow-y-auto"
        :style="{ borderLeft: '1px solid var(--surface-border)' }"
        :editor-ref="editorRef"
      />
    </div>

    <!-- Preview dialog -->
    <AgentPreviewDialog
      :visible="previewVisible"
      :rendered-content="previewContent"
      :loading="previewLoading"
      :error="previewError"
      @update:visible="emit('update:previewVisible', $event)"
    />
  </div>
</template>
