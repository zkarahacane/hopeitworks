-- =============================================================================
-- hopeitworks dev seed data
-- =============================================================================
-- Purpose: Pre-populate local dev database with test users and projects.
-- Run:     cd backend && make seed
-- Reset:   cd backend && make reset-db
--
-- Credentials:
--   admin@hopeitworks.dev / admin123  (role: admin)
--   dev@hopeitworks.dev   / dev123    (role: user)
--   alice@hopeitworks.dev / alice123  (role: user)
--
-- Idempotent: safe to run multiple times (uses ON CONFLICT).
-- =============================================================================

BEGIN;

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@hopeitworks.dev',
    crypt('admin123', gen_salt('bf', 10)),
    'Admin User',
    'admin'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'dev@hopeitworks.dev',
    crypt('dev123', gen_salt('bf', 10)),
    'Dev User',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

INSERT INTO users (id, email, password_hash, name, role)
VALUES (
    '00000000-0000-0000-0000-000000000003',
    'alice@hopeitworks.dev',
    crypt('alice123', gen_salt('bf', 10)),
    'Alice Developer',
    'user'
) ON CONFLICT (email) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    name = EXCLUDED.name,
    role = EXCLUDED.role;

-- ---------------------------------------------------------------------------
-- Projects
-- ---------------------------------------------------------------------------

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000101',
    'Todo App',
    'Reference todo application for pipeline validation and baseline testing',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/todo-app',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000102',
    'E-commerce API',
    'Sample e-commerce REST API for multi-project testing scenarios',
    '00000000-0000-0000-0000-000000000001',
    'https://github.com/hopeitworks/ecommerce-api',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

INSERT INTO projects (id, name, description, owner_id, repo_url, default_model)
VALUES (
    '00000000-0000-0000-0000-000000000103',
    'Frontend Kit',
    'Vue 3 component library project owned by dev user',
    '00000000-0000-0000-0000-000000000002',
    'https://github.com/hopeitworks/frontend-kit',
    'claude-sonnet-4-20250514'
) ON CONFLICT (name) DO UPDATE SET
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    repo_url = EXCLUDED.repo_url,
    default_model = EXCLUDED.default_model;

-- ---------------------------------------------------------------------------
-- Project memberships (conditional: only if project_users table exists)
-- ---------------------------------------------------------------------------

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'project_users'
    ) THEN
        -- Admin owns Todo App and E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000001', 'owner')
        ON CONFLICT DO NOTHING;

        -- Dev user is member of Todo App
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000101', '00000000-0000-0000-0000-000000000002', 'member')
        ON CONFLICT DO NOTHING;

        -- Alice is member of E-commerce API
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000102', '00000000-0000-0000-0000-000000000003', 'member')
        ON CONFLICT DO NOTHING;

        -- Dev user owns Frontend Kit
        INSERT INTO project_users (project_id, user_id, role)
        VALUES ('00000000-0000-0000-0000-000000000103', '00000000-0000-0000-0000-000000000002', 'owner')
        ON CONFLICT DO NOTHING;

        RAISE NOTICE 'Seed: project_users memberships inserted';
    ELSE
        RAISE NOTICE 'Seed: project_users table not found (Story 1-6 not applied), skipping memberships';
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Epics (Todo App project)
-- ---------------------------------------------------------------------------

