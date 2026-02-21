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
  <div class="flex min-h-screen items-center justify-center p-4">
    <div class="flex w-full max-w-md flex-col gap-6">
      <h1 class="text-center text-3xl font-bold">hopeitworks</h1>

      <template v-if="!submitted">
        <h2 class="text-center text-xl">Reset your password</h2>

        <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
          <div class="flex flex-col gap-1">
            <label for="email" class="text-sm font-medium">Email</label>
            <InputText
              id="email"
              v-model="email"
              type="email"
              placeholder="you@example.com"
              :invalid="!!emailError"
            />
            <small v-if="emailError" class="text-red-500">{{ emailError }}</small>
          </div>

          <Button
            type="submit"
            label="Send reset link"
            :loading="loading"
            :disabled="loading"
            class="mt-2"
          />
        </form>
      </template>

      <Message v-if="submitted" severity="success" :closable="false">
        Check your email. If an account exists for that email, you will receive a reset link
        shortly.
      </Message>

      <RouterLink to="/login" class="text-sm">&larr; Back to login</RouterLink>
    </div>
  </div>
</template>
