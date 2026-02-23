<script setup lang="ts">
import Button from 'primevue/button'

interface Props {
  isAdmin: boolean
  canSave: boolean
  isSaving: boolean
  isDirty: boolean
  isReadOnly: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  preview: []
  save: []
  cancel: []
}>()
</script>

<template>
  <div class="flex items-center justify-between px-4 py-2">
    <div class="flex items-center gap-2">
      <span v-if="isDirty" class="text-xs text-orange-500">Unsaved changes</span>
      <span v-if="isReadOnly" class="text-xs text-surface-500">Read-only</span>
    </div>
    <div class="flex items-center gap-2">
      <Button
        label="Preview"
        icon="pi pi-eye"
        severity="secondary"
        outlined
        size="small"
        @click="emit('preview')"
      />
      <Button
        v-if="isAdmin && !isReadOnly"
        label="Save"
        icon="pi pi-save"
        severity="success"
        size="small"
        :disabled="!canSave"
        :loading="isSaving"
        @click="emit('save')"
      />
      <Button
        label="Cancel"
        severity="secondary"
        text
        size="small"
        @click="emit('cancel')"
      />
    </div>
  </div>
</template>
