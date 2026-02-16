<script setup lang="ts">
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useProjects } from '@/composables/useProjects'
import type { Project } from '@/stores/projects'

defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  created: [project: Project]
}>()

const createProjectSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or fewer'),
    description: z
      .string()
      .max(1000, 'Description must be 1000 characters or fewer')
      .optional()
      .or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: createProjectSchema,
})

const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')

const { createProject } = useProjects()

const onSubmit = handleSubmit(async (values) => {
  const project = await createProject.execute({
    name: values.name,
    description: values.description || undefined,
  })
  if (project) {
    emit('created', project as Project)
    close()
  }
})

function close() {
  resetForm()
  createProject.error.value = null
  emit('update:visible', false)
}
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    header="Create Project"
    class="w-full max-w-lg"
    @update:visible="close"
  >
    <form class="flex flex-col gap-6" @submit.prevent="onSubmit">
      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="project-name"
            v-model="name"
            v-bind="nameAttrs"
            class="w-full"
            :invalid="!!errors.name"
          />
          <label for="project-name">Name *</label>
        </FloatLabel>
        <small v-if="errors.name" class="text-red-500">{{ errors.name }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Textarea
            id="project-description"
            v-model="description"
            v-bind="descriptionAttrs"
            class="w-full"
            rows="3"
            :invalid="!!errors.description"
          />
          <label for="project-description">Description</label>
        </FloatLabel>
        <small v-if="errors.description" class="text-red-500">{{ errors.description }}</small>
      </div>

      <Message v-if="createProject.error.value" severity="error" :closable="false">
        {{ createProject.error.value.message }}
      </Message>
    </form>

    <template #footer>
      <div class="flex justify-end gap-2">
        <Button label="Cancel" severity="secondary" text @click="close" />
        <Button
          label="Create"
          severity="success"
          icon="pi pi-check"
          :loading="createProject.isLoading.value"
          @click="onSubmit"
        />
      </div>
    </template>
  </Dialog>
</template>
