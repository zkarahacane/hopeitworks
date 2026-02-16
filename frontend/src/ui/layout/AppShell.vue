<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
import AppStatusBar from './AppStatusBar.vue'
import { useLayoutStore } from '@/stores/layout'
import { useKeyboard } from '@/composables/useKeyboard'
import { useBreakpoint } from '@/composables/useBreakpoint'

const layoutStore = useLayoutStore()
const { isMobile } = useBreakpoint()
const mobileSidebarOpen = ref(false)
const router = useRouter()

const mobileNavItems = [
  { label: 'Dashboard', icon: 'pi pi-home', route: '/' },
  { label: 'Projects', icon: 'pi pi-folder', route: '/projects' },
  { label: 'Runs', icon: 'pi pi-play', route: '/runs' },
  { label: 'Settings', icon: 'pi pi-cog', route: '/settings' },
]

useKeyboard({
  '[': () => {
    if (!isMobile.value) {
      layoutStore.toggleSidebar()
    }
  },
})

// Close mobile sidebar when switching to desktop
watch(isMobile, (mobile) => {
  if (!mobile) {
    mobileSidebarOpen.value = false
  }
})

function toggleMobileSidebar() {
  mobileSidebarOpen.value = !mobileSidebarOpen.value
}
</script>

<template>
  <div class="flex h-screen flex-col overflow-hidden">
    <!-- Skip navigation link -->
    <a
      href="#main-content"
      class="sr-only focus:not-sr-only focus:absolute focus:z-[100] focus:bg-primary focus:px-4 focus:py-2 focus:text-white"
    >
      Skip to main content
    </a>

    <AppHeader
      :show-hamburger="isMobile"
      @toggle-sidebar="toggleMobileSidebar"
    />

    <div class="flex min-h-0 flex-1">
      <AppSidebar
        :collapsed="layoutStore.sidebarCollapsed"
        :mobile-open="mobileSidebarOpen"
        @close="mobileSidebarOpen = false"
      />

      <main
        id="main-content"
        class="flex-1 overflow-auto bg-surface-100 p-4"
      >
        <router-view />
      </main>
    </div>

    <!-- Mobile bottom nav -->
    <nav
      v-if="isMobile"
      class="flex h-14 items-center justify-around border-t border-surface-200 bg-surface-0"
      aria-label="Mobile navigation"
    >
      <Button
        v-for="item in mobileNavItems"
        :key="item.route"
        :icon="item.icon"
        :label="item.label"
        text
        class="flex flex-col items-center gap-0.5 text-xs"
        @click="router.push(item.route)"
      />
    </nav>

    <AppStatusBar v-if="!isMobile" />
  </div>
</template>
