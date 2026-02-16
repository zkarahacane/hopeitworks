import { createRouter, createWebHistory } from 'vue-router'
import TestView from '@/views/TestView.vue'
import { setupAuthGuard } from './guards'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      name: 'test',
      component: TestView,
      meta: { requiresAuth: true },
    },
  ],
})

setupAuthGuard(router)

export default router
