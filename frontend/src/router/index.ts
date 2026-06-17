import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '@/views/LoginView.vue'
import ForgotPasswordView from '@/views/ForgotPasswordView.vue'
import ResetPasswordView from '@/views/ResetPasswordView.vue'
import DashboardView from '@/views/DashboardView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import ProjectDetailView from '@/views/ProjectDetailView.vue'
import RunDetailView from '@/views/RunDetailView.vue'
import StoryDetailView from '@/views/StoryDetailView.vue'
import ApprovalsView from '@/views/ApprovalsView.vue'
import RunsView from '@/views/RunsView.vue'
import PipelineConfigView from '@/views/PipelineConfigView.vue'
import AgentListView from '@/views/AgentListView.vue'
import BoardView from '@/views/BoardView.vue'
import NotFoundView from '@/views/NotFoundView.vue'
import ProjectOverview from '@/features/projects/ProjectOverview.vue'
import { setupAuthGuard, setupAdminGuard } from './guards'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: LoginView,
      meta: { requiresAuth: false, theme: 'dark' },
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: ForgotPasswordView,
      meta: { requiresAuth: false, theme: 'dark' },
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: ResetPasswordView,
      meta: { requiresAuth: false, theme: 'dark' },
    },
    {
      path: '/',
      name: 'dashboard',
      component: DashboardView,
      meta: { requiresAuth: true, theme: 'dark' },
    },
    {
      path: '/projects',
      name: 'projects',
      component: ProjectsView,
      meta: { requiresAuth: true, theme: 'light' },
    },
    {
      path: '/projects/:id',
      component: ProjectDetailView,
      meta: { requiresAuth: true, theme: 'light' },
      children: [
        {
          path: '',
          name: 'project-overview',
          component: ProjectOverview,
          meta: { theme: 'light' },
        },
        {
          path: 'board',
          name: 'project-board',
          component: BoardView,
          meta: { theme: 'light' },
        },
        {
          path: 'runs',
          name: 'project-runs',
          component: () => import('@/views/ProjectRunsView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'epics/:epicId',
          name: 'epic-detail',
          component: () => import('@/views/EpicDetailView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'epics/:epicId/dag',
          name: 'epic-dag',
          component: () => import('@/views/EpicDagView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'epic-runs/:epicRunId',
          name: 'epic-run-monitor',
          component: () => import('@/views/EpicRunView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'pipeline',
          name: 'project-pipeline',
          component: PipelineConfigView,
          meta: { theme: 'light' },
        },
        {
          path: 'agents',
          name: 'project-agents',
          component: AgentListView,
          meta: { theme: 'light' },
        },
        {
          path: 'templates',
          redirect: { name: 'project-agents' },
        },
        {
          path: 'costs',
          name: 'project-costs',
          component: () => import('@/views/CostDashboardView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'agents/new',
          name: 'agent-create',
          component: () => import('@/views/AgentEditorView.vue'),
          meta: { requiresAdmin: true, theme: 'light' },
        },
        {
          path: 'agents/:agentId',
          name: 'agent-editor',
          component: () => import('@/views/AgentEditorView.vue'),
          meta: { theme: 'light' },
        },
        {
          path: 'runs/:runId/approve/:stepId',
          name: 'hitl-approve',
          component: () => import('@/views/HITLApprovalView.vue'),
          meta: { theme: 'dark' },
        },
        {
          path: 'settings',
          name: 'project-settings',
          component: () => import('@/views/ProjectSettingsView.vue'),
          meta: { theme: 'light' },
        },
        {
          path: 'settings/notifications',
          name: 'project-notifications',
          component: () => import('@/views/NotificationSettingsView.vue'),
          meta: { theme: 'light' },
        },
      ],
    },
    {
      path: '/projects/:projectId/stories/:storyId',
      name: 'story-detail',
      component: StoryDetailView,
      meta: { requiresAuth: true, theme: 'dark' },
    },
    {
      path: '/runs/:id',
      name: 'run-detail',
      component: RunDetailView,
      meta: { requiresAuth: true, theme: 'dark' },
    },
    {
      path: '/approvals',
      name: 'approvals',
      component: ApprovalsView,
      meta: { requiresAuth: true, theme: 'dark' },
    },
    {
      path: '/runs',
      name: 'runs',
      component: RunsView,
      meta: { requiresAuth: true, theme: 'dark' },
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/admin/UserManagementView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true, theme: 'light' },
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { requiresAuth: true, theme: 'light' },
    },
    {
      path: '/settings',
      redirect: { name: 'profile' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundView,
      meta: { requiresAuth: false, theme: 'dark' },
    },
  ],
})

setupAuthGuard(router)
setupAdminGuard(router)

export default router
