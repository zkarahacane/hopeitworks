<script setup lang="ts">
import { computed } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import InputNumber from 'primevue/inputnumber'
import type { PipelineStep } from '@/stores/pipelineConfig'

const modelOptions = [
  { label: 'Claude Opus 4.6', value: 'claude-opus-4-6' },
  { label: 'Claude Sonnet 4.6', value: 'claude-sonnet-4-6' },
  { label: 'Claude Haiku 4.5', value: 'claude-haiku-4-5' },
]

const retryTypeOptions = [
  { label: 'None', value: 'none' },
  { label: 'On Failure', value: 'on-failure' },
  { label: 'Always', value: 'always' },
]

const props = defineProps<{
  step: PipelineStep
  index: number
  isAdmin: boolean
  expanded: boolean
  isFirst: boolean
  isLast: boolean
}>()

const emit = defineEmits<{
  toggle: []
  update: [step: PipelineStep]
  remove: []
  moveUp: []
  moveDown: []
}>()

const modelLabel = computed(() => {
  return modelOptions.find((o) => o.value === props.step.model)?.label ?? props.step.model
})

const autoApproveSeverity = computed(() => {
  return props.step.auto_approve ? 'success' : 'secondary'
})

function onModelChange(value: PipelineStep['model']) {
  emit('update', { ...props.step, model: value })
}

function onAutoApproveChange(value: boolean) {
  emit('update', { ...props.step, auto_approve: value })
}

function onMaxRetriesChange(value: number) {
  emit('update', {
    ...props.step,
    retry_policy: { ...props.step.retry_policy, max_retries: value },
  })
}

function onRetryTypeChange(value: PipelineStep['retry_policy']['retry_type']) {
  emit('update', {
    ...props.step,
    retry_policy: { ...props.step.retry_policy, retry_type: value },
  })
}
</script>

<template>
  <Card class="cursor-pointer" @click="emit('toggle')">
    <template #content>
      <div class="flex flex-col gap-3">
        <!-- Collapsed header row -->
        <div class="flex items-center gap-3">
          <span class="font-mono text-sm opacity-60">{{ index + 1 }}.</span>
          <span class="font-semibold">{{ step.name }}</span>
          <Tag :value="step.action_type" severity="info" />
          <span class="text-sm opacity-70">{{ modelLabel }}</span>
          <Tag
            :value="step.auto_approve ? 'Auto' : 'Manual'"
            :severity="autoApproveSeverity"
            class="ml-auto"
          />

          <template v-if="isAdmin">
            <Button
              icon="pi pi-arrow-up"
              text
              rounded
              size="small"
              :disabled="isFirst"
              aria-label="Move up"
              data-testid="move-up"
              @click.stop="emit('moveUp')"
            />
            <Button
              icon="pi pi-arrow-down"
              text
              rounded
              size="small"
              :disabled="isLast"
              aria-label="Move down"
              data-testid="move-down"
              @click.stop="emit('moveDown')"
            />
            <Button
              icon="pi pi-trash"
              text
              rounded
              size="small"
              severity="danger"
              aria-label="Remove step"
              data-testid="remove-step"
              @click.stop="emit('remove')"
            />
          </template>
        </div>

        <!-- Expanded details -->
        <div v-if="expanded" class="flex flex-col gap-4 pt-3" @click.stop>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            <div class="flex flex-col gap-2">
              <label class="text-sm font-medium">Model</label>
              <Select
                :model-value="step.model"
                :options="modelOptions"
                option-label="label"
                option-value="value"
                :disabled="!isAdmin"
                class="w-full"
                data-testid="model-select"
                @update:model-value="onModelChange"
              />
            </div>

            <div class="flex flex-col gap-2">
              <label class="text-sm font-medium">Auto Approve</label>
              <div class="flex items-center gap-2">
                <Checkbox
                  :model-value="step.auto_approve"
                  :binary="true"
                  :disabled="!isAdmin"
                  input-id="auto-approve"
                  data-testid="auto-approve-checkbox"
                  @update:model-value="onAutoApproveChange"
                />
                <label for="auto-approve" class="text-sm">
                  {{ step.auto_approve ? 'Enabled' : 'Disabled' }}
                </label>
              </div>
            </div>

            <div class="flex flex-col gap-2">
              <label class="text-sm font-medium">Max Retries</label>
              <InputNumber
                :model-value="step.retry_policy.max_retries"
                :min="0"
                :max="10"
                :disabled="!isAdmin"
                class="w-full"
                data-testid="max-retries-input"
                @update:model-value="onMaxRetriesChange"
              />
            </div>

            <div class="flex flex-col gap-2">
              <label class="text-sm font-medium">Retry Type</label>
              <Select
                :model-value="step.retry_policy.retry_type"
                :options="retryTypeOptions"
                option-label="label"
                option-value="value"
                :disabled="!isAdmin"
                class="w-full"
                data-testid="retry-type-select"
                @update:model-value="onRetryTypeChange"
              />
            </div>
          </div>
        </div>
      </div>
    </template>
  </Card>
</template>
