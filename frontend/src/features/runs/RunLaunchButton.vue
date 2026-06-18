<script setup lang="ts">
import { computed } from 'vue'
import Button from 'primevue/button'
import { statusFamily } from '@/utils/statusToken'

const props = defineProps<{
  storyId: string
  storyKey: string
  storyTitle: string
  /** Raw story status — normalized through statusFamily so any enum value works. */
  status: string
}>()

const emit = defineEmits<{
  launchClick: []
}>()

const family = computed(() => statusFamily(props.status))
</script>

<template>
  <!-- A run is active → not launchable -->
  <span v-if="family === 'running'" v-tooltip="'A run is already in progress'">
    <Button label="Running..." icon="pi pi-spin pi-spinner" disabled />
  </span>
  <span v-else-if="family === 'gate'" v-tooltip="'Run is paused on a human gate'">
    <Button label="Awaiting gate" icon="pi pi-pause" severity="warn" disabled />
  </span>

  <!-- Failed → relaunch -->
  <Button
    v-else-if="family === 'failed'"
    label="Retry Run"
    icon="pi pi-refresh"
    severity="danger"
    outlined
    @click="emit('launchClick')"
  />

  <!-- Done → re-run -->
  <Button
    v-else-if="family === 'done'"
    label="Re-run"
    icon="pi pi-replay"
    severity="secondary"
    outlined
    @click="emit('launchClick')"
  />

  <!-- Backlog / queued → launch -->
  <Button
    v-else
    label="Launch Run"
    icon="pi pi-play"
    severity="success"
    @click="emit('launchClick')"
  />
</template>
