<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import InputNumber from 'primevue/inputnumber'
import Button from 'primevue/button'
import Message from 'primevue/message'
import type { PipelineStep } from '@/stores/pipelineConfig'

const modelOptions = [
  { label: 'Claude Opus 4.6', value: 'claude-opus-4-6' },
  { label: 'Claude Sonnet 4.6', value: 'claude-sonnet-4-6' },
  { label: 'Claude Haiku 4.5', value: 'claude-haiku-4-5' },
]

const actionTypeOptions = [
  { label: 'Agent Run', value: 'agent_run' },
  { label: 'Git Branch', value: 'git_branch' },
  { label: 'Git PR', value: 'git_pr' },
  { label: 'Notification', value: 'notification' },
  { label: 'Human', value: 'human' },
  { label: 'CI Poll', value: 'ci_poll' },
  { label: 'HITL Gate', value: 'hitl_gate' },
]

const retryTypeOptions = [
  { label: 'None', value: 'none' },
  { label: 'On Failure', value: 'on-failure' },
  { label: 'Always', value: 'always' },
]

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  add: [step: PipelineStep]
  cancel: []
  'update:visible': [value: boolean]
}>()

const name = ref('')
const actionType = ref<PipelineStep['action_type']>('agent_run')
const model = ref<PipelineStep['model']>('claude-sonnet-4-6')
const autoApprove = ref(false)
const maxRetries = ref(2)
const retryType = ref<PipelineStep['retry_policy']['retry_type']>('on-failure')
const validationError = ref<string | null>(null)

function resetForm() {
  name.value = ''
  actionType.value = 'agent_run'
  model.value = 'claude-sonnet-4-6'
  autoApprove.value = false
  maxRetries.value = 2
  retryType.value = 'on-failure'
  validationError.value = null
}

watch(
  () => props.visible,
  (isVisible) => {
    if (isVisible) {
      resetForm()
    }
  },
)

function handleAdd() {
  if (!name.value.trim()) {
    validationError.value = 'Step name is required'
    return
  }

  const step: PipelineStep = {
    id: crypto.randomUUID(),
    name: name.value.trim(),
    action_type: actionType.value,
    model: model.value,
    auto_approve: autoApprove.value,
    retry_policy: {
      max_retries: maxRetries.value,
      retry_type: retryType.value,
    },
  }

  emit('add', step)
  close()
}

function close() {
  emit('update:visible', false)
  emit('cancel')
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Add Pipeline Step"
    class="w-full max-w-lg"
    @update:visible="close"
  >
    <form class="flex flex-col gap-4" @submit.prevent="handleAdd">
      <div class="flex flex-col gap-2">
        <label for="step-name" class="text-sm font-medium">Name *</label>
        <InputText
          id="step-name"
          v-model="name"
          class="w-full"
          placeholder="e.g., implement, review"
          :invalid="!!validationError"
          data-testid="step-name-input"
        />
        <small v-if="validationError" class="text-red-500">{{ validationError }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <label for="action-type" class="text-sm font-medium">Action Type</label>
        <Select
          id="action-type"
          v-model="actionType"
          :options="actionTypeOptions"
          option-label="label"
          option-value="value"
          class="w-full"
          data-testid="action-type-select"
        />
      </div>

      <div class="flex flex-col gap-2">
        <label for="model-select" class="text-sm font-medium">Model</label>
        <Select
          id="model-select"
          v-model="model"
          :options="modelOptions"
          option-label="label"
          option-value="value"
          class="w-full"
          data-testid="model-select"
        />
      </div>

      <div class="flex items-center gap-2">
        <Checkbox
          v-model="autoApprove"
          :binary="true"
          input-id="add-auto-approve"
          data-testid="auto-approve-checkbox"
        />
        <label for="add-auto-approve" class="text-sm font-medium">Auto Approve</label>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div class="flex flex-col gap-2">
          <label for="max-retries" class="text-sm font-medium">Max Retries</label>
          <InputNumber
            id="max-retries"
            v-model="maxRetries"
            :min="0"
            :max="10"
            class="w-full"
            data-testid="max-retries-input"
          />
        </div>

        <div class="flex flex-col gap-2">
          <label for="retry-type" class="text-sm font-medium">Retry Type</label>
          <Select
            id="retry-type"
            v-model="retryType"
            :options="retryTypeOptions"
            option-label="label"
            option-value="value"
            class="w-full"
            data-testid="retry-type-select"
          />
        </div>
      </div>

      <Message v-if="validationError" severity="error" :closable="false">
        {{ validationError }}
      </Message>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="close" />
        <Button
          label="Add"
          severity="success"
          icon="pi pi-plus"
          data-testid="add-step-submit"
          @click="handleAdd"
        />
      </div>
    </template>
  </Dialog>
</template>
