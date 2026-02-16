import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '@/views/LoginView.vue'
import DashboardView from '@/views/DashboardView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import ProjectDetailView from '@/views/ProjectDetailView.vue'
import RunDetailView from '@/views/RunDetailView.vue'
import ApprovalsView from '@/views/ApprovalsView.vue'
import StoryBoardView from '@/views/StoryBoardView.vue'
import { setupAuthGuard, setupAdminGuard } from './guards'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { requiresAuth: false },
    },
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects',
      name: 'projects',
      component: ProjectsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects/:id',
      name: 'project-detail',
      component: ProjectDetailView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects/:id/board',
      name: 'story-board',
      component: StoryBoardView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects/:id/epics/:epicId',
      name: 'epic-detail',
      component: () => import('@/views/ProjectDetailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/runs/:id',
      name: 'run-detail',
      component: RunDetailView,
      meta: { requiresAuth: true },
    },
    {
      path: '/approvals',
      name: 'approvals',
      component: ApprovalsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/admin/UserManagementView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
  ],
})

setupAuthGuard(router)
setupAdminGuard(router)

export default router
