<script setup lang="ts">
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

const router = useRouter()
</script>

<template>
  <div
    class="flex items-center justify-center min-h-screen"
    style="background: #0B0F0D;"
  >
    <div class="flex flex-col items-center gap-8 max-w-md px-6 text-center">
      <!-- Mini graph motif -->
      <svg width="120" height="80" viewBox="0 0 120 80" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <!-- Edges -->
        <line x1="20" y1="40" x2="55" y2="40" stroke="#28332C" stroke-width="1.5" stroke-dasharray="4 3"/>
        <line x1="65" y1="40" x2="100" y2="20" stroke="#28332C" stroke-width="1.5" stroke-dasharray="4 3"/>
        <line x1="65" y1="40" x2="100" y2="60" stroke="#28332C" stroke-width="1.5" stroke-dasharray="4 3"/>
        <!-- Done node -->
        <circle cx="20" cy="40" r="8" fill="#1A221D" stroke="#2D6A4F" stroke-width="1.5"/>
        <text x="20" y="44" text-anchor="middle" font-size="7" fill="#2D6A4F" font-family="monospace">✓</text>
        <!-- Failed / 404 node (center) -->
        <circle cx="60" cy="40" r="10" fill="#1A0A0A" stroke="var(--status-failed-color, #EF4444)" stroke-width="2"/>
        <text x="60" y="44" text-anchor="middle" font-size="8" fill="var(--status-failed-color, #EF4444)" font-family="monospace">×</text>
        <!-- Queued nodes -->
        <circle cx="100" cy="20" r="7" fill="#1A221D" stroke="#28332C" stroke-width="1.5"/>
        <circle cx="100" cy="60" r="7" fill="#1A221D" stroke="#28332C" stroke-width="1.5"/>
      </svg>

      <!-- Text -->
      <div class="flex flex-col gap-3">
        <h1
          class="text-3xl font-bold"
          style="font-family: var(--font-sans); color: var(--status-failed-color, #EF4444);"
        >
          404 · This node isn't in the graph
        </h1>
        <p
          class="text-sm"
          style="color: #8A9E93; font-family: var(--font-mono, monospace);"
        >
          /settings now lives under your profile.
        </p>
      </div>

      <!-- Actions -->
      <div class="flex gap-3">
        <Button
          label="Go to dashboard"
          icon="pi pi-home"
          severity="secondary"
          @click="router.push('/')"
        />
        <Button
          label="Profile & settings"
          icon="pi pi-user"
          @click="router.push({ name: 'profile' })"
        />
      </div>
    </div>
  </div>
</template>
