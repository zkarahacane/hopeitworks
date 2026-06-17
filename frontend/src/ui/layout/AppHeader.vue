<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Menu from 'primevue/menu'
import type { MenuItem } from 'primevue/menuitem'
import { useAuthStore } from '@/stores/auth'
import { useTheme, type ThemeMode } from '@/composables/useTheme'

defineProps<{
  showHamburger: boolean
}>()

const emit = defineEmits<{
  'toggle-sidebar': []
}>()

const router = useRouter()
const authStore = useAuthStore()
const userMenu = ref<InstanceType<typeof Menu> | null>(null)

// 3-state theme control — cycles Auto → Dark → Light. Persisted via useTheme.
const { mode, cycle } = useTheme()
const THEME_META: Record<ThemeMode, { icon: string; label: string }> = {
  auto: { icon: 'pi pi-desktop', label: 'Theme: Auto (follows page)' },
  dark: { icon: 'pi pi-moon', label: 'Theme: Dark' },
  light: { icon: 'pi pi-sun', label: 'Theme: Light' },
}
const themeIcon = computed(() => THEME_META[mode.value].icon)
const themeLabel = computed(() => THEME_META[mode.value].label)

const menuItems = computed<MenuItem[]>(() => [
  {
    label: authStore.user?.name ?? 'User',
    disabled: true,
    class: 'font-semibold',
  },
  {
    label: authStore.user?.email ?? '',
    disabled: true,
    class: 'text-sm',
  },
  { separator: true },
  {
    label: 'My Profile',
    icon: 'pi pi-user-edit',
    command: () => router.push({ name: 'profile' }),
  },
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      router.push('/login')
    },
  },
])

function toggleUserMenu(event: Event) {
  userMenu.value?.toggle(event)
}
</script>

<template>
  <header
    class="flex h-12 items-center justify-between px-4"
    :style="{
      borderBottom: '1px solid var(--surface-border)',
      background: 'var(--surface-raised)',
    }"
  >
    <div class="flex items-center gap-2">
      <Button
        v-if="showHamburger"
        icon="pi pi-bars"
        text
        rounded
        aria-label="Toggle sidebar"
        @click="emit('toggle-sidebar')"
      />
      <span class="text-lg" style="font-family: var(--font-sans)">
        <span class="font-semibold" style="color: var(--p-text-color)">hope</span><span class="font-normal" style="color: var(--p-text-muted-color)">it</span><span class="font-semibold" style="color: var(--p-text-color)">works</span>
      </span>
    </div>
    <div class="flex items-center gap-2">
      <Button
        :icon="themeIcon"
        text
        rounded
        :aria-label="themeLabel"
        :title="themeLabel"
        data-testid="theme-toggle"
        @click="cycle"
      />
      <Button
        icon="pi pi-user"
        text
        rounded
        aria-label="User menu"
        data-testid="user-menu-button"
        @click="toggleUserMenu"
      />
      <Menu
        ref="userMenu"
        :model="menuItems"
        :popup="true"
        data-testid="user-menu"
      />
    </div>
  </header>
</template>
