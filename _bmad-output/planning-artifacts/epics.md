---
stepsCompleted:
  - step-01-validate-prerequisites
  - step-02-design-epics
  - step-03-create-stories
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
userNotes:
  - Stories must be split between frontend and backend
  - Stories should be small (max ~5KB, absolute max ~10KB if indivisible)
  - CLAUDE.md files (base + backend + frontend + project) are deliverables
  - Development will use Docker containers with Claude Code agents before MVP is ready
  - Stories must be directly exploitable by a Claude agent in a container
---

# hopeitworks - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for hopeitworks, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: Admin can create a new project and connect it to a Git repository
FR2: Admin can configure a project's Git provider, agent runtime, default model, and budget limits
FR3: Users can view the list of projects they have access to
FR4: Users can view stories grouped by epic within a project
FR5: Users can view the status of each story (backlog, running, done, failed)
FR6: Users can view story details (objectives, target files, dependencies, acceptance criteria)
FR7: Users can import stories from markdown files in the repository
FR8: System can parse story frontmatter (key, epic, depends_on, scope, status)
FR9: Users can launch a single story run
FR10: Users can launch all stories of an epic as a batch run
FR11: System can build a DAG from story dependencies and file conflicts
FR12: System can execute stories in parallel groups based on the DAG
FR13: System can execute pipeline steps sequentially for each story (configurable step chain)
FR14: System can poll CI results via the GitProvider instead of launching an agent
FR15: System can perform incremental retry when CI fails (agent receives error context + existing diff)
FR16: System can fallback to full retry after 2 failed incremental attempts
FR17: Users can pause a running story or epic
FR18: Users can resume a paused story or epic
FR19: Admin can define pipeline steps per project (add, remove, reorder)
FR20: Admin can configure each step's agent, model, prompt template, auto/hitl gate, and retry policy
FR21: Users can view the current pipeline configuration for a project
FR22: System stores pipeline configuration as YAML per project
FR23: System can pause execution at configured HITL gates and notify the user
FR24: Users can approve a pending HITL request via web UI
FR25: Users can reject a pending HITL request with a reason via web UI
FR26: Users can approve or reject HITL requests via CLI
FR27: System can create an isolated Docker container per agent run
FR28: System can inject CLAUDE.md, secrets, and configuration as environment variables into containers
FR29: System can stream agent output (NDJSON) from containers in real-time
FR30: System can clean up containers after run completion or failure
FR31: System can enforce a hard timeout per container (configurable, default 30min)
FR32: System can apply circuit breaker (stop after N consecutive failures)
FR33: Admin can view all prompt templates (implement, retry, review, merge, custom)
FR34: Admin can edit prompt templates via the web UI
FR35: System can render prompt templates with story context variables (Handlebars)
FR36: Admin can create custom prompt templates for custom pipeline steps
FR37: Users can view live agent logs via SSE streaming in the web UI
FR38: Users can view live agent logs via CLI
FR39: Users can view run progress (current step, substep) in real-time
FR40: System can send notifications (Discord, webhook) on run events (success, failure, HITL pending)
FR41: System can track token usage and cost per step, run, and story
FR42: Users can view cost breakdown per run and per story
FR43: Admin can set budget limits per story and per project
FR44: System can halt execution when budget limit is exceeded
FR45: Users can authenticate via JWT
FR46: Admin can create and manage user accounts
FR47: Admin has full access to all projects and configurations
FR48: Users can only access projects they are assigned to
FR49: System can clone a repository, create branches, and push commits via the GitProvider interface
FR50: System can create pull/merge requests via the GitProvider interface
FR51: System can poll CI status via the GitProvider interface
FR52: System can merge pull/merge requests via the GitProvider interface
FR53: System includes a reference test project (todo app) with build, seed SQL, CI pipeline, and E2E tests for pipeline validation

### NonFunctional Requirements

NFR1: SSE event latency from container to browser < 1 second
NFR2: REST API response time < 500ms for CRUD operations
NFR3: DAG computation for 50+ stories < 2 seconds
NFR4: Container startup time budget < 30 seconds
NFR5: Agent containers fully isolated via Docker — no host filesystem access
NFR6: Docker socket access restricted via docker-socket-proxy (allowlisted operations only)
NFR7: API keys and Git tokens injected as environment variables, never persisted in database or logs
NFR8: slog output scrubs sensitive values (tokens, keys) from structured logs
NFR9: JWT tokens signed with configurable secret and expiration
NFR10: API crash must not terminate running agent containers
NFR11: All run/step state persisted in Postgres — resumable after API restart
NFR12: Orphan container cleanup on API startup (garbage collector)
NFR13: Circuit breaker halts execution after N consecutive failures (configurable, default 3)
NFR14: Hard timeout per container enforced (configurable, default 30min)
NFR15: Reference test project maintains permanently green CI baseline for pipeline validation

### Additional Requirements

**Architecture — Scaffolding & Stack:**
- Manual Go project scaffolding (no monolithic starter) — code-gen-first (oapi-codegen, sqlc, go-wire)
- Frontend scaffolding via create-vue with TypeScript, Vitest, ESLint, Prettier
- Backend: Go + chi router + pgx/v5 + sqlc + oapi-codegen
- Frontend: Vue 3 Composition API + PrimeVue 4 (Aura unstyled) + Tailwind CSS v4 + Pinia + Vue Router
- Postgres single data store + event bus (LISTEN/NOTIFY via pgxlisten)
- River (Postgres-based Go job queue) for transactional enqueue
- go-wire compile-time DI
- golang-migrate for SQL migrations
- golang-jwt/jwt/v5 for auth
- slog structured JSON logging (LGTM-compatible)
- testcontainers-go for integration tests

**Architecture — Infrastructure & Deployment:**
- docker-compose.yml in deploy/ as entry point
- docker-compose.prod.yml for production overrides
- docker-compose.test.yml for functional test stack
- docker-socket-proxy with allowlisted operations
- Network isolation: dedicated Docker network for agent containers
- No host filesystem access for agents, no privileged mode

