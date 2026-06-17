<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Button from 'primevue/button'
import Badge from 'primevue/badge'
import Divider from 'primevue/divider'
import { useAuthStore } from '@/stores/auth'
import { useHITLStore } from '@/stores/hitl'

const props = defineProps<{
  collapsed: boolean
  mobileOpen: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const hitlStore = useHITLStore()

const isAdmin = computed(() => authStore.user?.role === 'admin')

/** Determine if a nav item is active based on current route path */
function isActive(itemRoute: string): boolean {
  if (itemRoute === '/') return route.path === '/'
  return route.path.startsWith(itemRoute)
}

const navItems = [
  { label: 'Dashboard', icon: 'pi pi-home', route: '/' },
  { label: 'Projects', icon: 'pi pi-folder', route: '/projects' },
  { label: 'Runs', icon: 'pi pi-play', route: '/runs' },
  { label: 'Approvals', icon: 'pi pi-bell', route: '/approvals' },
  { label: 'Settings', icon: 'pi pi-cog', route: '/profile' },
]

const adminNavItems = [
  { label: 'Administration', icon: 'pi pi-shield', route: '/admin/users' },
]

const sidebarWidth = computed(() => (props.collapsed ? 'w-12' : 'w-60'))

/**
 * Active nav treatment — the new design language: a green status-accent left
 * bar + green text/icon + a subtle elevated surface. NOT a blue pill. Inactive
 * items are muted text on transparent. All colors are scheme-aware PrimeVue /
 * status tokens so the chrome flips with `.dark` / light automatically.
 */
function navItemStyle(active: boolean) {
  if (active) {
    return {
      color: 'var(--status-running-color)',
      background: 'var(--status-running-surface)',
      borderLeft: '3px solid var(--status-running-color)',
    }
  }
  return {
    color: 'var(--p-text-muted-color)',
    background: 'transparent',
    borderLeft: '3px solid transparent',
  }
}

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
      'flex flex-col',
      'transition-all duration-200 ease-in-out',
      // Mobile: overlay drawer
      mobileOpen
        ? 'fixed inset-y-0 left-0 z-50 w-60 shadow-lg lg:relative lg:z-auto lg:shadow-none'
        : 'hidden lg:flex',
      // Desktop: collapsible width
      !mobileOpen ? sidebarWidth : '',
    ]"
    :style="{
      borderRight: '1px solid var(--p-content-border-color)',
      background: 'var(--app-chrome-bg)',
    }"
  >
    <nav class="flex flex-1 flex-col gap-1 p-2" aria-label="Main navigation">
      <div
        v-for="item in navItems"
        :key="item.route"
        class="relative"
      >
        <Button
          text
          :icon="item.icon"
          :label="collapsed && !mobileOpen ? undefined : item.label"
          :aria-current="isActive(item.route) ? 'page' : undefined"
          :class="[
            'w-full justify-start !rounded-md',
            collapsed && !mobileOpen ? '!px-0 justify-center' : '',
          ]"
          :style="navItemStyle(isActive(item.route))"
          @click="navigate(item.route)"
        />
        <Badge
          v-if="item.route === '/approvals' && hitlStore.pendingCount > 0"
          :value="hitlStore.pendingCount"
          severity="danger"
          class="absolute -right-1 -top-1"
        />
      </div>

      <template v-if="isAdmin">
        <Divider />
        <Button
          v-for="item in adminNavItems"
          :key="item.route"
          text
          :icon="item.icon"
          :label="collapsed && !mobileOpen ? undefined : item.label"
          :aria-current="isActive(item.route) ? 'page' : undefined"
          :class="[
            'w-full justify-start !rounded-md',
            collapsed && !mobileOpen ? '!px-0 justify-center' : '',
          ]"
          :style="navItemStyle(isActive(item.route))"
          @click="navigate(item.route)"
        />
      </template>
    </nav>
  </aside>
</template>
