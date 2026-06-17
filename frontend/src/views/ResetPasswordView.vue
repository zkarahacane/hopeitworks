<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const route = useRoute()
const { resetPassword, loading, error } = useAuth()

const token = computed(() => (route.query.token as string) || '')
const hasToken = computed(() => !!token.value)

const resetSchema = toTypedSchema(
  z
    .object({
      password: z.string().min(8, 'Password must be at least 8 characters'),
      confirmPassword: z.string().min(1, 'Please confirm your password'),
    })
    .refine((data) => data.password === data.confirmPassword, {
      message: 'Passwords do not match',
      path: ['confirmPassword'],
    }),
)

const { handleSubmit } = useForm({ validationSchema: resetSchema })
const { value: password, errorMessage: passwordError } = useField<string>('password')
const { value: confirmPassword, errorMessage: confirmPasswordError } = useField<string>('confirmPassword')

const onSubmit = handleSubmit(async (values) => {
  const success = await resetPassword(token.value, values.password)
  if (success) {
    router.push('/login?reset=success')
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center p-6" style="background: var(--p-surface-900)">
    <div
      class="flex w-full max-w-sm flex-col gap-6 rounded-xl p-8"
      style="background: var(--p-surface-800)"
    >
      <div>
        <p class="text-xs font-semibold tracking-widest uppercase mb-4" style="color: var(--p-surface-400)">
          hopeitworks
        </p>
        <h1 class="text-2xl font-bold" style="color: var(--p-surface-0)">Set new password</h1>
      </div>

      <template v-if="hasToken">
        <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
          <div class="flex flex-col gap-1">
            <label for="password" class="text-sm font-medium" style="color: var(--p-surface-200)">New password</label>
            <Password
              inputId="password"
              v-model="password"
              :feedback="false"
              toggle-mask
              :invalid="!!passwordError"
              input-class="w-full"
              class="w-full"
            />
            <small v-if="passwordError" style="color: var(--p-red-400)">{{ passwordError }}</small>
          </div>

          <div class="flex flex-col gap-1">
            <label for="confirmPassword" class="text-sm font-medium" style="color: var(--p-surface-200)">Confirm new password</label>
            <Password
              inputId="confirmPassword"
              v-model="confirmPassword"
              :feedback="false"
              toggle-mask
              :invalid="!!confirmPasswordError"
              input-class="w-full"
              class="w-full"
            />
            <small v-if="confirmPasswordError" style="color: var(--p-red-400)">{{ confirmPasswordError }}</small>
          </div>

          <Button
            type="submit"
            label="Set new password"
            :loading="loading"
            :disabled="loading"
            class="w-full mt-1"
          />

          <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
        </form>
      </template>

      <template v-else>
        <Message severity="error" :closable="false">
          Invalid or expired link. This password reset link is invalid or has expired.
        </Message>

        <RouterLink to="/forgot-password" class="w-full">
          <Button label="Request a new link" class="w-full" />
        </RouterLink>
      </template>
    </div>
  </div>
</template>
