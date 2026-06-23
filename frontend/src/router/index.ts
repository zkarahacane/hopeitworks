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
import ProbeHaltsView from '@/views/ProbeHaltsView.vue'
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
      meta: { requiresAuth: false },
    },
    {
      path: '/forgot-password',
      name: 'forgot-password',
      component: ForgotPasswordView,
      meta: { requiresAuth: false },
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: ResetPasswordView,
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
          path: 'runs',
          name: 'project-runs',
          component: () => import('@/views/ProjectRunsView.vue'),
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
          path: 'agents',
          name: 'project-agents',
          component: AgentListView,
        },
        {
          path: 'templates',
          redirect: { name: 'project-agents' },
        },
        {
          path: 'environment',
          name: 'project-environment',
          component: () => import('@/views/ProjectEnvironmentView.vue'),
        },
        {
          path: 'costs',
          name: 'project-costs',
          component: () => import('@/views/CostDashboardView.vue'),
        },
        {
          path: 'agents/new',
          name: 'agent-create',
          component: () => import('@/views/AgentEditorView.vue'),
          meta: { requiresAdmin: true },
        },
        {
          path: 'agents/:agentId',
          name: 'agent-editor',
          component: () => import('@/views/AgentEditorView.vue'),
        },
        {
          path: 'runs/:runId/approve/:stepId',
          name: 'hitl-approve',
          component: () => import('@/views/HITLApprovalView.vue'),
        },
        {
          path: 'settings',
          name: 'project-settings',
          component: () => import('@/views/ProjectSettingsView.vue'),
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
      path: '/halts',
      name: 'halts',
      component: ProbeHaltsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/runs',
      name: 'runs',
      component: RunsView,
      meta: { requiresAuth: true },
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/admin/UserManagementView.vue'),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('@/views/ProfileView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/settings',
      redirect: { name: 'profile' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundView,
      meta: { requiresAuth: false },
    },
  ],
})

setupAuthGuard(router)
setupAdminGuard(router)

export default router
