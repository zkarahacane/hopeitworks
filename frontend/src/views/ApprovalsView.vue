<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { z } from 'zod'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Textarea from 'primevue/textarea'
import Toast from 'primevue/toast'
import HitlGateCard from '@/ui/composed/HitlGateCard.vue'
import StatusBadge from '@/ui/primitives/StatusBadge.vue'
import { useHITLStore, type HITLPendingItem } from '@/stores/hitl'
import { useApprovalActions } from '@/features/approvals/composables/useApprovalActions'

const hitlStore = useHITLStore()
const router = useRouter()
const toast = useToast()

onMounted(() => {
  hitlStore.fetchPending()
})

// Per-card pending action tracker (keyed by hitlRequestId)
const pendingActions = ref(new Map<string, 'approve' | 'request_changes' | 'reject' | null>())

// Reject dialog state
const showRejectDialog = ref(false)
const activeRejectItemId = ref<string | null>(null)
const rejectReason = ref('')
const rejectValidationError = ref<string | null>(null)

const rejectSchema = z.object({
  reason: z.string().min(10, 'Reason must be at least 10 characters'),
})

// Single shared approval actions instance — we serialise one action at a time per card
const { approveAction, rejectAction } = useApprovalActions()

function getItem(hitlRequestId: string): HITLPendingItem | undefined {
  return hitlStore.pendingItems.find((i) => i.hitlRequestId === hitlRequestId)
}

async function handleApprove(item: HITLPendingItem) {
  pendingActions.value.set(item.hitlRequestId, 'approve')
  await approveAction.execute(item.hitlRequestId)
  if (!approveAction.error.value) {
    hitlStore.handleResolvedEvent(item.hitlRequestId)
    toast.add({ severity: 'success', summary: 'Approved', detail: `${item.storyKey} unblocked`, life: 3000 })
    router.push({ name: 'run-detail', params: { id: item.runId }, query: { projectId: item.projectId } })
  } else {
    toast.add({ severity: 'error', summary: 'Approve failed', detail: approveAction.error.value.message, life: 4000 })
  }
  pendingActions.value.delete(item.hitlRequestId)
}

function openRejectDialog(hitlRequestId: string) {
  activeRejectItemId.value = hitlRequestId
  rejectReason.value = ''
  rejectValidationError.value = null
  showRejectDialog.value = true
}

async function handleReject() {
  const result = rejectSchema.safeParse({ reason: rejectReason.value })
  if (!result.success) {
    rejectValidationError.value = result.error.errors[0]?.message ?? 'Invalid input'
    return
  }
  rejectValidationError.value = null

  const id = activeRejectItemId.value
  if (!id) return
  const item = getItem(id)
  if (!item) return

  pendingActions.value.set(id, 'reject')
  await rejectAction.execute(id, rejectReason.value)
  if (!rejectAction.error.value) {
    showRejectDialog.value = false
    hitlStore.handleResolvedEvent(id)
    toast.add({ severity: 'info', summary: 'Rejected', detail: `${item.storyKey} rejected`, life: 3000 })
    router.push({ name: 'run-detail', params: { id: item.runId }, query: { projectId: item.projectId } })
  } else {
    toast.add({ severity: 'error', summary: 'Reject failed', detail: rejectAction.error.value.message, life: 4000 })
  }
  pendingActions.value.delete(id)
}

// Backend #10 gap: branch name, PR number, diffstat not in SSE payload
// stepName falls back to storyTitle when available, else generic label
function resolveStepName(item: HITLPendingItem): string {
  return item.storyTitle && item.storyTitle !== '' ? item.storyTitle : 'Review Gate'
}
</script>

<template>
  <div class="p-6 flex flex-col gap-6">
    <Toast />

    <!-- Header -->
    <div class="flex flex-col gap-1">
      <div class="flex items-center gap-3">
        <h1 class="m-0 text-2xl font-bold">Approvals</h1>
        <StatusBadge
          v-if="!hitlStore.isLoading"
          status="gate"
          :label="String(hitlStore.pendingCount)"
          :icon="false"
          :animated="false"
        />
      </div>
      <p class="m-0 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
        {{ hitlStore.pendingCount }} waiting · Pipelines paused on a human gate. The runtime holds the container until you decide.
      </p>
    </div>

    <!-- Loading skeleton -->
    <div v-if="hitlStore.isLoading && hitlStore.pendingItems.length === 0" class="flex flex-col gap-3">
      <Skeleton height="7rem" border-radius="0.5rem" />
      <Skeleton height="7rem" border-radius="0.5rem" />
      <Skeleton height="7rem" border-radius="0.5rem" />
    </div>

    <!-- Error state -->
    <Message
      v-else-if="hitlStore.error"
      severity="error"
      :closable="false"
    >
      <div class="flex items-center gap-2">
        <span>{{ hitlStore.error }}</span>
        <Button
          label="Retry"
          icon="pi pi-refresh"
          size="small"
          severity="danger"
          outlined
          @click="hitlStore.fetchPending()"
        />
      </div>
    </Message>

    <!-- Empty state -->
    <div
      v-else-if="!hitlStore.isLoading && hitlStore.pendingItems.length === 0"
      class="flex flex-col items-center justify-center py-16 gap-3"
      :style="{ color: 'var(--p-text-muted-color)' }"
    >
      <i class="pi pi-check-circle text-4xl" :style="{ color: 'var(--status-done-color)' }" />
      <p class="text-lg font-medium">No pending approvals</p>
      <p class="text-sm">All pipelines are running freely.</p>
    </div>

    <!-- Cards list -->
    <div v-else class="flex flex-col gap-4">
      <HitlGateCard
        v-for="item in hitlStore.pendingItems"
        :key="item.hitlRequestId"
        :story-key="item.storyKey"
        :step-name="resolveStepName(item)"
        :pr-url="item.prUrl"
        :pending-since="item.pendingSince"
        :busy="pendingActions.has(item.hitlRequestId)"
        :pending-action="pendingActions.get(item.hitlRequestId) ?? null"
        :animated="true"
        data-testid="approvals-gate-card"
        @approve="handleApprove(item)"
        @request-changes="openRejectDialog(item.hitlRequestId)"
        @reject="openRejectDialog(item.hitlRequestId)"
      />
    </div>

    <!-- Reject / Request-changes dialog -->
    <Dialog
      v-model:visible="showRejectDialog"
      header="Reject Review"
      modal
      :style="{ width: '32rem' }"
    >
      <div class="flex flex-col gap-3">
        <label for="approvals-reject-reason">Reason for rejection</label>
        <Textarea
          id="approvals-reject-reason"
          v-model="rejectReason"
          rows="4"
          placeholder="Explain why this review is being rejected (min 10 characters)..."
          :invalid="!!rejectValidationError"
        />
        <small v-if="rejectValidationError" class="p-error">{{ rejectValidationError }}</small>
        <Message v-if="rejectAction.error.value" severity="error" :closable="false">
          {{ rejectAction.error.value.message }}
        </Message>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <Button label="Cancel" severity="secondary" text @click="showRejectDialog = false" />
          <Button
            label="Confirm Rejection"
            severity="danger"
            :loading="rejectAction.isLoading.value"
            @click="handleReject"
          />
        </div>
      </template>
    </Dialog>
  </div>
</template>
