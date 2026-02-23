<script setup lang="ts">
import { ref, reactive, watch } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import ToggleSwitch from 'primevue/toggleswitch'
import InputNumber from 'primevue/inputnumber'
import Button from 'primevue/button'
import Message from 'primevue/message'
import type { PipelineStep } from '@/stores/pipelineConfig'
import type { Agent } from '@/stores/agents'

const actionTypeOptions = [
  { label: 'Agent Run', value: 'agent_run' },
  { label: 'Create Git Branch', value: 'git_branch' },
  { label: 'Create Pull Request', value: 'git_pr' },
  { label: 'Send Notification', value: 'notification' },
  { label: 'Human Task', value: 'human' },
  { label: 'Poll CI Status', value: 'ci_poll' },
  { label: 'HITL Gate', value: 'hitl_gate' },
]

const retryTypeOptions = [
  { label: 'None', value: 'none' },
  { label: 'On Failure', value: 'on-failure' },
  { label: 'Always', value: 'always' },
]

const props = defineProps<{
  visible: boolean
  agents: Agent[]
}>()

const emit = defineEmits<{
  add: [step: PipelineStep]
  cancel: []
  'update:visible': [value: boolean]
}>()

const name = ref('')
const actionType = ref<PipelineStep['action_type']>('agent_run')
const agentId = ref<string | undefined>(undefined)
const autoApprove = ref(false)
const maxRetries = ref(2)
const retryType = ref<PipelineStep['retry_policy']['retry_type']>('on-failure')
const validationError = ref<string | null>(null)
const config = reactive<Record<string, string>>({})
const configDraft = ref(false)

function resetForm() {
  name.value = ''
  actionType.value = 'agent_run'
  agentId.value = undefined
  autoApprove.value = false
  maxRetries.value = 2
  retryType.value = 'on-failure'
  validationError.value = null
  resetConfig()
}

function resetConfig() {
  Object.keys(config).forEach((key) => delete config[key])
  configDraft.value = false
}

watch(
  () => props.visible,
  (isVisible) => {
    if (isVisible) {
      resetForm()
    }
  },
)

watch(
  () => actionType.value,
  () => {
    resetConfig()
    if (actionType.value !== 'agent_run') {
      agentId.value = undefined
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
    agent_id: actionType.value === 'agent_run' ? agentId.value : undefined,
    auto_approve: autoApprove.value,
    retry_policy: {
      max_retries: maxRetries.value,
      retry_type: retryType.value,
    },
    config: Object.keys(config).length > 0 || (actionType.value === 'git_pr' && configDraft.value)
      ? { ...config, ...(actionType.value === 'git_pr' ? { draft: String(configDraft.value) } : {}) }
      : undefined,
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

      <!-- Agent selector: only visible for agent_run -->
      <div v-if="actionType === 'agent_run'" class="flex flex-col gap-2">
        <label for="agent-select" class="text-sm font-medium">Agent</label>
        <Select
          id="agent-select"
          v-model="agentId"
          :options="agents"
          option-label="name"
          option-value="id"
          placeholder="Select an agent"
          class="w-full"
          data-testid="agent-select"
        >
          <template #option="{ option }">
            <div class="flex flex-col">
              <span>{{ option.name }}</span>
              <span class="text-sm opacity-60">{{ option.model }}</span>
            </div>
          </template>
        </Select>
      </div>

      <!-- git_branch config fields -->
      <template v-if="actionType === 'git_branch'">
        <div class="flex flex-col gap-2">
          <label for="branch-pattern" class="text-sm font-medium">Branch Pattern</label>
          <InputText
            id="branch-pattern"
            v-model="config.branch_pattern"
            class="w-full"
            placeholder="e.g., feat/{story_key}-{slug}"
            data-testid="branch-pattern-input"
          />
        </div>
      </template>

      <!-- git_pr config fields -->
      <template v-if="actionType === 'git_pr'">
        <div class="flex flex-col gap-2">
          <label for="title-template" class="text-sm font-medium">PR Title Template</label>
          <InputText
            id="title-template"
            v-model="config.title_template"
            class="w-full"
            placeholder="e.g., feat({scope}): {summary}"
            data-testid="title-template-input"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label for="target-branch" class="text-sm font-medium">Target Branch</label>
          <InputText
            id="target-branch"
            v-model="config.target_branch"
            class="w-full"
            placeholder="e.g., develop"
            data-testid="target-branch-input"
          />
        </div>
        <div class="flex items-center gap-2">
          <ToggleSwitch
            v-model="configDraft"
            input-id="draft-toggle"
            data-testid="draft-toggle"
          />
          <label for="draft-toggle" class="text-sm font-medium">Draft PR</label>
        </div>
      </template>

      <!-- notification config fields -->
      <template v-if="actionType === 'notification'">
        <div class="flex flex-col gap-2">
          <label for="notification-message" class="text-sm font-medium">Message</label>
          <Textarea
            id="notification-message"
            v-model="config.message"
            class="w-full"
            rows="3"
            placeholder="Notification message"
            data-testid="notification-message-input"
          />
        </div>
      </template>

      <!-- human config fields -->
      <template v-if="actionType === 'human'">
        <div class="flex flex-col gap-2">
          <label for="human-message" class="text-sm font-medium">Message</label>
          <Textarea
            id="human-message"
            v-model="config.message"
            class="w-full"
            rows="3"
            placeholder="What to display to the human reviewer"
            data-testid="human-message-input"
          />
        </div>
        <div class="flex flex-col gap-2">
          <label for="human-instructions" class="text-sm font-medium">Instructions</label>
          <Textarea
            id="human-instructions"
            v-model="config.instructions"
            class="w-full"
            rows="3"
            placeholder="Optional detailed instructions"
            data-testid="human-instructions-input"
          />
        </div>
      </template>

      <!-- ci_poll: no config fields -->
      <Message v-if="actionType === 'ci_poll'" severity="info" :closable="false" data-testid="ci-poll-info">
        No additional configuration required
      </Message>

      <!-- hitl_gate: no config fields -->
      <Message v-if="actionType === 'hitl_gate'" severity="info" :closable="false" data-testid="hitl-gate-info">
        No additional configuration required
      </Message>

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
