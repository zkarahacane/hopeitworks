<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Skeleton from 'primevue/skeleton'
import Message from 'primevue/message'
import Toast from 'primevue/toast'
import { useToast } from 'primevue/usetoast'
import ProfileInfoForm from '@/features/profile/ProfileInfoForm.vue'
import ChangePasswordForm from '@/features/profile/ChangePasswordForm.vue'
import APIKeyList from '@/features/profile/APIKeyList.vue'
import { useProfile } from '@/composables/useProfile'

const toast = useToast()
const { user, fetchMe, updateMe, changePassword } = useProfile()
const passwordResetKey = ref(0)

onMounted(() => {
  fetchMe.execute()
})

async function handleProfileSave(payload: { name: string; email: string }) {
  const result = await updateMe.execute(payload)
  if (result !== null) {
    toast.add({ severity: 'success', summary: 'Saved', detail: 'Profile updated', life: 3000 })
  } else if (updateMe.error.value) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: updateMe.error.value.message,
      life: 5000,
    })
  }
}

async function handlePasswordSave(payload: {
  current_password: string
  new_password: string
}) {
  await changePassword.execute(payload)
  if (changePassword.error.value) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: changePassword.error.value.message,
      life: 5000,
    })
  } else {
    toast.add({
      severity: 'success',
      summary: 'Done',
      detail: 'Password updated',
      life: 3000,
    })
    passwordResetKey.value++
  }
}
</script>

<template>
  <div class="flex flex-col gap-6 p-6 max-w-4xl">
    <div>
      <h1 class="text-2xl font-bold" style="font-family: var(--font-sans)">My Profile</h1>
      <p class="mt-1 text-sm" style="color: var(--p-text-muted-color)">
        Manage your account, profile information, password, and API credentials.
      </p>
    </div>

    <!-- Loading state -->
    <div v-if="fetchMe.isLoading.value && !user" class="flex flex-col gap-4">
      <Skeleton height="2rem" />
      <Skeleton height="2.5rem" />
      <Skeleton height="2rem" />
    </div>

    <!-- Error state -->
    <div v-else-if="fetchMe.error.value && !user" class="flex flex-col gap-4">
      <Message severity="error" :closable="false">
        {{ fetchMe.error.value.message }}
      </Message>
      <Button label="Retry" icon="pi pi-refresh" @click="fetchMe.execute()" />
    </div>

    <!-- Profile content -->
    <div v-else-if="user" class="flex flex-col gap-6">
      <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
        <Card>
          <template #title>Profile Information</template>
          <template #content>
            <ProfileInfoForm
              :user="user"
              :is-saving="updateMe.isLoading.value"
              @save="handleProfileSave"
            />
          </template>
        </Card>

        <Card>
          <template #title>Change Password</template>
          <template #content>
            <ChangePasswordForm
              :is-saving="changePassword.isLoading.value"
              :reset-key="passwordResetKey"
              @save="handlePasswordSave"
            />
          </template>
        </Card>
      </div>

      <Card>
        <template #title>API Keys</template>
        <template #subtitle>Stored encrypted — only the last 4 characters are shown.</template>
        <template #content>
          <APIKeyList />
        </template>
      </Card>
    </div>

    <Toast />
  </div>
</template>
