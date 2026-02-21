<script setup lang="ts">
import { watch } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Password from 'primevue/password'
import Button from 'primevue/button'

const props = defineProps<{
  isSaving: boolean
  resetKey: number
}>()

const emit = defineEmits<{
  save: [payload: { current_password: string; new_password: string }]
}>()

const changePasswordSchema = z
  .object({
    current_password: z.string().min(1, 'Current password is required'),
    new_password: z.string().min(8, 'Password must be at least 8 characters'),
    confirm_password: z.string().min(1, 'Please confirm your new password'),
  })
  .refine((data) => data.new_password === data.confirm_password, {
    message: 'Passwords do not match',
    path: ['confirm_password'],
  })

const { handleSubmit, meta, resetForm } = useForm({
  validationSchema: toTypedSchema(changePasswordSchema),
})

const { value: currentPassword, errorMessage: currentPasswordError } =
  useField<string>('current_password')
const { value: newPassword, errorMessage: newPasswordError } = useField<string>('new_password')
const { value: confirmPassword, errorMessage: confirmPasswordError } =
  useField<string>('confirm_password')

watch(
  () => props.resetKey,
  () => {
    resetForm()
  },
)

const onSubmit = handleSubmit((values) => {
  emit('save', {
    current_password: values.current_password,
    new_password: values.new_password,
  })
})
</script>

<template>
  <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
    <div class="flex flex-col gap-1">
      <label for="current-password" class="text-sm font-medium">Current Password</label>
      <Password
        inputId="current-password"
        v-model="currentPassword"
        :feedback="false"
        toggle-mask
        :invalid="!!currentPasswordError"
        input-class="w-full"
        class="w-full"
      />
      <small v-if="currentPasswordError" class="text-red-500">{{ currentPasswordError }}</small>
    </div>

    <div class="flex flex-col gap-1">
      <label for="new-password" class="text-sm font-medium">New Password</label>
      <Password
        inputId="new-password"
        v-model="newPassword"
        :feedback="false"
        toggle-mask
        :invalid="!!newPasswordError"
        input-class="w-full"
        class="w-full"
      />
      <small v-if="newPasswordError" class="text-red-500">{{ newPasswordError }}</small>
    </div>

    <div class="flex flex-col gap-1">
      <label for="confirm-password" class="text-sm font-medium">Confirm New Password</label>
      <Password
        inputId="confirm-password"
        v-model="confirmPassword"
        :feedback="false"
        toggle-mask
        :invalid="!!confirmPasswordError"
        input-class="w-full"
        class="w-full"
      />
      <small v-if="confirmPasswordError" class="text-red-500">{{ confirmPasswordError }}</small>
    </div>

    <Button
      type="submit"
      label="Update Password"
      severity="secondary"
      :loading="isSaving"
      :disabled="!meta.dirty || !meta.valid"
    />
  </form>
</template>
