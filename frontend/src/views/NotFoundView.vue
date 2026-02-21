<template>
  <div class="flex items-center justify-center min-h-[60vh]">
    <div class="text-center space-y-6 max-w-md px-6">
      <div class="space-y-2">
        <h1 class="text-6xl font-bold text-surface-500">404</h1>
        <h2 class="text-2xl font-semibold text-surface-900 dark:text-surface-0">
          Page Not Found
        </h2>
        <p class="text-surface-600 dark:text-surface-400">
          The page you're looking for doesn't exist or has been moved.
        </p>
      </div>

      <div class="flex justify-center gap-3">
        <Button
          label="Go to Dashboard"
          icon="pi pi-home"
          @click="router.push('/')"
          severity="secondary"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

const router = useRouter()
</script>
