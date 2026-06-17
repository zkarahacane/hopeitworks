<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import ToggleButton from 'primevue/togglebutton'
import AgentChip from '@/ui/primitives/AgentChip.vue'
import type { PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

const ACTION_TYPE_ICONS: Record<string, string> = {
  agent_run: 'pi pi-microchip-ai',
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

const isHuman = computed(() => props.step.action_type === 'human')
const isAgentRun = computed(() => props.step.action_type === 'agent_run')
const isTypeChipGate = computed(() => props.step.action_type === 'human' || props.step.action_type === 'hitl_gate')

/** Display label for agent: agent name if agent_id present, else legacy model string */
const agentDisplay = computed(() => {
  if (props.step.agent_id) {
    return props.agents.find((a) => a.id === props.step.agent_id)?.name ?? props.step.agent_id
  }
  return props.step.model ?? null
})

/** Full agent object for AgentChip display */
const selectedAgent = computed(() => {
  if (props.step.agent_id) {
    return props.agents.find((a) => a.id === props.step.agent_id) ?? null
  }
  return null
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
  <div
    class="flex flex-col gap-0 rounded-md"
    :class="{ 'amber-breathe': isHuman }"
    :style="isHuman ? 'background-color: var(--status-gate-surface)' : ''"
    data-testid="pipeline-step-card"
    @click="emit('toggle')"
  >
    <!-- Main row -->
    <div class="flex items-center gap-2 px-2 py-1.5 flex-wrap">
      <!-- Drag handle -->
      <span class="opacity-30 cursor-grab select-none text-sm">⠿</span>

      <!-- Index -->
      <span class="font-mono text-sm opacity-60">{{ index + 1 }}.</span>

      <!-- Step name -->
      <span class="flex-1 font-medium text-sm min-w-0">{{ step.name }}</span>

      <!-- Human gate description -->
      <em v-if="isHuman" class="text-xs opacity-70" style="color: var(--status-gate-color)">human stops the pipeline here</em>

      <!-- Agent selector (inline, agent_run only, admin) -->
      <template v-if="isAgentRun && isAdmin">
        <div class="flex items-center gap-1" @click.stop>
          <div v-if="selectedAgent" data-testid="agent-display">
            <AgentChip
              :role="selectedAgent.name"
              :model="selectedAgent.model"
              :provider="selectedAgent.provider ?? null"
            />
          </div>
          <span v-else-if="!props.step.agent_id && agentDisplay" class="text-xs opacity-60" data-testid="agent-display">{{ agentDisplay }}</span>
          <Select
            :model-value="step.agent_id"
            :options="agents"
            option-label="name"
            option-value="id"
            placeholder="Select agent"
            size="small"
            class="w-36"
            data-testid="agent-select"
            @update:model-value="onAgentChange"
          >
            <template #option="{ option }">
              <div class="flex flex-col">
                <span>{{ option.name }}</span>
                <span class="text-xs opacity-60">{{ option.model }}</span>
              </div>
            </template>
          </Select>
        </div>
      </template>
      <!-- agent display for non-admin (no Select) -->
      <template v-else-if="isAgentRun && agentDisplay">
        <div data-testid="agent-display">
          <AgentChip
            v-if="selectedAgent"
            :role="selectedAgent.name"
            :model="selectedAgent.model"
            :provider="selectedAgent.provider ?? null"
          />
          <span v-else class="text-xs opacity-60">{{ agentDisplay }}</span>
        </div>
      </template>

      <!-- Type chip (backward compat: both data-testids) -->
      <div data-testid="action-type-tag">
        <span
          class="font-mono text-xs px-1.5 py-0.5 rounded inline-flex items-center gap-1"
          :class="isTypeChipGate ? 'type-chip--gate' : 'type-chip--normal'"
          :style="isTypeChipGate
            ? 'background-color: var(--status-gate-surface); border: 1px solid var(--status-gate-color); color: var(--status-gate-color)'
            : 'background-color: var(--surface-overlay); border: 1px solid var(--surface-border)'"
          data-testid="step-type-chip"
        >
          <i :class="ACTION_TYPE_ICONS[step.action_type] ?? 'pi pi-cog'" class="text-xs" />
          {{ step.action_type }}
        </span>
      </div>

      <!-- Auto/Manual toggle -->
      <div @click.stop>
        <ToggleButton
          :model-value="step.auto_approve"
          on-label="Auto"
          off-label="Manual"
          size="small"
          :disabled="!isAdmin"
          data-testid="auto-approve-toggle"
          @update:model-value="onAutoApproveChange"
        />
      </div>

      <!-- Admin actions -->
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
    <div v-if="expanded" class="flex flex-col gap-4 px-4 py-3" @click.stop>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div v-if="isAgentRun" class="flex flex-col gap-2">
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
