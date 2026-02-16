<script setup lang="ts">
import Button from 'primevue/button'
</script>

<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-4 p-8">
    <h1 class="text-4xl font-bold">hopeitworks Frontend Scaffold</h1>

    <div class="flex gap-4">
      <Button severity="primary"> PrimeVue Button </Button>

      <Button severity="secondary"> Secondary Button </Button>
    </div>

    <div class="grid grid-cols-2 gap-4 mt-8">
      <div class="p-4 border rounded">
        <p>Tailwind: flex, gap, grid work</p>
      </div>
      <div class="p-4 border rounded">
        <p>CSS Layers: utilities override PrimeVue</p>
      </div>
    </div>
  </div>
</template>
