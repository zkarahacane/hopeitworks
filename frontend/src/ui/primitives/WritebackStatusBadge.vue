<script setup lang="ts">
import { computed } from 'vue'
import Tag from 'primevue/tag'

/**
 * WritebackStatusBadge — canonical badge for the tracker write-back state of a story.
 *
 * Maps the `writeback_status` field (disabled | pending | synced | failed | null)
 * to a PrimeVue Tag with an appropriate severity, label and icon.
 *
 * Rendering rules:
 *  - null / undefined → renders nothing (manual stories with no connector configured)
 *  - 'disabled'       → renders nothing (not actionable, avoids badge noise)
 *  - 'pending'        → info tag  (write-back queued/in-flight)
 *  - 'synced'         → success   (tracker up-to-date)
 *  - 'failed'         → danger    (last write-back errored)
 *
 * Dumb + prop-driven; no data access.
 */
const props = defineProps<{
  status?: 'disabled' | 'pending' | 'synced' | 'failed' | null
  /**
   * When true, also render `disabled` as a secondary badge (e.g. in detail panel
   * where the user may want to understand why write-back is off).
   */
  showDisabled?: boolean
}>()

interface WritebackMeta {
  label: string
  icon: string
  severity: 'success' | 'info' | 'warn' | 'danger' | 'secondary'
}

const meta = computed<WritebackMeta | null>(() => {
  switch (props.status) {
    case 'synced':
      return { label: 'Synced', icon: 'pi pi-check-circle', severity: 'success' }
    case 'pending':
      return { label: 'Sync pending', icon: 'pi pi-spin pi-spinner', severity: 'info' }
    case 'failed':
      return { label: 'Sync failed', icon: 'pi pi-times-circle', severity: 'danger' }
    case 'disabled':
      return props.showDisabled
        ? { label: 'Sync disabled', icon: 'pi pi-ban', severity: 'secondary' }
        : null
    default:
      // null / undefined — nothing to show
      return null
  }
})
</script>

<template>
  <Tag
    v-if="meta"
    :value="meta.label"
    :icon="meta.icon"
    :severity="meta.severity"
    rounded
    :data-writeback-status="status ?? 'none'"
    data-testid="writeback-status-badge"
  />
</template>
