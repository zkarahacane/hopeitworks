<script setup lang="ts">
import Button from 'primevue/button'

defineProps<{
  storyId: string
  storyKey: string
  storyTitle: string
  status: 'backlog' | 'running' | 'done' | 'failed'
}>()

const emit = defineEmits<{
  launchClick: []
}>()
</script>

<template>
  <Button
    v-if="status === 'backlog'"
    label="Launch Run"
    icon="pi pi-play"
    severity="success"
    @click="emit('launchClick')"
  />
  <span v-else-if="status === 'running'" v-tooltip="'A run is already in progress'">
    <Button label="Running..." icon="pi pi-spin pi-spinner" disabled />
  </span>
</template>
