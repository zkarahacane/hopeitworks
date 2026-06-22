<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { useConfirm } from 'primevue/useconfirm'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import HaltGateCard from '@/ui/composed/HaltGateCard.vue'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import { useProbeHaltsStore, type ProbeHaltItem } from '@/stores/probeHalts'
import { useProbeHaltActions } from '@/features/approvals/composables/useProbeHaltActions'

const store = useProbeHaltsStore()
const toast = useToast()
const confirm = useConfirm()

const { resumeAction, overrideAction, sendBackAction, skipAction, abortAction } =
  useProbeHaltActions()

// Per-item pending action tracker (keyed by item id)
const pendingActions = ref(new Map<string, 'resume' | 'override' | 'send_back' | 'skip' | 'abort' | null>())

onMounted(() => {
  store.fetchPending()
})

async function handleAction(
  item: ProbeHaltItem,
  action: 'resume' | 'override' | 'send_back' | 'skip' | 'abort',
) {
  pendingActions.value.set(item.id, action)

  let actionInstance = resumeAction
  switch (action) {
    case 'resume':
      actionInstance = resumeAction
      break
    case 'override':
      actionInstance = overrideAction
      break
    case 'send_back':
      actionInstance = sendBackAction
      break
    case 'skip':
      actionInstance = skipAction
      break
    case 'abort':
      actionInstance = abortAction
      break
  }

  await actionInstance.execute(item.id)

  if (!actionInstance.error.value) {
    store.handleResolvedEvent(item.id)
    toast.add({
      severity: 'success',
      summary: `${capitalize(action.replace('_', ' '))}`,
      detail: `${item.storyKey || 'Halt'} resolved`,
      life: 3000,
    })
  } else {
    toast.add({
      severity: 'error',
      summary: 'Action failed',
      detail: actionInstance.error.value.message,
      life: 4000,
    })
  }

  pendingActions.value.delete(item.id)
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function confirmBulkResume(probeKey: string, groupItems: ProbeHaltItem[]) {
  confirm.require({
    header: `Resume all ${probeKey} halts`,
    message: `Resume ${groupItems.length} halted step(s) with probe type "${probeKey}"?`,
    icon: 'pi pi-play',
    acceptLabel: `Resume all (${groupItems.length})`,
    rejectLabel: 'Cancel',
    acceptClass: 'p-button-success',
    accept: async () => {
      for (const item of groupItems) {
        await handleAction(item, 'resume')
      }
    },
  })
}
</script>

<template>
  <div class="p-6 flex flex-col gap-6">
    <Toast />
    <ConfirmDialog />

    <!-- Header -->
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-3">
        <h1 class="m-0 text-2xl font-bold">Halt-gate triage</h1>
        <StatusBadge
          v-if="!store.isLoading"
          status="gate"
          :label="String(store.count)"
          :icon="false"
          :animated="false"
        />
      </div>
      <p class="m-0 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
        {{ store.count }} parked &middot; Runs blocked by a watchdog probe. Triage and resume to
        unblock the runtime.
      </p>
    </div>

    <!-- Loading skeleton -->
    <div v-if="store.isLoading && store.items.length === 0" class="flex flex-col gap-3">
      <Skeleton height="10rem" border-radius="0.5rem" />
      <Skeleton height="10rem" border-radius="0.5rem" />
      <Skeleton height="10rem" border-radius="0.5rem" />
    </div>

    <!-- Error state -->
    <Message v-else-if="store.error" severity="error" :closable="false">
      <div class="flex items-center gap-2">
        <span>{{ store.error }}</span>
        <Button
          label="Retry"
          icon="pi pi-refresh"
          size="small"
          severity="danger"
          outlined
          @click="store.fetchPending()"
        />
      </div>
    </Message>

    <!-- Empty state -->
    <div
      v-else-if="!store.isLoading && store.items.length === 0"
      class="flex flex-col items-center justify-center py-16 gap-3"
      :style="{ color: 'var(--p-text-muted-color)' }"
    >
      <i class="pi pi-check-circle text-4xl" :style="{ color: 'var(--status-done-color)' }" />
      <p class="text-lg font-medium">No parked halts</p>
      <p class="text-sm">All monitored runs are proceeding normally.</p>
    </div>

    <!-- Grouped sections -->
    <div v-else class="flex flex-col gap-8">
      <section
        v-for="(groupItems, probeKey) in store.byReason"
        :key="probeKey"
        class="flex flex-col gap-3"
      >
        <!-- Group heading -->
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-2">
            <i
              class="pi pi-exclamation-triangle"
              :style="{ color: 'var(--status-gate-color)' }"
              aria-hidden="true"
            />
            <span class="font-semibold font-mono">{{ probeKey }}</span>
            <span :style="{ color: 'var(--p-text-muted-color)', fontSize: '0.875rem' }">
              ({{ groupItems.length }})
            </span>
          </div>
          <Button
            :label="`Resume all (${groupItems.length})`"
            icon="pi pi-play"
            severity="success"
            size="small"
            outlined
            :disabled="groupItems.some((i) => pendingActions.has(i.id))"
            data-testid="halt-group-bulk-resume"
            @click="confirmBulkResume(probeKey, groupItems)"
          />
        </div>

        <!-- Cards for this group -->
        <HaltGateCard
          v-for="item in groupItems"
          :key="item.id"
          :story-key="item.storyKey"
          :step-name="item.stepName"
          :stage-name="item.stageName"
          :halt-reason="item.haltReason"
          :pending-since="item.pendingSince"
          :busy="pendingActions.has(item.id)"
          :pending-action="pendingActions.get(item.id) ?? null"
          data-testid="halt-gate-card-item"
          @resume="handleAction(item, 'resume')"
          @override="handleAction(item, 'override')"
          @send-back="handleAction(item, 'send_back')"
          @skip="handleAction(item, 'skip')"
          @abort="handleAction(item, 'abort')"
        />
      </section>
    </div>
  </div>
</template>