**Architecture — Database & Migrations:**
- Schema: projects, epics, stories, runs, run_steps, events, users, prompt_templates, pipeline_configs
- All tables include project_id FK for multi-tenancy readiness
- events table append-only for event log (source for LISTEN/NOTIFY triggers)
- Migrations: numbered SQL files (000001_init_schema.up.sql / .down.sql), auto-run at startup
- sqlc generates type-safe Go code from backend/queries/*.sql

**Architecture — API & Contract:**
- OpenAPI 3.0 spec as single source of truth in api/openapi.yaml
- Contract-first: spec written BEFORE implementation
- oapi-codegen generates chi handlers + types
- openapi-typescript + openapi-fetch generates TypeScript typed client
- DomainError pattern for standardized error handling
- Prefix: /api/v1/

**Architecture — Auth & Security:**
- JWT in httpOnly secure cookie
- CORS configurable (dev localhost:*, prod specific domain(s))
- Secrets via environment variables only (.env in dev)
- slog ScrubHandler strips tokens/keys from logs
- Docker socket security: allowlisted operations, no host FS access, network isolation

**Architecture — Event Bus & Real-Time:**
- pgxlisten wrapper around Postgres LISTEN/NOTIFY with auto-reconnection
- Dedicated connection for event listening (separate from query pool)
- SSE endpoint: GET /api/v1/events/stream?project_id={id}
- Client reconnection via Last-Event-ID header
- Event format: {entity}.{action} dot-notation with JSON snake_case payload

**Architecture — Pipeline & Actions:**
- Actions: agent_run, ci_poll, git_create_branch, git_create_pr, git_merge, hitl_gate, notify, script
- ActionRegistry central mapping
- Retry: incremental (diff + error → agent fixes) after 2 failures → full retry
- State machine: pending → running → completed|failed|cancelled

**Architecture — Scheduler & DAG:**
- DAG builder for epic runs (parallel execution)
- Dependency resolution: explicit (depends_on) + implicit (provides/requires matching)
- Topological sort for execution order
- Max concurrency configurable per project (max_parallel_runs)

**Architecture — Agent Runtime:**
- Container lifecycle: create → inject env → stream logs → monitor → cleanup
- Dockerfile.base: Claude Code + git + gh CLI + common tools
- entrypoint.sh: clone repo → checkout branch → inject CLAUDE.md (composed) → run Claude Code
- NDJSON stream to stdout
- CLAUDE.md composed runtime: base.md + backend.md OR frontend.md + project.md
- Prompt rendering: Handlebars templates (implement.hbs, implement-retry.hbs, review.hbs, merge-conflict.hbs)

**Architecture — Git Provider:**
- gh CLI via CommandRunner interface (testable)
- Branch naming: feat/S-{key}-{slug}, fix/S-{key}-{slug}
- Conventional commits, squash merge by default, delete branch after merge

**Architecture — Notifications:**
- Dispatcher pattern: event → routing rules → channels
- MVP: Discord webhook + SSE (built-in)

**Architecture — Transaction Management:**
- Transactor pattern: transaction in context
- Repository methods extract tx from context, fallback to pool

**Architecture — Testing Strategy (4 levels):**
- Unit: Go testing + table-driven, Vitest + Vue Test Utils
- Integration: testcontainers-go + real Postgres, Vitest + MSW
- E2E: Playwright 10 critical flows
- Pipeline Functional: test-project/ (todo app) with full flow validation

**Architecture — Frontend Architecture:**
- Hybrid structure: ui/ (shared atomic) + features/ (by business domain)
- Composition API exclusively
- Components are visual assemblers — zero business logic in .vue files
- Composables are the logic layer
- useAsyncAction pattern wraps every async operation
- Props down, events up
- Stores handle SSE dispatch — useSSE composable dispatches events to Pinia stores

**Architecture — Code-Gen Philosophy:**
- API handlers: openapi.yaml → oapi-codegen → chi server interfaces + types
- API client: openapi.yaml → openapi-typescript + openapi-fetch → TypeScript typed fetch client
- Database: queries/*.sql → sqlc → type-safe Go functions
- DI wiring: wire.go → go-wire → wire_gen.go

**Architecture — Project Boundaries:**
- Backend agents NEVER touch frontend/
- Frontend agents NEVER touch backend/
- api/openapi.yaml shared contract — changes require coordination
- Each side has own CLAUDE.md scoping agent behavior

**Architecture — Deferred to Phase 2:**
- Budget enforcement (FR43-FR44) — MVP = measurement only
- CLI client for log viewing (FR38) — curl suffices for MVP
- Visual pipeline editor — MVP = YAML editing
- Rate limiting, caching, API versioning beyond v1, monitoring dashboards, agent builder UI

**CLAUDE.md Files (Deliverables):**
- CLAUDE.md base (project-level rules, conventions, git workflow)
- CLAUDE.md backend (Go-specific rules, architecture patterns, testing conventions)
- CLAUDE.md frontend (Vue/TypeScript-specific rules, PrimeVue patterns, composable conventions)
- CLAUDE.md project (project-specific context, current state, known issues)

**UX — Responsive Design:**
- Desktop-first (1440px+ primary), progressive reduction to tablet/mobile
- Mobile (<1024px) is monitoring/approving only, NOT configuring
- Container queries for component adaptation
- Main content max-width: 1600px

**UX — Accessibility (WCAG 2.1 AA):**
- Color contrast ratios meet AA standards
- Triple-channel status communication: Color + Icon + Text
- Full keyboard navigation with context-aware shortcuts
- ARIA patterns for all interactive components
- Screen reader support (VoiceOver, NVDA)
- prefers-reduced-motion respected

**UX — Components & Pages:**
- 16 screens/pages/views
- 14 custom components (StoryStatusCard, PipelineTimeline, LogViewer, CommandPalette, etc.)
- 6 layout components (Header, Sidebar, Status Bar, etc.)
- 27 PrimeVue components used
- 6 third-party libraries (@vue-flow/core, diff2html, fuse.js, ansi-to-html, monaco-editor, vue-virtual-scroller)

**UX — Design System:**
- PrimeVue 4 unstyled + Aura preset + Tailwind v4 (layout only)
- CSS Layers: tailwind-base, primevue, tailwind-utilities
- Dark mode first, 3-level design token architecture
- System fonts (Inter, JetBrains Mono)
- Contextual density model (compact for scanning, comfortable for decisions)

**UX — User Flows:**
- Launch & Monitor Epic (power user flow)
- HITL Approval flow (1-click from notification)
- First Run Onboarding (new user flow)
- Failure Investigation flow

**UX — Form Validation & Feedback:**
- Validate on blur, not on type
- Smart defaults strategy (user only fills unique fields)
- Toast notification system (success/info/warning/error/action)
- Button hierarchy (Primary/Secondary/Ghost/Danger/Danger Ghost)

### FR Coverage Map

FR1: Epic 1 - Admin can create a new project and connect it to a Git repository
FR2: Epic 1 - Admin can configure a project's Git provider, agent runtime, default model, and budget limits
FR3: Epic 2 - Users can view the list of projects they have access to
FR4: Epic 2 - Users can view stories grouped by epic within a project
FR5: Epic 2 - Users can view the status of each story (backlog, running, done, failed)
FR6: Epic 2 - Users can view story details (objectives, target files, dependencies, acceptance criteria)
FR7: Epic 2 - Users can import stories from markdown files in the repository
FR8: Epic 2 - System can parse story frontmatter (key, epic, depends_on, scope, status)
FR9: Epic 3 - Users can launch a single story run
FR10: Epic 7 - Users can launch all stories of an epic as a batch run
FR11: Epic 7 - System can build a DAG from story dependencies and file conflicts
FR12: Epic 7 - System can execute stories in parallel groups based on the DAG
FR13: Epic 3 - System can execute pipeline steps sequentially for each story
FR14: Epic 8 - System can poll CI results via the GitProvider
FR15: Epic 8 - System can perform incremental retry when CI fails
FR16: Epic 8 - System can fallback to full retry after 2 failed incremental attempts
FR17: Epic 7 - Users can pause a running story or epic
FR18: Epic 7 - Users can resume a paused story or epic
FR19: Epic 6 - Admin can define pipeline steps per project
FR20: Epic 6 - Admin can configure each step's agent, model, prompt template, auto/hitl gate, and retry policy
FR21: Epic 6 - Users can view the current pipeline configuration for a project
FR22: Epic 6 - System stores pipeline configuration as YAML per project
FR23: Epic 5 - System can pause execution at configured HITL gates and notify the user
FR24: Epic 5 - Users can approve a pending HITL request via web UI
FR25: Epic 5 - Users can reject a pending HITL request with a reason via web UI
FR26: Epic 5 - Users can approve or reject HITL requests via CLI — **[DEFERRED - Phase 2]**
FR27: Epic 3 - System can create an isolated Docker container per agent run
FR28: Epic 3 - System can inject CLAUDE.md, secrets, and configuration as environment variables
FR29: Epic 3 - System can stream agent output (NDJSON) from containers in real-time
FR30: Epic 3 - System can clean up containers after run completion or failure
FR31: Epic 3 - System can enforce a hard timeout per container
FR32: Epic 8 - System can apply circuit breaker (stop after N consecutive failures)
FR33: Epic 6 - Admin can view all prompt templates
FR34: Epic 6 - Admin can edit prompt templates via the web UI
FR35: Epic 6 - System can render prompt templates with story context variables (Handlebars)
FR36: Epic 6 - Admin can create custom prompt templates for custom pipeline steps
FR37: Epic 4 - Users can view live agent logs via SSE streaming in the web UI
FR38: Epic 4 - Users can view live agent logs via CLI — **[DEFERRED - Phase 2]** (curl suffices for MVP)
FR39: Epic 4 - Users can view run progress (current step, substep) in real-time
FR40: Epic 9 - System can send notifications (Discord, webhook) on run events
FR41: Epic 9 - System can track token usage and cost per step, run, and story
FR42: Epic 9 - Users can view cost breakdown per run and per story
FR43: Epic 9 - Admin can set budget limits per story and per project
FR44: Epic 9 - System can halt execution when budget limit is exceeded — **[DEFERRED - Phase 2]** (tracking only in MVP)
FR45: Epic 1 - Users can authenticate via JWT
FR46: Epic 1 - Admin can create and manage user accounts
FR47: Epic 1 - Admin has full access to all projects and configurations
FR48: Epic 1 - Users can only access projects they are assigned to
FR49: Epic 3 - System can clone a repository, create branches, and push commits via GitProvider
FR50: Epic 3 - System can create pull/merge requests via GitProvider
FR51: Epic 8 - System can poll CI status via GitProvider
FR52: Epic 3 - System can merge pull/merge requests via GitProvider
FR53: Epic 10 - System includes a reference test project for pipeline validation

## Epic List

### Epic 1: Project Foundation & Authentication
Admin can create an account, authenticate, and configure a first project connected to a Git repository.
**FRs covered:** FR1, FR2, FR45, FR46, FR47, FR48
**Includes:** Go + Vue scaffolding, CLAUDE.md files, initial DB schema, docker-compose dev, JWT auth, project CRUD, RBAC

### Epic 2: Story Board & Management
Users can view, import, and manage stories organized by epic within a project.
**FRs covered:** FR3, FR4, FR5, FR6, FR7, FR8
**Includes:** Story board UI, markdown import, story detail view, frontmatter parsing, epic grouping

### Epic 3: Single Story Pipeline Execution
User can launch a story that executes automatically: Docker container → agent code → branch → commit → PR → merge.
**FRs covered:** FR9, FR13, FR27, FR28, FR29, FR30, FR31, FR49, FR50, FR52
**Includes:** Agent runtime (Docker), git provider (gh CLI), pipeline execution engine, container lifecycle, NDJSON streaming, event bus (pgxlisten)

### Epic 4: Real-time Monitoring & Live Logs
User can follow execution in real-time with live logs and step progress.
**FRs covered:** FR37, FR38, FR39
**Includes:** SSE streaming, LogViewer component, run progress tracking (event bus in Epic 3)

### Epic 5: HITL Gates & Approval Workflow
User can approve or reject at human-in-the-loop checkpoints, with inline diff and notifications.
**FRs covered:** FR23, FR24, FR25, FR26
**Includes:** Approval UI with DiffViewer, reject with reason, HITL gate action (CLI approval deferred to Phase 2)

### Epic 6: Pipeline Configuration & Prompt Templates
Admin can customize pipeline steps and prompt templates per project.
**FRs covered:** FR19, FR20, FR21, FR22, FR33, FR34, FR35, FR36
**Includes:** Pipeline YAML editor, prompt template CRUD, Handlebars rendering, pipeline config UI

### Epic 7: Epic Batch Execution & DAG Scheduling
User can launch an entire epic with intelligent parallel execution based on story dependencies.
**FRs covered:** FR10, FR11, FR12, FR17, FR18
**Includes:** DAG builder, topological sort, parallel groups, pause/resume, max concurrency

### Epic 8: Retry, CI Polling & Resilience
System handles CI failures intelligently: incremental retry, full retry fallback, circuit breaker.
**FRs covered:** FR14, FR15, FR16, FR32, FR51
**Includes:** CI polling via GitProvider, incremental retry (diff + error context), full retry fallback, circuit breaker

### Epic 9: Cost Tracking & Notifications
User can see costs in real-time and receive notifications on pipeline events.
**FRs covered:** FR40, FR41, FR42, FR43, FR44
**Includes:** Cost tracking per step/run/story, budget display (measurement only MVP), Discord webhooks, notification dispatcher

### Epic 10: Reference Test Project & Validation
System includes a reference project (todo app) to validate the pipeline end-to-end.
**FRs covered:** FR53
**Includes:** Todo app with build, seed SQL, CI pipeline, E2E tests, permanently green baseline

## Epic 1: Project Foundation & Authentication

Admin can create an account, authenticate, and configure a first project connected to a Git repository. Includes Go + Vue scaffolding, CLAUDE.md files, initial DB schema, docker-compose dev, JWT auth, project CRUD, RBAC.

### Story 1.1: [BACK] Go project scaffolding + docker-compose dev stack

As a backend developer,
I want a complete Go project scaffold with a running local development stack,
So that I can start implementing features immediately with a working foundation.

**Acceptance Criteria:**

**Given** a fresh repository clone
**When** I run `go build ./cmd/api`
**Then** the build succeeds and a binary is produced

**Given** the project structure is initialized
**When** I examine the directory layout
**Then** I see cmd/api/main.go, internal/, deploy/, and config.yaml

**Given** docker-compose.yml is configured in deploy/
**When** I run `docker compose -f deploy/docker-compose.yml up`
**Then** Postgres and API containers start successfully

**Given** the API service is running
**When** I send GET /health
**Then** I receive HTTP 200 with status ok

**Given** the API service is running and Postgres is ready
**When** I send GET /ready
**Then** I receive HTTP 200 and readiness check pings database

**Given** config.yaml contains settings
**When** the API starts
**Then** config is loaded from YAML with env var overrides and slog outputs JSON logs

### Story 1.2: [BACK] OpenAPI spec + code-gen pipeline

As a backend developer,
I want a contract-first development workflow with automated code generation,
So that API contracts are the source of truth.

**Acceptance Criteria:**

**Given** api/openapi.yaml exists
**When** I examine the spec
**Then** I see OpenAPI 3.0 with auth, user, and project endpoints defined

**Given** oapi-codegen and sqlc configs exist
**When** I run `make generate`
**Then** Go server interfaces, types, and DB query functions are generated without errors

**Given** api/openapi.yaml is valid
**When** I run openapi-lint validation
**Then** the spec passes with no errors

### Story 1.3: [BACK] Users table + JWT auth API (register, login, auth middleware)

As a user,
I want a users table and JWT authentication,
So that user data is persisted and I can securely access protected endpoints.

**Acceptance Criteria:**

**Given** migration 000001 exists
**When** migrations are applied
**Then** a users table is created with: id (UUID PK), email (unique), password_hash, role (admin/user), created_at, updated_at

**Given** sqlc queries are defined
**When** I run `make generate`
**Then** Go functions for CreateUser, GetUserByEmail, GetUserByID, ListUsers, UpdateUser, DeleteUser are generated

**Given** the API is running
**When** I POST /api/v1/auth/register with valid email and password
**Then** I receive HTTP 201 with user object and password is bcrypt-hashed

**Given** a user exists
**When** I POST /api/v1/auth/login with correct credentials
**Then** I receive HTTP 200 with JWT in httpOnly secure cookie containing user_id, role, exp

**Given** a user exists
**When** I POST /api/v1/auth/login with wrong password
**Then** I receive HTTP 401

**Given** a request includes a valid JWT cookie
**When** auth middleware runs
**Then** user context (id, role) is injected into request context

**Given** a request has no JWT or invalid JWT
**When** auth middleware runs
**Then** HTTP 401 is returned

### Story 1.4: [BACK] User management API (admin CRUD)

As an admin,
I want to manage user accounts,
So that I can control platform access.

**Acceptance Criteria:**

**Given** I am authenticated as admin
**When** I GET /api/v1/users
**Then** I receive paginated list of users

**Given** I am authenticated as non-admin
**When** I GET /api/v1/users
**Then** I receive HTTP 403

**Given** I am admin
**When** I PUT /api/v1/users/{id} with role change
**Then** the user role is updated

**Given** I am admin
**When** I DELETE /api/v1/users/{id}
**Then** the user is deactivated

### Story 1.5: [BACK] Projects table + Project CRUD API

As an admin,
I want a projects table and CRUD API to create and configure projects with Git connections,
So that the platform can orchestrate agents per project.

**Acceptance Criteria:**

**Given** migration 000002 exists
**When** migrations are applied
**Then** a projects table is created with: id (UUID PK), name, repo_url, git_provider, git_token_env, agent_runtime, default_model, max_budget, created_at, updated_at

**Given** sqlc queries are defined
**When** I run `make generate`
**Then** Go functions for CreateProject, GetProject, ListProjects, UpdateProject, DeleteProject are generated

**Given** I am admin
**When** I POST /api/v1/projects with valid payload
**Then** I receive HTTP 201 with created project

**Given** I am admin
**When** I GET /api/v1/projects
**Then** I receive all projects with pagination

**Given** I am a regular user
**When** I GET /api/v1/projects
**Then** I receive only projects I am assigned to

**Given** I am non-admin
**When** I POST/PUT/DELETE /api/v1/projects
**Then** I receive HTTP 403

### Story 1.6: [BACK] RBAC middleware + project_users table

As a platform administrator,
I want role-based access control on all project-scoped endpoints,
So that users only access their assigned projects.

**Acceptance Criteria:**

**Given** migration 000003 exists
**When** migrations are applied
**Then** a project_users table is created with composite PK (project_id, user_id)

**Given** a request to a project-scoped endpoint by an admin
**When** middleware runs
**Then** access is granted without assignment check

**Given** a request by a user assigned to the project
**When** middleware runs
**Then** access is granted

**Given** a request by a user NOT assigned to the project
**When** middleware runs
**Then** HTTP 403 is returned

**Given** I am admin
**When** I POST /api/v1/projects/{id}/users with user_id
**Then** the user is assigned to the project

### Story 1.7: [FRONT] Vue scaffolding + PrimeVue + Tailwind setup

As a developer,
I want a fully configured Vue 3 project with PrimeVue and Tailwind,
So that I can start implementing features with correct theming.

**Acceptance Criteria:**

**Given** frontend setup is complete
**When** I run `npm run dev`
**Then** dev server starts, PrimeVue renders with Aura unstyled, dark mode toggles via useTheme() composable, Tailwind layout utilities work, CSS layers are configured, design tokens available, API client types generated from openapi.yaml

### Story 1.8: [FRONT] App shell layout (Header, Sidebar, Status Bar)

As a user,
I want a responsive app shell with navigation,
So that I can navigate efficiently on all devices.

**Acceptance Criteria:**

**Given** the user is logged in
**When** they access any page
**Then** header (48px), sidebar (240px collapsible), and status bar (24px) display correctly

**Given** desktop viewport
**When** I press [ key
**Then** sidebar toggles between 240px and 48px

**Given** mobile viewport (<1024px)
**When** the page loads
**Then** sidebar is hidden, hamburger menu appears, bottom nav with 4 tabs shows

**Given** the layout renders
**When** I inspect the HTML
**Then** semantic elements (nav, main, aside) and skip navigation link are present

### Story 1.9: [FRONT] Login page + auth guard

As a user,
I want to log in with email and password,
So that I can access protected features.

**Acceptance Criteria:**

**Given** user is not authenticated
**When** they navigate to a protected route
**Then** they are redirected to /login

**Given** user submits valid credentials
**When** login succeeds
**Then** user state is stored in Pinia auth store and redirected to dashboard

**Given** user submits invalid credentials
**When** login fails
**Then** error message displays below form

**Given** form fields are present
**When** user blurs with invalid input
**Then** validation errors show (vee-validate + zod)

### Story 1.10: [FRONT] Project list page

As a user,
I want to see my projects,
So that I can select one to work with.

**Acceptance Criteria:**

**Given** user is authenticated
**When** they navigate to /projects
**Then** DataTable shows project name, repo URL, provider

**Given** no projects exist
**When** page loads
**Then** empty state shows "Create your first project" CTA

### Story 1.11: [FRONT] Project creation form

As an admin,
I want to create a new project,
So that I can connect a Git repository.

**Acceptance Criteria:**

**Given** user clicks "New Project"
**When** form displays
**Then** inputs for name, repo URL, git token are shown

**Given** user enters a repo URL
**When** URL contains gitlab.com or github.com
**Then** provider is auto-detected

**Given** user clicks "Validate Connection"
**When** API tests git access
**Then** success or error feedback is shown

**Given** user submits valid form
**When** POST /api/v1/projects succeeds
**Then** redirected to project detail page

### Story 1.12: [FRONT] Project settings page

As an admin,
I want to configure project settings,
So that I can control pipeline behavior.

**Acceptance Criteria:**

**Given** admin navigates to /projects/:id/settings
**When** page loads
**Then** TabView shows: General, Git, Agent, Budget tabs

**Given** admin edits and saves
**When** PUT succeeds
**Then** success toast shown

**Given** non-admin views settings
**When** page loads
**Then** all fields are read-only with banner

### Story 1.13: [FRONT] User management page (admin only)

As an admin,
I want to manage user accounts,
So that I can control access.

**Acceptance Criteria:**

**Given** admin navigates to /admin/users
**When** page loads
**Then** DataTable shows email, role, created date

**Given** admin clicks "Create User"
**When** dialog opens
**Then** form with email, password, role; submit creates user

**Given** admin clicks delete
**When** ConfirmDialog confirmed
**Then** user is deleted and table refreshes

**Given** non-admin navigates to /admin/users
**When** guard checks role
**Then** redirected to dashboard

### Story 1.14: [SHARED] CLAUDE.md files for agent scoping

As a platform architect,
I want composed CLAUDE.md files from modular templates,
So that agents enforce project boundaries and follow best practices.

**Acceptance Criteria:**

**Given** agent/claude-md/ directory exists
**When** I examine base.md
**Then** it documents git workflow, branch naming, conventional commits, quality standards

**Given** agent/claude-md/backend.md exists
**When** I examine it
**Then** it documents hexagonal architecture, chi, sqlc, DomainError, slog, testing patterns

**Given** agent/claude-md/frontend.md exists
**When** I examine it
**Then** it documents Composition API, PrimeVue first, Tailwind layout, useAsyncAction, Pinia patterns

**Given** agent/claude-md/project.md exists
**When** I examine it
**Then** it documents current state, key file paths, shared contract (api/openapi.yaml)

**Given** agent/claude-md/README.md exists
**When** I examine it
**Then** it specifies composition: base + (backend OR frontend) + project

## Epic 2: Story Board & Management

Users can view, import, and manage stories organized by epic within a project.

### Story 2.1: [BACK] Epics table + Epic CRUD API

As a user,
I want an epics table and CRUD API to manage epics within a project,
So that I can organize stories by feature area.

**Acceptance Criteria:**

**Given** migration 000004 exists
**When** migrations are applied
**Then** an epics table is created with: id (UUID PK), project_id (FK CASCADE), name, description, status, created_at, updated_at

**Given** sqlc queries are defined
**When** I run `make generate`
**Then** Go functions for CreateEpic, GetEpic, ListEpicsByProject, UpdateEpic, DeleteEpic are generated

**Given** OpenAPI spec includes epic endpoints
**When** routes are implemented
**Then** GET/POST /api/v1/projects/{id}/epics and GET/PUT/DELETE /api/v1/projects/{id}/epics/{epicId} work

**Given** a user with project access
**When** GET /api/v1/projects/{id}/epics is called
**Then** HTTP 200 returns epics for that project only

**Given** a user without project access
**When** any epic endpoint is called
**Then** HTTP 403 is returned

### Story 2.2: [BACK] Stories table + Story CRUD API + status filtering

As a user,
I want a stories table and CRUD API to manage stories and filter by status,
So that story data with frontmatter fields can be persisted and tracked.

**Acceptance Criteria:**

**Given** migration 000005 exists
**When** migrations are applied
**Then** a stories table is created with: id (UUID PK), project_id (FK CASCADE), epic_id (FK SET NULL), key (unique per project), title, objective, target_files (JSONB), depends_on (JSONB), scope, status (backlog/running/done/failed), acceptance_criteria, created_at, updated_at

**Given** sqlc queries are defined
**When** I run `make generate`
**Then** Go functions for CreateStory, GetStory, GetStoryByKey, ListStoriesByProject, ListStoriesByStatus, UpdateStory, DeleteStory are generated

**Given** OpenAPI spec includes story endpoints
**When** routes are implemented
**Then** GET/POST /api/v1/projects/{id}/stories and GET/PUT/DELETE .../stories/{storyId} work

**Given** a user with project access
**When** GET /api/v1/projects/{id}/stories?status=backlog,running
**Then** only stories with matching status are returned

**Given** a user with project access
**When** GET /api/v1/projects/{id}/stories/{storyId}
**Then** full detail with objectives, target_files, depends_on, acceptance_criteria is returned

**Given** a story with duplicate key is posted
**When** POST is processed
**Then** HTTP 409 Conflict is returned

### Story 2.3: [BACK] Story markdown import + frontmatter parsing

As a project maintainer,
I want to bulk import stories from markdown with YAML frontmatter,
So that I can manage stories as code.

**Acceptance Criteria:**

**Given** POST /api/v1/projects/{id}/stories/import is called with markdown content
**When** the system parses YAML frontmatter
**Then** key, epic, depends_on, scope, status are extracted

**Given** a story key already exists
**When** import includes that key
**Then** the existing story is updated

**Given** import has valid and invalid stories
**When** processed
**Then** valid ones are saved, errors reported per-story, partial success returned

**Given** frontmatter has invalid YAML
**When** parsing fails
**Then** error reported without failing entire import

### Story 2.4: [FRONT] Story board page — epic list with story counts

As a project user,
I want to see epics with story counts by status,
So that I can understand project progress at a glance.

**Acceptance Criteria:**

**Given** I navigate to /projects/:id/board
**When** page loads
**Then** epic cards show title, description, story counts by status with correct colors

**Given** no epics exist
**When** page loads
**Then** empty state with CTA shown

**Given** I click an epic card
**When** click registers
**Then** I navigate to /projects/:id/epics/:epicId

### Story 2.5: [FRONT] Epic detail — split focus + story list + filters

As a project user,
I want a split layout to browse and select stories within an epic,
So that I can efficiently review story details.

**Acceptance Criteria:**

**Given** I navigate to /projects/:id/epics/:epicId
**When** page loads
**Then** left panel (300px) shows compact story list, right panel shows selected story detail

**Given** story list is displayed
**When** I examine StoryStatusCard
**Then** it shows key (monospace), title, status badge with correct colors

**Given** filter bar is present
**When** I filter by status or search by text
**Then** story list updates with 200ms debounce, filters preserved in URL

**Given** keyboard is used
**When** I press J/K
**Then** selection moves, Enter shows detail in right panel

**Given** I click a story
**When** click registers
**Then** detail loads in right panel without route change

### Story 2.6: [FRONT] Story detail view (read-only)

As a project user,
I want to see story details including objectives and acceptance criteria,
So that I can understand what needs to be built.

**Acceptance Criteria:**

**Given** a story is selected in epic detail
**When** detail displays in right panel
**Then** title, status badge, key (monospace), objective (markdown rendered), target files, dependencies (clickable keys), and acceptance criteria are shown

**Given** a dependency key is clicked
**When** click registers
**Then** that story is selected in the left panel

### Story 2.7: [FRONT] Story editor (create + edit mode)

As a project user,
I want to create and edit stories,
So that I can manage story content.

**Acceptance Criteria:**

**Given** I click "Edit" on story detail
**When** edit mode activates
**Then** title becomes input, objective becomes textarea, target files become editable list

**Given** I save with valid data
**When** PUT succeeds
**Then** edit mode exits, data refreshes, success toast shown

**Given** I save with invalid data
**When** validation fails
**Then** inline errors shown, form not submitted

**Given** I click "Create Story"
**When** editor opens
**Then** empty form with title, objective, target files; POST creates on submit

### Story 2.8: [FRONT] Story import from markdown

As a project user,
I want to import stories from markdown files,
So that I can populate the board quickly.

**Acceptance Criteria:**

**Given** I click "Import Stories" on board page
**When** dialog opens
**Then** file upload zone accepts .md files (drag & drop or click)

**Given** I upload a file
**When** parsed locally
**Then** preview shows stories with frontmatter fields

**Given** I click "Import"
**When** POST succeeds
**Then** results show created/updated/error counts, board refreshes on close

## Epic 3: Single Story Pipeline Execution

User can launch a story that executes automatically: Docker container → agent code → branch → commit → PR → merge. Includes agent runtime (Docker), git provider (gh CLI), pipeline execution engine, container lifecycle, NDJSON streaming, event bus (pgxlisten).

### Story 3.1: [BACK] Runs & RunSteps tables + Run creation API + state machine

As a backend developer,
I want database schemas for runs/run_steps and a run creation service with proper state machine transitions,
So that the system can track pipeline execution state persistently and runs progress through defined states reliably.

**Acceptance Criteria:**

**Given** the database is at migration 000005
**When** migration 000006_create_runs_table is applied
**Then** a runs table is created with columns: id (UUID PK), project_id (FK projects CASCADE), story_id (FK stories CASCADE), status (VARCHAR NOT NULL default 'pending' CHECK IN pending/running/completed/failed/cancelled), started_at (TIMESTAMPTZ), completed_at (TIMESTAMPTZ), error_message (TEXT), created_at, updated_at
**And** indexes exist on (project_id, status) and (story_id)

**Given** migration 000006 is complete
**When** migration 000007_create_run_steps_table is applied
**Then** a run_steps table is created with columns: id (UUID PK), run_id (FK runs CASCADE), step_name (VARCHAR NOT NULL), step_order (INT NOT NULL), status (VARCHAR NOT NULL default 'pending'), started_at, completed_at, error_message (TEXT), container_id (VARCHAR), log_tail (TEXT), created_at
**And** an index exists on (run_id, step_order)

**Given** migrations are applied
**When** sqlc queries are generated
**Then** run queries exist: CreateRun, GetRun, ListRunsByProject, ListRunsByStory, UpdateRunStatus
**And** run_step queries exist: CreateRunStep, GetRunStep, ListRunStepsByRun, UpdateRunStepStatus
**And** all generated code compiles without errors

**Given** a valid story_id and project_id
**When** a run is created via the service layer
**Then** a run record is inserted with status 'pending'
**And** run_steps are created for each pipeline step configured for the project
**And** steps are ordered by pipeline configuration

**Given** a run exists with status 'pending'
**When** transition to 'running' is requested
**Then** status updates to 'running' and started_at is set

**Given** a run exists with status 'running'
**When** transition to 'completed' is requested
**Then** status updates to 'completed' and completed_at is set

**Given** a run exists with status 'running'
**When** transition to 'failed' is requested with error message
**Then** status updates to 'failed', completed_at is set, error_message is stored

**Given** a run exists with status 'completed'
**When** transition to 'running' is attempted
**Then** the transition is rejected with DomainError invalid_state_transition

**Given** a run_step transitions to 'completed'
**When** all steps in the run are 'completed'
**Then** the run automatically transitions to 'completed'

### Story 3.2: [BACK] GitProvider port + gh CLI adapter (clone, branch, push)

As a backend developer,
I want a GitProvider port with gh CLI implementation for repository operations,
So that the pipeline can interact with Git repositories through a testable interface.

**Acceptance Criteria:**

**Given** a GitProvider port interface is defined
**When** I examine the interface
**Then** it declares methods: CloneRepo, CreateBranch, Push, CreatePR, MergePR, GetCIStatus

**Given** the gh CLI adapter implements GitProvider
**When** CloneRepo is called with repo URL and target directory
**Then** the repository is cloned via CommandRunner interface
**And** errors are wrapped in DomainError with context

**Given** a cloned repository
**When** CreateBranch is called with story key and slug
**Then** a branch named feat/S-{key}-{slug} is created and checked out

**Given** changes exist in the working directory
**When** Push is called with commit message
**Then** changes are committed with conventional commit message and pushed to remote

**Given** the CommandRunner returns a non-zero exit code
**When** any GitProvider method fails
**Then** a DomainError is returned with the command output as context

### Story 3.3: [BACK] GitProvider PR operations (create PR, merge PR)

As a backend developer,
I want PR creation and merge operations via the GitProvider,
So that the pipeline can complete the code review and merge cycle.

**Acceptance Criteria:**

**Given** a branch with commits exists
**When** CreatePR is called with title, body, and base branch
**Then** a pull/merge request is created via gh CLI
**And** the PR URL is returned

**Given** a PR exists and CI has passed
**When** MergePR is called with PR identifier
**Then** the PR is squash-merged via gh CLI
**And** the source branch is deleted after merge

**Given** a merge conflict exists
**When** MergePR is called
**Then** a DomainError merge_conflict is returned with conflict details

**Given** the gh CLI is not authenticated
**When** any PR operation is called
**Then** a DomainError git_auth_failed is returned

### Story 3.4: [BACK] Docker container lifecycle manager (create, start, stop, cleanup)

As a backend developer,
I want a container lifecycle manager for agent execution,
So that the system can safely create, run, and clean up isolated Docker containers.

**Acceptance Criteria:**

**Given** a ContainerManager interface is defined
**When** I examine the interface
**Then** it declares: Create, Start, Stop, Remove, InjectEnv, StreamLogs

**Given** docker-socket-proxy is running with allowlisted operations
**When** Create is called with image name and configuration
**Then** a container is created on the dedicated agent network
**And** no host filesystem mounts are attached
**And** no privileged mode is enabled

**Given** a container is created
**When** Start is called
**Then** the container starts and container ID is returned

**Given** a running container
**When** Stop is called
**Then** the container is stopped gracefully (SIGTERM, 10s timeout, then SIGKILL)

**Given** a stopped container
**When** Remove is called
**Then** the container and its volumes are removed

**Given** environment variables are provided
**When** InjectEnv is called during creation
**Then** CLAUDE.md content, secrets, and config are injected as environment variables
**And** API keys are never written to container filesystem

### Story 3.5: [BACK] NDJSON log streaming from container

As a backend developer,
I want to stream NDJSON logs from running agent containers,
So that the system can capture and forward real-time agent output.

**Acceptance Criteria:**

**Given** a running agent container
**When** StreamLogs is called
**Then** stdout is read line by line as NDJSON
**And** each line is parsed and validated as JSON

**Given** a stream is active
**When** a new NDJSON line is received
**Then** it is forwarded to the log channel with run_id and step_id context

**Given** the container exits
**When** the stream reaches EOF
**Then** the stream channel is closed cleanly
**And** final exit code is captured

**Given** a malformed line is received (not valid JSON)
**When** the parser encounters it
**Then** the line is wrapped as a raw text event and forwarded
**And** parsing continues for subsequent lines

**Given** the container produces no output for 60 seconds
**When** the idle timeout is reached
**Then** a warning event is emitted but the stream continues

### Story 3.6: [BACK] Events table + pgxlisten event bus

As a backend developer,
I want an event log table with Postgres LISTEN/NOTIFY integration,
So that all system events are persisted and broadcast to subscribers in real-time.

**Acceptance Criteria:**

**Given** migration 000008 exists
**When** migrations are applied
**Then** an events table is created with: id (UUID PK), project_id (FK projects CASCADE), entity_type (VARCHAR), entity_id (UUID), action (VARCHAR), payload (JSONB), created_at (TIMESTAMPTZ)
**And** indexes exist on (project_id, created_at) and (entity_type, entity_id)

**Given** the events table exists
**When** a new row is inserted
**Then** a Postgres NOTIFY trigger fires with channel name matching entity_type.action pattern

**Given** pgxlisten wrapper is implemented
**When** the eventbus starts
**Then** a dedicated Postgres connection is established separate from the query pool
**And** auto-reconnection is configured with exponential backoff

**Given** the eventbus is running
**When** a service publishes an event via EventPublisher.Publish(ctx, "run.started", payload)
**Then** the event is inserted into the events table with snake_case JSON payload
**And** NOTIFY is triggered on the "run.started" channel

**Given** a subscriber calls eventbus.Subscribe("run.started")
**When** events are published on that channel
**Then** the subscriber receives events via returned channel

**Given** the Postgres connection is lost
**When** the eventbus detects disconnection
**Then** it attempts reconnection with exponential backoff and logs reconnection events

### Story 3.7: [BACK] Pipeline executor — sequential step runner

As a backend developer,
I want a pipeline executor that runs steps sequentially for a story,
So that each pipeline step executes in order with proper state tracking.

**Acceptance Criteria:**

**Given** a run with ordered run_steps exists
**When** the pipeline executor starts
**Then** steps are executed in step_order sequence
**And** each step transitions through pending → running → completed/failed

**Given** the current step completes successfully
**When** the executor advances
**Then** the next step begins execution
**And** the run_step record is updated with started_at

**Given** a step fails
**When** the executor handles the failure
**Then** the run_step is marked as 'failed' with error_message
**And** the run is marked as 'failed'
**And** remaining steps are NOT executed

**Given** the executor needs to run a step
**When** it looks up the step action
**Then** it uses the ActionRegistry to find the appropriate action handler
**And** the action handler is invoked with run context

**Given** the executor is running
**When** the run is cancelled
**Then** the current step is stopped and marked 'cancelled'
**And** the run is marked 'cancelled'

### Story 3.8: [BACK] Agent run action (compose CLAUDE.md + launch container + stream logs)

As a backend developer,
I want an agent_run action that composes CLAUDE.md, launches a container, and streams output,
So that Claude Code agents execute with proper context and real-time log capture.

**Acceptance Criteria:**

**Given** an agent_run action is triggered for a story
**When** the action starts
**Then** CLAUDE.md is composed from base.md + (backend.md OR frontend.md based on story scope) + project.md

**Given** CLAUDE.md is composed
**When** the container is created
**Then** CLAUDE.md content, repo URL, branch name, story context, and prompt are injected as environment variables

**Given** the container is started
**When** the agent begins execution
**Then** NDJSON logs are streamed and forwarded to the event system

**Given** the agent completes successfully
**When** the container exits with code 0
**Then** the action reports success
**And** the container is cleaned up

**Given** the agent fails
**When** the container exits with non-zero code
**Then** the action reports failure with log tail as error context
**And** the container is cleaned up

> **Note:** For MVP, prompt templates are loaded from the filesystem (`agent/prompts/*.hbs`) and rendered with Handlebars. Epic 6 upgrades this to DB-backed templates with CRUD UI.

### Story 3.9: [BACK] Container timeout enforcement + orphan cleanup

As a platform operator,
I want container timeouts and orphan cleanup,
So that runaway containers don't consume resources indefinitely.

**Acceptance Criteria:**

**Given** a container is running
**When** it exceeds the configured timeout (default 30 minutes)
**Then** the container is forcefully stopped
**And** the run_step is marked 'failed' with error "container_timeout"
**And** the run is marked 'failed'

**Given** the API service starts up
**When** the orphan cleanup routine runs
**Then** it lists all containers with the agent label
**And** containers not associated with an active run are removed

**Given** multiple orphan containers exist
**When** cleanup runs
**Then** all orphans are removed and a summary is logged via slog

**Given** the timeout is configurable per project
**When** a project has max_container_timeout set
**Then** that value is used instead of the default

### Story 3.10: [BACK] Run launch API endpoint (single story)

As a frontend developer,
I want an API endpoint to launch a pipeline run for a single story,
So that users can trigger story execution from the UI.

**Acceptance Criteria:**

**Given** I am authenticated with access to the project
**When** I POST /api/v1/projects/{projectId}/stories/{storyId}/runs
**Then** HTTP 201 returns the created run object with status 'pending'
**And** the pipeline executor is enqueued via River job queue

**Given** a story already has a run with status 'running'
**When** I POST to launch another run
**Then** HTTP 409 is returned with DomainError story_already_running

**Given** a story has status 'done'
**When** I POST to launch a run
**Then** HTTP 400 is returned with DomainError story_already_completed

**Given** I do not have access to the project
**When** I POST to launch a run
**Then** HTTP 403 is returned

**Given** the story_id does not exist
**When** I POST to launch a run
**Then** HTTP 404 is returned

### Story 3.11: [FRONT] Run launch button + confirmation dialog

As a user,
I want to launch a story run from the story detail view,
So that I can trigger the AI pipeline for a specific story.

**Acceptance Criteria:**

**Given** I am viewing a story with status 'backlog'
**When** I see the story detail
**Then** a "Launch Run" primary button is visible

**Given** I click "Launch Run"
**When** the confirmation dialog appears
**Then** it shows story key, title, and a warning about resource usage
**And** "Confirm" and "Cancel" buttons are present

**Given** I click "Confirm" in the dialog
**When** POST /api/v1/projects/{id}/stories/{storyId}/runs succeeds
**Then** success toast is shown
**And** story status updates to 'running' in the UI
**And** the button changes to disabled "Running..." state

**Given** the API returns an error (409 already running)
**When** the error is handled
**Then** error toast shows the reason
**And** dialog remains open

**Given** a story has status 'running'
**When** I view the story detail
**Then** "Launch Run" button is disabled with tooltip "Already running"

### Story 3.12: [FRONT] Run status display on story card

As a user,
I want to see run status on story cards in the board view,
So that I can monitor execution progress at a glance.

**Acceptance Criteria:**

**Given** a story has an active run
**When** the StoryStatusCard renders
**Then** it shows a status indicator: spinning icon for 'running', check for 'completed', X for 'failed'
**And** status text is shown alongside the icon

**Given** a story has a completed run
**When** the card renders
**Then** it shows completion time as relative time (e.g., "2h ago")

**Given** a story has a failed run
**When** the card renders
**Then** it shows a red indicator with "Failed" text
**And** clicking the card shows error details in the detail panel

**Given** a story has no runs
**When** the card renders
**Then** it shows neutral gray "Backlog" status

## Epic 4: Real-time Monitoring & Live Logs

User can follow execution in real-time with live logs and step progress. Includes SSE streaming, LogViewer component, run progress tracking. Event bus (pgxlisten) is implemented in Epic 3.

### Story 4.1: [BACK] SSE streaming endpoint

As a backend developer,
I want an SSE endpoint that streams events to browser clients,
So that the frontend can receive real-time updates without polling.

**Acceptance Criteria:**

**Given** OpenAPI spec defines GET /api/v1/events/stream
**When** the endpoint is implemented
**Then** it returns Content-Type: text/event-stream with no-cache headers

**Given** a client requests /api/v1/events/stream?project_id={id}
**When** the handler starts
**Then** it subscribes to eventbus channels for that project_id
**And** a per-client goroutine is spawned to forward events

**Given** the SSE handler is running
**When** an event is published to the eventbus
**Then** it is formatted as SSE data: event_type\ndata: json_payload\nid: event_id\n\n
**And** written to all connected clients for that project

**Given** a client connection drops
**When** the handler detects write failure
**Then** the goroutine exits cleanly and unsubscribes from eventbus

**Given** a client reconnects with Last-Event-ID header
**When** the handler receives the connection
**Then** it queries events table for missed events since last_event_id
**And** replays those events before streaming new ones

**Given** no events occur for 30 seconds
**When** the heartbeat timer fires
**Then** a comment line ": keepalive\n\n" is sent to all connected clients

### Story 4.2: [BACK] Run progress tracking API

As a backend developer,
I want API endpoints that return run details with step-level progress,
So that clients can display execution status and calculate completion percentage.

**Acceptance Criteria:**

**Given** a run exists with multiple run_steps
**When** GET /api/v1/projects/{projectId}/runs/{runId} is called
**Then** HTTP 200 returns run object with nested steps array
**And** each step includes: id, step_name, step_order, status, started_at, completed_at

**Given** a run has 5 steps with 3 completed and 2 pending
**When** the run detail is retrieved
**Then** the response includes progress field with percentage: 60
**And** progress is calculated as: (completed_steps / total_steps) * 100

**Given** a user with project access
**When** GET /api/v1/projects/{projectId}/runs is called
**Then** HTTP 200 returns paginated list of runs with default sort by created_at DESC
**And** each run includes summary fields: id, story_id, status, started_at, completed_at, progress

**Given** pagination params page=2&per_page=10 are provided
**When** the runs list endpoint is called
**Then** results are offset correctly and pagination metadata is returned

**Given** a run_step transitions to completed
**When** the step is updated
**Then** an event "step.completed" is published with run_id, step_id, step_name in payload

**Given** all steps in a run are completed
**When** the run status is calculated
**Then** progress is 100 and run status is completed

### Story 4.3: [FRONT] LogViewer component + live log page

As a user,
I want to view live agent logs with ANSI color rendering,
So that I can monitor agent execution in real-time.

**Acceptance Criteria:**

**Given** LogViewer component exists
**When** it receives SSE events filtered by run_id
**Then** log lines are rendered with ANSI color codes converted to HTML spans

**Given** the log volume exceeds 1000 lines
**When** LogViewer renders
**Then** vue-virtual-scroller is used for efficient rendering without lag

**Given** new log lines are appended
**When** auto-scroll is enabled
**Then** the viewport scrolls to bottom automatically
**And** manual scroll up disables auto-scroll with visual indicator

**Given** the user scrolls manually
**When** they scroll to within 50px of bottom
**Then** auto-scroll re-enables automatically

**Given** the LogViewer toolbar is present
**When** the user clicks "Clear"
**Then** all displayed logs are removed and scroll resets

**Given** the page /projects/:id/runs/:runId/logs is accessed
**When** the page loads
**Then** LogViewer component connects to SSE stream filtered by run_id
**And** displays logs in real-time with timestamps

### Story 4.4: [FRONT] Run progress timeline display

As a user,
I want to see a step-by-step timeline of run execution,
So that I can understand which steps are completed, running, or pending.

**Acceptance Criteria:**

**Given** PipelineTimeline component exists
**When** it receives run data with steps
**Then** each step is rendered as a timeline entry with name, status icon, and duration

**Given** a step has status 'running'
**When** the timeline renders
**Then** that step shows an animated spinner icon

**Given** a step has status 'completed'
**When** the timeline renders
**Then** it shows a green checkmark icon and displays duration in seconds

**Given** a step has status 'failed'
**When** the timeline renders
**Then** it shows a red X icon and displays error message below step name

**Given** useSSE composable is active
**When** a "step.completed" event is received
**Then** the event is dispatched to Pinia run store
**And** the store updates the corresponding step status reactively

**Given** the page /projects/:id/runs/:runId is accessed
**When** the page loads
**Then** PipelineTimeline and LogViewer components are displayed side-by-side
**And** timeline updates in sync with log events via SSE

## Epic 5: HITL Gates & Approval Workflow

User can approve or reject at human-in-the-loop checkpoints, with inline diff and notifications.

### Story 5.1: [BACK] HITL gate action + pending state

As a backend developer,
I want a hitl_gate action that pauses pipeline execution,
So that a human can review and approve before proceeding.

**Acceptance Criteria:**

**Given** the ActionRegistry includes hitl_gate action
**When** the pipeline executor reaches a HITL step
**Then** the action is looked up and Execute is called with run context

**Given** hitl_gate action executes
**When** it starts
**Then** the run_step status is set to 'pending_approval'
**And** started_at is recorded

**Given** a HITL request is created
**When** the gate action stores the request
**Then** a record is created with: run_step_id, gate_type (default "approval"), diff_content (from git provider), created_at

**Given** the HITL request is stored
**When** the action publishes an event
**Then** "hitl_gate.pending" event is emitted with payload: run_id, step_id, story_key, diff_url, created_at

**Given** the pipeline executor calls hitl_gate
**When** the action returns
**Then** the executor pauses naturally and does NOT enqueue the next step
**And** the run remains in 'running' status but progress is blocked on pending_approval step

### Story 5.2: [BACK] HITL approval/rejection API

As a backend developer,
I want API endpoints to approve or reject HITL gates,
So that users can control pipeline progression from the UI.

**Acceptance Criteria:**

**Given** OpenAPI spec defines POST /api/v1/projects/{projectId}/runs/{runId}/steps/{stepId}/approve
**When** a user with project access calls the endpoint
**Then** HTTP 200 is returned and the run_step status is updated to 'completed'

**Given** a run_step is approved
**When** the approval is processed
**Then** the pipeline executor enqueues the next step via River job queue
**And** the run transitions from blocked to progressing

**Given** OpenAPI spec defines POST /api/v1/projects/{projectId}/runs/{runId}/steps/{stepId}/reject
**When** the endpoint is called with body { "reason": "code quality issues" }
**Then** HTTP 200 is returned and run_step status is set to 'failed'
**And** the rejection reason is stored in run_step.error_message

**Given** a run_step is rejected
**When** the rejection is processed
**Then** the run status is set to 'failed'
**And** remaining steps are NOT executed

**Given** an approval/rejection event occurs
**When** the action completes
**Then** "hitl_gate.approved" or "hitl_gate.rejected" event is published with run_id, step_id, user_id, reason (if rejected)

**Given** a user without project access tries to approve
**When** the middleware checks authorization
**Then** HTTP 403 is returned

### Story 5.3: [DEFERRED - Phase 2] [BACK] HITL CLI approval support (curl-based)

As a platform maintainer,
I want to document curl commands for HITL approval,
So that users can approve from terminal without a full CLI client.

**Acceptance Criteria:**

**Given** the API endpoints for approve/reject exist
**When** a user has a valid JWT cookie
**Then** they can approve via: curl -X POST -b cookies.txt /api/v1/projects/{id}/runs/{runId}/steps/{stepId}/approve

**Given** the user needs to list pending HITL requests
**When** they call GET /api/v1/projects/{projectId}/hitl/pending
**Then** HTTP 200 returns array of pending requests with: run_id, step_id, story_key, created_at, diff_url

**Given** the API documentation includes curl examples
**When** a developer reads the docs
**Then** they see working examples with placeholders for project_id, run_id, step_id, and auth cookie

**Given** a user rejects via curl with JSON body
**When** they execute: curl -X POST -b cookies.txt -H "Content-Type: application/json" -d '{"reason":"needs refactor"}' .../reject
**Then** the rejection is processed successfully

### Story 5.4: [FRONT] HITL approval page + DiffViewer

As a user,
I want to review proposed changes with a visual diff before approving,
So that I can make informed decisions about code changes.

**Acceptance Criteria:**

**Given** the page /projects/:id/runs/:runId/approve/:stepId exists
**When** the page loads
**Then** it fetches the HITL request and displays story context: key, title, objective

**Given** the HITL request includes diff_content
**When** the DiffViewer component renders
**Then** diff2html library renders the diff with side-by-side or unified view toggle

**Given** the approval page is displayed
**When** the user reviews the diff
**Then** "Approve" button (green, primary) and "Reject" button (red, danger ghost) are visible

**Given** the user clicks "Approve"
**When** POST /api/v1/.../approve succeeds
**Then** success toast is shown with message "Step approved"
**And** the user is redirected to /projects/:id/runs/:runId

**Given** the user clicks "Reject"
**When** the button is clicked
**Then** a textarea appears requiring a rejection reason
**And** submit is disabled until reason is entered

**Given** SSE event "hitl_gate.pending" is received
**When** the event is dispatched to stores
**Then** a toast notification is shown with action button "Review Now" that navigates to approval page

### Story 5.5: [FRONT] HITL pending list + notification badge

As a user,
I want to see all pending approvals for a project,
So that I can review and act on them efficiently.

**Acceptance Criteria:**

**Given** the page /projects/:id/approvals exists
**When** the page loads
**Then** it calls GET /api/v1/projects/{id}/hitl/pending and displays a list of pending HITL requests

**Given** pending requests are displayed
**When** each item renders
**Then** it shows: story key (monospace), step name, waiting time (relative, e.g., "5 minutes ago"), and "Review" button

**Given** the sidebar navigation is visible
**When** there are pending approvals
**Then** a badge with count is shown next to "Approvals" menu item
**And** the badge updates reactively via SSE events

**Given** a user clicks the "Review" button on a pending item
**When** the click is processed
**Then** the user navigates to /projects/:id/runs/:runId/approve/:stepId

**Given** no pending approvals exist
**When** the /projects/:id/approvals page loads
**Then** an empty state is displayed with icon and text "No pending approvals"

**Given** an approval is completed or rejected elsewhere
**When** the corresponding SSE event is received
**Then** the item is removed from the pending list reactively
**And** the sidebar badge count decrements

## Epic 6: Pipeline Configuration & Prompt Templates

Admin can customize pipeline steps and prompt templates per project.

**FRs covered:** FR19, FR20, FR21, FR22, FR33, FR34, FR35, FR36

### Story 6.1: [BACK] Pipeline configs table + default pipeline seed + CRUD API

As an admin,
I want a pipeline_configs table with default seeding and CRUD API to view and update pipeline configurations,
So that I can customize execution steps per project.

**Acceptance Criteria:**

**Given** migration 000009 exists
**When** migrations are applied
**Then** a pipeline_configs table is created with: id (UUID PK), project_id (FK), config_yaml (TEXT NOT NULL), version (INT default 1), created_at, updated_at

**Given** migration 000009 runs on a fresh database
**When** the migration completes
**Then** a default pipeline config is seeded with steps: [agent_run, hitl_gate, git_create_pr, git_merge]

**Given** sqlc queries are defined for pipeline configs
**When** I run `make generate`
**Then** Go functions for GetPipelineConfig, UpsertPipelineConfig are generated

**Given** the default pipeline YAML structure
**When** I examine the seeded config
**Then** each step contains: name, action, model, auto_approve, retry_policy fields

**Given** a project is created
**When** I query its pipeline config
**Then** the default pipeline config is returned if no custom config exists

**Given** I am authenticated and have project access
**When** I GET /api/v1/projects/{projectId}/pipeline
**Then** I receive HTTP 200 with the current pipeline config as parsed YAML object

**Given** I am admin
**When** I PUT /api/v1/projects/{projectId}/pipeline with valid YAML structure
**Then** I receive HTTP 200 and the pipeline config is updated with version incremented

**Given** I am admin
**When** I PUT /api/v1/projects/{projectId}/pipeline with invalid YAML (non-existent action)
**Then** I receive HTTP 400 with validation error listing invalid action name

**Given** I am a regular user (non-admin)
**When** I PUT /api/v1/projects/{projectId}/pipeline
**Then** I receive HTTP 403

**Given** the ActionRegistry contains valid actions
**When** pipeline config is validated during update
**Then** each step's action field must reference a valid action from the registry

**Given** a pipeline config update succeeds
**When** I GET the pipeline config
**Then** the version number is incremented by 1

### Story 6.2: [BACK] Prompt templates table + CRUD API

> **Note:** This story UPGRADES prompt storage from file-based (`agent/prompts/*.hbs`, implemented in Epic 3 Story 3.8) to DB-backed templates with CRUD UI.

As a backend developer,
I want a prompt_templates table with CRUD endpoints,
So that users can manage agent prompts per project via API.

**Acceptance Criteria:**

**Given** migration 000010 exists
**When** migrations are applied
**Then** a prompt_templates table is created with: id (UUID PK), project_id (FK), name (VARCHAR), template_content (TEXT), type (VARCHAR: implement/retry/review/merge/custom), created_at, updated_at

**Given** the prompt_templates table schema
**When** I examine constraints
**Then** a unique constraint exists on (project_id, name)

**Given** I am authenticated
**When** I GET /api/v1/projects/{projectId}/templates
**Then** I receive HTTP 200 with a list of all templates for the project

**Given** I am admin
**When** I POST /api/v1/projects/{projectId}/templates with name, template_content, and type
**Then** I receive HTTP 201 with the created template

**Given** I am admin
**When** I PUT /api/v1/projects/{projectId}/templates/{id} with updated content
**Then** I receive HTTP 200 and the template is updated

**Given** I am admin
**When** I DELETE /api/v1/projects/{projectId}/templates/{id}
**Then** I receive HTTP 204 and the template is deleted

### Story 6.3: [BACK] Handlebars rendering engine + default template seeding

As a backend developer,
I want a Handlebars rendering engine for prompt templates and default template seeding,
So that prompts are rendered with story context variables and projects start with sensible defaults.

**Acceptance Criteria:**

**Given** a Handlebars template with context variables
**When** RenderTemplate(templateId, context) is called
**Then** the template is rendered with variables: story_key, story_title, story_objective, target_files, acceptance_criteria, error_context, diff_content

**Given** a template contains invalid Handlebars syntax
**When** RenderTemplate is called
**Then** a DomainError template_render_failed is returned with syntax error details

**Given** migration 000010 runs on a fresh database
**When** the migration completes
**Then** default templates are seeded: implement.hbs, implement-retry.hbs, review.hbs, merge-conflict.hbs

**Given** the rendering engine is initialized
**When** the agent_run action needs a prompt
**Then** it resolves the template from the DB (falling back to defaults) and renders it with story context

### Story 6.4: [FRONT] Pipeline configuration page

As an admin,
I want a visual pipeline configuration page,
So that I can customize pipeline steps without editing raw YAML.

**Acceptance Criteria:**

**Given** I am logged in as admin
**When** I navigate to /projects/:id/pipeline
**Then** I see the current pipeline steps displayed as an ordered list with step name, action type, model, and auto_approve toggle

**Given** the pipeline configuration page is loaded
**When** I examine each step
**Then** each step is expandable to show: model selector, auto_approve checkbox, retry policy (max_retries, retry_type)

**Given** I am admin on the pipeline page
**When** I drag a step to reorder or use move up/down buttons
**Then** the step order updates immediately in the UI

**Given** I am admin on the pipeline page
**When** I click "Add Step" button
**Then** a new step form appears with action type selector and configuration fields

**Given** I am admin on the pipeline page
**When** I click "Remove" on a step
**Then** the step is removed from the pipeline

**Given** I am admin and have modified the pipeline
**When** I click "Save" button
**Then** PUT /api/v1/projects/{id}/pipeline is called and success feedback is shown

**Given** I am a non-admin user
**When** I navigate to /projects/:id/pipeline
**Then** I see the pipeline in read-only mode with no edit controls visible

### Story 6.5: [FRONT] Prompt template list page

As a user,
I want to view all prompt templates for a project,
So that I can see which templates are available.

**Acceptance Criteria:**

**Given** I am logged in and viewing a project
**When** I navigate to /projects/:id/templates
**Then** I see a PrimeVue DataTable listing templates with columns: name, type, last updated

**Given** the template list page is loaded
**When** I click on a template row
**Then** I navigate to the template detail page

**Given** I am admin on the templates page
**When** I click "Create Template" button
**Then** I navigate to the template editor in create mode

**Given** templates of multiple types exist
**When** I use the type filter dropdown
**Then** the table filters to show only templates of selected type (implement/retry/review/merge/custom)

**Given** I am a non-admin user
**When** I view the templates page
**Then** the "Create Template" button is not visible

### Story 6.6: [FRONT] Prompt template editor

As an admin,
I want to edit prompt templates with syntax support,
So that I can customize agent prompts effectively.

**Acceptance Criteria:**

**Given** I am admin
**When** I navigate to /projects/:id/templates/:templateId
**Then** I see a Monaco editor displaying the Handlebars template content

**Given** the template editor is loaded
**When** I examine the sidebar
**Then** I see available context variables with descriptions (story_key, story_title, acceptance_criteria, etc.)

**Given** I am editing a template
**When** I click the "Preview" button
**Then** the template is rendered with sample context data and displayed

**Given** I am admin and have modified the template
**When** I click "Save"
**Then** the template is updated via PUT /api/v1/projects/{id}/templates/{templateId} and success feedback is shown

**Given** I am admin on the template editor
**When** I click "Cancel"
**Then** I navigate back to the templates list without saving changes

**Given** I am a non-admin user
**When** I navigate to /projects/:id/templates/:templateId
**Then** the Monaco editor is in read-only mode with no Save button

## Epic 7: Epic Batch Execution & DAG Scheduling

User can launch an entire epic with intelligent parallel execution based on story dependencies.

**FRs covered:** FR10, FR11, FR12, FR17, FR18

### Story 7.1: [BACK] DAG builder + topological sort

As a backend developer,
I want a DAG builder service that resolves story dependencies,
So that epic runs can execute stories in parallel where possible.

**Acceptance Criteria:**

**Given** a list of stories with depends_on fields
**When** the DAG builder processes the stories
**Then** a dependency graph is built from explicit depends_on relationships

**Given** stories with provides and requires metadata
**When** the DAG builder processes the stories
**Then** implicit dependencies are detected by matching provides/requires fields

**Given** a dependency graph is built
**When** topological sort is performed
**Then** execution groups are returned as ordered parallel layers: [[S-001, S-002], [S-003], [S-004, S-005]]

**Given** stories with circular dependencies exist
**When** the DAG builder processes them
**Then** a DomainError with code dag_cycle_detected is returned

**Given** stories have both explicit and implicit dependencies
**When** the DAG builder resolves dependencies
**Then** the final graph includes all dependencies from both sources

**Given** stories with no dependencies between them
**When** the DAG builder groups them
**Then** they are placed in the same execution group for parallel execution

### Story 7.2: [BACK] Epic run creation + parallel group executor

As a backend developer,
I want an epic batch run orchestrator using River,
So that multiple stories can execute in parallel with proper dependency ordering.

**Acceptance Criteria:**

**Given** I am authenticated
**When** I POST /api/v1/projects/{projectId}/epics/{epicId}/runs
**Then** I receive HTTP 202 with a parent run record for the epic

**Given** an epic run is launched
**When** the epic run is created
**Then** child run records are created for each story in the epic

**Given** the DAG builder has produced execution groups
**When** the epic executor processes groups
**Then** all stories in group 1 are enqueued in River, then group 2 after group 1 completes, etc.

**Given** a project has max_parallel_runs configured
**When** stories are executed in parallel
**Then** no more than max_parallel_runs stories execute simultaneously

**Given** stories in an execution group
**When** River workers pick up jobs
**Then** each story run is executed as an independent River job

**Given** a story in a group fails
**When** the group completes
**Then** subsequent groups are not started and the epic run status is set to failed

### Story 7.3: [BACK] Pause/Resume for story and epic runs

As a backend developer,
I want pause and resume functionality for runs,
So that users can control execution flow.

**Acceptance Criteria:**

**Given** a story run is in running status
**When** I POST /api/v1/projects/{projectId}/runs/{runId}/pause
**Then** the run status is set to paused and no new steps are launched (current step continues)

**Given** a paused story run
**When** I POST /api/v1/projects/{projectId}/runs/{runId}/resume
**Then** the run status is set to running and execution resumes from the last completed step

**Given** a run is paused
**When** the status change completes
**Then** an event run.paused is published to the event bus

**Given** a run is resumed
**When** the status change completes
**Then** an event run.resumed is published to the event bus

**Given** an epic run is in progress
**When** I POST /api/v1/projects/{projectId}/epics/{epicId}/runs/{runId}/pause
**Then** all child runs that have not started are paused and running ones continue to completion

**Given** a paused epic run
**When** I POST /api/v1/projects/{projectId}/epics/{epicId}/runs/{runId}/resume
**Then** all paused child runs are resumed and execution continues

### Story 7.4: [FRONT] Epic launch page + DAG visualization

As a user,
I want to visualize the epic DAG before launching,
So that I understand dependencies and execution order.

**Acceptance Criteria:**

**Given** I am viewing an epic detail page
**When** I click "Launch Epic" button
**Then** I navigate to a pre-launch view showing DAG visualization

**Given** the DAG visualization is rendered
**When** I examine the graph
**Then** I see story nodes grouped by execution layer using @vue-flow/core

**Given** a story node in the DAG
**When** I examine the node
**Then** it displays story key, title, and status (if already completed/running)

**Given** stories with dependencies
**When** the DAG is rendered
**Then** dependency edges are drawn between nodes showing the dependency direction

**Given** some stories are already done or running
**When** the DAG is rendered
**Then** those nodes are displayed as disabled/non-launchable

**Given** I have reviewed the DAG
**When** I click "Launch" button
**Then** POST /api/v1/projects/{projectId}/epics/{epicId}/runs is called and the epic run starts

### Story 7.5: [FRONT] Epic run monitoring dashboard

As a user,
I want to monitor epic run progress in real-time,
So that I can track execution and intervene if needed.

**Acceptance Criteria:**

**Given** an epic run is in progress
**When** I navigate to /projects/:id/epics/:epicId/run
**Then** I see overall epic progress showing X/Y stories completed

**Given** the epic run dashboard is loaded
**When** I examine the page
**Then** I see a grid of story cards with real-time status updates via SSE

**Given** I am viewing the epic run dashboard
**When** story statuses change
**Then** the cards update automatically without page refresh

**Given** I am admin on the epic run dashboard
**When** I examine the controls
**Then** I see Pause/Resume buttons for the epic run

**Given** a story has failed in the epic run
**When** I examine the story card
**Then** it is highlighted with visual indication and error summary is shown

**Given** I click on a story card
**When** the navigation completes
**Then** I am taken to the individual run detail page with logs and timeline

## Epic 8: Retry, CI Polling & Resilience

System handles CI failures intelligently: incremental retry, full retry fallback, circuit breaker.

> **Depends on:** Epic 6 (prompt templates) for implement-retry.hbs template rendering during incremental retries.

### Story 8.1: [BACK] CI polling action via GitProvider

As a developer,
I want the system to poll CI status automatically after PR creation,
So that my pipeline only advances when tests pass.

**Acceptance Criteria:**

**Given** the ActionRegistry is initialized
**When** I lookup the "ci_poll" action
**Then** I receive a valid Action implementation

**Given** a PR has CI configured
**When** the ci_poll action executes with valid prURL parameter
**Then** GitProvider.GetCIStatus is called with the PR URL

**Given** CI status is "passing"
**When** ci_poll completes polling
**Then** the action returns success and the step completes

**Given** CI status is "failing"
**When** ci_poll completes polling
**Then** the action returns failure with CI error output

**Given** polling interval is 30s and max wait is 15min
**When** ci_poll is executed
**Then** the action polls at 30s intervals until pass, fail, or timeout

**Given** polling exceeds max wait time
**When** ci_poll times out
**Then** the step fails with error code "ci_poll_timeout"

**Given** ci_poll action is configured in pipeline YAML
**When** the pipeline step executes
**Then** interval and timeout values are read from config with defaults (30s, 15min)

### Story 8.2: [BACK] Incremental retry action

> **Depends on:** Epic 6 Story 6.3 (Handlebars rendering engine) for implement-retry.hbs template rendering.

As a developer,
I want failed steps to retry with error context,
So that the agent can fix errors without starting from scratch.

**Acceptance Criteria:**

**Given** a step fails during execution
**When** the retry mechanism triggers
**Then** the agent receives the original prompt plus error output plus current diff

**Given** the implement-retry.hbs template exists
**When** an incremental retry is triggered
**Then** the prompt is rendered using implement-retry.hbs with error context

**Given** a step has failed once
**When** the first incremental retry executes
**Then** a new run_step record is created and linked to the same run

**Given** a step has failed twice incrementally
**When** the system evaluates retry strategy
**Then** the system falls back to full retry mode

**Given** run_step metadata tracks retry count
**When** each retry attempt executes
**Then** the retry count is incremented and stored in run_step metadata

**Given** max incremental retries is 2
**When** a step fails three times
**Then** the system switches to full retry strategy

### Story 8.3: [BACK] Full retry fallback + circuit breaker

As a platform administrator,
I want failed pipelines to retry from scratch after incremental retries fail,
So that transient environment issues can be resolved.

**Acceptance Criteria:**

**Given** a step has failed 2 incremental retries
**When** the retry logic evaluates next action
**Then** a full retry is triggered with a fresh container and fresh branch

**Given** a full retry is initiated
**When** the prompt is rendered
**Then** the original implement.hbs template is used without error context

**Given** a project has 3 consecutive failed runs
**When** the circuit breaker evaluates project state
**Then** the circuit breaker is triggered and the run is halted

**Given** circuit breaker state is tracked per project
**When** a circuit breaker is triggered
**Then** the state is persisted in the projects table

**Given** a circuit breaker is triggered
**When** the event is emitted
**Then** an event "circuit_breaker.triggered" is published with project context

**Given** an admin wants to reset the circuit breaker
**When** POST /api/v1/projects/{id}/circuit-breaker/reset is called
**Then** the circuit breaker state is cleared and runs can be attempted again

**Given** circuit breaker max failures is configurable
**When** the config is loaded
**Then** the default is 3 consecutive failures

**Given** migration 000011 for circuit breaker columns exists
**When** the migration is applied
**Then** the projects table gains columns: circuit_breaker_count (INT default 0), circuit_breaker_active (BOOLEAN default false), circuit_breaker_max (INT default 3)

### Story 8.4: [FRONT] Retry status display + circuit breaker indicator

As a user,
I want to see retry attempts and circuit breaker status,
So that I understand why my pipeline stopped and can take action.

**Acceptance Criteria:**

**Given** a run has retry attempts
**When** I view the run detail page
**Then** I see a list of all retry attempts with attempt number, type (incremental/full), and result

**Given** a retry attempt exists
**When** I click on the attempt in the list
**Then** the attempt expands to show the error context that triggered it

**Given** a project has circuit breaker triggered
**When** I view the project dashboard
**Then** I see the circuit breaker status indicator

**Given** circuit breaker is triggered on a project
**When** I view the project page
**Then** I see a red banner with message "Circuit breaker triggered" and a "Reset" button

**Given** I am an admin and circuit breaker is triggered
**When** I click the "Reset" button
**Then** a confirmation dialog appears and on confirm, the circuit breaker is reset

**Given** a circuit breaker is triggered
**When** the event is received via SSE
**Then** a toast notification appears with the circuit breaker message

## Epic 9: Cost Tracking & Notifications

User can see costs in real-time and receive notifications on pipeline events.

### Story 9.1: [BACK] Cost tracking table + per-step cost recording

As a platform administrator,
I want to track token usage and costs per step,
So that I can monitor spending and optimize model usage.

**Acceptance Criteria:**

**Given** migration 000012 exists
**When** the migration is applied
**Then** a cost_records table is created with columns: id (UUID PK), run_step_id (FK run_steps CASCADE), tokens_input (BIGINT), tokens_output (BIGINT), cost_usd (DECIMAL(10,6)), model (VARCHAR), created_at

**Given** an agent outputs NDJSON with cost events
**When** the agent_run step parses output
**Then** token usage (input/output) is extracted from the NDJSON

**Given** a step completes successfully
**When** the cost record is created
**Then** the record is inserted into cost_records linked to the run_step_id

**Given** model-specific pricing is configured
**When** a cost record is created
**Then** the cost_usd is calculated based on the model pricing from project settings

**Given** pricing is not configured for a model
**When** a cost record is created
**Then** the cost_usd defaults to zero with a warning logged

### Story 9.2: [BACK] Cost aggregation API

As a user,
I want to retrieve cost data via API,
So that I can see spending breakdowns by story, run, and model.

**Acceptance Criteria:**

**Given** I am authenticated
**When** I GET /api/v1/projects/{projectId}/costs?period=7d
**Then** I receive aggregated costs for the last 7 days with breakdown by story, run, and model

**Given** a story has multiple runs
**When** I GET /api/v1/projects/{projectId}/stories/{storyId}/costs
**Then** I receive total costs for that story across all runs

**Given** a run has multiple steps
**When** I GET /api/v1/projects/{projectId}/runs/{runId}/costs
**Then** I receive total costs for that run with per-step breakdown

**Given** projects table includes budget fields
**When** I GET /api/v1/projects/{id}
**Then** the response includes max_budget_per_story and max_budget_per_project fields

**Given** budget limits are set on a project
**When** cost aggregation is returned
**Then** current usage is displayed alongside budget limits for informational purposes only

**Given** no cost data exists for a resource
**When** I query the cost API
**Then** I receive zero costs with empty breakdown arrays

### Story 9.3: [BACK] Notification dispatcher + Discord webhook

As a user,
I want to receive notifications on pipeline events,
So that I stay informed about run progress without polling.

**Acceptance Criteria:**

**Given** migration 000013 exists
**When** the migration is applied
**Then** a notification_configs table is created with columns: id (UUID PK), project_id (FK projects CASCADE), channel_type (VARCHAR), config (JSONB), events_filter (JSONB array), enabled (BOOLEAN), created_at

**Given** a notification config exists for a project
**When** an event matching events_filter is published
**Then** the dispatcher sends the event to all matching notification channels

**Given** a Discord webhook is configured
**When** an event is dispatched to Discord
**Then** a POST request is sent to the webhook URL with a formatted message

**Given** a generic webhook is configured
**When** an event is dispatched to the webhook
**Then** a POST request is sent with the event payload as JSON

**Given** supported events include run.completed, run.failed, hitl_gate.pending, circuit_breaker.triggered
**When** a notification config is created
**Then** the events_filter can include any of these event types

**Given** a notification channel is disabled
**When** an event is dispatched
**Then** the disabled channel is skipped and no notification is sent

### Story 9.4: [FRONT] Cost dashboard page

As a user,
I want to view cost data in the UI,
So that I can monitor spending and identify expensive runs.

**Acceptance Criteria:**

**Given** I am authenticated
**When** I navigate to /projects/:id/costs
**Then** I see the cost dashboard page

**Given** I am on the cost dashboard
**When** the page loads
**Then** I see summary cards showing total cost this week, total cost this month, and average cost per story

**Given** cost data exists for the project
**When** I view the cost dashboard
**Then** I see a PrimeVue Chart (line chart) showing cost over time

**Given** the cost chart is displayed
**When** I toggle between 7d and 30d view
**Then** the chart updates to show the selected time period

**Given** recent runs have cost data
**When** I view the cost dashboard
**Then** I see a table of recent runs with a cost column

**Given** I click on a run in the cost table
**When** the click event fires
**Then** I am navigated to the run detail page

**Given** budget limits are configured
**When** I view the cost dashboard
**Then** I see configured limits displayed alongside current usage (informational only)

### Story 9.5: [FRONT] Notification settings page

As an admin,
I want to configure notification channels,
So that I receive alerts on the channels I prefer.

**Acceptance Criteria:**

**Given** I am admin
**When** I navigate to /projects/:id/settings/notifications
**Then** I see the notification settings page as a tab in project settings

**Given** I am on the notification settings page
**When** the page loads
**Then** I see a list of configured notification channels

**Given** I want to add a notification channel
**When** I click "Add Channel"
**Then** a dialog appears with fields for type (Discord/Webhook), config (URL), and events to subscribe

**Given** I configure a Discord webhook
**When** I save the channel
**Then** the channel is created with enabled=true by default

**Given** a notification channel exists
**When** I toggle the enable/disable switch
**Then** the channel's enabled status is updated via API

**Given** a notification channel is configured
**When** I click the "Test" button
**Then** a test notification is sent to the channel

**Given** I want to remove a channel
**When** I click "Delete" and confirm
**Then** the channel is deleted from the database

## Epic 10: Reference Test Project & Validation

System includes a reference project (todo app) to validate the pipeline end-to-end.

### Story 10.1: [SHARED] Todo app reference project structure

As a developer,
I want a reference project with a simple app,
So that I can validate the pipeline with a known baseline.

**Acceptance Criteria:**

**Given** the repository is initialized
**When** I examine the test-project/ directory
**Then** I see a complete todo app structure

**Given** the todo app backend exists
**When** I examine the backend code
**Then** I see a minimal Node.js API with CRUD endpoints for todos

**Given** the todo app frontend exists
**When** I examine the frontend code
**Then** I see a minimal Vue/HTML page for managing todos

**Given** the todo app has a Dockerfile
**When** I run docker build in test-project/
**Then** the app image builds successfully

**Given** docker-compose.yml exists in test-project/
**When** I run docker compose up
**Then** the todo app runs locally with all services

**Given** test-project/README.md exists
**When** I read the README
**Then** I see documentation explaining the project purpose and setup

### Story 10.2: [SHARED] Todo app CI pipeline + seed data

As a developer,
I want the todo app to have a functioning CI pipeline,
So that I can validate CI polling in the main system.

**Acceptance Criteria:**

**Given** the todo app has a CI config file
**When** I examine the config
**Then** I see stages for build, test, and lint

**Given** the CI pipeline is configured
**When** a PR is created for the todo app
**Then** the CI pipeline runs all stages

**Given** seed data exists
**When** I examine test-project/seed.sql
**Then** I see 5-10 sample todos for testing

**Given** E2E tests exist
**When** I run the E2E tests
**Then** CRUD operations are validated via Playwright or curl-based tests

**Given** the CI pipeline is functional
**When** all tests pass
**Then** the pipeline is permanently green (baseline for validation)

### Story 10.3: [SHARED] Pipeline validation stories + integration test

As a platform developer,
I want integration tests using the reference project,
So that I can verify the entire pipeline works end-to-end.

**Acceptance Criteria:**

**Given** sample stories exist in test-project/stories/
**When** I examine the markdown files
**Then** I see 3-5 stories in standard frontmatter format (key, epic, depends_on, scope)

**Given** the integration test suite exists
**When** I run the integration tests
**Then** stories are imported into the system

**Given** stories are imported
**When** the integration test triggers an epic run
**Then** the pipeline completes successfully

**Given** the integration test runs
**When** I examine the test steps
**Then** the test validates: story import, DAG building, container launch, agent execution, PR creation

**Given** the integration test uses testcontainers-go
**When** the test runs
**Then** an isolated test environment is created with all dependencies

**Given** the integration test completes
**When** I examine test-project/README.md
**Then** I see documentation of the validation flow and how to run the test
