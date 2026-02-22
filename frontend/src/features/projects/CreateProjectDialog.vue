<script setup lang="ts">
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
    repo_url: z
      .string()
      .min(1, 'Repository URL is required')
      .url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm, validate, setTouched } = useForm({
  validationSchema: createProjectSchema,
  initialValues: {
    git_provider: 'github',
    agent_runtime: 'docker',
  },
})

const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [repoUrl, repoUrlAttrs] = defineField('repo_url')
const [gitProvider, gitProviderAttrs] = defineField('git_provider')
const [agentRuntime, agentRuntimeAttrs] = defineField('agent_runtime')
const [defaultModel, defaultModelAttrs] = defineField('default_model')

const { createProject } = useProjects()

async function onSubmit() {
  // Mark all fields as touched so validation errors are shown
  setTouched({
    name: true,
    description: true,
    repo_url: true,
    git_provider: true,
    agent_runtime: true,
    default_model: true,
  })

  const { valid } = await validate()
  if (!valid) {
    return
  }

  await handleSubmit(async (values) => {
    const project = await createProject.execute({
      name: values.name,
      description: values.description || undefined,
      repo_url: values.repo_url,
      git_provider: values.git_provider,
      agent_runtime: values.agent_runtime,
      default_model: values.default_model || undefined,
    })
    if (project) {
      emit('created', project as Project)
      close()
    }
  })()
}

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
    class="w-full max-w-2xl"
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

      <div class="flex flex-col gap-1">
        <p class="text-sm font-semibold text-surface-600">Pipeline Configuration</p>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="project-repo-url"
            v-model="repoUrl"
            v-bind="repoUrlAttrs"
            class="w-full"
            :invalid="!!errors.repo_url"
          />
          <label for="project-repo-url">Repository URL *</label>
        </FloatLabel>
        <small v-if="errors.repo_url" class="text-red-500">{{ errors.repo_url }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Select
            id="project-git-provider"
            v-model="gitProvider"
            v-bind="gitProviderAttrs"
            :options="['github']"
            class="w-full"
            :invalid="!!errors.git_provider"
          />
          <label for="project-git-provider">Git Provider *</label>
        </FloatLabel>
        <small v-if="errors.git_provider" class="text-red-500">{{ errors.git_provider }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Select
            id="project-agent-runtime"
            v-model="agentRuntime"
            v-bind="agentRuntimeAttrs"
            :options="['docker']"
            class="w-full"
            :invalid="!!errors.agent_runtime"
          />
          <label for="project-agent-runtime">Agent Runtime *</label>
        </FloatLabel>
        <small v-if="errors.agent_runtime" class="text-red-500">{{ errors.agent_runtime }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="project-default-model"
            v-model="defaultModel"
            v-bind="defaultModelAttrs"
            class="w-full"
            placeholder="claude-opus-4-5"
          />
          <label for="project-default-model">Default Model</label>
        </FloatLabel>
        <small v-if="errors.default_model" class="text-red-500">{{ errors.default_model }}</small>
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
