<script setup lang="ts">
import { ref } from 'vue'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAuth } from '@/composables/useAuth'

const { forgotPassword, loading } = useAuth()
const submitted = ref(false)

const forgotSchema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
  }),
)

const { handleSubmit } = useForm({ validationSchema: forgotSchema })
const { value: email, errorMessage: emailError } = useField<string>('email')

const onSubmit = handleSubmit(async (values) => {
  await forgotPassword(values.email)
  submitted.value = true
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
        <h1 class="text-2xl font-bold" style="color: var(--p-surface-0)">Reset your password</h1>
      </div>

      <template v-if="!submitted">
        <p class="text-sm" style="color: var(--p-surface-400)">
          Enter your email and we'll send you a reset link.
        </p>

        <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
          <div class="flex flex-col gap-1">
            <label for="email" class="text-sm font-medium" style="color: var(--p-surface-200)">Email</label>
            <InputText
              id="email"
              v-model="email"
              type="email"
              placeholder="you@example.com"
              :invalid="!!emailError"
              class="w-full"
            />
            <small v-if="emailError" style="color: var(--p-red-400)">{{ emailError }}</small>
          </div>

          <Button
            type="submit"
            label="Send reset link"
            :loading="loading"
            :disabled="loading"
            class="w-full mt-1"
          />
        </form>
      </template>

      <Message v-if="submitted" severity="success" :closable="false">
        Check your email. If an account exists for that email, you will receive a reset link shortly.
      </Message>

      <RouterLink
        to="/login"
        class="text-sm"
        style="color: var(--p-primary-400)"
      >
        &larr; Back to login
      </RouterLink>
    </div>
  </div>
</template>
