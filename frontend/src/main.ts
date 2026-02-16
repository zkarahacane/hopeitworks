import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import App from './App.vue'
import router from './router'
import { setupAuthGuard } from './router/guards'
import { HopeTheme } from '@/theme'

const app = createApp(App)

app.use(createPinia())
setupAuthGuard(router)
app.use(router)
app.use(PrimeVue, {
  theme: {
    preset: HopeTheme,
    options: {
      darkModeSelector: '.dark',
      cssLayer: {
        name: 'primevue',
        order: 'tailwind-base, primevue, tailwind-utilities',
      },
    },
  },
})

app.mount('#app')
