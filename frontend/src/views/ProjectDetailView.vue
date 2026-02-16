<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import Button from 'primevue/button'

const route = useRoute()
const router = useRouter()
const projectId = route.params.id as string

function goToSettings() {
  router.push({ name: 'project-settings', params: { id: projectId } })
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Project Detail</h1>
      <Button label="Settings" icon="pi pi-cog" severity="secondary" @click="goToSettings" />
    </div>
  </div>
</template>