INSERT INTO epics (id, project_id, name, description, status)
VALUES (
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000101',
    'Foundation',
    'Project scaffolding, database setup, and core infrastructure',
    'completed'
) ON CONFLICT (project_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    status = EXCLUDED.status;

INSERT INTO epics (id, project_id, name, description, status)
VALUES (
    '00000000-0000-0000-0000-000000000202',
    '00000000-0000-0000-0000-000000000101',
    'Task Management',
    'CRUD operations for todo items with priorities and due dates',
    'in_progress'
) ON CONFLICT (project_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    status = EXCLUDED.status;

INSERT INTO epics (id, project_id, name, description, status)
VALUES (
    '00000000-0000-0000-0000-000000000203',
    '00000000-0000-0000-0000-000000000101',
    'User Authentication',
    'JWT-based auth with refresh tokens and role management',
    'backlog'
) ON CONFLICT (project_id, name) DO UPDATE SET
    description = EXCLUDED.description,
    status = EXCLUDED.status;

-- ---------------------------------------------------------------------------
-- Stories (Todo App project — various statuses for UI testing)
-- ---------------------------------------------------------------------------

-- Epic 1: Foundation (completed stories)
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000301',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000201',
    'S-01',
    'Project scaffolding',
    'Initialize Go module with chi router, Postgres, and Docker Compose dev stack',
    'backend',
    'completed',
    '[]',
    'make run works, /health returns 200, DB connection established'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000302',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000201',
    'S-02',
    'OpenAPI spec and code generation',
    'Define REST API contract in openapi.yaml and wire oapi-codegen + openapi-typescript',
    'api',
    'completed',
    '["S-01"]',
    'make generate succeeds, generated types match spec, frontend client compiles'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

-- Epic 2: Task Management (mixed statuses)
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000303',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-03',
    'Todo CRUD API',
    'Implement GET /todos, POST /todos, PUT /todos/{id}, DELETE /todos/{id} endpoints',
    'backend',
    'completed',
    '["S-02"]',
    'All CRUD endpoints return correct status codes, pagination works, soft delete implemented'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000304',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-04',
    'Todo list Vue component',
    'Build TodoList and TodoItem components with PrimeVue DataTable and inline editing',
    'frontend',
    'running',
    '["S-03"]',
    'List renders items, inline edit saves via API, delete with confirmation dialog'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000305',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-05',
    'Priority and due date filters',
    'Add filter sidebar with priority (low/medium/high) and date range picker',
    'frontend',
    'backlog',
    '["S-04"]',
    'Filters persist in URL query params, combine correctly, reset button works'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000306',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000202',
    'S-06',
    'Real-time updates via SSE',
    'Push todo changes to all connected clients using Server-Sent Events',
    'backend',
    'failed',
    '["S-03"]',
    'SSE endpoint streams events, frontend reconnects on drop, no duplicate events'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

-- Epic 3: Auth (backlog)
INSERT INTO stories (id, project_id, epic_id, key, title, objective, scope, status, depends_on, acceptance_criteria)
VALUES (
    '00000000-0000-0000-0000-000000000307',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000203',
    'S-07',
    'JWT authentication middleware',
    'Implement login endpoint, JWT generation, and chi middleware for auth',
    'backend',
    'backlog',
    '["S-01"]',
    'POST /auth/login returns JWT, expired tokens return 401, protected routes enforced'
) ON CONFLICT (project_id, key) DO UPDATE SET
    title = EXCLUDED.title,
    status = EXCLUDED.status;

-- ---------------------------------------------------------------------------
-- Pipeline config (Todo App)
-- ---------------------------------------------------------------------------

INSERT INTO pipeline_configs (id, project_id, config_yaml, version)
VALUES (
    '00000000-0000-0000-0000-000000000401',
    '00000000-0000-0000-0000-000000000101',
    'steps:
  - id: 10000000-0000-0000-0000-000000000001
    name: implement
    action_type: implement
    model: claude-sonnet-4-5
    auto_approve: false
    retry_policy:
      max_retries: 2
      retry_type: on-failure
  - id: 10000000-0000-0000-0000-000000000002
    name: review
    action_type: review
    model: claude-sonnet-4-5
    auto_approve: true
    retry_policy:
      max_retries: 1
      retry_type: on-failure
  - id: 10000000-0000-0000-0000-000000000003
    name: merge
    action_type: merge
    model: claude-sonnet-4-5
    auto_approve: false
    retry_policy:
      max_retries: 0
      retry_type: none
',
    1
) ON CONFLICT (project_id) DO UPDATE SET
    config_yaml = EXCLUDED.config_yaml,
    version = pipeline_configs.version + 1;

-- ---------------------------------------------------------------------------
-- Runs (various statuses for UI testing)
-- ---------------------------------------------------------------------------

-- Run 1: completed (S-03)
INSERT INTO runs (id, project_id, story_id, status, started_at, completed_at)
VALUES (
    '00000000-0000-0000-0000-000000000501',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000303',
    'completed',
    now() - interval '3 days',
    now() - interval '3 days' + interval '18 minutes'
) ON CONFLICT DO NOTHING;

-- Run 2: running (S-04 — currently active)
INSERT INTO runs (id, project_id, story_id, status, started_at)
VALUES (
    '00000000-0000-0000-0000-000000000502',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000304',
    'running',
    now() - interval '12 minutes'
) ON CONFLICT DO NOTHING;

-- Run 3: failed (S-06)
INSERT INTO runs (id, project_id, story_id, status, started_at, completed_at, error_message)
VALUES (
    '00000000-0000-0000-0000-000000000503',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000306',
    'failed',
    now() - interval '1 day',
    now() - interval '1 day' + interval '7 minutes',
    'CI pipeline failed: TestSSEReconnect timed out after 30s'
) ON CONFLICT DO NOTHING;

-- Run 4: pending (S-05)
INSERT INTO runs (id, project_id, story_id, status)
VALUES (
    '00000000-0000-0000-0000-000000000504',
    '00000000-0000-0000-0000-000000000101',
    '00000000-0000-0000-0000-000000000305',
    'pending'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Run steps (Run 1 — completed, all steps done)
-- ---------------------------------------------------------------------------

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at)
VALUES
    ('00000000-0000-0000-0000-000000000601', '00000000-0000-0000-0000-000000000501', 'git-branch',   1, 'git_branch',  'completed', now() - interval '3 days',                          now() - interval '3 days' + interval '15 seconds'),
    ('00000000-0000-0000-0000-000000000602', '00000000-0000-0000-0000-000000000501', 'dev-agent',    2, 'agent_run',   'completed', now() - interval '3 days' + interval '20 seconds',   now() - interval '3 days' + interval '11 minutes'),
    ('00000000-0000-0000-0000-000000000603', '00000000-0000-0000-0000-000000000501', 'ci-wait',      3, 'ci_poll',     'completed', now() - interval '3 days' + interval '11 minutes',   now() - interval '3 days' + interval '14 minutes'),
    ('00000000-0000-0000-0000-000000000604', '00000000-0000-0000-0000-000000000501', 'review-agent', 4, 'agent_run',   'completed', now() - interval '3 days' + interval '14 minutes',   now() - interval '3 days' + interval '16 minutes'),
    ('00000000-0000-0000-0000-000000000605', '00000000-0000-0000-0000-000000000501', 'hitl-gate',    5, 'hitl_gate',   'completed', now() - interval '3 days' + interval '16 minutes',   now() - interval '3 days' + interval '17 minutes'),
    ('00000000-0000-0000-0000-000000000606', '00000000-0000-0000-0000-000000000501', 'merge',        6, 'git_merge',   'completed', now() - interval '3 days' + interval '17 minutes',   now() - interval '3 days' + interval '18 minutes')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Run steps (Run 2 — running, dev-agent in progress with retry)
-- ---------------------------------------------------------------------------

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at)
VALUES
    ('00000000-0000-0000-0000-000000000611', '00000000-0000-0000-0000-000000000502', 'git-branch',   1, 'git_branch',  'completed', now() - interval '12 minutes', now() - interval '11 minutes' - interval '45 seconds')
ON CONFLICT DO NOTHING;

-- dev-agent: first attempt failed
INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at, error_message, retry_count, retry_type)
VALUES (
    '00000000-0000-0000-0000-000000000612',
    '00000000-0000-0000-0000-000000000502',
    'dev-agent',
    2,
    'agent_run',
    'failed',
    now() - interval '11 minutes',
    now() - interval '5 minutes',
    'Claude agent exited with code 1: ESLint errors in TodoList.vue',
    0,
    NULL
) ON CONFLICT DO NOTHING;

