<script setup lang="ts">
import Timeline from 'primevue/timeline'
import Tag from 'primevue/tag'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Button from 'primevue/button'
import ProgressSpinner from 'primevue/progressspinner'
import { formatStepDuration } from '@/utils/formatStepDuration'
import { useStepTimer } from './composables/useStepTimer'
import type { RunStep } from './composables/useRunDetail'

defineProps<{
  steps: RunStep[]
  projectId: string
  runId: string
  isLoading: boolean
  error: Error | null
}>()

const stepSeverity: Record<string, 'success' | 'info' | 'danger' | 'secondary' | 'warn'> = {
  completed: 'success',
  running: 'info',
  failed: 'danger',
  pending: 'secondary',
  waiting_approval: 'warn',
  cancelled: 'secondary',
}

const markerIcon: Record<string, string> = {
  completed: 'pi-check-circle',
  running: '',
  failed: 'pi-times-circle',
  pending: 'pi-circle',
  waiting_approval: 'pi-hourglass',
  cancelled: 'pi-ban',
}

/** Creates a reactive timer for a running step. Called per running step instance. */
function getStepTimer(step: RunStep) {
  if (step.status === 'running' && step.started_at) {
    return useStepTimer(step.started_at)
  }
  return null
}
</script>

<template>
  <div data-testid="run-progress-timeline">
    <!-- Loading state -->
    <Skeleton v-if="isLoading" height="120px" data-testid="timeline-skeleton" />

    <!-- Error state -->
    <Message v-else-if="error" severity="error" :closable="false" data-testid="timeline-error">
      {{ error.message }}
    </Message>

    <!-- Empty state -->
    <Message
      v-else-if="steps.length === 0"
      severity="info"
      :closable="false"
      data-testid="timeline-empty"
    >
      No pipeline steps found for this run
    </Message>

    <!-- Timeline -->
    <Timeline
      v-else
      :value="steps"
      layout="vertical"
      align="left"
      data-testid="timeline"
    >
      <template #marker="{ item }">
        <ProgressSpinner
          v-if="(item as RunStep).status === 'running'"
          style="width: 1.5rem; height: 1.5rem"
          data-testid="step-spinner"
        />
        <span
          v-else
          class="pi"
          :class="markerIcon[(item as RunStep).status] ?? 'pi-circle'"
        />
      </template>
      <template #content="{ item }">
        <div class="flex flex-col gap-1 mb-4" :data-testid="`step-${(item as RunStep).id}`">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="font-medium">{{ (item as RunStep).step_name }}</span>
            <Tag
              :value="(item as RunStep).status"
              :severity="stepSeverity[(item as RunStep).status] ?? 'secondary'"
              class="text-xs"
              data-testid="step-status-tag"
            />

            <!-- Waiting approval badge and link -->
            <template v-if="(item as RunStep).status === 'waiting_approval'">
              <Tag
                value="Awaiting Approval"
                severity="warn"
                class="text-xs"
                data-testid="hitl-tag"
              />
              <router-link
                :to="`/projects/${projectId}/runs/${runId}/approve/${(item as RunStep).id}`"
              >
                <Button
                  label="Review"
                  severity="warn"
                  size="small"
                  data-testid="hitl-button"
                />
              </router-link>
            </template>
          </div>

          <!-- Duration or live timer -->
          <div class="text-sm text-surface-500">
            <template
              v-if="(item as RunStep).status === 'completed' && (item as RunStep).started_at && (item as RunStep).completed_at"
            >
              <span data-testid="step-duration">
                {{ formatStepDuration((item as RunStep).started_at!, (item as RunStep).completed_at!) }}
              </span>
            </template>
            <template v-else-if="(item as RunStep).status === 'running' && (item as RunStep).started_at">
              <span data-testid="step-elapsed">
                {{ getStepTimer(item as RunStep)?.elapsed.value }}
              </span>
            </template>
          </div>

          <!-- Error message -->
          <div
            v-if="(item as RunStep).error_message"
            class="text-sm text-red-500 mt-1"
            data-testid="step-error"
          >
            {{ (item as RunStep).error_message }}
          </div>
        </div>
      </template>
    </Timeline>
  </div>
</template>
