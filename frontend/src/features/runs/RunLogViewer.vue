<script setup lang="ts">
import LogViewer from '@/ui/composed/LogViewer.vue'
import { useRunLogs } from './composables/useRunLogs'

const props = defineProps<{
  projectId: string
  runId: string
}>()

const { lines, sseStatus, clearLogs } = useRunLogs(props.projectId, props.runId)
</script>

<template>
  <LogViewer :lines="lines" :status="sseStatus" @clear="clearLogs" />
</template>
