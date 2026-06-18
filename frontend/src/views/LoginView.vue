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

// TODO: wire up GitHub OAuth when backend endpoint exists
function onGitHub() {
  // TODO: redirect to /auth/github
}
</script>

<template>
  <div class="flex min-h-screen" style="background: var(--surface-base)">
    <!-- Left hero panel — hidden on mobile, visible lg+ -->
    <div
      class="hidden lg:flex lg:w-1/2 flex-col justify-between p-12"
      style="background: var(--surface-base); border-right: 1px solid var(--surface-border)"
    >
      <!-- Top wordmark -->
      <div class="text-sm font-semibold tracking-widest uppercase" style="color: var(--p-text-muted-color)">
        hopeitworks
      </div>

      <!-- Center tagline + graph motif -->
      <div class="flex flex-col gap-10">
        <div>
          <p class="text-5xl font-bold leading-tight" style="color: var(--p-text-color)">Plan anywhere.</p>
          <p class="text-5xl font-bold leading-tight" style="color: var(--p-text-color)">Watch it run.</p>
        </div>

        <!-- Inline SVG graph motif: 3 nodes, 2 edges, pulse on center node -->
        <svg width="160" height="80" viewBox="0 0 160 80" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
          <!-- Edges -->
          <line x1="20" y1="40" x2="80" y2="40" stroke="var(--surface-border)" stroke-width="1.5" stroke-dasharray="4 3" />
          <line x1="80" y1="40" x2="140" y2="40" stroke="var(--surface-border)" stroke-width="1.5" stroke-dasharray="4 3" />
          <!-- Left node -->
          <circle cx="20" cy="40" r="7" fill="var(--surface-overlay)" stroke="var(--p-text-muted-color)" stroke-width="1.5" />
          <!-- Right node -->
          <circle cx="140" cy="40" r="7" fill="var(--surface-overlay)" stroke="var(--p-text-muted-color)" stroke-width="1.5" />
          <!-- Center node (larger, with pulse) -->
          <circle cx="80" cy="40" r="9" fill="var(--surface-overlay)" stroke="var(--p-text-muted-color)" stroke-width="1.5" />
          <!-- Live pulse dot on center node -->
          <circle cx="80" cy="40" r="4" class="live-pulse" />
        </svg>
      </div>

      <!-- Footer -->
      <div class="flex items-center gap-2 text-xs" style="color: var(--p-text-muted-color)">
        <span>© 2026 · runtime online</span>
        <span class="live-pulse-dot" style="font-size: 10px; color: var(--status-running-color)">●</span>
      </div>
    </div>

    <!-- Right panel -->
    <div class="flex flex-1 items-center justify-center p-6 lg:p-12">
      <div
        class="flex w-full max-w-sm flex-col gap-6 rounded-xl p-8"
        style="background: var(--surface-raised)"
      >
        <!-- Mobile wordmark -->
        <p class="text-center text-xs font-semibold tracking-widest uppercase lg:hidden" style="color: var(--p-text-muted-color)">
          hopeitworks
        </p>

        <h1 class="text-2xl font-bold" style="color: var(--p-text-color)">Sign in</h1>

        <Message v-if="route.query.reset === 'success'" severity="success" :closable="false">
          Password reset successfully. Please sign in.
        </Message>

        <form class="flex flex-col gap-4" @submit.prevent="onSubmit">
          <div class="flex flex-col gap-1">
            <label for="email" class="text-sm font-medium" style="color: var(--p-text-color)">Email</label>
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

          <div class="flex flex-col gap-1">
            <div class="flex items-center justify-between">
              <label for="password" class="text-sm font-medium" style="color: var(--p-text-color)">Password</label>
              <RouterLink
                to="/forgot-password"
                class="text-xs"
                style="color: var(--p-primary-400)"
              >
                Forgot password?
              </RouterLink>
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
            <small v-if="passwordError" style="color: var(--p-red-400)">{{ passwordError }}</small>
          </div>

          <Button type="submit" label="Sign in" :loading="loading" :disabled="loading" class="w-full mt-1" />

          <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
        </form>

        <div class="flex items-center gap-3">
          <div class="flex-1 h-px" style="background: var(--surface-border)" />
          <span class="text-xs" style="color: var(--p-text-muted-color)">or</span>
          <div class="flex-1 h-px" style="background: var(--surface-border)" />
        </div>

        <!-- TODO: wire up backend /auth/github endpoint -->
        <Button
          type="button"
          label="Continue with GitHub"
          icon="pi pi-github"
          outlined
          class="w-full"
          @click="onGitHub"
        />

        <p class="text-center text-xs" style="color: var(--p-text-muted-color)">
          No account? Ask your workspace admin.
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* SVG center node pulse */
.live-pulse {
  fill: var(--status-running-color);
  animation: node-pulse 2s ease-in-out infinite;
}

/* Footer dot pulse */
.live-pulse-dot {
  animation: node-pulse 2s ease-in-out infinite;
}

@keyframes node-pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.4;
    transform: scale(0.85);
  }
}
</style>
