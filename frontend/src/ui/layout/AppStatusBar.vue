<script setup lang="ts"></script>

<template>
  <footer
    class="flex h-6 items-center justify-between px-4 text-xs"
    :style="{
      borderTop: '1px solid var(--p-content-border-color)',
      background: 'var(--app-chrome-bg)',
      color: 'var(--p-text-muted-color)',
    }"
  >
    <div class="flex items-center gap-1">
      <i
        class="pi pi-circle-fill text-[0.5rem] live-pulse"
        style="color: var(--status-running-color)"
      />
      <span>Connected</span>
    </div>
    <span class="font-mono">v0.0.0</span>
  </footer>
</template>
