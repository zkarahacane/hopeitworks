<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import { useAuthStore } from '@/stores/auth'
import { useNotifications, type NotificationConfig } from '@/composables/useNotifications'
import NotificationChannelRow from '@/features/notifications/NotificationChannelRow.vue'
import AddChannelDialog from '@/features/notifications/AddChannelDialog.vue'

const route = useRoute()
const toast = useToast()
const confirm = useConfirm()
const authStore = useAuthStore()

const projectId = route.params.id as string
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { configs, isLoading, toggleEnabled, deleteConfig, testConfig } =
  useNotifications(projectId)

const showAddDialog = ref(false)

async function handleToggle(config: NotificationConfig) {
  try {
    await toggleEnabled(config)
  } catch {
    toast.add({
      severity: 'error',
      summary: 'Failed to update channel',
      detail: 'Could not update the channel status. Please try again.',
      life: 4000,
    })
  }
}

function handleDelete(config: NotificationConfig) {
  confirm.require({
    message: `Delete ${config.channel_type} channel?`,
    header: 'Confirm Delete',
    icon: 'pi pi-trash',
    accept: async () => {
      await deleteConfig(config.id)
      toast.add({
        severity: 'success',
        summary: 'Channel deleted',
        life: 3000,
      })
    },
  })
}

async function handleTest(config: NotificationConfig) {
  try {
    await testConfig(config.id)
    toast.add({
      severity: 'success',
      summary: 'Test sent',
      life: 3000,
    })
  } catch (e) {
    const reason = e instanceof Error ? e.message : 'Unknown error'
    toast.add({
      severity: 'error',
      summary: `Test failed: ${reason}`,
      life: 4000,
    })
  }
}

function handleCreated() {
  toast.add({
    severity: 'success',
    summary: 'Channel added',
    life: 3000,
  })
}
</script>

<template>
  <div class="p-6">
    <!-- Page header -->
    <div class="mb-6 flex items-center justify-between">
      <div>
        <h2 class="text-xl font-semibold">Notification Channels</h2>
        <p class="mt-1 text-sm text-surface-500">
          Configure Discord or webhook alerts for pipeline events.
        </p>
      </div>
      <Button
        v-if="isAdmin"
        label="Add Channel"
        icon="pi pi-plus"
        @click="showAddDialog = true"
        data-testid="add-channel-btn"
      />
    </div>

    <!-- Loading skeleton -->
    <div v-if="isLoading" class="flex flex-col gap-3" data-testid="loading-skeleton">
      <Skeleton v-for="i in 3" :key="i" height="4rem" class="rounded-lg" />
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!isLoading && configs.length === 0"
      class="flex flex-col items-center justify-center rounded-lg border border-dashed border-surface-300 py-16 text-center"
      data-testid="empty-state"
    >
      <i class="pi pi-bell mb-4 text-4xl text-surface-300" />
      <p class="text-surface-500">No notification channels configured</p>
      <Button
        v-if="isAdmin"
        label="Add your first channel"
        text
        severity="secondary"
        class="mt-3"
        @click="showAddDialog = true"
      />
    </div>

    <!-- Channel list -->
    <div v-else class="flex flex-col gap-3" data-testid="channel-list">
      <NotificationChannelRow
        v-for="config in configs"
        :key="config.id"
        :config="config"
        :is-admin="isAdmin"
        @toggle="handleToggle"
        @delete="handleDelete"
        @test="handleTest"
      />
    </div>

    <!-- Add channel dialog -->
    <AddChannelDialog
      v-model:visible="showAddDialog"
      :project-id="projectId"
      @created="handleCreated"
    />
  </div>
</template>
