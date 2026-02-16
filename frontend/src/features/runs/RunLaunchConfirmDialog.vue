<script setup lang="ts">
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'

defineProps<{
  visible: boolean
  storyKey: string
  storyTitle: string
  loading: boolean
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
  'update:visible': [value: boolean]
}>()

function handleCancel() {
  emit('cancel')
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Launch Story Run"
    class="w-full max-w-lg"
    @update:visible="handleCancel"
  >
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <span class="font-semibold">{{ storyKey }}</span>
        <span>{{ storyTitle }}</span>
      </div>

      <p>
        Launching this run will start an AI agent container. The run will consume
        Claude API credits and Docker resources. Do you want to proceed?
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="handleCancel" />
        <Button
          label="Confirm"
          severity="success"
          icon="pi pi-play"
          :loading="loading"
          :disabled="loading"
          @click="emit('confirm')"
        />
      </div>
    </template>
  </Dialog>
</template>
