<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'

const props = defineProps<{
  visible: boolean
  projectName: string
  loading: boolean
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
  'update:visible': [value: boolean]
}>()

const confirmationName = ref('')

// Reset the typed name whenever the dialog (re)opens, so a previous attempt
// never leaves a stale match that would pre-enable the destructive button.
watch(
  () => props.visible,
  (isOpen) => {
    if (isOpen) confirmationName.value = ''
  },
)

const nameMatches = computed(() => confirmationName.value === props.projectName)

function handleCancel() {
  emit('cancel')
  emit('update:visible', false)
}

function handleConfirm() {
  if (!nameMatches.value || props.loading) return
  emit('confirm')
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Delete project"
    class="w-full max-w-lg"
    data-testid="project-delete-dialog"
    @update:visible="handleCancel"
  >
    <div class="flex flex-col gap-4">
      <div id="delete-cascade-warning">
        <Message severity="error" :closable="false" data-testid="delete-cascade-warning">
          This permanently deletes <strong>{{ projectName }}</strong> and everything linked to it,
          runs, stories, epics, and pipeline configs. This action cannot be undone.
        </Message>
      </div>

      <div class="flex flex-col gap-2">
        <label for="delete-confirm-name" class="text-sm">
          Type <strong>{{ projectName }}</strong> to confirm.
        </label>
        <InputText
          id="delete-confirm-name"
          v-model="confirmationName"
          class="w-full"
          autocomplete="off"
          aria-describedby="delete-cascade-warning"
          data-testid="delete-confirm-input"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button
          label="Cancel"
          severity="secondary"
          text
          :disabled="loading"
          data-testid="delete-cancel-btn"
          @click="handleCancel"
        />
        <Button
          label="Delete permanently"
          severity="danger"
          icon="pi pi-trash"
          :loading="loading"
          :disabled="!nameMatches || loading"
          data-testid="delete-confirm-btn"
          @click="handleConfirm"
        />
      </div>
    </template>
  </Dialog>
</template>
