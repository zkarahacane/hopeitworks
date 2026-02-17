import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '@/views/LoginView.vue'
import DashboardView from '@/views/DashboardView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import ProjectDetailView from '@/views/ProjectDetailView.vue'
import RunDetailView from '@/views/RunDetailView.vue'
import StoryDetailView from '@/views/StoryDetailView.vue'
import ApprovalsView from '@/views/ApprovalsView.vue'
import PipelineConfigView from '@/views/PipelineConfigView.vue'
import PromptTemplatesView from '@/views/PromptTemplatesView.vue'
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
      path: '/projects/:id/templates',
      name: 'project-templates',
      component: PromptTemplatesView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects/:projectId/stories/:storyId',
      name: 'story-detail',
      component: StoryDetailView,
      meta: { requiresAuth: true },
    },
    {
      path: '/projects/:id/pipeline',
      name: 'project-pipeline',
      component: PipelineConfigView,
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