-- dev-agent: incremental retry (currently running)
INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, retry_count, retry_type, parent_step_id,
                       container_id, log_tail)
VALUES (
    '00000000-0000-0000-0000-000000000613',
    '00000000-0000-0000-0000-000000000502',
    'dev-agent (retry 1)',
    3,
    'agent_run',
    'running',
    now() - interval '4 minutes',
    1,
    'incremental',
    '00000000-0000-0000-0000-000000000612',
    'abc123def456',
    E'[14:32:01] Fixing ESLint errors in src/features/todos/TodoList.vue\n[14:32:05] Removed unused import: defineProps\n[14:32:08] Fixed missing key prop in v-for loop\n[14:32:12] Running npm run lint...\n[14:32:15] All lint checks passed'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Run steps (Run 3 — failed, ci-wait timed out)
-- ---------------------------------------------------------------------------

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at)
VALUES
    ('00000000-0000-0000-0000-000000000621', '00000000-0000-0000-0000-000000000503', 'git-branch',   1, 'git_branch',  'completed', now() - interval '1 day',                          now() - interval '1 day' + interval '10 seconds'),
    ('00000000-0000-0000-0000-000000000622', '00000000-0000-0000-0000-000000000503', 'dev-agent',    2, 'agent_run',   'completed', now() - interval '1 day' + interval '15 seconds',   now() - interval '1 day' + interval '4 minutes')
