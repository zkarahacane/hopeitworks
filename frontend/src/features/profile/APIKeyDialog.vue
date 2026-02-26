<script setup lang="ts">
import { ref } from 'vue'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAPIKeys } from '@/composables/useAPIKeys'

interface Props {
  visible: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  created: []
}>()

const { createKey, error } = useAPIKeys()

const providerOptions = [
  { label: 'Claude', value: 'claude' },
  { label: 'OpenCode', value: 'opencode' },
]

const provider = ref('claude')
const keyName = ref('')
const apiKey = ref('')
const isSaving = ref(false)
const localError = ref<string | null>(null)

function resetForm() {
  provider.value = 'claude'
  keyName.value = ''
  apiKey.value = ''
  localError.value = null
}

function handleClose() {
  resetForm()
  emit('update:visible', false)
}

async function handleSubmit() {
  if (!provider.value || !keyName.value.trim() || !apiKey.value.trim()) {
    localError.value = 'All fields are required'
    return
  }
  isSaving.value = true
  localError.value = null
  const success = await createKey(provider.value, keyName.value.trim(), apiKey.value.trim())
  isSaving.value = false
  if (success) {
    emit('created')
    handleClose()
  } else {
    localError.value = error.value ?? 'Failed to create API key'
  }
}
</script>

<template>
  <Dialog
    :visible="visible"
    header="Add API Key"
    :modal="true"
    :closable="true"
    :style="{ width: '28rem' }"
    @update:visible="handleClose"
  >
    <div class="flex flex-col gap-4">
      <Message v-if="localError" severity="error" :closable="false">
        {{ localError }}
      </Message>

      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium">Provider</label>
        <Select
          v-model="provider"
          :options="providerOptions"
          option-label="label"
          option-value="value"
          class="w-full"
          placeholder="Select provider"
        />
      </div>

      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium">Key Name</label>
        <InputText
          v-model="keyName"
          placeholder="e.g. default, work"
          class="w-full"
        />
      </div>

      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium">API Key</label>
        <InputText
          v-model="apiKey"
          type="password"
          placeholder="sk-ant-..."
          class="w-full"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="handleClose" />
        <Button
          label="Save"
          :loading="isSaving"
          :disabled="isSaving"
          @click="handleSubmit"
        />
      </div>
    </template>
  </Dialog>
</template>
