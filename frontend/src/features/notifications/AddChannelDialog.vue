<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import MultiSelect from 'primevue/multiselect'
import ToggleSwitch from 'primevue/toggleswitch'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useNotifications, type NotificationConfig } from '@/composables/useNotifications'

const addChannelSchema = toTypedSchema(
  z.object({
    channel_type: z.enum(['discord', 'webhook']),
    url: z
      .string()
      .min(1, 'URL is required')
      .startsWith('https://', 'URL must start with https://'),
    events_filter: z.array(z.string()).default([]),
    enabled: z.boolean().default(true),
  }),
)

const props = defineProps<{
  visible: boolean
  projectId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  created: [config: NotificationConfig]
}>()

const channelTypeOptions = [
  { label: 'Discord', value: 'discord' },
  { label: 'Webhook', value: 'webhook' },
]

const EVENT_OPTIONS = [
  { label: 'Run Completed', value: 'run.completed' },
  { label: 'Run Failed', value: 'run.failed' },
  { label: 'Approval Pending', value: 'hitl_gate.pending' },
  { label: 'Circuit Breaker Triggered', value: 'circuit_breaker.triggered' },
]

const { defineField, handleSubmit, errors, resetForm, validate } = useForm({
  validationSchema: addChannelSchema,
  initialValues: {
    enabled: true,
    events_filter: [],
  },
})

const [channelType, channelTypeAttrs] = defineField('channel_type')
const [url, urlAttrs] = defineField('url')
const [eventsFilter, eventsFilterAttrs] = defineField('events_filter')
const [enabled, enabledAttrs] = defineField('enabled')

const apiError = ref<string | null>(null)
const isSaving = ref(false)

const notifs = useNotifications(props.projectId)

async function onSubmit() {
  const { valid } = await validate()
  if (!valid) return

  await handleSubmit(async (values) => {
    isSaving.value = true
    apiError.value = null
    try {
      const result = await notifs.createConfig({
        channel_type: values.channel_type,
        config: { url: values.url },
        events_filter: values.events_filter ?? [],
        enabled: values.enabled ?? true,
      })
      if (result) {
        emit('created', result)
        close()
      } else {
        apiError.value = 'Failed to add channel'
      }
    } finally {
      isSaving.value = false
    }
  })()
}

function close() {
  resetForm()
  apiError.value = null
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Add Notification Channel"
    class="w-full max-w-lg"
    @update:visible="close"
  >
    <form class="flex flex-col gap-6" @submit.prevent="onSubmit">
      <Message v-if="apiError" severity="error" :closable="false">
        {{ apiError }}
      </Message>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Select
            id="channel-type"
            v-model="channelType"
            v-bind="channelTypeAttrs"
            :options="channelTypeOptions"
            option-label="label"
            option-value="value"
            class="w-full"
            :invalid="!!errors.channel_type"
          />
          <label for="channel-type">Channel Type *</label>
        </FloatLabel>
        <small v-if="errors.channel_type" :style="{ color: 'var(--status-failed-color)' }">{{ errors.channel_type }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="webhook-url"
            v-model="url"
            v-bind="urlAttrs"
            class="w-full"
            :invalid="!!errors.url"
          />
          <label for="webhook-url">Webhook URL *</label>
        </FloatLabel>
        <small v-if="errors.url" :style="{ color: 'var(--status-failed-color)' }">{{ errors.url }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <MultiSelect
            id="events-filter"
            v-model="eventsFilter"
            v-bind="eventsFilterAttrs"
            :options="EVENT_OPTIONS"
            option-label="label"
            option-value="value"
            class="w-full"
            placeholder="Select events"
          />
          <label for="events-filter">Events</label>
        </FloatLabel>
      </div>

      <div class="flex items-center gap-3">
        <ToggleSwitch
          id="channel-enabled"
          v-model="enabled"
          v-bind="enabledAttrs"
        />
        <label for="channel-enabled" class="text-sm">Enabled</label>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="close" />
        <Button
          label="Add Channel"
          severity="success"
          icon="pi pi-plus"
          :loading="isSaving"
          @click="onSubmit"
        />
      </div>
    </template>
  </Dialog>
</template>
