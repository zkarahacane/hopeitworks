<script setup lang="ts">
import { watch } from 'vue'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { apiClient } from '@/api/client'

const props = defineProps<{
  /** The project ID for which the circuit breaker is active */
  projectId: string
  /** Whether the current user has admin privileges */
  isAdmin: boolean
}>()

const emit = defineEmits<{
  /** Fired after the circuit breaker is successfully reset */
  reset: []
}>()

const confirm = useConfirm()
const toast = useToast()

const {
  execute: doReset,
  isLoading,
  error: resetError,
} = useAsyncAction(async () => {
  const { response } = await apiClient.POST('/projects/{id}/circuit-breaker/reset', {
    params: { path: { id: props.projectId } },
  })
  if (!response.ok) {
    throw new Error(`Reset failed with status ${response.status}`)
  }
  emit('reset')
  toast.add({
    severity: 'success',
    summary: 'Circuit breaker reset',
    detail: 'Pipeline runs can now proceed.',
    life: 3000,
  })
})

// Show error toast whenever the reset action fails
watch(resetError, (err) => {
  if (err) {
    toast.add({
      severity: 'error',
      summary: 'Reset failed',
      detail: 'Could not reset the circuit breaker. Please try again.',
      life: 5000,
    })
  }
})

function handleReset() {
  confirm.require({
    message: 'This will allow new pipeline runs to start. Continue?',
    header: 'Reset Circuit Breaker',
    icon: 'pi pi-exclamation-triangle',
    rejectLabel: 'Cancel',
    acceptLabel: 'Reset',
    acceptClass: 'p-button-danger',
    accept: () => doReset(),
  })
}
</script>

<template>
  <Message severity="error" :closable="false" class="mb-4" data-testid="circuit-breaker-banner">
    <div class="flex items-center justify-between w-full gap-4">
      <span>Circuit breaker active — all pipeline runs are paused.</span>
      <Button
        v-if="isAdmin"
        label="Reset"
        severity="danger"
        size="small"
        :loading="isLoading"
        data-testid="reset-button"
        @click="handleReset"
      />
    </div>
  </Message>
</template>
