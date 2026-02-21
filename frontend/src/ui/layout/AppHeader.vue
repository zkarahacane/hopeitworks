<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Menu from 'primevue/menu'
import type { MenuItem } from 'primevue/menuitem'
import { useAuthStore } from '@/stores/auth'

defineProps<{
  showHamburger: boolean
}>()

const emit = defineEmits<{
  'toggle-sidebar': []
}>()

const router = useRouter()
const authStore = useAuthStore()
const userMenu = ref<InstanceType<typeof Menu> | null>(null)

const menuItems: MenuItem[] = [
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      router.push('/login')
    },
  },
]

function toggleUserMenu(event: Event) {
  userMenu.value?.toggle(event)
}
</script>

<template>
  <header
    class="flex h-12 items-center justify-between border-b border-surface-200 bg-surface-0 px-4"
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
      <span class="text-lg font-semibold text-surface-700">Hope</span>
    </div>
    <div class="flex items-center gap-2">
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
