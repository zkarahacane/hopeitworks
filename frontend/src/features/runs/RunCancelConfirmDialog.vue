<script setup lang="ts">
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'

defineProps<{
  visible: boolean
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
    header="Cancel Run"
    class="w-full max-w-lg"
    @update:visible="handleCancel"
  >
    <div class="flex flex-col gap-4">
      <p>
        Are you sure you want to cancel this run? Running containers will be stopped
        and pending steps will be skipped. This action cannot be undone.
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Keep Running" severity="secondary" text @click="handleCancel" />
        <Button
          label="Cancel Run"
          severity="danger"
          icon="pi pi-times"
          :loading="loading"
          :disabled="loading"
          data-testid="confirm-cancel-run-btn"
          @click="emit('confirm')"
        />
      </div>
    </template>
  </Dialog>
</template>
