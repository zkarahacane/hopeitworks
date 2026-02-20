<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useRunProgress } from '@/features/runs/composables/useRunProgress'
import RunProgressTimeline from '@/features/runs/RunProgressTimeline.vue'
import RunLogViewer from '@/features/runs/RunLogViewer.vue'

const route = useRoute()
const projectId = route.params.id as string
const runId = route.params.runId as string

const { steps, isLoading, error } = useRunProgress(projectId, runId)
</script>

<template>
  <div class="flex flex-col gap-6 p-4">
    <RunProgressTimeline
      :steps="steps"
      :project-id="projectId"
      :run-id="runId"
      :is-loading="isLoading"
      :error="error"
    />
    <RunLogViewer :project-id="projectId" :run-id="runId" />
  </div>
</template>
