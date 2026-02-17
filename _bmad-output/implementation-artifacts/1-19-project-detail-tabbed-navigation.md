# Story 1.19: [FRONT] Project Detail — Tabbed Navigation Hub

Status: ready-for-dev

## Story

As a project user, I want a project detail page with tabbed navigation, So that I can access Board, Pipeline Config, and Prompt Templates from within a project context.

## Context

Wave 5 delivered Board (2-4), Pipeline Config (6-4), and Prompt Templates (6-5) as standalone route pages under `/projects/:id/board`, `/projects/:id/pipeline`, and `/projects/:id/templates`. However, `ProjectDetailView.vue` is still a stub and there is no navigation between these sub-pages. This story replaces the stub with a tabbed layout that serves as the project navigation hub.

## Acceptance Criteria (BDD)

**AC1: Project detail shows tabbed navigation**
- **Given** I navigate to `/projects/:id`
- **When** the page loads
- **Then** I see the project name as page title and a horizontal tab bar with tabs: Overview, Board, Pipeline, Templates
- **And** the Overview tab is selected by default

**AC2: Tabs route to existing sub-pages**
- **Given** I am on the project detail page
- **When** I click the "Board" tab
- **Then** I navigate to `/projects/:id/board` and the Board tab becomes active
- **When** I click the "Pipeline" tab
- **Then** I navigate to `/projects/:id/pipeline` and the Pipeline tab becomes active
- **When** I click the "Templates" tab
- **Then** I navigate to `/projects/:id/templates` and the Templates tab becomes active

**AC3: Active tab reflects current route**
- **Given** I navigate directly to `/projects/:id/board`
- **When** the page loads
- **Then** the "Board" tab is highlighted as active
- **And** the BoardView content is displayed

**AC4: Overview tab shows project summary**
- **Given** I am on `/projects/:id` (Overview tab)
- **When** the page loads
- **Then** I see basic project info: name, description, created date
- **And** quick stats if available (epic count, story count) or a simple welcome message

**AC5: Project name in header with back navigation**
- **Given** I am on any project sub-page
- **When** I look at the page header
- **Then** I see the project name and a back link/breadcrumb to the projects list

**AC6: Sidebar shows project context navigation**
- **Given** I am within a project (any sub-route)
- **When** I look at the sidebar
- **Then** the "Projects" nav item is highlighted as active

## Technical Notes

- Use PrimeVue `TabMenu` component for the tab bar
- Use Vue Router nested routes — `ProjectDetailView` becomes a layout with `<router-view>` for child content
- Refactor existing routes: Board, Pipeline, Templates become children of `/projects/:id`
- The project data (name, description) can be fetched from the projects store or a new `useProject` composable
- Keep existing BoardView, PipelineConfigView, PromptTemplatesView as-is — they become the child route components

## Tasks / Subtasks

- [ ] [FRONT] Task 1: Create useProject composable
  - [ ] Fetch single project by ID from API
  - [ ] Use useAsyncAction pattern
  - [ ] Return project, isLoading, error

- [ ] [FRONT] Task 2: Refactor router — nest sub-routes under project detail
  - [ ] Make `/projects/:id` the parent route with ProjectDetailView as layout
  - [ ] Move board, pipeline, templates as children routes
  - [ ] Add default redirect from `/projects/:id` to overview child
  - [ ] Create ProjectOverview.vue for the default tab content

- [ ] [FRONT] Task 3: Implement ProjectDetailView as tabbed layout
  - [ ] Fetch project data on mount via useProject
  - [ ] Render project name in header with breadcrumb back to /projects
  - [ ] Render PrimeVue TabMenu with Overview, Board, Pipeline, Templates
  - [ ] Bind TabMenu to router (active tab = current route)
  - [ ] Render `<router-view>` below tabs for child content

- [ ] [FRONT] Task 4: Create ProjectOverview.vue
  - [ ] Display project name, description, created date
  - [ ] Show basic stats or welcome message
  - [ ] Keep it simple — this is the landing tab

- [ ] [FRONT] Task 5: Update AppSidebar project highlight
  - [ ] Ensure "Projects" nav item is active when on any /projects/* route

- [ ] [FRONT] Task 6: Write unit tests
  - [ ] Test useProject composable
  - [ ] Test tab routing logic

- [ ] [FRONT] Task 7: Write E2E test
  - [ ] Navigate to project → see tabs
  - [ ] Click each tab → correct content shown
  - [ ] Direct URL to sub-page → correct tab active

## Dependencies

- Depends on: 2-4 (BoardView), 6-4 (PipelineConfigView), 6-5 (PromptTemplatesView) — all done in wave 5
- No API changes needed — uses existing project GET endpoint

## Estimation

- Complexity: Small-Medium
- Scope: Frontend only
- Files: ~5-6 files (router refactor + new components + tests)
