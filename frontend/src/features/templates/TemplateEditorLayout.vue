<script setup lang="ts">
import { ref } from 'vue'
import MonacoEditorWrapper from './MonacoEditorWrapper.vue'
import TemplateVariableSidebar from './TemplateVariableSidebar.vue'
import TemplateEditorToolbar from './TemplateEditorToolbar.vue'
import TemplatePreviewDialog from './TemplatePreviewDialog.vue'

interface Props {
  content: string
  isAdmin: boolean
  isDirty: boolean
  isSaving: boolean
  canSave: boolean
  previewVisible: boolean
  previewContent: string
  previewLoading: boolean
  previewError: string | null
}

defineProps<Props>()

const emit = defineEmits<{
  'update:content': [content: string]
  'update:previewVisible': [visible: boolean]
  save: []
  cancel: []
  preview: []
}>()

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
