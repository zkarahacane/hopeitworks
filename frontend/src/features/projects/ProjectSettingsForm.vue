<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import FloatLabel from 'primevue/floatlabel'
import Button from 'primevue/button'
import { useAuthStore } from '@/stores/auth'
import ProjectDeleteDialog from './ProjectDeleteDialog.vue'
import type { Project, UpdateProjectPayload } from '@/stores/projects'

const props = defineProps<{
  project: Project
  isSaving: boolean
  isDeleting: boolean
}>()

const emit = defineEmits<{
  save: [payload: UpdateProjectPayload]
  delete: []
}>()

const authStore = useAuthStore()
const deleteDialogVisible = ref(false)

// Admin-only danger zone — mirrors the gating used elsewhere (AppSidebar).
// The backend enforces admin via requireAdmin (403); the UI never exposes it.
const isAdmin = computed(() => authStore.user?.role === 'admin')

const projectSettingsSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(1, 'Project name is required')
      .max(255, 'Name must be 255 characters or less'),
    description: z.string().max(1000, 'Description must be 1000 characters or less').default(''),
    repo_url: z.string().min(1, 'Repository URL is required').url('Must be a valid URL'),
    git_provider: z.enum(['github', 'gitea']).default('github'),
    git_token_env: z.string().optional().or(z.literal('')),
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
    git_provider: (props.project.git_provider as 'github' | 'gitea') ?? 'github',
    git_token_env: props.project.git_token_env ?? '',
    agent_runtime: (props.project.agent_runtime as 'docker') ?? 'docker',
    default_model: props.project.default_model ?? '',
  },
})

const [name, nameAttrs] = defineField('name')
const [description, descriptionAttrs] = defineField('description')
const [repoUrl, repoUrlAttrs] = defineField('repo_url')
const [gitProvider, gitProviderAttrs] = defineField('git_provider')
const [gitTokenEnv, gitTokenEnvAttrs] = defineField('git_token_env')
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
        git_provider: (proj.git_provider as 'github' | 'gitea') ?? 'github',
        git_token_env: proj.git_token_env ?? '',
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
    git_token_env: values.git_token_env || undefined,
    agent_runtime: values.agent_runtime,
    default_model: values.default_model || undefined,
  })
})
</script>

<template>
  <div class="flex flex-col gap-8">
    <form
      class="flex flex-col gap-6"
      data-testid="project-settings-form"
      @submit.prevent="onSubmit"
    >
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
        <small v-if="errors.name" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.name
        }}</small>
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
        <small v-if="errors.description" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.description
        }}</small>
      </div>

      <div class="flex flex-col gap-1">
        <p class="text-sm font-semibold" :style="{ color: 'var(--p-text-muted-color)' }">
          Pipeline Configuration
        </p>
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
        <small v-if="errors.repo_url" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.repo_url
        }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <Select
            id="settings-git-provider"
            v-model="gitProvider"
            v-bind="gitProviderAttrs"
            :options="['github', 'gitea']"
            class="w-full"
            :invalid="!!errors.git_provider"
          />
          <label for="settings-git-provider">Git Provider *</label>
        </FloatLabel>
        <small v-if="errors.git_provider" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.git_provider
        }}</small>
      </div>

      <div class="flex flex-col gap-2">
        <FloatLabel>
          <InputText
            id="settings-git-token-env"
            v-model="gitTokenEnv"
            v-bind="gitTokenEnvAttrs"
            class="w-full"
            placeholder="GITEA_TOKEN"
          />
          <label for="settings-git-token-env">Git Token Env Var</label>
        </FloatLabel>
        <small :style="{ color: 'var(--p-text-muted-color)' }"
          >Name of the environment variable holding the git token (defaults to GITHUB_TOKEN)</small
        >
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
        <small v-if="errors.agent_runtime" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.agent_runtime
        }}</small>
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
        <small v-if="errors.default_model" :style="{ color: 'var(--status-failed-color)' }">{{
          errors.default_model
        }}</small>
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

    <section
      v-if="isAdmin"
      class="flex flex-col gap-3"
      data-testid="project-danger-zone"
      aria-label="Danger zone"
    >
      <h3 class="text-sm font-semibold" :style="{ color: 'var(--status-failed-color)' }">
        Danger zone
      </h3>
      <div class="flex items-center justify-between gap-4">
        <p class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          Permanently delete this project and all of its runs, stories, epics, and configs. This
          cannot be undone.
        </p>
        <Button
          label="Delete project"
          severity="danger"
          icon="pi pi-trash"
          type="button"
          data-testid="open-delete-dialog-btn"
          @click="deleteDialogVisible = true"
        />
      </div>
    </section>

    <ProjectDeleteDialog
      v-if="isAdmin"
      v-model:visible="deleteDialogVisible"
      :project-name="project.name"
      :loading="isDeleting"
      @confirm="emit('delete')"
    />
  </div>
</template>
