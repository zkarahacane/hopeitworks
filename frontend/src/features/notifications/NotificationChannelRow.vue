<script setup lang="ts">
import Tag from 'primevue/tag'
import Chip from 'primevue/chip'
import ToggleSwitch from 'primevue/toggleswitch'
import Button from 'primevue/button'
import type { NotificationConfig } from '@/composables/useNotifications'
import { maskUrl } from '@/utils/maskUrl'

defineProps<{
  config: NotificationConfig
  isAdmin: boolean
}>()

const emit = defineEmits<{
  toggle: [config: NotificationConfig]
  edit: [config: NotificationConfig]
  delete: [config: NotificationConfig]
  test: [config: NotificationConfig]
}>()

const channelTagSeverity: Record<string, 'info' | 'secondary'> = {
  discord: 'info',
  webhook: 'secondary',
}
</script>

<template>
  <div
    class="flex items-center gap-4 rounded-lg border border-surface-200 bg-surface-0 p-4"
    data-testid="notification-channel-row"
  >
    <!-- Channel type badge -->
    <Tag
      :value="config.channel_type"
      :severity="channelTagSeverity[config.channel_type] ?? 'secondary'"
      class="capitalize"
      data-testid="channel-type-badge"
    />

    <!-- Masked URL -->
    <span
      class="flex-1 font-mono text-sm text-surface-600"
      data-testid="masked-url"
    >
      {{ maskUrl(config.config.url) }}
    </span>

    <!-- Events filter chips -->
    <div class="flex flex-wrap gap-1" data-testid="events-chips">
      <Chip
        v-for="event in config.events_filter"
        :key="event"
        :label="event"
        class="text-xs"
      />
      <span v-if="config.events_filter.length === 0" class="text-xs text-surface-400">
        No events
      </span>
    </div>

    <!-- Enabled toggle -->
    <ToggleSwitch
      :model-value="config.enabled"
      :disabled="!isAdmin"
      data-testid="enabled-toggle"
      @update:model-value="emit('toggle', config)"
    />

    <!-- Admin action buttons -->
    <div v-if="isAdmin" class="flex gap-1" data-testid="admin-actions">
      <Button
        icon="pi pi-send"
        text
        rounded
        severity="secondary"
        size="small"
        aria-label="Test channel"
        data-testid="test-btn"
        @click="emit('test', config)"
      />
      <Button
        icon="pi pi-pencil"
        text
        rounded
        severity="secondary"
        size="small"
        aria-label="Edit channel"
        data-testid="edit-btn"
        @click="emit('edit', config)"
      />
      <Button
        icon="pi pi-trash"
        text
        rounded
        severity="danger"
        size="small"
        aria-label="Delete channel"
        data-testid="delete-btn"
        @click="emit('delete', config)"
      />
    </div>
  </div>
</template>
