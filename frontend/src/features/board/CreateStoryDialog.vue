<script setup lang="ts">
import { ref } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useStoriesStore, type Story } from '@/stores/stories'

const props = defineProps<{
  visible: boolean
  projectId: string
  epicId: string
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  created: [story: Story]
}>()

const createStorySchema = toTypedSchema(
  z.object({
    key: z.string().min(1, 'Key is required').max(50, 'Key must be 50 characters or fewer'),
    title: z.string().min(1, 'Title is required').max(255, 'Title must be 255 characters or fewer'),
    objective: z.string().optional().or(z.literal('')),
    acceptance_criteria: z.string().optional().or(z.literal('')),
    scope: z.enum(['backend', 'frontend', 'shared']).optional(),
  }),
)

const { defineField, handleSubmit, errors, resetForm, validate } = useForm({
  validationSchema: createStorySchema,
})

const [key, keyAttrs] = defineField('key')
const [title, titleAttrs] = defineField('title')
const [objective, objectiveAttrs] = defineField('objective')
const [acceptanceCriteria, acceptanceCriteriaAttrs] = defineField('acceptance_criteria')
const [scope, scopeAttrs] = defineField('scope')

const scopeOptions = [
  { label: 'Backend', value: 'backend' },
  { label: 'Frontend', value: 'frontend' },
  { label: 'Shared', value: 'shared' },
]

const targetFiles = ref<string[]>([])
const store = useStoriesStore()
const apiError = ref<string | null>(null)
const isSaving = ref(false)

function addFile() {
  targetFiles.value.push('')
}

function removeFile(index: number) {
  targetFiles.value.splice(index, 1)
}

async function onSubmit() {
  const { valid } = await validate()
  if (!valid) return

  await handleSubmit(async (values) => {
    isSaving.value = true
    apiError.value = null
    try {
      const files = targetFiles.value.filter((f) => f.trim() !== '')
      const result = await store.createStory(props.projectId, {
        key: values.key,
        title: values.title,
        objective: values.objective || undefined,
        acceptance_criteria: values.acceptance_criteria || undefined,
        scope: values.scope as 'backend' | 'frontend' | 'shared' | undefined,
        epic_id: props.epicId,
        target_files: files.length > 0 ? files : undefined,
      })
      if (result) {
        emit('created', result)
        close()
      } else {
        apiError.value = store.error ?? 'Failed to create story'
      }
    } finally {
      isSaving.value = false
    }
  })()
}

function close() {
  resetForm()
  targetFiles.value = []
  apiError.value = null
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Create Story"
    class="w-full max-w-lg"
    @update:visible="close"
  >
    <form class="flex flex-col gap-6" @submit.prevent="onSubmit">
      <Message v-if="apiError" severity="error" :closable="false">
        {{ apiError }}
      </Message>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="story-key"
            v-model="key"
            v-bind="keyAttrs"
            class="w-full"
            :invalid="!!errors.key"
          />
          <label for="story-key">Key *</label>
        </FloatLabel>
        <small v-if="errors.key" class="text-red-500">{{ errors.key }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="story-title"
            v-model="title"
            v-bind="titleAttrs"
            class="w-full"
            :invalid="!!errors.title"
          />
          <label for="story-title">Title *</label>
        </FloatLabel>
        <small v-if="errors.title" class="text-red-500">{{ errors.title }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Textarea
            id="story-objective"
            v-model="objective"
            v-bind="objectiveAttrs"
            class="w-full"
            rows="3"
          />
          <label for="story-objective">Objective</label>
        </FloatLabel>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Textarea
            id="story-acceptance-criteria"
            v-model="acceptanceCriteria"
            v-bind="acceptanceCriteriaAttrs"
            class="w-full"
            rows="4"
          />
          <label for="story-acceptance-criteria">Acceptance Criteria</label>
        </FloatLabel>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Select
            id="story-scope"
            v-model="scope"
            v-bind="scopeAttrs"
            :options="scopeOptions"
            option-label="label"
            option-value="value"
            class="w-full"
            show-clear
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
          v-for="(file, index) in targetFiles"
          :key="index"
          class="flex items-center gap-2"
        >
          <InputText
            :model-value="file"
            class="flex-1"
            placeholder="path/to/file"
            style="font-family: monospace; font-size: 0.85rem"
            @update:model-value="targetFiles[index] = $event as string"
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
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="close" />
        <Button
          label="Create"
          severity="success"
          icon="pi pi-check"
          :loading="isSaving"
          @click="onSubmit"
        />
      </div>
    </template>
  </Dialog>
</template>
