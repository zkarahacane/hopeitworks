<script setup lang="ts">
import { watch } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import type { User } from '@/stores/auth'

const props = defineProps<{
  user: User
  isSaving: boolean
}>()

const emit = defineEmits<{
  save: [payload: { name: string; email: string }]
}>()

const profileInfoSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255, 'Name must be 255 characters or less'),
  email: z.string().min(1, 'Email is required').email('Invalid email format'),
})

const { handleSubmit, meta, resetForm } = useForm({
  validationSchema: toTypedSchema(profileInfoSchema),
  initialValues: {
    name: props.user.name,
    email: props.user.email,
  },
})

const { value: name, errorMessage: nameError } = useField<string>('name')
const { value: email, errorMessage: emailError } = useField<string>('email')

watch(
  () => props.user,
  (u) => {
    resetForm({ values: { name: u.name, email: u.email } })
  },
)

const onSubmit = handleSubmit((values) => {
  emit('save', { name: values.name, email: values.email })
})

function formatDate(dateStr?: string): string {
  if (!dateStr) return 'N/A'
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}
</script>

<template>
  <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
    <div class="flex flex-col gap-1">
      <label for="profile-name" class="text-sm font-medium">Name</label>
      <InputText id="profile-name" v-model="name" :invalid="!!nameError" />
      <small v-if="nameError" :style="{ color: 'var(--status-failed-color)' }">{{ nameError }}</small>
    </div>

    <div class="flex flex-col gap-1">
      <label for="profile-email" class="text-sm font-medium">Email</label>
      <InputText id="profile-email" v-model="email" type="email" :invalid="!!emailError" />
      <small v-if="emailError" :style="{ color: 'var(--status-failed-color)' }">{{ emailError }}</small>
    </div>

    <div class="flex items-center gap-4 text-sm" :style="{ color: 'var(--p-text-muted-color)' }">
      <div class="flex items-center gap-2">
        <span>Role:</span>
        <Tag
          :value="user.role"
          :severity="user.role === 'admin' ? 'danger' : 'info'"
        />
      </div>
      <div>
        <span>Member since: {{ formatDate(user.created_at) }}</span>
      </div>
    </div>

    <Button
      type="submit"
      label="Save Changes"
      severity="primary"
      :loading="isSaving"
      :disabled="!meta.dirty || !meta.valid"
    />
  </form>
</template>
