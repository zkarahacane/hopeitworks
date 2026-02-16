<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

const props = defineProps<{
  collapsed: boolean
  mobileOpen: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()

const navItems = [
  { label: 'Dashboard', icon: 'pi pi-home', route: '/' },
  { label: 'Projects', icon: 'pi pi-folder', route: '/projects' },
  { label: 'Runs', icon: 'pi pi-play', route: '/runs' },
  { label: 'Settings', icon: 'pi pi-cog', route: '/settings' },
]

const sidebarWidth = computed(() => (props.collapsed ? 'w-12' : 'w-60'))

function navigate(route: string) {
  router.push(route)
  emit('close')
}
</script>

<template>
  <!-- Mobile overlay backdrop -->
  <div
    v-if="mobileOpen"
    class="fixed inset-0 z-40 bg-black/50 lg:hidden"
    @click="emit('close')"
  />

  <aside
    :class="[
      'flex flex-col border-r border-surface-200 bg-surface-50',
      'transition-all duration-200 ease-in-out',
      // Mobile: overlay drawer
      mobileOpen
        ? 'fixed inset-y-0 left-0 z-50 w-60 shadow-lg lg:relative lg:z-auto lg:shadow-none'
        : 'hidden lg:flex',
      // Desktop: collapsible width
      !mobileOpen ? sidebarWidth : '',
    ]"
  >
    <nav class="flex flex-1 flex-col gap-1 p-2" aria-label="Main navigation">
      <Button
        v-for="item in navItems"
        :key="item.route"
        :icon="item.icon"
        :label="collapsed && !mobileOpen ? undefined : item.label"
        text
        :class="[
          'justify-start',
          collapsed && !mobileOpen ? '!px-0 justify-center' : '',
        ]"
        @click="navigate(item.route)"
      />
    </nav>
  </aside>
</template>
