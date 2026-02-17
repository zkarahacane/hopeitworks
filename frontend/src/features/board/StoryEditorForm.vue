<script setup lang="ts">
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import Button from 'primevue/button'
import Message from 'primevue/message'
import FloatLabel from 'primevue/floatlabel'
import type { UpdateStoryFields } from '@/stores/stories'

const props = defineProps<{
  modelValue: UpdateStoryFields
  errors: Record<string, string>
  apiError: string | null
  isSaving: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: UpdateStoryFields]
  save: []
  cancel: []
}>()

const scopeOptions = [
  { label: 'Backend', value: 'backend' },
  { label: 'Frontend', value: 'frontend' },
  { label: 'Shared', value: 'shared' },
]

function updateField<K extends keyof UpdateStoryFields>(key: K, value: UpdateStoryFields[K]) {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

function addFile() {
  emit('update:modelValue', {
    ...props.modelValue,
    target_files: [...(props.modelValue.target_files ?? []), ''],
  })
}

function removeFile(index: number) {
  const updated = [...(props.modelValue.target_files ?? [])]
  updated.splice(index, 1)
  emit('update:modelValue', { ...props.modelValue, target_files: updated })
}

function updateFile(index: number, value: string) {
  const updated = [...(props.modelValue.target_files ?? [])]
  updated[index] = value
  emit('update:modelValue', { ...props.modelValue, target_files: updated })
}
</script>

<template>
  <form class="flex flex-col gap-4" @submit.prevent="emit('save')">
    <Message v-if="apiError" severity="error" :closable="false">
      {{ apiError }}
    </Message>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText
          id="story-title"
          :model-value="modelValue.title"
          class="w-full"
          :invalid="!!errors.title"
          @update:model-value="updateField('title', $event as string)"
        />
        <label for="story-title">Title *</label>
      </FloatLabel>
      <small v-if="errors.title" class="text-red-500">{{ errors.title }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Textarea
          id="story-objective"
          :model-value="modelValue.objective ?? ''"
          class="w-full"
          rows="3"
          @update:model-value="updateField('objective', $event as string)"
        />
        <label for="story-objective">Objective</label>
      </FloatLabel>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Textarea
          id="story-acceptance-criteria"
          :model-value="modelValue.acceptance_criteria ?? ''"
          class="w-full"
          rows="4"
          @update:model-value="updateField('acceptance_criteria', $event as string)"
        />
        <label for="story-acceptance-criteria">Acceptance Criteria</label>
      </FloatLabel>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Select
          id="story-scope"
          :model-value="modelValue.scope"
          :options="scopeOptions"
          option-label="label"
          option-value="value"
          class="w-full"
          show-clear
          @update:model-value="updateField('scope', $event)"
        />
        <label for="story-scope">Scope</label>
      </FloatLabel>
    </div>

    <div class="flex flex-col gap-2">
      <div class="flex items-center justify-between">
        <h3
          class="m-0"
          style="
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
            color: var(--p-text-muted-color);
          "
        >
          Target Files
        </h3>
        <Button
          type="button"
          icon="pi pi-plus"
          text
          severity="secondary"
          size="small"
          aria-label="Add file"
          @click="addFile"
        />
      </div>
      <div
        v-for="(file, index) in modelValue.target_files ?? []"
        :key="index"
        class="flex items-center gap-2"
      >
        <InputText
          :model-value="file"
          class="flex-1"
          placeholder="path/to/file"
          style="font-family: monospace; font-size: 0.85rem"
          @update:model-value="updateFile(index, $event as string)"
        />
        <Button
          type="button"
          icon="pi pi-trash"
          text
          severity="danger"
          size="small"
          aria-label="Remove file"
          @click="removeFile(index)"
        />
      </div>
    </div>

    <div class="flex justify-end gap-2 pt-2">
      <Button
        type="button"
        label="Cancel"
        severity="secondary"
        text
        @click="emit('cancel')"
      />
      <Button
        type="submit"
        label="Save"
        :loading="isSaving"
      />
    </div>
  </form>
</template>
