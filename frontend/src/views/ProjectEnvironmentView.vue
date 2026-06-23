<script setup lang="ts">
import { useRoute } from 'vue-router'
import Toast from 'primevue/toast'
import EnvironmentEditor from '@/features/environments/EnvironmentEditor.vue'

const route = useRoute()
const projectId = route.params.id as string
</script>

<template>
  <div class="flex flex-col gap-6 p-6">
    <h2 class="text-xl font-semibold">Environment</h2>
    <EnvironmentEditor :project-id="projectId" />
    <Toast />
  </div>
</template>
