<script setup lang="ts">
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import Message from 'primevue/message'
import ProgressSpinner from 'primevue/progressspinner'
import ConfirmDialog from 'primevue/confirmdialog'
import { useEnvironmentEditor } from '@/composables/useEnvironmentEditor'
import { useInFlightGuard } from '@/composables/useInFlightGuard'

const props = defineProps<{
  projectId: string
}>()

const toast = useToast()
const confirm = useConfirm()
// Guard the Delete button against double-click while the DELETE is in flight (#295).
const deleteGuard = useInFlightGuard()

const {
  stacks,
  source,
  services,
  commandsPairs,
  isLoading,
  isSaving,
  error,
  exists,
  canSave,
  addService,
  removeService,
  addEnvPair,
  removeEnvPair,
  addCommand,
  removeCommand,
  save,
  remove,
} = useEnvironmentEditor(props.projectId)

const STACK_OPTIONS = ['go', 'node', 'python', 'go-node']
const SOURCE_OPTIONS = [
  { label: 'devcontainer', value: 'devcontainer' },
  { label: 'compose', value: 'compose' },
  { label: 'makefile', value: 'makefile' },
  { label: 'declared', value: 'declared' },
]

async function handleSave() {
  const result = await save()
  if (result) {
    toast.add({ severity: 'success', summary: 'Environment saved', life: 3000 })
  } else {
    toast.add({ severity: 'error', summary: 'Failed to save environment', life: 4000 })
  }
}

function handleDelete() {
  if (deleteGuard.isBusy()) return
  confirm.require({
    message:
      'Delete the environment configuration for this project? This cannot be undone.',
    header: 'Delete Environment',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    // Guard the deletion body against double-click while the DELETE is in flight (#295).
    accept: () =>
      deleteGuard.run(async () => {
        const ok = await remove()
        if (ok) {
          toast.add({ severity: 'success', summary: 'Environment deleted', life: 3000 })
        } else {
          toast.add({ severity: 'error', summary: 'Failed to delete environment', life: 4000 })
        }
      }),
  })
}
</script>

<template>
  <ConfirmDialog />
  <div class="flex flex-col gap-6 p-6">
    <!-- Loading -->
    <div v-if="isLoading" class="flex items-center justify-center p-12">
      <ProgressSpinner />
    </div>

    <template v-else>
      <!-- API error -->
      <Message v-if="error" severity="error" :closable="false">
        {{ error }}
      </Message>

      <!-- Validation error (invalid services) -->
      <Message v-if="!canSave && services.length > 0" severity="warn" :closable="false">
        All services must have a non-empty image.
      </Message>

      <!-- Stacks -->
      <div class="flex flex-col gap-2">
        <label class="font-medium">Stacks</label>
        <MultiSelect
          v-model="stacks"
          :options="STACK_OPTIONS"
          placeholder="Select stacks"
          class="w-full"
        />
      </div>

      <!-- Source -->
      <div class="flex flex-col gap-2">
        <label class="font-medium">Source</label>
        <Select
          v-model="source"
          :options="SOURCE_OPTIONS"
          option-label="label"
          option-value="value"
          placeholder="Select source"
          class="w-full"
        />
      </div>

      <!-- Services -->
      <div class="flex flex-col gap-4">
        <div class="flex items-center justify-between">
          <label class="font-medium">Services (sidecars)</label>
          <Button label="Add service" icon="pi pi-plus" size="small" severity="secondary" @click="addService" />
        </div>

        <div
          v-for="(service, si) in services"
          :key="si"
          class="flex flex-col gap-3 p-4"
          :style="{ border: '1px solid var(--surface-border)', borderRadius: '6px' }"
        >
          <div class="flex items-center gap-2">
            <div class="flex flex-1 gap-2">
              <InputText v-model="service.name" placeholder="name" class="flex-1" />
              <InputText
                v-model="service.image"
                placeholder="image (required)"
                class="flex-1"
                :invalid="service.image.trim() === ''"
              />
            </div>
            <Button icon="pi pi-trash" severity="danger" text rounded @click="removeService(si)" />
          </div>

          <!-- Service env vars -->
          <div class="flex flex-col gap-2 pl-2">
            <div class="flex items-center justify-between">
              <span class="text-sm" :style="{ color: 'var(--p-text-muted-color)' }">Environment variables</span>
              <Button label="Add variable" icon="pi pi-plus" size="small" text @click="addEnvPair(si)" />
            </div>
            <div
              v-for="(pair, pi) in service.envPairs"
              :key="pi"
              class="flex items-center gap-2"
            >
              <InputText v-model="pair.key" placeholder="KEY" class="flex-1" />
              <InputText v-model="pair.value" placeholder="value" class="flex-1" />
              <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="removeEnvPair(si, pi)" />
            </div>
          </div>
        </div>

        <p
          v-if="services.length === 0"
          class="text-sm"
          :style="{ color: 'var(--p-text-muted-color)' }"
        >
          No sidecars configured. Add a service to inject sidecar containers alongside agent runs.
        </p>
      </div>

      <!-- Commands -->
      <div class="flex flex-col gap-4">
        <div class="flex items-center justify-between">
          <label class="font-medium">Commands</label>
          <Button label="Add command" icon="pi pi-plus" size="small" severity="secondary" @click="addCommand" />
        </div>

        <div
          v-for="(cmd, ci) in commandsPairs"
          :key="ci"
          class="flex items-center gap-2"
        >
          <InputText v-model="cmd.key" placeholder="name (e.g. build, test, migrate)" class="flex-1" />
          <InputText v-model="cmd.value" placeholder="command" class="flex-1" />
          <Button icon="pi pi-times" severity="secondary" text rounded size="small" @click="removeCommand(ci)" />
        </div>

        <p
          v-if="commandsPairs.length === 0"
          class="text-sm"
          :style="{ color: 'var(--p-text-muted-color)' }"
        >
          No commands defined. Typical keys: build, migrate, seed, test.
        </p>
      </div>

      <!-- Action buttons -->
      <div class="flex items-center gap-3">
        <Button
          label="Save"
          icon="pi pi-check"
          severity="success"
          :loading="isSaving"
          :disabled="!canSave"
          @click="handleSave"
        />
        <Button
          label="Delete"
          icon="pi pi-trash"
          severity="danger"
          :disabled="!exists || deleteGuard.isBusy()"
          :loading="deleteGuard.isBusy()"
          @click="handleDelete"
        />
      </div>
    </template>
  </div>
</template>
