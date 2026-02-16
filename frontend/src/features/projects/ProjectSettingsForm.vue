<script setup lang="ts">
import { watch } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import FloatLabel from 'primevue/floatlabel'
import Message from 'primevue/message'
import type { Project } from '@/stores/projects'

const props = defineProps<{
  project: Project
  isSaving: boolean
}>()

const emit = defineEmits<{
  save: [payload: { name: string; description: string }]
}>()

const schema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or less'),
    description: z
      .string()
      .max(1000, 'Description must be 1000 characters or less')
      .default(''),
  }),
)

const { handleSubmit, resetForm, meta } = useForm({
  validationSchema: schema,
  initialValues: {
    name: props.project.name,
    description: props.project.description ?? '',
  },
})

const { value: name, errorMessage: nameError } = useField<string>('name')
const { value: description, errorMessage: descriptionError } = useField<string>('description')

watch(
  () => props.project,
  (newProject) => {
    resetForm({
      values: {
        name: newProject.name,
        description: newProject.description ?? '',
      },
    })
  },
)

const onSubmit = handleSubmit((values) => {
  emit('save', { name: values.name, description: values.description })
})
</script>

<template>
  <form class="flex flex-col gap-6 max-w-xl" @submit.prevent="onSubmit">
    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText id="name" v-model="name" class="w-full" />
        <label for="name">Project Name</label>
      </FloatLabel>
      <small v-if="nameError" class="text-red-500">{{ nameError }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Textarea id="description" v-model="description" rows="4" class="w-full" />
        <label for="description">Description</label>
      </FloatLabel>
      <small v-if="descriptionError" class="text-red-500">{{ descriptionError }}</small>
    </div>

    <div class="flex justify-end">
      <Button
        type="submit"
        label="Save"
        severity="success"
        :loading="isSaving"
        :disabled="!meta.dirty || !meta.valid"
      />
    </div>

    <Message severity="info" :closable="false">
      Git, Agent, and Budget settings will be available in a future release.
    </Message>
  </form>
</template>
