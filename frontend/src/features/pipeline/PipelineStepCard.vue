<script setup lang="ts">
import { computed } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import InputNumber from 'primevue/inputnumber'
import type { PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

const ACTION_TYPE_ICONS: Record<string, string> = {
  agent_run: 'pi pi-android',
  git_branch: 'pi pi-code-branch',
  git_pr: 'pi pi-arrow-right-arrow-left',
  notification: 'pi pi-bell',
  human: 'pi pi-user',
  ci_poll: 'pi pi-sync',
  hitl_gate: 'pi pi-shield',
}

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
  agents: Agent[]
}>()

const emit = defineEmits<{
  toggle: []
  update: [step: PipelineStep]
  remove: []
  moveUp: []
  moveDown: []
}>()

const actionTypeIcon = computed(() => {
  return ACTION_TYPE_ICONS[props.step.action_type] ?? 'pi pi-cog'
})

/** Display label for the collapsed header: agent name if agent_id present, else legacy model string */
const agentDisplay = computed(() => {
  if (props.step.agent_id) {
    return props.agents.find((a) => a.id === props.step.agent_id)?.name ?? props.step.agent_id
  }
  return props.step.model ?? null
})

const autoApproveSeverity = computed(() => {
  return props.step.auto_approve ? 'success' : 'secondary'
})

function onAgentChange(value: string) {
  emit('update', { ...props.step, agent_id: value, model: undefined })
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
          <Tag :value="step.action_type" severity="info" :icon="actionTypeIcon" data-testid="action-type-tag" />
          <span v-if="agentDisplay" class="text-sm opacity-70" data-testid="agent-display">{{ agentDisplay }}</span>
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
            <div v-if="step.action_type === 'agent_run'" class="flex flex-col gap-2">
              <label class="text-sm font-medium">Agent</label>
              <Select
                :model-value="step.agent_id"
                :options="agents"
                option-label="name"
                option-value="id"
                placeholder="Select an agent"
                :disabled="!isAdmin"
                class="w-full"
                data-testid="agent-select"
                @update:model-value="onAgentChange"
              >
                <template #option="{ option }">
                  <div class="flex flex-col">
                    <span>{{ option.name }}</span>
                    <span class="text-sm opacity-60">{{ option.model }}</span>
                  </div>
                </template>
              </Select>
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
