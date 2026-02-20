<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Toast from 'primevue/toast'
import { useToast } from 'primevue/usetoast'
import AppHeader from './AppHeader.vue'
import AppSidebar from './AppSidebar.vue'
import AppStatusBar from './AppStatusBar.vue'
import { useLayoutStore } from '@/stores/layout'
import { useHITLStore } from '@/stores/hitl'
import { useSSE } from '@/composables/useSSE'
import { useKeyboard } from '@/composables/useKeyboard'
import { useBreakpoint } from '@/composables/useBreakpoint'

const layoutStore = useLayoutStore()
const hitlStore = useHITLStore()
const toast = useToast()
const { isMobile } = useBreakpoint()
const mobileSidebarOpen = ref(false)
const router = useRouter()

/** Global SSE subscription for HITL events */
const { close: closeSSE } = useSSE('', (eventName, data) => {
  if (eventName === 'hitl.pending') {
    hitlStore.handlePendingEvent(
      data as {
        hitl_request_id: string
        run_id: string
        step_id: string
        project_id: string
        story_key: string
        pr_url?: string
        pending_since?: string
      },
    )
  }
  if (eventName === 'hitl.approved' || eventName === 'hitl.rejected') {
    hitlStore.handleResolvedEvent(
      (data as { hitl_request_id: string }).hitl_request_id,
    )
  }
})

onBeforeUnmount(closeSSE)

/** Watch for new pending approvals and show toast notifications */
watch(
  () => hitlStore.pendingCount,
  (newCount, oldCount) => {
    if (newCount > oldCount) {
      const latest = hitlStore.pendingItems[hitlStore.pendingItems.length - 1]
      if (latest) {
        toast.add({
          severity: 'warn',
          summary: 'Review Required',
          detail: `Review required for ${latest.storyKey}`,
          life: 0,
          group: 'hitl',
        })
      }
    }
  },
)

function navigateToApproval() {
  const latest = hitlStore.pendingItems[hitlStore.pendingItems.length - 1]
  if (!latest) return
  hitlStore.handleResolvedEvent(latest.hitlRequestId)
  router.push({
    name: 'hitl-approve',
    params: {
      id: latest.projectId,
      runId: latest.runId,
      stepId: latest.stepId,
    },
  })
}

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

    <!-- Global toast for SSE notifications (HITL approvals) -->
    <Toast position="top-right" group="hitl">
      <template #message="slotProps">
        <div class="flex flex-col gap-2">
          <div class="flex items-center gap-2">
            <i class="pi pi-exclamation-triangle" />
            <span class="font-semibold">{{ slotProps.message.summary }}</span>
          </div>
          <span>{{ slotProps.message.detail }}</span>
          <Button
            label="Review Now"
            size="small"
            severity="warn"
            @click="navigateToApproval()"
          />
        </div>
      </template>
    </Toast>
  </div>
</template>
