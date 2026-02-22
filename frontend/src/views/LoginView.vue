<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { useForm, useField } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { z } from 'zod'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const route = useRoute()
const { login, loading, error } = useAuth()

const loginSchema = toTypedSchema(
  z.object({
    email: z.string().min(1, 'Email is required').email('Invalid email format'),
    password: z.string().min(1, 'Password is required'),
  }),
)

const { handleSubmit } = useForm({ validationSchema: loginSchema })

const { value: email, errorMessage: emailError } = useField<string>('email')
const { value: password, errorMessage: passwordError } = useField<string>('password')

const onSubmit = handleSubmit(async (values) => {
  const success = await login(values.email, values.password)
  if (success) {
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  }
})
</script>

<template>
  <div class="flex min-h-screen items-center justify-center p-4">
    <div class="flex w-full max-w-md flex-col gap-6">
      <h1 class="text-center text-3xl font-bold">hopeitworks</h1>

      <Message v-if="route.query.reset === 'success'" severity="success" :closable="false">
        Password reset successfully. Please sign in.
      </Message>

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

        <div class="flex flex-col gap-1">
          <div class="flex items-center justify-between">
            <label for="password" class="text-sm font-medium">Password</label>
            <RouterLink to="/forgot-password" class="text-sm">Forgot password?</RouterLink>
          </div>
          <Password
            inputId="password"
            v-model="password"
            :feedback="false"
            toggle-mask
            :invalid="!!passwordError"
            input-class="w-full"
            class="w-full"
          />
          <small v-if="passwordError" class="text-red-500">{{ passwordError }}</small>
        </div>

        <Button type="submit" label="Sign In" :loading="loading" :disabled="loading" class="mt-2" />

        <Message v-if="error" severity="error" :closable="false">
          {{ error }}
        </Message>
      </form>
    </div>
  </div>
</template>
