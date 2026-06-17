<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { z } from 'zod'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import Message from 'primevue/message'
import Panel from 'primevue/panel'
import Skeleton from 'primevue/skeleton'
import Tag from 'primevue/tag'
import Textarea from 'primevue/textarea'
import Toast from 'primevue/toast'
import { useAsyncAction } from '@/composables/useAsyncAction'
import { useApprovalActions } from '@/features/approvals/composables/useApprovalActions'
import { apiClient } from '@/api/client'
import { getApiErrorMessage } from '@/utils/apiError'
import DiffViewer from '@/features/approvals/DiffViewer.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const runId = computed(() => route.params.runId as string)
const stepId = computed(() => route.params.stepId as string)

const diffMode = ref<'side-by-side' | 'line-by-line'>('side-by-side')
const showRejectDialog = ref(false)
const rejectReason = ref('')
const rejectValidationError = ref<string | null>(null)

const rejectSchema = z.object({
  reason: z.string().min(10, 'Reason must be at least 10 characters'),
})

const fetchAction = useAsyncAction(async () => {
  const { data, error } = await apiClient.GET('/hitl-requests/by-step/{stepId}', {
    params: { path: { stepId: stepId.value } },
  })
  if (error) throw new Error(getApiErrorMessage(error, 'Failed to load review'))
  return data
})

fetchAction.execute()

const { approveAction, rejectAction } = useApprovalActions()

async function handleApprove() {
  if (!fetchAction.data.value) return
  await approveAction.execute(fetchAction.data.value.id)
  if (!approveAction.error.value) {
    toast.add({ severity: 'success', summary: 'Approval submitted', life: 3000 })
    router.push({ name: 'run-detail', params: { id: runId.value } })
  }
}

function openRejectDialog() {
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

  if (!fetchAction.data.value) return
  await rejectAction.execute(fetchAction.data.value.id, rejectReason.value)
  if (!rejectAction.error.value) {
    showRejectDialog.value = false
    toast.add({ severity: 'success', summary: 'Rejection submitted', life: 3000 })
    router.push({ name: 'run-detail', params: { id: runId.value } })
  }
}

function navigateBack() {
  router.push({ name: 'run-detail', params: { id: runId.value } })
}
</script>

<template>
  <div class="flex flex-col h-full p-6">
    <Toast />

    <div class="flex items-start gap-3 mb-4">
      <Button
        icon="pi pi-arrow-left"
        severity="secondary"
        text
        rounded
        aria-label="Back to run"
        @click="navigateBack"
      />
      <div class="flex flex-col gap-1">
        <h1 class="m-0 text-2xl font-bold">Review &amp; Approve</h1>
        <p class="m-0 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
          The runtime is holding the container — your decision unblocks the pipeline.
        </p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="fetchAction.isLoading.value" class="flex flex-col gap-4">
      <div class="flex items-center gap-3">
        <Skeleton width="4rem" height="1.5rem" />
        <Skeleton width="60%" height="1.75rem" />
      </div>
      <Skeleton width="100%" height="4rem" />
      <Skeleton width="100%" height="20rem" />
    </div>

    <!-- Error -->
    <Message v-else-if="fetchAction.error.value" severity="error" :closable="false">
      <div class="flex items-center gap-3">
        <span>{{ fetchAction.error.value.message }}</span>
        <Button label="Retry" severity="secondary" text size="small" @click="fetchAction.execute()" />
      </div>
    </Message>

    <!-- Content -->
    <div v-else-if="fetchAction.data.value" class="flex flex-col gap-4">
      <!-- Story context -->
      <div class="flex items-center gap-3">
        <Tag :value="fetchAction.data.value.story_key" severity="secondary" />
        <h2 class="m-0 text-xl font-semibold">{{ fetchAction.data.value.story_title }}</h2>
      </div>

      <Panel v-if="fetchAction.data.value.story_objective" header="Objective" toggleable collapsed>
        <p class="m-0">{{ fetchAction.data.value.story_objective }}</p>
      </Panel>

      <!-- Diff viewer -->
      <DiffViewer
        :diff="fetchAction.data.value.diff_content"
        :mode="diffMode"
        @update:mode="diffMode = $event"
      />

      <!-- Action buttons -->
      <div class="flex gap-3 justify-end pt-4">
        <Button
          label="Reject"
          severity="danger"
          :loading="rejectAction.isLoading.value"
          @click="openRejectDialog"
        />
        <Button
          label="Approve"
          severity="warning"
          :loading="approveAction.isLoading.value"
          @click="handleApprove"
        />
      </div>

      <!-- Approve error -->
      <Message v-if="approveAction.error.value" severity="error" :closable="false">
        {{ approveAction.error.value.message }}
      </Message>
    </div>

    <!-- Reject dialog -->
    <Dialog
      v-model:visible="showRejectDialog"
      header="Reject Review"
      modal
      :style="{ width: '32rem' }"
    >
      <div class="flex flex-col gap-3">
        <label for="reject-reason">Reason for rejection</label>
        <Textarea
          id="reject-reason"
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
