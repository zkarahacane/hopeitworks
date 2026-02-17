<script setup lang="ts">
import { ref } from 'vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import MonacoEditorWrapper from './MonacoEditorWrapper.vue'
import TemplateVariableSidebar from './TemplateVariableSidebar.vue'
import TemplateEditorToolbar from './TemplateEditorToolbar.vue'
import TemplatePreviewDialog from './TemplatePreviewDialog.vue'
import type { PromptTemplateType } from '@/stores/promptTemplates'

interface Props {
  content: string
  isAdmin: boolean
  isDirty: boolean
  isSaving: boolean
  canSave: boolean
  isNewTemplate: boolean
  templateName: string
  templateType: PromptTemplateType
  previewVisible: boolean
  previewContent: string
  previewLoading: boolean
  previewError: string | null
}

defineProps<Props>()

const emit = defineEmits<{
  'update:content': [content: string]
  'update:templateName': [name: string]
  'update:templateType': [type: PromptTemplateType]
  'update:previewVisible': [visible: boolean]
  save: []
  cancel: []
  preview: []
}>()

const typeOptions: { label: string; value: PromptTemplateType }[] = [
  { label: 'Implement', value: 'implement' },
  { label: 'Retry', value: 'retry' },
  { label: 'Review', value: 'review' },
  { label: 'Merge', value: 'merge' },
  { label: 'Custom', value: 'custom' },
]

const editorRef = ref<InstanceType<typeof MonacoEditorWrapper> | null>(null)
</script>

<template>
  <div class="flex h-full flex-col">
    <!-- Toolbar -->
    <TemplateEditorToolbar
      class="border-b border-surface-200 dark:border-surface-700"
      :is-admin="isAdmin"
      :can-save="canSave"
      :is-saving="isSaving"
      :is-dirty="isDirty"
      @preview="emit('preview')"
      @save="emit('save')"
      @cancel="emit('cancel')"
    />

    <!-- Name / type fields shown only when creating a new template -->
    <div
      v-if="isNewTemplate"
      class="flex items-center gap-4 border-b border-surface-200 px-4 py-3 dark:border-surface-700"
    >
      <div class="flex flex-1 flex-col gap-1">
        <label class="text-xs font-medium text-surface-500">Template Name</label>
        <InputText
          :value="templateName"
          placeholder="e.g. Default Implement Prompt"
          size="small"
          class="w-full"
          @input="emit('update:templateName', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-xs font-medium text-surface-500">Type</label>
        <Select
          :model-value="templateType"
          :options="typeOptions"
          option-label="label"
          option-value="value"
          size="small"
          class="w-40"
          @update:model-value="emit('update:templateType', $event)"
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
          :readonly="!isAdmin"
          @update:model-value="emit('update:content', $event)"
        />
      </div>

      <!-- Variable sidebar -->
      <TemplateVariableSidebar
        class="w-[250px] shrink-0 overflow-y-auto border-l border-surface-200 dark:border-surface-700"
        :editor-ref="editorRef"
      />
    </div>

    <!-- Preview dialog -->
    <TemplatePreviewDialog
      :visible="previewVisible"
      :rendered-content="previewContent"
      :loading="previewLoading"
      :error="previewError"
      @update:visible="emit('update:previewVisible', $event)"
    />
  </div>
</template>
