import { createRouter, createWebHistory } from 'vue-router'
import TestView from '@/views/TestView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'dashboard',
      component: TestView,
    },
    {
      path: '/projects',
      name: 'projects',
      component: TestView,
    },
    {
      path: '/runs',
      name: 'runs',
      component: TestView,
    },
    {
      path: '/settings',
      name: 'settings',
      component: TestView,
    },
  ],
})

export default router
