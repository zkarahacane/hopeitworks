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
import BoardView from '@/views/BoardView.vue'
import ProjectOverview from '@/features/projects/ProjectOverview.vue'
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
      component: ProjectDetailView,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'project-overview',
          component: ProjectOverview,
        },
        {
          path: 'board',
          name: 'project-board',
          component: BoardView,
        },
        {
          path: 'epics/:epicId',
          name: 'epic-detail',
          component: () => import('@/views/EpicDetailView.vue'),
        },
        {
          path: 'epics/:epicId/dag',
          name: 'epic-dag',
          component: () => import('@/views/EpicDagView.vue'),
        },
        {
          path: 'epic-runs/:epicRunId',
          name: 'epic-run-monitor',
          component: () => import('@/views/EpicRunView.vue'),
        },
        {
          path: 'pipeline',
          name: 'project-pipeline',
          component: PipelineConfigView,
        },
        {
          path: 'templates',
          name: 'project-templates',
          component: PromptTemplatesView,
        },
        {
          path: 'costs',
          name: 'project-costs',
          component: () => import('@/views/CostDashboardView.vue'),
        },
        {
          path: 'templates/new',
          name: 'template-create',
          component: () => import('@/views/TemplateEditorView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'templates/:templateId',
          name: 'template-editor',
          component: () => import('@/views/TemplateEditorView.vue'),
        },
        {
          path: 'runs/:runId/approve/:stepId',
          name: 'hitl-approve',
          component: () => import('@/views/HITLApprovalView.vue'),
        },
        {
          path: 'settings/notifications',
          name: 'project-notifications',
          component: () => import('@/views/NotificationSettingsView.vue'),
        },
      ],
    },
    {
      path: '/projects/:projectId/stories/:storyId',
      name: 'story-detail',
      component: StoryDetailView,
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