ON CONFLICT DO NOTHING;

INSERT INTO run_steps (id, run_id, step_name, step_order, action, status, started_at, completed_at, error_message)
VALUES (
    '00000000-0000-0000-0000-000000000623',
    '00000000-0000-0000-0000-000000000503',
    'ci-wait',
    3,
    'ci_poll',
    'failed',
    now() - interval '1 day' + interval '4 minutes',
    now() - interval '1 day' + interval '7 minutes',
    'CI pipeline failed: TestSSEReconnect timed out after 30s'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- HITL request (Run 1 — approved)
-- ---------------------------------------------------------------------------

INSERT INTO hitl_requests (id, run_step_id, gate_type, diff_content, status, resolved_at, resolved_by)
VALUES (
    '00000000-0000-0000-0000-000000000701',
    '00000000-0000-0000-0000-000000000605',
    'approval',
    E'diff --git a/internal/handler/todo.go b/internal/handler/todo.go\n+++ b/internal/handler/todo.go\n@@ -45,6 +45,15 @@ func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {\n+    if req.Title == "" {\n+        renderError(w, errors.NewValidation("title", "cannot be empty"))\n+        return\n+    }',
    'approved',
    now() - interval '3 days' + interval '17 minutes',
    '00000000-0000-0000-0000-000000000001'
) ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- Events (append-only — representative pipeline events)
-- ---------------------------------------------------------------------------

INSERT INTO events (id, project_id, entity_type, entity_id, action, payload)
VALUES
    -- Run 1 lifecycle
    ('00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000501', 'started',   '{"story_key": "S-03"}'),
    ('00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000501', 'completed', '{"story_key": "S-03", "duration_ms": 1080000}'),
    -- Run 2 lifecycle
    ('00000000-0000-0000-0000-000000000803', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000502', 'started',   '{"story_key": "S-04"}'),
    -- Run 3 lifecycle
    ('00000000-0000-0000-0000-000000000804', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000503', 'started',   '{"story_key": "S-06"}'),
    ('00000000-0000-0000-0000-000000000805', '00000000-0000-0000-0000-000000000101', 'run', '00000000-0000-0000-0000-000000000503', 'failed',    '{"story_key": "S-06", "error": "CI pipeline failed: TestSSEReconnect timed out after 30s"}'),
    -- HITL event
    ('00000000-0000-0000-0000-000000000806', '00000000-0000-0000-0000-000000000101', 'hitl', '00000000-0000-0000-0000-000000000701', 'approved',  '{"resolved_by": "00000000-0000-0000-0000-000000000001", "run_id": "00000000-0000-0000-0000-000000000501"}')
ON CONFLICT DO NOTHING;

COMMIT;
