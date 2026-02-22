<script setup lang="ts">
import { watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import type { Project, UpdateProjectPayload } from '@/stores/projects'

const props = defineProps<{
  project: Project
  isSaving: boolean
}>()

const emit = defineEmits<{
  save: [payload: UpdateProjectPayload]
}>()

const projectSettingsSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or less'),
    description: z
      .string()
      .max(1000, 'Description must be 1000 characters or less')
      .default(''),
    repo_url: z
      .string()
      .min(1, 'Repository URL is required')
      .url('Must be a valid URL'),
    git_provider: z.enum(['github']).default('github'),
    agent_runtime: z.enum(['docker']).default('docker'),
    default_model: z.string().optional().or(z.literal('')),
  }),
)

const { defineField, handleSubmit, errors, resetForm } = useForm({
  validationSchema: projectSettingsSchema,
  initialValues: {
    name: props.project.name,
    description: props.project.description ?? '',
    repo_url: props.project.repo_url ?? '',
    git_provider: (props.project.git_provider as 'github') ?? 'github',
    agent_runtime: (props.project.agent_runtime as 'docker') ?? 'docker',
    default_model: props.project.default_model ?? '',
  },
})

const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [repoUrl, repoUrlAttrs] = defineField('repo_url')
const [gitProvider, gitProviderAttrs] = defineField('git_provider')
const [agentRuntime, agentRuntimeAttrs] = defineField('agent_runtime')
const [defaultModel, defaultModelAttrs] = defineField('default_model')

watch(
  () => props.project,
  (proj) => {
    resetForm({
      values: {
        name: proj.name,
        description: proj.description ?? '',
        repo_url: proj.repo_url ?? '',
        git_provider: (proj.git_provider as 'github') ?? 'github',
        agent_runtime: (proj.agent_runtime as 'docker') ?? 'docker',
        default_model: proj.default_model ?? '',
      },
    })
  },
)

const onSubmit = handleSubmit((values) => {
  emit('save', {
    name: values.name,
    description: values.description || undefined,
    repo_url: values.repo_url,
    git_provider: values.git_provider,
    agent_runtime: values.agent_runtime,
    default_model: values.default_model || undefined,
  })
})
</script>

<template>
  <form class="flex flex-col gap-6" data-testid="project-settings-form" @submit.prevent="onSubmit">
    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText
          id="settings-name"
          v-model="name"
          v-bind="nameAttrs"
          class="w-full"
          :invalid="!!errors.name"
        />
        <label for="settings-name">Name *</label>
      </FloatLabel>
      <small v-if="errors.name" class="text-red-500">{{ errors.name }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Textarea
          id="settings-description"
          v-model="description"
          v-bind="descriptionAttrs"
          class="w-full"
          rows="3"
          :invalid="!!errors.description"
        />
        <label for="settings-description">Description</label>
      </FloatLabel>
      <small v-if="errors.description" class="text-red-500">{{ errors.description }}</small>
    </div>

    <div class="flex flex-col gap-1">
      <p class="text-sm font-semibold text-surface-600">Pipeline Configuration</p>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText
          id="settings-repo-url"
          v-model="repoUrl"
          v-bind="repoUrlAttrs"
          class="w-full"
          :invalid="!!errors.repo_url"
        />
        <label for="settings-repo-url">Repository URL *</label>
      </FloatLabel>
      <small v-if="errors.repo_url" class="text-red-500">{{ errors.repo_url }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Select
          id="settings-git-provider"
          v-model="gitProvider"
          v-bind="gitProviderAttrs"
          :options="['github']"
          class="w-full"
          :invalid="!!errors.git_provider"
        />
        <label for="settings-git-provider">Git Provider *</label>
      </FloatLabel>
      <small v-if="errors.git_provider" class="text-red-500">{{ errors.git_provider }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <Select
          id="settings-agent-runtime"
          v-model="agentRuntime"
          v-bind="agentRuntimeAttrs"
          :options="['docker']"
          class="w-full"
          :invalid="!!errors.agent_runtime"
        />
        <label for="settings-agent-runtime">Agent Runtime *</label>
      </FloatLabel>
      <small v-if="errors.agent_runtime" class="text-red-500">{{ errors.agent_runtime }}</small>
    </div>

    <div class="flex flex-col gap-2">
      <FloatLabel>
        <InputText
          id="settings-default-model"
          v-model="defaultModel"
          v-bind="defaultModelAttrs"
          class="w-full"
          placeholder="claude-opus-4-5"
        />
        <label for="settings-default-model">Default Model</label>
      </FloatLabel>
      <small v-if="errors.default_model" class="text-red-500">{{ errors.default_model }}</small>
    </div>

    <div class="flex justify-end">
      <Button
        label="Save"
        severity="success"
        icon="pi pi-save"
        type="submit"
        :loading="isSaving"
        data-testid="save-settings-btn"
      />
    </div>
  </form>
</template>
